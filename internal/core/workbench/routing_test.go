package workbench

import (
	"testing"

	"github.com/verstak/verstak-desktop/internal/core/contribution"
	"github.com/verstak/verstak-desktop/internal/core/plugin"
)

func provider(pluginID, id string, priority int, component string, supports ...plugin.OpenProviderSupport) contribution.ContributionOpenProvider {
	return contribution.ContributionOpenProvider{
		PluginID: pluginID,
		Item: plugin.ContributionOpenProvider{
			ID:        id,
			Title:     id,
			Priority:  priority,
			Component: component,
			Supports:  supports,
		},
	}
}

func TestSelectProviderUsesNotesMarkdownPreference(t *testing.T) {
	r := NewRouter(Preferences{
		DefaultNotesMarkdownEditorProvider: "community.notes-editor",
	})
	providers := []contribution.ContributionOpenProvider{
		provider("official.editor", "official.markdown", 100, "OfficialMarkdown", plugin.OpenProviderSupport{
			Kind:       "vault-file",
			Extensions: []string{".md", ".markdown"},
			Contexts:   []string{"generic-markdown", "notes-markdown"},
		}),
		provider("community.editor", "community.notes-editor", 10, "CommunityNotes", plugin.OpenProviderSupport{
			Kind:       "vault-file",
			Extensions: []string{".md"},
			Contexts:   []string{"notes-markdown"},
		}),
	}

	selected, err := r.SelectProvider(OpenResourceRequest{
		Kind:      "vault-file",
		Path:      "Clients/Acme/Notes/Overview.md",
		Extension: ".md",
		Mode:      "edit",
		Context: OpenResourceContext{
			SourceView:          "notes",
			IsInsideNotesFolder: true,
			NotesMode:           true,
		},
	}, providers)
	if err != nil {
		t.Fatalf("SelectProvider: %v", err)
	}
	if selected.Item.ID != "community.notes-editor" {
		t.Fatalf("provider = %q, want community.notes-editor", selected.Item.ID)
	}
}

func TestSelectProviderFallsBackByPriorityThenID(t *testing.T) {
	r := NewRouter(Preferences{
		DefaultMarkdownEditorProvider: "disabled.or.missing",
	})
	providers := []contribution.ContributionOpenProvider{
		provider("b.plugin", "b.provider", 100, "B", plugin.OpenProviderSupport{
			Kind:       "vault-file",
			Extensions: []string{".md"},
			Contexts:   []string{"generic-markdown"},
		}),
		provider("a.plugin", "a.provider", 100, "A", plugin.OpenProviderSupport{
			Kind:       "vault-file",
			Extensions: []string{".md"},
			Contexts:   []string{"generic-markdown"},
		}),
		provider("high.plugin", "high.provider", 200, "High", plugin.OpenProviderSupport{
			Kind:       "vault-file",
			Extensions: []string{".txt"},
		}),
	}

	selected, err := r.SelectProvider(OpenResourceRequest{
		Kind:      "vault-file",
		Path:      "Docs/readme.md",
		Extension: ".md",
		Mode:      "view",
	}, providers)
	if err != nil {
		t.Fatalf("SelectProvider: %v", err)
	}
	if selected.Item.ID != "a.provider" {
		t.Fatalf("provider = %q, want deterministic tie winner a.provider", selected.Item.ID)
	}
}

func TestSelectProviderTieBreaksByPluginIDThenProviderID(t *testing.T) {
	r := NewRouter(Preferences{})
	providers := []contribution.ContributionOpenProvider{
		provider("b.plugin", "a.provider", 100, "B", plugin.OpenProviderSupport{
			Kind:       "vault-file",
			Extensions: []string{".md"},
			Contexts:   []string{ContextGenericMarkdown},
		}),
		provider("a.plugin", "z.provider", 100, "A", plugin.OpenProviderSupport{
			Kind:       "vault-file",
			Extensions: []string{".md"},
			Contexts:   []string{ContextGenericMarkdown},
		}),
	}

	selected, err := r.SelectProvider(OpenResourceRequest{
		Kind: "vault-file",
		Path: "Docs/readme.md",
		Mode: "view",
	}, providers)
	if err != nil {
		t.Fatalf("SelectProvider: %v", err)
	}
	if selected.PluginID != "a.plugin" || selected.Item.ID != "z.provider" {
		t.Fatalf("provider = %+v, want a.plugin/z.provider", selected)
	}
}

