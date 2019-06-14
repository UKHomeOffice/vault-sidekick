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
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/golang/glog"
)

var (
	prog    = "vault-sidekick"
	release = "v0.3.10"
	gitsha  = ""
)

func main() {
	version := fmt.Sprintf("%s (git+sha %s)", release, gitsha)
	// step: parse and validate the command line / environment options
	if err := parseOptions(); err != nil {
		showUsage("invalid options, %s", err)
	}
	if options.showVersion {
		fmt.Printf("%s %s\n", prog, version)
		return
	}
	glog.Infof("starting the %s, %s", prog, version)

	if options.oneShot {
		glog.Infof("running in one-shot mode")
	}

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

	toProcess := options.resources.items
	toProcessLock := &sync.Mutex{}
	failedResource := false
	if options.oneShot && len(toProcess) == 0 {
		glog.Infof("nothing to retrieve from vault. exiting...")
		os.Exit(0)
	}
	// step: we simply wait for events i.e. secrets from vault and write them to the output directory
	for {
		select {
		case evt := <-updates:
			glog.V(10).Infof("recieved an update from the resource: %s", evt.Resource)
			go func(r VaultEvent) {
				toProcessLock.Lock()
				defer toProcessLock.Unlock()
				switch r.Type {
				case EventTypeSuccess:
					if err := processResource(evt.Resource, evt.Secret); err != nil {
						glog.Errorf("failed to write out the update, error: %s", err)
					}
					if options.oneShot {
						for i, r := range toProcess {
							if evt.Resource == r {
								toProcess = append(toProcess[:i], toProcess[i+1:]...)
							}
						}
					}
				case EventTypeFailure:
					if evt.Resource.MaxRetries > 0 && evt.Resource.MaxRetries < evt.Resource.Retries {
						for i, r := range toProcess {
							if evt.Resource == r {
								toProcess = append(toProcess[:i], toProcess[i+1:]...)
								failedResource = true
							}
						}
					}
				}
				if len(toProcess) == 0 {
					glog.Infof("no resources left to process. exiting...")
					if failedResource {
						os.Exit(1)
					} else {
						os.Exit(0)
					}
				}
			}(evt)
		case <-signalChannel:
			glog.Infof("recieved a termination signal, shutting down the service")
			os.Exit(0)
		}
	}
}
