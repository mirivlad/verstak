// Package workspace contains legacy API DTOs. Deal lifecycle and metadata are
// owned exclusively by workspacetree.Service.
package workspace

type NodeType string

const (
	TypeSpace  NodeType = "space"
	TypeCase   NodeType = "case"
	TypeFolder NodeType = "folder"
)

type NodeStatus string

const (
	StatusActive   NodeStatus = "active"
	StatusSleeping NodeStatus = "sleeping"
	StatusArchived NodeStatus = "archived"
)

type Workspace struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	RootPath string `json:"rootPath"`
}

type TemplateSnapshot struct {
	TemplateID      string   `json:"templateId"`
	TemplateName    string   `json:"templateName"`
	TemplateVersion int      `json:"templateVersion"`
	AppliedAt       string   `json:"appliedAt"`
	WorkspaceTools  []string `json:"workspaceTools,omitempty"`
}

type Metadata struct {
	WorkspaceID         string            `json:"workspaceId,omitempty"`
	WorkspaceName       string            `json:"workspaceName"`
	CreatedFromTemplate *TemplateSnapshot `json:"createdFromTemplate,omitempty"`
	Features            map[string]bool   `json:"features,omitempty"`
	Folders             map[string]string `json:"folders,omitempty"`
	WorkspaceTools      []string          `json:"workspaceTools,omitempty"`
	UpdatedAt           string            `json:"updatedAt,omitempty"`
}

type MetadataPatch struct {
	Features       map[string]bool   `json:"features,omitempty"`
	Folders        map[string]string `json:"folders,omitempty"`
	WorkspaceTools []string          `json:"workspaceTools,omitempty"`
}

type TrashResult struct {
	WorkspaceID  string `json:"workspaceId"`
	OriginalPath string `json:"originalPath"`
	TrashPath    string `json:"trashPath"`
	TrashID      string `json:"trashId"`
	DeletedAt    string `json:"deletedAt"`
}

type WorkspaceIdentity struct {
	WorkspaceID string `json:"workspaceId"`
	RootPath    string `json:"rootPath"`
	State       string `json:"state"`
}

type WorkspaceNode struct {
	ID        string     `json:"id"`
	ParentID  string     `json:"parentId,omitempty"`
	Type      NodeType   `json:"type"`
	Title     string     `json:"title"`
	Name      string     `json:"name,omitempty"`
	RootPath  string     `json:"rootPath,omitempty"`
	Path      string     `json:"-"`
	Status    NodeStatus `json:"status"`
	Tags      []string   `json:"tags,omitempty"`
	Order     int        `json:"order"`
	CreatedAt string     `json:"createdAt,omitempty"`
	UpdatedAt string     `json:"updatedAt,omitempty"`
}

type WorkspaceTree struct {
	SchemaVersion int             `json:"schemaVersion"`
	Nodes         []WorkspaceNode `json:"nodes"`
	CurrentNodeID string          `json:"currentNodeId,omitempty"`
	UpdatedAt     string          `json:"updatedAt"`
}
