package browserreceiver

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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
