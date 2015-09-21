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
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"strings"
	"time"

	"github.com/golang/glog"
	"gopkg.in/yaml.v2"
)

func init() {
	rand.Seed(int64(time.Now().Nanosecond()))
}

// showUsage ... prints the command usage and exits
//	message		: an error message to display if exiting with an error
func showUsage(message string, args ...interface{}) {
	flag.PrintDefaults()
	if message != "" {
		fmt.Printf("\n[error] "+message+"\n", args...)
		os.Exit(1)
	}

	os.Exit(0)
}

// randomWait ... wait for a random amount of time
// 	min			: the minimum amount of time willing to wait
//	max			: the maximum amount of time willing to wait
func randomWait(min, max int) <-chan time.Time {
	return time.After(time.Duration(getRandomWithin(min, max)) * time.Second)
}

// hasKey ... checks to see if a key is present
//	key			: the key we are looking for
//	data		: a map of strings to something we are looking at
func hasKey(key string, data map[string]interface{}) bool {
	_, found := data[key]
	return found
}

// getKeys ... retrieve a list of keys from the map
// 	data		: the map which you wish to extract the keys from
func getKeys(data map[string]interface{}) []string {
	var list []string
	for key := range data {
		list = append(list, key)
	}
	return list
}

// readConfigFile ... read in a configuration file
//	filename		: the path to the file
func readConfigFile(filename string) (map[string]string, error) {
	suffix := path.Ext(filename)
	switch suffix {
	case ".json":
		return readJSONFile(filename)
	case ".yaml":
		return readYamlFile(filename)
	case ".yml":
		return readYamlFile(filename)
	}
	return nil, fmt.Errorf("unsupported config file format: %s", suffix)
}

// readJsonFile ... read in and unmarshall the data into a map
//	filename	: the path to the file container the json data
func readJSONFile(filename string) (map[string]string, error) {
	data := make(map[string]string, 0)

	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return data, err
	}
	// unmarshall the data
	err = json.Unmarshal(content, data)
	if err != nil {
		return data, err
	}

	return data, nil
}

// readYamlFile ... read in and unmarshall the data into a map
//	filename	: the path to the file container the yaml data
func readYamlFile(filename string) (map[string]string, error) {
	data := make(map[string]string, 0)

	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return data, err
	}
	// unmarshall the data
	err = yaml.Unmarshal(content, data)
	if err != nil {
		return data, err
	}

	return data, nil
}

// randomInt ... generate a random integer between min and max
//	min			: the smallest number we can accept
//	max			: the largest number we can accept
func getRandomWithin(min, max int) int {
	return rand.Intn(max-min) + min
}

// getEnv ... checks to see if an environment variable exists otherwise uses the default
//	env			: the name of the environment variable you are checking for
//	value		: the default value to return if the value is not there
func getEnv(env, value string) string {
	if v := os.Getenv(env); v != "" {
		return v
	}

	return value
}

// fileExists ... checks to see if a file exists
//	filename		: the full path to the file you are checking for
func fileExists(filename string) (bool, error) {
	if _, err := os.Stat(filename); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// writeResourceContent ... is resposinle for generate the specific content from the resource
// 	rn			: a point to the vault resource
//	data		: a map of the related secret associated to the resource
func writeResource(rn *vaultResource, data map[string]interface{}) error {
	var content []byte
	var err error

	// step: determine the resource path
	resourcePath := rn.filename()
	if !strings.HasPrefix(resourcePath, "/") {
		resourcePath = fmt.Sprintf("%s/%s", options.outputDir, resourcePath)
	}

	// step: get the output format
	glog.V(3).Infof("saving resource: %s, format: %s", rn, rn.format)

	if rn.format == "yaml" {
		// marshall the content to yaml
		if content, err = yaml.Marshal(data); err != nil {
			return err
		}

		return writeFile(resourcePath, content)
	}

	if rn.format == "ini" {
		var buf bytes.Buffer
		for key, val := range data {
			buf.WriteString(fmt.Sprintf("%s = %s\n", key, val))
		}
		content = buf.Bytes()

		return writeFile(resourcePath, content)
	}

	if rn.format == "cert" {
		files := map[string]string{
			"certificate": "crt",
			"issuing_ca":  "ca",
			"private_key": "key",
		}
		for key, suffix := range files {
			filename := fmt.Sprintf("%s.%s", resourcePath, suffix)
			content, found := data[key]
			if !found {
				continue
			}

			// step: write the file
			if err := writeFile(filename, []byte(fmt.Sprintf("%s", content))); err != nil {
				glog.Errorf("failed to write resource: %s, elemment: %s, filename: %s, error: %s", rn, suffix, filename, err)
				continue
			}
		}

		return nil
	}

	if rn.format == "csv" {
		var buf bytes.Buffer
		for key, val := range data {
			buf.WriteString(fmt.Sprintf("%s,%s\n", key, val))
		}
		content = buf.Bytes()

		return writeFile(resourcePath, content)
	}

	if rn.format == "txt" {
		keys := getKeys(data)
		if len(keys) > 1 {
			// step: for plain formats we need to iterate the keys and produce a file per key
			for suffix, content := range data {
				filename := fmt.Sprintf("%s.%s", resourcePath, suffix)
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

		return writeFile(resourcePath, content)

	}

	if rn.format == "json" {
		if content, err = json.MarshalIndent(data, "", "    "); err != nil {
			return err
		}

		return writeFile(resourcePath, content)
	}

	return fmt.Errorf("unknown output format: %s", rn.format)
}

// writeFile ... writes the content to a file .. dah
//	filename		: the path to the file
//	content			: the content to be written
func writeFile(filename string, content []byte) error {
	if options.dryRun {
		glog.Infof("dry-run: filename: %s, content:", filename)
		fmt.Printf("%s\n", string(content))
		return nil
	}

	return ioutil.WriteFile(filename, content, 0660)
}
