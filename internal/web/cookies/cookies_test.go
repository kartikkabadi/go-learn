package cookies

import (
	"testing"
)

func TestSignVerify(t *testing.T) {
	key := []byte("test-key-16bytes")
	value := "session-token-value"

	signed := Sign(value, key)
	if signed == value {
		t.Fatal("expected signed value to differ from raw value")
	}

	got, err := Verify(signed, key)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if got != value {
		t.Fatalf("expected %q, got %q", value, got)
	}
}

func TestVerifyTampered(t *testing.T) {
	key := []byte("test-key-16bytes")
	signed := Sign("token", key)

	_, err := Verify(signed+"x", key)
	if err == nil {
		t.Fatal("expected error for tampered cookie")
	}

	_, err = Verify("not-signed", key)
	if err == nil {
		t.Fatal("expected error for malformed cookie")
	}
}

func TestVerifyEmptyKey(t *testing.T) {
	value := "legacy-token"
	signed := Sign(value, nil)
	if signed != value {
		t.Fatal("expected no signing when key is empty")
	}

	got, err := Verify(value, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != value {
		t.Fatalf("expected %q, got %q", value, got)
	}
}
