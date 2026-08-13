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
	followUpID, err := store.createFollowUp(client.ID, client.Name, client.Contact, "2026-08-12", "2026-08-13", "Enviar retorno", "Equipe", PriorityHigh, "Observação", "")
	if err != nil {
		t.Fatal(err)
	}

	clients, err := store.searchClients("berta")
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

func TestNormalizePhone(t *testing.T) {
	valid := map[string]string{
		"32999991234":     "(32) 99999-1234",
		"(32) 99999-1234": "(32) 99999-1234",
		" 32999991234 ":   "(32) 99999-1234",
	}
	for input, want := range valid {
		got, err := normalizePhone(input)
		if err != nil || got != want {
			t.Errorf("normalizePhone(%q) = %q, %v; want %q", input, got, err, want)
		}
	}

	invalid := []string{
		"",
		"3299999123",
		"329999912345",
		"32A999991234",
		"(32) 9999-1234",
		"32.99999.1234",
		"cliente@example.com",
	}
	for _, input := range invalid {
		if got, err := normalizePhone(input); err == nil {
			t.Errorf("normalizePhone(%q) = %q, want error", input, got)
		}
	}
}

func TestClientPhoneIsRequired(t *testing.T) {
	store, _, _ := newTestStore(t)
	for _, contact := range []string{"", "3299999123", "329999912345", "phone32999991234"} {
		if _, err := store.createClient("Cliente", contact); err == nil {
			t.Errorf("createClient contact %q succeeded, want error", contact)
		}
	}

	client, err := store.createClient("Cliente", "32999991234")
	if err != nil {
		t.Fatal(err)
	}
	if client.Contact != "(32) 99999-1234" {
		t.Fatalf("stored phone = %q, want formatted phone", client.Contact)
	}
	if err := store.updateClient(client.ID, client.Name, ""); err == nil {
		t.Fatal("updateClient accepted an empty phone")
	}
}

func TestSearchClientsUsesNameOnly(t *testing.T) {
	store, _, _ := newTestStore(t)
	client, err := store.createClient("Ana Silva", "(32) 99999-4321")
	if err != nil {
		t.Fatal(err)
	}

	clients, err := store.searchClients("Ana S")
	if err != nil || len(clients) != 1 || clients[0].ID != client.ID {
		t.Fatalf("name search = %#v, %v; want client %d", clients, err, client.ID)
	}
	clients, err = store.searchClients("99999")
	if err != nil {
		t.Fatal(err)
	}
	if len(clients) != 0 {
		t.Fatalf("phone search returned %#v, want no clients", clients)
	}
}

func TestFindClientsByExactNameDoesNotMatchPartialNames(t *testing.T) {
	store, _, _ := newTestStore(t)
	first, err := store.createClient("Ana Silva", "(32) 99999-1111")
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.createClient("ana silva", "(32) 98888-2222")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.createClient("Ana Souza", "(32) 97777-3333"); err != nil {
		t.Fatal(err)
	}

	matches, err := store.findClientsByExactName("  ANA SILVA  ")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 2 || matches[0].ID != first.ID || matches[1].ID != second.ID {
		t.Fatalf("exact matches = %#v; want clients %d and %d", matches, first.ID, second.ID)
	}
	matches, err = store.findClientsByExactName("Ana")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("partial name returned exact matches: %#v", matches)
	}
}

func TestSelectedClientPhoneResolution(t *testing.T) {
	t.Run("unchanged phone reuses ID", func(t *testing.T) {
		store, _, _ := newTestStore(t)
		client, err := store.createClient("Ana Silva", "(32) 99999-1111")
		if err != nil {
			t.Fatal(err)
		}
		followUpID, err := store.createFollowUp(
			client.ID, client.Name, "32999991111", "2026-08-12", "2026-08-13",
			"Pendência", "", PriorityMedium, "", "",
		)
		if err != nil {
			t.Fatal(err)
		}
		if got := followUpClientID(t, store.db, followUpID); got != client.ID {
			t.Fatalf("follow-up client ID = %d, want %d", got, client.ID)
		}
	})

	t.Run("different phone requires explicit decision", func(t *testing.T) {
		store, _, _ := newTestStore(t)
		client, err := store.createClient("Ana Silva", "(32) 99999-1111")
		if err != nil {
			t.Fatal(err)
		}
		_, err = store.createFollowUp(
			client.ID, client.Name, "(32) 98888-2222", "2026-08-12", "2026-08-13",
			"Pendência", "", PriorityMedium, "", "",
		)
		var confirmation *clientPhoneChangeRequiredError
		if !errors.As(err, &confirmation) {
			t.Fatalf("createFollowUp error = %v, want phone confirmation", err)
		}
		assertClientPhone(t, store, client.ID, "(32) 99999-1111")
		assertTableCount(t, store.db, "followups", 0)
	})

	t.Run("update decision keeps ID", func(t *testing.T) {
		store, _, _ := newTestStore(t)
		client, err := store.createClient("Ana Silva", "(32) 99999-1111")
		if err != nil {
			t.Fatal(err)
		}
		followUpID, err := store.createFollowUp(
			client.ID, client.Name, "(32) 98888-2222", "2026-08-12", "2026-08-13",
			"Pendência", "", PriorityMedium, "", PhoneChangeUpdate,
		)
		if err != nil {
			t.Fatal(err)
		}
		if got := followUpClientID(t, store.db, followUpID); got != client.ID {
			t.Fatalf("follow-up client ID = %d, want %d", got, client.ID)
		}
		assertClientPhone(t, store, client.ID, "(32) 98888-2222")
	})

	t.Run("new client decision preserves original", func(t *testing.T) {
		store, _, _ := newTestStore(t)
		client, err := store.createClient("Ana Silva", "(32) 99999-1111")
		if err != nil {
			t.Fatal(err)
		}
		followUpID, err := store.createFollowUp(
			client.ID, client.Name, "(32) 98888-2222", "2026-08-12", "2026-08-13",
			"Pendência", "", PriorityMedium, "", PhoneChangeNewClient,
		)
		if err != nil {
			t.Fatal(err)
		}
		newClientID := followUpClientID(t, store.db, followUpID)
		if newClientID == client.ID {
			t.Fatalf("new homonymous client reused ID %d", client.ID)
		}
		assertClientPhone(t, store, client.ID, "(32) 99999-1111")
		assertClientPhone(t, store, newClientID, "(32) 98888-2222")
	})
}

