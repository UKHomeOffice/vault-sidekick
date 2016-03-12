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
	"strings"
)

// VaultResources is a collection of type resource
type VaultResources struct {
	// an array of resource to retrieve
	items []*VaultResource
}

// Set is the implementation for the parser
// secret:test:file=filename.test,fmt=yaml
func (r *VaultResources) Set(value string) error {
	rn := defaultVaultResource()

	// step: split on the ':'
	items := strings.Split(value, ":")
	if len(items) < 2 {
		return fmt.Errorf("invalid resource, must have at least two sections TYPE:PATH")
	}
	if len(items) > 3 {
		return fmt.Errorf("invalid resource, can only has three sections, TYPE:PATH[:OPTIONS]")
	}
	if items[0] == "" || items[1] == "" {
		return fmt.Errorf("invalid resource, neither type or path can be empty")
	}

	// step: extract the elements
	rn.resource = items[0]
	rn.path = items[1]
	rn.options = make(map[string]string, 0)

	// step: extract any options
	if len(items) > 2 {
		for _, x := range strings.Split(items[2], ",") {
			kp := strings.Split(x, "=")
			if len(kp) != 2 {
				return fmt.Errorf("invalid resource option: %s, must be KEY=VALUE", x)
			}
			if kp[1] == "" {
				return fmt.Errorf("invalid resource option: %s, must have a value", x)
			}

			rn.options[kp[0]] = kp[1]
		}
	}
	// step: append to the list of resources
	r.items = append(r.items, rn)

	return nil
}

// String returns a string representation of the struct
func (r VaultResources) String() string {
	return ""
}
