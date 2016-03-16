
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
