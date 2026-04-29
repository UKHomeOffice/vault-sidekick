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

// IBMCloud auth plugin
type authIBMCloudPlugin struct {
	// vault client
	client *api.Client
}

type IBMCloudLogin struct {
	Token string `json:"token,omitempty"`
}

// Create a new IBMCloud plugin
func NewIBMCloudPlugin(client *api.Client) AuthInterface {
	return &authIBMCloudPlugin{
		client: client,
	}
}

func (r authIBMCloudPlugin) Create(cfg *vaultAuthOptions) (string, error) {
	loginPath := "/v1/auth/ibmcloud/login"
	// TODO read token from config.json instead
	iamToken := os.Getenv("IAM_TOKEN")
	if iamToken == "" {
		return "", fmt.Errorf("Miss IAM token")
	}

	// build the token request
	request := r.client.NewRequest("PUT", loginPath)
	login := IBMCloudLogin{Token: string(iamToken)}
	if err := request.SetJSONBody(login); err != nil {
		return "", err
	}

	// send the request to Vault
	resp, err := r.client.RawRequest(request)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// parse the auth object into something useful
	secret, err := api.ParseSecret(resp.Body)
	if err != nil {
		return "", err
	}

	return secret.Auth.ClientToken, nil
}
