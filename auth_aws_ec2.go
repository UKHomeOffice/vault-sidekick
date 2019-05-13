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
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/hashicorp/vault/api"
)

// aws ec2 authentication plugin
type authAWSEC2Plugin struct {
	// the vault client
	client *api.Client
}

// NewUserTokenPlugin creates a new User Token plugin
func NewAWSEC2Plugin(client *api.Client) AuthInterface {
	return &authAWSEC2Plugin{
		client: client,
	}
}

// Create retrieves the token from an environment variable or file
func (r authAWSEC2Plugin) Create(cfg *vaultAuthOptions) (string, error) {
	role := os.Getenv("VAULT_SIDEKICK_ROLE_ID")
	if cfg.FileName != "" {
		content, err := readConfigFile(cfg.FileName, cfg.FileFormat)
		if err != nil {
			return "", err
		}

		role = content.RoleID
	}

	identity, err := getAWSIdentityDocument()
	if err != nil {
		return "", err
	}
	pkcs := strings.Replace(string(identity), "\n", "", -1)
	payload := map[string]interface{}{
		"role":  role,
		"pkcs7": pkcs,
	}

	nonceFile := os.Getenv("VAULT_SIDEKICK_NONCE_FILE")
	nonce, err := ioutil.ReadFile(nonceFile)
	if err != nil {
		return "", err
	}
	if string(nonce) != "" {
		payload["nonce"] = string(nonce)
	}

	resp, err := r.client.Logical().Write("auth/aws/login", payload)
	if err != nil {
		return "", err
	}

	return resp.Auth.ClientToken, nil
}

func getAWSIdentityDocument() ([]byte, error) {
	resp, err := http.Get("http://169.254.169.254/latest/dynamic/instance-identity/pkcs7")
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}
