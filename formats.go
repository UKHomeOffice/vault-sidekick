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
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"text/template"

	"github.com/golang/glog"
	"gopkg.in/yaml.v2"
)

func writeIniFile(filename string, data map[string]interface{}, mode os.FileMode) error {
	var buf bytes.Buffer
	for key, val := range data {
		buf.WriteString(fmt.Sprintf("%s = %v\n", key, val))
	}

	return writeFile(filename, buf.Bytes(), mode)
}

func writeCSVFile(filename string, data map[string]interface{}, mode os.FileMode) error {
	var buf bytes.Buffer
	for key, val := range data {
		buf.WriteString(fmt.Sprintf("%s,%v\n", key, val))
	}

	return writeFile(filename, buf.Bytes(), mode)
}

func writeYAMLFile(filename string, data map[string]interface{}, mode os.FileMode) error {
	// marshall the content to yaml
	content, err := yaml.Marshal(data)
	if err != nil {
		return err
	}

	return writeFile(filename, content, mode)
}

func writeEnvFile(filename string, data map[string]interface{}, mode os.FileMode) error {
	var buf bytes.Buffer
	for key, val := range data {
		buf.WriteString(fmt.Sprintf("%s='%v'\n", strings.ToUpper(key), val))
	}

	return writeFile(filename, buf.Bytes(), mode)
}

func writeCertificateFile(filename string, data map[string]interface{}, mode os.FileMode) error {
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
		if err := writeFile(name, []byte(fmt.Sprintf("%s", content)), mode); err != nil {
			glog.Errorf("failed to write resource: %s, element: %s, filename: %s, error: %s", filename, suffix, name, err)
			continue
		}
	}

	return nil

}

func writeCertificateBundleFile(filename string, data map[string]interface{}, mode os.FileMode) error {
	bundleFile := fmt.Sprintf("%s-bundle.pem", filename)
	keyFile := fmt.Sprintf("%s-key.pem", filename)
	caFile := fmt.Sprintf("%s-ca.pem", filename)
	certFile := fmt.Sprintf("%s.pem", filename)

	bundle := fmt.Sprintf("%s\n\n%s\n\n%s", data["certificate"], data["issuing_ca"], data["private_key"])
	key := fmt.Sprintf("%s\n", data["private_key"])
	ca := fmt.Sprintf("%s\n", data["issuing_ca"])
	certificate := fmt.Sprintf("%s\n", data["certificate"])

	if err := writeFile(bundleFile, []byte(bundle), mode); err != nil {
		glog.Errorf("failed to write the bundled certificate file, error: %s", err)
		return err
	}

	if err := writeFile(certFile, []byte(certificate), mode); err != nil {
		glog.Errorf("failed to write the certificate file, error: %s", err)
		return err
	}

	if err := writeFile(caFile, []byte(ca), mode); err != nil {
		glog.Errorf("failed to write the ca file, error: %s", err)
		return err
	}

	if err := writeFile(keyFile, []byte(key), mode); err != nil {
		glog.Errorf("failed to write the key file, error: %s", err)
		return err
	}

	return nil
}

func writeCertificateChainFile(filename string, data map[string]interface{}, mode os.FileMode) error {
	certChainFile := fmt.Sprintf("%s-cert-chain.pem", filename)
	keyFile := fmt.Sprintf("%s-key.pem", filename)
	caFile := fmt.Sprintf("%s-ca.pem", filename)
	certFile := fmt.Sprintf("%s.pem", filename)

	ca_chain := []string{}

	// the chain should be a slice so assert that the type is []interface
	chain, ok := data["ca_chain"].([]interface{})
	if ok {
		for _, cert := range chain {
			ca_chain = append(ca_chain, fmt.Sprintf("%s", cert))
		}
	} else {
		// In some circumstances we won't have a ca_chain and should fallback to using just the issuing_ca.
		ca_chain = []string{fmt.Sprintf("%s", data["issuing_ca"])}
	}

	certChain := fmt.Sprintf("%s\n\n%s", data["certificate"], strings.Join(ca_chain, "\n"))
	key := fmt.Sprintf("%s\n", data["private_key"])
	ca := fmt.Sprintf("%s\n", data["issuing_ca"])
	certificate := fmt.Sprintf("%s\n", data["certificate"])

	if err := writeFile(certChainFile, []byte(certChain), mode); err != nil {
		glog.Errorf("failed to write the bundle chain certificate file, error: %s", err)
		return err
	}

	if err := writeFile(certFile, []byte(certificate), mode); err != nil {
		glog.Errorf("failed to write the certificate file, error: %s", err)
		return err
	}

	if err := writeFile(caFile, []byte(ca), mode); err != nil {
		glog.Errorf("failed to write the ca file, error: %s", err)
		return err
	}

	if err := writeFile(keyFile, []byte(key), mode); err != nil {
		glog.Errorf("failed to write the key file, error: %s", err)
		return err
	}

	return nil
}

