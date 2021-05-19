package auth0

import (
	"fmt"
	"net/http"
	"net/url"

	jose "github.com/square/go-jose/v3"
	"github.com/square/go-jose/v3/json"
	"github.com/square/go-jose/v3/jwt"
)

const jwksEndpoint = "/.well-known/jwks.json"

// JWKS represents the KWT Key Set returned by auth0.com.
// See https://auth0.com/docs/tokens/json-web-tokens/json-web-key-set-properties
type JWKS struct {
	*jose.JSONWebKeySet
}

type Option func(*Authenticator)

func RS256() Option {
	return func(a *Authenticator) {
		a.algo = "RS256"
	}
}

func StaticJWKS(jwks *JWKS) Option {
	return func(a *Authenticator) {
		a.staticJWKS = jwks
	}
}

func ensureHTTPS(raw string) string {
	u, _ := url.Parse(raw)
	u.Scheme = "https"
	return u.String()
}

func NewAuthenticator(domain, audience string, opts ...Option) (*Authenticator, error) {
	a := &Authenticator{
		domain:   ensureHTTPS(domain),
		audience: ensureHTTPS(audience),
		cn:       "CN=" + domain,
	}
	for _, fn := range opts {
		fn(a)
	}
	if len(a.algo) == 0 {
		RS256()(a)
	}
	return a, a.refresh()
}

func (a *Authenticator) refresh() error {
	if a.staticJWKS != nil {
		a.jwks = a.staticJWKS
		return nil
	}
	jwks, err := JWKSForDomain(a.domain)
	if err != nil {
		return err
	}
	keys := []jose.JSONWebKey{}
	a.jwks = jwks
	for _, key := range jwks.Keys {
		if key.Algorithm != a.algo {
			continue
		}
		if key.Use != "sig" {
			continue
		}
		if len(key.Certificates) == 0 {
			continue
		}
		if key.Certificates[0].Issuer.String() != a.cn {
			continue
		}
		keys = append(keys, key)
	}
	a.jwks.Keys = keys
	return nil
}

type Authenticator struct {
	domain, audience string
	algo             string
	cn               string
	staticJWKS       *JWKS
	jwks             *JWKS
}

func (a *Authenticator) CheckJWT(token string) error {
	tok, err := jwt.ParseSigned(token)
	if err != nil {
		return fmt.Errorf("failed to parse jwt: %v", err)
	}
	claims := jwt.Claims{}
	if err := tok.Claims(a.jwks.JSONWebKeySet, &claims); err != nil {
		fmt.Printf("failed to obtain claims from jwt: %v\n", err)
		return err
	}
	expected := jwt.Expected{
		Issuer:   a.domain,
		Audience: []string{a.audience},
	}
	if err := claims.Validate(expected); err != nil {
		fmt.Printf("claims validation failed: %v: %v", expected, err)
		return err
	}
	return nil
}

func JWKSForDomain(tenant string) (*JWKS, error) {
	endpoint := ensureHTTPS(tenant) + jwksEndpoint
	resp, err := http.Get(endpoint)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to access %v: %v", endpoint, resp.StatusCode)
	}
	defer resp.Body.Close()
	var jwks = &JWKS{}
	if err := json.NewDecoder(resp.Body).Decode(jwks); err != nil {
		return nil, fmt.Errorf("failed to decode response from %v: %v", endpoint, err)
	}
	return jwks, nil
}

/*
func PublicKeyForID(cert *JWKS, kid string) (crypto.PublicKey, error) {
	pkms, err := PublicKeys(cert)
	if err != nil {
		return nil, err
	}
	pk, ok := pkms[kid]
	if !ok {
		return nil, fmt.Errorf("failed to find public key for %s", kid)
	}
	return pk, nil
}

func PublicKeys(cert *JWKS) (map[string]crypto.PublicKey, error) {
	pkm := map[string]crypto.PublicKey{}
	errs := errors.M{}
	for _, v := range cert.Keys {
		pemCert := v.Certificates[0] //formatCert(v.Certificates[0])
		pk, err := decodePK(pemCert)
		if err == nil {
			pkm[v.KeyID] = pk
		}
		errs.Append(err)
	}
	if len(pkm) == 0 {
		errs.Append(fmt.Errorf("failed to find any public keys in JWT Key Set"))
		return nil, errs.Err()
	}
	return pkm, nil
}
*/

/*
func formatCert(x5c string) string {
	return "-----BEGIN CERTIFICATE-----\n" + x5c + "\n-----END CERTIFICATE-----"
}


func decodePK(cert *x509.Certificate) (crypto.PublicKey, error) {
	fmt.Printf("CERT: %v\n", cert.Issuer)
	fmt.Printf("CERT: %v\n", cert.Subject)
	fmt.Printf("CERT: %v\n", cert.PublicKey)

	block, _ := pem.Decode(cert.Raw)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}
	errs := errors.M{}
	pk, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err == nil {
		return pk, nil
	}
	errs.Append(err)
	pcert, err := x509.ParseCertificate(block.Bytes)
	if err == nil {
		return pcert.PublicKey, nil
	}
	return nil, errs.Err()
}
*/
