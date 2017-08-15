package main

import (
	"os"
	"testing"
)

func TestValidateOptionsWithoutVaultURL(t *testing.T) {
	os.Setenv("VAULT_ADDR", "")

	cfg := &config{}
	err := validateOptions(cfg)

	if err == nil {
		t.Errorf("should have raised error: %v", err)
	}

}

func TestValidateOptionsWithEnvFallback(t *testing.T) {
	os.Setenv("VAULT_ADDR", "http://testurl:8080")

	cfg := &config{}
	err := validateOptions(cfg)

	if err != nil {
		t.Errorf("raised an error: %v", err)
	}

	actual := cfg.vaultURL
	expected := "http://testurl:8080"

	if actual != expected {
		t.Errorf("Expected Vault URL to be %s got %s", expected, actual)
	}

}

func TestValidateOptionsWithInvalidVaultURL(t *testing.T) {
	cfg := &config{
		vaultURL: "%invalid_url",
	}
	err := validateOptions(cfg)

	if err == nil {
		t.Errorf("should have raised error")
	}
}

func TestValidateOptionsWithInvalidVaultURLFromAuthFile(t *testing.T) {
	cfg := &config{
		vaultAuthFile: "tests/invalid_kubernetes_vault_auth_file.json",
	}
	err := validateOptions(cfg)

	if err == nil {
		t.Errorf("should have raised error")
	}
}

func TestValidateOptionsWithVaultURLFromAuthFile(t *testing.T) {
	cfg := &config{
		vaultAuthFile: "tests/kubernetes_vault_auth_file.json",
	}
	err := validateOptions(cfg)

	if err != nil {
		t.Errorf("raising an error %v", err)
	}

	actual := cfg.vaultURL
	expected := "http://testurl:8080"

	if actual != expected {
		t.Errorf("Expected Vault URL to be %s got %s", expected, actual)
	}
}
