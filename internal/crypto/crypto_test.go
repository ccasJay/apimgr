package crypto

import (
	"strings"
	"testing"
)

func TestKeyManager(t *testing.T) {
	km, err := NewKeyManager()
	if err != nil {
		t.Fatalf("Failed to create KeyManager: %v", err)
	}

	t.Run("Encrypt and Decrypt", func(t *testing.T) {
		testCases := []struct {
			name      string
			plaintext string
		}{
			{"Empty string", ""},
			{"Short API key", "test-123"},
			{"Normal API key", "test-fake-key-abcdefghijklmnopqrstuvwxyz"},
			{"Long API key", "test-proj-verylongfakekeywithalotofcharacters1234567890abcdefghijklmnopqrstuvwxyz"},
			{"Special characters", "test-!@#$%^&*()_+-=[]{}|;:',.<>?/~`"},
			{"Unicode characters", "test-🔑-こんにちは-世界"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Encrypt
				encrypted, err := km.Encrypt(tc.plaintext)
				if err != nil {
					t.Errorf("Encrypt failed: %v", err)
					return
				}

				// Empty plaintext should result in empty ciphertext
				if tc.plaintext == "" {
					if encrypted != "" {
						t.Errorf("Expected empty ciphertext for empty plaintext, got %q", encrypted)
					}
					return
				}

				// Ciphertext should be different from plaintext
				if encrypted == tc.plaintext {
					t.Errorf("Encrypted value should differ from plaintext")
				}

				// Decrypt
				decrypted, err := km.Decrypt(encrypted)
				if err != nil {
					t.Errorf("Decrypt failed: %v", err)
					return
				}

				// Decrypted should match original
				if decrypted != tc.plaintext {
					t.Errorf("Decrypted value %q doesn't match original %q", decrypted, tc.plaintext)
				}
			})
		}
	})

	t.Run("Different encryptions of same plaintext", func(t *testing.T) {
		plaintext := "test-fake-key-test"
		
		encrypted1, err := km.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("First encryption failed: %v", err)
		}

		encrypted2, err := km.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("Second encryption failed: %v", err)
		}

		// Due to random nonce, same plaintext should produce different ciphertexts
		if encrypted1 == encrypted2 {
			t.Errorf("Same plaintext produced identical ciphertexts (should use random nonce)")
		}

		// But both should decrypt to the same plaintext
		decrypted1, _ := km.Decrypt(encrypted1)
		decrypted2, _ := km.Decrypt(encrypted2)

		if decrypted1 != plaintext || decrypted2 != plaintext {
			t.Errorf("Decrypted values don't match original plaintext")
		}
	})

	t.Run("Invalid ciphertext", func(t *testing.T) {
		invalidCiphertexts := []string{
			"not-base64!@#$",
			"aGVsbG8=", // Valid base64 but too short for encrypted data
			"invalid",
		}

		for _, invalid := range invalidCiphertexts {
			_, err := km.Decrypt(invalid)
			if err == nil {
				t.Errorf("Expected error when decrypting invalid ciphertext %q", invalid)
			}
		}
	})
}

func TestIsEncrypted(t *testing.T) {
	km, err := NewKeyManager()
	if err != nil {
		t.Fatalf("Failed to create KeyManager: %v", err)
	}

	t.Run("Detect encrypted values", func(t *testing.T) {
		plaintext := "test-fake-key-test"
		encrypted, err := km.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("Encryption failed: %v", err)
		}

		if !IsEncrypted(encrypted) {
			t.Errorf("Failed to detect encrypted value")
		}

		// Verify encrypted value has the prefix
		if !strings.HasPrefix(encrypted, EncryptedPrefix) {
			t.Errorf("Encrypted value should have prefix %q, got %q", EncryptedPrefix, encrypted)
		}
	})

	t.Run("Detect non-encrypted values", func(t *testing.T) {
		nonEncrypted := []string{
			"",
			"test-fake-key-plaintext",
			"not-base64!@#$",
			"aGVsbG8=", // Valid base64 but no prefix
			"sk-ant-api03-abcdefghijklmnopqrstuvwxyz", // Looks like API key
		}

		for _, value := range nonEncrypted {
			if IsEncrypted(value) {
				t.Errorf("Incorrectly detected %q as encrypted", value)
			}
		}
	})

	t.Run("Prefix detection", func(t *testing.T) {
		// Value with prefix but invalid base64 should not be detected
		invalidWithPrefix := EncryptedPrefix + "not-valid-base64!@#"
		if IsEncrypted(invalidWithPrefix) {
			t.Errorf("Should not detect invalid base64 with prefix as encrypted")
		}

		// Value with prefix but too short should not be detected
		shortWithPrefix := EncryptedPrefix + "aGVsbG8="
		if IsEncrypted(shortWithPrefix) {
			t.Errorf("Should not detect too short data with prefix as encrypted")
		}
	})
}

func TestKeyDerivation(t *testing.T) {
	// Test that key derivation is consistent
	key1, err := deriveKey()
	if err != nil {
		t.Fatalf("First key derivation failed: %v", err)
	}

	key2, err := deriveKey()
	if err != nil {
		t.Fatalf("Second key derivation failed: %v", err)
	}

	// Keys should be identical when derived multiple times
	if string(key1) != string(key2) {
		t.Errorf("Key derivation is not consistent")
	}

	// Key should be 32 bytes for AES-256
	if len(key1) != 32 {
		t.Errorf("Expected 32-byte key, got %d bytes", len(key1))
	}
}

func TestKeyManagerConsistency(t *testing.T) {
	// Create two KeyManagers - they should use the same derived key
	km1, err := NewKeyManager()
	if err != nil {
		t.Fatalf("Failed to create first KeyManager: %v", err)
	}

	km2, err := NewKeyManager()
	if err != nil {
		t.Fatalf("Failed to create second KeyManager: %v", err)
	}

	plaintext := "test-fake-key-consistency-test"
	
	// Encrypt with first manager
	encrypted, err := km1.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	// Decrypt with second manager
	decrypted, err := km2.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Cross-manager encryption/decryption failed")
	}
}

func BenchmarkEncrypt(b *testing.B) {
	km, _ := NewKeyManager()
	plaintext := "test-fake-key-benchmark-test-key-123456789"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = km.Encrypt(plaintext)
	}
}

func BenchmarkDecrypt(b *testing.B) {
	km, _ := NewKeyManager()
	plaintext := "test-fake-key-benchmark-test-key-123456789"
	encrypted, _ := km.Encrypt(plaintext)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = km.Decrypt(encrypted)
	}
}

func TestGetOrCreateKeyFile(t *testing.T) {
	// This test might need to be skipped in CI environments
	t.Run("Create and retrieve key file", func(t *testing.T) {
		keyFile1, err := GetOrCreateKeyFile()
		if err != nil {
			// Skip test if we can't create key file (might be in CI)
			if strings.Contains(err.Error(), "permission denied") {
				t.Skip("Skipping key file test - no permission to create files")
			}
			t.Fatalf("Failed to get/create key file: %v", err)
		}

		// Second call should return the same file
		keyFile2, err := GetOrCreateKeyFile()
		if err != nil {
			t.Fatalf("Failed to get existing key file: %v", err)
		}

		if keyFile1 != keyFile2 {
			t.Errorf("Key file paths don't match: %q vs %q", keyFile1, keyFile2)
		}
	})
}
