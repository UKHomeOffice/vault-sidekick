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
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/golang/glog"

	"github.com/hashicorp/vault/api"
)

// Kubernetes auth plugin
type authKubernetesPlugin struct {
	// vault client
	client *api.Client
}

type kubernetesLogin struct {
	Role string `json:"role,omitempty"`
	Jwt  string `json:"jwt,omitempty"`
}

// Create a new Kubernetes plugin
func NewKubernetesPlugin(client *api.Client) AuthInterface {
	return &authKubernetesPlugin{
		client: client,
	}
}

func (r authKubernetesPlugin) Create(cfg *vaultAuthOptions) (string, error) {
	vaultRole, ok := os.LookupEnv("VAULT_SIDEKICK_ROLE")

	if !ok {
		return "", fmt.Errorf("VAULT_SIDEKICK_ROLE not provided")
	}

	// in case you mounted your kubernetes auth engine somewhere else
	loginPath := getEnv("VAULT_K8S_LOGIN_PATH", "/v1/auth/kubernetes/login")

	tokenPath := getEnv("VAULT_K8S_TOKEN_PATH", "/var/run/secrets/kubernetes.io/serviceaccount/token")

	// read the JWT from the token file
	token, err := ioutil.ReadFile(tokenPath)
	if err != nil {
		glog.Error("Error reading token file: ", err.Error())
		return "", err
	}

	glog.Infof("Requesting for role %s vault-token..", vaultRole)

	secret, err := r.client.Logical().Write(path.Join(loginPath, "login"), map[string]interface{}{
		"jwt":  string(bytes.TrimSpace(token)),
		"role": vaultRole,
	})
	if err != nil {
		glog.Fatal(err.Error())
	}

	return secret.Auth.ClientToken, nil
}
