package files

const MaxTextFileBytes int64 = 2 * 1024 * 1024
const MaxBinaryReadBytes int64 = 8 * 1024 * 1024

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

type FileBytes struct {
	RelativePath string `json:"relativePath"`
	Size         int64  `json:"size"`
	MimeHint     string `json:"mimeHint"`
	DataBase64   string `json:"dataBase64"`
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

type CopyOptions struct {
	Overwrite bool `json:"overwrite"`
}

type RestoreOptions struct {
	TargetPath string `json:"targetPath,omitempty"`
	Overwrite  bool   `json:"overwrite"`
}

// PathTransfer is one source-to-destination pair in a bulk move or copy.
type PathTransfer struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// TransferResult reports what became of a single item in a bulk transfer. An
// empty Error means it succeeded; Skipped marks an item the batch never reached
// because it was cancelled.
type TransferResult struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Error   string `json:"error,omitempty"`
	Skipped bool   `json:"skipped,omitempty"`
}

// TransferOutcome is the whole result of a bulk transfer.
//
// One bad item does not abandon the rest: somebody pasting two hundred files
// should not lose a hundred and ninety-nine of them to a single name clash. The
// caller gets a per-item verdict and can report exactly what did not land.
type TransferOutcome struct {
	Results   []TransferResult `json:"results"`
	Succeeded int              `json:"succeeded"`
	Failed    int              `json:"failed"`
	Cancelled bool             `json:"cancelled"`
}

type TrashResult struct {
	OriginalPath string `json:"originalPath"`
	TrashPath    string `json:"trashPath"`
	TrashID      string `json:"trashId"`
	DeletedAt    string `json:"deletedAt"`
	Size         int64  `json:"size"`
}

type TrashEntry struct {
	OriginalPath string   `json:"originalPath"`
	TrashPath    string   `json:"trashPath"`
	TrashID      string   `json:"trashId"`
	DeletedAt    string   `json:"deletedAt"`
	OriginalType FileType `json:"originalType"`
	Basename     string   `json:"basename"`
	Size         int64    `json:"size"`
}
