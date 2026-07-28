package api

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/verstak/verstak-desktop/internal/core/events"
	"github.com/verstak/verstak-desktop/internal/core/pluginsync"
	syncsvc "github.com/verstak/verstak-desktop/internal/core/sync"
)

// pluginRecordsChangedEvent tells a plugin that something it owns arrived from
// another device. Without it a synced todo appears only after the view is
// mounted again, which looks exactly like sync not working.
const pluginRecordsChangedEvent = "plugin.records.changed"

// pluginRecordSets returns the record sets a plugin declared and that are
// usable. A set declared wrongly is logged once and skipped: a plugin with a
// typo in its manifest still has to run.
func (a *App) pluginRecordSets(pluginID string) []pluginsync.RecordSet {
	if a == nil {
		return nil
	}
	for _, loaded := range a.plugins {
		if loaded.Manifest.ID != pluginID {
			continue
		}
		if loaded.Manifest.Sync == nil {
			return nil
		}
		sets := make([]pluginsync.RecordSet, 0, len(loaded.Manifest.Sync.Records))
		for _, declared := range loaded.Manifest.Sync.Records {
			set := pluginsync.RecordSet{
				PluginID: pluginID,
				ID:       declared.ID,
				Storage:  declared.Storage,
				Key:      declared.Key,
				Name:     declared.Name,
				Identity: declared.Identity,
			}
			if err := set.Validate(); err != nil {
				log.Printf("[sync] plugin record set ignored: %v", err)
				continue
			}
			sets = append(sets, set)
		}
		return sets
	}
	return nil
}

func (a *App) readPluginRecordSet(set pluginsync.RecordSet) ([]map[string]interface{}, error) {
	if a.storage == nil {
		return nil, fmt.Errorf("storage not initialized")
	}
	switch set.Storage {
	case pluginsync.StorageData:
		return a.storage.ReadPluginDataNDJSON(set.PluginID, set.Name)
	case pluginsync.StorageSettings:
		settings, err := a.storage.ReadPluginSettings(set.PluginID)
		if err != nil {
			return nil, err
		}
		return recordsFromSettingsValue(settings[set.Key]), nil
	default:
		return nil, fmt.Errorf("unknown record storage %q", set.Storage)
	}
}

func (a *App) writePluginRecordSet(set pluginsync.RecordSet, records []map[string]interface{}) error {
	if a.storage == nil {
		return fmt.Errorf("storage not initialized")
	}
	switch set.Storage {
	case pluginsync.StorageData:
		return a.storage.WritePluginDataNDJSON(set.PluginID, set.Name, records)
	case pluginsync.StorageSettings:
		return a.storage.UpdatePluginSettings(set.PluginID, func(settings map[string]interface{}) error {
			value := make([]interface{}, 0, len(records))
			for _, record := range records {
				value = append(value, record)
			}
			settings[set.Key] = value
			return nil
		})
	default:
		return fmt.Errorf("unknown record storage %q", set.Storage)
	}
}

// recordsFromSettingsValue reads a settings value that is meant to be a list of
// records. Anything else is treated as an empty list rather than an error: the
// plugin owns the value, and sync is not the place to argue about its shape.
func recordsFromSettingsValue(value interface{}) []map[string]interface{} {
	list, isList := value.([]interface{})
	if !isList {
		return []map[string]interface{}{}
	}
	records := make([]map[string]interface{}, 0, len(list))
	for _, item := range list {
		if record, isRecord := item.(map[string]interface{}); isRecord {
			records = append(records, record)
		}
	}
	return records
}

// snapshotPluginRecords reads every declared set before a plugin writes, so the
// write can be told apart from what was already there.
func (a *App) snapshotPluginRecords(pluginID string) map[string][]map[string]interface{} {
	sets := a.pluginRecordSets(pluginID)
	if len(sets) == 0 || a.syncSvc == nil {
		return nil
	}
	before := make(map[string][]map[string]interface{}, len(sets))
	for _, set := range sets {
		records, err := a.readPluginRecordSet(set)
		if err != nil {
			log.Printf("[sync] read %s/%s before write: %v", set.PluginID, set.ID, err)
			continue
		}
		before[set.ID] = records
	}
	return before
}