func TestCreateFollowUpPreservesHomonymousClientIdentity(t *testing.T) {
	store, _, _ := newTestStore(t)

	firstFollowUpID, err := store.createFollowUp(
		0, "Ana Silva", "(32) 99999-1111", "2026-08-12", "2026-08-13",
		"Pendência da primeira Ana", "", PriorityMedium, "", "",
	)
	if err != nil {
		t.Fatal(err)
	}
	firstClientID := followUpClientID(t, store.db, firstFollowUpID)

	secondFollowUpID, err := store.createFollowUp(
		0, "Ana Silva", "(32) 98888-2222", "2026-08-12", "2026-08-14",
		"Pendência da segunda Ana", "", PriorityMedium, "", "",
	)
	if err != nil {
		t.Fatal(err)
	}
	secondClientID := followUpClientID(t, store.db, secondFollowUpID)
	if firstClientID == secondClientID {
		t.Fatalf("homonymous clients share ID %d; want distinct IDs", firstClientID)
	}

	firstClient, err := store.getClient(firstClientID)
	if err != nil {
		t.Fatal(err)
	}
	if firstClient.Contact != "(32) 99999-1111" {
		t.Fatalf("first client contact = %q, want original contact", firstClient.Contact)
	}

	reusedFollowUpID, err := store.createFollowUp(
		firstClientID, "Ana Silva", firstClient.Contact, "2026-08-12", "2026-08-15",
		"Nova pendência da primeira Ana", "", PriorityHigh, "", "",
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := followUpClientID(t, store.db, reusedFollowUpID); got != firstClientID {
		t.Fatalf("explicitly selected client ID = %d, want %d", got, firstClientID)
	}

	var clientCount int
	if err := store.db.QueryRow("SELECT COUNT(*) FROM clients WHERE name = ?", "Ana Silva").Scan(&clientCount); err != nil {
		t.Fatal(err)
	}
	if clientCount != 2 {
		t.Fatalf("homonymous client count = %d, want 2", clientCount)
	}
}

func TestFollowUpStateTransitionsAndTimestamps(t *testing.T) {
	store, _, fixedNow := newTestStore(t)
	client, err := store.createClient("Cliente", "(32) 99999-0001")
	if err != nil {
		t.Fatal(err)
	}
	id, err := store.createFollowUp(client.ID, client.Name, client.Contact, "2026-08-12", "2026-08-13", "Pendência", "", PriorityMedium, "", "")
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
	client, _ := store.createClient("Cliente", "(32) 99999-0002")
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
		if _, err := store.createFollowUp(client.ID, client.Name, client.Contact, "2026-08-01", test.due, test.desc, "", test.priority, "", ""); err != nil {
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
	client, _ := store.createClient("Cliente preservada", "(32) 99999-0003")
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

func followUpClientID(t *testing.T, database *sql.DB, followUpID int64) int64 {
	t.Helper()
	var clientID int64
	if err := database.QueryRow("SELECT client_id FROM followups WHERE id = ?", followUpID).Scan(&clientID); err != nil {
		t.Fatal(err)
	}
	return clientID
}

func assertClientPhone(t *testing.T, store *Store, clientID int64, want string) {
	t.Helper()
	client, err := store.getClient(clientID)
	if err != nil {
		t.Fatal(err)
	}
	if client.Contact != want {
		t.Fatalf("client %d phone = %q, want %q", clientID, client.Contact, want)
	}
}

func assertTableCount(t *testing.T, database *sql.DB, table string, want int) {
	t.Helper()
	var got int
	if err := database.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("%s count = %d, want %d", table, got, want)
	}
}

func insertLegacyClient(t *testing.T, store *Store, name, contact string) Client {
	t.Helper()
	now := store.now()
	result, err := store.db.Exec(
		"INSERT INTO clients (name, contact, created_at, updated_at) VALUES (?, ?, ?, ?)",
		name, contact, now, now,
	)
	if err != nil {
		t.Fatal(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	return Client{ID: id, Name: name, Contact: contact, CreatedAt: now, UpdatedAt: now}
}
