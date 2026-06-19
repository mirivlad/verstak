package sync

import (
	"bytes"
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
func (c *Client) PairDevice(serverURL, username, password, deviceName, clientVersion string) (deviceID, deviceToken string, err error) {
	body := map[string]string{
		"login":          username,
		"password":       password,
		"device_name":    deviceName,
		"client_version": clientVersion,
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
}

// PullResponse is the response from POST /sync/pull.
type PullResponse struct {
	ServerSequence int  `json:"server_sequence"`
	Ops            []Op `json:"ops"`
}

// Pull fetches remote operations since a given sequence.
func (c *Client) Pull(sinceSequence int) (*PullResponse, error) {
	req := PullRequest{SinceSequence: sinceSequence}
	var resp PullResponse
	if err := c.post("/api/v1/sync/pull", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UploadBlob uploads a file to the server and returns its SHA-256.
func (c *Client) UploadBlob(localPath string) (sha256 string, err error) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fw, err := w.CreateFormFile("file", filepath.Base(localPath))
	if err != nil {
		return "", err
	}
	f, err := os.Open(localPath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(fw, f); err != nil {
		return "", err
	}
	w.Close()

	req, err := http.NewRequest("POST", c.ServerURL+"/api/v1/blobs/", &b)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+c.bearerToken())

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		SHA256 string `json:"sha256"`
		Size   int    `json:"size"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.SHA256, nil
}

// DownloadBlob downloads a blob by SHA-256 hash.
func (c *Client) DownloadBlob(sha256, destPath string) error {
	req, err := http.NewRequest("GET", c.ServerURL+"/api/v1/blobs/"+sha256, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.bearerToken())

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download blob: HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
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
	req, err := http.NewRequest("POST", c.ServerURL+path, &b)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.bearerToken())

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return c.readErrorBody(resp, resp.StatusCode)
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

func (c *Client) get(path string, result interface{}) error {
	req, err := http.NewRequest("GET", c.ServerURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.bearerToken())

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return c.readErrorBody(resp, resp.StatusCode)
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

func (c *Client) readErrorBody(resp *http.Response, statusCode int) error {
	buf := make([]byte, 4096)
	n, _ := io.ReadFull(resp.Body, buf)
	body := string(buf[:minInt(n, 500)])

	lower := strings.ToLower(body)
	if strings.Contains(lower, "<html") || strings.Contains(lower, "<!doctype") {
		return fmt.Errorf("not a Verstak Sync server (HTTP %d)", statusCode)
	}
	return fmt.Errorf("server error (HTTP %d)", statusCode)
}
