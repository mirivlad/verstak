package browserreceiver

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/verstak/verstak-desktop/internal/core/events"
)

func TestReceiverAcceptsSelectionCaptureAndPublishesEvent(t *testing.T) {
	bus := events.NewBus()
	received := make(chan events.Event, 1)
	bus.Subscribe("browser.capture.selection", func(event events.Event) {
		received <- event
	})

	receiver := New(bus)
	body := `{
		"schemaVersion": 1,
		"captureId": "capture-123",
		"capturedAt": "2026-06-27T00:00:00.000Z",
		"source": "verstak-browser-extension",
		"kind": "selection",
		"page": {
			"url": "https://example.com/article",
			"title": "Example Article",
			"domain": "example.com"
		},
		"selection": {
			"text": "selected text"
		},
		"browser": {
			"name": "Chromium"
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/browser-inbox/v1/captures", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	receiver.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusAccepted, rec.Body.String())
	}
	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("response json: %v", err)
	}
	if response["status"] != "accepted" {
		t.Fatalf("response status = %q, want accepted", response["status"])
	}
	if response["captureId"] != "capture-123" {
		t.Fatalf("response captureId = %q, want capture-123", response["captureId"])
	}

	event := <-received
	if event.Name != "browser.capture.selection" {
		t.Fatalf("event name = %q, want browser.capture.selection", event.Name)
	}
	payload, ok := event.Payload.(map[string]interface{})
	if !ok {
		t.Fatalf("event payload type = %T, want map[string]interface{}", event.Payload)
	}
	if payload["captureId"] != "capture-123" {
		t.Fatalf("payload captureId = %v, want capture-123", payload["captureId"])
	}
	if payload["url"] != "https://example.com/article" {
		t.Fatalf("payload url = %v, want https://example.com/article", payload["url"])
	}
	if payload["title"] != "Example Article" {
		t.Fatalf("payload title = %v, want Example Article", payload["title"])
	}
	if payload["text"] != "selected text" {
		t.Fatalf("payload text = %v, want selected text", payload["text"])
	}
	if payload["capturedAt"] != "2026-06-27T00:00:00.000Z" {
		t.Fatalf("payload capturedAt = %v, want documented timestamp", payload["capturedAt"])
	}
	if payload["domain"] != "example.com" {
		t.Fatalf("payload domain = %v, want example.com", payload["domain"])
	}
}

func TestReceiverAcceptsFileCaptureAndPublishesEvent(t *testing.T) {
	bus := events.NewBus()
	received := make(chan events.Event, 1)
	bus.Subscribe("browser.capture.file", func(event events.Event) {
		received <- event
	})

	receiver := New(bus)
	body := `{
		"schemaVersion": 1,
		"captureId": "capture-file",
		"capturedAt": "2026-06-29T02:00:00.000Z",
		"source": "verstak-browser-extension",
		"kind": "file",
		"page": {
			"url": "https://example.com/files",
			"title": "Example Files",
			"domain": "example.com"
		},
		"file": {
			"name": "notes.txt",
			"mime": "text/plain",
			"size": 11,
			"text": "hello file",
			"dataBase64": "aGVsbG8gZmlsZQ=="
		},
		"browser": {
			"name": "Firefox"
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/browser-inbox/v1/captures", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	receiver.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusAccepted, rec.Body.String())
	}
	event := <-received
	if event.Name != "browser.capture.file" {
		t.Fatalf("event name = %q, want browser.capture.file", event.Name)
	}
	payload, ok := event.Payload.(map[string]interface{})
	if !ok {
		t.Fatalf("event payload type = %T, want map[string]interface{}", event.Payload)
	}
	if payload["kind"] != "file" {
		t.Fatalf("payload kind = %v, want file", payload["kind"])
	}
	if payload["fileName"] != "notes.txt" {
		t.Fatalf("payload fileName = %v, want notes.txt", payload["fileName"])
	}
	if payload["fileMime"] != "text/plain" {
		t.Fatalf("payload fileMime = %v, want text/plain", payload["fileMime"])
	}
	if payload["fileSize"] != int64(11) {
		t.Fatalf("payload fileSize = %v, want 11", payload["fileSize"])
	}
	if payload["fileText"] != "hello file" {
		t.Fatalf("payload fileText = %v, want hello file", payload["fileText"])
	}
	if payload["fileDataBase64"] != "aGVsbG8gZmlsZQ==" {
		t.Fatalf("payload fileDataBase64 = %v, want aGVsbG8gZmlsZQ==", payload["fileDataBase64"])
	}
}

func TestReceiverAnnotatesCaptureWithCurrentWorkspace(t *testing.T) {
	bus := events.NewBus()
	received := make(chan events.Event, 1)
	bus.Subscribe("browser.capture.page", func(event events.Event) {
		received <- event
	})

	receiver := New(bus, func() string { return "Project" })
	body := `{
		"schemaVersion": 1,
		"captureId": "capture-workspace",
		"capturedAt": "2026-06-27T00:00:00.000Z",
		"source": "verstak-browser-extension",
		"kind": "page",
		"page": {
			"url": "https://example.com/article",
			"title": "Example Article"
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/browser-inbox/v1/captures", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	receiver.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusAccepted, rec.Body.String())
	}

	event := <-received
	payload, ok := event.Payload.(map[string]interface{})
	if !ok {
		t.Fatalf("event payload type = %T, want map[string]interface{}", event.Payload)
	}
	if payload["workspaceRootPath"] != "Project" {
		t.Fatalf("payload workspaceRootPath = %v, want Project", payload["workspaceRootPath"])
	}
	if payload["workspaceName"] != "Project" {
		t.Fatalf("payload workspaceName = %v, want Project", payload["workspaceName"])
	}
}

func TestReceiverRequiresTokenWhenPaired(t *testing.T) {
	bus := events.NewBus()
	received := make(chan events.Event, 1)
	bus.Subscribe("browser.capture.page", func(event events.Event) {
		received <- event
	})
	receiver := NewWithOptions(bus, Options{RequireToken: true, ReceiverToken: "pair-token"})
	body := `{
		"schemaVersion": 1,
		"captureId": "capture-paired",
		"capturedAt": "2026-06-27T00:00:00.000Z",
		"source": "verstak-browser-extension",
		"kind": "page",
		"page": {
			"url": "https://example.com/article",
			"title": "Example Article"
		}
	}`

	for _, tc := range []struct {
		name      string
		token     string
		wantError string
	}{
		{name: "missing", token: "", wantError: "receiver token required"},
		{name: "wrong", token: "wrong-token", wantError: "receiver token invalid"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/browser-inbox/v1/captures", bytes.NewBufferString(body))
			if tc.token != "" {
				req.Header.Set("X-Verstak-Receiver-Token", tc.token)
			}
			rec := httptest.NewRecorder()

			receiver.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusUnauthorized, rec.Body.String())
			}
			if !bytes.Contains(rec.Body.Bytes(), []byte(tc.wantError)) {
				t.Fatalf("response body = %q, want %q", rec.Body.String(), tc.wantError)
			}
			select {
			case event := <-received:
				t.Fatalf("unexpected event published for rejected capture: %#v", event)
			default:
			}
		})
	}
}

func TestReceiverAcceptsPairedToken(t *testing.T) {
	bus := events.NewBus()
	received := make(chan events.Event, 1)
	bus.Subscribe("browser.capture.page", func(event events.Event) {
		received <- event
	})
	receiver := NewWithOptions(bus, Options{RequireToken: true, ReceiverToken: "pair-token"})
	body := `{
		"schemaVersion": 1,
		"captureId": "capture-paired",
		"capturedAt": "2026-06-27T00:00:00.000Z",
		"source": "verstak-browser-extension",
		"kind": "page",
		"page": {
			"url": "https://example.com/article",
			"title": "Example Article"
		}
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/browser-inbox/v1/captures", bytes.NewBufferString(body))
	req.Header.Set("X-Verstak-Receiver-Token", "pair-token")
	rec := httptest.NewRecorder()

	receiver.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusAccepted, rec.Body.String())
	}
	event := <-received
	if event.Name != "browser.capture.page" {
		t.Fatalf("event name = %q, want browser.capture.page", event.Name)
	}
}

