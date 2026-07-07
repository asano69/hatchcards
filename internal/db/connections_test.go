package db

import "testing"

// TestGetMirrorConnectionReturnsEnabled verifies that GetMirrorConnection
// surfaces the "enabled" field, since the sync API relies on it to refuse
// syncing a disabled connection.
func TestGetMirrorConnectionReturnsEnabled(t *testing.T) {
	database, err := OpenScratch()
	if err != nil {
		t.Fatalf("OpenScratch: %v", err)
	}
	defer database.Close()

	record, err := database.CreateConnection("", ConnectionInput{
		Name:      "test",
		RemoteURL: "https://example.com/repo.git",
		Enabled:   false,
	})
	if err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	mc, err := database.GetMirrorConnection(record.Id)
	if err != nil {
		t.Fatalf("GetMirrorConnection: %v", err)
	}
	if mc.Enabled {
		t.Error("Enabled = true, want false")
	}
}
