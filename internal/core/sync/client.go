package sync

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DeviceTokenPath returns the path to the device_token file inside the vault.
func DeviceTokenPath(vaultRoot string) string {
	return filepath.Join(vaultRoot, ".verstak", "device_token.json")
}

// SaveDeviceToken writes the device token to a file with 0600 perms.
func SaveDeviceToken(vaultRoot, token string) error {
	path := DeviceTokenPath(vaultRoot)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	data := fmt.Sprintf(`{"device_token":%q}`, token)
	return os.WriteFile(path, []byte(data), 0o600)
}

// LoadDeviceToken reads the device token from the vault.
func LoadDeviceToken(vaultRoot string) string {
	path := DeviceTokenPath(vaultRoot)
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var v struct {
		DeviceToken string `json:"device_token"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return ""
	}
	return v.DeviceToken
}

// RemoveDeviceToken deletes the device token file.
func RemoveDeviceToken(vaultRoot string) error {
	path := DeviceTokenPath(vaultRoot)
	return os.Remove(path)
}

// Client communicates with the Verstak Sync Server.
type Client struct {
	ServerURL   string
	APIKey      string
	DeviceToken string
	DeviceID    string
	VaultRoot   string
	HTTP        *http.Client
}

var syncRetrySleep = time.Sleep

const syncHTTPAttempts = 3

// NewClient creates a sync client.
func NewClient(serverURL, apiKey, deviceID, vaultRoot string) *Client {
	return &Client{
		ServerURL: serverURL,
		APIKey:    apiKey,
		DeviceID:  deviceID,
		VaultRoot: vaultRoot,
		HTTP:      &http.Client{Timeout: 30 * time.Second},
	}
}

// PairDevice calls POST /api/client/pair and returns device_id and device_token.
func (c *Client) PairDevice(serverURL, username, password, deviceName, clientVersion, vaultID string) (deviceID, deviceToken string, err error) {
	body := map[string]string{
		"login":          username,
		"password":       password,
		"device_name":    deviceName,
		"client_version": clientVersion,
		"vault_id":       vaultID,
	}
	var resp struct {
		DeviceID    string `json:"device_id"`
		DeviceToken string `json:"device_token"`
	}
	savedURL := c.ServerURL
	c.ServerURL = serverURL
	err = c.post("/api/client/pair", body, &resp)
	c.ServerURL = savedURL
	if err != nil {
		return "", "", err
	}
	return resp.DeviceID, resp.DeviceToken, nil
}

// DeviceInfo holds device information from the server.
type DeviceInfo struct {
	DeviceID      string `json:"device_id"`
	UserID        string `json:"user_id"`
	Username      string `json:"username"`
	DeviceName    string `json:"device_name"`
	ClientVersion string `json:"client_version"`
	LastSeen      string `json:"last_seen"`
	RevokedAt     string `json:"revoked_at"`
	CreatedAt     string `json:"created_at"`
}

// GetMe calls GET /api/client/me and returns device info.
func (c *Client) GetMe() (*DeviceInfo, error) {
	var resp DeviceInfo
	if err := c.get("/api/client/me", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// RevokeCurrent calls POST /api/client/revoke-current.
func (c *Client) RevokeCurrent() error {
	var resp struct {
		Status string `json:"status"`
	}
	return c.post("/api/client/revoke-current", nil, &resp)
}

// TestAuth checks credentials without creating a device.
func (c *Client) TestAuth(serverURL, username, password string) error {
	// First, check if this is a Verstak Sync server
	healthURL := strings.TrimSuffix(serverURL, "/") + "/api/v1/health"
	req, err := http.NewRequest("GET", healthURL, nil)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("not a Verstak Sync server (HTTP %d)", resp.StatusCode)
	}

	data, _ := io.ReadAll(resp.Body)
	body := string(data)
	if !strings.Contains(body, "status") && !strings.Contains(body, "ok") {
		return fmt.Errorf("not a Verstak Sync server (unexpected response)")
	}

	// Now test actual auth
	authBody := map[string]string{"username": username, "password": password}
	savedURL := c.ServerURL
	savedKey := c.APIKey
	c.ServerURL = serverURL
	c.APIKey = ""
	err = c.post("/api/auth/test", authBody, nil)
	c.ServerURL = savedURL
	c.APIKey = savedKey
	return err
}

// PushRequest is the payload for POST /sync/push.
type PushRequest struct {
	DeviceID       string   `json:"device_id"`
	IdempotencyKey string   `json:"idempotency_key,omitempty"`
	Ops            []PushOp `json:"ops"`
}

// PushOp is a single operation in a push request.
type PushOp struct {
	OpID              string `json:"op_id"`
	EntityType        string `json:"entity_type"`
	EntityID          string `json:"entity_id"`
	OpType            string `json:"op_type"`
	PayloadJSON       string `json:"payload_json"`
	ClientSequence    int    `json:"client_sequence"`
	LastSeenServerSeq int    `json:"last_seen_server_seq"`
	CreatedAt         string `json:"created_at"`
}

// PushResponse is the response from POST /sync/push.
type PushResponse struct {
	Accepted  []string                 `json:"accepted"`
	Count     int                      `json:"count"`
	Conflicts []map[string]interface{} `json:"conflicts"`
}

// Push sends local operations to the server.
func (c *Client) Push(ops []Op) (*PushResponse, error) {
	pushOps := make([]PushOp, len(ops))
	for i, op := range ops {
		pushOps[i] = PushOp{
			OpID:              op.OpID,
			EntityType:        op.EntityType,
			EntityID:          op.EntityID,
			OpType:            op.OpType,
			PayloadJSON:       op.PayloadJSON,
			ClientSequence:    op.ClientSequence,
			LastSeenServerSeq: op.LastSeenServerSeq,
			CreatedAt:         op.CreatedAt,
		}
	}
	req := PushRequest{DeviceID: c.DeviceID, Ops: pushOps}
	var resp PushResponse
	if err := c.post("/api/v1/sync/push", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// PullRequest is the payload for POST /sync/pull.
type PullRequest struct {
	SinceSequence int `json:"since_sequence"`
	PageLimit     int `json:"page_limit,omitempty"`
}

// PullResponse is the response from POST /sync/pull.
type PullResponse struct {
	ServerSequence   int  `json:"server_sequence"`
	PageLastSequence int  `json:"page_last_sequence"`
	HasMore          bool `json:"has_more"`
	Ops              []Op `json:"ops"`
}

// BlobReference is the only binary content representation permitted inside a
// sync operation. The bytes travel via the Blob API, never payload_json.
type BlobReference struct {
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

// ServerError carries a public stable error code. UI layers map Code to their
// own localized wording and must not rely on a server diagnostic string.
type ServerError struct {
	Status int
	Code   string
}

func (e *ServerError) Error() string {
	if e.Code == "" {
		return fmt.Sprintf("sync-server:request_failed (HTTP %d)", e.Status)
	}
	return fmt.Sprintf("sync-server:%s (HTTP %d)", e.Code, e.Status)
}

// Pull fetches remote operations since a given sequence.
func (c *Client) Pull(sinceSequence int) (*PullResponse, error) {
	return c.PullPage(sinceSequence, 0)
}

// PullPage fetches one bounded ordered page. The caller advances its durable
// cursor only after every returned operation has applied successfully.
func (c *Client) PullPage(sinceSequence, pageLimit int) (*PullResponse, error) {
	req := PullRequest{SinceSequence: sinceSequence, PageLimit: pageLimit}
	var resp PullResponse
	if err := c.post("/api/v1/sync/pull", req, &resp); err != nil {
		return nil, err
	}
	// Servers before pull pagination did not include page_last_sequence. Keep
	// the desktop compatible during rolling upgrades without ever advancing
	// beyond an operation actually present in the response.
	if resp.PageLastSequence == 0 && len(resp.Ops) > 0 {
		resp.PageLastSequence = resp.Ops[len(resp.Ops)-1].ServerSequence
	}
	return &resp, nil
}

// UploadBlob streams a local file through a multipart pipe. It keeps the
// process memory bounded even when the file is many times larger than an
// inline sync payload.
func (c *Client) UploadBlob(localPath string) (BlobReference, error) {
	info, err := os.Stat(localPath)
	if err != nil {
		return BlobReference{}, err
	}
	if !info.Mode().IsRegular() {
		return BlobReference{}, fmt.Errorf("blob source is not a regular file")
	}
	reader, writer := io.Pipe()
	multipartWriter := multipart.NewWriter(writer)
	writeDone := make(chan error, 1)
	go func() {
		defer func() {
			_ = writer.Close()
		}()
		part, err := multipartWriter.CreateFormFile("file", filepath.Base(localPath))
		if err == nil {
			file, openErr := os.Open(localPath)
			if openErr != nil {
				err = openErr
			} else {
				_, err = io.Copy(part, file)
				closeErr := file.Close()
				if err == nil {
					err = closeErr
				}
			}
		}
		if closeErr := multipartWriter.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			_ = writer.CloseWithError(err)
		}
		writeDone <- err
	}()

	req, err := http.NewRequest("POST", c.ServerURL+"/api/v1/blobs/", reader)
	if err != nil {
		_ = reader.Close()
		return BlobReference{}, err
	}
	req.Header.Set("Content-Type", multipartWriter.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+c.bearerToken())

	resp, err := c.HTTP.Do(req)
	if err != nil {
		_ = reader.Close()
		<-writeDone
		return BlobReference{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		writeErr := <-writeDone
		if writeErr != nil {
			return BlobReference{}, writeErr
		}
		return BlobReference{}, c.readErrorBody(resp, resp.StatusCode)
	}
	var result BlobReference
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return BlobReference{}, err
	}
	if err := <-writeDone; err != nil {
		return BlobReference{}, err
	}
	if result.Size != info.Size() || !validBlobSHA256(result.SHA256) {
		return BlobReference{}, fmt.Errorf("invalid blob upload response")
	}
	return result, nil
}

// DownloadBlob downloads a blob by SHA-256 hash.
func (c *Client) DownloadBlob(shaHex, destPath string) error {
	return c.DownloadBlobVerified(shaHex, -1, destPath)
}

// DownloadBlobVerified streams to a temporary file, verifies the announced
// hash and size, and only then atomically makes the file visible to the vault.
func (c *Client) DownloadBlobVerified(shaHex string, expectedSize int64, destPath string) error {
	if !validBlobSHA256(shaHex) || expectedSize < -1 {
		return fmt.Errorf("invalid blob reference")
	}
	req, err := http.NewRequest("GET", c.ServerURL+"/api/v1/blobs/"+shaHex, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.bearerToken())

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.readErrorBody(resp, resp.StatusCode)
	}

	if expectedSize >= 0 && resp.ContentLength >= 0 && resp.ContentLength != expectedSize {
		return fmt.Errorf("download blob: size mismatch")
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0o750); err != nil {
		return err
	}
	out, err := os.CreateTemp(filepath.Dir(destPath), ".verstak-blob-*")
	if err != nil {
		return err
	}
	tmpPath := out.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()
	hash := sha256.New()
	limit := int64(1<<63 - 1)
	if expectedSize >= 0 {
		limit = expectedSize + 1
	}
	written, err := io.Copy(io.MultiWriter(out, hash), io.LimitReader(resp.Body, limit))
	if err != nil {
		_ = out.Close()
		return err
	}
	if expectedSize >= 0 && written != expectedSize {
		_ = out.Close()
		return fmt.Errorf("download blob: size mismatch")
	}
	if actual := hex.EncodeToString(hash.Sum(nil)); actual != shaHex {
		_ = out.Close()
		return fmt.Errorf("download blob: SHA-256 mismatch")
	}
	if err := out.Sync(); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, destPath); err != nil {
		return err
	}
	cleanup = false
	return nil
}

func validBlobSHA256(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (c *Client) bearerToken() string {
	if c.DeviceToken != "" {
		return c.DeviceToken
	}
	return c.APIKey
}

func (c *Client) post(path string, body, result interface{}) error {
	var b bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&b).Encode(body); err != nil {
			return err
		}
	}
	payload := b.Bytes()

	var lastErr error
	for attempt := 1; attempt <= syncHTTPAttempts; attempt++ {
		req, err := http.NewRequest("POST", c.ServerURL+path, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.bearerToken())

		resp, err := c.HTTP.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("http: %w", err)
			if attempt < syncHTTPAttempts {
				syncRetrySleep(syncBackoffDelay(attempt))
				continue
			}
			return lastErr
		}
		if resp.StatusCode >= 400 {
			if isTransientHTTPStatus(resp.StatusCode) && attempt < syncHTTPAttempts {
				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
				syncRetrySleep(syncBackoffDelay(attempt))
				continue
			}
			err := c.readErrorBody(resp, resp.StatusCode)
			_ = resp.Body.Close()
			return err
		}

		if result != nil {
			err := json.NewDecoder(resp.Body).Decode(result)
			_ = resp.Body.Close()
			return err
		}
		_ = resp.Body.Close()
		return nil
	}
	return lastErr
}

func (c *Client) get(path string, result interface{}) error {
	var lastErr error
	for attempt := 1; attempt <= syncHTTPAttempts; attempt++ {
		req, err := http.NewRequest("GET", c.ServerURL+path, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.bearerToken())

		resp, err := c.HTTP.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("http: %w", err)
			if attempt < syncHTTPAttempts {
				syncRetrySleep(syncBackoffDelay(attempt))
				continue
			}
			return lastErr
		}
		if resp.StatusCode >= 400 {
			if isTransientHTTPStatus(resp.StatusCode) && attempt < syncHTTPAttempts {
				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
				syncRetrySleep(syncBackoffDelay(attempt))
				continue
			}
			err := c.readErrorBody(resp, resp.StatusCode)
			_ = resp.Body.Close()
			return err
		}

		if result != nil {
			err := json.NewDecoder(resp.Body).Decode(result)
			_ = resp.Body.Close()
			return err
		}
		_ = resp.Body.Close()
		return nil
	}
	return lastErr
}

func syncBackoffDelay(attempt int) time.Duration {
	return time.Duration(attempt) * 250 * time.Millisecond
}

func isTransientHTTPStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusRequestTimeout, http.StatusTooManyRequests, http.StatusInternalServerError,
		http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func (c *Client) readErrorBody(resp *http.Response, statusCode int) error {
	buf := make([]byte, 4096)
	n, _ := io.ReadFull(resp.Body, buf)
	body := string(buf[:minInt(n, 500)])

	lower := strings.ToLower(body)
	if strings.Contains(lower, "<html") || strings.Contains(lower, "<!doctype") {
		return fmt.Errorf("not a Verstak Sync server (HTTP %d)", statusCode)
	}
	var payload struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal([]byte(body), &payload); err == nil && payload.Code != "" {
		return &ServerError{Status: statusCode, Code: payload.Code}
	}
	return &ServerError{Status: statusCode, Code: "request_failed"}
}
