package main

import (
	"encoding/json"
	"io/ioutil"
	"math"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/golang/glog"
	"github.com/hashicorp/vault/api"
)

const (
	renewalPercentage = 0.02 // Renew the vault-token after 2% of it's TTL has lapsed

)

var (
	vaultToken = getEnv("VAULT_TOKEN", "")
)

func renew(client *api.Client) {
	glog.Infof("Renewing the vault-token...")

	if _, err := client.Auth().Token().RenewSelf(0); err != nil {
		glog.Fatal(err.Error())
	}
}

func renewToken() {
	glog.Info("Configuring vault client..")
	var auth authFile

	client, err := api.NewClient(&api.Config{Address: vaultAddress, HttpClient: httpClient})
	if err != nil {
		glog.Fatal(err.Error())
	}

	if len(vaultToken) == 0 {
		glog.Infof("Retrieving vault-token from %s file..", vaultTokenFilePath)

		vaultToken, err := ioutil.ReadFile(vaultTokenFilePath)
		if err != nil {
			glog.Fatal(err.Error())
		}

		client.SetToken(string(vaultToken))
	}

	glog.Info("Retrieving vault-token metadata...")

	token, err := client.Auth().Token().LookupSelf()
	glog.Infof("token  %s", token.Data)
	if err != nil {
		glog.Fatal(err.Error())
	}

	glog.Info("Parsing vault-token metadata...")

	ttl, err := token.Data["creation_ttl"].(json.Number).Float64()
	glog.Infof("token TTL %d", ttl)
	if err != nil {
		glog.Fatal(err.Error())
	}

	expireTime, err := time.Parse(time.RFC3339, token.Data["expire_time"].(string))
	glog.Infof("expireTime  %s", expireTime.Format("Mon Jan 2 15:04:05 MST 2006"))
	if err != nil {
		glog.Fatal(err.Error())
	}

	renewalPeriod := int(math.Round(ttl * renewalPercentage))
	glog.Infof("renewPeriod %d", renewalPeriod)
	secondsUntilExpires := int(time.Until(expireTime).Seconds())

	// If the time left on the token is less than the renewal period
	// we should renew immediatelly to prevent the token from expiring
	// whilst waiting for the configured renewal period
	if secondsUntilExpires < renewalPeriod {
		glog.Infof("Time left until expiration %d is less than the renewal period of %d..", secondsUntilExpires, renewalPeriod)

		renew(client)
		auth.Token = client.Token()
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
	}

	// An infinite loop which will wait a configured renewal
	// period before issuing a renewal call ensuring the token
	// is always renewed before it has a chance to expire
	for {
		glog.Infof("Waiting for the configured renewal period of %d seconds..", renewalPeriod)

		time.Sleep(time.Duration(renewalPeriod) * time.Second)
		renew(client)

		auth.Token = client.Token()
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

	}

}
