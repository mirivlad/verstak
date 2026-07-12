package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAppendPluginDataNDJSONCompactsAndDeduplicates(t *testing.T) {
	s, vaultDir := newTestStorage(t)
	retention := NDJSONRetention{
		TimestampField:   "occurredAt",
		MaxAge:           60 * 24 * time.Hour,
		MaxEntries:       2,
		MaxBytes:         16 * 1024,
		DeduplicateField: "sourceBatchId",
		DeduplicateValue: "batch-1",
	}
	old := time.Now().UTC().Add(-61 * 24 * time.Hour).Format(time.RFC3339)
	now := time.Now().UTC().Format(time.RFC3339)

	stored, err := s.AppendPluginDataNDJSON("verstak.activity", "activity-events", []map[string]interface{}{
		{"activityId": "old", "sourceBatchId": "old-batch", "occurredAt": old},
	}, retention)
	if err != nil || !stored {
		t.Fatalf("old append = (%v, %v), want (true, nil)", stored, err)
	}

	stored, err = s.AppendPluginDataNDJSON("verstak.activity", "activity-events", []map[string]interface{}{
		{"activityId": "current-1", "sourceBatchId": "batch-1", "occurredAt": now},
	}, retention)
	if err != nil || !stored {
		t.Fatalf("first current append = (%v, %v), want (true, nil)", stored, err)
	}

	stored, err = s.AppendPluginDataNDJSON("verstak.activity", "activity-events", []map[string]interface{}{
		{"activityId": "duplicate", "sourceBatchId": "batch-1", "occurredAt": now},
	}, retention)
	if err != nil || stored {
		t.Fatalf("duplicate append = (%v, %v), want (false, nil)", stored, err)
	}

	for _, item := range []struct {
		id    string
		batch string
	}{
		{id: "current-2", batch: "batch-3"},
		{id: "current-3", batch: "batch-4"},
	} {
		retention.DeduplicateValue = item.batch
		if stored, err := s.AppendPluginDataNDJSON("verstak.activity", "activity-events", []map[string]interface{}{
			{"activityId": item.id, "sourceBatchId": item.batch, "occurredAt": now},
		}, retention); err != nil || !stored {
			t.Fatalf("append %s = (%v, %v), want (true, nil)", item.id, stored, err)
		}
	}

	records, err := s.ReadPluginDataNDJSON("verstak.activity", "activity-events")
	if err != nil {
		t.Fatalf("ReadPluginDataNDJSON: %v", err)
	}
	if len(records) != 2 || records[0]["activityId"] != "current-2" || records[1]["activityId"] != "current-3" {
		t.Fatalf("records = %+v, want current-2 and current-3", records)
	}
	if _, err := os.Stat(filepath.Join(vaultDir, "VerstakVault", ".verstak", "plugin-data", "verstak.activity", "activity-events.ndjson")); err != nil {
		t.Fatalf("activity data file missing: %v", err)
	}
}

func TestWritePluginDataNDJSONReplacesRecordsForExplicitUserClear(t *testing.T) {
	s, _ := newTestStorage(t)
	if _, err := s.AppendPluginDataNDJSON("verstak.activity", "activity-events", []map[string]interface{}{
		{"activityId": "one", "occurredAt": time.Now().UTC().Format(time.RFC3339)},
		{"activityId": "two", "occurredAt": time.Now().UTC().Format(time.RFC3339)},
	}, NDJSONRetention{}); err != nil {
		t.Fatalf("AppendPluginDataNDJSON: %v", err)
	}
	if err := s.WritePluginDataNDJSON("verstak.activity", "activity-events", []map[string]interface{}{
		{"activityId": "two", "occurredAt": time.Now().UTC().Format(time.RFC3339)},
	}); err != nil {
		t.Fatalf("WritePluginDataNDJSON: %v", err)
	}
	records, err := s.ReadPluginDataNDJSON("verstak.activity", "activity-events")
	if err != nil {
		t.Fatalf("ReadPluginDataNDJSON: %v", err)
	}
	if len(records) != 1 || records[0]["activityId"] != "two" {
		t.Fatalf("records = %+v, want only two", records)
	}
}
