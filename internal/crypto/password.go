package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

const (
	// ExportVersion is the current export format version
	ExportVersion = "1"
	// Algorithm specifies the encryption algorithm used
	Algorithm = "aes-256-gcm"
	// SaltLength is the length of the salt in bytes
	SaltLength = 16
	// KeyLength is the length of the derived key in bytes (256 bits)
	KeyLength = 32
	// PBKDF2Iterations is the number of iterations for PBKDF2
	PBKDF2Iterations = 100000
	// MinPasswordLength is the minimum required password length
	MinPasswordLength = 8
)

// ExportFormat represents the unencrypted export format
type ExportFormat struct {
	Version   string           `json:"version"`
	Timestamp int64            `json:"timestamp"`
	Configs   []ExportConfig   `json:"configs"`
}

// ExportConfig represents a single exported configuration
type ExportConfig struct {
	Alias     string   `json:"alias"`
	Provider  string   `json:"provider"`
	APIKey    string   `json:"api_key"`
	AuthToken string   `json:"auth_token"`
	BaseURL   string   `json:"base_url"`
	Model     string   `json:"model"`
	Models    []string `json:"models,omitempty"`
}

// EncryptedExport represents the encrypted export format
type EncryptedExport struct {
	Version   string `json:"version"`
	Algorithm string `json:"algorithm"`
	Salt      string `json:"salt"`
	Nonce     string `json:"nonce"`
	Data      string `json:"data"`
}

// PasswordManager handles password-based encryption and decryption
type PasswordManager struct {
	password string
}

// NewPasswordManager creates a new PasswordManager with the given password
func NewPasswordManager(password string) *PasswordManager {
	return &PasswordManager{
		password: password,
	}
}

// deriveKey derives a cryptographic key from the password and salt using PBKDF2
func (pm *PasswordManager) deriveKey(salt []byte) []byte {
	// Use PBKDF2 with HMAC-SHA256 to derive a key from the password
	// This is a slow, intentional process to make brute-force attacks expensive
	return pbkdf2.Key([]byte(pm.password), salt, PBKDF2Iterations, KeyLength, sha256.New)
}

// Encrypt encrypts an ExportFormat and returns a base64-encoded encrypted string
func (pm *PasswordManager) Encrypt(export *ExportFormat) (string, error) {
	// Serialize the export format to JSON
	plaintext, err := json.Marshal(export)
	if err != nil {
		return "", fmt.Errorf("failed to marshal export format: %w", err)
	}

	// Generate a random salt
	salt := make([]byte, SaltLength)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive the encryption key from the password and salt
	key := pm.deriveKey(salt)

	// Create AES cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode (Galois/Counter Mode - provides both encryption and authentication)
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate a random nonce (number used once)
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt the data with GCM (nonce is prepended to ciphertext)
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	// Build the encrypted export structure
	encrypted := EncryptedExport{
		Version:   ExportVersionVersion(),
		Algorithm: Algorithm,
		Salt:      base64.StdEncoding.EncodeToString(salt),
		Nonce:     base64.StdEncoding.EncodeToString(nonce),
		Data:      base64.StdEncoding.EncodeToString(ciphertext),
	}

	// Serialize and base64 encode the encrypted export
	encryptedJSON, err := json.Marshal(encrypted)
	if err != nil {
		return "", fmt.Errorf("failed to marshal encrypted export: %w", err)
	}

	return base64.StdEncoding.EncodeToString(encryptedJSON), nil
}

// Decrypt decrypts a base64-encoded encrypted string and returns an ExportFormat
func (pm *PasswordManager) Decrypt(encoded string) (*ExportFormat, error) {
	// Decode the base64 input
	encryptedJSON, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 input: %w", err)
	}

	// Parse the encrypted export structure
	var encrypted EncryptedExport
	if err := json.Unmarshal(encryptedJSON, &encrypted); err != nil {
		return nil, fmt.Errorf("failed to unmarshal encrypted export: %w", err)
	}

	// Verify version compatibility
	if encrypted.Version != ExportVersion {
		return nil, fmt.Errorf("unsupported export version: %s", encrypted.Version)
	}

	// Verify algorithm
	if encrypted.Algorithm != Algorithm {
		return nil, fmt.Errorf("unsupported algorithm: %s", encrypted.Algorithm)
	}

	// Decode salt, nonce, and ciphertext
	salt, err := base64.StdEncoding.DecodeString(encrypted.Salt)
	if err != nil {
		return nil, fmt.Errorf("failed to decode salt: %w", err)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encrypted.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	// Derive the encryption key from the password and salt
	key := pm.deriveKey(salt)

	// Create AES cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Verify nonce size
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	// Extract nonce and encrypted data
	nonce := ciphertext[:nonceSize]
	encryptedData := ciphertext[nonceSize:]

	// Decrypt the data (GCM will also verify the authentication tag)
	plaintext, err := gcm.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed (wrong password or corrupted data): %w", err)
	}

	// Parse the export format
	var export ExportFormat
	if err := json.Unmarshal(plaintext, &export); err != nil {
		return nil, fmt.Errorf("failed to unmarshal export format: %w", err)
	}

	return &export, nil
}

// ValidatePassword validates that the password meets security requirements
func ValidatePassword(password string) error {
	if len(password) < MinPasswordLength {
		return fmt.Errorf("password must be at least %d characters", MinPasswordLength)
	}
	return nil
}

// ExportVersionVersion returns the current export version
func ExportVersionVersion() string {
	return ExportVersion
}

// NewExportFormat creates a new ExportFormat from configurations
func NewExportFormat(configs []ExportConfig) *ExportFormat {
	return &ExportFormat{
		Version:   ExportVersion,
		Timestamp: 0, // Can be set to time.Now().Unix() if needed
		Configs:   configs,
	}
}

// NewExportConfig creates a new ExportConfig from API config values
func NewExportConfig(alias, provider, apiKey, authToken, baseURL, model string, models []string) ExportConfig {
	return ExportConfig{
		Alias:     alias,
		Provider:  provider,
		APIKey:    apiKey,
		AuthToken: authToken,
		BaseURL:   baseURL,
		Model:     model,
		Models:    models,
	}
}
