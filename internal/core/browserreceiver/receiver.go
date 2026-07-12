// Package browserreceiver hosts the local HTTP protocol used by the browser extension.
package browserreceiver

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/verstak/verstak-desktop/internal/core/events"
	"github.com/verstak/verstak-desktop/internal/core/hostname"
)

const (
	capturePath         = "/api/browser-inbox/v1/captures"
	activityBatchPath   = "/api/browser-activity/v1/batches"
	DefaultAddr         = "127.0.0.1:47731"
	DefaultCaptureURL   = "http://" + DefaultAddr + capturePath
	DefaultActivityURL  = "http://" + DefaultAddr + activityBatchPath
	receiverTokenHeader = "X-Verstak-Receiver-Token"
)

const (
	maxCaptureBodyBytes    = 12 * 1024 * 1024
	maxCaptureIDBytes      = 256
	maxCapturedAtBytes     = 64
	maxCaptureSourceBytes  = 128
	maxPageURLBytes        = 4096
	maxPageTitleBytes      = 512
	maxPageDomainBytes     = 255
	maxSelectionTextBytes  = 20 * 1024
	maxLinkTextBytes       = 512
	maxBrowserNameBytes    = 64
	maxFileNameBytes       = 255
	maxFileMimeBytes       = 128
	maxFileTextBytes       = 2 * 1024 * 1024
	maxFileBytes           = 8 * 1024 * 1024
	maxFileDataBase64Bytes = 4 * ((maxFileBytes + 2) / 3)
	maxActivityBodyBytes   = 256 * 1024
	maxActivityEntries     = 100
	maxActivityDuration    = 10 * time.Minute
)

type Receiver struct {
	bus               *events.Bus
	workspaceProvider WorkspaceProvider
	optionsMu         sync.RWMutex
	options           Options
}

type WorkspaceProvider func() string

type Options struct {
	RequireToken      bool
	ReceiverToken     string
	Available         func() bool
	Persist           func(events.Event) error
	ActivityAvailable func() bool
	PersistActivity   func(events.Event) error
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
	File          *CaptureFile      `json:"file,omitempty"`
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

type CaptureFile struct {
	Name       string `json:"name"`
	Mime       string `json:"mime"`
	Size       int64  `json:"size"`
	Text       string `json:"text"`
	DataBase64 string `json:"dataBase64"`
}

type CaptureBrowser struct {
	Name string `json:"name"`
}

// ActivityBatchPayload contains only domain-level time accounting. It never
// accepts or emits page URLs, titles, content, or navigation history.
type ActivityBatchPayload struct {
	SchemaVersion int             `json:"schemaVersion"`
	BatchID       string          `json:"batchId"`
	CreatedAt     string          `json:"createdAt"`
	Source        string          `json:"source"`
	Entries       []ActivityEntry `json:"entries"`
}

type ActivityEntry struct {
	Hostname        string `json:"hostname"`
	StartedAt       string `json:"startedAt"`
	EndedAt         string `json:"endedAt"`
	DurationSeconds int64  `json:"durationSeconds"`
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

// SetReceiverToken updates the active token without restarting the local server.
func (r *Receiver) SetReceiverToken(token string) {
	if r == nil {
		return
	}
	r.optionsMu.Lock()
	defer r.optionsMu.Unlock()
	r.options.RequireToken = true
	r.options.ReceiverToken = strings.TrimSpace(token)
}

// SetPersistence configures durable capture storage and its availability gate.
func (r *Receiver) SetPersistence(available func() bool, persist func(events.Event) error) {
	if r == nil {
		return
	}
	r.optionsMu.Lock()
	defer r.optionsMu.Unlock()
	r.options.Available = available
	r.options.Persist = persist
}

// SetActivityPersistence configures durable passive browser activity storage.
// An acknowledgement is sent only after persist returns successfully.
func (r *Receiver) SetActivityPersistence(available func() bool, persist func(events.Event) error) {
	if r == nil {
		return
	}
	r.optionsMu.Lock()
	defer r.optionsMu.Unlock()
	r.options.ActivityAvailable = available
	r.options.PersistActivity = persist
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

	if req.URL.Path == activityBatchPath {
		r.serveActivityBatch(w, req)
		return
	}
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
	options := r.currentOptions()
	if options.Available != nil && !options.Available() {
		writeError(w, http.StatusServiceUnavailable, "browser inbox unavailable")
		return
	}

	defer req.Body.Close()
	decoder := json.NewDecoder(http.MaxBytesReader(w, req.Body, maxCaptureBodyBytes))
	var payload CapturePayload
	if err := decoder.Decode(&payload); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "capture payload exceeds limit")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "capture payload exceeds limit")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := payload.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	eventName := "browser.capture." + payload.Kind
	if options.Persist == nil && (r.bus == nil || !r.bus.HasSubscribers(eventName)) {
		writeError(w, http.StatusServiceUnavailable, "browser inbox unavailable")
		return
	}
	eventPayload := payload.EventPayload()
	r.annotateWorkspace(eventPayload)
	event := events.Event{
		Name:      eventName,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Payload:   eventPayload,
	}
	if options.Persist != nil {
		if err := options.Persist(event); err != nil {
			log.Printf("[browserreceiver] persist %s: %v", payload.CaptureID, err)
			writeError(w, http.StatusServiceUnavailable, "browser inbox unavailable")
			return
		}
	}
	if r.bus != nil {
		r.bus.Publish(event)
	}

	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":    "accepted",
		"captureId": payload.CaptureID,
	})
}

