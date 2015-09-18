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
	OptionRenew = "rn"
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

func newVaultResource() *vaultResource {
	return &vaultResource{
		options: make(map[string]string, 0),
	}
}

// resource ... the structure which defined a resource set from vault
type vaultResource struct {
	// the namespace of the resource
	resource string
	// the name of the resource
	name string
	// additional options to the resource
	options map[string]string
}

// leaseTime ... get the renew time otherwise return 0
func (r vaultResource) leaseTime() time.Duration {
	if _, found := r.options[OptionRenew]; found {
		duration, _ := time.ParseDuration(r.options[OptionRenew])
		return duration
	}

	return time.Duration(0)
}

// isValid ... checks to see if the resource is valid
func (r vaultResource) isValid() error {
	// step: check the resource type
	if _, found := validResources[r.resource]; !found {
		return fmt.Errorf("unsupported resource type: %s", r.resource)
	}

	// step: check the options
	if err := r.isValidOptions(); err != nil {
		return fmt.Errorf("invalid resource options: %s, %s", r.options, err)
	}

	// step: check is have all the required options to this resource type
	if err := r.isValidResource(); err != nil {
		return fmt.Errorf("invalid resource: %s, %s", r, err)
	}

	return nil
}

// getFormat ... get the format of the resource
func (r vaultResource) getFormat() string {
	if format, found := r.options[OptionFormat]; found {
		return format
	}
	return "txt"
}


// isValidResource ... validate the resource meets the requirements
func (r vaultResource) isValidResource() error {
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

// isValidOptions ... iterates through the options and check they are ok
func (r vaultResource) isValidOptions() error {
	// check the filename directive
	for opt, val := range r.options {
		switch opt {
		case OptionFormat:
			if matched := resourceFormatRegex.MatchString(r.options[OptionFormat]); !matched {
				return fmt.Errorf("unsupported output format: %s", r.options[OptionFormat])
			}
		case OptionRenew:
			if _, err := time.ParseDuration(val); err != nil {
				return fmt.Errorf("the renew option: %s is not value", val)
			}
		case OptionFilename:
		case OptionCommonName:
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
	return fmt.Sprintf("%s/%s", r.resource, r.name)
}
