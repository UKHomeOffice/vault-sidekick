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
	"fmt"

	"github.com/hashicorp/vault/api"
)

// gcp authentication plugin
type authGCPGCEPlugin struct {
	// the vault client
	client *api.Client
}

// NewGCPGCEPlugin creates a new User Token plugin
func NewGCPGCEPlugin(client *api.Client) AuthInterface {
	return &authGCPGCEPlugin{
		client: client,
	}
}

// Create retrieves the token from an environment variable or file
func (r authGCPGCEPlugin) Create(cfg *vaultAuthOptions) (string, error) {
	role := os.Getenv("VAULT_SIDEKICK_ROLE_ID")
	if cfg.FileName != "" {
		content, err := readConfigFile(cfg.FileName, cfg.FileFormat)
		if err != nil {
			return "", err
		}

		role = content.RoleID
	}

	jwtToken, err := getGCPServiceAccountToken(role)
	if err != nil {
		return "", err
	}
	payload := map[string]interface{}{
		"role": role,
		"jwt": string(jwtToken),
	}

	resp, err := r.client.Logical().Write("auth/gcp/login", payload)
	if err != nil {
		return "", err
	}

	return resp.Auth.ClientToken, nil
}

// getGCPServiceAccountToken retrieves a JWT token from GCP metadata service
func getGCPServiceAccountToken(role string) ([]byte, error) {
	// Vault GCP auth backend only parses vault/<role> from aud
	url := fmt.Sprintf("http://metadata/computeMetadata/v1/instance/service-accounts/default/identity?audience=http://localhost/vault/%s&format=full", role)
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Metadata-Flavor", "Google")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}