func (r *Receiver) serveActivityBatch(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
		return
	}
	if err := r.validateReceiverToken(req); err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	options := r.currentOptions()
	if options.ActivityAvailable == nil || !options.ActivityAvailable() || options.PersistActivity == nil {
		writeError(w, http.StatusServiceUnavailable, "activity storage unavailable")
		return
	}

	defer req.Body.Close()
	decoder := json.NewDecoder(http.MaxBytesReader(w, req.Body, maxActivityBodyBytes))
	var batch ActivityBatchPayload
	if err := decoder.Decode(&batch); err != nil {
		writeActivityDecodeError(w, err)
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeActivityDecodeError(w, err)
		return
	}
	if err := batch.NormalizeAndValidate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	event := events.Event{
		Name:      "browser.activity.batch",
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Payload:   batch.EventPayload(),
	}
	if err := options.PersistActivity(event); err != nil {
		log.Printf("[browserreceiver] persist activity %s: %v", batch.BatchID, err)
		writeError(w, http.StatusServiceUnavailable, "activity storage unavailable")
		return
	}
	if r.bus != nil {
		r.bus.Publish(event)
	}

	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "accepted",
		"batchId": batch.BatchID,
	})
}

func (r *Receiver) currentOptions() Options {
	if r == nil {
		return Options{}
	}
	r.optionsMu.RLock()
	defer r.optionsMu.RUnlock()
	return r.options
}

func (r *Receiver) validateReceiverToken(req *http.Request) error {
	if r == nil {
		return nil
	}
	r.optionsMu.RLock()
	requireToken := r.options.RequireToken
	expected := strings.TrimSpace(r.options.ReceiverToken)
	r.optionsMu.RUnlock()
	if !requireToken {
		return nil
	}
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
	if err := validateCaptureText(p.CaptureID, "captureId", maxCaptureIDBytes); err != nil {
		return err
	}
	if strings.TrimSpace(p.CapturedAt) == "" {
		return fmt.Errorf("capturedAt is required")
	}
	if err := validateCaptureText(p.CapturedAt, "capturedAt", maxCapturedAtBytes); err != nil {
		return err
	}
	if p.Kind != "page" && p.Kind != "selection" && p.Kind != "link" && p.Kind != "file" {
		return fmt.Errorf("unsupported kind")
	}
	if strings.TrimSpace(p.Page.URL) == "" {
		return fmt.Errorf("page.url is required")
	}
	if err := validateCaptureText(p.Source, "source", maxCaptureSourceBytes); err != nil {
		return err
	}
	if err := validateCaptureText(p.Page.URL, "page.url", maxPageURLBytes); err != nil {
		return err
	}
	if err := validateCaptureText(p.Page.Title, "page.title", maxPageTitleBytes); err != nil {
		return err
	}
	if err := validateCaptureText(p.Page.Domain, "page.domain", maxPageDomainBytes); err != nil {
		return err
	}
	if p.Browser != nil {
		if err := validateCaptureText(p.Browser.Name, "browser.name", maxBrowserNameBytes); err != nil {
			return err
		}
	}
	if p.Kind == "selection" && (p.Selection == nil || strings.TrimSpace(p.Selection.Text) == "") {
		return fmt.Errorf("selection.text is required")
	}
	if p.Kind == "selection" {
		if err := validateCaptureText(p.Selection.Text, "selection.text", maxSelectionTextBytes); err != nil {
			return err
		}
	}
	if p.Kind == "link" {
		if p.Link == nil || strings.TrimSpace(p.Link.URL) == "" {
			return fmt.Errorf("link.url is required")
		}
		if err := validateCaptureText(p.Link.URL, "link.url", maxPageURLBytes); err != nil {
			return err
		}
		if err := validateCaptureText(p.Link.Text, "link.text", maxLinkTextBytes); err != nil {
			return err
		}
	}
	if p.Kind == "file" {
		if err := p.validateFile(); err != nil {
			return err
		}
	}
	return nil
}

