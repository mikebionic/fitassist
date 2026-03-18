package crypto

import (
	"testing"
)

func TestGenerateKey(t *testing.T) {
	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}
	// 32 bytes = 64 hex chars
	if len(key) != 64 {
		t.Errorf("expected key length 64, got %d", len(key))
	}

	// Generate another — should be different
	key2, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}
	if key == key2 {
		t.Error("two generated keys should not be equal")
	}
}

func TestEncryptDecrypt(t *testing.T) {
	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	plaintext := []byte("my-secret-password-123")

	ciphertext, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	if string(ciphertext) == string(plaintext) {
		t.Error("ciphertext should differ from plaintext")
	}

	decrypted, err := Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("decrypted %q != original %q", decrypted, plaintext)
	}
}

func TestDecryptWrongKey(t *testing.T) {
	key1, _ := GenerateKey()
	key2, _ := GenerateKey()

	ciphertext, err := Encrypt([]byte("secret"), key1)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	_, err = Decrypt(ciphertext, key2)
	if err == nil {
		t.Error("expected error decrypting with wrong key")
	}
}

func TestEncryptInvalidKey(t *testing.T) {
	_, err := Encrypt([]byte("test"), "not-hex")
	if err == nil {
		t.Error("expected error with invalid hex key")
	}

	// Too short key (16 bytes = 32 hex chars, need 32 bytes = 64 hex chars for AES-256)
	_, err = Encrypt([]byte("test"), "aabbccdd")
	if err == nil {
		t.Error("expected error with too-short key")
	}
}

func TestDecryptTooShort(t *testing.T) {
	key, _ := GenerateKey()
	_, err := Decrypt([]byte{1, 2}, key)
	if err == nil {
		t.Error("expected error with too-short ciphertext")
	}
}

func TestEncryptEmptyPlaintext(t *testing.T) {
	key, _ := GenerateKey()

	ciphertext, err := Encrypt([]byte{}, key)
	if err != nil {
		t.Fatalf("Encrypt empty: %v", err)
	}

	decrypted, err := Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("Decrypt empty: %v", err)
	}

	if len(decrypted) != 0 {
		t.Errorf("expected empty, got %q", decrypted)
	}
}
