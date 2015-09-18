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
	"regexp"
	"time"
	"strconv"
"github.com/golang/glog"
)

const (
	// OptionFilename ... option to set the filename of the resource
	OptionFilename = "fn"
	// OptionsFormat ... option to set the output format (yaml, xml, json)
	OptionFormat  = "fmt"
	// OptionsCommonName ... use by the PKI resource
	OptionCommonName = "cn"
	// OptionTemplatePath ... the full path to a template
	OptionsTemplatePath = "tpl"
	// OptionRenew ... a duration to renew the resource
	OptionRenewal = "rn"
	// OptionRevoke ... revoke an old lease when retrieving a new one
	OptionRevoke = "rv"
	// OptionUpdate ... override the lease of the resource
	OptionUpdate = "up"

	DefaultRenewable = "false"
)

var (
	resourceFormatRegex = regexp.MustCompile("^(yaml|json|ini|txt)$")

	// a map of valid resource to retrieve from vault
	validResources = map[string]bool{
		"pki":    true,
		"aws":    true,
		"secret": true,
		"mysql":  true,
		"tpl":    true,
	}
)

func defaultVaultResource() *vaultResource {
	return &vaultResource{
		format: "yaml",
		renewable: false,
		revoked: false,
		options: make(map[string]string, 0),
	}
}

// resource ... the structure which defined a resource set from vault
type vaultResource struct {
	// the namespace of the resource
	resource string
	// the name of the resource
	name string
	// the format of the resource
	format string
	// whether the resource should be renewed?
	renewable bool
	// whether the resource should be revoked?
	revoked bool
	// the lease duration
	update time.Duration
	// additional options to the resource
	options map[string]string
}

// isValid ... checks to see if the resource is valid
func (r *vaultResource) isValid() error {
	// step: check the resource type
	if _, found := validResources[r.resource]; !found {
		return fmt.Errorf("unsupported resource type: %s", r.resource)
	}

	// step: check the options
	if err := r.isValidOptions(); err != nil {
		return fmt.Errorf("invalid resource options, %s", err)
	}

	// step: check is have all the required options to this resource type
	if err := r.isValidResource(); err != nil {
		return fmt.Errorf("invalid resource: %s, %s", r, err)
	}

	return nil
}

// isValidResource ... validate the resource meets the requirements
func (r *vaultResource) isValidResource() error {
	switch r.resource {
	case "pki":
		if _, found := r.options[OptionCommonName]; !found {
			return fmt.Errorf("pki resource requires a common name specified")
		}
	case "tpl":
		if _, found := r.options[OptionsTemplatePath]; !found {
			return fmt.Errorf("template resource requires a template path option")
		}
	}

	return nil
}

// isValidOptions ... iterates through the options, converts the options and so forth
func (r *vaultResource) isValidOptions() error {
	// check the filename directive
	for opt, val := range r.options {
		switch opt {
		case OptionFormat:
			if matched := resourceFormatRegex.MatchString(r.options[OptionFormat]); !matched {
				return fmt.Errorf("unsupported output format: %s", r.options[OptionFormat])
			}
			glog.V(20).Infof("setting the format: %s on resource: %s", val, r)
			r.format = val
		case OptionUpdate:
			duration, err := time.ParseDuration(val)
			if err != nil {
				return fmt.Errorf("the update option: %s is not value, should be a duration format", val)
			}
			glog.V(20).Infof("setting the update time: %s on resource: %s", duration, r)
			r.update = duration
		case OptionRevoke:
			choice, err := strconv.ParseBool(val)
			if err != nil {
				return fmt.Errorf("the revoke option: %s is invalid, should be a boolean", val)
			}
			glog.V(20).Infof("setting the revoked: %t on resource: %s", choice, r)
			r.revoked = choice
		case OptionRenewal:
			choice, err := strconv.ParseBool(val)
			if err != nil {
				return fmt.Errorf("the renewal option: %s is invalid, should be a boolean", val)
			}
			glog.V(20).Infof("setting the renewable: %t on resource: %s", choice, r)
			r.renewable = choice
		case OptionFilename:
			// @TODO need to check it's valid filename / path
		case OptionCommonName:
			// @TODO need to check it's a valid hostname
		case OptionsTemplatePath:
			if exists, _ := fileExists(val); !exists {
				return fmt.Errorf("the template file: %s does not exist", val)
			}
		}
	}

	return nil
}

// resourceFilename ... generates a resource filename by default the resource name and resource type, which
// can override by the OPTION_FILENAME option
func (r vaultResource) filename() string {
	if path, found := r.options[OptionFilename]; found {
		return path
	}

	return fmt.Sprintf("%s.%s", r.name, r.resource)
}

// String ... a string representation of the struct
func (r vaultResource) String() string {
	return fmt.Sprintf("%s/%s (%s|%t|%t)", r.resource, r.name, r.update, r.renewable, r.revoked)
}
