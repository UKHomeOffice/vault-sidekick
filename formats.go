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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"text/template"

	"github.com/golang/glog"
	"gopkg.in/yaml.v2"
)

func writeIniFile(filename string, data map[string]interface{}, mode os.FileMode, append bool) error {
	var buf bytes.Buffer
	for key, val := range data {
		buf.WriteString(fmt.Sprintf("%s = %v\n", key, val))
	}

	return writeFile(filename, buf.Bytes(), mode, append)
}

func writeCSVFile(filename string, data map[string]interface{}, mode os.FileMode, append bool) error {
	var buf bytes.Buffer
	for key, val := range data {
		buf.WriteString(fmt.Sprintf("%s,%v\n", key, val))
	}

	return writeFile(filename, buf.Bytes(), mode, append)
}

func writeYAMLFile(filename string, data map[string]interface{}, mode os.FileMode, append bool) error {
	// marshall the content to yaml
	content, err := yaml.Marshal(data)
	if err != nil {
		return err
	}

	return writeFile(filename, content, mode, append)
}

func writeEnvFile(filename string, data map[string]interface{}, mode os.FileMode, append bool) error {
	var buf bytes.Buffer
	for key, val := range data {
		buf.WriteString(fmt.Sprintf("%s='%v'\n", strings.ToUpper(key), val))
	}

	return writeFile(filename, buf.Bytes(), mode, append)
}

func writeCAChain(filename string, data map[string]interface{}, mode os.FileMode, append bool) error {
	const element = "ca_chain"
	const suffix = "ca"

	// the chain should be a slice so assert that the type is []interface
	chain, ok := data[element].([]interface{})
	if !ok {
		glog.Errorf("didn't find the certification option: %s", element)
		return nil
	}

	name := fmt.Sprintf("%s.%s.%s", filename, element, suffix)

	certChain := ""
	for count, cert := range chain {
		certChain += fmt.Sprintf("%s", cert)
		// append a newline after each cert except last
		if count < len(chain)-1 {
			certChain += "\n"
		}
	}

	if err := writeFile(name, []byte(fmt.Sprintf("%s", certChain)), mode, append); err != nil {
		return fmt.Errorf("failed to write resource: %s, element: %s, filename: %s, error: %s", filename, suffix, name, err)
	}

	return nil
}
func writeCertificateFile(filename string, data map[string]interface{}, mode os.FileMode, append bool) error {
	if err := writeCAChain(filename, data, mode, append); err != nil {
		glog.Errorf("failed to write CA chain: %s", err)
	}

	files := map[string]string{
		"certificate": "crt",
		"issuing_ca":  "ca",
		"private_key": "key",
	}
	for key, suffix := range files {
		name := fmt.Sprintf("%s.%s", filename, suffix)
		content, found := data[key]
		if !found {
			glog.Errorf("didn't find the certification option: %s in the resource: %s", key, name)
			continue
		}

		// step: write the file
		if err := writeFile(name, []byte(fmt.Sprintf("%s", content)), mode, append); err != nil {
			glog.Errorf("failed to write resource: %s, element: %s, filename: %s, error: %s", filename, suffix, name, err)
			continue
		}
	}

	return nil

}

func writeCertificateBundleFile(filename string, data map[string]interface{}, mode os.FileMode, append bool) error {
	bundleFile := fmt.Sprintf("%s-bundle.pem", filename)
	keyFile := fmt.Sprintf("%s-key.pem", filename)
	caFile := fmt.Sprintf("%s-ca.pem", filename)
	certFile := fmt.Sprintf("%s.pem", filename)

	bundle := fmt.Sprintf("%s\n\n%s\n\n%s", data["certificate"], data["issuing_ca"], data["private_key"])
	key := fmt.Sprintf("%s\n", data["private_key"])
	ca := fmt.Sprintf("%s\n", data["issuing_ca"])
	certificate := fmt.Sprintf("%s\n", data["certificate"])

	if err := writeFile(bundleFile, []byte(bundle), mode, append); err != nil {
		glog.Errorf("failed to write the bundled certificate file, error: %s", err)
		return err
	}

	if err := writeFile(certFile, []byte(certificate), mode, append); err != nil {
		glog.Errorf("failed to write the certificate file, errro: %s", err)
		return err
	}

	if err := writeFile(caFile, []byte(ca), mode, append); err != nil {
		glog.Errorf("failed to write the ca file, errro: %s", err)
		return err
	}

	if err := writeFile(keyFile, []byte(key), mode, append); err != nil {
		glog.Errorf("failed to write the key file, errro: %s", err)
		return err
	}

	return nil
}

