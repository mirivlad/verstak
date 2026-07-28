package pluginsync

import "testing"

func record(values map[string]interface{}) map[string]interface{} {
	return values
}

func TestDiffReportsCreatesUpdatesAndDeletes(t *testing.T) {
	previous := []map[string]interface{}{
		record(map[string]interface{}{"id": "a", "title": "one"}),
		record(map[string]interface{}{"id": "b", "title": "two"}),
	}
	next := []map[string]interface{}{
		record(map[string]interface{}{"id": "b", "title": "two changed"}),
		record(map[string]interface{}{"id": "c", "title": "three"}),
	}

	changes := Diff(previous, next, "id")
	if len(changes) != 3 {
		t.Fatalf("changes = %#v", changes)
	}
	if changes[0].RecordID != "b" || changes[0].OpType != "update" {
		t.Fatalf("first change = %#v", changes[0])
	}
	if changes[1].RecordID != "c" || changes[1].OpType != "create" {
		t.Fatalf("second change = %#v", changes[1])
	}
	if changes[2].RecordID != "a" || changes[2].OpType != "delete" {
		t.Fatalf("third change = %#v", changes[2])
	}
}

// A save that changed nothing must record nothing, or every mount of a plugin
// that rewrites its own file would push the whole list again.
func TestDiffIgnoresUnchangedRecords(t *testing.T) {
	same := []map[string]interface{}{record(map[string]interface{}{"id": "a", "title": "one"})}
	if changes := Diff(same, same, "id"); len(changes) != 0 {
		t.Fatalf("changes = %#v", changes)
	}
}

func TestDiffSkipsRecordsWithoutIdentity(t *testing.T) {
	previous := []map[string]interface{}{record(map[string]interface{}{"title": "no id"})}
	next := []map[string]interface{}{
		record(map[string]interface{}{"title": "still no id"}),
		record(map[string]interface{}{"id": "a"}),
	}
	changes := Diff(previous, next, "id")
	if len(changes) != 1 || changes[0].RecordID != "a" || changes[0].OpType != "create" {
		t.Fatalf("changes = %#v", changes)
	}
}

func TestApplyInsertsUpdatesAndRemoves(t *testing.T) {
	records := []map[string]interface{}{record(map[string]interface{}{"id": "a", "title": "one"})}

	records, changed := Apply(records, "id", Payload{RecordID: "b", Record: map[string]interface{}{"id": "b", "title": "two"}}, "create")
	if !changed || len(records) != 2 {
		t.Fatalf("insert: changed=%v records=%#v", changed, records)
	}
	records, changed = Apply(records, "id", Payload{RecordID: "a", Record: map[string]interface{}{"id": "a", "title": "one changed"}}, "update")
	if !changed || records[0]["title"] != "one changed" {
		t.Fatalf("update: changed=%v records=%#v", changed, records)
	}
	records, changed = Apply(records, "id", Payload{RecordID: "b"}, "delete")
	if !changed || len(records) != 1 {
		t.Fatalf("delete: changed=%v records=%#v", changed, records)
	}
}

// Pulling the same operation twice has to leave the same result, because a
// device that failed mid-apply will see it again.
func TestApplyIsIdempotent(t *testing.T) {
	records := []map[string]interface{}{record(map[string]interface{}{"id": "a", "title": "one"})}
	payload := Payload{RecordID: "a", Record: map[string]interface{}{"id": "a", "title": "one"}}

	if _, changed := Apply(records, "id", payload, "create"); changed {
		t.Fatal("re-creating an identical record reported a change")
	}
	after, changed := Apply(records, "id", Payload{RecordID: "missing"}, "delete")
	if changed || len(after) != 1 {
		t.Fatalf("deleting what is not there changed something: %#v", after)
	}
}

func TestEntityIDRoundTripsRecordIDsContainingSeparators(t *testing.T) {
	entityID := EntityID("verstak.todo", "todos", "todo|Project|1")
	pluginID, documentID, recordID, ok := ParseEntityID(entityID)
	if !ok || pluginID != "verstak.todo" || documentID != "todos" || recordID != "todo|Project|1" {
		t.Fatalf("parsed %q -> %q %q %q ok=%v", entityID, pluginID, documentID, recordID, ok)
	}
	if _, _, _, ok := ParseEntityID("Notes/one.md"); ok {
		t.Fatal("a vault path parsed as a plugin record identity")
	}
}

func TestValidateRejectsSetsThatCannotBeCarried(t *testing.T) {
	valid := RecordSet{PluginID: "verstak.todo", ID: "todos", Storage: StorageSettings, Key: "todos:global", Identity: "id"}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid set rejected: %v", err)
	}
	for name, set := range map[string]RecordSet{
		"no identity":  {PluginID: "p", ID: "d", Storage: StorageSettings, Key: "k"},
		"no key":       {PluginID: "p", ID: "d", Storage: StorageSettings, Identity: "id"},
		"no name":      {PluginID: "p", ID: "d", Storage: StorageData, Identity: "id"},
		"bad storage":  {PluginID: "p", ID: "d", Storage: "cache", Identity: "id"},
		"pipe in id":   {PluginID: "p|q", ID: "d", Storage: StorageData, Name: "n", Identity: "id"},
		"no set id":    {PluginID: "p", Storage: StorageData, Name: "n", Identity: "id"},
		"no plugin id": {ID: "d", Storage: StorageData, Name: "n", Identity: "id"},
	} {
		if err := set.Validate(); err == nil {
			t.Fatalf("%s was accepted", name)
		}
	}
}
