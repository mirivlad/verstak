// Package pluginsync carries plugin-owned data between devices.
//
// Plugin data lives under `.verstak`, which the vault scanner skips on purpose:
// it is not the user's documents. So a todo written on the laptop never reached
// the desktop, and neither did anything else a plugin kept for itself.
//
// The unit that travels is a record, not a file. A plugin declares that one of
// its documents is a list of objects and which field identifies them; every
// change to that list becomes one operation per record in the same log that
// carries files. Two devices adding different todos then produce two records
// rather than two versions of one document, and nobody has to be asked which
// copy to keep -- which matters, because a JSON blob under `.verstak` is not
// something a person can open and merge by hand.
//
// What this deliberately does not do is guess. A record changed on both devices
// is resolved the way every other operation is: by server sequence, last write
// wins. That is a real loss of one edit, and it is the same rule the rest of
// sync already follows rather than a new one invented here.
package pluginsync

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Storage kinds a record set can live in.
const (
	StorageSettings = "settings"
	StorageData     = "data"
)

// EntityType is the sync entity every plugin record operation carries.
const EntityType = "plugin-record"

// RecordSet is one declared list of records inside a plugin's own storage.
type RecordSet struct {
	PluginID string
	// ID names the set inside the plugin and appears in the operation log, so
	// it must stay stable across versions.
	ID string
	// Storage is StorageSettings (an array under Key in settings.json) or
	// StorageData (a named NDJSON file).
	Storage  string
	Key      string
	Name     string
	Identity string
}

// Validate reports why a declared set cannot be carried, if it cannot.
func (set RecordSet) Validate() error {
	if strings.TrimSpace(set.PluginID) == "" {
		return fmt.Errorf("record set has no plugin")
	}
	if strings.TrimSpace(set.ID) == "" {
		return fmt.Errorf("%s: record set has no id", set.PluginID)
	}
	if strings.ContainsAny(set.PluginID+set.ID, "|") {
		return fmt.Errorf("%s/%s: plugin and set ids may not contain '|'", set.PluginID, set.ID)
	}
	if strings.TrimSpace(set.Identity) == "" {
		return fmt.Errorf("%s/%s: record set has no identity field", set.PluginID, set.ID)
	}
	switch set.Storage {
	case StorageSettings:
		if strings.TrimSpace(set.Key) == "" {
			return fmt.Errorf("%s/%s: a settings record set needs a key", set.PluginID, set.ID)
		}
	case StorageData:
		if strings.TrimSpace(set.Name) == "" {
			return fmt.Errorf("%s/%s: a data record set needs a name", set.PluginID, set.ID)
		}
	default:
		return fmt.Errorf("%s/%s: unknown record storage %q", set.PluginID, set.ID, set.Storage)
	}
	return nil
}

// Change is one record's fate between two versions of a set.
type Change struct {
	RecordID string
	OpType   string
	Record   map[string]interface{}
}

// Payload travels in the operation. It names the set as well as the record so a
// device that has never loaded the plugin can still store what it is told.
type Payload struct {
	PluginID string                 `json:"pluginId"`
	Document string                 `json:"document"`
	RecordID string                 `json:"recordId"`
	Record   map[string]interface{} `json:"record,omitempty"`
}

// EntityID identifies one record in the operation log. It is not a path and
// must not look like one: the pull side compares entity ids against vault paths
// to find conflicts, and a plugin id with a slash in it would collide with a
// folder of the same name.
func EntityID(pluginID, documentID, recordID string) string {
	return pluginID + "|" + documentID + "|" + recordID
}

// ParseEntityID splits an entity id back into its three parts. The record id is
// last so it may contain anything, including the separator.
func ParseEntityID(entityID string) (pluginID, documentID, recordID string, ok bool) {
	parts := strings.SplitN(entityID, "|", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", false
	}
	return parts[0], parts[1], parts[2], true
}

// RecordID reads a record's identity as a string. Anything without one cannot
// be carried, because there would be nothing to merge it against.
func RecordID(record map[string]interface{}, identity string) string {
	if record == nil {
		return ""
	}
	value, present := record[identity]
	if !present || value == nil {
		return ""
	}
	if text, isText := value.(string); isText {
		return strings.TrimSpace(text)
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

// Diff reports what changed between two versions of a set, in a stable order:
// the order of `next` for records that survive, then deletions in the order
// they had in `previous`. Records without an identity are ignored rather than
// rejected -- one malformed row must not stop a plugin from saving.
func Diff(previous, next []map[string]interface{}, identity string) []Change {
	previousByID := indexByID(previous, identity)
	nextByID := indexByID(next, identity)
	changes := make([]Change, 0)
	for _, record := range next {
		recordID := RecordID(record, identity)
		if recordID == "" {
			continue
		}
		before, existed := previousByID[recordID]
		if existed && sameRecord(before, record) {
			continue
		}
		opType := "update"
		if !existed {
			opType = "create"
		}
		changes = append(changes, Change{RecordID: recordID, OpType: opType, Record: record})
	}
	for _, record := range previous {
		recordID := RecordID(record, identity)
		if recordID == "" {
			continue
		}
		if _, survives := nextByID[recordID]; survives {
			continue
		}
		changes = append(changes, Change{RecordID: recordID, OpType: "delete"})
	}
	return changes
}

// Apply puts one incoming record change into a set and reports whether the set
// changed. A create for a record that is already there is an update, and a
// delete for one that is gone is nothing at all: pulling the same operation
// twice must leave the same result.
func Apply(records []map[string]interface{}, identity string, payload Payload, opType string) ([]map[string]interface{}, bool) {
	recordID := strings.TrimSpace(payload.RecordID)
	if recordID == "" {
		return records, false
	}
	if opType == "delete" {
		kept := make([]map[string]interface{}, 0, len(records))
		removed := false
		for _, record := range records {
			if RecordID(record, identity) == recordID {
				removed = true
				continue
			}
			kept = append(kept, record)
		}
		return kept, removed
	}
	if payload.Record == nil {
		return records, false
	}
	incoming := withIdentity(payload.Record, identity, recordID)
	for index, record := range records {
		if RecordID(record, identity) != recordID {
			continue
		}
		if sameRecord(record, incoming) {
			return records, false
		}
		updated := make([]map[string]interface{}, len(records))
		copy(updated, records)
		updated[index] = incoming
		return updated, true
	}
	return append(append(make([]map[string]interface{}, 0, len(records)+1), records...), incoming), true
}

func withIdentity(record map[string]interface{}, identity, recordID string) map[string]interface{} {
	if RecordID(record, identity) == recordID {
		return record
	}
	copied := make(map[string]interface{}, len(record)+1)
	for key, value := range record {
		copied[key] = value
	}
	copied[identity] = recordID
	return copied
}

func indexByID(records []map[string]interface{}, identity string) map[string]map[string]interface{} {
	index := make(map[string]map[string]interface{}, len(records))
	for _, record := range records {
		recordID := RecordID(record, identity)
		if recordID == "" {
			continue
		}
		index[recordID] = record
	}
	return index
}

func sameRecord(left, right map[string]interface{}) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return string(leftJSON) == string(rightJSON)
}
