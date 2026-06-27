package files

const MaxTextFileBytes int64 = 2 * 1024 * 1024

type FileType string

const (
	FileTypeFile    FileType = "file"
	FileTypeFolder  FileType = "folder"
	FileTypeSymlink FileType = "symlink"
	FileTypeUnknown FileType = "unknown"
)

type FileEntry struct {
	Name         string   `json:"name"`
	RelativePath string   `json:"relativePath"`
	Type         FileType `json:"type"`
	Size         int64    `json:"size"`
	ModifiedAt   string   `json:"modifiedAt"`
	Extension    string   `json:"extension"`
	IsHidden     bool     `json:"isHidden"`
	IsReserved   bool     `json:"isReserved"`
	CanRead      bool     `json:"canRead"`
	CanWrite     bool     `json:"canWrite"`
}

type FileMetadata struct {
	RelativePath string   `json:"relativePath"`
	Type         FileType `json:"type"`
	Size         int64    `json:"size"`
	ModifiedAt   string   `json:"modifiedAt"`
	CreatedAt    string   `json:"createdAt,omitempty"`
	Extension    string   `json:"extension"`
	MimeHint     string   `json:"mimeHint"`
	IsText       bool     `json:"isText"`
	IsHidden     bool     `json:"isHidden"`
	IsReserved   bool     `json:"isReserved"`
	CanRead      bool     `json:"canRead"`
	CanWrite     bool     `json:"canWrite"`
}

type ExternalOpenTarget struct {
	RelativePath string       `json:"relativePath"`
	AbsolutePath string       `json:"absolutePath"`
	Metadata     FileMetadata `json:"metadata"`
}

type WriteOptions struct {
	CreateIfMissing bool `json:"createIfMissing"`
	Overwrite       bool `json:"overwrite"`
}

type MoveOptions struct {
	Overwrite bool `json:"overwrite"`
}

type TrashResult struct {
	OriginalPath string `json:"originalPath"`
	TrashPath    string `json:"trashPath"`
	TrashID      string `json:"trashId"`
	DeletedAt    string `json:"deletedAt"`
}