func TestSelectProviderMatchesGenericTextContext(t *testing.T) {
	r := NewRouter(Preferences{})
	selected, err := r.SelectProvider(OpenResourceRequest{
		Kind: "vault-file",
		Path: "Docs/readme.txt",
		Mode: "view",
	}, []contribution.ContributionOpenProvider{
		provider("text.plugin", "text.provider", 10, "Text", plugin.OpenProviderSupport{
			Kind:       "vault-file",
			Extensions: []string{".txt"},
			Contexts:   []string{ContextGenericText},
		}),
	})
	if err != nil {
		t.Fatalf("SelectProvider: %v", err)
	}
	if selected.Item.ID != "text.provider" {
		t.Fatalf("provider = %q, want text.provider", selected.Item.ID)
	}
}

func TestGenericMarkdownDoesNotSelectNotesOnlyProvider(t *testing.T) {
	r := NewRouter(Preferences{})
	_, err := r.SelectProvider(OpenResourceRequest{
		Kind: "vault-file",
		Path: "Docs/readme.md",
		Mode: "view",
	}, []contribution.ContributionOpenProvider{
		provider("notes.plugin", "notes.provider", 10, "Notes", plugin.OpenProviderSupport{
			Kind:       "vault-file",
			Extensions: []string{".md"},
			Contexts:   []string{ContextNotesMarkdown},
		}),
	})
	if err == nil {
		t.Fatal("expected no provider for generic markdown with notes-only provider")
	}
}

func TestOpenResourceStoresSelectedProviderAndRequest(t *testing.T) {
	r := NewRouter(Preferences{})
	result, err := r.OpenResource(OpenResourceRequest{
		Kind:      "vault-file",
		Path:      "Notes/Overview.md",
		Extension: ".md",
		Mode:      "edit",
		Context: OpenResourceContext{
			IsInsideNotesFolder: true,
			NotesMode:           true,
		},
	}, []contribution.ContributionOpenProvider{
		provider("official.editor", "official.markdown", 100, "MarkdownEditor", plugin.OpenProviderSupport{
			Kind:       "vault-file",
			Extensions: []string{".md"},
			Contexts:   []string{"notes-markdown"},
		}),
	})
	if err != nil {
		t.Fatalf("OpenResource: %v", err)
	}
	if result.Status != "opened" || result.ProviderID != "official.markdown" || result.ProviderComponent != "MarkdownEditor" {
		t.Fatalf("result = %+v", result)
	}
	opened := r.OpenedResources()
	if len(opened) != 1 || opened[0].Request.Path != "Notes/Overview.md" {
		t.Fatalf("opened = %+v", opened)
	}
}

func TestOpenResourceReturnsNoProviderFallback(t *testing.T) {
	r := NewRouter(Preferences{})
	result, err := r.OpenResource(OpenResourceRequest{
		Kind: "vault-file",
		Path: "Docs/unknown.bin",
	}, nil)
	if err != nil {
		t.Fatalf("OpenResource: %v", err)
	}
	if result.Status != "no-provider" || result.Request.Path != "Docs/unknown.bin" || result.Message == "" {
		t.Fatalf("result = %+v", result)
	}
	if len(r.OpenedResources()) != 0 {
		t.Fatalf("no-provider result should not store opened resource: %+v", r.OpenedResources())
	}
}

func TestDetermineContextName(t *testing.T) {
	tests := []struct {
		name    string
		request OpenResourceRequest
		want    string
	}{
		{
			name:    "text",
			request: OpenResourceRequest{Kind: "vault-file", Path: "Docs/readme.txt"},
			want:    ContextGenericText,
		},
		{
			name:    "markdown",
			request: OpenResourceRequest{Kind: "vault-file", Path: "Docs/readme.md"},
			want:    ContextGenericMarkdown,
		},
		{
			name: "notes markdown",
			request: OpenResourceRequest{
				Kind: "vault-file",
				Path: "Notes/Overview.md",
				Context: OpenResourceContext{
					IsInsideNotesFolder: true,
					NotesMode:           true,
				},
			},
			want: ContextNotesMarkdown,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DetermineContextName(tt.request); got != tt.want {
				t.Fatalf("DetermineContextName = %q, want %q", got, tt.want)
			}
		})
	}
}
