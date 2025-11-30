package h2go

import (
	"crypto/hmac"
	"crypto/sha1"
	"fmt"
)

// HMACAuthenticator implements the Authenticator interface using HMAC-SHA1.
// It provides secure request signing and verification using a shared secret.
type HMACAuthenticator struct {
	secret string
}

// Ensure HMACAuthenticator implements the Authenticator interface.
var _ Authenticator = (*HMACAuthenticator)(nil)

// NewHMACAuthenticator creates a new HMACAuthenticator with the given secret.
func NewHMACAuthenticator(secret string) *HMACAuthenticator {
	return &HMACAuthenticator{secret: secret}
}

// Sign generates an HMAC-SHA1 signature for the given data.
func (a *HMACAuthenticator) Sign(data string) string {
	return GenHMACSHA1(a.secret, data)
}

// Verify checks if the provided signature is valid for the given data.
func (a *HMACAuthenticator) Verify(data, signature string) bool {
	return VerifyHMACSHA1(a.secret, data, signature)
}

// GenHMACSHA1 generates an HMAC-SHA1 signature for the given key and data.
// This is a low-level function; prefer using HMACAuthenticator for most use cases.
func GenHMACSHA1(key, raw string) string {
	k := []byte(key)
	mac := hmac.New(sha1.New, k)
	mac.Write([]byte(raw))
	return fmt.Sprintf("%x", mac.Sum(nil))
}

// VerifyHMACSHA1 verifies an HMAC-SHA1 signature.
// This is a low-level function; prefer using HMACAuthenticator for most use cases.
func VerifyHMACSHA1(key, raw, sign string) bool {
	return GenHMACSHA1(key, raw) == sign
}
