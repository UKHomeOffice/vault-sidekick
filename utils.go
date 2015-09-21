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
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"
"io/ioutil"
	"encoding/json"
	"gopkg.in/yaml.v2"
	"path"
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
func hasKey(key string, data map [string]interface{}) bool {
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
		return readJsonFile(filename)
	case ".yaml":
		return readYamlFile(filename)
	case ".yml":
		return readYamlFile(filename)
	}
	return nil, fmt.Errorf("unsupported config file format: %s", suffix)
}

// readJsonFile ... read in and unmarshall the data into a map
//	filename	: the path to the file container the json data
func readJsonFile(filename string) (map[string]string, error) {
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
