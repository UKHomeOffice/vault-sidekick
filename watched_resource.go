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

	"github.com/golang/glog"
	"github.com/hashicorp/vault/api"
)

const (
	renewalMinimum = 0.8
	renewalMaximum = 0.95
)

// watchedResource ... is a resource which is being watched - i.e. when the item is coming up for renewal
// lets grab it and renew the lease
type watchedResource struct {
	// the upstream listener to the event
	listener chan vaultResourceEvent
	// the resource itself
	resource *vaultResource
	// the last time the resource was retrieved
	lastUpdated time.Time
	// the time which the lease expires
	leaseExpireTime time.Time
	// the duration until we next time to renew lease
	renewalTime time.Duration
	// the secret
	secret *api.Secret
}

// notifyOnRenewal ... creates a trigger and notifies when a resource is up for renewal
func (r *watchedResource) notifyOnRenewal(ch chan *watchedResource) {
	go func() {
		// step: check if the resource has a pre-configured renewal time
		r.renewalTime = r.resource.update
		// step: if the answer is no, we set the notification between 80-95% of the lease time of the secret
		if r.renewalTime <= 0 {
			r.renewalTime = r.calculateRenewal()
		}
		glog.V(3).Infof("setting a renewal notification on resource: %s, time: %s", r.resource, r.renewalTime)
		// step: wait for the duration
		<-time.After(r.renewalTime)
		// step: send the notification on the renewal channel
		ch <- r
	}()
}

// calculateRenewal ... calculate the renewal between
func (r watchedResource) calculateRenewal() time.Duration {
	return time.Duration(getRandomWithin(
		int(float64(r.secret.LeaseDuration)*renewalMinimum),
		int(float64(r.secret.LeaseDuration)*renewalMaximum))) * time.Second
}