// recordPluginRecordChanges turns what a plugin just saved into one operation
// per changed record. A record is the unit that travels: two devices adding
// different todos then produce two records instead of two versions of one file.
func (a *App) recordPluginRecordChanges(pluginID string, before map[string][]map[string]interface{}) {
	if before == nil || a.syncSvc == nil {
		return
	}
	for _, set := range a.pluginRecordSets(pluginID) {
		previous, seen := before[set.ID]
		if !seen {
			continue
		}
		next, err := a.readPluginRecordSet(set)
		if err != nil {
			log.Printf("[sync] read %s/%s after write: %v", set.PluginID, set.ID, err)
			continue
		}
		for _, change := range pluginsync.Diff(previous, next, set.Identity) {
			payload := pluginsync.Payload{
				PluginID: set.PluginID,
				Document: set.ID,
				RecordID: change.RecordID,
				Record:   change.Record,
			}
			entityID := pluginsync.EntityID(set.PluginID, set.ID, change.RecordID)
			if err := a.syncSvc.RecordOp(pluginsync.EntityType, entityID, change.OpType, payload); err != nil {
				log.Printf("[sync] record %s %s: %v", change.OpType, entityID, err)
			}
		}
	}
}

// applyRemotePluginRecordOp puts one record from another device into the
// plugin's own storage. It writes through storage directly, never through the
// bridge that records operations, or applying a pull would push it straight
// back.
func (a *App) applyRemotePluginRecordOp(op syncsvc.Op) error {
	var payload pluginsync.Payload
	if op.PayloadJSON != "" {
		if err := json.Unmarshal([]byte(op.PayloadJSON), &payload); err != nil {
			return fmt.Errorf("plugin record payload: %w", err)
		}
	}
	pluginID, documentID, recordID, ok := pluginsync.ParseEntityID(op.EntityID)
	if !ok {
		return fmt.Errorf("plugin record identity: %s", op.EntityID)
	}
	if payload.PluginID == "" {
		payload.PluginID = pluginID
	}
	if payload.Document == "" {
		payload.Document = documentID
	}
	if payload.RecordID == "" {
		payload.RecordID = recordID
	}
	if payload.PluginID != pluginID || payload.Document != documentID || payload.RecordID != recordID {
		return fmt.Errorf("plugin record identity mismatch: entity %s payload %s/%s/%s", op.EntityID, payload.PluginID, payload.Document, payload.RecordID)
	}

	var target *pluginsync.RecordSet
	for _, set := range a.pluginRecordSets(pluginID) {
		if set.ID == documentID {
			found := set
			target = &found
			break
		}
	}
	if target == nil {
		// The plugin is not installed here, or no longer shares this set. The
		// operation stays in the log; a device that has the plugin will apply
		// it. Refusing would stop the whole pull over somebody else's plugin.
		log.Printf("[sync] plugin record %s skipped: %s does not share %s here", op.EntityID, pluginID, documentID)
		return nil
	}

	records, err := a.readPluginRecordSet(*target)
	if err != nil {
		return err
	}
	updated, changed := pluginsync.Apply(records, target.Identity, payload, op.OpType)
	if !changed {
		return nil
	}
	if err := a.writePluginRecordSet(*target, updated); err != nil {
		return err
	}
	a.publishPluginRecordsChanged(target.PluginID, target.ID)
	return nil
}

func (a *App) publishPluginRecordsChanged(pluginID, documentID string) {
	if a.eventBus == nil {
		return
	}
	a.eventBus.Publish(events.Event{
		Name:      pluginRecordsChangedEvent,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Payload: map[string]interface{}{
			"pluginId": pluginID,
			"document": documentID,
		},
	})
}