// NormalizeAndValidate validates a batch before it is made durable and replaces
// each submitted hostname with the shared canonical A-label form.
func (p *ActivityBatchPayload) NormalizeAndValidate() error {
	if p == nil || p.SchemaVersion != 1 {
		return fmt.Errorf("unsupported schemaVersion")
	}
	if strings.TrimSpace(p.BatchID) == "" {
		return fmt.Errorf("batchId is required")
	}
	if err := validateCaptureText(p.BatchID, "batchId", maxCaptureIDBytes); err != nil {
		return err
	}
	if strings.TrimSpace(p.CreatedAt) == "" {
		return fmt.Errorf("createdAt is required")
	}
	if _, err := time.Parse(time.RFC3339, p.CreatedAt); err != nil {
		return fmt.Errorf("createdAt is invalid")
	}
	if err := validateCaptureText(p.CreatedAt, "createdAt", maxCapturedAtBytes); err != nil {
		return err
	}
	if strings.TrimSpace(p.Source) == "" {
		return fmt.Errorf("source is required")
	}
	if err := validateCaptureText(p.Source, "source", maxCaptureSourceBytes); err != nil {
		return err
	}
	if len(p.Entries) == 0 || len(p.Entries) > maxActivityEntries {
		return fmt.Errorf("entries must contain between 1 and %d items", maxActivityEntries)
	}
	for index := range p.Entries {
		entry := &p.Entries[index]
		canonical := hostname.NormalizeHostnameV1(entry.Hostname)
		if canonical == "" {
			return fmt.Errorf("entries[%d].hostname is invalid", index)
		}
		startedAt, err := time.Parse(time.RFC3339, entry.StartedAt)
		if err != nil {
			return fmt.Errorf("entries[%d].startedAt is invalid", index)
		}
		endedAt, err := time.Parse(time.RFC3339, entry.EndedAt)
		if err != nil {
			return fmt.Errorf("entries[%d].endedAt is invalid", index)
		}
		interval := endedAt.Sub(startedAt)
		if interval <= 0 || interval > maxActivityDuration {
			return fmt.Errorf("entries[%d] interval must be between 1 second and %s", index, maxActivityDuration)
		}
		if entry.DurationSeconds <= 0 || entry.DurationSeconds > int64(maxActivityDuration/time.Second) || time.Duration(entry.DurationSeconds)*time.Second > interval {
			return fmt.Errorf("entries[%d].durationSeconds is invalid", index)
		}
		entry.Hostname = canonical
	}
	return nil
}

func (p ActivityBatchPayload) EventPayload() map[string]interface{} {
	entries := make([]map[string]interface{}, 0, len(p.Entries))
	for _, entry := range p.Entries {
		entries = append(entries, map[string]interface{}{
			"hostname":        entry.Hostname,
			"startedAt":       entry.StartedAt,
			"endedAt":         entry.EndedAt,
			"durationSeconds": entry.DurationSeconds,
		})
	}
	return map[string]interface{}{
		"batchId":   strings.TrimSpace(p.BatchID),
		"createdAt": strings.TrimSpace(p.CreatedAt),
		"source":    strings.TrimSpace(p.Source),
		"entries":   entries,
	}
}

func (p CapturePayload) validateFile() error {
	if p.File == nil || strings.TrimSpace(p.File.Name) == "" {
		return fmt.Errorf("file.name is required")
	}
	if p.File.Text == "" && strings.TrimSpace(p.File.DataBase64) == "" {
		return fmt.Errorf("file.text or file.dataBase64 is required")
	}
	if p.File.Size < 0 || p.File.Size > maxFileBytes {
		return fmt.Errorf("file.size exceeds limit")
	}
	if err := validateCaptureText(p.File.Name, "file.name", maxFileNameBytes); err != nil {
		return err
	}
	if err := validateCaptureText(p.File.Mime, "file.mime", maxFileMimeBytes); err != nil {
		return err
	}
	if err := validateCaptureText(p.File.Text, "file.text", maxFileTextBytes); err != nil {
		return err
	}
	dataBase64 := strings.TrimSpace(p.File.DataBase64)
	if dataBase64 == "" {
		return nil
	}
	if len(dataBase64) > maxFileDataBase64Bytes {
		return fmt.Errorf("file.dataBase64 exceeds limit")
	}
	data, err := base64.StdEncoding.DecodeString(dataBase64)
	if err != nil {
		return fmt.Errorf("file.dataBase64 is invalid")
	}
	if len(data) > maxFileBytes {
		return fmt.Errorf("file.dataBase64 exceeds limit")
	}
	return nil
}

func validateCaptureText(value, field string, maxBytes int) error {
	if len(value) > maxBytes {
		return fmt.Errorf("%s exceeds limit", field)
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
	case "file":
		result["fileName"] = strings.TrimSpace(p.File.Name)
		result["fileMime"] = strings.TrimSpace(p.File.Mime)
		result["fileSize"] = p.File.Size
		result["fileText"] = p.File.Text
		result["fileDataBase64"] = strings.TrimSpace(p.File.DataBase64)
	}
	return result
}

func captureDomain(rawURL, fallback string) string {
	if normalized := hostname.NormalizeURLHostnameV1(rawURL); normalized != "" {
		return normalized
	}
	return hostname.NormalizeHostnameV1(fallback)
}

func writeActivityDecodeError(w http.ResponseWriter, err error) {
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		writeError(w, http.StatusRequestEntityTooLarge, "activity payload exceeds limit")
		return
	}
	writeError(w, http.StatusBadRequest, "invalid JSON")
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
