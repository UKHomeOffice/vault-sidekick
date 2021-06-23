package main

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/golang/glog"
	"github.com/hashicorp/vault/api"
)

const (
	serviceAccountTokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	vaultTokenFilePath      = "/var/run/secrets/vaultproject.io/.vault-token"
	vaultAuthFilePath       = "/var/run/secrets/vaultproject.io/.vault-auth.yml"
)

var (
	vaultAuthPath = getEnv("VAULT_AUTH_PATH", "auth/kubernetes")
	vaultAddress  = getEnv("VAULT_ADDR", "http://vault.svc.cluster.local")
	vaultRole     = getEnv("VAULT_ROLE", "default")
	httpClient    = &http.Client{
		Timeout: 5 * time.Second,
	}
)

type authFile struct {
	Method string `yaml:"method"`
	Token  string `yaml:"token"`
}

func requestToken() {

	var auth authFile

	glog.Info("Configuring vault client..")

	client, err := api.NewClient(&api.Config{Address: vaultAddress, HttpClient: httpClient})
	if err != nil {
		glog.Fatal(err.Error())
	}

	glog.Info("Retrieving service account token..")

	serviceAccountToken, err := ioutil.ReadFile(serviceAccountTokenPath)
	if err != nil {
		glog.Fatal(err.Error())
	}

	glog.Infof("Requesting for role %s vault-token..", vaultRole)

	secret, err := client.Logical().Write(path.Join(vaultAuthPath, "login"), map[string]interface{}{
		"jwt":  string(bytes.TrimSpace(serviceAccountToken)),
		"role": vaultRole,
	})
	if err != nil {
		glog.Fatal(err.Error())
	}

	glog.Infof("Writing the vault-token out to %s", vaultTokenFilePath)

	if err := ioutil.WriteFile(vaultTokenFilePath, []byte(secret.Auth.ClientToken), 0644); err != nil {
		glog.Fatal(err.Error())
	}

	auth.Token = secret.Auth.ClientToken
	auth.Method = "token"

	authFileData, err := yaml.Marshal(auth)
	if err != nil {
		glog.Fatal(err.Error())
	}
	glog.Infof("Writing the vault-auth file out to %s", vaultAuthFilePath)
	if err := ioutil.WriteFile(vaultAuthFilePath, authFileData, 0644); err != nil {
		glog.Fatal(err.Error())
	}

	glog.Info("Done")
	os.Exit(0)
}
