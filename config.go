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
	"net/url"
	"os"
	"strconv"
	"time"
)

type vaultAuthOptions struct {
	ClientToken   string
	Token         string
	LeaseDuration int
	Renewable     bool
	Method        string
	VaultURL      string `json:"vaultAddr"`
	RoleID        string `json:"role_id" yaml:"role_id"`
	SecretID      string `json:"secret_id" yaml:"secret_id"`
	FileName      string
	FileFormat    string
	Username      string
	Password      string
}

type config struct {
	// the url for th vault server
	vaultURL string
	// a file containing the authenticate options
	vaultAuthFile string
	// whether or not the auth file format is default
	vaultAuthFileFormat string
	// the authentication options
	vaultAuthOptions *vaultAuthOptions
	// renew the token based on ttl
	vaultRenewToken bool
	// the vault ca file
	vaultCaFile string
	// the place to write the resources
	outputDir string
	// switch on dry run
	dryRun bool
	// skip tls verify
	skipTLSVerify bool
	// the resource items to retrieve
	resources *VaultResources
	// the interval for producing statistics
	statsInterval time.Duration
	// the timeout for a exec command
	execTimeout time.Duration
	// version flag
	showVersion bool
	// one-shot mode
	oneShot bool
}

var (
	options config
)

func init() {
	// step: setup some defaults
	options.resources = new(VaultResources)
	authMethod := getEnv("VAULT_AUTH_METHOD", "token")
	options.vaultAuthOptions = &vaultAuthOptions{
		Method: authMethod,
	}

	defaultRenewToken, err := strconv.ParseBool(getEnv("VAULT_SIDEKICK_RENEW_TOKEN", "false"))
	if err != nil {
		defaultRenewToken = false
	}

	defaultDryRun, err := strconv.ParseBool(getEnv("VAULT_SIDEKICK_DRY_RUN", "false"))
	if err != nil {
		defaultDryRun = false
	}

	defaultSkipTLSVerify, err := strconv.ParseBool(getEnv("VAULT_SIDEKICK_SKIP_TLS_VERIFY", "false"))
	if err != nil {
		defaultSkipTLSVerify = false
	}

	defaultStatsInterval, err := time.ParseDuration(getEnv("VAULT_SIDEKICK_STATS_INTERVAL", "1h"))
	if err != nil {
		defaultStatsInterval = time.Duration(1) * time.Hour
	}

	defaultExecTimeout, err := time.ParseDuration(getEnv("VAULT_SIDEKICK_EXEC_TIMEOUT", "60s"))
	if err != nil {
		defaultExecTimeout = time.Duration(60) * time.Second
	}

	defaultOneShot, err := strconv.ParseBool(getEnv("VAULT_SIDEKICK_ONE_SHOT", "false"))
	if err != nil {
		defaultOneShot = false
	}

	flag.StringVar(&options.vaultURL, "vault", getEnv("VAULT_ADDR", "https://127.0.0.1:8200"), "url the vault service or VAULT_ADDR")
	flag.StringVar(&options.vaultAuthFile, "auth", getEnv("AUTH_FILE", ""), "a configuration file in json or yaml containing authentication arguments")
	flag.BoolVar(&options.vaultRenewToken, "renew-token", defaultRenewToken, "renew vault token according to its ttl")
	flag.StringVar(&options.vaultAuthFileFormat, "format", getEnv("AUTH_FORMAT", "default"), "the auth file format")
	flag.StringVar(&options.outputDir, "output", getEnv("VAULT_OUTPUT", "/etc/secrets"), "the full path to write resources or VAULT_OUTPUT")
	flag.BoolVar(&options.dryRun, "dryrun", defaultDryRun, "perform a dry run, printing the content to screen")
	flag.BoolVar(&options.skipTLSVerify, "tls-skip-verify", defaultSkipTLSVerify, "whether to check and verify the vault service certificate")
	flag.StringVar(&options.vaultCaFile, "ca-cert", getEnv("VAULT_SIDEKICK_CA_CERT", ""), "the path to the file container the CA used to verify the vault service")
	flag.DurationVar(&options.statsInterval, "stats", defaultStatsInterval, "the interval to produce statistics on the accessed resources")
	flag.DurationVar(&options.execTimeout, "exec-timeout", defaultExecTimeout, "the timeout applied to commands on the exec option")
	flag.BoolVar(&options.showVersion, "version", false, "show the vault-sidekick version")
	flag.Var(options.resources, "cn", "a resource to retrieve and monitor from vault")
	flag.BoolVar(&options.oneShot, "one-shot", defaultOneShot, "retrieve resources from vault once and then exit")
}

// parseOptions validate the command line options and validates them
func parseOptions() error {
	flag.Parse()
	return validateOptions(&options)
}

// validateOptions parses and validates the command line options
func validateOptions(cfg *config) (err error) {
	// step: read in the token if required

	if cfg.vaultAuthFile != "" {
		if exists, _ := fileExists(cfg.vaultAuthFile); !exists {
			return fmt.Errorf("the token file: %s does not exists, please check", cfg.vaultAuthFile)
		}

		cfg.vaultAuthOptions, err = readConfigFile(cfg.vaultAuthFile, cfg.vaultAuthFileFormat)
		if err != nil {
			return fmt.Errorf("unable to read in authentication options from: %s, error: %s", cfg.vaultAuthFile, err)
		}
		if cfg.vaultAuthOptions.VaultURL != "" {
			cfg.vaultURL = cfg.vaultAuthOptions.VaultURL
		}
	}

	if cfg.vaultURL == "" {
		cfg.vaultURL = os.Getenv("VAULT_ADDR")
	}

	if cfg.vaultURL == "" {
		return fmt.Errorf("VAULT_ADDR is unset")
	}

	// step: validate the vault url
	if _, err = url.Parse(cfg.vaultURL); err != nil {
		return fmt.Errorf("invalid vault url: '%s' specified", cfg.vaultURL)
	}

	if cfg.vaultCaFile != "" {
		if exists, _ := fileExists(cfg.vaultCaFile); !exists {
			return fmt.Errorf("the ca certificate file: %s does not exist", cfg.vaultCaFile)
		}
	}

	if cfg.skipTLSVerify == true && cfg.vaultCaFile != "" {
		return fmt.Errorf("you are skipping the tls but supplying a CA, doesn't make sense")
	}

	return nil
}
