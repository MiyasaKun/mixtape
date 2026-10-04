package crypto_test

import (
	"strings"
	"testing"

	"fiber_test/internal/adapters/outbound/crypto"
)

// 32-byte key, base64("0123456789abcdef0123456789abcdef").
const testKey = "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY="

func TestEncryptRoundTrips(t *testing.T) {
	enc, err := crypto.New(testKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cipher, err := enc.Encrypt("refresh-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(cipher, "refresh-token") {
		t.Fatal("ciphertext leaks plaintext")
	}

	plain, err := enc.Decrypt(cipher)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plain != "refresh-token" {
		t.Errorf("want refresh-token, got %q", plain)
	}
}

func TestEncryptUsesFreshNonce(t *testing.T) {
	enc, _ := crypto.New(testKey)
	a, _ := enc.Encrypt("same")
	b, _ := enc.Encrypt("same")
	if a == b {
		t.Error("two encryptions of the same plaintext must differ")
	}
}

func TestNewRejectsWrongKeyLength(t *testing.T) {
	if _, err := crypto.New("dG9vc2hvcnQ="); err == nil {
		t.Error("want error for non-32-byte key")
	}
}

func TestDecryptRejectsTamperedCiphertext(t *testing.T) {
	enc, _ := crypto.New(testKey)
	cipher, _ := enc.Encrypt("refresh-token")
	if _, err := enc.Decrypt(cipher[:len(cipher)-2] + "AA"); err == nil {
		t.Error("want error for tampered ciphertext")
	}
}
