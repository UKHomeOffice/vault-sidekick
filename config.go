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
	"flag"
	"fmt"
	"io/ioutil"
	"net/url"
	"github.com/golang/glog"
	"time"
)

// config ... the command line configuration
type config struct {
	// the url for th vault server
	vaultURL string
	// the token to connect to vault with
	vaultToken string
	// a file / path containing a token for vault
	vaultTokenFile string
	// the place to write the resources
	secretsDirectory string
	// whether or not to renew the leases on our resources
	renewResources bool
	// whether of not to remove the token post connection
	deleteToken bool
	// switch on dry run
	dryRun bool
	// the resource items to retrieve
	resources *vaultResources
	// the interval for producing statistics
	statsInterval time.Duration
}

var (
	options config
)

func init() {
	options.resources = new(vaultResources)
	flag.StringVar(&options.vaultURL, "vault", getEnv("VAULT_ADDR", "https://127.0.0.1:8200"), "the url the vault service is running behind (VAULT_ADDR if available)")
	flag.StringVar(&options.vaultToken, "token", "", "the token used to authenticate to teh vault service (VAULT_TOKEN if available)")
	flag.StringVar(&options.vaultTokenFile, "tokenfile", getEnv("VAULT_TOKEN_FILE", ""), "the full path to file containing the vault token used for authentication (VAULT_TOKEN_FILE if available)")
	flag.StringVar(&options.secretsDirectory, "output", getEnv("VAULT_OUTPUT", "/etc/secrets"), "the full path to write the protected resources (VAULT_OUTPUT if available)")
	flag.BoolVar(&options.deleteToken, "delete-token", false, "once the we have connected to vault, delete the token file from disk")
	flag.BoolVar(&options.renewResources, "renew", true, "whether or not to renew secrets from vault")
	flag.BoolVar(&options.dryRun, "dry-run", false, "perform a dry run, printing the content to screen")
	flag.DurationVar(&options.statsInterval, "stats", time.Duration(5) * time.Minute, "the interval to produce statistics on the accessed resources")
	flag.Var(options.resources, "cn", "a resource to retrieve and monitor from vault (e.g. pki:name:cert.name, secret:db_password, aws:s3_backup)")
}

// parseOptions ... validate the command line options and validates them
func parseOptions() error {
	flag.Parse()

	return validateOptions(&options)
}

// validateOptions ... parses and validates the command line options
func validateOptions(cfg *config) error {
	// step: validate the vault url
	url, err := url.Parse(cfg.vaultURL)
	if err != nil {
		return fmt.Errorf("invalid vault url: '%s' specified", cfg.vaultURL)
	}
	// step: ensure the protocol scheme
	if url.Scheme != "https" {
		glog.Warningf("protocol scheme: %s is not secure and should be https", url.Scheme)
	}

	// step: check if the token is in the VAULT_TOKEN var
	if cfg.vaultToken == "" {
		cfg.vaultToken = getEnv("VAULT_TOKEN", "")
	}

	// step: ensure we have a token
	if cfg.vaultToken == "" && cfg.vaultTokenFile == "" {
		return fmt.Errorf("you have not either a token or a token file to authenticate with")
	}

	// step: read in the token if required
	if cfg.vaultTokenFile != "" {
		exists, err := fileExists(cfg.vaultTokenFile)
		if !exists {
			return fmt.Errorf("the token file: %s does not exists, please check", cfg.vaultTokenFile)
		}
		if err != nil {
			return fmt.Errorf("unable to check for token file: %s, error: %s", cfg.vaultTokenFile, err)
		}
		// step: read in the token
		content, err := ioutil.ReadFile(cfg.vaultTokenFile)
		if err != nil {
			return fmt.Errorf("unable to read in token from file: %s, error: %s", cfg.vaultTokenFile, err)
		}
		cfg.vaultToken = string(content)
	}

	return nil
}