func writeCredentialFile(filename string, data map[string]interface{}, mode os.FileMode) error {
	privateKeyData := fmt.Sprintf("%s", data["private_key_data"])
	key, err := base64.StdEncoding.DecodeString(privateKeyData)
	if err != nil {
		glog.Errorf("failed to decode private key data, error: %s", err)
		return err
	}

	if err := writeFile(filename, key, mode); err != nil {
		glog.Errorf("failed to write the bundled certificate file, error: %s", err)
		return err
	}

	return nil
}

func writeAwsCredentialFile(filename string, data map[string]interface{}, mode os.FileMode) error {
	if err := writeFile(filename, generateAwsCredentialFile(data), mode); err != nil {
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

func writeTxtFile(filename string, data map[string]interface{}, mode os.FileMode) error {
	keys := getKeys(data)
	if len(keys) > 1 {
		// step: for plain formats we need to iterate the keys and produce a file per key
		for suffix, content := range data {
			name := fmt.Sprintf("%s.%s", filename, suffix)
			if err := writeFile(name, []byte(fmt.Sprintf("%v", content)), mode); err != nil {
				glog.Errorf("failed to write resource: %s, elemment: %s, filename: %s, error: %s",
					filename, suffix, name, err)
				continue
			}
		}
		return nil
	}

	// step: we only have the one key, so will write plain
	value, _ := data[keys[0]]
	content := []byte(fmt.Sprintf("%s", value))

	return writeFile(filename, content, mode)
}

func writeRootCAFile(filename string, data map[string]interface{}, mode os.FileMode) error {
	keys := getKeys(data)
	if len(keys) != 1 {
		return errors.New("rootca format is only supported for secrets with a single key")
	}

	// step: we only have the one key, so will write plain
	value, _ := data[keys[0]].(string)
	pemCerts := []byte(value)
	var lastValidBlock *pem.Block

	for len(pemCerts) > 0 {
		var block *pem.Block
		block, pemCerts = pem.Decode(pemCerts)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
			continue
		}

		lastValidBlock = block
	}

	if lastValidBlock == nil {
		return errors.New("no certificate blocks in secret data, cannot write root CA")
	}
	content := pem.EncodeToMemory(lastValidBlock)
	return writeFile(filename, content, mode)
}

func writeJSONFile(filename string, data map[string]interface{}, mode os.FileMode) error {
	content, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}

	return writeFile(filename, content, mode)
}

func writeTemplateFile(filename string, data map[string]interface{}, mode os.FileMode, templateFile string) error {
	tpl := template.Must(template.ParseFiles(templateFile))

	var templateOutput bytes.Buffer
	if err := tpl.Execute(&templateOutput, data); err != nil {
		return err
	}

	content := []byte(fmt.Sprintf("%s", templateOutput.String()))

	return writeFile(filename, content, mode)
}

// writeFile writes the file to stdout or an actual file
func writeFile(filename string, content []byte, mode os.FileMode) error {
	if options.dryRun {
		glog.Infof("dry-run: filename: %s, content:", filename)
		fmt.Printf("%s\n", string(content))
		return nil
	}
	glog.V(3).Infof("saving the file: %s", filename)

	return ioutil.WriteFile(filename, content, mode)
}
