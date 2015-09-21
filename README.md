
### **Vault Side Kick**

**Summary:**
Vault Sidekick is a add-on container which can be used as a generic entry-point for interacting with Hashicorp [Vault](https://vaultproject.io) service, retrieving secrets
(both static and dynamic) and PKI certs. The sidekick will take care of renewal's and extension of leases for you and renew the credentials in the specified format for you.

**Usage:**

```shell
[jest@starfury vault-sidekick]$ build/vault-sidekick -help
Usage of build/vault-sidekick:
  -alsologtostderr=false: log to standard error as well as files
  -cn=: a resource to retrieve and monitor from vault (e.g. pki:name:cert.name, secret:db_password, aws:s3_backup)
  -log_backtrace_at=:0: when logging hits line file:N, emit a stack trace
  -log_dir="": If non-empty, write log files in this directory
  -logtostderr=false: log to standard error instead of files
  -output="/etc/secrets": the full path to write the protected resources (VAULT_OUTPUT if available)
  -stderrthreshold=0: logs at or above this threshold go to stderr
  -token="": the token used to authenticate to teh vault service (VAULT_TOKEN if available)
  -tokenfile="": the full path to file containing the vault token used for authentication (VAULT_TOKEN_FILE if available)
  -v=0: log level for V logs
  -vault="https://127.0.0.1:8200": the url the vault service is running behind (VAULT_ADDR if available)
  -vmodule=: comma-separated list of pattern=N settings for file-filtered logging
```

**Example Usage**

The below is taken from a [Kubernetes](https://github.com/kubernetes/kubernetes) pod specification;

```YAML
spec:
      containers:
      - name: vault-side-kick
        image: gambol99/vault-sidekick:latest
        args:
          - -output=/etc/secrets
          - -rn=pki:example.com:cn=commons.example.com,exec=/usr/bin/nginx_restart.sh,ctr=.*nginx_server.*
          - -rn=secret:db/prod/username:fn=.credentials
          - -rn=secret:db/prod/password
          - -rn=aws:s3_backsup:fn=.s3_creds
          - -rb=template:database_credentials:tpl=/etc/templates/db.tmpl,fn=/etc/credentials
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

A authentication file can be specified 

**Secret Renewals**

The default behaviour of vault-sidekick is **not** to renew a lease, but to retrieve a new secret and allow the previous to
expire, in order ensure the rotation of secrets. If you don't want this behaviour on a resource you can override using resource options. For exmaple,
your using the mysql dynamic secrets, you want to renew the secret not replace it

```shell
[jest@starfury vault-sidekick]$ build/vault-sidekick -cn=mysql:my_database:fmt=yaml,rn=true
or an iam policy renewed every hour
[jest@starfury vault-sidekick]$ build/vault-sidekick -cn=aws:aws_policy_path:fmt=yaml,rn=true,up=1h

```

Or you want to rotate the secret every **1h** and **revoke** the previous one

```shell
[jest@starfury vault-sidekick]$ build/vault-sidekick -cn=aws:my_s3_bucket:fmt=yaml,up=1h,rv=true
```

**Output Formatting**

The following output formats are supported: json, yaml, ini, txt, cert

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
[jest@starfury vault-sidekick]$ build/vault-sidekick -cn=secret:password:fmt=ini -logtostderr=true -dry-run
[jest@starfury vault-sidekick]$ build/vault-sidekick -cn=secret:password:fmt=json -logtostderr=true -dry-run
[jest@starfury vault-sidekick]$ build/vault-sidekick -cn=secret:password:fmt=yaml -logtostderr=true -dry-run
```

The default format is 'txt' which has the following behavour. If the number of keys in a resource is > 1, a file is created per key. Thus using the example
(build/vault-sidekick -cn=secret:password:fn=test) we would end up with files: test.this, test.nothing and test.demo

Format: 'cert' is less of a format of more file scheme i.e. is just extracts the 'certificate', 'issuing_ca' and 'private_key' and creates the three files FILE.{ca,key,crt}

**Resource Options**

- **fn**: (filaname) by default all file are relative to the output directory specified and will have the name NAME.RESOURCE; the fn options allows you to switch names and paths to write the files
- **up**: (update) override the lease time of this resource and get/renew a secret on the specified duration e.g 1m, 2d, 5m10s
- **rn**: (renewal) override the default behavour on this resource, renew the resource when coming close to expiration e.g true, TRUE
- **rv**: (revoke) revoke the old lease when you get retrieve a old one e.g. true, TRUE (default to allow the lease to expire and naturally revoke)
- **fmt**: (format) allows you to specify the output format of the resource / secret, e.g json, yaml, ini, txt
- **cn**: (comman name) is used in conjunction with the PKI resource. The common argument is passed as an argument when make a request to issue the certs.
