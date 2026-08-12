package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newTestStore(t *testing.T) (*Store, string, time.Time) {
	t.Helper()
	fixedNow := time.Date(2026, time.August, 12, 10, 30, 0, 0, mustLocation(t))
	databasePath := filepath.Join(t.TempDir(), "client-followup.db")
	store, err := openStore(databasePath, func() time.Time { return fixedNow })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	return store, databasePath, fixedNow
}

func TestDatabaseCreationAndPersistence(t *testing.T) {
	store, databasePath, fixedNow := newTestStore(t)

	var clientsTable string
	if err := store.db.QueryRow("SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'clients'").Scan(&clientsTable); err != nil {
		t.Fatalf("schema was not created: %v", err)
	}
	client, err := store.createClient("Roberta", "(32) 99999-0000")
	if err != nil {
		t.Fatal(err)
	}
	followUpID, err := store.createFollowUp(client.ID, client.Name, client.Contact, "2026-08-12", "2026-08-13", "Enviar retorno", "Equipe", PriorityHigh, "Observação")
	if err != nil {
		t.Fatal(err)
	}

	clients, err := store.searchClients("99999")
	if err != nil || len(clients) != 1 || clients[0].ID != client.ID {
		t.Fatalf("partial client search = %#v, %v", clients, err)
	}
	items, err := store.dashboardFollowUps(DashboardFilters{Query: "retor"}, "2026-08-12")
	if err != nil || len(items) != 1 || items[0].ID != followUpID {
		t.Fatalf("partial follow-up search = %#v, %v", items, err)
	}

	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := openStore(databasePath, func() time.Time { return fixedNow })
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	var count int
	if err := reopened.db.QueryRow("SELECT COUNT(*) FROM followups WHERE id = ?", followUpID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("persisted follow-up count = %d, %v", count, err)
	}
}

func TestFollowUpStateTransitionsAndTimestamps(t *testing.T) {
	store, _, fixedNow := newTestStore(t)
	client, err := store.createClient("Cliente", "")
	if err != nil {
		t.Fatal(err)
	}
	id, err := store.createFollowUp(client.ID, client.Name, "", "2026-08-12", "2026-08-13", "Pendência", "", PriorityMedium, "")
	if err != nil {
		t.Fatal(err)
	}

	if err := store.transitionFollowUp(id, StatusPending, StatusArchived); !errors.Is(err, errInvalidTransition) {
		t.Fatalf("invalid archive error = %v", err)
	}
	if err := store.transitionFollowUp(id, StatusPending, StatusCompleted); err != nil {
		t.Fatal(err)
	}
	status, completedAt, archivedAt := followUpState(t, store.db, id)
	if status != StatusCompleted || !completedAt.Valid || !completedAt.Time.Equal(fixedNow) || archivedAt.Valid {
		t.Fatalf("completed state = %s, %#v, %#v", status, completedAt, archivedAt)
	}

	if err := store.transitionFollowUp(id, StatusCompleted, StatusPending); err != nil {
		t.Fatal(err)
	}
	status, completedAt, archivedAt = followUpState(t, store.db, id)
	if status != StatusPending || completedAt.Valid || archivedAt.Valid {
		t.Fatalf("reopened state = %s, %#v, %#v", status, completedAt, archivedAt)
	}

	if err := store.transitionFollowUp(id, StatusPending, StatusCompleted); err != nil {
		t.Fatal(err)
	}
	if err := store.transitionFollowUp(id, StatusCompleted, StatusArchived); err != nil {
		t.Fatal(err)
	}
	status, completedAt, archivedAt = followUpState(t, store.db, id)
	if status != StatusArchived || !completedAt.Valid || !archivedAt.Valid || !archivedAt.Time.Equal(fixedNow) {
		t.Fatalf("archived state = %s, %#v, %#v", status, completedAt, archivedAt)
	}
}

func TestOperationalOrdering(t *testing.T) {
	store, _, _ := newTestStore(t)
	client, _ := store.createClient("Cliente", "")
	tests := []struct {
		due      string
		priority string
		desc     string
	}{
		{due: "2026-08-14", priority: PriorityHigh, desc: "future"},
		{due: "2026-08-10", priority: PriorityLow, desc: "overdue low"},
		{due: "2026-08-10", priority: PriorityHigh, desc: "overdue high"},
		{due: "2026-08-13", priority: PriorityLow, desc: "near"},
	}
	for _, test := range tests {
		if _, err := store.createFollowUp(client.ID, client.Name, "", "2026-08-01", test.due, test.desc, "", test.priority, ""); err != nil {
			t.Fatal(err)
		}
	}
	items, err := store.dashboardFollowUps(DashboardFilters{}, "2026-08-12")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"overdue high", "overdue low", "near", "future"}
	for index, description := range want {
		if items[index].Description != description {
			t.Fatalf("posição %d = %q, want %q", index, items[index].Description, description)
		}
	}
}

func TestDailyBackupAndRetention(t *testing.T) {
	store, _, _ := newTestStore(t)
	client, _ := store.createClient("Cliente preservada", "")
	backupDirectory := filepath.Join(t.TempDir(), "backups")
	today := time.Date(2026, time.August, 12, 0, 0, 0, 0, mustLocation(t))
	backupPath, err := store.createDailyBackup(backupDirectory, today)
	if err != nil {
		t.Fatal(err)
	}
	backupDB, err := sql.Open("sqlite3", backupPath)
	if err != nil {
		t.Fatal(err)
	}
	defer backupDB.Close()
	var name string
	if err := backupDB.QueryRow("SELECT name FROM clients WHERE id = ?", client.ID).Scan(&name); err != nil || name != "Cliente preservada" {
		t.Fatalf("backup content = %q, %v", name, err)
	}

	for day := 1; day <= 16; day++ {
		path := filepath.Join(backupDirectory, fmt.Sprintf("client-followup-2026-07-%02d.db", day))
		if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := removeOldBackups(backupDirectory, backupRetention); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(backupDirectory)
	if len(entries) != backupRetention {
		t.Fatalf("backup count = %d, want %d", len(entries), backupRetention)
	}
}

func followUpState(t *testing.T, database *sql.DB, id int64) (string, sql.NullTime, sql.NullTime) {
	t.Helper()
	var status string
	var completedAt, archivedAt sql.NullTime
	if err := database.QueryRow("SELECT status, completed_at, archived_at FROM followups WHERE id = ?", id).Scan(&status, &completedAt, &archivedAt); err != nil {
		t.Fatal(err)
	}
	return status, completedAt, archivedAt
}