func TestReceiverRotatesPairedToken(t *testing.T) {
	bus := events.NewBus()
	bus.Subscribe("browser.capture.page", func(event events.Event) {})
	receiver := NewWithOptions(bus, Options{RequireToken: true, ReceiverToken: "old-token"})
	body := `{
		"schemaVersion": 1,
		"captureId": "capture-rotated-token",
		"capturedAt": "2026-06-27T00:00:00.000Z",
		"kind": "page",
		"page": {"url": "https://example.com"}
	}`

	request := func(token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, capturePath, bytes.NewBufferString(body))
		req.Header.Set(receiverTokenHeader, token)
		res := httptest.NewRecorder()
		receiver.ServeHTTP(res, req)
		return res
	}

	if res := request("old-token"); res.Code != http.StatusAccepted {
		t.Fatalf("old token before rotation status = %d, want %d", res.Code, http.StatusAccepted)
	}
	receiver.SetReceiverToken("new-token")
	if res := request("old-token"); res.Code != http.StatusUnauthorized {
		t.Fatalf("old token after rotation status = %d, want %d", res.Code, http.StatusUnauthorized)
	}
	if res := request("new-token"); res.Code != http.StatusAccepted {
		t.Fatalf("new token after rotation status = %d, want %d", res.Code, http.StatusAccepted)
	}
}

