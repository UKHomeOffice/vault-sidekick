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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetResources(t *testing.T) {
	var items VaultResources

	assert.Nil(t, items.Set("secret:test:file=filename.test,fmt=yaml"))
	assert.Nil(t, items.Set("secret:test:file=filename.test,"))
	assert.Nil(t, items.Set("secret:/db/prod/username"))
	assert.Nil(t, items.Set("secret:/db/prod:file=filename.test,fmt=yaml"))
	assert.Nil(t, items.Set("secret:test:fn=filename.test,"))
	assert.Nil(t, items.Set("pki:example-dot-com:cn=blah.example.com"))
	assert.Nil(t, items.Set("pki:example-dot-com:cn=blah.example.com,file=/etc/certs/ssl/blah.example.com"))
	assert.Nil(t, items.Set("pki:example-dot-com:cn=blah.example.com,renew=10s"))
	assert.NotNil(t, items.Set("secret:"))
	assert.NotNil(t, items.Set("secret:test:file=filename.test,fmt="))
	assert.NotNil(t, items.Set("secret::file=filename.test,fmt=yaml"))
	assert.NotNil(t, items.Set("secret:te1st:file=filename.test,fmt="))
	assert.NotNil(t, items.Set("file=filename.test,fmt=yaml"))
}

func TestResources(t *testing.T) {
	var items VaultResources
	items.Set("secret:test:file=filename.test,fmt=yaml")
	items.Set("secret:test:file=fileame.test")

	if passed := assert.Equal(t, len(items.items), 2); !passed {
		t.FailNow()
	}

	rn := items.items[0]
	assert.Equal(t, "secret", rn.resource)
	assert.Equal(t, "test", rn.path)
	assert.Equal(t, 2, len(rn.options))
	assert.Equal(t, "filename.test", rn.options[optionFilename])
	assert.Equal(t, "yaml", rn.options[optionFormat])
	rn = items.items[1]
	assert.Equal(t, "secret", rn.resource)
	assert.Equal(t, "test", rn.path)
	assert.Equal(t, 1, len(rn.options))
	assert.Equal(t, "fileame.test", rn.options[optionFilename])
}
