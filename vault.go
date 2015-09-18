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
	"time"

	"github.com/hashicorp/vault/api"
	"github.com/golang/glog"
	"fmt"
)

// a channel to send resource
type resourceChannel chan *vaultResource

// vaultService ... is the main interface into the vault API - placing into a structure
// allows one to easily mock it and two to simplify the interface for us
type vaultService struct {
	// the vault client
	client *api.Client
	// the vault config
	config *api.Config
	// a channel to inform of a new resource to processor
	resourceCh chan *watchedResource
	// the statistics channel
	statCh *time.Ticker
}

type vaultResourceEvent struct {
	// the resource this relates to
	resource *vaultResource
	// the secret associated
	secret map[string]interface{}
}

// a channel of events
type vaultEventsChannel chan vaultResourceEvent

// watchedResource ... is a resource which is being watched - i.e. when the item is coming up for renewal
// lets grab it and renew the lease
type watchedResource struct {
	listener vaultEventsChannel
	// the resource itself
	resource *vaultResource
	// the last time the resource was retrieved
	lastUpdated time.Time
	// the duration until the next renewal
	renewalTime time.Duration
	// the secret
	secret *api.Secret
}

// notifyOnRenewal ... creates a trigger and notifies when a resource is up for renewal
func (r *watchedResource) notifyOnRenewal(ch chan *watchedResource) {
	go func() {
		// step: check if the resource has a pre-configured renewal time
		r.renewalTime = r.resource.leaseTime()

		// step: if the answer is no, we set the notification between 80-95% of the lease time of the secret
		if r.renewalTime <= 0 {
			glog.V(10).Infof("Calculating the renewal between 80-95 pcent of lease time: %d seconds", r.secret.LeaseDuration)
			r.renewalTime = time.Duration(getRandomWithin(
				int(float64(r.secret.LeaseDuration) * 0.8),
				int(float64(r.secret.LeaseDuration) * 0.95))) * time.Second
		}
		glog.V(3).Infof("Setting a renewal notification on resource: %s, time: %s", r.resource, r.renewalTime)
		// step: wait for the duration
		<- time.After(r.renewalTime)
		// step: send the notification on the renewal channel
		ch <- r
	}()
}

// newVaultService ... creates a new implementation to speak to vault and retrieve the resources
//	url			: the url of the vault service
//	token		: the token to use when speaking to vault
func newVaultService(url, token string) (*vaultService, error) {
	var err error
	glog.Infof("Creating a new vault client: %s", url)

	// step: create the config for client
	service := new(vaultService)
	service.config = api.DefaultConfig()
	service.config.Address = url

	// step: create the service processor channels
	service.resourceCh = make(chan *watchedResource, 20)
	service.statCh = time.NewTicker(options.statsInterval)

	// step: create the actual client
	service.client, err = api.NewClient(service.config)
	if err != nil {
		return nil, err
	}

	// step: set the token for the client
	service.client.SetToken(token)

	// step: start the service processor off
	service.vaultServiceProcessor()

	return service, nil
}

// vaultServiceProcessor ... is the background routine responsible for retrieving the resources, renewing when required and
// informing those who are watching the resource that something has changed
func (r vaultService) vaultServiceProcessor() {
	go func() {
		// a list of resource being watched
		items := make([]*watchedResource, 0)
		// the channel to receive renewal notifications on
		renewing := make(chan *watchedResource, 5)

		for {
			select {
			// A new resource is being added to the service processor;
			//  - we retrieve the resource from vault
			//  - if we error attempting to retrieve the secret, we background and reschedule an attempt to add it
			//  - if ok, we grab the lease it and lease time, we setup a notification on renewal
			case x := <-r.resourceCh:
				glog.V(3).Infof("Adding a resource into the service processor, resource: %s", x.resource)

				// step: retrieve the resource from vault
				secret, err := r.get(x.resource)
				if err != nil {
					glog.Errorf("Failed to retrieve the resource: %s from vault, error: %s", x.resource, err)
					// reschedule the attempt for later
					go func(x *watchedResource) {
						<- time.After(time.Duration(getRandomWithin(2,10)) * time.Second)
						r.resourceCh <- x
					}(x)
					break
				}
				// step: update the item references
				x.secret = secret
				x.lastUpdated = time.Now()

				// step: setup a timer for renewal
				x.notifyOnRenewal(renewing)

				// step: add to the list of resources
				items = append(items, x)

				r.upstream(x, secret)

			// A watched resource is coming up for renewal
			// 	- we attempt to grab the resource from vault
			//	- if we encounter an error, we reschedule the attempt for the future
			//	- if we're ok, we update the watchedResource and we send a notification of the change upstream
			case x := <-renewing:
				glog.V(3).Infof("Resource: %s coming up for renewal, attempting to renew now", x.resource)
				// step: we attempt to renew the lease on a resource and if not successfully we reschedule
				// a renewal notification for the future

				secret, err := r.get(x.resource)
				if err != nil {
					glog.Errorf("Failed to retrieve the resounce: %s for renewal, error: %s", x.resource, err)
					// reschedule the attempt for later
					go func(x *watchedResource) {
						<- time.After(time.Duration(getRandomWithin(3,20)) * time.Second)
						renewing <- x
					}(x)
					break
				}

				// step: update the item references
				x.secret = secret
				x.lastUpdated = time.Now()

				// step: setup a timer for renewal
				x.notifyOnRenewal(renewing)

				// step: update any listener upstream
				r.upstream(x, secret)

			// The statistics timer has gone off; we iterate the watched items and
			case <-r.statCh.C:
				glog.V(3).Infof("Stats: %d resources being watched", len(items))
				for _, item := range items {
					glog.V(3).Infof("resourse: %s, lease id: %s, renewal in: %s seconds",
						item.resource, item.secret.LeaseID, item.renewalTime)
				}
			}
		}
	}()
}

func (r vaultService) upstream(item *watchedResource, s *api.Secret) {
	// step: chunk this into a go-routine not to block us
	go func() {
		glog.V(6).Infof("Sending the event for resource: %s upstream to listener: %v", item.resource, item.listener)
		item.listener <- vaultResourceEvent{
			resource: item.resource,
			secret: s.Data,
		}
	}()
}

// get ... retrieve a secret from the vault
func (r vaultService) get(rn *vaultResource) (*api.Secret, error) {
	var err error
	var secret *api.Secret

	glog.V(5).Infof("Attempting to retrieve the resource: %s from vault", rn)
	switch rn.resource {
	case "pki":
		secret, err = r.client.Logical().Write(fmt.Sprintf("%s/issue/%s", rn.resource, rn.name),
			map[string]interface{}{
				"common_name": rn.options[OptionCommonName],
			})
	case "aws":
		secret, err = r.client.Logical().Read(fmt.Sprintf("%s/creds/%s", rn.resource, rn.name))
	case "mysql":
		secret, err = r.client.Logical().Read(fmt.Sprintf("%s/creds/%s", rn.resource, rn.name))
	case "secret":
		secret, err = r.client.Logical().Read(fmt.Sprintf("%s/%s", rn.resource, rn.name))
	}
	if secret == nil && err == nil {
		return nil, fmt.Errorf("does not exist")
	}

	return secret, err
}

// watch ... add a watch on a resource and inform, renew which required and inform us when
// the resource is ready
func (r *vaultService) watch(rn *vaultResource, ch vaultEventsChannel) error {
	glog.V(10).Infof("Adding the resource: %s, listener: %v to service processor", rn, ch)
	r.resourceCh <- &watchedResource{
		resource: rn,
		listener: ch,
	}

	return nil
}
