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
	"strings"

	"github.com/golang/glog"
	"gopkg.in/yaml.v2"
)

func writeIniFile(filename string, data map[string]interface{}) error {
	var buf bytes.Buffer
	for key, val := range data {
		buf.WriteString(fmt.Sprintf("%s = %v\n", key, val))
	}

	return writeFile(filename, buf.Bytes())
}

func writeCSVFile(filename string, data map[string]interface{}) error {
	var buf bytes.Buffer
	for key, val := range data {
		buf.WriteString(fmt.Sprintf("%s,%v\n", key, val))
	}

	return writeFile(filename, buf.Bytes())
}

func writeYAMLFile(filename string, data map[string]interface{}) error {
	// marshall the content to yaml
	content, err := yaml.Marshal(data)
	if err != nil {
		return err
	}

	return writeFile(filename, content)
}

func writeEnvFile(filename string, data map[string]interface{}) error {
	var buf bytes.Buffer
	for key, val := range data {
		buf.WriteString(fmt.Sprintf("%s=%v\n", strings.ToUpper(key), val))
	}

	return writeFile(filename, buf.Bytes())
}

func writeCertificateFile(filename string, data map[string]interface{}) error {
	files := map[string]string{
		"certificate": "crt",
		"issuing_ca":  "ca",
		"private_key": "key",
	}
	for key, suffix := range files {
		filename := fmt.Sprintf("%s.%s", filename, suffix)
		content, found := data[key]
		if !found {
			glog.Errorf("didn't find the certification option: %s in the resource: %s", key, filename)
			continue
		}

		// step: write the file
		if err := writeFile(filename, []byte(fmt.Sprintf("%s", content))); err != nil {
			glog.Errorf("failed to write resource: %s, elemment: %s, filename: %s, error: %s", filename, suffix, filename, err)
			continue
		}
	}

	return nil

}

func writeCertificateBundleFile(filename string, data map[string]interface{}) error {
	bundleFile := fmt.Sprintf("%s-bundle.pem", filename)
	keyFile := fmt.Sprintf("%s-key.pem", filename)
	caFile := fmt.Sprintf("%s-ca.pem", filename)
	certFile := fmt.Sprintf("%s.pem", filename)

	bundle := fmt.Sprintf("%s\n\n%s", data["certificate"], data["issuing_ca"])
	key := fmt.Sprintf("%s\n", data["private_key"])
	ca := fmt.Sprintf("%s\n", data["issuing_ca"])
	certificate := fmt.Sprintf("%s\n", data["certificate"])

	if err := writeFile(bundleFile, []byte(bundle)); err != nil {
		glog.Errorf("failed to write the bundled certificate file, error: %s", err)
		return err
	}

	if err := writeFile(certFile, []byte(certificate)); err != nil {
		glog.Errorf("failed to write the certificate file, errro: %s", err)
		return err
	}

	if err := writeFile(caFile, []byte(ca)); err != nil {
		glog.Errorf("failed to write the ca file, errro: %s", err)
		return err
	}

	if err := writeFile(keyFile, []byte(key)); err != nil {
		glog.Errorf("failed to write the key file, errro: %s", err)
		return err
	}

	return nil
}

func writeTxtFile(filename string, data map[string]interface{}) error {
	keys := getKeys(data)
	if len(keys) > 1 {
		// step: for plain formats we need to iterate the keys and produce a file per key
		for suffix, content := range data {
			filename := fmt.Sprintf("%s.%s", filename, suffix)
			if err := writeFile(filename, []byte(fmt.Sprintf("%v", content))); err != nil {
				glog.Errorf("failed to write resource: %s, elemment: %s, filename: %s, error: %s",
					filename, suffix, filename, err)
				continue
			}
		}
		return nil
	}

	// step: we only have the one key, so will write plain
	value, _ := data[keys[0]]
	content := []byte(fmt.Sprintf("%s", value))

	return writeFile(filename, content)
}

func writeJSONFile(filename string, data map[string]interface{}) error {
	content, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}

	return writeFile(filename, content)
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
	glog.V(3).Infof("saving the file: %s", filename)

	return ioutil.WriteFile(filename, content, 0664)
}
