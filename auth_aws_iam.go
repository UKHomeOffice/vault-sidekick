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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/hashicorp/vault/api"
)

// aws iam authentication plugin
type authAWSIAMPlugin struct {
	// the vault client
	client *api.Client
}

// NewUserTokenPlugin creates a new User Token plugin
func NewAWSIAMPlugin(client *api.Client) AuthInterface {
	return &authAWSIAMPlugin{
		client: client,
	}
}

// Create retrieves the token from an environment variable or file
func (r authAWSIAMPlugin) Create(cfg *vaultAuthOptions) (string, error) {
	role := os.Getenv("VAULT_SIDEKICK_ROLE_ID")
	if cfg.FileName != "" {
		content, err := readConfigFile(cfg.FileName, cfg.FileFormat)
		if err != nil {
			return "", err
		}

		role = content.RoleID
	}

	creds, err := generateCredentialChain()
	if err != nil {
		return "", err
	}

	loginData, err := generateLoginData(creds)
	if err != nil {
		return "", err
	}
	if loginData == nil {
		return "", fmt.Errorf("got nil response from GenerateLoginData")
	}
	loginData["role"] = role
	path := fmt.Sprintf("auth/aws/login")
	resp, err := r.client.Logical().Write(path, loginData)

	if err != nil {
		return "", err
	}
	if resp == nil {
		return "", fmt.Errorf("empty response from credential provider")
	}

	return resp.Auth.ClientToken, nil
}

// generateLoginData populates the necessary data to send to the Vault server for generating a token
// from github.com/hashicorp/vault/builtin/credential/aws/cli.go
func generateLoginData(creds *credentials.Credentials) (map[string]interface{}, error) {
	loginData := make(map[string]interface{})

	// Use the credentials we've found to construct an STS session
	stsSession, err := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			Credentials: creds,
		},
	})
	if err != nil {
		return nil, err
	}

	var params *sts.GetCallerIdentityInput
	svc := sts.New(stsSession)
	stsRequest, _ := svc.GetCallerIdentityRequest(params)

	if err := stsRequest.Sign(); err != nil {
		return nil, err
	}

	// Now extract out the relevant parts of the request
	headersJson, err := json.Marshal(stsRequest.HTTPRequest.Header)
	if err != nil {
		return nil, err
	}
	requestBody, err := ioutil.ReadAll(stsRequest.HTTPRequest.Body)
	if err != nil {
		return nil, err
	}
	loginData["iam_http_request_method"] = stsRequest.HTTPRequest.Method
	loginData["iam_request_url"] = base64.StdEncoding.EncodeToString([]byte(stsRequest.HTTPRequest.URL.String()))
	loginData["iam_request_headers"] = base64.StdEncoding.EncodeToString(headersJson)
	loginData["iam_request_body"] = base64.StdEncoding.EncodeToString(requestBody)

	return loginData, nil
}

// from github.com/hashicorp/vault/helper/awsutil/generate_credentials.go
func generateCredentialChain() (*credentials.Credentials, error) {
	var providers []credentials.Provider

	// Add the environment credential provider
	providers = append(providers, &credentials.EnvProvider{})

	// Add the shared credentials provider
	providers = append(providers, &credentials.SharedCredentialsProvider{})

	// Add the remote provider
	def := defaults.Get()

	providers = append(providers, defaults.RemoteCredProvider(*def.Config, def.Handlers))

	// Create the credentials required to access the API.
	creds := credentials.NewChainCredentials(providers)
	if creds == nil {
		return nil, fmt.Errorf("could not compile valid credential providers from environment, shared, or instance metadata")
	}

	_, err := creds.Get()
	if err != nil {
		return nil, err
	}

	return creds, nil
}
