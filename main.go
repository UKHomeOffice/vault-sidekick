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
	"os"
	"os/signal"
	"syscall"

	"github.com/golang/glog"
)

const (
	Prog    = "vault-sidekick"
	Version = "v0.1.1"
)

func main() {
	// step: parse and validate the command line / environment options

	if err := parseOptions(); err != nil {
		showUsage("invalid options, %s", err)
	}

	glog.Infof("starting the %s, version: %s", Prog, Version)

	// step: create a client to vault
	vault, err := NewVaultService(options.vaultURL)
	if err != nil {
		showUsage("unable to create the vault client: %s", err)
	}
	// step: create a channel to receive events upon and add our resources for renewal
	updates := make(chan VaultEvent, 10)
	vault.AddListener(updates)

	// step: setup the termination signals
	signalChannel := make(chan os.Signal)
	signal.Notify(signalChannel, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// step: add each of the resources to the service processor
	for _, rn := range options.resources.items {
		if err := rn.IsValid(); err != nil {
			showUsage("%s", err)
		}
		vault.Watch(rn)
	}

	// step: we simply wait for events i.e. secrets from vault and write them to the output directory
	for {
		select {
		case evt := <-updates:
			glog.V(10).Infof("recieved an update from the resource: %s", evt.Resource)
			go func(r VaultEvent) {
				if err := processResource(evt.Resource, evt.Secret); err != nil {
					glog.Errorf("failed to write out the update, error: %s", err)
				}
			}(evt)
		case <-signalChannel:
			glog.Infof("recieved a termination signal, shutting down the service")
			os.Exit(0)
		}
	}
}
