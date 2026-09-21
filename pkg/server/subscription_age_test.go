package server

import (
	"bytes"
	"strings"
	"testing"
)

func TestAgeKeyGenerationAndDerivation(t *testing.T) {
	// 1. Test X25519 Key Generation
	secX, pubX, typeX, err := GenerateAgeKeyPair("x25519")
	if err != nil {
		t.Fatalf("failed to generate x25519 key: %v", err)
	}
	if !strings.HasPrefix(secX, "AGE-SECRET-KEY-1") {
		t.Errorf("expected x25519 secret key to start with AGE-SECRET-KEY-1, got: %s", secX)
	}
	if !strings.HasPrefix(pubX, "age1") || strings.HasPrefix(pubX, "age1pq") {
		t.Errorf("expected x25519 public key to start with age1, got: %s", pubX)
	}
	if typeX != "X25519" {
		t.Errorf("expected type X25519, got: %s", typeX)
	}

	// 2. Test X25519 Derivation
	derivedPubX, derivedTypeX, err := DeriveAgePublicKey(secX)
	if err != nil {
		t.Fatalf("failed to derive x25519 public key: %v", err)
	}
	if derivedPubX != pubX {
		t.Errorf("derived public key mismatch: %s vs %s", derivedPubX, pubX)
	}
	if derivedTypeX != "X25519" {
		t.Errorf("derived type mismatch: %s", derivedTypeX)
	}

	// 3. Test MLKEM768-X25519 Key Generation
	secPQ, pubPQ, typePQ, err := GenerateAgeKeyPair("mlkem768-x25519")
	if err != nil {
		t.Fatalf("failed to generate mlkem768-x25519 key: %v", err)
	}
	if !strings.HasPrefix(secPQ, "AGE-SECRET-KEY-PQ-1") {
		t.Errorf("expected hybrid secret key to start with AGE-SECRET-KEY-PQ-1, got: %s", secPQ)
	}
	if !strings.HasPrefix(pubPQ, "age1pq1") {
		t.Errorf("expected hybrid public key to start with age1pq1, got: %s", pubPQ)
	}
	if typePQ != "MLKEM768-X25519" {
		t.Errorf("expected type MLKEM768-X25519, got: %s", typePQ)
	}

	// 4. Test MLKEM768-X25519 Derivation
	derivedPubPQ, derivedTypePQ, err := DeriveAgePublicKey(secPQ)
	if err != nil {
		t.Fatalf("failed to derive mlkem768-x25519 public key: %v", err)
	}
	if derivedPubPQ != pubPQ {
		t.Errorf("derived hybrid public key mismatch: %s vs %s", derivedPubPQ, pubPQ)
	}
	if derivedTypePQ != "MLKEM768-X25519" {
		t.Errorf("derived hybrid type mismatch: %s", derivedTypePQ)
	}
}

func TestAgeEncryptionAndDecryptionRoundTrip(t *testing.T) {
	testPayload := []byte("vless://11111111-2222-3333-4444-555555555555@203.0.113.10:443?security=reality#Node1\nhysteria2://secret@203.0.113.10:8443#Node2")

	// 1. Test with X25519
	secX, pubX, _, err := GenerateAgeKeyPair("x25519")
	if err != nil {
		t.Fatalf("failed to generate x25519 key: %v", err)
	}

	encryptedX, err := EncryptWithAge(testPayload, pubX)
	if err != nil {
		t.Fatalf("failed to encrypt with X25519: %v", err)
	}
	if !bytes.HasPrefix(encryptedX, []byte("-----BEGIN AGE ENCRYPTED FILE-----")) {
		t.Fatalf("expected armor header, got:\n%s", string(encryptedX))
	}

	decryptedX, err := DecryptWithAge(encryptedX, secX)
	if err != nil {
		t.Fatalf("failed to decrypt with X25519: %v", err)
	}
	if !bytes.Equal(decryptedX, testPayload) {
		t.Fatalf("decrypted payload mismatch:\n%s\nvs\n%s", string(decryptedX), string(testPayload))
	}

	// 2. Test with MLKEM768-X25519
	secPQ, pubPQ, _, err := GenerateAgeKeyPair("mlkem768-x25519")
	if err != nil {
		t.Fatalf("failed to generate mlkem768-x25519 key: %v", err)
	}

	encryptedPQ, err := EncryptWithAge(testPayload, pubPQ)
	if err != nil {
		t.Fatalf("failed to encrypt with MLKEM768-X25519: %v", err)
	}
	if !bytes.HasPrefix(encryptedPQ, []byte("-----BEGIN AGE ENCRYPTED FILE-----")) {
		t.Fatalf("expected armor header in hybrid encryption, got:\n%s", string(encryptedPQ))
	}

	decryptedPQ, err := DecryptWithAge(encryptedPQ, secPQ)
	if err != nil {
		t.Fatalf("failed to decrypt with MLKEM768-X25519: %v", err)
	}
	if !bytes.Equal(decryptedPQ, testPayload) {
		t.Fatalf("decrypted hybrid payload mismatch:\n%s\nvs\n%s", string(decryptedPQ), string(testPayload))
	}

	// 3. Decrypt with wrong key must fail
	_, err = DecryptWithAge(encryptedX, secPQ)
	if err == nil {
		t.Fatalf("expected decryption with wrong key to fail")
	}
}
