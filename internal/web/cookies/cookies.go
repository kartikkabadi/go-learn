// Package cookies provides HMAC-signed cookie helpers for session integrity.
package cookies

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
)

// Sign signs value with key using HMAC-SHA256 and returns "value.signature".
func Sign(value string, key []byte) string {
	if len(key) == 0 {
		return value
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(value))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return value + "." + sig
}

// Verify checks signed is "value.signature" and returns value if valid.
// If key is empty, it returns signed unchanged (legacy mode).
func Verify(signed string, key []byte) (string, error) {
	if len(key) == 0 {
		return signed, nil
	}
	parts := strings.SplitN(signed, ".", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid signed cookie")
	}
	value, sigB64 := parts[0], parts[1]
	expected := Sign(value, key)
	if !hmac.Equal([]byte(expected), []byte(signed)) {
		return "", fmt.Errorf("invalid cookie signature")
	}
	_ = sigB64 // signature already validated via constant-time comparison of full signed string
	return value, nil
}
