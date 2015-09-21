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

	"github.com/hashicorp/vault/api"
	"github.com/golang/glog"
)

// the userpass authentication plugin
type authUserPass struct {
	// the vault client
	client *api.Client
}

// auth token
type UserPassLogin struct {
	// the password for the account
	Password string `json:"password,omitempty"`
}

func newUserPass(client *api.Client) *authUserPass {
	return &authUserPass{
		client: client,
	}
}

// create ... login with the username and password an
func (r authUserPass) create(username, password string) (*api.Secret, error) {
	glog.V(10).Infof("using the userpass plugin, username: %s, password: %s", username, password)

	req := r.client.NewRequest("POST", fmt.Sprintf("/v1/auth/userpass/login/%s", username))
	// step: create the token request
	if err := req.SetJSONBody(UserPassLogin{Password: password}); err != nil {
		return nil, err
	}
	// step: make the request
	resp, err := r.client.RawRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// step: parse and return auth
	return api.ParseSecret(resp.Body)
}
