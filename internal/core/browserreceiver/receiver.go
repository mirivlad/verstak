// Package browserreceiver hosts the local HTTP protocol used by the browser extension.
package browserreceiver

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/verstak/verstak-desktop/internal/core/events"
)

const capturePath = "/api/browser-inbox/v1/captures"
const DefaultAddr = "127.0.0.1:47731"
const receiverTokenHeader = "X-Verstak-Receiver-Token"

type Receiver struct {
	bus               *events.Bus
	workspaceProvider WorkspaceProvider
	options           Options
}

type WorkspaceProvider func() string

type Options struct {
	RequireToken  bool
	ReceiverToken string
}

type Server struct {
	listener net.Listener
	server   *http.Server
}

type CapturePayload struct {
	SchemaVersion int               `json:"schemaVersion"`
	CaptureID     string            `json:"captureId"`
	CapturedAt    string            `json:"capturedAt"`
	Source        string            `json:"source"`
	Kind          string            `json:"kind"`
	Page          CapturePage       `json:"page"`
	Selection     *CaptureSelection `json:"selection,omitempty"`
	Link          *CaptureLink      `json:"link,omitempty"`
	Browser       *CaptureBrowser   `json:"browser,omitempty"`
	Context       interface{}       `json:"context,omitempty"`
}

type CapturePage struct {
	URL    string `json:"url"`
	Title  string `json:"title"`
	Domain string `json:"domain"`
}

type CaptureSelection struct {
	Text string `json:"text"`
}

type CaptureLink struct {
	URL  string `json:"url"`
	Text string `json:"text"`
}

type CaptureBrowser struct {
	Name string `json:"name"`
}

func New(bus *events.Bus, providers ...WorkspaceProvider) *Receiver {
	return NewWithOptions(bus, Options{}, providers...)
}

func NewWithOptions(bus *events.Bus, options Options, providers ...WorkspaceProvider) *Receiver {
	var provider WorkspaceProvider
	if len(providers) > 0 {
		provider = providers[0]
	}
	return &Receiver{bus: bus, workspaceProvider: provider, options: options}
}

func Start(addr string, receiver *Receiver) (*Server, error) {
	if receiver == nil {
		return nil, fmt.Errorf("receiver is required")
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	s := &Server{
		listener: listener,
		server: &http.Server{
			Handler: receiver,
		},
	}
	go func() {
		if err := s.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("[browserreceiver] serve: %v", err)
		}
	}()
	return s, nil
}

func (s *Server) URL() string {
	if s == nil || s.listener == nil {
		return ""
	}
	return "http://" + s.listener.Addr().String()
}

func (s *Server) Close() error {
	if s == nil || s.server == nil {
		return nil
	}
	return s.server.Shutdown(context.Background())
}

func (r *Receiver) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if req.URL.Path != capturePath {
		http.NotFound(w, req)
		return
	}
	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
		return
	}
	if err := r.validateReceiverToken(req); err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	var payload CapturePayload
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := payload.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	eventName := "browser.capture." + payload.Kind
	if r.bus == nil || !r.bus.HasSubscribers(eventName) {
		writeError(w, http.StatusServiceUnavailable, "browser inbox unavailable")
		return
	}
	eventPayload := payload.EventPayload()
	r.annotateWorkspace(eventPayload)
	r.bus.Publish(events.Event{
		Name:      eventName,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Payload:   eventPayload,
	})

	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":    "accepted",
		"captureId": payload.CaptureID,
	})
}

func (r *Receiver) validateReceiverToken(req *http.Request) error {
	if r == nil || !r.options.RequireToken {
		return nil
	}
	expected := strings.TrimSpace(r.options.ReceiverToken)
	if expected == "" {
		return fmt.Errorf("receiver token required")
	}
	supplied := strings.TrimSpace(req.Header.Get(receiverTokenHeader))
	if supplied == "" {
		return fmt.Errorf("receiver token required")
	}
	if subtle.ConstantTimeCompare([]byte(supplied), []byte(expected)) != 1 {
		return fmt.Errorf("receiver token invalid")
	}
	return nil
}

func (r *Receiver) annotateWorkspace(payload map[string]interface{}) {
	if r == nil || r.workspaceProvider == nil || payload == nil {
		return
	}
	if _, ok := payload["workspaceRootPath"]; ok {
		return
	}
	workspaceRoot := strings.TrimSpace(r.workspaceProvider())
	if workspaceRoot == "" {
		return
	}
	payload["workspaceRootPath"] = workspaceRoot
	payload["workspaceName"] = workspaceRoot
}

func (p CapturePayload) Validate() error {
	if p.SchemaVersion != 1 {
		return fmt.Errorf("unsupported schemaVersion")
	}
	if strings.TrimSpace(p.CaptureID) == "" {
		return fmt.Errorf("captureId is required")
	}
	if strings.TrimSpace(p.CapturedAt) == "" {
		return fmt.Errorf("capturedAt is required")
	}
	if p.Kind != "page" && p.Kind != "selection" && p.Kind != "link" {
		return fmt.Errorf("unsupported kind")
	}
	if strings.TrimSpace(p.Page.URL) == "" {
		return fmt.Errorf("page.url is required")
	}
	if p.Kind == "selection" && (p.Selection == nil || strings.TrimSpace(p.Selection.Text) == "") {
		return fmt.Errorf("selection.text is required")
	}
	if p.Kind == "link" && (p.Link == nil || strings.TrimSpace(p.Link.URL) == "") {
		return fmt.Errorf("link.url is required")
	}
	return nil
}

func (p CapturePayload) EventPayload() map[string]interface{} {
	pageURL := strings.TrimSpace(p.Page.URL)
	result := map[string]interface{}{
		"captureId":  strings.TrimSpace(p.CaptureID),
		"capturedAt": strings.TrimSpace(p.CapturedAt),
		"source":     strings.TrimSpace(p.Source),
		"kind":       p.Kind,
		"url":        pageURL,
		"title":      strings.TrimSpace(p.Page.Title),
		"domain":     captureDomain(pageURL, p.Page.Domain),
	}
	if p.Browser != nil {
		result["browserName"] = strings.TrimSpace(p.Browser.Name)
	}
	if p.Context != nil {
		result["context"] = p.Context
	}

	switch p.Kind {
	case "selection":
		result["text"] = strings.TrimSpace(p.Selection.Text)
	case "link":
		linkURL := strings.TrimSpace(p.Link.URL)
		result["url"] = linkURL
		result["title"] = strings.TrimSpace(p.Link.Text)
		result["domain"] = captureDomain(linkURL, "")
	}
	return result
}

func captureDomain(rawURL, fallback string) string {
	if u, err := url.Parse(strings.TrimSpace(rawURL)); err == nil && u.Hostname() != "" {
		return u.Hostname()
	}
	return strings.TrimSpace(fallback)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
