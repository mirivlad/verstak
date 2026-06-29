// Package secrets provides encrypted local storage for secret values.
package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	keySize   = 32
	nonceSize = 12
)

// Store encrypts secret records before writing them to disk.
type Store struct {
	mu   sync.RWMutex
	root string
	key  []byte
}

type encryptedRecord struct {
	Version    int    `json:"version"`
	Nonce      []byte `json:"nonce"`
	Ciphertext []byte `json:"ciphertext"`
	UpdatedAt  string `json:"updatedAt"`
}

type plaintextRecord struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

// NewStore creates an encrypted secret store rooted at root.
func NewStore(root string, key []byte) (*Store, error) {
	if root == "" {
		return nil, fmt.Errorf("secret store root is empty")
	}
	if len(key) != keySize {
		return nil, fmt.Errorf("secret store key must be %d bytes", keySize)
	}

	copiedKey := make([]byte, keySize)
	copy(copiedKey, key)
	return &Store{
		root: root,
		key:  copiedKey,
	}, nil
}

// Write encrypts and stores a secret value by ID.
func (s *Store) Write(id, value string) error {
	if err := validateID(id); err != nil {
		return err
	}

	plaintext, err := json.Marshal(plaintextRecord{ID: id, Value: value})
	if err != nil {
		return fmt.Errorf("marshal secret: %w", err)
	}

	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return fmt.Errorf("generate nonce: %w", err)
	}

	aead, err := s.aead()
	if err != nil {
		return err
	}
	record := encryptedRecord{
		Version:    1,
		Nonce:      nonce,
		Ciphertext: aead.Seal(nil, nonce, plaintext, []byte(id)),
		UpdatedAt:  time.Now().UTC().Format(time.RFC3339),
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal encrypted secret: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	return atomicWrite0600(s.pathForID(id), data)
}

// Read decrypts and returns a secret value by ID.
func (s *Store) Read(id string) (string, error) {
	if err := validateID(id); err != nil {
		return "", err
	}

	s.mu.RLock()
	data, err := os.ReadFile(s.pathForID(id))
	s.mu.RUnlock()
	if err != nil {
		return "", fmt.Errorf("read secret %q: %w", id, err)
	}

	var record encryptedRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return "", fmt.Errorf("decode encrypted secret %q: %w", id, err)
	}
	if record.Version != 1 {
		return "", fmt.Errorf("unsupported secret version %d", record.Version)
	}

	aead, err := s.aead()
	if err != nil {
		return "", err
	}
	plaintext, err := aead.Open(nil, record.Nonce, record.Ciphertext, []byte(id))
	if err != nil {
		return "", fmt.Errorf("decrypt secret %q: %w", id, err)
	}

	var decoded plaintextRecord
	if err := json.Unmarshal(plaintext, &decoded); err != nil {
		return "", fmt.Errorf("decode secret %q: %w", id, err)
	}
	if decoded.ID != id {
		return "", fmt.Errorf("secret %q contains mismatched id", id)
	}
	return decoded.Value, nil
}

func (s *Store) aead() (cipher.AEAD, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, fmt.Errorf("create secret cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create secret gcm: %w", err)
	}
	return aead, nil
}

func (s *Store) pathForID(id string) string {
	sum := sha256.Sum256([]byte(id))
	return filepath.Join(s.root, hex.EncodeToString(sum[:])+".json")
}

func validateID(id string) error {
	if id == "" {
		return fmt.Errorf("secret id is empty")
	}
	if len(id) > 256 {
		return fmt.Errorf("secret id is too long")
	}
	if id == "." || id == ".." {
		return fmt.Errorf("secret id %q is a path traversal reference", id)
	}
	if strings.ContainsAny(id, `/\`) {
		return fmt.Errorf("secret id %q contains path separators", id)
	}
	if filepath.Clean(id) != id {
		return fmt.Errorf("secret id %q contains path traversal", id)
	}
	return nil
}

func atomicWrite0600(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create secret store dir: %w", err)
	}

	tmpFile := filepath.Join(dir, fmt.Sprintf(".tmp.%d", time.Now().UnixNano()))
	if err := os.WriteFile(tmpFile, data, 0o600); err != nil {
		return fmt.Errorf("write secret temp file: %w", err)
	}
	if err := os.Rename(tmpFile, path); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("commit secret file: %w", err)
	}
	return nil
}
