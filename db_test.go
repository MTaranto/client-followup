package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	followUpID, err := store.createFollowUp(client.ID, client.Name, client.Contact, "2026-08-12", "2026-08-13", "Enviar retorno", "Equipe", PriorityHigh, "Observação", "", ClientResolutionExisting, "", "")
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
	if err := store.updateClient(client.ID, client.Name, "", "", "", ""); err == nil {
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
		t.Fatalf("exact matches = %#v, want [%d, %d]", matches, first.ID, second.ID)
	}
	matches, err = store.findClientsByExactName("Ana")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("partial name returned exact matches: %#v", matches)
	}
}

func TestFindClientsByExactNameSupportsLegacyWhitespace(t *testing.T) {
	store, _, _ := newTestStore(t)
	legacy := insertLegacyClient(t, store, "  Ana   Paula  ", "(32) 99999-1111")

	matches, err := store.findClientsByExactName("Ana Paula")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0].ID != legacy.ID {
		t.Fatalf("exact matches for legacy client = %#v, want client %d", matches, legacy.ID)
	}
}

func TestFindClientsByExactNameUsesInsertionOrderTiebreak(t *testing.T) {
	store, _, _ := newTestStore(t)
	first := insertLegacyClient(t, store, "Carlos Silva", "(32) 99999-1111")
	second := insertLegacyClient(t, store, "carlos silva", "(32) 98888-2222")

	matches, err := store.findClientsByExactName("Carlos Silva")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 2 || matches[0].ID != first.ID || matches[1].ID != second.ID {
		t.Fatalf("exact matches order = %#v, want [%d, %d]", matches, first.ID, second.ID)
	}
}

func TestClientNameMatchingPreservesTextAndIgnoresPortugueseDiacritics(t *testing.T) {
	store, _, _ := newTestStore(t)
	client, err := store.createClient("João d'Ávila-Souza", "(32) 99999-1111")
	if err != nil {
		t.Fatal(err)
	}
	if client.Name != "João d'Ávila-Souza" {
		t.Fatalf("stored name = %q, want original spelling", client.Name)
	}

	for _, query := range []string{"avila", "JOAO D'AVILA", "vila-souza"} {
		clients, err := store.searchClients(query)
		if err != nil || len(clients) != 1 || clients[0].ID != client.ID {
			t.Fatalf("searchClients(%q) = %#v, %v; want client %d", query, clients, err, client.ID)
		}
	}
	matches, err := store.findClientsByExactName("JOAO D'AVILA-SOUZA")
	if err != nil || len(matches) != 1 || matches[0].ID != client.ID {
		t.Fatalf("accent-insensitive exact matches = %#v, %v; want client %d", matches, err, client.ID)
	}
}

func TestValidateClientNameRejectsLeadingDigitsAndForbiddenCharacters(t *testing.T) {
	store, _, _ := newTestStore(t)
	for _, name := range []string{"123 Ana", "Ana 2", "Ana ٢", "Ana ２"} {
		if _, err := store.createClient(name, "(32) 99999-1111"); err == nil {
			t.Fatalf("createClient accepted invalid name %q", name)
		}
	}

	client, err := store.createClient("Ana Silva", "(32) 99999-1111")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.updateClient(client.ID, "Ana ٢", client.Contact, "", "", ""); err == nil {
		t.Fatal("updateClient accepted a Unicode digit")
	}
	unchanged, err := store.getClient(client.ID)
	if err != nil || unchanged.Name != client.Name {
		t.Fatalf("client changed after invalid update: %#v, %v", unchanged, err)
	}
}

