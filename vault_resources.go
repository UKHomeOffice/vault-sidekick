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
	"strings"
)

var (
	resourceRegex        = regexp.MustCompile("^([\\w]+):([\\w\\\\/\\-_\\.]+):?(.*)")
	resourceOptionsRegex = regexp.MustCompile("([\\w\\d]{2,3})=([\\w\\d\\/\\.\\-_]+)[,]?")
)

// resources ... a collection of type resource
type vaultResources struct {
	// an array of resource to retrieve
	items []*vaultResource
}

func (r vaultResources) size() int {
	return len(r.items)
}

// Set ... implementation for the parser
func (r *vaultResources) Set(value string) error {
	rn := new(vaultResource)

	// step: extract the resource type and name
	if matched := resourceRegex.MatchString(value); !matched {
		return fmt.Errorf("invalid resource specification, should be TYPE:NAME:?(OPTION_NAME=VALUE,)")
	}

	// step: extract the matches
	matches := resourceRegex.FindAllStringSubmatch(value, -1)
	rn.resource = matches[0][1]
	rn.name = matches[0][2]
	rn.options = make(map[string]string, 0)

	// step: do we have any options for the resource?
	if len(matches[0]) == 4 {
		opts := matches[0][3]
		for len(opts) > 0 {
			if matched := resourceOptionsRegex.MatchString(opts); !matched {
				return fmt.Errorf("invalid resource options specified: '%s', please check usage", opts)
			}

			matches := resourceOptionsRegex.FindAllStringSubmatch(opts, -1)
			rn.options[matches[0][1]] = matches[0][2]
			opts = strings.TrimPrefix(opts, matches[0][0])
		}
	}
	// step: append to the list of resources
	r.items = append(r.items, rn)

	return nil
}

// String ... returns a string representation of the struct
func (r vaultResources) String() string {
	return ""
}
