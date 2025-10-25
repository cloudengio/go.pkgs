# Package [cloudeng.io/webapp/webauth/acme](https://pkg.go.dev/cloudeng.io/webapp/webauth/acme?tab=doc)

```go
import cloudeng.io/webapp/webauth/acme
```

Package acme provides support for working with ACNE service providers such
as letsencrypt.org.

## Constants
### LetsEncryptStaging, LetsEncryptProduction
```go
// LetsEncryptStaging is the URL for the letsencrypt.org staging service
// and is used as the default by this package.
LetsEncryptStaging = "https://acme-staging-v02.api.letsencrypt.org/directory"
// LetsEncryptProduction is the URL for the letsencrypt.org production service.
LetsEncryptProduction = acme.LetsEncryptURL

```



## Functions
### Func NewAutocertManager
```go
func NewAutocertManager(ctx context.Context, cache autocert.Cache, cl AutocertConfig, allowedHosts ...string) (*autocert.Manager, error)
```
NewAutocertManager creates a new autocert.Manager from the supplied config.
Any supplied hosts, along with the ClientHost, are used to specify the
allowed hosts for the manager.



## Types
### Type AutocertConfig
```go
type AutocertConfig struct {
	// Contact email for the ACME account, note, changing this may create
	// a new account with the ACME provider. The key associated with an account
	// is required for revoking certificates issued using that account.
	Email       string        `yaml:"email"`
	UserAgent   string        `yaml:"user_agent"`    // User agent to use when connecting to the ACME service.
	Provider    string        `yaml:"acme_provider"` // ACME service provider URL or 'letsencrypt' or 'letsencrypt-staging'.
	RenewBefore time.Duration `yaml:"renew_before"`  // How early certificates should be renewed before they expire.
	ClientHost  string        `yaml:"client_host"`   // Host running the ACME client responsible for refreshing certificates, always added to allowed hosts by NewManager.
}
```
AutocertConfig represents the configuration required to create an
autocert.Manager.


### Type ServiceFlags
```go
type ServiceFlags struct {
	ClientHost  string        `subcmd:"acme-client-host,,'host running the acme client responsible for refreshing certificates, https requests to this host for one of the certificate hosts will result in the certificate for the certificate host being refreshed if necessary'"`
	Provider    string        `subcmd:"acme-service,letsencrypt-staging,'the acme service to use, specify letsencrypt or letsencrypt-staging or a url'"`
	RenewBefore time.Duration `subcmd:"acme-renew-before,720h,how early certificates should be renewed before they expire."`
	Email       string        `subcmd:"acme-email,,email to contact for information on the domain"`
	UserAgent   string        `subcmd:"acme-user-agent,cloudeng.io/webapp/webauth/acme,'user agent to use when connecting to the acme service'"`
}
```
ServiceFlags represents the flags required to configure an ACME client
instance for managing TLS certificates for hosts/domains using the acme
http-01 challenge. Note that wildcard domains are not supported by this
challenge. The currently supported/tested acme service providers are
letsencrypt staging and production via the values 'letsencrypt-staging' and
'letsencrypt' for the --acme-service flag; however any URL can be specified
via this flag, in particular to use pebble for testing set this to the URL
of the local pebble instance and also set the --acme-testing-ca flag to
point to the pebble CA certificate pem file.

### Methods

```go
func (f ServiceFlags) AutocertConfig() AutocertConfig
```
AutocertConfig converts the flag values to a AutocertConfig instance.







