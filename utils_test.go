package main

import (
	"testing"
)

func TestReadConfigFileKubernetesVault(t *testing.T) {
	o, err := readConfigFile("tests/kubernetes_vault_auth_file.json", "kubernetes-vault")
	if err != nil {
		t.Errorf("raising an error: %v", err)
	}

	tokenExpected := "foobar"

	if o.Token != tokenExpected {
		t.Errorf("Expected user %s got %s", tokenExpected, o.Token)
	}
}

func TestReadConfigUserPassJSON(t *testing.T) {
	o, err := readConfigFile("tests/userpass_auth_file.json", "default")
	if err != nil {
		t.Errorf("raising an error: %v", err)
	}

	userExpected := "admin"
	passwordExpected := "foobar"

	if o.Username != userExpected {
		t.Errorf("Expected user %s got %s", userExpected, o.Username)
	}

	if o.Password != passwordExpected {
		t.Errorf("Expected user %s got %s", passwordExpected, o.Password)
	}
}

func TestReadConfigUserPassYAML(t *testing.T) {
	o, err := readConfigFile("tests/userpass_auth_file.yml", "default")
	if err != nil {
		t.Errorf("raising an error: %v", err)
	}

	userExpected := "admin"
	passwordExpected := "foobar"

	if o.Username != userExpected {
		t.Errorf("Expected user %s got %s", userExpected, o.Username)
	}

	if o.Password != passwordExpected {
		t.Errorf("Expected user %s got %s", passwordExpected, o.Password)
	}
}

func TestReadConfigAppRoleJSON(t *testing.T) {
	o, err := readConfigFile("tests/approle_auth_file.json", "default")
	if err != nil {
		t.Errorf("raising an error: %v", err)
	}

	roleIDExpected := "admin"
	secretIDExpected := "foobar"

	if o.RoleID != roleIDExpected {
		t.Errorf("Expected roleID %s got %s", roleIDExpected, o.RoleID)
	}

	if o.SecretID != secretIDExpected {
		t.Errorf("Expected secretID %s got %s", secretIDExpected, o.SecretID)
	}
}

func TestReadConfigAppRoleYAML(t *testing.T) {
	o, err := readConfigFile("tests/approle_auth_file.yml", "default")
	if err != nil {
		t.Errorf("raising an error: %v", err)
	}

	roleIDExpected := "admin"
	secretIDExpected := "foobar"

	if o.RoleID != roleIDExpected {
		t.Errorf("Expected roleID %s got %s", roleIDExpected, o.RoleID)
	}

	if o.SecretID != secretIDExpected {
		t.Errorf("Expected secretID %s got %s", secretIDExpected, o.SecretID)
	}
}
func TestReadConfigTokenJSON(t *testing.T) {
	o, err := readConfigFile("tests/token_auth_file.json", "default")
	if err != nil {
		t.Errorf("raising an error: %v", err)
	}

	expected := "foobar"

	if o.Token != expected {
		t.Errorf("Expected token %s got %s", expected, o.Token)
	}
}

func TestReadConfigTokenYAML(t *testing.T) {
	o, err := readConfigFile("tests/token_auth_file.yml", "default")
	if err != nil {
		t.Errorf("raising an error: %v", err)
	}

	expected := "foobar"

	if o.Token != expected {
		t.Errorf("Expected token %s got %s", expected, o.Token)
	}
}

func TestGetDurationWithin(t *testing.T) {
	duration := getDurationWithin(1, 1)

	if duration <= 0 {
		t.Errorf("Expected duration to be higher than 0 got %d", duration)
	}
}
