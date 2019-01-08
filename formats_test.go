package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWriteAwsCredentialFileIAMUser(t *testing.T) {
	data := map[string]interface{}{
		"access_key": "AKIAJIVWU52VCBFROFFA",
		"secret_key": "oocha7Wahma3bahmaitoo8ufae6Yahzouphooy2p",
		"security_token": nil,
	}
	expected := `[default]
aws_access_key_id=AKIAJIVWU52VCBFROFFA
aws_secret_access_key=oocha7Wahma3bahmaitoo8ufae6Yahzouphooy2p
`
	assert.Equal(t, expected, string(generateAwsCredentialFile(data)))
}

func TestWriteAwsCredentialFileAssumedRole(t *testing.T) {
	data := map[string]interface{}{
		"access_key":     "AKIAJIVWN52VCBFROAFA",
		"secret_key":     "oocha7Wahma3bahmaitoo8ufae6Yahzouphooy2p",
		"security_token": "phe2lahD7oofoo8eibohpu1kuwohn0eir7wieH7E",
	}

	expected := `[default]
aws_access_key_id=AKIAJIVWN52VCBFROAFA
aws_secret_access_key=oocha7Wahma3bahmaitoo8ufae6Yahzouphooy2p
aws_session_token=phe2lahD7oofoo8eibohpu1kuwohn0eir7wieH7E
`
	assert.Equal(t, expected, string(generateAwsCredentialFile(data)))
}