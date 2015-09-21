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

func main() {
	// step: parse and validate the command line / environment options
	if err := parseOptions(); err != nil {
		showUsage("invalid options, %s", err)
	}
	// step: create a client to vault
	vault, err := newVaultService(options.vaultURL)
	if err != nil {
		showUsage("unable to create the vault client: %s", err)
	}

	// step: setup the termination signals
	signalChannel := make(chan os.Signal)
	signal.Notify(signalChannel, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// step: create a channel to receive events upon and add our resources for renewal
	ch := make(chan vaultResourceEvent, 10)
	// step: add each of the resources to the service processor
	for _, rn := range options.resources.items {
		// step: valid the resource
		if err := rn.isValid(); err != nil {
			showUsage("%s", err)
		}
		vault.watch(rn, ch)
	}

	// step: we simply wait for events i.e. secrets from vault and write them to the output directory
	for {
		select {
		case evt := <-ch:
			// step: write the secret to the output directory
			go writeResource(evt.resource, evt.secret)

		case <-signalChannel:
			glog.Infof("recieved a termination signal, shutting down the service")
			os.Exit(0)
		}
	}
}
