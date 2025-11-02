// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package certcache

import (
	"context"
	"crypto"
	"encoding/pem"
	"fmt"
	"strings"

	"cloudeng.io/errors"
	"cloudeng.io/webapp"
	"golang.org/x/crypto/acme"
)

// GetAccountKey retrieves the ACME account private key from the cache.
func (dc *CachingStore) GetAccountKey(ctx context.Context) (crypto.Signer, error) {
	var keyData []byte
	for _, kn := range []string{"acme_account+key", "acme_account.key"} {
		if data, err := dc.Get(ctx, kn); err == nil {
			keyData = data
			break
		}
	}
	if keyData == nil {
		return nil, fmt.Errorf("no account key found in cert store")
	}
	priv, _ := pem.Decode(keyData)
	if priv == nil || !strings.Contains(priv.Type, "PRIVATE") {
		return nil, errors.New("acme/autocert: invalid account key found in cache")
	}
	return webapp.ParsePrivateKeyDER(priv.Bytes)
}

// ParseRevocationReason parses the supplied revocation reason string
// and returns the corresponding acme.CRLReasonCode.
func ParseRevocationReason(reason string) (acme.CRLReasonCode, error) {
	switch reason {
	case "", "unspecified":
		return acme.CRLReasonUnspecified, nil
	case "keyCompromise":
		return acme.CRLReasonKeyCompromise, nil
	case "affiliationChanged":
		return acme.CRLReasonAffiliationChanged, nil
	case "superseded":
		return acme.CRLReasonSuperseded, nil
	case "cessationOfOperation":
		return acme.CRLReasonCessationOfOperation, nil
	case "certificateHold":
		return acme.CRLReasonCertificateHold, nil
	default:
		return acme.CRLReasonUnspecified, fmt.Errorf("unknown revocation reason: %q", reason)
	}
}