func TestCreateFollowUpRequiresExplicitHomonymResolution(t *testing.T) {
	store, _, _ := newTestStore(t)
	existing, err := store.createClient("João Çosta", "(32) 99999-1111")
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.createFollowUp(
		0, "JOAO COSTA", "(32) 98888-2222", "2026-08-12", "2026-08-13",
		"Retorno ambíguo", "", PriorityMedium, "", "", ClientResolutionNew, "", "",
	)
	if !errors.Is(err, errClientResolutionRequired) {
		t.Fatalf("ambiguous creation error = %v, want explicit resolution error", err)
	}
	assertTableCount(t, store.db, "clients", 1)
	assertTableCount(t, store.db, "followups", 0)

	followUpID, err := store.createFollowUp(
		0, "JOAO COSTA", "(32) 98888-2222", "2026-08-12", "2026-08-13",
		"Retorno da homônima", "", PriorityMedium, "", "", ClientResolutionNewHomonym, "", "",
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := followUpClientID(t, store.db, followUpID); got == existing.ID {
		t.Fatalf("explicit homonym creation reused client ID %d", existing.ID)
	}
}

func TestUpdateClientPreservesIdentityAndRequiresPhoneConfirmation(t *testing.T) {
	store, _, _ := newTestStore(t)
	client, err := store.createClient("Ana Silva", "(32) 99999-1111")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.updateClient(client.ID, "Ana Souza", client.Contact, "", "", ""); err != nil {
		t.Fatal(err)
	}
	assertTableCount(t, store.db, "clients", 1)

	err = store.updateClient(client.ID, "Ana Souza", "(32) 98888-2222", "", "", "")
	var confirmation *clientEditPhoneChangeRequiredError
	if !errors.As(err, &confirmation) {
		t.Fatalf("phone update error = %v, want confirmation", err)
	}
	assertClientPhone(t, store, client.ID, "(32) 99999-1111")

	if err := store.updateClient(client.ID, "Ana Souza", "(32) 98888-2222", ClientPhoneChangeConfirmation, "", ""); err != nil {
		t.Fatal(err)
	}
	updated, err := store.getClient(client.ID)
	if err != nil || updated.Name != "Ana Souza" || updated.Contact != "(32) 98888-2222" {
		t.Fatalf("updated client = %#v, %v", updated, err)
	}
	assertTableCount(t, store.db, "clients", 1)
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
			"Pendência", "", PriorityMedium, "", "", ClientResolutionExisting, "", "",
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
			"Pendência", "", PriorityMedium, "", "", ClientResolutionExisting, "", "",
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
			"Pendência", "", PriorityMedium, "", PhoneChangeUpdate, ClientResolutionExisting, "", "",
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
			"Pendência", "", PriorityMedium, "", PhoneChangeNewClient, ClientResolutionExisting, "", "",
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
		"Pendência da primeira Ana", "", PriorityMedium, "", "", ClientResolutionNew, "", "",
	)
	if err != nil {
		t.Fatal(err)
	}
	firstClientID := followUpClientID(t, store.db, firstFollowUpID)

	secondFollowUpID, err := store.createFollowUp(
		0, "Ana Silva", "(32) 98888-2222", "2026-08-12", "2026-08-14",
		"Pendência da segunda Ana", "", PriorityMedium, "", "", ClientResolutionNewHomonym, "", "",
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
		"Nova pendência da primeira Ana", "", PriorityHigh, "", "", ClientResolutionExisting, "", "",
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

func TestDuplicatePhoneDetectionRequiresLegitimateToken(t *testing.T) {
	store, _, _ := newTestStore(t)
	existing, err := store.createClient("Carlos Silva", "(32) 99999-5555")
	if err != nil {
		t.Fatal(err)
	}

	// Tentativa de criar novo cliente com mesmo telefone sem decisão e sem token
	_, err = store.createFollowUp(
		0, "Marcos Souza", "(32) 99999-5555", "2026-08-12", "2026-08-13",
		"Retorno", "", PriorityMedium, "", "", ClientResolutionNew, "", "",
	)
	var duplicateErr *clientDuplicatePhoneRequiredError
	if !errors.As(err, &duplicateErr) {
		t.Fatalf("expected clientDuplicatePhoneRequiredError, got: %v", err)
	}
	if len(duplicateErr.ExistingClients) != 1 || duplicateErr.ExistingClients[0].ID != existing.ID {
		t.Fatalf("conflicts = %#v, want client %d", duplicateErr.ExistingClients, existing.ID)
	}
	if duplicateErr.Token == "" {
		t.Fatal("token should not be empty")
	}

	// Tentativa com allow mas token forjado/inválido
	_, err = store.createFollowUp(
		0, "Marcos Souza", "(32) 99999-5555", "2026-08-12", "2026-08-13",
		"Retorno", "", PriorityMedium, "", "", ClientResolutionNew, "allow", "token_forjado_invalido",
	)
	if !errors.As(err, &duplicateErr) {
		t.Fatalf("expected failure with forged token, got: %v", err)
	}

	// Tentativa com allow e token legítimo
	followUpID, err := store.createFollowUp(
		0, "Marcos Souza", "(32) 99999-5555", "2026-08-12", "2026-08-13",
		"Retorno confirmado", "", PriorityMedium, "", "", ClientResolutionNew, "allow", duplicateErr.Token,
	)
	if err != nil {
		t.Fatalf("creation with legitimate token failed: %v", err)
	}

	newClientID := followUpClientID(t, store.db, followUpID)
	if newClientID == existing.ID {
		t.Fatalf("expected new client ID, got existing %d", existing.ID)
	}
	assertClientPhone(t, store, existing.ID, "(32) 99999-5555")
	assertClientPhone(t, store, newClientID, "(32) 99999-5555")
}

func TestDuplicatePhoneWithMultipleExistingClients(t *testing.T) {
	store, _, _ := newTestStore(t)
	c1, err := store.createClient("Cliente Um", "(32) 98888-0000")
	if err != nil {
		t.Fatal(err)
	}
	c2 := insertLegacyClient(t, store, "Cliente Dois", "(32) 98888-0000")

	matches, err := store.findClientsByPhone("(32) 98888-0000", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 2 {
		t.Fatalf("findClientsByPhone returned %d clients, want 2", len(matches))
	}
	if matches[0].ID != c1.ID || matches[1].ID != c2.ID {
		t.Fatalf("conflicts = %#v, want [%d, %d]", matches, c1.ID, c2.ID)
	}
}

func TestUpdateClientRequiresDuplicatePhoneConfirmation(t *testing.T) {
	store, _, _ := newTestStore(t)
	c1, err := store.createClient("Cliente Um", "(32) 99999-1111")
	if err != nil {
		t.Fatal(err)
	}
	c2, err := store.createClient("Cliente Dois", "(32) 99999-2222")
	if err != nil {
		t.Fatal(err)
	}

	err = store.updateClient(c2.ID, c2.Name, c1.Contact, ClientPhoneChangeConfirmation, "", "")
	var duplicateErr *clientEditDuplicatePhoneRequiredError
	if !errors.As(err, &duplicateErr) {
		t.Fatalf("expected clientEditDuplicatePhoneRequiredError, got: %v", err)
	}
	if len(duplicateErr.ExistingClients) != 1 || duplicateErr.ExistingClients[0].ID != c1.ID {
		t.Fatalf("conflicts = %#v, want client %d", duplicateErr.ExistingClients, c1.ID)
	}

	if err := store.updateClient(c2.ID, c2.Name, c1.Contact, ClientPhoneChangeConfirmation, "allow", duplicateErr.Token); err != nil {
		t.Fatalf("updateClient with legitimate token failed: %v", err)
	}
	assertClientPhone(t, store, c1.ID, "(32) 99999-1111")
	assertClientPhone(t, store, c2.ID, "(32) 99999-1111")
}

func TestPhoneConfirmationTokenIsUnforgeableAcrossStores(t *testing.T) {
	store1, _, _ := newTestStore(t)
	store2, _, _ := newTestStore(t)

	c1, _ := store1.createClient("Cliente Um", "(32) 99999-1111")
	conflicts := []Client{c1}

	token1 := store1.phoneConfirmationToken("(32) 99999-1111", 0, conflicts)
	token2 := store2.phoneConfirmationToken("(32) 99999-1111", 0, conflicts)

	if token1 == token2 {
		t.Fatalf("tokens generated by different stores must differ due to distinct server secrets, got identical %q", token1)
	}
}

func TestExistingClientWithDuplicatePhoneRequiresBothConfirmations(t *testing.T) {
	store, _, _ := newTestStore(t)
	cA, err := store.createClient("Cliente A", "(32) 99999-1111")
	if err != nil {
		t.Fatal(err)
	}
	cB, err := store.createClient("Cliente B", "(32) 99999-2222")
	if err != nil {
		t.Fatal(err)
	}

	// 1. Tenta criar pendência para Cliente A com telefone alterado sem phone_change_action -> exige alteração PRIMEIRO
	_, err = store.createFollowUp(
		cA.ID, cA.Name, cB.Contact, "2026-08-12", "2026-08-13",
		"Pendência", "", PriorityMedium, "", "", ClientResolutionExisting, "", "",
	)
	var changeErr *clientPhoneChangeRequiredError
	if !errors.As(err, &changeErr) {
		t.Fatalf("expected clientPhoneChangeRequiredError first, got: %v", err)
	}

	// 2. Com PhoneChangeUpdate mas sem decisão de duplicidade -> exige confirmação de duplicidade no destino cA.ID
	_, err = store.createFollowUp(
		cA.ID, cA.Name, cB.Contact, "2026-08-12", "2026-08-13",
		"Pendência", "", PriorityMedium, "", PhoneChangeUpdate, ClientResolutionExisting, "", "",
	)
	var dupErrUpdate *clientDuplicatePhoneRequiredError
	if !errors.As(err, &dupErrUpdate) {
		t.Fatalf("expected clientDuplicatePhoneRequiredError for update, got: %v", err)
	}

	// 3. O token emitido para update (targetID = cA.ID) não é aceito para PhoneChangeNewClient (targetID = 0)
	_, err = store.createFollowUp(
		cA.ID, cA.Name, cB.Contact, "2026-08-12", "2026-08-13",
		"Pendência", "", PriorityMedium, "", PhoneChangeNewClient, ClientResolutionExisting, "allow", dupErrUpdate.Token,
	)
	var dupErrNew *clientDuplicatePhoneRequiredError
	if !errors.As(err, &dupErrNew) {
		t.Fatalf("expected failure when reusing update token for new_client, got: %v", err)
	}
	if dupErrNew.Token == dupErrUpdate.Token {
		t.Fatalf("tokens for update and new_client should differ, got identical %q", dupErrNew.Token)
	}

	// 4. Com PhoneChangeNewClient e token legítimo de new_client -> cria novo cadastro
	followUpNewID, err := store.createFollowUp(
		cA.ID, cA.Name, cB.Contact, "2026-08-12", "2026-08-13",
		"Pendência novo cadastro", "", PriorityMedium, "", PhoneChangeNewClient, ClientResolutionExisting, "allow", dupErrNew.Token,
	)
	if err != nil {
		t.Fatalf("createFollowUp with new_client and legit token failed: %v", err)
	}
	newClientID := followUpClientID(t, store.db, followUpNewID)
	if newClientID == cA.ID || newClientID == cB.ID {
		t.Fatalf("new client expected, got %d", newClientID)
	}
	// 5. Tentativa com PhoneChangeUpdate: como a base agora possui cB e newClientID com este telefone, o token anterior é invalidado e um novo token com os 2 conflitos é exigido
	_, err = store.createFollowUp(
		cA.ID, cA.Name, cB.Contact, "2026-08-12", "2026-08-13",
		"Pendência update", "", PriorityMedium, "", PhoneChangeUpdate, ClientResolutionExisting, "allow", dupErrUpdate.Token,
	)
	var dupErrUpdatedBase *clientDuplicatePhoneRequiredError
	if !errors.As(err, &dupErrUpdatedBase) {
		t.Fatalf("expected clientDuplicatePhoneRequiredError after base changed, got: %v", err)
	}
	if len(dupErrUpdatedBase.ExistingClients) != 2 {
		t.Fatalf("expected 2 conflicts, got: %#v", dupErrUpdatedBase.ExistingClients)
	}

	// Com o token atualizado para o conjunto de 2 conflitos -> sucesso no update de cA
	followUpUpdateID, err := store.createFollowUp(
		cA.ID, cA.Name, cB.Contact, "2026-08-12", "2026-08-13",
		"Pendência update", "", PriorityMedium, "", PhoneChangeUpdate, ClientResolutionExisting, "allow", dupErrUpdatedBase.Token,
	)
	if err != nil {
		t.Fatalf("createFollowUp with update and updated token failed: %v", err)
	}
	if followUpClientID(t, store.db, followUpUpdateID) != cA.ID {
		t.Fatalf("expected follow-up on cA (%d), got %d", cA.ID, followUpClientID(t, store.db, followUpUpdateID))
	}
	assertClientPhone(t, store, cA.ID, "(32) 99999-2222")
}

func TestFollowUpStateTransitionsAndTimestamps(t *testing.T) {
	store, _, fixedNow := newTestStore(t)
	client, err := store.createClient("Cliente", "(32) 99999-0001")
	if err != nil {
		t.Fatal(err)
	}
	id, err := store.createFollowUp(client.ID, client.Name, client.Contact, "2026-08-12", "2026-08-13", "Pendência", "", PriorityMedium, "", "", ClientResolutionExisting, "", "")
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

func TestOnlyPendingFollowUpsCanBeEditedWithoutChangingClient(t *testing.T) {
	store, _, _ := newTestStore(t)
	client, err := store.createClient("Cliente", "(32) 99999-1111")
	if err != nil {
		t.Fatal(err)
	}
	id, err := store.createFollowUp(
		client.ID, client.Name, client.Contact, "2026-08-12", "2026-08-13",
		"Descrição inicial", "Equipe A", PriorityMedium, "Nota inicial", "", ClientResolutionExisting, "", "",
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.updatePendingFollowUp(id, "2026-08-11", "2026-08-15", "Descrição corrigida", "Equipe B", PriorityHigh, "Nota corrigida"); err != nil {
		t.Fatal(err)
	}
	updated, err := store.getFollowUp(id)
	if err != nil {
		t.Fatal(err)
	}
	if updated.ClientID != client.ID || updated.Description != "Descrição corrigida" || updated.DueDate != "2026-08-15" || updated.Priority != PriorityHigh {
		t.Fatalf("updated follow-up = %#v", updated)
	}

	if err := store.transitionFollowUp(id, StatusPending, StatusCompleted); err != nil {
		t.Fatal(err)
	}
	if err := store.updatePendingFollowUp(id, "2026-08-10", "2026-08-16", "Alteração proibida", "", PriorityLow, ""); !errors.Is(err, errInvalidTransition) {
		t.Fatalf("completed edit error = %v, want invalid transition", err)
	}
	afterRejectedEdit, err := store.getFollowUp(id)
	if err != nil || afterRejectedEdit.Description != "Descrição corrigida" {
		t.Fatalf("completed follow-up changed: %#v, %v", afterRejectedEdit, err)
	}

	if err := store.transitionFollowUp(id, StatusCompleted, StatusArchived); err != nil {
		t.Fatal(err)
	}
	if err := store.updatePendingFollowUp(id, "2026-08-10", "2026-08-16", "Outra alteração proibida", "", PriorityLow, ""); !errors.Is(err, errInvalidTransition) {
		t.Fatalf("archived edit error = %v, want invalid transition", err)
	}
}

func TestDeletePendingFollowUpPreservesClientsWithAnyHistory(t *testing.T) {
	statuses := []string{StatusPending, StatusCompleted, StatusArchived}
	for _, remainingStatus := range statuses {
		t.Run("remaining "+remainingStatus, func(t *testing.T) {
			store, _, _ := newTestStore(t)
			client, err := store.createClient("Cliente "+remainingStatus, "(32) 99999-1111")
			if err != nil {
				t.Fatal(err)
			}
			deleteID, err := store.createFollowUp(
				client.ID, client.Name, client.Contact, "2026-08-12", "2026-08-13",
				"Excluir", "", PriorityMedium, "", "", ClientResolutionExisting, "", "",
			)
			if err != nil {
				t.Fatal(err)
			}
			remainingID, err := store.createFollowUp(
				client.ID, client.Name, client.Contact, "2026-08-12", "2026-08-14",
				"Preservar", "", PriorityMedium, "", "", ClientResolutionExisting, "", "",
			)
			if err != nil {
				t.Fatal(err)
			}
			if remainingStatus != StatusPending {
				if err := store.transitionFollowUp(remainingID, StatusPending, StatusCompleted); err != nil {
					t.Fatal(err)
				}
			}
			if remainingStatus == StatusArchived {
				if err := store.transitionFollowUp(remainingID, StatusCompleted, StatusArchived); err != nil {
					t.Fatal(err)
				}
			}

			clientDeleted, err := store.deletePendingFollowUp(deleteID)
			if err != nil || clientDeleted {
				t.Fatalf("delete result = %t, %v; want preserved client", clientDeleted, err)
			}
			if _, err := store.getClient(client.ID); err != nil {
				t.Fatalf("client with %s history was deleted: %v", remainingStatus, err)
			}
			if _, err := store.getFollowUp(remainingID); err != nil {
				t.Fatalf("remaining follow-up was deleted: %v", err)
			}
		})
	}
}

func TestDeleteOnlyPendingFollowUpAlsoDeletesOrphanClient(t *testing.T) {
	store, _, _ := newTestStore(t)
	client, err := store.createClient("Cliente órfã", "(32) 99999-1111")
	if err != nil {
		t.Fatal(err)
	}
	id, err := store.createFollowUp(
		client.ID, client.Name, client.Contact, "2026-08-12", "2026-08-13",
		"Única pendência", "", PriorityMedium, "", "", ClientResolutionExisting, "", "",
	)
	if err != nil {
		t.Fatal(err)
	}
	clientDeleted, err := store.deletePendingFollowUp(id)
	if err != nil || !clientDeleted {
		t.Fatalf("delete result = %t, %v; want orphan client deleted", clientDeleted, err)
	}
	assertTableCount(t, store.db, "clients", 0)
	assertTableCount(t, store.db, "followups", 0)
}

func TestCompletedAndArchivedFollowUpsCannotBeDeleted(t *testing.T) {
	for _, status := range []string{StatusCompleted, StatusArchived} {
		t.Run(status, func(t *testing.T) {
			store, _, _ := newTestStore(t)
			client, err := store.createClient("Cliente "+status, "(32) 99999-1111")
			if err != nil {
				t.Fatal(err)
			}
			id, err := store.createFollowUp(
				client.ID, client.Name, client.Contact, "2026-08-12", "2026-08-13",
				"Registro protegido", "", PriorityMedium, "", "", ClientResolutionExisting, "", "",
			)
			if err != nil {
				t.Fatal(err)
			}
			if err := store.transitionFollowUp(id, StatusPending, StatusCompleted); err != nil {
				t.Fatal(err)
			}
			if status == StatusArchived {
				if err := store.transitionFollowUp(id, StatusCompleted, StatusArchived); err != nil {
					t.Fatal(err)
				}
			}

			if _, err := store.deletePendingFollowUp(id); !errors.Is(err, errInvalidTransition) {
				t.Fatalf("delete error = %v, want invalid transition", err)
			}
			assertTableCount(t, store.db, "clients", 1)
			assertTableCount(t, store.db, "followups", 1)
		})
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
		if _, err := store.createFollowUp(client.ID, client.Name, client.Contact, "2026-08-01", test.due, test.desc, "", test.priority, "", "", ClientResolutionExisting, "", ""); err != nil {
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

func TestDailyBaselineAndSingleRetention(t *testing.T) {
	store, _, _ := newTestStore(t)
	client, _ := store.createClient("Cliente preservada", "(32) 99999-0003")
	backupDirectory := filepath.Join(t.TempDir(), "backups")
	_ = os.MkdirAll(backupDirectory, 0o700)

	for day := 1; day <= 5; day++ {
		path := filepath.Join(backupDirectory, fmt.Sprintf("client-followup-2026-08-%02d.db", day))
		if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	today := time.Date(2026, time.August, 12, 0, 0, 0, 0, mustLocation(t))
	baselinePath, err := store.initializeBackups(backupDirectory, today)
	if err != nil {
		t.Fatal(err)
	}

	backupDB, err := sql.Open("sqlite3", baselinePath)
	if err != nil {
		t.Fatal(err)
	}
	defer backupDB.Close()
	var name string
	if err := backupDB.QueryRow("SELECT name FROM clients WHERE id = ?", client.ID).Scan(&name); err != nil || name != "Cliente preservada" {
		t.Fatalf("backup content = %q, %v", name, err)
	}

	entries, _ := os.ReadDir(backupDirectory)
	var baselineCount int
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "client-followup-") && strings.HasSuffix(entry.Name(), ".db") {
			baselineCount++
		}
	}
	if baselineCount != 1 {
		t.Fatalf("baseline count = %d, want 1", baselineCount)
	}
}

func TestRecoverySnapshotsRotationOnMutations(t *testing.T) {
	store, _, _ := newTestStore(t)
	backupDirectory := filepath.Join(t.TempDir(), "backups")
	today := time.Date(2026, time.August, 12, 0, 0, 0, 0, mustLocation(t))
	if _, err := store.initializeBackups(backupDirectory, today); err != nil {
		t.Fatal(err)
	}

	clientA, err := store.createClient("Cliente A", "(32) 99999-0001")
	if err != nil {
		t.Fatal(err)
	}

	f1ID, err := store.createFollowUp(clientA.ID, clientA.Name, clientA.Contact, "2026-08-12", "2026-08-13", "Pendência 1", "Equipe", PriorityHigh, "", "", ClientResolutionExisting, "", "")
	if err != nil {
		t.Fatal(err)
	}

	checkFollowUpCount := func(snapshotFile string, wantCount int) {
		t.Helper()
		dbPath := filepath.Join(backupDirectory, snapshotFile)
		snapDB, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			t.Fatalf("abrir %s: %v", snapshotFile, err)
		}
		defer snapDB.Close()
		var count int
		if err := snapDB.QueryRow("SELECT COUNT(*) FROM followups").Scan(&count); err != nil {
			t.Fatalf("contar followups em %s: %v", snapshotFile, err)
		}
		if count != wantCount {
			t.Fatalf("%s followups count = %d, want %d", snapshotFile, count, wantCount)
		}
	}

	checkFollowUpCount("recent-1.db", 0)

	f2ID, err := store.createFollowUp(clientA.ID, clientA.Name, clientA.Contact, "2026-08-12", "2026-08-14", "Pendência 2", "Equipe", PriorityMedium, "", "", ClientResolutionExisting, "", "")
	if err != nil {
		t.Fatal(err)
	}
	checkFollowUpCount("recent-1.db", 1)
	checkFollowUpCount("recent-2.db", 0)

	if err := store.transitionFollowUp(f1ID, StatusPending, StatusCompleted); err != nil {
		t.Fatal(err)
	}
	checkFollowUpCount("recent-1.db", 2)
	checkFollowUpCount("recent-2.db", 1)
	checkFollowUpCount("recent-3.db", 0)

	if err := store.updatePendingFollowUp(f2ID, "2026-08-12", "2026-08-15", "Pendência 2 alterada", "Outro", PriorityLow, "Nota"); err != nil {
		t.Fatal(err)
	}
	checkFollowUpCount("recent-1.db", 2)
	checkFollowUpCount("recent-2.db", 2)
	checkFollowUpCount("recent-3.db", 1)

	if _, err := os.Stat(filepath.Join(backupDirectory, "recent-tmp.db")); !os.IsNotExist(err) {
		t.Fatalf("recent-tmp.db não deveria existir após operações normais")
	}
}

func TestFailedMutationDoesNotPromoteSnapshot(t *testing.T) {
	store, _, _ := newTestStore(t)
	backupDirectory := filepath.Join(t.TempDir(), "backups")
	today := time.Date(2026, time.August, 12, 0, 0, 0, 0, mustLocation(t))
	if _, err := store.initializeBackups(backupDirectory, today); err != nil {
		t.Fatal(err)
	}

	_, err := store.createClient("Cliente Inicial", "(32) 99999-0001")
	if err != nil {
		t.Fatal(err)
	}

	snap1Before, err := os.ReadFile(filepath.Join(backupDirectory, "recent-1.db"))
	if err != nil {
		t.Fatal(err)
	}

	// Provoca falha após prepareRecoverySnapshot (status válidos, mas ID inexistente)
	err = store.transitionFollowUp(999999, StatusPending, StatusCompleted)
	if !errors.Is(err, errInvalidTransition) {
		t.Fatalf("esperava errInvalidTransition, obtido: %v", err)
	}

	snap1After, err := os.ReadFile(filepath.Join(backupDirectory, "recent-1.db"))
	if err != nil {
		t.Fatal(err)
	}
	if string(snap1Before) != string(snap1After) {
		t.Fatalf("recent-1.db foi alterado em operação que falhou")
	}
	if _, err := os.Stat(filepath.Join(backupDirectory, "recent-2.db")); !os.IsNotExist(err) {
		t.Fatalf("recent-2.db não deveria existir")
	}
	if _, err := os.Stat(filepath.Join(backupDirectory, "recent-tmp.db")); !os.IsNotExist(err) {
		t.Fatalf("recent-tmp.db deveria ter sido descartado")
	}
}

func TestPreRestorePreservedFromCleanDailyBaselines(t *testing.T) {
	store, _, _ := newTestStore(t)
	backupDirectory := filepath.Join(t.TempDir(), "backups")
	_ = os.MkdirAll(backupDirectory, 0o700)

	preRestorePath := filepath.Join(backupDirectory, "pre-restore.db")
	if err := os.WriteFile(preRestorePath, []byte("pre-restore-content"), 0o600); err != nil {
		t.Fatal(err)
	}

	today := time.Date(2026, time.August, 13, 0, 0, 0, 0, mustLocation(t))
	if _, err := store.initializeBackups(backupDirectory, today); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(preRestorePath)
	if err != nil {
		t.Fatalf("pre-restore.db deveria existir: %v", err)
	}
	if string(content) != "pre-restore-content" {
		t.Fatalf("conteúdo de pre-restore.db = %q", string(content))
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

func TestDueDateCannotBeBeforeStartDate(t *testing.T) {
	store, _, _ := newTestStore(t)
	client, err := store.createClient("Cliente Teste", "(32) 99999-0001")
	if err != nil {
		t.Fatal(err)
	}

	// Tentativa de criar pendência com due_date anterior a start_date
	_, err = store.createFollowUp(
		client.ID, client.Name, client.Contact,
		"2026-08-15", "2026-08-10",
		"Descrição válida", "", PriorityMedium, "", "", ClientResolutionExisting, "", "",
	)
	if err == nil || !strings.Contains(err.Error(), "a data limite não pode ser anterior à data de início") {
		t.Fatalf("expected error for due_date before start_date in createFollowUp, got: %v", err)
	}

	// Criar pendência válida
	id, err := store.createFollowUp(
		client.ID, client.Name, client.Contact,
		"2026-08-10", "2026-08-15",
		"Descrição válida", "", PriorityMedium, "", "", ClientResolutionExisting, "", "",
	)
	if err != nil {
		t.Fatalf("createFollowUp failed: %v", err)
	}

	// Tentativa de atualizar pendência com due_date anterior a start_date
	err = store.updatePendingFollowUp(id, "2026-08-20", "2026-08-18", "Descrição atualizada", "", PriorityHigh, "")
	if err == nil || !strings.Contains(err.Error(), "a data limite não pode ser anterior à data de início") {
		t.Fatalf("expected error for due_date before start_date in updatePendingFollowUp, got: %v", err)
	}

	// Atualização válida
	err = store.updatePendingFollowUp(id, "2026-08-10", "2026-08-20", "Descrição atualizada", "", PriorityHigh, "")
	if err != nil {
		t.Fatalf("updatePendingFollowUp failed: %v", err)
	}
}

func TestReportFollowUpsAccentAndCaseInsensitive(t *testing.T) {
	store, _, _ := newTestStore(t)
	c1, err := store.createClient("João Valença", "(32) 99999-0001")
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.createFollowUp(
		c1.ID, c1.Name, c1.Contact,
		"2026-08-10", "2026-08-15",
		"Exame cardiológico", "Márcio Silva", PriorityHigh, "", "", ClientResolutionExisting, "", "",
	)
	if err != nil {
		t.Fatal(err)
	}

	// Busca por "joao" (sem acento)
	results, err := store.reportFollowUps(ReportFilters{Client: "joao"}, "2026-08-10")
	if err != nil || len(results) != 1 {
		t.Fatalf("expected 1 result for 'joao', got %d (err: %v)", len(results), err)
	}

	// Busca por "valenca" (sem cedilha)
	results, err = store.reportFollowUps(ReportFilters{Client: "valenca"}, "2026-08-10")
	if err != nil || len(results) != 1 {
		t.Fatalf("expected 1 result for 'valenca', got %d (err: %v)", len(results), err)
	}

	// Busca por encaminhamento "marcio" (sem acento)
	results, err = store.reportFollowUps(ReportFilters{ForwardTo: "marcio"}, "2026-08-10")
	if err != nil || len(results) != 1 {
		t.Fatalf("expected 1 result for 'marcio', got %d (err: %v)", len(results), err)
	}

	// Busca por encaminhamento "MÁRCIO"
	results, err = store.reportFollowUps(ReportFilters{ForwardTo: "MÁRCIO"}, "2026-08-10")
	if err != nil || len(results) != 1 {
		t.Fatalf("expected 1 result for 'MÁRCIO', got %d (err: %v)", len(results), err)
	}
}
