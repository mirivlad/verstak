package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/verstak/verstak-desktop/internal/core/gitservice"
)

// PluginGitClone creates a device-local checkout below the selected Deal's
// managed Repositories root. Credentials are resolved from a secret reference
// here and never cross the frontend-to-Git process boundary as persistent data.
func (a *App) PluginGitClone(pluginID string, rawRequest map[string]interface{}) (map[string]interface{}, string) {
	request, credential, err := a.decodePluginGitRequest(pluginID, rawRequest, true, true)
	if err != nil {
		return nil, err.Error()
	}
	checkout, err := gitservice.NewService(a.vaultPath()).Clone(gitservice.CloneRequest{
		WorkspaceID: request.WorkspaceID, WorkspaceRoot: request.WorkspaceRoot,
		RepositoryID: request.RepositoryID, CheckoutName: request.CheckoutName,
		RemoteURL: request.RemoteURL, Branch: request.Branch, Credential: credential,
	})
	if err != nil {
		return nil, err.Error()
	}
	return map[string]interface{}{"checkoutPath": checkout}, ""
}

// PluginGitRegisterExisting copies a selected external Git worktree into the
// managed Deal checkout root without retaining the external source path.
func (a *App) PluginGitRegisterExisting(pluginID string, rawRequest map[string]interface{}) (map[string]interface{}, string) {
	if _, err := a.requirePluginAccess(pluginID, "imports.readExternal"); err != nil {
		return nil, err.Error()
	}
	request, _, err := a.decodePluginGitRequest(pluginID, rawRequest, false, false)
	if err != nil {
		return nil, err.Error()
	}
	if request.SourcePath == "" {
		dialog := a.selectImportDirectory
		if dialog == nil {
			dialog = runtime.OpenDirectoryDialog
		}
		home, _ := os.UserHomeDir()
		selected, dialogErr := dialog(a.ctx, runtime.OpenDialogOptions{Title: "Select existing Git repository", DefaultDirectory: home})
		if dialogErr != nil {
			return nil, dialogErr.Error()
		}
		if selected == "" {
			return map[string]interface{}{"checkoutPath": ""}, ""
		}
		request.SourcePath = selected
	}
	checkout, err := gitservice.NewService(a.vaultPath()).RegisterExisting(gitservice.ExistingRepositoryRequest{
		RepositoryRequest: gitservice.RepositoryRequest{WorkspaceID: request.WorkspaceID, WorkspaceRoot: request.WorkspaceRoot, RepositoryID: request.RepositoryID, CheckoutName: request.CheckoutName},
		SourcePath:        request.SourcePath,
	})
	if err != nil {
		return nil, err.Error()
	}
	return map[string]interface{}{"checkoutPath": checkout}, ""
}

func (a *App) PluginGitStatus(pluginID string, rawRequest map[string]interface{}) (map[string]interface{}, string) {
	request, _, err := a.decodePluginGitRequest(pluginID, rawRequest, false, false)
	if err != nil {
		return nil, err.Error()
	}
	status, err := gitservice.NewService(a.vaultPath()).Status(gitservice.RepositoryRequest{WorkspaceID: request.WorkspaceID, WorkspaceRoot: request.WorkspaceRoot, RepositoryID: request.RepositoryID, CheckoutName: request.CheckoutName})
	if err != nil {
		return nil, err.Error()
	}
	return pluginGitStatusMap(status), ""
}

func (a *App) PluginGitFetch(pluginID string, rawRequest map[string]interface{}) string {
	return a.pluginGitRemoteOperation(pluginID, rawRequest, func(service *gitservice.Service, request gitservice.RepositoryRequest) error {
		return service.Fetch(request)
	})
}

func (a *App) PluginGitPull(pluginID string, rawRequest map[string]interface{}) string {
	return a.pluginGitRemoteOperation(pluginID, rawRequest, func(service *gitservice.Service, request gitservice.RepositoryRequest) error {
		return service.Pull(request)
	})
}

func (a *App) PluginGitPush(pluginID string, rawRequest map[string]interface{}) string {
	return a.pluginGitRemoteOperation(pluginID, rawRequest, func(service *gitservice.Service, request gitservice.RepositoryRequest) error {
		return service.Push(request)
	})
}

func (a *App) PluginGitOpenDirectory(pluginID string, rawRequest map[string]interface{}) string {
	if _, err := a.requirePluginAccess(pluginID, "files.openExternal"); err != nil {
		return err.Error()
	}
	request, _, err := a.decodePluginGitRequest(pluginID, rawRequest, false, false)
	if err != nil {
		return err.Error()
	}
	path, err := gitservice.NewService(a.vaultPath()).CheckoutPath(gitservice.RepositoryRequest{WorkspaceID: request.WorkspaceID, WorkspaceRoot: request.WorkspaceRoot, RepositoryID: request.RepositoryID, CheckoutName: request.CheckoutName})
	if err != nil {
		return err.Error()
	}
	if err := a.externalOpenService().ShowInFolder(path, true); err != nil {
		return err.Error()
	}
	return ""
}

type pluginGitRequest struct {
	WorkspaceID   string `json:"workspaceId"`
	RepositoryID  string `json:"repositoryId"`
	CheckoutName  string `json:"checkoutName"`
	RemoteURL     string `json:"remoteUrl"`
	Branch        string `json:"branch"`
	CredentialRef string `json:"credentialRef"`
	SourcePath    string `json:"sourcePath"`
	WorkspaceRoot string
}

