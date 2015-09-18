
### **Vault Side Kick**
-----
**Summary:**
> Vault Sidekick is a add-on container which can be used as a generic entry-point for interacting with Hashicorp [Vault](https://vaultproject.io) service, retrieving secrets 
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
  -renew=true: whether or not to renew secrets from vault
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
 
**Output Formatting**

The following output formats are supported: json, yaml, ini, txt
 
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

**Resource Options**

- **fn**: (filaname) by default all file are relative to the output directory specified and will have the name NAME.RESOURCE; the fn options allows you to switch names and paths to write the files
- **rn**: (renewal) allow you to set the renewal time on a resource, but default we take the lease time from the secret and use that, the rn feature allows use to override it
- **fmt**: (format) allows you to specify the output format of the resource / secret. 
- **cn**: (comman name) is used in conjunction with the PKI resource. The common argument is passed as an argument when make a request to issue the certs. 