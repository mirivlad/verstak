package dealmigration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	journalPluginID       = "verstak.journal"
	legacyWorklogPrefix   = "worklog:workspace:"
	journalFolderName     = "Журнал"
	journalEntryMarkStart = "<!-- verstak-entry "
	journalEntryMarkEnd   = " -->"
)

// NewJournalDataTransform retires the path-keyed Journal settings store. It
// writes its useful content into ordinary Markdown files inside the current
// Deal before removing the legacy runtime keys; the transaction backup keeps
// the original bytes for recovery.
func NewJournalDataTransform() Transform {
	return FuncTransform("journal-settings-to-deal-markdown", migrateJournalData, verifyJournalData)
}

func migrateJournalData(ctx context.Context, vault string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path := filepath.Join(vault, ".verstak", "plugin-settings", journalPluginID, "settings.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("decode legacy Journal settings: %w", err)
	}
	rootToWorkspace, err := currentWorkspaceRootIDs(vault)
	if err != nil {
		return err
	}
	changed := false
	for key, value := range settings {
		if !strings.HasPrefix(key, legacyWorklogPrefix) {
			continue
		}
		root, err := url.PathUnescape(strings.TrimPrefix(key, legacyWorklogPrefix))
		if err != nil {
			return fmt.Errorf("decode legacy Journal Deal key %q: %w", key, err)
		}
		if _, exists := rootToWorkspace[root]; !exists {
			continue
		}
		records, ok := value.([]any)
		if !ok {
			return fmt.Errorf("legacy Journal key %q must contain an array", key)
		}
		if err := writeLegacyJournalRecords(vault, root, records); err != nil {
			return err
		}
		delete(settings, key)
		changed = true
	}
	if !changed {
		return nil
	}
	encoded, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomically(path, encoded)
}

func writeLegacyJournalRecords(vault, root string, records []any) error {
	byMonth := map[string][]map[string]any{}
	for index, raw := range records {
		record, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		date := strings.TrimSpace(stringValue(record["date"]))
		if len(date) < 7 {
			date = "1970-01-01"
		}
		month := date[:7]
		if !isJournalMonth(month) {
			month, date = "1970-01", "1970-01-01"
		}
		record = cloneJournalRecord(record)
		record["date"] = date
		if strings.TrimSpace(stringValue(record["entryId"])) == "" {
			record["entryId"] = fmt.Sprintf("legacy-journal:%s:%d", root, index)
		}
		byMonth[month] = append(byMonth[month], record)
	}
	for month, entries := range byMonth {
		path := filepath.Join(vault, filepath.FromSlash(root), journalFolderName, month+".md")
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return err
		}
		existing, err := os.ReadFile(path)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		content := string(existing)
		for _, entry := range entries {
			id := stringValue(entry["entryId"])
			if strings.Contains(content, `"entryId":"`+id+`"`) {
				continue
			}
			content += renderLegacyJournalEntry(entry)
		}
		if err := writeFileAtomically(path, []byte(content)); err != nil {
			return err
		}
	}
	return nil
}

func cloneJournalRecord(input map[string]any) map[string]any {
	result := make(map[string]any, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

func renderLegacyJournalEntry(entry map[string]any) string {
	metadata := map[string]any{"entryId": entry["entryId"], "minutes": entry["minutes"], "billable": entry["billable"]}
	for _, key := range []string{"sourceCandidateId", "sourceTodoId", "activityIds"} {
		if value, exists := entry[key]; exists {
			metadata[key] = value
		}
	}
	encoded, _ := json.Marshal(metadata)
	title := strings.Join(strings.Fields(stringValue(entry["title"])), " ")
	if title == "" {
		title = "Worklog entry"
	}
	summary := strings.TrimRight(stringValue(entry["summary"]), " \t\r\n")
	minutes := int(numberValue(entry["minutes"]))
	billing := "non-billable"
	if entry["billable"] == true {
		billing = "billable"
	}
	return "\n## " + stringValue(entry["date"]) + "\n\n### " + title + "\n\n" + fmt.Sprintf("%d min · %s", minutes, billing) + "\n\n" + summary + "\n\n" + journalEntryMarkStart + string(encoded) + journalEntryMarkEnd + "\n"
}

func isJournalMonth(value string) bool {
	if len(value) != 7 || value[4] != '-' {
		return false
	}
	for index, char := range value {
		if index == 4 {
			continue
		}
		if char < '0' || char > '9' {
			return false
		}
	}
	return value[5:] >= "01" && value[5:] <= "12"
}

func numberValue(value any) float64 {
	switch item := value.(type) {
	case float64:
		return item
	case int:
		return float64(item)
	default:
		return 0
	}
}

func verifyJournalData(ctx context.Context, vault string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path := filepath.Join(vault, ".verstak", "plugin-settings", journalPluginID, "settings.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return err
	}
	keys := make([]string, 0, len(settings))
	for key := range settings {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if strings.HasPrefix(key, legacyWorklogPrefix) {
			return fmt.Errorf("legacy Journal runtime key remains: %s", key)
		}
	}
	return nil
}
