package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	coresecrets "github.com/verstak/verstak-desktop/internal/core/secrets"
)

func (a *App) requirePluginSecretsAccess(pluginID string, write bool) error {
	if _, err := a.requirePluginAccess(pluginID, "secrets.read"); err != nil {
		return err
	}
	if write {
		if _, err := a.requirePluginAccess(pluginID, "secrets.write"); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) ensureSecretsSession() (*coresecrets.VaultSession, error) {
	if err := a.requireVault(); err != nil {
		return nil, err
	}
	if a.secretsSession == nil {
		a.secretsSession = coresecrets.NewVaultSession(filepath.Join(a.vaultPath(), ".verstak", "secrets"))
	}
	return a.secretsSession, nil
}

func (a *App) requireUnlockedSecretStore() (*coresecrets.Store, error) {
	session, err := a.ensureSecretsSession()
	if err != nil {
		return nil, err
	}
	return session.Store()
}

func (a *App) PluginSecretsStatus(pluginID string) (map[string]interface{}, string) {
	if err := a.requirePluginSecretsAccess(pluginID, false); err != nil {
		return nil, err.Error()
	}
	session, err := a.ensureSecretsSession()
	if err != nil {
		return nil, err.Error()
	}
	initialized, err := session.Initialized()
	if err != nil {
		return nil, err.Error()
	}
	return map[string]interface{}{
		"initialized": initialized,
		"unlocked":    session.Unlocked(),
	}, ""
}

func (a *App) PluginSecretsUnlock(pluginID, masterPassword string) string {
	if err := a.requirePluginSecretsAccess(pluginID, false); err != nil {
		return err.Error()
	}
	session, err := a.ensureSecretsSession()
	if err != nil {
		return err.Error()
	}
	if _, err := session.Unlock(masterPassword); err != nil {
		return err.Error()
	}
	return ""
}

func (a *App) PluginSecretsList(pluginID string) ([]map[string]interface{}, string) {
	if err := a.requirePluginSecretsAccess(pluginID, false); err != nil {
		return nil, err.Error()
	}
	store, err := a.requireUnlockedSecretStore()
	if err != nil {
		return nil, err.Error()
	}
	records, err := store.ListRecords()
	if err != nil {
		return nil, err.Error()
	}
	result := make([]map[string]interface{}, 0, len(records))
	for _, record := range records {
		result = append(result, secretRecordMap(record, false))
	}
	return result, ""
}

func (a *App) PluginSecretsRead(pluginID, secretID string) (map[string]interface{}, string) {
	if err := a.requirePluginSecretsAccess(pluginID, false); err != nil {
		return nil, err.Error()
	}
	store, err := a.requireUnlockedSecretStore()
	if err != nil {
		return nil, err.Error()
	}
	record, err := store.ReadRecord(secretID)
	if err != nil {
		return nil, err.Error()
	}
	return secretRecordMap(record, true), ""
}

func (a *App) PluginSecretsWrite(pluginID string, rawRecord map[string]interface{}) (map[string]interface{}, string) {
	if err := a.requirePluginSecretsAccess(pluginID, true); err != nil {
		return nil, err.Error()
	}
	store, err := a.requireUnlockedSecretStore()
	if err != nil {
		return nil, err.Error()
	}
	record, err := decodeSecretRecord(rawRecord)
	if err != nil {
		return nil, err.Error()
	}
	if err := store.WriteRecord(record); err != nil {
		return nil, err.Error()
	}
	written, err := store.ReadRecord(record.ID)
	if err != nil {
		return nil, err.Error()
	}
	return secretRecordMap(written, false), ""
}

func (a *App) PluginSecretsDelete(pluginID, secretID string) string {
	if err := a.requirePluginSecretsAccess(pluginID, true); err != nil {
		return err.Error()
	}
	store, err := a.requireUnlockedSecretStore()
	if err != nil {
		return err.Error()
	}
	if err := store.Delete(secretID); err != nil {
		return err.Error()
	}
	return ""
}

func (a *App) PluginSecretsCopyLink(pluginID, secretID string) (string, string) {
	if err := a.requirePluginSecretsAccess(pluginID, false); err != nil {
		return "", err.Error()
	}
	store, err := a.requireUnlockedSecretStore()
	if err != nil {
		return "", err.Error()
	}
	record, err := store.ReadRecord(secretID)
	if err != nil {
		return "", err.Error()
	}
	title := strings.TrimSpace(record.Title)
	if title == "" {
		title = record.ID
	}
	return fmt.Sprintf("[%s](verstak-secret://%s)", title, url.PathEscape(record.ID)), ""
}

func decodeSecretRecord(raw map[string]interface{}) (coresecrets.SecretRecord, error) {
	data, err := json.Marshal(raw)
	if err != nil {
		return coresecrets.SecretRecord{}, err
	}
	var record coresecrets.SecretRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return coresecrets.SecretRecord{}, err
	}
	if strings.TrimSpace(record.ID) == "" {
		record.ID = generatedSecretID(record.Title)
	}
	return record, nil
}

func generatedSecretID(title string) string {
	base := strings.ToLower(strings.TrimSpace(title))
	base = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '.' || r == '_' || r == '-' {
			return r
		}
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			return '-'
		}
		return -1
	}, base)
	base = strings.Trim(base, ".-_")
	if base == "" {
		base = "secret"
	}
	return fmt.Sprintf("%s-%d", base, time.Now().UnixNano())
}

func secretRecordMap(record coresecrets.SecretRecord, includeValue bool) map[string]interface{} {
	result := map[string]interface{}{
		"id":        record.ID,
		"title":     record.Title,
		"scope":     map[string]interface{}{"kind": record.Scope.Kind, "workspaceRootPath": record.Scope.WorkspaceRootPath},
		"username":  record.Username,
		"updatedAt": record.UpdatedAt,
	}
	if includeValue {
		result["value"] = record.Value
	}
	return result
}
