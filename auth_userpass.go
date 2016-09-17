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

	"github.com/hashicorp/vault/api"
)

// the userpass authentication plugin
type authUserPassPlugin struct {
	client *api.Client
}

type userPassLogin struct {
	// the password for the account
	Password string `json:"password,omitempty"`
}

// NewUserPassPlugin creates a new User Pass plugin
func NewUserPassPlugin(client *api.Client) AuthInterface {
	return &authUserPassPlugin{
		client: client,
	}
}

// Create a userpass plugin with the username and password provide in the file
func (r authUserPassPlugin) Create(cfg map[string]string) (string, error) {
	// step: extract the options
	username, _ := cfg["username"]
	password, _ := cfg["password"]

	if username == "" {
		username = os.Getenv("VAULT_SIDEKICK_USERNAME")
	}
	if password == "" {
		password = os.Getenv("VAULT_SIDEKICK_PASSWORD")
	}

	// step: create the token request
	request := r.client.NewRequest("POST", fmt.Sprintf("/v1/auth/userpass/login/%s", username))
	if err := request.SetJSONBody(userPassLogin{Password: password}); err != nil {
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
