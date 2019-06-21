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
	"regexp"
	"time"
)

const (
	// optionFilename option to set the filename of the resource
	optionFilename = "file"
	// optionFormat set the output format (yaml, xml, json)
	optionFormat = "fmt"
	// optionTemplatePath is the full path to a template
	optionTemplatePath = "tpl"
	// optionRenewal sets the duration to renew the resource
	optionRenewal = "renew"
	// optionRevoke revokes an old lease when retrieving a new one
	optionRevoke = "revoke"
	// optionRevokeDelay
	optionsRevokeDelay = "delay"
	// optionUpdate overrides the lease of the resource
	optionUpdate = "update"
	// optionsExec executes something on a change
	optionExec = "exec"
	// optionCreate creates a secret if it doesn't exist
	optionCreate = "create"
	// optionSize sets the initial size of a password secret
	optionSize = "size"
	// optionsMode is the file permissions on the secret
	optionMode = "mode"
	// optionMaxRetries is the maximum number of retries that should be attempted
	optionMaxRetries = "retries"
	// optionMaxJitter is the maximum amount of jitter that should be applied
	// to updates for this resource. If non-zero, a random value between 0 and
	// maxJitter will be subtracted from the update period.
	optionMaxJitter = "jitter"
	// defaultSize sets the default size of a generic secret
	defaultSize = 20
)

var (
	resourceFormatRegex = regexp.MustCompile("^(yaml|yml|json|env|ini|txt|rootca|cert|certchain|bundle|csv|template|credential|aws)$")

	// a map of valid resource to retrieve from vault
	validResources = map[string]bool{
		"raw":       true,
		"pki":       true,
		"aws":       true,
		"gcp":       true,
		"secret":    true,
		"mysql":     true,
		"tpl":       true,
		"postgres":  true,
		"transit":   true,
		"cubbyhole": true,
		"cassandra": true,
		"ssh":       true,
		"database":  true,
	}
)

func defaultVaultResource() *VaultResource {
	return &VaultResource{
		FileMode:  os.FileMode(0664),
		Format:    "yaml",
		Options:   make(map[string]string, 0),
		Renewable: false,
		Revoked:   false,
		Size:      defaultSize,
	}
}

// VaultResource is the structure which defined a resource set from vault
type VaultResource struct {
	// the namespace of the resource
	Resource string
	// the name of the resource
	Path string
	// the format of the resource
	Format string
	// whether the resource should be renewed?
	Renewable bool
	// whether the resource should be revoked?
	Revoked bool
	// the revoke delay
	RevokeDelay time.Duration
	// the lease duration
	Update time.Duration
	// whether the resource should be created?
	Create bool
	// the size of a secret to create
	Size int64
	// the filename to save the secret
	Filename string
	// the template file
	TemplateFile string
	// the path to an exec to run on a change
	ExecPath []string
	// additional options to the resource
	Options map[string]string
	// the file permissions on the resource
	FileMode os.FileMode
	// maxRetries is the maximum number of times this resource should be
	// attempted to be retrieved from Vault before failing
	MaxRetries int
	// retries is the number of times this resource has been retried since it
	// last succeeded
	Retries int
	// maxJitter is the maximum jitter duration to use for this resource when
	// performing renewals
	MaxJitter time.Duration
}

// GetFilename generates a resource filename by default the resource name and resource type, which
// can override by the OPTION_FILENAME option
func (r VaultResource) GetFilename() string {
	if r.Filename != "" {
		return r.Filename
	}

	return fmt.Sprintf("%s.%s", r.Path, r.Resource)
}

// IsValid checks to see if the resource is valid
func (r *VaultResource) IsValid() error {
	// step: check the resource type
	if _, found := validResources[r.Resource]; !found {
		return fmt.Errorf("unsupported resource type: %s", r.Resource)
	}

	// step: check is have all the required options to this resource type
	if err := r.isValidResource(); err != nil {
		return fmt.Errorf("invalid resource: %s, %s", r, err)
	}

	return nil
}

// isValidResource validates the resource meets the requirements
func (r *VaultResource) isValidResource() error {
	switch r.Resource {
	case "pki":
		if _, found := r.Options["common_name"]; !found {
			return fmt.Errorf("pki resource requires a common name specified")
		}
	case "transit":
		if _, found := r.Options["ciphertext"]; !found {
			return fmt.Errorf("transit requires a ciphertext option")
		}
	case "tpl":
		if _, found := r.Options[optionTemplatePath]; !found {
			return fmt.Errorf("template resource requires a template path option")
		}
	case "ssh":
		if _, found := r.Options["public_key_path"]; !found {
			return fmt.Errorf("ssh resource requires a public key file path specified")
		}
		if _, found := r.Options["cert_type"]; !found {
			return fmt.Errorf("ssh resource requires cert_type to be either host or user")
		}
	}

	return nil
}

// String returns a string representation of the struct
func (r VaultResource) String() string {
	str := fmt.Sprintf("type: %s, path: %s", r.Resource, r.Path)
	if r.MaxRetries > 0 {
		str = fmt.Sprintf("%s, attempts: %d/%d", str, r.Retries, r.MaxRetries+1)
	}
	return str
}

func (r VaultResource) ID() string {
	return r.Path
}