func TestServerStartsOnLocalAddressAndAcceptsCapture(t *testing.T) {
	bus := events.NewBus()
	bus.Subscribe("browser.capture.page", func(event events.Event) {})
	receiver := New(bus)
	server, err := Start("127.0.0.1:0", receiver)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer server.Close()

	response, err := http.Post(server.URL()+capturePath, "application/json", bytes.NewBufferString(`{
		"schemaVersion": 1,
		"captureId": "capture-server",
		"capturedAt": "2026-06-27T00:00:00.000Z",
		"source": "verstak-browser-extension",
		"kind": "page",
		"page": {
			"url": "https://example.com/article",
			"title": "Example Article"
		}
	}`))
	if err != nil {
		t.Fatalf("post capture: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("status = %d, want %d; body=%s", response.StatusCode, http.StatusAccepted, string(body))
	}
}

func TestReceiverRejectsCaptureWhenNoConsumerIsRegistered(t *testing.T) {
	receiver := New(events.NewBus())
	body := `{
		"schemaVersion": 1,
		"captureId": "capture-queued",
		"capturedAt": "2026-06-27T00:00:00.000Z",
		"source": "verstak-browser-extension",
		"kind": "page",
		"page": {
			"url": "https://example.com/article",
			"title": "Example Article"
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/browser-inbox/v1/captures", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	receiver.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusServiceUnavailable, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("browser inbox unavailable")) {
		t.Fatalf("response body = %q, want unavailable error", rec.Body.String())
	}
}

func TestReceiverRejectsInvalidCapturePayload(t *testing.T) {
	receiver := New(events.NewBus())
	body := `{
		"schemaVersion": 1,
		"captureId": "capture-123",
		"capturedAt": "2026-06-27T00:00:00.000Z",
		"source": "verstak-browser-extension",
		"kind": "link",
		"page": {
			"url": "https://example.com/article",
			"title": "Example Article"
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/browser-inbox/v1/captures", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	receiver.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("link.url is required")) {
		t.Fatalf("response body = %q, want validation error", rec.Body.String())
	}
}

func TestReceiverRejectsOversizedCaptureBody(t *testing.T) {
	bus := events.NewBus()
	received := make(chan events.Event, 1)
	bus.Subscribe("browser.capture.page", func(event events.Event) {
		received <- event
	})
	receiver := New(bus)

	const maxCaptureBodyBytes = 12 * 1024 * 1024
	body := fmt.Sprintf(`{
		"schemaVersion": 1,
		"captureId": "capture-oversized",
		"capturedAt": "2026-06-27T00:00:00.000Z",
		"kind": "page",
		"page": {"url": "https://example.com", "title": %q}
	}`, strings.Repeat("x", maxCaptureBodyBytes))
	req := httptest.NewRequest(http.MethodPost, "/api/browser-inbox/v1/captures", strings.NewReader(body))
	rec := httptest.NewRecorder()

	receiver.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusRequestEntityTooLarge, rec.Body.String())
	}
	select {
	case event := <-received:
		t.Fatalf("unexpected event published for oversized capture: %#v", event)
	default:
	}
}

func TestCapturePayloadRejectsUnsafeFileContent(t *testing.T) {
	const maxFileBytes = 8 * 1024 * 1024
	const maxFileTextBytes = 2 * 1024 * 1024

	newFilePayload := func() CapturePayload {
		return CapturePayload{
			SchemaVersion: 1,
			CaptureID:     "capture-file-validation",
			CapturedAt:    "2026-06-27T00:00:00.000Z",
			Kind:          "file",
			Page:          CapturePage{URL: "https://example.com"},
			File:          &CaptureFile{Name: "attachment.bin", Size: 1, DataBase64: "AQ=="},
		}
	}

	tests := []struct {
		name string
		edit func(*CapturePayload)
		want string
	}{
		{
			name: "oversized declared size",
			edit: func(payload *CapturePayload) {
				payload.File.Size = maxFileBytes + 1
			},
			want: "file.size exceeds limit",
		},
		{
			name: "invalid base64",
			edit: func(payload *CapturePayload) {
				payload.File.DataBase64 = "not base64"
			},
			want: "file.dataBase64 is invalid",
		},
		{
			name: "oversized decoded data",
			edit: func(payload *CapturePayload) {
				payload.File.DataBase64 = base64.StdEncoding.EncodeToString(make([]byte, maxFileBytes+1))
			},
			want: "file.dataBase64 exceeds limit",
		},
		{
			name: "oversized text",
			edit: func(payload *CapturePayload) {
				payload.File.DataBase64 = ""
				payload.File.Text = strings.Repeat("x", maxFileTextBytes+1)
			},
			want: "file.text exceeds limit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := newFilePayload()
			tt.edit(&payload)
			if err := payload.Validate(); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Validate() error = %v, want %q", err, tt.want)
			}
		})
	}
}
