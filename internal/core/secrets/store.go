// Package secrets provides encrypted local storage for secret values.
package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
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
	keySize                 = 32
	nonceSize               = 12
	masterSaltSize          = 16
	masterPBKDF2Iterations  = 200000
	masterVerifierPlaintext = "verstak-secret-store:v1"
	masterMetadataVersion   = 1
	minMasterPasswordLength = 8
	recordsDirName          = "records"
	masterMetadataFileName  = "metadata.json"
	ScopeGlobal             = "global"
	ScopeWorkspace          = "workspace"
)

// Store encrypts secret records before writing them to disk.
type Store struct {
	mu   sync.RWMutex
	root string
	key  []byte
}

type SecretScope struct {
	Kind              string `json:"kind"`
	WorkspaceRootPath string `json:"workspaceRootPath,omitempty"`
}

type SecretRecord struct {
	ID        string      `json:"id"`
	Title     string      `json:"title"`
	Value     string      `json:"value,omitempty"`
	Scope     SecretScope `json:"scope"`
	Username  string      `json:"username,omitempty"`
	UpdatedAt string      `json:"updatedAt"`
}

type encryptedRecord struct {
	Version    int    `json:"version"`
	Nonce      []byte `json:"nonce"`
	Ciphertext []byte `json:"ciphertext"`
	UpdatedAt  string `json:"updatedAt"`
}

type plaintextRecord struct {
	ID        string      `json:"id"`
	Title     string      `json:"title,omitempty"`
	Value     string      `json:"value"`
	Scope     SecretScope `json:"scope"`
	Username  string      `json:"username,omitempty"`
	UpdatedAt string      `json:"updatedAt,omitempty"`
}

type masterMetadata struct {
	Version    int    `json:"version"`
	Salt       []byte `json:"salt"`
	Nonce      []byte `json:"nonce"`
	Ciphertext []byte `json:"ciphertext"`
	CreatedAt  string `json:"createdAt"`
}

type VaultSession struct {
	mu    sync.RWMutex
	root  string
	store *Store
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
	return s.WriteRecord(SecretRecord{
		ID:    id,
		Title: id,
		Value: value,
		Scope: SecretScope{Kind: ScopeGlobal},
	})
}

