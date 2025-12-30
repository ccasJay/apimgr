package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// EncryptedPrefix is the prefix used to identify encrypted values
const EncryptedPrefix = "ENC:"

// KeyManager handles encryption and decryption of API keys
type KeyManager struct {
	key []byte
}

// NewKeyManager creates a new KeyManager with a derived key
func NewKeyManager() (*KeyManager, error) {
	// Derive key from machine-specific data
	key, err := deriveKey()
	if err != nil {
		return nil, fmt.Errorf("failed to derive encryption key: %w", err)
	}

	return &KeyManager{
		key: key,
	}, nil
}

// deriveKey generates an encryption key based on machine-specific data
func deriveKey() ([]byte, error) {
	// Use a combination of machine ID and user home directory for key derivation
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	// Get hostname for additional entropy
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}

	// Create a machine-specific seed with multiple factors
	seed := fmt.Sprintf("%s-%s-apimgr-encryption-key-v1", homeDir, hostname)
	
	// Generate a 32-byte key using SHA-256
	hash := sha256.Sum256([]byte(seed))
	return hash[:], nil
}

// Encrypt encrypts the plaintext API key
func (km *KeyManager) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(km.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Create a nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt the data
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	
	// Encode to base64 and add prefix for identification
	return EncryptedPrefix + base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts the encrypted API key
func (km *KeyManager) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	// Remove prefix if present
	data := ciphertext
	if strings.HasPrefix(ciphertext, EncryptedPrefix) {
		data = strings.TrimPrefix(ciphertext, EncryptedPrefix)
	}

	// Decode from base64
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	block, err := aes.NewCipher(km.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(decoded) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertextBytes := decoded[:nonceSize], decoded[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// IsEncrypted checks if a string appears to be encrypted
// Uses the ENC: prefix for reliable detection
func IsEncrypted(value string) bool {
	if value == "" {
		return false
	}
	
	// Check for the encryption prefix
	if !strings.HasPrefix(value, EncryptedPrefix) {
		return false
	}
	
	// Verify the base64 part is valid
	data := strings.TrimPrefix(value, EncryptedPrefix)
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return false
	}
	
	// Encrypted data should have at least nonce + some ciphertext
	// AES-GCM nonce is 12 bytes, plus at least some encrypted data
	return len(decoded) >= 20
}

// GetOrCreateKeyFile gets or creates a key file for additional security (optional)
func GetOrCreateKeyFile() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	keyDir := filepath.Join(homeDir, ".config", "apimgr", ".keys")
	if err := os.MkdirAll(keyDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create key directory: %w", err)
	}

	keyFile := filepath.Join(keyDir, ".master.key")
	
	// Check if key file exists
	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		// Generate a new key
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return "", fmt.Errorf("failed to generate key: %w", err)
		}
		
		// Write key to file with restricted permissions
		if err := os.WriteFile(keyFile, key, 0600); err != nil {
			return "", fmt.Errorf("failed to write key file: %w", err)
		}
	}
	
	return keyFile, nil
}