func (a *App) pluginGitRemoteOperation(pluginID string, rawRequest map[string]interface{}, operation func(*gitservice.Service, gitservice.RepositoryRequest) error) string {
	request, credential, err := a.decodePluginGitRequest(pluginID, rawRequest, true, true)
	if err != nil {
		return err.Error()
	}
	if err := operation(gitservice.NewService(a.vaultPath()), gitservice.RepositoryRequest{WorkspaceID: request.WorkspaceID, WorkspaceRoot: request.WorkspaceRoot, RepositoryID: request.RepositoryID, CheckoutName: request.CheckoutName, Credential: credential}); err != nil {
		return err.Error()
	}
	return ""
}

func (a *App) decodePluginGitRequest(pluginID string, raw map[string]interface{}, remote, requireRemoteURL bool) (pluginGitRequest, gitservice.Credential, error) {
	if _, err := a.requirePluginAccess(pluginID, "process.spawn"); err != nil {
		return pluginGitRequest{}, gitservice.Credential{}, err
	}
	if remote {
		if _, err := a.requirePluginAccess(pluginID, "network.remote"); err != nil {
			return pluginGitRequest{}, gitservice.Credential{}, err
		}
	}
	if err := a.requireVault(); err != nil {
		return pluginGitRequest{}, gitservice.Credential{}, err
	}
	if a.treeV2 == nil {
		return pluginGitRequest{}, gitservice.Credential{}, fmt.Errorf("workspace tree not initialized")
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return pluginGitRequest{}, gitservice.Credential{}, err
	}
	var request pluginGitRequest
	if err := json.Unmarshal(encoded, &request); err != nil {
		return pluginGitRequest{}, gitservice.Credential{}, err
	}
	request.WorkspaceID = strings.TrimSpace(request.WorkspaceID)
	request.RepositoryID = strings.TrimSpace(request.RepositoryID)
	request.CheckoutName = strings.TrimSpace(request.CheckoutName)
	request.RemoteURL = strings.TrimSpace(request.RemoteURL)
	request.Branch = strings.TrimSpace(request.Branch)
	request.CredentialRef = strings.TrimSpace(request.CredentialRef)
	request.SourcePath = strings.TrimSpace(request.SourcePath)
	if request.RepositoryID == "" || request.CheckoutName == "" {
		return pluginGitRequest{}, gitservice.Credential{}, fmt.Errorf("repositoryId and checkoutName are required")
	}
	workspace, ok := a.treeV2.GetWorkspaceByID(request.WorkspaceID)
	if !ok {
		return pluginGitRequest{}, gitservice.Credential{}, fmt.Errorf("workspace not found")
	}
	if !a.workspaceHasTool(workspace.ID, pluginID) {
		return pluginGitRequest{}, gitservice.Credential{}, fmt.Errorf("Git is not enabled for this Deal")
	}
	request.WorkspaceRoot = workspace.RootPath
	if requireRemoteURL {
		if err := validateGitRemoteURL(request.RemoteURL); err != nil {
			return pluginGitRequest{}, gitservice.Credential{}, err
		}
	}
	if !remote || request.CredentialRef == "" {
		return request, gitservice.Credential{}, nil
	}
	if _, err := a.requirePluginAccess(pluginID, "secrets.read"); err != nil {
		return pluginGitRequest{}, gitservice.Credential{}, err
	}
	secretID, err := parseGitCredentialReference(request.CredentialRef)
	if err != nil {
		return pluginGitRequest{}, gitservice.Credential{}, err
	}
	store, err := a.requireUnlockedSecretStore()
	if err != nil {
		return pluginGitRequest{}, gitservice.Credential{}, err
	}
	record, err := store.ReadRecord(secretID)
	if err != nil {
		return pluginGitRequest{}, gitservice.Credential{}, err
	}
	return request, gitservice.Credential{Username: record.Username, Value: record.Value, PrivateKey: gitRemoteUsesSSH(request.RemoteURL)}, nil
}

func validateGitRemoteURL(rawURL string) error {
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil || parsed.Host == "" {
		if strings.Count(rawURL, "@") == 1 && strings.Contains(rawURL, ":") && !strings.Contains(rawURL, "://") {
			return nil // scp-like SSH remote, e.g. git@example.com:owner/repo.git
		}
		return fmt.Errorf("Git remote URL is invalid")
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" && parsed.Scheme != "ssh" {
		return fmt.Errorf("Git remote URL scheme is unsupported")
	}
	if parsed.User != nil && parsed.User.Username() != "" && (parsed.Scheme == "http" || parsed.Scheme == "https") {
		return fmt.Errorf("Git remote URL must not contain credentials")
	}
	if _, passwordSet := parsed.User.Password(); passwordSet {
		return fmt.Errorf("Git remote URL must not contain credentials")
	}
	return nil
}

func gitRemoteUsesSSH(remoteURL string) bool {
	return strings.HasPrefix(remoteURL, "ssh://") || (strings.Contains(remoteURL, "@") && strings.Contains(remoteURL, ":") && !strings.Contains(remoteURL, "://"))
}

func parseGitCredentialReference(reference string) (string, error) {
	parsed, err := url.ParseRequestURI(reference)
	if err != nil || parsed.Scheme != "verstak-secret" || parsed.Host == "" || parsed.Path != "" {
		return "", fmt.Errorf("Git credential reference is invalid")
	}
	secretID, err := url.PathUnescape(parsed.Host)
	if err != nil || strings.TrimSpace(secretID) == "" {
		return "", fmt.Errorf("Git credential reference is invalid")
	}
	return secretID, nil
}

func pluginGitStatusMap(status gitservice.Status) map[string]interface{} {
	return map[string]interface{}{"state": status.State, "branch": status.Branch, "clean": status.Clean, "changedCount": status.ChangedCount, "untrackedCount": status.UntrackedCount, "changedFiles": status.ChangedFiles, "ahead": status.Ahead, "behind": status.Behind, "recentCommits": status.RecentCommits}
}
