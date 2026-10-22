package storage

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestUpsertPluginSettingRecordPreservesNamespaceAndBoundsList(t *testing.T) {
	s, _ := newTestStorage(t)
	if err := s.WritePluginSettings("records", map[string]interface{}{
		"other": "keep", "entries": []map[string]interface{}{{"id": "a"}, {"id": "b"}, {"id": "c"}},
	}); err != nil {
		t.Fatal(err)
	}
	for _, record := range []map[string]interface{}{{"id": "b", "text": "updated"}, {"id": "d"}} {
		if err := s.UpsertPluginSettingRecord("records", "entries", "id", record, 3); err != nil {
			t.Fatal(err)
		}
	}
	settings, err := s.ReadPluginSettings("records")
	if err != nil {
		t.Fatal(err)
	}
	want := []interface{}{map[string]interface{}{"id": "d"}, map[string]interface{}{"id": "b", "text": "updated"}, map[string]interface{}{"id": "a"}}
	if settings["other"] != "keep" || !reflect.DeepEqual(settings["entries"], want) {
		t.Fatalf("settings = %#v", settings)
	}
	if err := s.UpsertPluginSettingRecord("other-plugin", "entries", "id", map[string]interface{}{"id": "d", "text": "separate"}, 3); err != nil {
		t.Fatal(err)
	}
	unchanged, err := s.ReadPluginSettings("records")
	if err != nil || !reflect.DeepEqual(unchanged, settings) {
		t.Fatalf("another namespace changed: %#v, %v", unchanged, err)
	}
}

func TestUpsertPluginSettingRecordRejectsInvalidInputWithoutWriting(t *testing.T) {
	for _, test := range []struct {
		name, pluginID, key, identity string
		record                        map[string]interface{}
		limit                         int
		stored                        interface{}
	}{
		{"plugin", "../escape", "entries", "id", map[string]interface{}{"id": "a"}, 100, nil},
		{"key", "records", "", "id", map[string]interface{}{"id": "a"}, 100, nil},
		{"identity", "records", "entries", "", map[string]interface{}{"id": "a"}, 100, nil},
		{"missing-id", "records", "entries", "id", map[string]interface{}{}, 100, nil},
		{"blank-id", "records", "entries", "id", map[string]interface{}{"id": " "}, 100, nil},
		{"numeric-id", "records", "entries", "id", map[string]interface{}{"id": 1}, 100, nil},
		{"zero-limit", "records", "entries", "id", map[string]interface{}{"id": "a"}, 0, nil},
		{"large-limit", "records", "entries", "id", map[string]interface{}{"id": "a"}, 10001, nil},
		{"not-an-array", "records", "entries", "id", map[string]interface{}{"id": "a"}, 100, "keep"},
		{"not-a-record", "records", "entries", "id", map[string]interface{}{"id": "a"}, 100, []interface{}{"keep"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			s, _ := newTestStorage(t)
			if err := s.WritePluginSettings("records", map[string]interface{}{"entries": test.stored, "other": "keep"}); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(s.vault.GetVaultPath(), ".verstak", "plugin-settings", "records", "settings.json")
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := s.UpsertPluginSettingRecord(test.pluginID, test.key, test.identity, test.record, test.limit); err == nil {
				t.Fatal("expected an error")
			}
			after, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatalf("failed operation changed settings: %v", err)
			}
		})
	}
}
