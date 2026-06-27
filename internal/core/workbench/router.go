// Package workbench routes open/edit resource requests to contributed providers.
package workbench

import (
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/verstak/verstak-desktop/internal/core/contribution"
	"github.com/verstak/verstak-desktop/internal/core/plugin"
)

type Preferences struct {
	DefaultTextEditorProvider          string `json:"defaultTextEditorProvider,omitempty"`
	DefaultMarkdownEditorProvider      string `json:"defaultMarkdownEditorProvider,omitempty"`
	DefaultNotesMarkdownEditorProvider string `json:"defaultNotesMarkdownEditorProvider,omitempty"`
}

const (
	ContextGenericText     = "generic-text"
	ContextGenericMarkdown = "generic-markdown"
	ContextNotesMarkdown   = "notes-markdown"
)

type OpenResourceContext struct {
	SourcePluginID      string `json:"sourcePluginId,omitempty"`
	SourceView          string `json:"sourceView,omitempty"`
	IsInsideNotesFolder bool   `json:"isInsideNotesFolder,omitempty"`
	NotesScopePath      string `json:"notesScopePath,omitempty"`
	NotesMode           bool   `json:"notesMode,omitempty"`
}

type OpenResourceRequest struct {
	Kind      string              `json:"kind"`
	Path      string              `json:"path"`
	Mode      string              `json:"mode,omitempty"`
	Mime      string              `json:"mime,omitempty"`
	Extension string              `json:"extension,omitempty"`
	Context   OpenResourceContext `json:"context,omitempty"`
}

type OpenResourceResult struct {
	Status            string              `json:"status"`
	ProviderID        string              `json:"providerId,omitempty"`
	ProviderPluginID  string              `json:"providerPluginId,omitempty"`
	ProviderComponent string              `json:"providerComponent,omitempty"`
	Request           OpenResourceRequest `json:"request"`
	Message           string              `json:"message,omitempty"`
}

type OpenedResource struct {
	ID                string              `json:"id"`
	ProviderID        string              `json:"providerId"`
	ProviderPluginID  string              `json:"providerPluginId"`
	ProviderComponent string              `json:"providerComponent"`
	Request           OpenResourceRequest `json:"request"`
	OpenedAt          string              `json:"openedAt"`
}

type Router struct {
	preferences Preferences
	opened      []OpenedResource
}

func NewRouter(preferences Preferences) *Router {
	return &Router{preferences: preferences}
}

func (r *Router) Preferences() Preferences {
	return r.preferences
}

func (r *Router) SetPreferences(preferences Preferences) {
	r.preferences = preferences
}

func (r *Router) SelectProvider(request OpenResourceRequest, providers []contribution.ContributionOpenProvider) (contribution.ContributionOpenProvider, error) {
	request = normalizeRequest(request)
	var matches []contribution.ContributionOpenProvider
	for _, provider := range providers {
		if providerMatches(request, provider.Item) {
			matches = append(matches, provider)
		}
	}
	if len(matches) == 0 {
		return contribution.ContributionOpenProvider{}, fmt.Errorf("no open provider supports %s %q", request.Kind, request.Path)
	}

	preferred := r.preferenceFor(request)
	if preferred != "" {
		for _, provider := range matches {
			if provider.Item.ID == preferred {
				return provider, nil
			}
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Item.Priority != matches[j].Item.Priority {
			return matches[i].Item.Priority > matches[j].Item.Priority
		}
		if matches[i].PluginID != matches[j].PluginID {
			return matches[i].PluginID < matches[j].PluginID
		}
		return matches[i].Item.ID < matches[j].Item.ID
	})
	return matches[0], nil
}

func (r *Router) OpenResource(request OpenResourceRequest, providers []contribution.ContributionOpenProvider) (OpenResourceResult, error) {
	request = normalizeRequest(request)
	provider, err := r.SelectProvider(request, providers)
	if err != nil {
		return OpenResourceResult{
			Status:  "no-provider",
			Request: request,
			Message: err.Error(),
		}, nil
	}

	result := OpenResourceResult{
		Status:            "opened",
		ProviderID:        provider.Item.ID,
		ProviderPluginID:  provider.PluginID,
		ProviderComponent: provider.Item.Component,
		Request:           request,
	}
	r.opened = append(r.opened, OpenedResource{
		ID:                fmt.Sprintf("%s:%d", provider.Item.ID, len(r.opened)+1),
		ProviderID:        result.ProviderID,
		ProviderPluginID:  result.ProviderPluginID,
		ProviderComponent: result.ProviderComponent,
		Request:           result.Request,
		OpenedAt:          time.Now().UTC().Format(time.RFC3339Nano),
	})
	return result, nil
}

