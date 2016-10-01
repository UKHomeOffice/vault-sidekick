[![Build Status](https://travis-ci.org/UKHomeOffice/vault-sidekick.svg?branch=master)](https:/
/travis-ci.org/UKHomeOffice/vault-sidekick)
[![GoDoc](http://godoc.org/github.com/UKHomeOffice/vault-sidekick?status.png)](http://godoc.or
g/github.com/UKHomeOffice/vault-sidekick)
[![Docker Repository on Quay](https://quay.io/repository/UKHomeOffice/vault-sidekick/status "D
ocker Repository on Quay")](https://quay.io/repository/UKHomeOffice/vault-sidekick)
[![GitHub version](https://badge.fury.io/gh/UKHomeOffice%2Fvault-sidekick.svg)](https://badge.
fury.io/gh/UKHomeOffice%2Fvault-sidekick)

### **Vault Side Kick**

**Summary:**
Vault Sidekick is a add-on container which can be used as a generic entry-point for interacting with Hashicorp [Vault](https://vaultproject.io) service, retrieving secrets
(both static and dynamic) and PKI certs. The sidekick will take care of renewal's and extension of leases for you and renew the credentials in the specified format for you.

**Usage:**

```shell
[jest@starfury vault-sidekick]$ bin/vault-sidekick --help
Usage of bin/vault-sidekick:
  -alsologtostderr=false: log to standard error as well as files
  -auth="": a configuration file in a json or yaml containing authentication arguments
  -cn=: a resource to retrieve and monitor from vault (e.g. pki:name:cert.name, secret:db_password, aws:s3_backup)
  -ca-cert="": a CA certificate to use in order to validate the vault service certificate
  -delete-token=false: once the we have connected to vault, delete the token file from disk
  -dryrun=false: perform a dry run, printing the content to screen
  -log_backtrace_at=:0: when logging hits line file:N, emit a stack trace
  -log_dir="": If non-empty, write log files in this directory
  -logtostderr=false: log to standard error instead of files
  -output="/etc/secrets": the full path to write the protected resources (VAULT_OUTPUT if available)
  -stats=5m0s: the interval to produce statistics on the accessed resources
  -stderrthreshold=0: logs at or above this threshold go to stderr
  -tls-skip-verify=false: skip verifying the vault certificate
  -token="": the token used to authenticate to teh vault service (VAULT_TOKEN if available)
  -v=0: log level for V logs
  -vault="https://127.0.0.1:8200": the url the vault service is running behind (VAULT_ADDR if available)
  -vmodule=: comma-separated list of pattern=N settings for file-filtered logging
```

**Building**

There is a Makefile in the base repository, so assuming you have make and go: # make

**Example Usage**

The below is taken from a [Kubernetes](https://github.com/kubernetes/kubernetes) pod specification;

```YAML
spec:
      containers:
      - name: vault-side-kick
        image: gambol99/vault-sidekick:latest
        args:
          - -output=/etc/secrets
          - -cn=pki:project1/certs/example.com:common_name=commons.example.com,revoke=true,update=2h
          - -cn=secret:secret/db/prod/username:file=.credentials
          - -cn=secret:secret/db/prod/password
          - -cn=aws:aws/creds/s3_backup_policy:file=.s3_creds
        volumeMounts:
          - name: secrets
            mountPath: /etc/secrets
```

The above say's

 - Write all the secrets to the /etc/secrets directory
 - Retrieve a dynamic certificate pair for me, with the common name: 'commons.example.com' and renew the cert when it expires automatically
 - Retrieve the two static secrets /db/prod/{username,password} and write them to .credentials and password.secret respectively
 - Apply the IAM policy, renew the policy when required and file the API tokens to .s3_creds in the /etc/secrets directory
 - Read the template at /etc/templates/db.tmpl, produce the content from Vault and write to /etc/credentials file

**Authentication**

A authentication file can be specified in either yaml of json format which contains a method field, indicating one of the authentication
methods provided by vault i.e. userpass, token, github etc and then followed by the required arguments for that plugin.

If the required arguments for that plugin are not contained in the authentication file, fallbacks from environment variables are used.
Environment variables are prefixed with `VAULT_SIDEKICK`, i.e. `VAULT_SIDEKICK_USERNAME`, `VAULT_SIDEKICK_PASSWORD`.

**Secret Renewals**

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

The sidekick supports the following resource types: mysql, postgres, pki, aws, secret, cubbyhole, raw, cassandra and transit

**Environment Variable Expansion**

The resource paths can contain environment variables which the sidekick will resolve beforehand. A use case being, using a environment
or domain within the resource e.g -cn=secret:secrets/myservice/${ENV}/config:fmt=yaml

**Output Formatting**

The following output formats are supported: json, yaml, ini, txt, cert, csv, bundle, env

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

**Resource Options**

- **file**: (filaname) by default all file are relative to the output directory specified and will have the name NAME.RESOURCE; the fn options allows you to switch names and paths to write the files
- **create**: (create) create the resource
- **update**: (update) override the lease time of this resource and get/renew a secret on the specified duration e.g 1m, 2d, 5m10s
- **renew**: (renewal) override the default behavour on this resource, renew the resource when coming close to expiration e.g true, TRUE
- **delay**: (renewal-delay) delay the revoking the lease of a resource for x period once time e.g 1m, 1h20s
- **revoke**: (revoke) revoke the old lease when you get retrieve a old one e.g. true, TRUE (default to allow the lease to expire and naturally revoke)
- **fmt**: (format) allows you to specify the output format of the resource / secret, e.g json, yaml, ini, txt
- **exec** (execute) execute's a command when resource is updated or changed