func writeCredentialFile(filename string, data map[string]interface{}, mode os.FileMode, append bool) error {
	privateKeyData := fmt.Sprintf("%s", data["private_key_data"])
	key, err := base64.StdEncoding.DecodeString(privateKeyData)
	if err != nil {
		glog.Errorf("failed to decode private key data, error: %s", err)
		return err
	}

	if err := writeFile(filename, key, mode, append); err != nil {
		glog.Errorf("failed to write the bundled certificate file, error: %s", err)
		return err
	}

	return nil
}

func writeAwsCredentialFile(filename string, data map[string]interface{}, mode os.FileMode, append bool) error {
	if err := writeFile(filename, generateAwsCredentialFile(data), mode, append); err != nil {
		glog.Errorf("failed to write aws credentials file, error: %s", err)
		return err
	}
	return nil
}

func generateAwsCredentialFile(data map[string]interface{}) []byte {
	const profileName = "[default]"
	accessKey := fmt.Sprintf("aws_access_key_id=%s", data["access_key"])
	secretKey := fmt.Sprintf("aws_secret_access_key=%s", data["secret_key"])

	// Credentials of type IAM User do not have a security token, and are returned as nil
	if data["security_token"] != nil {
		sessionToken := fmt.Sprintf("aws_session_token=%s", data["security_token"])

		// Support clients that are using boto
		securityToken := fmt.Sprintf("aws_security_token=%s", data["security_token"])

		return []byte(fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n",
			profileName, accessKey, secretKey, securityToken, sessionToken))
	}

	return []byte(fmt.Sprintf("%s\n%s\n%s\n", profileName, accessKey, secretKey))
}

func writeTxtFile(filename string, data map[string]interface{}, mode os.FileMode, append bool) error {
	keys := getKeys(data)
	if len(keys) > 1 {
		// step: for plain formats we need to iterate the keys and produce a file per key
		for suffix, content := range data {
			name := fmt.Sprintf("%s.%s", filename, suffix)
			if err := writeFile(name, []byte(fmt.Sprintf("%v", content)), mode, append); err != nil {
				glog.Errorf("failed to write resource: %s, element: %s, filename: %s, error: %s",
					filename, suffix, name, err)
				continue
			}
		}
		return nil
	}

	// step: we only have the one key, so will write plain
	value, _ := data[keys[0]]
	content := []byte(fmt.Sprintf("%s", value))

	return writeFile(filename, content, mode, append)
}

func writeJSONFile(filename string, data map[string]interface{}, mode os.FileMode, append bool) error {
	content, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}

	return writeFile(filename, content, mode, append)
}

func writeTemplateFile(filename string, data map[string]interface{}, mode os.FileMode, templateFile string, append bool) error {
	tpl := template.Must(template.ParseFiles(templateFile))

	var templateOutput bytes.Buffer
	if err := tpl.Execute(&templateOutput, data); err != nil {
		return err
	}

	content := []byte(fmt.Sprintf("%s", templateOutput.String()))

	return writeFile(filename, content, mode, append)
}

// writeFile writes the file to stdout or an actual file
func writeFile(filename string, content []byte, mode os.FileMode, append bool) error {
	if options.dryRun {
		glog.Infof("dry-run: filename: %s, content:", filename)
		fmt.Printf("%s\n", string(content))
		return nil
	}
	glog.V(3).Infof("saving the file: %s", filename)

	if append == true {
		return appendFile(filename, content, mode)
	}

	return ioutil.WriteFile(filename, content, mode)
}

// appendFile writes data to a file named by filename.
// If the file does not exist, appendFile creates it with permissions perm;
// otherwise appendFile appends to it.
func appendFile(filename string, data []byte, perm os.FileMode) error {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, perm)
	if err != nil {
		return err
	}
	n, err := f.Write(data)
	if err == nil && n < len(data) {
		err = io.ErrShortWrite
	}
	if err1 := f.Close(); err == nil {
		err = err1
	}
	return err
}
