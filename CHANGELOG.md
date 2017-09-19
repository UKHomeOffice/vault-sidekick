
#### **Version v0.3.4**

##### FEATURES

 * Adding a jitter option to the resources

#### **Version v0.3.3**

##### FEATURES

 * Loading vault url from kubernetes vault auth file, exit if vault url is not set

#### **Version v0.3.2**

##### FEATURES

 * Added kubernetes-vault support
 * Added onetime only mode via the one-shot option
 * Added the 'retries' parameter to resources to allow optional maxRetries

#### **Version v0.3.1**

##### FEATURES

 * Added a mode option to the resource specification enabling secrets to set the file permissions
 * Fixed a bug in the renewal time, when a resource does not have a custom update and the lease time is 0s
 * Cleaned up some of the vetting issues
 * Change the travis build to use golang v1.8.1
 * Added a version flag -version and passing the gitsha in the version
 * Updated the kubernete deployment files

#### **Version v0.1.0**

##### FEATURES

BUGS
 * Fixed the bundle format to produce four file, a bundle with cert+ca, and the FILENAME-ca.pem, FILENAME-key.pem,
   and the FILENAME.pem certificate

#### **Version v0.0.9-1**

##### FEATURES

 * Adding the ability to perform environment variable substituted of the resource path i.e.
   -resource=secret:/secrets/%ENV%/myset : %ENV% will substituted

#### **Version v0.0.9**

##### FEATURES

 * Adding the ability to create random secrets via the create option

#### **Version v0.0.8**

##### FEATURES

 * Adding an exec option to the control set, the command is called whenever a change is made on the resource with a
   condfigurable timeout (default to 60s)
   -cn=secret:platform/secrets/se2:fmt=yaml,exec=tests/runme.sh,update=1s

#### **Version v0.0.7**

##### FEATURES
 * Adding the ability to the perform raw queries to vault, the formatting stays the same - a single
   data key 'content' is returned, example: -cn=raw:platform/pki/ca/pem:fmt=txt,file=ca

#### **Version v0.0.6**

##### FEATURES:

 * Fixed up a number of niggling issues
 * Added the bundle format to pki paths can write a bundle private and certificate file and a separate ca file
 * Added the env format which will create a environment variables file
 * Adding comma separated list as resource arguments comes in the form <ARG>|<ARG> i.e.
   -cn=pki:platform/pki/issue/example-dot-com:common_name=blah.example.com,alt_names='me.example.com|ted.example.com'

##### BREAKING CHANGES:
 * Note, because all params excluding the control options are passed as arguments to vault the arguments must be the
   same as those for vault, i.e. for pki cn -> common_name

##### BUGS:

 * Fixed the formatting of values in various formats, i.e. %!s(bool=true)
