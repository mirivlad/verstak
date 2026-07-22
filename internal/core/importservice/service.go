package importservice

import (
	"bytes"
	"context"
	"io"
	"strconv"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

type sourceSessionState struct {
	pluginID string
	handle   string
	source   indexedSource
	lastUsed time.Time
	cancel   context.CancelFunc
	phase    string
}

type Service struct {
	mu              sync.Mutex
	applyMu         sync.Mutex
	vaultDir        string
	limits          limits
	now             func() time.Time
	sessions        map[string]*sourceSessionState
	closedOwners    map[string]string
	onProgress      func(pluginID string, progress Progress)
	refresh         func() error
	promoteRegistry func(sourcePath, targetPath string) error
}

func New(vaultDir string, options Options) *Service {
	resolved, now := resolveOptions(options)
	return &Service{vaultDir: vaultDir, limits: resolved, now: now, sessions: make(map[string]*sourceSessionState), closedOwners: make(map[string]string), onProgress: options.OnProgress, refresh: options.Refresh, promoteRegistry: promoteRegistryFile}
}

func (service *Service) OpenDirectory(pluginID, selectedPath string) (SourceSession, error) {
	source, err := newDirectorySource(selectedPath, service.limits)
	if err != nil {
		return SourceSession{}, err
	}
	return service.add(pluginID, source), nil
}

func (service *Service) OpenArchive(pluginID, selectedPath string) (SourceSession, error) {
	source, err := newArchiveSource(selectedPath, service.limits)
	if err != nil {
		return SourceSession{}, err
	}
	return service.add(pluginID, source), nil
}

func (service *Service) add(pluginID string, source indexedSource) SourceSession {
	handle := uuid.NewString()
	service.mu.Lock()
	defer service.mu.Unlock()
	service.expireLocked()
	service.sessions[handle] = &sourceSessionState{pluginID: pluginID, handle: handle, source: source, lastUsed: service.now()}
	return SourceSession{SourceHandle: handle, Kind: source.kind(), DisplayPath: source.displayPath(), DisplayName: source.displayName(),
		Fingerprint: source.fingerprint(), EntryCount: len(source.entries()), TotalBytes: source.totalBytes()}
}

func (service *Service) get(pluginID, handle string) (*sourceSessionState, error) {
	service.mu.Lock()
	defer service.mu.Unlock()
	service.expireLocked()
	session, ok := service.sessions[handle]
	if !ok {
		return nil, sourceError("source-session-not-found", "%s", handle)
	}
	if session.pluginID != pluginID {
		return nil, sourceError("source-session-owner", "%s", handle)
	}
	session.lastUsed = service.now()
	return session, nil
}

func (service *Service) ListEntries(pluginID, handle, cursor string) (EntryPage, error) {
	session, err := service.get(pluginID, handle)
	if err != nil {
		return EntryPage{}, err
	}
	offset := 0
	if cursor != "" {
		offset, err = strconv.Atoi(cursor)
		if err != nil || offset < 0 {
			return EntryPage{}, sourceError("invalid-cursor", "%q", cursor)
		}
	}
	items := session.source.entries()
	if offset > len(items) {
		return EntryPage{}, sourceError("invalid-cursor", "%q", cursor)
	}
	end := offset + service.limits.pageSize
	if end > len(items) {
		end = len(items)
	}
	entries := make([]Entry, end-offset)
	for index := offset; index < end; index++ {
		entries[index-offset] = items[index].Entry
	}
	nextCursor := ""
	if end < len(items) {
		nextCursor = strconv.Itoa(end)
	}
	return EntryPage{Entries: entries, NextCursor: nextCursor, Fingerprint: session.source.fingerprint()}, nil
}

func (service *Service) ReadText(pluginID, handle, entryIDValue string) (string, error) {
	session, err := service.get(pluginID, handle)
	if err != nil {
		return "", err
	}
	reader, err := session.source.open(entryIDValue)
	if err != nil {
		return "", err
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, service.limits.maxTextBytes+1))
	if err != nil {
		return "", sourceError("source-unavailable", "could not read source entry")
	}
	if int64(len(data)) > service.limits.maxTextBytes {
		return "", sourceError("text-entry-too-large", "more than %d bytes", service.limits.maxTextBytes)
	}
	data = bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})
	if !utf8.Valid(data) || looksBinary(data) {
		return "", sourceError("binary-entry", "%s", entryIDValue)
	}
	return string(data), nil
}

func looksBinary(data []byte) bool {
	for _, value := range data {
		if value == 0 || (value < 0x08) || (value > 0x0d && value < 0x20) {
			return true
		}
	}
	return false
}

func (service *Service) Verify(pluginID, handle string) error {
	session, err := service.get(pluginID, handle)
	if err != nil {
		return err
	}
	return session.source.verify()
}

func (service *Service) Cancel(pluginID, handle string) error {
	service.mu.Lock()
	defer service.mu.Unlock()
	if session, ok := service.sessions[handle]; ok {
		if session.pluginID != pluginID {
			return sourceError("source-session-owner", "%s", handle)
		}
		if session.phase == "publishing" || session.phase == "refreshing" {
			return sourceError("import-not-cancellable", "publication has started")
		}
		if session.cancel != nil {
			session.cancel()
		}
		return nil
	}
	owner, closed := service.closedOwners[handle]
	if !closed {
		return sourceError("source-session-not-found", "%s", handle)
	}
	if owner != pluginID {
		return sourceError("source-session-owner", "%s", handle)
	}
	return nil
}

func (service *Service) Close(pluginID, handle string) error {
	service.mu.Lock()
	defer service.mu.Unlock()
	if session, ok := service.sessions[handle]; ok {
		if session.pluginID != pluginID {
			return sourceError("source-session-owner", "%s", handle)
		}
		delete(service.sessions, handle)
		service.closedOwners[handle] = pluginID
		return session.source.close()
	}
	if owner, ok := service.closedOwners[handle]; ok && owner != pluginID {
		return sourceError("source-session-owner", "%s", handle)
	}
	return nil
}

func (service *Service) ClosePlugin(pluginID string) {
	service.mu.Lock()
	defer service.mu.Unlock()
	for handle, session := range service.sessions {
		if session.pluginID != pluginID {
			continue
		}
		_ = session.source.close()
		delete(service.sessions, handle)
		service.closedOwners[handle] = pluginID
	}
}

func (service *Service) CloseAll() {
	service.mu.Lock()
	defer service.mu.Unlock()
	for handle, session := range service.sessions {
		_ = session.source.close()
		delete(service.sessions, handle)
		service.closedOwners[handle] = session.pluginID
	}
}

func (service *Service) expireLocked() {
	now := service.now()
	for handle, session := range service.sessions {
		if now.Sub(session.lastUsed) <= service.limits.sessionTTL {
			continue
		}
		_ = session.source.close()
		delete(service.sessions, handle)
		service.closedOwners[handle] = session.pluginID
	}
}
