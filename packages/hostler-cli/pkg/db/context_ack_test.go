package db

import (
	"strings"
	"testing"
)

// Context ack unit tests.

func TestGenerateWorkTicket(t *testing.T) {
	ticket, err := GenerateWorkTicket("T100")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(ticket, "WT-T100-") {
		t.Errorf("ticket=%q has wrong prefix", ticket)
	}
	// 8 hex chars
	suffix := strings.TrimPrefix(ticket, "WT-T100-")
	if len(suffix) != 8 {
		t.Errorf("suffix=%q expected length 8", suffix)
	}
	// Two tickets must be unique.
	t2, _ := GenerateWorkTicket("T100")
	if ticket == t2 {
		t.Errorf("duplicate ticket generated")
	}
}

func TestContextAckInsertAndQuery(t *testing.T) {
	setupTestDB(t)

	ticket := "WT-T999-testcase"
	hash := "abc123"
	if err := InsertContextAck(_db, ticket, hash, 100); err != nil {
		t.Fatal(err)
	}
	// Duplicate INSERT is idempotent.
	if err := InsertContextAck(_db, ticket, hash, 100); err != nil {
		t.Fatal(err)
	}

	ok, err := HasContextAck(_db, ticket, hash)
	if err != nil || !ok {
		t.Errorf("HasContextAck=%v err=%v", ok, err)
	}

	ok, _ = HasContextAck(_db, ticket, "wrong_hash")
	if ok {
		t.Errorf("wrong hash matched")
	}

	cnt, _ := CountContextAcks(_db, ticket)
	if cnt != 1 {
		t.Errorf("count after duplicate INSERT=%d, want 1", cnt)
	}
}

// setupTestDB initialises a temporary DB for tests.
func setupTestDB(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HSTL_DB_PATH", tmpDir+"/test.db")
	if err := InitDB(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { Close() })
}
