/*
Copyright 2015 Home Office All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/hashicorp/vault/api"

	"github.com/UKHomeOffice/vault-sidekick/metrics"
)

// AuthInterface is the authentication interface
type AuthInterface interface {
	// Create and handle renewals of the token
	Create(*vaultAuthOptions) (string, error)
}

// VaultService is the main interface into the vault API - placing into a structure
// allows one to easily mock it and two to simplify the interface for us
type VaultService struct {
	vaultURL string
	// the vault client
	client *api.Client
	// the vault config
	config *api.Config
	// the token to authenticate with
	token string
	// the listener channel - technically we only have the one listener but there a long term reasons for adding this
	listeners []chan VaultEvent
	// a channel to inform of a new resource to processor
	resourceChannel chan *watchedResource
}

// VaultEvent is the definition which captures a change
type VaultEvent struct {
	// the resource this relates to
	Resource *VaultResource
	// the secret associated
	Secret map[string]interface{}
	// type of this event (success or failure)
	Type EventType
}

type EventType int

const (
	EventTypeSuccess EventType = iota
	EventTypeFailure EventType = iota
)

// NewVaultService creates a new implementation to speak to vault and retrieve the resources
//	url			: the url of the vault service
func NewVaultService(url string) (*VaultService, error) {
	var err error

	// step: create the config for client
	service := new(VaultService)
	service.vaultURL = url
	service.listeners = make([]chan VaultEvent, 0)

	// step: create the service processor channels
	service.resourceChannel = make(chan *watchedResource, 20)

	// step: retrieve a vault client
	service.client, err = newVaultClient(&options)
	if err != nil {
		return nil, err
	}

	// step: start the service processor off
	service.vaultServiceProcessor()

	return service, nil
}

// AddListener ... add a listener to the events listeners
func (r *VaultService) AddListener(ch chan VaultEvent) {
	glog.V(10).Infof("adding the listener: %v", ch)
	r.listeners = append(r.listeners, ch)
}

// Watch adds a watch on a resource and inform, renew which required and inform us when
// the resource is ready
func (r VaultService) Watch(rn *VaultResource) {
	r.resourceChannel <- &watchedResource{resource: rn}
}

// vaultServiceProcessor is the background routine responsible for retrieving the resources, renewing when required and
// informing those who are watching the resource that something has changed
func (r *VaultService) vaultServiceProcessor() {
	go func() {
		// a list of resource being watched
		var items []*watchedResource

		// the channel to receive renewal notifications on
		renewChannel := make(chan *watchedResource, 10)
		retrieveChannel := make(chan *watchedResource, 10)
		revokeChannel := make(chan *watchedResource, 10)
		statsChannel := time.NewTicker(options.statsInterval)

		for {
			select {
			// A new resource is being added to the service processor;
			//  - schedule the resource for retrieval
			case x := <-r.resourceChannel:
				glog.V(4).Infof("adding a resource into the service processor, resource: %s", x.resource)
				// step: add to the list of resources
				items = append(items, x)
				// step: push into the retrieval channel
				r.scheduleNow(x, retrieveChannel)

			// Retrieve a resource from vault
			//  - we retrieve the resource from vault
			//  - if we error attempting to retrieve the secret, we background and reschedule an attempt to add it
			//  - if ok, we grab the lease it and lease time, we setup a notification on renewal
			case x := <-retrieveChannel:
				// step: skip this resource if it's reached maxRetries
				if x.resource.MaxRetries > 0 && x.resource.Retries > x.resource.MaxRetries {
					glog.V(4).Infof("skipping resource %s as it's failed %d/%d times", x.resource.Retries, x.resource.MaxRetries+1)
					break
				}

				// step: save the current lease if we have one
				leaseID := ""
				if x.secret != nil && x.secret.LeaseID != "" {
					leaseID = x.secret.LeaseID
					glog.V(10).Infof("resource: %s has a previous lease: %s", x.resource, leaseID)
				}

				metrics.ResourceTotal(x.resource.ID())

				err := r.get(x)
				if err != nil {
					metrics.ResourceError(x.resource.ID())
					glog.Errorf("failed to retrieve the resource: %s from vault, error: %s", x.resource, err)
					// reschedule the attempt for later
					r.scheduleIn(x, retrieveChannel, getDurationWithin(3, 10))
					x.resource.Retries++
					r.upstream(VaultEvent{
						Resource: x.resource,
						Type:     EventTypeFailure,
					})
					break
				}

				metrics.ResourceSuccess(x.resource.ID())

				glog.V(4).Infof("successfully retrieved resource: %s, leaseID: %s", x.resource, x.secret.LeaseID)
				x.resource.Retries = 0

				// step: if we had a previous lease and the option is to revoke, lets throw into the revoke channel
				if leaseID != "" && x.resource.Revoked {
					// step: make a rough copy
					copy := &watchedResource{
						secret: &api.Secret{
							LeaseID: x.secret.LeaseID,
						},
					}

					r.scheduleIn(copy, revokeChannel, x.resource.RevokeDelay)
				}

				// step: setup a timer for renewal
				x.notifyOnRenewal(renewChannel)

				// step: update the upstream consumers
				r.upstream(VaultEvent{
					Resource: x.resource,
					Secret:   x.secret.Data,
					Type:     EventTypeSuccess,
				})

			// A watched resource is coming up for renewal
			// 	- we attempt to renew the resource from vault
			//	- if we encounter an error, we reschedule the attempt for the future
			//	- if we're ok, we update the watchedResource and we send a notification of the change upstream
			case x := <-renewChannel:
				// step: skip this resource if it's reached maxRetries
				if x.resource.MaxRetries > 0 && x.resource.Retries > x.resource.MaxRetries {
					glog.V(4).Infof("skipping resource %s as it's failed %d/%d times", x.resource.Retries, x.resource.MaxRetries+1)
					break
				}

				glog.V(4).Infof("resource: %s, lease: %s up for renewal, renewable: %t, revoked: %t", x.resource,
					x.secret.LeaseID, x.resource.Renewable, x.resource.Revoked)

				// step: we need to check if the lease has expired?
				if time.Now().Before(x.leaseExpireTime) {
					glog.V(3).Infof("the lease on resource: %s has expired, we need to get a new lease", x.resource)
					// push into the retrieval channel and break
					r.scheduleNow(x, retrieveChannel)
					break
				}

				// step: are we renewing the resource?
				if x.resource.Renewable {
					metrics.ResourceTotal(x.resource.ID())

					// step: is the underlining resource even renewable? - otherwise we can just grab a new lease
					if !x.secret.Renewable {
						glog.V(10).Infof("the resource: %s is not renewable, retrieving a new lease instead", x.resource)
						r.scheduleNow(x, retrieveChannel)
						break
					}

					// step: lets renew the resource
					err := r.renew(x)
					if err != nil {
						metrics.ResourceError(x.resource.ID())
						glog.Errorf("failed to renew the resource: %s for renewal, error: %s", x.resource, err)
						// reschedule the attempt for later
						r.scheduleIn(x, renewChannel, getDurationWithin(3, 10))
						x.resource.Retries++
						r.upstream(VaultEvent{
							Resource: x.resource,
							Type:     EventTypeFailure,
						})
						break
					}

					metrics.ResourceSuccess(x.resource.ID())

					glog.V(4).Infof("successfully renewed resource: %s, leaseID: %s", x.resource, x.secret.LeaseID)
					x.resource.Retries = 0
				}

				// step: the option for this resource is not to renew the secret but regenerate a new secret
				if !x.resource.Renewable {
					glog.V(4).Infof("resource: %s flagged as not renewable, shifting to regenerating the resource", x.resource)
					r.scheduleNow(x, retrieveChannel)
					break
				}

				// step: setup a timer for renewal
				x.notifyOnRenewal(renewChannel)

				// step: update any listener upstream
				r.upstream(VaultEvent{
					Resource: x.resource,
					Secret:   x.secret.Data,
					Type:     EventTypeSuccess,
				})

			// We receive a lease ID along on the channel, just revoke the lease when you can
			case x := <-revokeChannel:
				err := r.revoke(x.secret.LeaseID)
				if err != nil {
					glog.Errorf("failed to revoke the lease: %s, error: %s", x.secret.LeaseID, err)
				}

			// The statistics timer has gone off; we iterate the watched items and
			case <-statsChannel.C:
				glog.V(3).Infof("stats: %d resources being watched", len(items))
				for _, item := range items {
					glog.V(3).Infof("resourse: %s, lease id: %s, renewal in: %s seconds, expiration: %s",
						item.resource, item.secret.LeaseID, item.renewalTime, item.leaseExpireTime)
				}
			}
		}
	}()
}

// scheduleNow ... a helper method to perform an immediate reschedule into a channel
//	rn			: a pointer to the watched resource you wish to reschedule
//	ch			: the channel the resource should be placed into
func (r VaultService) scheduleNow(rn *watchedResource, ch chan *watchedResource) {
	r.scheduleIn(rn, ch, time.Duration(0))
}

// scheduleIn ... schedules an event back into a channel after n seconds
//	rn			: a referrence some reason you wish to pass
//	ch			: the channel the resource should be placed into
//	min			: the minimum amount of time i'm willing to wait
//	max			: the maximum amount of time i'm willing to wait
func (r VaultService) scheduleIn(rn *watchedResource, ch chan *watchedResource, duration time.Duration) {
	go func(x *watchedResource) {
		glog.V(3).Infof("rescheduling the resource: %s, channel: %v", rn.resource, ch)
		// step: are we doing a random wait?
		if duration > 0 {
			<-time.After(duration)
		}
		ch <- x
	}(rn)
}

// upstream ... the resource has changed thus we notify the upstream listener
//	item		: the item which has changed
func (r VaultService) upstream(item VaultEvent) {
	// step: chunk this into a go-routine not to block us
	for _, listener := range r.listeners {
		go func(ch chan VaultEvent) {
			ch <- item
		}(listener)
	}
}

// renew attempts to renew the lease on a resource
// 	rn			: the resource we wish to renew the lease on
func (r VaultService) renew(rn *watchedResource) error {
	glog.V(4).Infof("attempting to renew the lease: %s on resource: %s", rn.secret.LeaseID, rn.resource)
	// step: check the resource is renewable
	if !rn.secret.Renewable {
		return fmt.Errorf("the resource: %s is not renewable", rn.resource)
	}

	secret, err := r.client.Sys().Renew(rn.secret.LeaseID, 0)
	if err != nil {
		return err
	}

	// step: update the resource
	rn.lastUpdated = time.Now()
	rn.leaseExpireTime = rn.lastUpdated.Add(time.Duration(secret.LeaseDuration))

	glog.V(3).Infof("renewed resource: %s, leaseId: %s, lease_time: %s, expiration: %s",
		rn.resource, rn.secret.LeaseID, rn.secret.LeaseID, rn.leaseExpireTime)

	return nil
}

// revoke attempts to revoke the lease of a resource
//	lease		: the lease lease which was given when you got it
func (r VaultService) revoke(lease string) error {
	glog.V(3).Infof("attemping to revoking the lease: %s", lease)

	err := r.client.Sys().Revoke(lease)
	if err != nil {
		return err
	}
	glog.V(3).Infof("successfully revoked the leaseId: %s", lease)

	return nil
}

// get retrieves a secret from the vault
//	rn			: the watched resource
func (r VaultService) get(rn *watchedResource) error {
	var err error
	var secret *api.Secret
	// step: not sure who to cast map[string]string to map[string]interface{} doesn't like it anyway i try and do it

	params := make(map[string]interface{}, 0)
	for k, v := range rn.resource.Options {
		params[k] = interface{}(v)
	}
	glog.V(10).Infof("resource: %s, path: %s, params: %v", rn.resource.Resource, rn.resource.Path, params)

	glog.V(5).Infof("attempting to retrieve the resource: %s from vault", rn.resource)
	// step: perform a request to vault
	switch rn.resource.Resource {
	case "raw":
		request := r.client.NewRequest("GET", "/v1/"+rn.resource.Path)
		for k, v := range rn.resource.Options {
			request.Params.Add(k, v)
		}
		resp, err := r.client.RawRequest(request)
		if err != nil {
			return err
		}
		// step: read the response
		content, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		// step: construct a secret from the response
		secret = &api.Secret{
			LeaseID:   "raw",
			Renewable: false,
			Data: map[string]interface{}{
				"content": fmt.Sprintf("%s", content),
			},
		}
		if rn.resource.Update > 0 {
			secret.LeaseDuration = int(rn.resource.Update.Seconds())
		} else {
			secret.LeaseDuration = int((time.Duration(24) * time.Hour).Seconds())
		}
	case "pki":
		secret, err = r.client.Logical().Write(fmt.Sprintf(rn.resource.Path), params)
	case "transit":
		secret, err = r.client.Logical().Write(fmt.Sprintf(rn.resource.Path), params)
	case "aws":
		fallthrough
	case "cubbyhole":
		fallthrough
	case "gcp":
		fallthrough
	case "mysql":
		fallthrough
	case "postgres":
		fallthrough
	case "database":
		fallthrough
	case "secret":
		secret, err = r.client.Logical().Read(rn.resource.Path)
		// We must generate the secret if we have the create flag
		if rn.resource.Create && secret == nil && err == nil {
			glog.V(3).Infof("Create param specified, creating resource: %s", rn.resource.Path)
			params["value"] = newPassword(int(rn.resource.Size))
			secret, err = r.client.Logical().Write(fmt.Sprintf(rn.resource.Path), params)
			glog.V(3).Infof("Secret created: %s", rn.resource.Path)
			if err == nil {
				// Populate the secret data as stored in Vault...
				secret, err = r.client.Logical().Read(rn.resource.Path)
			}
		}
		// if there is a top-level metadata key this is from a v2 kv store
		if err == nil {
			if _, ok := secret.Data["metadata"]; ok {
				secret.Data = secret.Data["data"].(map[string]interface{})
			}
		}
	case "ssh":
		publicKeyData, err := ioutil.ReadFile(params["public_key_path"].(string))

		if err != nil {
			return fmt.Errorf("could not read data at specified public_key_path")
		}

		publicKeyDataString := string(publicKeyData)

		sshParams := map[string]interface{}{
			"public_key": publicKeyDataString,
			"cert_type":  params["cert_type"].(string),
		}

		secret, err = r.client.Logical().Write(fmt.Sprintf(rn.resource.Path), sshParams)
	}
	// step: check the error if any
	if err != nil {
		if strings.Contains(err.Error(), "missing client token") {
			// decision: until the rewrite, lets just exit for now
			glog.Fatalf("the vault token is no longer valid, exitting, error: %s", err)
		}
		return err
	}
	if secret == nil && err == nil {
		return fmt.Errorf("the resource does not exist")
	}

	if secret == nil {
		return fmt.Errorf("unable to retrieve the secret")
	}

	// step: update the watched resource
	rn.lastUpdated = time.Now()
	rn.secret = secret
	rn.leaseExpireTime = rn.lastUpdated.Add(time.Duration(secret.LeaseDuration))

	glog.V(3).Infof("retrieved resource: %s, leaseId: %s, lease_time: %s",
		rn.resource, rn.secret.LeaseID, time.Duration(rn.secret.LeaseDuration)*time.Second)

	return err
}

// newVaultClient creates and authenticates a vault client
func newVaultClient(opts *config) (*api.Client, error) {
	var err error
	var token string

	config := api.DefaultConfig()
	config.Address = opts.vaultURL

	config.HttpClient.Transport, err = buildHTTPTransport(opts)
	if err != nil {
		return nil, err
	}

	// step: create the actual client
	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}

	plugin := opts.vaultAuthOptions.Method
	switch plugin {
	case "userpass":
		token, err = NewUserPassPlugin(client).Create(opts.vaultAuthOptions)
	case "approle":
		token, err = NewAppRolePlugin(client).Create(opts.vaultAuthOptions)
	case "aws-ec2":
		token, err = NewAWSEC2Plugin(client).Create(opts.vaultAuthOptions)
	case "aws-iam":
		token, err = NewAWSIAMPlugin(client).Create(opts.vaultAuthOptions)
	case "gcp-gce":
		token, err = NewGCPGCEPlugin(client).Create(opts.vaultAuthOptions)
	case "kubernetes":
		token, err = NewKubernetesPlugin(client).Create(opts.vaultAuthOptions)
	case "token":
		opts.vaultAuthOptions.FileName = options.vaultAuthFile
		opts.vaultAuthOptions.FileFormat = options.vaultAuthFileFormat
		token, err = NewUserTokenPlugin(client).Create(opts.vaultAuthOptions)
	default:
		return nil, fmt.Errorf("unsupported authentication plugin: %s", plugin)
	}
	if err != nil {
		return nil, err
	}

	// step: set the token for the client
	client.SetToken(token)

	if opts.vaultRenewToken {
		tokeninfo, err := client.Auth().Token().LookupSelf()
		if err != nil {
			return nil, fmt.Errorf("failed to lookup token info: %s", err)
		}

		tokenttl, err := tokeninfo.TokenTTL()
		if err != nil {
			return nil, fmt.Errorf("failed to lookup token ttl: %s", err)
		}
		glog.Infof("token ttl is %v", tokenttl)
		renewPeriod := tokenttl / 2
		go func() {
			for {
				if renewPeriod < 1*time.Second {
					glog.Fatalf("fatal: token renew period is <1s, aborting")
				}
				glog.Infof("scheduling token renew in %v", renewPeriod)
				<-time.After(renewPeriod)

				glog.Infof("attempting token renew")
				newtokeninfo, err := client.Auth().Token().RenewSelf(0)
				if err != nil {
					renewPeriod = renewPeriod / 2
					glog.Warningf("error: failed to renew token, retrying in %v: %v", renewPeriod, err)
					continue
				}

				tokenttl, err := newtokeninfo.TokenTTL()
				if err != nil {
					glog.Warningf("error: failed to get new token ttl, using previous value %s: %s", renewPeriod, err)
				} else {
					glog.Infof("token ttl is %v", tokenttl)
					renewPeriod = tokenttl / 2
				}
			}
		}()
	}

	return client, nil
}

// buildHTTPTransport constructs a http transport for the http client
func buildHTTPTransport(opts *config) (*http.Transport, error) {
	// step: create the vault sidekick
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 10 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: opts.skipTLSVerify,
		},
	}
	if opts.skipTLSVerify {
		glog.Warning("skipping TLS verification is not recommended")
	}
	// step: are we loading a CA file
	if opts.vaultCaFile != "" {
		glog.V(3).Infof("loading the ca certificate: %s", opts.vaultCaFile)
		caCert, err := ioutil.ReadFile(opts.vaultCaFile)
		if err != nil {
			return nil, fmt.Errorf("unable to read in the ca: %s, reason: %s", opts.vaultCaFile, err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		transport.TLSClientConfig.RootCAs = caCertPool
	}

	return transport, nil
}