func (s *Store) WriteRecord(record SecretRecord) error {
	record.ID = strings.TrimSpace(record.ID)
	record.Title = strings.TrimSpace(record.Title)
	record.Scope.Kind = strings.TrimSpace(record.Scope.Kind)
	record.Scope.WorkspaceRootPath = cleanWorkspaceRoot(record.Scope.WorkspaceRootPath)
	if err := validateRecord(record); err != nil {
		return err
	}
	record.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	plaintext, err := json.Marshal(plaintextRecord{
		ID:        record.ID,
		Title:     record.Title,
		Value:     record.Value,
		Scope:     record.Scope,
		Username:  record.Username,
		UpdatedAt: record.UpdatedAt,
	})
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
	encrypted := encryptedRecord{
		Version:    1,
		Nonce:      nonce,
		Ciphertext: aead.Seal(nil, nonce, plaintext, nil),
		UpdatedAt:  time.Now().UTC().Format(time.RFC3339),
	}
	data, err := json.MarshalIndent(encrypted, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal encrypted secret: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	return atomicWrite0600(s.pathForID(record.ID), data)
}

// Read decrypts and returns a secret value by ID.
func (s *Store) Read(id string) (string, error) {
	record, err := s.ReadRecord(id)
	if err != nil {
		return "", err
	}
	return record.Value, nil
}

func (s *Store) ReadRecord(id string) (SecretRecord, error) {
	if err := validateID(id); err != nil {
		return SecretRecord{}, err
	}

	s.mu.RLock()
	data, err := os.ReadFile(s.pathForID(id))
	s.mu.RUnlock()
	if err != nil {
		return SecretRecord{}, fmt.Errorf("read secret %q: %w", id, err)
	}

	decoded, err := s.decryptRecord(id, data)
	if err != nil {
		return SecretRecord{}, err
	}
	return decoded, nil
}

func (s *Store) Delete(id string) error {
	if err := validateID(id); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.Remove(s.pathForID(id)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete secret %q: %w", id, err)
	}
	return nil
}

func (s *Store) ListRecords() ([]SecretRecord, error) {
	s.mu.RLock()
	entries, err := os.ReadDir(s.root)
	s.mu.RUnlock()
	if err != nil {
		if os.IsNotExist(err) {
			return []SecretRecord{}, nil
		}
		return nil, fmt.Errorf("list secrets: %w", err)
	}

	records := make([]SecretRecord, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(s.root, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read secret list item %s: %w", entry.Name(), err)
		}
		record, err := s.decryptRecord("", data)
		if err != nil {
			return nil, fmt.Errorf("decrypt secret list item %s: %w", entry.Name(), err)
		}
		record.Value = ""
		records = append(records, record)
	}
	return records, nil
}

func (s *Store) decryptRecord(expectedID string, data []byte) (SecretRecord, error) {
	var record encryptedRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return SecretRecord{}, fmt.Errorf("decode encrypted secret %q: %w", expectedID, err)
	}
	if record.Version != 1 {
		return SecretRecord{}, fmt.Errorf("unsupported secret version %d", record.Version)
	}
	aead, err := s.aead()
	if err != nil {
		return SecretRecord{}, err
	}
	plaintext, err := aead.Open(nil, record.Nonce, record.Ciphertext, nil)
	if err != nil {
		return SecretRecord{}, fmt.Errorf("decrypt secret %q: %w", expectedID, err)
	}

	var decoded plaintextRecord
	if err := json.Unmarshal(plaintext, &decoded); err != nil {
		return SecretRecord{}, fmt.Errorf("decode secret %q: %w", expectedID, err)
	}
	if expectedID != "" && decoded.ID != expectedID {
		return SecretRecord{}, fmt.Errorf("secret %q contains mismatched id", expectedID)
	}
	return SecretRecord{
		ID:        decoded.ID,
		Title:     decoded.Title,
		Value:     decoded.Value,
		Scope:     decoded.Scope,
		Username:  decoded.Username,
		UpdatedAt: decoded.UpdatedAt,
	}, nil
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

func validateRecord(record SecretRecord) error {
	if err := validateID(record.ID); err != nil {
		return err
	}
	if record.Title == "" {
		return fmt.Errorf("secret title is empty")
	}
	switch record.Scope.Kind {
	case ScopeGlobal:
		if record.Scope.WorkspaceRootPath != "" {
			return fmt.Errorf("global secret must not have workspace root path")
		}
	case ScopeWorkspace:
		if record.Scope.WorkspaceRootPath == "" {
			return fmt.Errorf("workspace secret requires workspace root path")
		}
		if strings.ContainsAny(record.Scope.WorkspaceRootPath, `\`) {
			return fmt.Errorf("workspace root path contains path separators")
		}
	default:
		return fmt.Errorf("unsupported secret scope %q", record.Scope.Kind)
	}
	return nil
}

func cleanWorkspaceRoot(value string) string {
	return strings.Trim(strings.TrimSpace(value), "/")
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

func NewVaultSession(root string) *VaultSession {
	return &VaultSession{root: root}
}

func (s *VaultSession) Unlocked() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.store != nil
}

func (s *VaultSession) Store() (*Store, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.store == nil {
		return nil, fmt.Errorf("secret store locked")
	}
	return s.store, nil
}

func (s *VaultSession) Initialized() (bool, error) {
	metadata, err := readMasterMetadata(s.root)
	if err != nil {
		return false, err
	}
	return metadata != nil, nil
}

func (s *VaultSession) Unlock(masterPassword string) (*Store, error) {
	if strings.TrimSpace(masterPassword) == "" {
		return nil, fmt.Errorf("master password is empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.store != nil {
		return s.store, nil
	}

	metadata, err := readMasterMetadata(s.root)
	if err != nil {
		return nil, err
	}
	if metadata == nil {
		if err := validateInitialMasterPassword(masterPassword); err != nil {
			return nil, err
		}
		created, key, err := createMasterMetadata(masterPassword)
		if err != nil {
			return nil, err
		}
		if err := writeMasterMetadata(s.root, *created); err != nil {
			return nil, err
		}
		store, err := NewStore(filepath.Join(s.root, recordsDirName), key)
		if err != nil {
			return nil, err
		}
		s.store = store
		return store, nil
	}

	key := deriveMasterKey(masterPassword, metadata.Salt)
	if err := verifyMasterMetadata(*metadata, key); err != nil {
		return nil, err
	}
	store, err := NewStore(filepath.Join(s.root, recordsDirName), key)
	if err != nil {
		return nil, err
	}
	s.store = store
	return store, nil
}

func validateInitialMasterPassword(masterPassword string) error {
	if len([]rune(masterPassword)) < minMasterPasswordLength {
		return fmt.Errorf("master password must be at least %d characters", minMasterPasswordLength)
	}
	return nil
}

func readMasterMetadata(root string) (*masterMetadata, error) {
	data, err := os.ReadFile(filepath.Join(root, masterMetadataFileName))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read secret metadata: %w", err)
	}
	var metadata masterMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("decode secret metadata: %w", err)
	}
	if metadata.Version != masterMetadataVersion {
		return nil, fmt.Errorf("unsupported secret metadata version %d", metadata.Version)
	}
	return &metadata, nil
}

func createMasterMetadata(masterPassword string) (*masterMetadata, []byte, error) {
	salt := make([]byte, masterSaltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, nil, fmt.Errorf("generate secret salt: %w", err)
	}
	key := deriveMasterKey(masterPassword, salt)
	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, fmt.Errorf("generate verifier nonce: %w", err)
	}
	aead, err := newAEAD(key)
	if err != nil {
		return nil, nil, err
	}
	return &masterMetadata{
		Version:    masterMetadataVersion,
		Salt:       salt,
		Nonce:      nonce,
		Ciphertext: aead.Seal(nil, nonce, []byte(masterVerifierPlaintext), nil),
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}, key, nil
}

func verifyMasterMetadata(metadata masterMetadata, key []byte) error {
	aead, err := newAEAD(key)
	if err != nil {
		return err
	}
	plaintext, err := aead.Open(nil, metadata.Nonce, metadata.Ciphertext, nil)
	if err != nil {
		return fmt.Errorf("invalid master password")
	}
	if !hmac.Equal(plaintext, []byte(masterVerifierPlaintext)) {
		return fmt.Errorf("invalid master password")
	}
	return nil
}

func writeMasterMetadata(root string, metadata masterMetadata) error {
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal secret metadata: %w", err)
	}
	return atomicWrite0600(filepath.Join(root, masterMetadataFileName), data)
}

func deriveMasterKey(masterPassword string, salt []byte) []byte {
	return pbkdf2SHA256([]byte(masterPassword), salt, masterPBKDF2Iterations, keySize)
}

func newAEAD(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create secret cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create secret gcm: %w", err)
	}
	return aead, nil
}

func pbkdf2SHA256(password, salt []byte, iterations, size int) []byte {
	if iterations <= 0 || size <= 0 {
		return nil
	}
	hashLen := sha256.Size
	blocks := (size + hashLen - 1) / hashLen
	output := make([]byte, 0, blocks*hashLen)
	for block := 1; block <= blocks; block++ {
		mac := hmac.New(sha256.New, password)
		mac.Write(salt)
		mac.Write([]byte{byte(block >> 24), byte(block >> 16), byte(block >> 8), byte(block)})
		u := mac.Sum(nil)
		t := make([]byte, len(u))
		copy(t, u)
		for i := 1; i < iterations; i++ {
			mac = hmac.New(sha256.New, password)
			mac.Write(u)
			u = mac.Sum(nil)
			for j := range t {
				t[j] ^= u[j]
			}
		}
		output = append(output, t...)
	}
	return output[:size]
}