func (r *Router) OpenedResources() []OpenedResource {
	result := make([]OpenedResource, len(r.opened))
	copy(result, r.opened)
	return result
}

func normalizeRequest(request OpenResourceRequest) OpenResourceRequest {
	if request.Mode == "" {
		request.Mode = "view"
	}
	if request.Extension == "" {
		request.Extension = path.Ext(request.Path)
	}
	request.Extension = strings.ToLower(request.Extension)
	request.Mime = strings.ToLower(request.Mime)
	return request
}

// DetermineContextName derives the current routing context from a request.
// Future Files/Notes callers can move canonical Notes folder auto-detection here.
func DetermineContextName(request OpenResourceRequest) string {
	request = normalizeRequest(request)
	return resourceContextName(request)
}

func providerMatches(request OpenResourceRequest, provider plugin.ContributionOpenProvider) bool {
	for _, support := range provider.Supports {
		if support.Kind != request.Kind {
			continue
		}
		if !supportMatchesExtensionOrMime(request, support) {
			continue
		}
		if !supportMatchesContext(request, support) {
			continue
		}
		return true
	}
	return false
}

func supportMatchesExtensionOrMime(request OpenResourceRequest, support plugin.OpenProviderSupport) bool {
	hasExtensionRules := len(support.Extensions) > 0
	hasMimeRules := len(support.Mime) > 0
	if !hasExtensionRules && !hasMimeRules {
		return true
	}

	if hasExtensionRules {
		for _, ext := range support.Extensions {
			if strings.ToLower(ext) == request.Extension {
				return true
			}
		}
	}
	if hasMimeRules && request.Mime != "" {
		for _, mime := range support.Mime {
			if strings.ToLower(mime) == request.Mime {
				return true
			}
		}
	}
	return false
}

func supportMatchesContext(request OpenResourceRequest, support plugin.OpenProviderSupport) bool {
	if len(support.Contexts) == 0 {
		return true
	}
	context := resourceContextName(request)
	for _, supported := range support.Contexts {
		if supported == context {
			return true
		}
	}
	return false
}

func (r *Router) preferenceFor(request OpenResourceRequest) string {
	context := resourceContextName(request)
	switch {
	case context == ContextNotesMarkdown:
		return r.preferences.DefaultNotesMarkdownEditorProvider
	case context == ContextGenericMarkdown:
		return r.preferences.DefaultMarkdownEditorProvider
	case context == ContextGenericText:
		return r.preferences.DefaultTextEditorProvider
	default:
		return ""
	}
}

func resourceContextName(request OpenResourceRequest) string {
	ext := strings.ToLower(request.Extension)
	if ext == ".md" || ext == ".markdown" {
		// Auto-detect Notes context: either explicitly set in request context
		// or path-based detection using the canonical Notes/ folder layout.
		if request.Context.NotesMode || request.Context.IsInsideNotesFolder || isInsideNotesPath(request.Path) {
			return ContextNotesMarkdown
		}
		return ContextGenericMarkdown
	}
	if isTextResource(request) {
		return ContextGenericText
	}
	return ""
}

func isInsideNotesPath(relativePath string) bool {
	if relativePath == "" {
		return false
	}
	cleaned := strings.TrimSpace(relativePath)
	cleaned = strings.TrimPrefix(cleaned, "./")
	cleaned = strings.TrimPrefix(cleaned, "/")
	for _, part := range strings.Split(cleaned, "/") {
		if part == "Notes" {
			return true
		}
	}
	return false
}

func isTextResource(request OpenResourceRequest) bool {
	if strings.HasPrefix(request.Mime, "text/") {
		return true
	}
	switch strings.ToLower(request.Extension) {
	case ".txt", ".log", ".json", ".yaml", ".yml", ".toml", ".ini", ".conf":
		return true
	default:
		return false
	}
}
