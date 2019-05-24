[![Build Status](https://travis-ci.org/UKHomeOffice/vault-sidekick.svg?branch=master)](https://travis-ci.org/UKHomeOffice/vault-sidekick)
[![GoDoc](http://godoc.org/github.com/UKHomeOffice/vault-sidekick?status.png)](http://godoc.org/github.com/UKHomeOffice/vault-sidekick)
[![Docker Repository on Quay](https://quay.io/repository/ukhomeofficedigital/vault-sidekick/status "Docker Repository on Quay")](https://quay.io/repository/ukhomeofficedigital/vault-sidekick)
[![GitHub version](https://badge.fury.io/gh/UKHomeOffice%2Fvault-sidekick.svg)](https://badge.fury.io/gh/UKHomeOffice%2Fvault-sidekick)

# Vault Side Kick

## Summary
Vault Sidekick is a add-on container which can be used as a generic entry-point for interacting with Hashicorp [Vault](https://vaultproject.io) service, retrieving secrets
(both static and dynamic) and PKI certs. The sidekick will take care of renewal's and extension of leases for you and renew the credentials in the specified format for you.

## Usage

```shell
$ sudo docker run --rm quay.io/ukhomeofficedigital/vault-sidekick:v0.3.3 -help
Usage of /vault-sidekick:
  -alsologtostderr
    	log to standard error as well as files
  -auth string
    	a configuration file in json or yaml containing authentication arguments
  -ca-cert string
    	the path to the file container the CA used to verify the vault service
  -cn value
    	a resource to retrieve and monitor from vault
  -dryrun
    	perform a dry run, printing the content to screen
  -exec-timeout duration
    	the timeout applied to commands on the exec option (default 1m0s)
  -format string
    	the auth file format (default "default")
  -log_backtrace_at value
    	when logging hits line file:N, emit a stack trace
  -log_dir string
    	If non-empty, write log files in this directory
  -logtostderr
    	log to standard error instead of files
  -one-shot
    	retrieve resources from vault once and then exit
  -output string
    	the full path to write resources or VAULT_OUTPUT (default "/etc/secrets")
  -renew-token
      renew vault token according to its ttl
  -stats duration
    	the interval to produce statistics on the accessed resources (default 1h0m0s)
  -stderrthreshold value
    	logs at or above this threshold go to stderr
  -tls-skip-verify
    	whether to check and verify the vault service certificate
  -v value
    	log level for V logs
  -vault string
    	url the vault service or VAULT_ADDR (default "https://127.0.0.1:8200")
  -version
    	show the vault-sidekick version
  -vmodule value
    	comma-separated list of pattern=N settings for file-filtered logging
```

## Building

There is a Makefile in the base repository, so assuming you have make and go: `$ make`

## Example Usage

The below is taken from a [Kubernetes](https://github.com/kubernetes/kubernetes) pod specification;

```YAML
spec:
  containers:
  - name: vault-side-kick
    image: quay.io/ukhomeofficedigital/vault-sidekick:v0.3.3
    args:
      - -output=/etc/secrets
      - -cn=pki:project1/certs/example.com:common_name=commons.example.com,revoke=true,update=2h
      - -cn=secret:secret/db/prod/username:file=.credentials
      - -cn=secret:secret/db/prod/password:retries=true
      - -cn=secret:secret/data/db/dev/username:file=.kv2credentials
      - -cn=aws:aws/creds/s3_backup_policy:file=.s3_creds
    volumeMounts:
      - name: secrets
        mountPath: /etc/secrets
```

The above equates to:

- Write all the secrets to the /etc/secrets directory
- Retrieve a dynamic certificate pair for me, with the common name: 'commons.example.com' and renew the cert when it expires automatically
- Retrieve the two static secrets /db/prod/{username,password} and write them to .credentials and password.secret respectively
- Retrieve the latest version of static secret /db/dev/username from a v2 kv store and write it to .kv2credentials
- Apply the IAM policy, renew the policy when required and file the API tokens to .s3_creds in the /etc/secrets directory
- Read the template at /etc/templates/db.tmpl, produce the content from Vault and write to /etc/credentials file

## Authentication

An authentication file can be specified in either yaml of json format which contains a method field, indicating one of the authentication
methods provided by vault i.e. userpass, token, github etc and then followed by the required arguments for that plugin.

If the required arguments for that plugin are not contained in the authentication file, fallbacks from environment variables are used.
Environment variables are prefixed with `VAULT_SIDEKICK`, i.e. `VAULT_SIDEKICK_USERNAME`, `VAULT_SIDEKICK_PASSWORD`.

### Kubernetes Authentication

The Kubernetes auth plugin supports the following environment variables:

- `VAULT_SIDEKICK_ROLE` - The Vault role name against which to authenticate (**REQUIRED**)
- `VAULT_K8S_LOGIN_PATH` - If your Kubernetes auth backend is mounted at a path other than `kubernetes/` you will need to set this. Default `/v1/auth/kubernetes/login`
- `VAULT_K8S_TOKEN_PATH` - If you mount in-pod service account tokens to a non-default path, you will need to set this. Default `/var/run/secrets/kubernetes.io/serviceaccount/token`

## Secret Renewals

The default behaviour of vault-sidekick is **not** to renew a lease, but to retrieve a new secret and allow the previous to
expire, in order ensure the rotation of secrets. If you don't want this behaviour on a resource you can override using resource options. For exmaple,
your using the mysql dynamic secrets, you want to renew the secret not replace it

```shell
[jest@starfury vault-sidekick]$ build/vault-sidekick -cn=mysql:mysql/creds/my_database:fmt=yaml,renew=true
or an iam policy renewed every hour
[jest@starfury vault-sidekick]$ build/vault-sidekick -cn=aws:aws/creds/policy:fmt=yaml,renew=true,update=1h

```

Or you want to rotate the secret every **1h** and **revoke** the previous one

```shell
[jest@starfury vault-sidekick]$ build/vault-sidekick -cn=aws:project/creds/my_s3_bucket:fmt=yaml,update=1h,revoke=true

The format is;

-cn=RESOURCE_TYPE:PATH:OPTIONS
```

The sidekick supports the following resource types: mysql, postgres, database, pki, aws, gcp, secret, cubbyhole, raw, cassandra and transit

## Environment Variable Expansion

The resource paths can contain environment variables which the sidekick will resolve beforehand. A use case being, using a environment
or domain within the resource e.g -cn=secret:secrets/myservice/${ENV}/config:fmt=yaml

## Output Formatting

The following output formats are supported: json, yaml, ini, txt, cert, certchain, csv, bundle, env, credential, aws

Using the following at the demo secrets

```shell
[jest@starfury vault-sidekick]$ vault write secret/password this=is demo=value nothing=more
Success! Data written to: secret/password
[jest@starfury vault-sidekick]$ vault read secret/password
Key            	Value
lease_id       	secret/password/7908eceb-9bde-e7de-23da-96131505214a
lease_duration 	2592000
lease_renewable	false
demo           	value
nothing        	more
this           	is
```

In order to change the output format:

```shell
[jest@starfury vault-sidekick]$ build/vault-sidekick -cn=secret:secret/password:fmt=ini -logtostderr=true -dry-run
[jest@starfury vault-sidekick]$ build/vault-sidekick -cn=secret:secret/password:fmt=json -logtostderr=true -dry-run
[jest@starfury vault-sidekick]$ build/vault-sidekick -cn=secret:secret/password:fmt=yaml -logtostderr=true -dry-run
```

Format: 'cert' is less of a format of more file scheme i.e. is just extracts the 'certificate', 'issuing_ca' and 'private_key' and creates the three files FILE.{ca,key,crt}. The
bundle format is very similar in the sense it similar takes the private key and certificate and places into a single file.
'credential' will attempt to decode a GCP credential file and 'aws' will write an AWS credentials file.

## Resource Options

- **file**: (filename) by default all file are relative to the output directory specified and will have the name NAME.RESOURCE; the fn options allows you to switch names and paths to write the files
- **mode**: (mode) overrides the default file permissions of the secret from 0664
- **create**: (create) create the resource
- **update**: (update) override the lease time of this resource and get/renew a secret on the specified duration e.g 1m, 2d, 5m10s
- **renew**: (renewal) override the default behavour on this resource, renew the resource when coming close to expiration e.g true, TRUE
- **delay**: (renewal-delay) delay the revoking the lease of a resource for x period once time e.g 1m, 1h20s
- **revoke**: (revoke) revoke the old lease when you get retrieve a old one e.g. true, TRUE (default to allow the lease to expire and naturally revoke)
- **fmt**: (format) allows you to specify the output format of the resource / secret, e.g json, yaml, ini, txt
- **exec** (execute) execute's a command when resource is updated or changed
- **retries**: (retries) the maximum number of times to retry retrieving a resource. If not set, resources will be retried indefinitely
- **jitter**: (jitter) an optional maximum jitter duration. If specified, a random duration between 0 and `jitter` will be subtracted from the renewal time for the resource
