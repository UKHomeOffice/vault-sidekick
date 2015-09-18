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
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/golang/glog"
	"gopkg.in/yaml.v2"
)

func main() {
	// step: parse and validate the command line / environment options
	if err := parseOptions(); err != nil {
		showUsage("invalid options, %s", err)
	}

	// step: create a client to vault
	vault, err := newVaultService(options.vaultURL, options.vaultToken)
	if err != nil {
		glog.Errorf("failed to create a vault client, error: %s", err)
	}

	// step: setup the termination signals
	signalChannel := make(chan os.Signal)
	signal.Notify(signalChannel, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// step: create a channel to receive events upon and add our resources for renewal
	ch := make(vaultEventsChannel, 10)

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
			go processResource(evt.resource, evt.secret)

		case <-signalChannel:
			glog.Infof("recieved a termination signal, shutting down the service")
			os.Exit(0)
		}
	}

}

// processResource ... write the resource to file, converting into the selected format
func processResource(rn *vaultResource, data map[string]interface{}) error {
	var content []byte
	var err error

	// step: determine the resource path
	resourcePath := rn.filename()
	if !strings.HasPrefix(resourcePath, "/") {
		resourcePath = fmt.Sprintf("%s/%s", options.secretsDirectory, resourcePath)
	}

	// step: get the output format
	contentFormat := rn.getFormat()
	glog.V(3).Infof("saving resource: %s, format: %s", rn, contentFormat)

	switch contentFormat {
	case "yaml":
		// marshall the content to yaml
		if content, err = yaml.Marshal(data); err != nil {
			return err
		}
	case "ini":
		var buf bytes.Buffer
		for key, val := range data {
			buf.WriteString(fmt.Sprintf("%s = %s\n", key, val))
		}
		content = buf.Bytes()
	case "txt":
		keys := getKeys(data)
		if len(keys) > 1 {
			// step: for plain formats we need to iterate the keys and produce a file per key
			for suffix, content := range data {
				filename := fmt.Sprintf("%s.%s", resourcePath, suffix)
				// step: write the file
				if err := writeFile(filename, []byte(fmt.Sprintf("%s", content))); err != nil {
					glog.Errorf("failed to write resource: %s, elemment: %s, filename: %s, error: %s",
						rn, suffix, filename, err)
					continue
				}
			}
			return nil
		}

		// step: we only have the one key, so will write plain
		value, _ := data[keys[0]]
		content = []byte(fmt.Sprintf("%s", value))
	case "json":
		if content, err = json.MarshalIndent(data, "", "    "); err != nil {
			return err
		}
	}

	// step: write the content to file
	if err := writeFile(resourcePath, content); err != nil {
		glog.Errorf("failed to write the resource: %s to file: %s, error: %s", rn, resourcePath, err)
		return err
	}

	return nil
}

// writeFile ... writes the content of a file
func writeFile(filename string, content []byte) error {
	// step: are we dry running?
	if options.dryRun {
		glog.Infof("dry-run: filename: %s, content:", filename)
		fmt.Printf("%s\n", string(content))
		return nil
	}

	if err := ioutil.WriteFile(filename, content, 0440); err != nil {
		return err
	}

	return nil
}