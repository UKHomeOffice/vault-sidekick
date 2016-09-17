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

	"github.com/hashicorp/vault/api"
)

// the userpass authentication plugin
type authAppRolePlugin struct {
	client *api.Client
}

type appRoleLogin struct {
	RoleId   string `json:"role_id,omitempty"`
	SecretId string `json:"secret_id,omitempty"`
}

// NewAppRolePlugin creates a new App Role plugin
func NewAppRolePlugin(client *api.Client) AuthInterface {
	return &authAppRolePlugin{
		client: client,
	}
}

// Create a approle plugin with the secret id and role id provided in the file
func (r authAppRolePlugin) Create(cfg map[string]string) (string, error) {
	// step: extract the options
	roleId, _ := cfg["role_id"]
	secretId, _ := cfg["secret_id"]

	if roleId == "" {
		roleId = os.Getenv("VAULT_SIDEKICK_ROLE_ID")
	}
	if secretId == "" {
		secretId = os.Getenv("VAULT_SIDEKICK_SECRET_ID")
	}

	// step: create the token request
	request := r.client.NewRequest("POST", "/v1/auth/approle/login")
	login := appRoleLogin{SecretId: secretId, RoleId: roleId}
	if err := request.SetJSONBody(login); err != nil {
		return "", err
	}
	// step: make the request
	resp, err := r.client.RawRequest(request)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// step: parse and return auth
	secret, err := api.ParseSecret(resp.Body)
	if err != nil {
		return "", err
	}

	return secret.Auth.ClientToken, nil
}
