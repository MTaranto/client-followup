package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

func TestDashboardEscapesUserContent(t *testing.T) {
	store, _, _ := newTestStore(t)
	client, err := store.createClient(`<script>alert("client")</script>`, "(32) 99999-0004")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.createFollowUp(client.ID, client.Name, client.Contact, "2026-08-12", "2026-08-13", `<img src=x onerror=alert("description")>`, "", PriorityMedium, "", "", ClientResolutionExisting, "", ""); err != nil {
		t.Fatal(err)
	}
	app, err := newApplication(store, mustLocation(t))
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	app.routes().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	body := response.Body.String()
	if strings.Contains(body, `<script>alert`) || strings.Contains(body, `<img src=x`) {
		t.Fatalf("user content was rendered without escaping: %s", body)
	}
	if !strings.Contains(body, "&lt;script&gt;") || !strings.Contains(body, "&lt;img") {
		t.Fatalf("escaped user content not found: %s", body)
	}
}

func TestFollowUpHTTPWorkflow(t *testing.T) {
	store, _, _ := newTestStore(t)
	app, err := newApplication(store, mustLocation(t))
	if err != nil {
		t.Fatal(err)
	}
	handler := app.routes()

	form := url.Values{
		"client_name":       {"Cliente integração"},
		"client_resolution": {ClientResolutionNew},
		"contact":           {"32999990005"},
		"start_date":        {"2026-08-12"},
		"due_date":          {"2026-08-13"},
		"priority":          {PriorityHigh},
		"description":       {"Confirmar retorno"},
		"forward_to":        {"Equipe A"},
		"notes":             {"Administrativo"},
	}
	response := performRequest(handler, http.MethodPost, "/followups", form)
	if response.Code != http.StatusOK || response.Header().Get("HX-Refresh") != "true" {
		t.Fatalf("create response = %d, HX-Refresh %q, body %s", response.Code, response.Header().Get("HX-Refresh"), response.Body.String())
	}

	var followUpID, clientID int64
	if err := store.db.QueryRow("SELECT id, client_id FROM followups").Scan(&followUpID, &clientID); err != nil {
		t.Fatal(err)
	}
	response = performRequest(handler, http.MethodGet, "/clients/search?client_name=integra", nil)
	assertResponseContains(t, response, "Cliente integração", "(32) 99999-0005")
	response = performRequest(handler, http.MethodGet, "/reminders", nil)
	assertResponseContains(t, response, "Vence amanhã", "highlight="+strconv.FormatInt(followUpID, 10))

	response = performRequest(handler, http.MethodPost, "/followups/"+strconv.FormatInt(followUpID, 10)+"/complete", nil)
	if response.Code != http.StatusOK || response.Header().Get("HX-Refresh") != "true" {
		t.Fatalf("expected 200 OK with HX-Refresh on complete, got %d (header %q)", response.Code, response.Header().Get("HX-Refresh"))
	}
	response = performRequest(handler, http.MethodPost, "/followups/"+strconv.FormatInt(followUpID, 10)+"/reopen", nil)
	if response.Code != http.StatusOK || response.Header().Get("HX-Refresh") != "true" {
		t.Fatalf("expected 200 OK with HX-Refresh on reopen, got %d (header %q)", response.Code, response.Header().Get("HX-Refresh"))
	}
	performRequest(handler, http.MethodPost, "/followups/"+strconv.FormatInt(followUpID, 10)+"/complete", nil)
	response = performRequest(handler, http.MethodPost, "/followups/"+strconv.FormatInt(followUpID, 10)+"/archive", nil)
	if response.Code != http.StatusOK || response.Header().Get("HX-Refresh") != "true" {
		t.Fatalf("expected 200 OK with HX-Refresh on archive, got %d (header %q)", response.Code, response.Header().Get("HX-Refresh"))
	}

	response = performRequest(handler, http.MethodGet, "/dashboard", nil)
	if strings.Contains(response.Body.String(), "Confirmar retorno") {
		t.Fatal("archived follow-up remained on dashboard")
	}
	response = performRequest(handler, http.MethodGet, "/clients/"+strconv.FormatInt(clientID, 10), nil)
	if strings.Contains(response.Body.String(), "Confirmar retorno") {
		t.Fatal("archived follow-up remained in client detail")
	}
	response = performRequest(handler, http.MethodGet, "/reports/results?status=ARCHIVED", nil)
	assertResponseContains(t, response, "Confirmar retorno", "Arquivado")
}

func TestClientSuggestionsDistinguishHomonymousClients(t *testing.T) {
	store, _, _ := newTestStore(t)
	withContact, err := store.createClient("Ana Silva", "(32) 99999-1111")
	if err != nil {
		t.Fatal(err)
	}
	withoutContact := insertLegacyClient(t, store, "Ana Silva", "")
	app, err := newApplication(store, mustLocation(t))
	if err != nil {
		t.Fatal(err)
	}

	response := performRequest(app.routes(), http.MethodGet, "/clients/search?client_name=Ana", nil)
	assertResponseContains(
		t,
		response,
		`hx-get="/followups/new?client_id=`+strconv.FormatInt(withContact.ID, 10)+`"`,
		"(32) 99999-1111",
		`hx-get="/followups/new?client_id=`+strconv.FormatInt(withoutContact.ID, 10)+`"`,
		"Cadastro #"+strconv.FormatInt(withoutContact.ID, 10),
	)
}

func TestDashboardClientLookupDoesNotTargetFollowUpTable(t *testing.T) {
	store, _, _ := newTestStore(t)
	client, err := store.createClient("Cliente operacional", "(32) 99999-1234")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.createFollowUp(client.ID, client.Name, client.Contact, "2026-08-12", "2026-08-14", "Pendência preservada", "", PriorityMedium, "", "", ClientResolutionExisting, "", ""); err != nil {
		t.Fatal(err)
	}
	app, err := newApplication(store, mustLocation(t))
	if err != nil {
		t.Fatal(err)
	}

	response := performRequest(app.routes(), http.MethodGet, "/", nil)
	assertResponseContains(
		t,
		response,
		`id="client-lookup"`,
		`id="dashboard-client-search"`,
		`hx-get="/clients/search"`,
		`hx-target="#client-search-results"`,
		`hx-trigger="input queue:last, search queue:last"`,
		`type="submit">Buscar</button>`,
		"Pendência preservada",
	)
	body := response.Body.String()
	for _, removed := range []string{`<select name="priority"`, `<select name="status"`, `<select name="due"`, `>Limpar<`} {
		if strings.Contains(body, removed) {
			t.Fatalf("dashboard still contains removed filter %q: %s", removed, body)
		}
	}
	if strings.Contains(body, `hx-target="#dashboard-results" hx-swap="innerHTML"`) {
		t.Fatalf("client lookup targets the follow-up table: %s", body)
	}
}

func TestFollowUpFormCanBeOpenedFromClientOrTypedName(t *testing.T) {
	store, _, _ := newTestStore(t)
	client, err := store.createClient("Beatriz Lima", "(32) 96666-4444")
	if err != nil {
		t.Fatal(err)
	}
	app, err := newApplication(store, mustLocation(t))
	if err != nil {
		t.Fatal(err)
	}
	handler := app.routes()

	response := performRequest(handler, http.MethodGet, "/followups/new?client_id="+strconv.FormatInt(client.ID, 10), nil)
	assertResponseContains(
		t,
		response,
		`name="client_id" id="client-id" value="`+strconv.FormatInt(client.ID, 10)+`"`,
		`name="client_name" value="Beatriz Lima"`,
		`name="contact" value="(32) 96666-4444"`,
		`name="start_date" value="2026-08-12"`,
		`name="client_resolution" id="client-resolution" value="existing"`,
		`<fieldset id="followup-fields" class="followup-fields full-width" >`,
		`id="followup-save" type="submit" class="button primary" >`,
	)

	response = performRequest(handler, http.MethodGet, "/followups/new?client_name=Juliana+Carvalho", nil)
	assertResponseContains(
		t,
		response,
		`name="client_name" value="Juliana Carvalho"`,
		`name="contact" value=""`,
		`name="due_date"`,
		`<fieldset id="followup-fields" class="followup-fields full-width" disabled>`,
		`id="followup-save" type="submit" class="button primary" disabled>`,
	)
	if strings.Contains(response.Body.String(), `name="client_id" id="client-id" value="`+strconv.FormatInt(client.ID, 10)+`"`) {
		t.Fatalf("typed name inherited an existing client ID: %s", response.Body.String())
	}
	assertTableCount(t, store.db, "clients", 1)
	assertTableCount(t, store.db, "followups", 0)

	response = performRequest(handler, http.MethodGet, "/followups/new", nil)
	assertResponseContains(t, response, `name="client_id" id="client-id" value=""`, `name="client_name" value=""`, `name="contact" value=""`)
}

func TestClientNameEndpointsRejectDigitsAndAmbiguousCreation(t *testing.T) {
	store, _, _ := newTestStore(t)
	client, err := store.createClient("João Costa", "(32) 99999-1111")
	if err != nil {
		t.Fatal(err)
	}
	app, err := newApplication(store, mustLocation(t))
	if err != nil {
		t.Fatal(err)
	}
	handler := app.routes()

	response := performRequest(handler, http.MethodGet, "/clients/search?client_name=Ana2", nil)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("digit search status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	response = performRequest(handler, http.MethodGet, "/clients/exact?client_name=Ana%D9%A2", nil)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("Unicode digit exact lookup status = %d, want %d", response.Code, http.StatusBadRequest)
	}

	form := followUpFormValues(Client{}, "(32) 98888-2222")
	form.Set("client_name", "JOAO COSTA")
	response = performRequest(handler, http.MethodPost, "/followups", form)
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "escolha um cliente existente") {
		t.Fatalf("ambiguous creation response = %d, body = %s", response.Code, response.Body.String())
	}
	assertTableCount(t, store.db, "clients", 1)
	assertTableCount(t, store.db, "followups", 0)
	assertClientPhone(t, store, client.ID, "(32) 99999-1111")
}

func TestExactClientLookupDistinguishesZeroOnePartialAndMultipleMatches(t *testing.T) {
	store, _, _ := newTestStore(t)
	first, err := store.createClient("Ana Silva", "(32) 99999-1111")
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.createClient("ana silva", "(32) 98888-2222")
	if err != nil {
		t.Fatal(err)
	}
	beatriz, err := store.createClient("Beatriz Lima", "(32) 96666-4444")
	if err != nil {
		t.Fatal(err)
	}
	app, err := newApplication(store, mustLocation(t))
	if err != nil {
		t.Fatal(err)
	}
	handler := app.routes()

	response := performRequest(handler, http.MethodGet, "/clients/exact?client_name=Ana", nil)
	assertResponseContains(t, response, `"clients":[]`)
	response = performRequest(handler, http.MethodGet, "/clients/exact?client_name=beatriz+lima", nil)
	assertResponseContains(t, response, `"id":`+strconv.FormatInt(beatriz.ID, 10), `"contact":"(32) 96666-4444"`)
	response = performRequest(handler, http.MethodGet, "/clients/exact?client_name=ANA+SILVA", nil)
	assertResponseContains(
		t,
		response,
		`"id":`+strconv.FormatInt(first.ID, 10),
		`"id":`+strconv.FormatInt(second.ID, 10),
		"(32) 99999-1111",
		"(32) 98888-2222",
	)
	response = performRequest(handler, http.MethodGet, "/clients/exact?client_name=Inexistente", nil)
	assertResponseContains(t, response, `"clients":[]`)

	response = performRequest(handler, http.MethodGet, "/followups/new", nil)
	body := response.Body.String()
	if strings.Contains(body, `hx-get="/clients/search"`) || strings.Contains(body, `client-suggestions`) {
		t.Fatalf("follow-up form still contains incremental suggestions: %s", body)
	}
}

func TestFollowUpPhoneChangeRequiresAndAppliesExplicitDecision(t *testing.T) {
	t.Run("update existing client", func(t *testing.T) {
		store, _, _ := newTestStore(t)
		client, err := store.createClient("Ana Silva", "(32) 99999-1111")
		if err != nil {
			t.Fatal(err)
		}
		app, err := newApplication(store, mustLocation(t))
		if err != nil {
			t.Fatal(err)
		}
		form := followUpFormValues(client, "(32) 98888-2222")

		response := performRequest(app.routes(), http.MethodPost, "/followups", form)
		if response.Code != http.StatusConflict {
			t.Fatalf("confirmation status = %d, body = %s", response.Code, response.Body.String())
		}
		assertBodyContains(
			t,
			response.Body.String(),
			"(32) 99999-1111",
			"(32) 98888-2222",
			`data-phone-change-action="update"`,
			`data-phone-change-action="new_client"`,
		)
		assertClientPhone(t, store, client.ID, "(32) 99999-1111")
		assertTableCount(t, store.db, "followups", 0)

		form.Set("phone_change_action", PhoneChangeUpdate)
		response = performRequest(app.routes(), http.MethodPost, "/followups", form)
		if response.Code != http.StatusOK {
			t.Fatalf("update decision status = %d, body = %s", response.Code, response.Body.String())
		}
		assertClientPhone(t, store, client.ID, "(32) 98888-2222")
		var followUpID int64
		if err := store.db.QueryRow("SELECT id FROM followups").Scan(&followUpID); err != nil {
			t.Fatal(err)
		}
		if got := followUpClientID(t, store.db, followUpID); got != client.ID {
			t.Fatalf("follow-up client ID = %d, want %d", got, client.ID)
		}
	})

	t.Run("create homonymous client", func(t *testing.T) {
		store, _, _ := newTestStore(t)
		client, err := store.createClient("Ana Silva", "(32) 99999-1111")
		if err != nil {
			t.Fatal(err)
		}
		app, err := newApplication(store, mustLocation(t))
		if err != nil {
			t.Fatal(err)
		}
		form := followUpFormValues(client, "(32) 98888-2222")
		form.Set("phone_change_action", PhoneChangeNewClient)

		response := performRequest(app.routes(), http.MethodPost, "/followups", form)
		if response.Code != http.StatusOK {
			t.Fatalf("new client decision status = %d, body = %s", response.Code, response.Body.String())
		}
		var followUpID int64
		if err := store.db.QueryRow("SELECT id FROM followups").Scan(&followUpID); err != nil {
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

func TestPhoneChangeConfirmationOnFieldExitDoesNotPersist(t *testing.T) {
	store, _, _ := newTestStore(t)
	client, err := store.createClient("Ana Silva", "(32) 99999-1111")
	if err != nil {
		t.Fatal(err)
	}
	app, err := newApplication(store, mustLocation(t))
	if err != nil {
		t.Fatal(err)
	}
	query := url.Values{
		"client_id":   {strconv.FormatInt(client.ID, 10)},
		"client_name": {client.Name},
		"contact":     {"(32) 98888-2222"},
	}
	response := performRequest(app.routes(), http.MethodGet, "/clients/phone-change-confirmation?"+query.Encode(), nil)
	assertResponseContains(t, response, "(32) 99999-1111", "(32) 98888-2222", `data-phone-change-action="update"`)
	assertClientPhone(t, store, client.ID, "(32) 99999-1111")
	assertTableCount(t, store.db, "clients", 1)
	assertTableCount(t, store.db, "followups", 0)

	query.Set("contact", client.Contact)
	response = performRequest(app.routes(), http.MethodGet, "/clients/phone-change-confirmation?"+query.Encode(), nil)
	if response.Code != http.StatusOK || strings.TrimSpace(response.Body.String()) != "" {
		t.Fatalf("unchanged phone produced confirmation: status %d body %q", response.Code, response.Body.String())
	}
}

func TestFollowUpRejectsInvalidSelectedClientIdentity(t *testing.T) {
	store, _, _ := newTestStore(t)
	client, err := store.createClient("Ana Silva", "(32) 99999-1111")
	if err != nil {
		t.Fatal(err)
	}
	app, err := newApplication(store, mustLocation(t))
	if err != nil {
		t.Fatal(err)
	}

	form := followUpFormValues(client, client.Contact)
	form.Set("client_name", "Outra Pessoa")
	response := performRequest(app.routes(), http.MethodPost, "/followups", form)
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "não corresponde") {
		t.Fatalf("mismatched identity response = %d, body = %s", response.Code, response.Body.String())
	}

	form = followUpFormValues(client, client.Contact)
	form.Set("client_id", "999999")
	response = performRequest(app.routes(), http.MethodPost, "/followups", form)
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "não foi encontrado") {
		t.Fatalf("missing identity response = %d, body = %s", response.Code, response.Body.String())
	}
	assertTableCount(t, store.db, "clients", 1)
	assertTableCount(t, store.db, "followups", 0)
}

func TestFollowUpAndClientUpdateRejectInvalidPhones(t *testing.T) {
	store, _, _ := newTestStore(t)
	client, err := store.createClient("Ana Silva", "(32) 99999-1111")
	if err != nil {
		t.Fatal(err)
	}
	app, err := newApplication(store, mustLocation(t))
	if err != nil {
		t.Fatal(err)
	}

	for _, phone := range []string{"", "3299999111", "329999911112", "phone"} {
		response := performRequest(app.routes(), http.MethodPost, "/followups", followUpFormValues(Client{}, phone))
		if response.Code != http.StatusBadRequest {
			t.Errorf("follow-up phone %q status = %d, want %d", phone, response.Code, http.StatusBadRequest)
		}
	}
	response := performRequest(app.routes(), http.MethodPost, "/clients/"+strconv.FormatInt(client.ID, 10), url.Values{
		"name":    {client.Name},
		"contact": {""},
	})
	if response.Code != http.StatusBadRequest {
		t.Fatalf("client update status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestClientEditRequiresExplicitPhoneConfirmation(t *testing.T) {
	store, _, _ := newTestStore(t)
	client, err := store.createClient("Ana Silva", "(32) 99999-1111")
	if err != nil {
		t.Fatal(err)
	}
	app, err := newApplication(store, mustLocation(t))
	if err != nil {
		t.Fatal(err)
	}
	handler := app.routes()
	clientPath := "/clients/" + strconv.FormatInt(client.ID, 10)

	response := performRequest(handler, http.MethodGet, clientPath, nil)
	assertResponseContains(t, response, `id="client-edit-region" class="client-summary no-print"`, "Editar cliente")
	if strings.Contains(response.Body.String(), `name="contact"`) {
		t.Fatalf("client detail opened in edit mode: %s", response.Body.String())
	}
	response = performRequest(handler, http.MethodGet, clientPath+"/edit", nil)
	assertResponseContains(t, response, `name="contact" value="(32) 99999-1111"`, `data-original-phone="(32) 99999-1111"`)

	form := url.Values{"name": {"Ana Souza"}, "contact": {"(32) 98888-2222"}}
	response = performRequest(handler, http.MethodPost, clientPath, form)
	if response.Code != http.StatusConflict {
		t.Fatalf("phone confirmation status = %d, body = %s", response.Code, response.Body.String())
	}
	assertBodyContains(t, response.Body.String(), "Confirmar alteração", "(32) 99999-1111", "(32) 98888-2222")
	unchanged, err := store.getClient(client.ID)
	if err != nil || unchanged.Name != "Ana Silva" || unchanged.Contact != "(32) 99999-1111" {
		t.Fatalf("client persisted before confirmation: %#v, %v", unchanged, err)
	}

	form.Set("phone_change_confirmation", ClientPhoneChangeConfirmation)
	response = performRequest(handler, http.MethodPost, clientPath, form)
	if response.Code != http.StatusOK || !strings.Contains(response.Header().Get("HX-Trigger"), "clientChanged") {
		t.Fatalf("confirmed update response = %d, trigger %q, body = %s", response.Code, response.Header().Get("HX-Trigger"), response.Body.String())
	}
	updated, err := store.getClient(client.ID)
	if err != nil || updated.Name != "Ana Souza" || updated.Contact != "(32) 98888-2222" {
		t.Fatalf("confirmed client = %#v, %v", updated, err)
	}
	assertTableCount(t, store.db, "clients", 1)
}

func TestFollowUpEditAndDeleteRoutesEnforcePendingStatus(t *testing.T) {
	store, _, _ := newTestStore(t)
	client, err := store.createClient("Cliente protegida", "(32) 99999-1111")
	if err != nil {
		t.Fatal(err)
	}
	id, err := store.createFollowUp(
		client.ID, client.Name, client.Contact, "2026-08-12", "2026-08-13",
		"Descrição inicial", "Equipe A", PriorityMedium, "Nota", "", ClientResolutionExisting, "", "",
	)
	if err != nil {
		t.Fatal(err)
	}
	app, err := newApplication(store, mustLocation(t))
	if err != nil {
		t.Fatal(err)
	}
	handler := app.routes()
	followUpPath := "/followups/" + strconv.FormatInt(id, 10)

	response := performRequest(handler, http.MethodGet, followUpPath+"/edit", nil)
	assertResponseContains(t, response, "Editar pendência", "Cliente protegida", `name="description"`)
	if strings.Contains(response.Body.String(), `name="client_id"`) || strings.Contains(response.Body.String(), `name="status"`) {
		t.Fatalf("edit form exposes protected fields: %s", response.Body.String())
	}
	response = performRequest(handler, http.MethodPost, followUpPath+"/edit", url.Values{
		"start_date":  {"2026-08-11"},
		"due_date":    {"2026-08-15"},
		"priority":    {PriorityHigh},
		"description": {"Descrição atualizada"},
		"forward_to":  {"Equipe B"},
		"notes":       {"Nota atualizada"},
	})
	if response.Code != http.StatusOK {
		t.Fatalf("pending edit status = %d, body = %s", response.Code, response.Body.String())
	}
	updated, err := store.getFollowUp(id)
	if err != nil || updated.ClientID != client.ID || updated.Description != "Descrição atualizada" {
		t.Fatalf("updated follow-up = %#v, %v", updated, err)
	}

	response = performRequest(handler, http.MethodPost, followUpPath+"/complete", nil)
	if response.Code != http.StatusOK {
		t.Fatalf("complete status = %d, body = %s", response.Code, response.Body.String())
	}
	for _, request := range []struct {
		method string
		path   string
		form   url.Values
	}{
		{method: http.MethodGet, path: followUpPath + "/edit"},
		{method: http.MethodPost, path: followUpPath + "/edit", form: url.Values{
			"start_date": {"2026-08-11"}, "due_date": {"2026-08-16"}, "priority": {PriorityLow}, "description": {"Proibida"},
		}},
		{method: http.MethodPost, path: followUpPath + "/delete"},
	} {
		response = performRequest(handler, request.method, request.path, request.form)
		if response.Code != http.StatusConflict {
			t.Errorf("%s %s status = %d, body = %s", request.method, request.path, response.Code, response.Body.String())
		}
	}
	assertTableCount(t, store.db, "clients", 1)
	assertTableCount(t, store.db, "followups", 1)
}

func TestDeletePendingRouteDeletesOrphanClient(t *testing.T) {
	store, _, _ := newTestStore(t)
	client, err := store.createClient("Cliente temporária", "(32) 99999-1111")
	if err != nil {
		t.Fatal(err)
	}
	id, err := store.createFollowUp(
		client.ID, client.Name, client.Contact, "2026-08-12", "2026-08-13",
		"Excluir", "", PriorityMedium, "", "", ClientResolutionExisting, "", "",
	)
	if err != nil {
		t.Fatal(err)
	}
	app, err := newApplication(store, mustLocation(t))
	if err != nil {
		t.Fatal(err)
	}
	response := performRequest(app.routes(), http.MethodPost, "/followups/"+strconv.FormatInt(id, 10)+"/delete", nil)
	if response.Code != http.StatusOK || response.Header().Get("HX-Refresh") != "true" {
		t.Fatalf("delete response = %d, HX-Refresh %q, body = %s", response.Code, response.Header().Get("HX-Refresh"), response.Body.String())
	}
	assertTableCount(t, store.db, "clients", 0)
	assertTableCount(t, store.db, "followups", 0)
}

func followUpFormValues(client Client, phone string) url.Values {
	values := url.Values{
		"client_name":       {"Nova Cliente"},
		"client_resolution": {ClientResolutionNew},
		"contact":           {phone},
		"start_date":        {"2026-08-12"},
		"due_date":          {"2026-08-13"},
		"priority":          {PriorityMedium},
		"description":       {"Confirmar retorno"},
	}
	if client.ID > 0 {
		values.Set("client_id", strconv.FormatInt(client.ID, 10))
		values.Set("client_name", client.Name)
		values.Set("client_resolution", ClientResolutionExisting)
	}
	return values
}

func performRequest(handler http.Handler, method, target string, form url.Values) *httptest.ResponseRecorder {
	var body *strings.Reader
	if form == nil {
		body = strings.NewReader("")
	} else {
		body = strings.NewReader(form.Encode())
	}
	request := httptest.NewRequest(method, target, body)
	if form != nil {
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func assertResponseContains(t *testing.T, response *httptest.ResponseRecorder, values ...string) {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	for _, value := range values {
		if !strings.Contains(response.Body.String(), value) {
			t.Fatalf("response does not contain %q: %s", value, response.Body.String())
		}
	}
}

func assertBodyContains(t *testing.T, body string, values ...string) {
	t.Helper()
	for _, value := range values {
		if !strings.Contains(body, value) {
			t.Fatalf("response does not contain %q: %s", value, body)
		}
	}
}

func TestHealth(t *testing.T) {
	store, _, _ := newTestStore(t)
	app, err := newApplication(store, mustLocation(t))
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	app.routes().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health", nil))
	if response.Code != http.StatusOK || response.Body.String() != "ok\n" {
		t.Fatalf("health response = %d %q", response.Code, response.Body.String())
	}
}

func TestDashboardHeaderAndSearchPlaceholder(t *testing.T) {
	store, _, _ := newTestStore(t)
	app, err := newApplication(store, mustLocation(t))
	if err != nil {
		t.Fatal(err)
	}
	response := performRequest(app.routes(), http.MethodGet, "/", nil)
	assertResponseContains(t, response, "Acompanhamento", "Pendências de clientes", `placeholder="Digite o nome do cliente"`)
}

func TestFollowUpActionsOrderAndTerminology(t *testing.T) {
	store, _, _ := newTestStore(t)
	client, err := store.createClient("Carlos Santos", "(32) 99999-0001")
	if err != nil {
		t.Fatal(err)
	}
	followUpID, err := store.createFollowUp(client.ID, client.Name, client.Contact, "2026-08-12", "2026-08-13", "Retorno agendado", "", PriorityMedium, "", "", ClientResolutionExisting, "", "")
	if err != nil {
		t.Fatal(err)
	}
	app, err := newApplication(store, mustLocation(t))
	if err != nil {
		t.Fatal(err)
	}
	handler := app.routes()

	checkOrder := func(body, contextName string) {
		actionsIdx := strings.Index(body, `class="actions`)
		if actionsIdx == -1 {
			t.Fatalf("[%s] actions container not found in body", contextName)
		}
		actionsBlock := body[actionsIdx:]
		finalizeIdx := strings.Index(actionsBlock, "Finalizar")
		editIdx := strings.Index(actionsBlock, ">Editar</button>")
		deleteIdx := strings.Index(actionsBlock, "Excluir")
		if finalizeIdx == -1 || editIdx == -1 || deleteIdx == -1 {
			t.Fatalf("[%s] actions not found in actions block: %s", contextName, actionsBlock)
		}
		if !(finalizeIdx < editIdx && editIdx < deleteIdx) {
			t.Fatalf("[%s] expected order Finalizar -> Editar -> Excluir, got indices %d, %d, %d", contextName, finalizeIdx, editIdx, deleteIdx)
		}
	}

	dashboardResp := performRequest(handler, http.MethodGet, "/dashboard", nil)
	checkOrder(dashboardResp.Body.String(), "dashboard")

	clientResp := performRequest(handler, http.MethodGet, "/clients/"+strconv.FormatInt(client.ID, 10), nil)
	checkOrder(clientResp.Body.String(), "client-detail")
	assertResponseContains(t, clientResp, "Ficha do cliente", "Editar cliente")
	_ = followUpID
}

func TestClientDetailNotesSemanticPrefix(t *testing.T) {
	store, _, _ := newTestStore(t)
	client, err := store.createClient("Mariana Ribeiro", "(32) 99999-0002")
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.createFollowUp(client.ID, client.Name, client.Contact, "2026-08-12", "2026-08-13", "Checar exames", "", PriorityHigh, "Retorno na sexta-feira", "", ClientResolutionExisting, "", "")
	if err != nil {
		t.Fatal(err)
	}
	app, err := newApplication(store, mustLocation(t))
	if err != nil {
		t.Fatal(err)
	}

	response := performRequest(app.routes(), http.MethodGet, "/clients/"+strconv.FormatInt(client.ID, 10), nil)
	assertResponseContains(t, response, `<small class="followup-notes">Retorno na sexta-feira</small>`)
}

func TestDuplicatePhoneHTTPWorkflow(t *testing.T) {
	store, _, _ := newTestStore(t)
	existing, err := store.createClient("Cliente Um", "(32) 99999-8888")
	if err != nil {
		t.Fatal(err)
	}
	app, err := newApplication(store, mustLocation(t))
	if err != nil {
		t.Fatal(err)
	}
	handler := app.routes()

	// Tentativa sem confirmação
	form := url.Values{
		"client_name":       {"Cliente Dois"},
		"client_resolution": {ClientResolutionNew},
		"contact":           {existing.Contact},
		"start_date":        {"2026-08-12"},
		"due_date":          {"2026-08-13"},
		"priority":          {PriorityHigh},
		"description":       {"Retorno com duplicidade"},
	}
	response := performRequest(handler, http.MethodPost, "/followups", form)
	if response.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d", response.Code)
	}
	assertBodyContains(t, response.Body.String(), "Telefone já utilizado", "Cliente Um", "data-phone-duplicate-action")

	// GET para verificação prévia de telefone
	verifyResp := performRequest(handler, http.MethodGet, "/clients/phone-change-confirmation?contact=32999998888", nil)
	if verifyResp.Code != http.StatusOK {
		t.Fatalf("verify status = %d", verifyResp.Code)
	}
	assertBodyContains(t, verifyResp.Body.String(), "Telefone já utilizado", "Cliente Um", "data-duplicate-token")

	// Extrai token legítimo
	body := verifyResp.Body.String()
	tokenMarker := `data-duplicate-token="`
	idx := strings.Index(body, tokenMarker)
	if idx == -1 {
		t.Fatalf("token marker not found in body: %s", body)
	}
	token := body[idx+len(tokenMarker):]
	endIdx := strings.Index(token, `"`)
	token = token[:endIdx]

	// Tentativa com allow e token forjado
	forgedForm := url.Values{
		"client_name":              {"Cliente Dois"},
		"client_resolution":        {ClientResolutionNew},
		"contact":                  {existing.Contact},
		"start_date":               {"2026-08-12"},
		"due_date":                 {"2026-08-13"},
		"priority":                 {PriorityHigh},
		"description":              {"Retorno com token forjado"},
		"duplicate_phone_decision": {"allow"},
		"duplicate_phone_token":    {"token_forjado"},
	}
	response = performRequest(handler, http.MethodPost, "/followups", forgedForm)
	if response.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for forged token, got %d", response.Code)
	}

	// Tentativa com allow e token legítimo
	legitForm := url.Values{
		"client_name":              {"Cliente Dois"},
		"client_resolution":        {ClientResolutionNew},
		"contact":                  {existing.Contact},
		"start_date":               {"2026-08-12"},
		"due_date":                 {"2026-08-13"},
		"priority":                 {PriorityHigh},
		"description":              {"Retorno legítimo"},
		"duplicate_phone_decision": {"allow"},
		"duplicate_phone_token":    {token},
	}
	response = performRequest(handler, http.MethodPost, "/followups", legitForm)
	if response.Code != http.StatusOK || response.Header().Get("HX-Refresh") != "true" {
		t.Fatalf("expected 200 OK with HX-Refresh, got %d (header: %q, body: %s)", response.Code, response.Header().Get("HX-Refresh"), response.Body.String())
	}
}

func TestDeleteAndArchiveHTTPWorkflowWithTriggers(t *testing.T) {
	store, _, _ := newTestStore(t)
	app, err := newApplication(store, mustLocation(t))
	if err != nil {
		t.Fatal(err)
	}
	handler := app.routes()

	client, err := store.createClient("Cliente Único", "(32) 99999-0001")
	if err != nil {
		t.Fatal(err)
	}
	id, err := store.createFollowUp(client.ID, client.Name, client.Contact, "2026-08-12", "2026-08-13", "Para arquivar", "", PriorityMedium, "", "", ClientResolutionExisting, "", "")
	if err != nil {
		t.Fatal(err)
	}

	// Finaliza -> HX-Refresh: true
	completeResp := performRequest(handler, http.MethodPost, "/followups/"+strconv.FormatInt(id, 10)+"/complete", nil)
	if completeResp.Code != http.StatusOK || completeResp.Header().Get("HX-Refresh") != "true" {
		t.Fatalf("expected HX-Refresh on complete, got code %d header %q", completeResp.Code, completeResp.Header().Get("HX-Refresh"))
	}
	if strings.Contains(completeResp.Header().Get("HX-Trigger"), "followupsChanged") {
		t.Fatalf("complete handler should not emit followupsChanged trigger: %s", completeResp.Header().Get("HX-Trigger"))
	}

	// Arquiva -> HX-Refresh: true
	archiveResp := performRequest(handler, http.MethodPost, "/followups/"+strconv.FormatInt(id, 10)+"/archive", nil)
	if archiveResp.Code != http.StatusOK || archiveResp.Header().Get("HX-Refresh") != "true" {
		t.Fatalf("expected HX-Refresh on archive, got code %d header %q", archiveResp.Code, archiveResp.Header().Get("HX-Refresh"))
	}
	if strings.Contains(archiveResp.Header().Get("HX-Trigger"), "followupsChanged") {
		t.Fatalf("archive handler should not emit followupsChanged trigger: %s", archiveResp.Header().Get("HX-Trigger"))
	}

	// Novo cliente órfão para exclusão -> HX-Refresh: true
	clientOrphan, err := store.createClient("Cliente Órfão", "(32) 99999-0002")
	if err != nil {
		t.Fatal(err)
	}
	orphanFollowUpID, err := store.createFollowUp(clientOrphan.ID, clientOrphan.Name, clientOrphan.Contact, "2026-08-12", "2026-08-13", "Para excluir", "", PriorityMedium, "", "", ClientResolutionExisting, "", "")
	if err != nil {
		t.Fatal(err)
	}

	deleteResp := performRequest(handler, http.MethodPost, "/followups/"+strconv.FormatInt(orphanFollowUpID, 10)+"/delete", nil)
	if deleteResp.Code != http.StatusOK || deleteResp.Header().Get("HX-Refresh") != "true" {
		t.Fatalf("delete status = %d, HX-Refresh = %q", deleteResp.Code, deleteResp.Header().Get("HX-Refresh"))
	}
	if strings.Contains(deleteResp.Header().Get("HX-Trigger"), "followupsChanged") {
		t.Fatalf("delete handler should not emit followupsChanged trigger: %s", deleteResp.Header().Get("HX-Trigger"))
	}
}

func TestReportsIncludeNotesInHTML(t *testing.T) {
	store, _, _ := newTestStore(t)
	client, err := store.createClient("Cliente Relatório", "(32) 99999-0003")
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.createFollowUp(client.ID, client.Name, client.Contact, "2026-08-12", "2026-08-13", "Descrição do exame", "", PriorityHigh, "Observação importante do relatório", "", ClientResolutionExisting, "", "")
	if err != nil {
		t.Fatal(err)
	}
	app, err := newApplication(store, mustLocation(t))
	if err != nil {
		t.Fatal(err)
	}

	response := performRequest(app.routes(), http.MethodGet, "/reports?status=PENDING", nil)
	assertResponseContains(t, response, "Descrição do exame", "Observação importante do relatório")
}

func TestExistingClientDuplicatePhoneHTTPWorkflow(t *testing.T) {
	store, _, _ := newTestStore(t)
	cA, _ := store.createClient("Cliente Alfa", "(32) 99999-1111")
	cB, _ := store.createClient("Cliente Beta", "(32) 99999-2222")

	app, err := newApplication(store, mustLocation(t))
	if err != nil {
		t.Fatal(err)
	}
	handler := app.routes()

	// 1. Verificação prévia no estágio Telefone sem phone_change_action -> retorna confirmação de alteração PRIMEIRO
	verifyResp := performRequest(handler, http.MethodGet, "/clients/phone-change-confirmation?client_id="+strconv.FormatInt(cA.ID, 10)+"&client_name="+url.QueryEscape(cA.Name)+"&contact="+url.QueryEscape(cB.Contact), nil)
	if verifyResp.Code != http.StatusOK {
		t.Fatalf("verify status = %d", verifyResp.Code)
	}
	assertBodyContains(t, verifyResp.Body.String(), "já possui cadastro com", "Atualizar telefone deste cliente", "Cadastrar outro cliente com este nome")

	// 2. Com phone_change_action="update" -> agora retorna confirmação de duplicidade com token para cA.ID
	verifyResp = performRequest(handler, http.MethodGet, "/clients/phone-change-confirmation?client_id="+strconv.FormatInt(cA.ID, 10)+"&client_name="+url.QueryEscape(cA.Name)+"&contact="+url.QueryEscape(cB.Contact)+"&phone_change_action=update", nil)
	if verifyResp.Code != http.StatusOK {
		t.Fatalf("verify update duplicate status = %d", verifyResp.Code)
	}
	assertBodyContains(t, verifyResp.Body.String(), "Telefone já utilizado", "Cliente Beta")
	tokenUpdate := extractTokenFromBody(t, verifyResp.Body.String())

	// 3. Com phone_change_action="update", allow e tokenUpdate -> Telefone totalmente resolvido (corpo VAZIO)
	verifyResp = performRequest(handler, http.MethodGet, "/clients/phone-change-confirmation?client_id="+strconv.FormatInt(cA.ID, 10)+"&client_name="+url.QueryEscape(cA.Name)+"&contact="+url.QueryEscape(cB.Contact)+"&phone_change_action=update&duplicate_phone_decision=allow&duplicate_phone_token="+tokenUpdate, nil)
	if verifyResp.Code != http.StatusOK || strings.TrimSpace(verifyResp.Body.String()) != "" {
		t.Fatalf("expected empty response for fully resolved phone, got: %s", verifyResp.Body.String())
	}

	// 4. Com phone_change_action="new_client" mas reutilizando tokenUpdate -> duplicidade não resolvida (token para new_client difere)
	verifyResp = performRequest(handler, http.MethodGet, "/clients/phone-change-confirmation?client_id="+strconv.FormatInt(cA.ID, 10)+"&client_name="+url.QueryEscape(cA.Name)+"&contact="+url.QueryEscape(cB.Contact)+"&phone_change_action=new_client&duplicate_phone_decision=allow&duplicate_phone_token="+tokenUpdate, nil)
	if verifyResp.Code != http.StatusOK {
		t.Fatalf("verify new_client with update token status = %d", verifyResp.Code)
	}
	assertBodyContains(t, verifyResp.Body.String(), "Telefone já utilizado", "Cliente Beta")
	tokenNew := extractTokenFromBody(t, verifyResp.Body.String())
	if tokenNew == tokenUpdate {
		t.Fatalf("token for new_client (%s) should differ from update token (%s)", tokenNew, tokenUpdate)
	}

	// 5. POST com token legítimo de update e phone_change_action="update" -> 200 OK e followupSaved
	form := url.Values{
		"client_id":                {strconv.FormatInt(cA.ID, 10)},
		"client_name":              {cA.Name},
		"client_resolution":        {ClientResolutionExisting},
		"contact":                  {cB.Contact},
		"start_date":               {"2026-08-12"},
		"due_date":                 {"2026-08-13"},
		"priority":                 {PriorityHigh},
		"description":              {"Retorno com dupla confirmação"},
		"phone_change_action":      {"update"},
		"duplicate_phone_decision": {"allow"},
		"duplicate_phone_token":    {tokenUpdate},
	}
	response := performRequest(handler, http.MethodPost, "/followups", form)
	if response.Code != http.StatusOK || response.Header().Get("HX-Refresh") != "true" {
		t.Fatalf("expected 200 OK with HX-Refresh, got %d (header: %q, body: %s)", response.Code, response.Header().Get("HX-Refresh"), response.Body.String())
	}
}

func TestExistingClientNonDuplicatePhoneHTTPWorkflow(t *testing.T) {
	store, _, _ := newTestStore(t)
	cA, _ := store.createClient("Cliente Único", "(32) 99999-1111")

	app, err := newApplication(store, mustLocation(t))
	if err != nil {
		t.Fatal(err)
	}
	handler := app.routes()

	newPhone := "(32) 99999-3333"

	// 1. Sem phone_change_action -> exige alteração de telefone
	verifyResp := performRequest(handler, http.MethodGet, "/clients/phone-change-confirmation?client_id="+strconv.FormatInt(cA.ID, 10)+"&client_name="+url.QueryEscape(cA.Name)+"&contact="+url.QueryEscape(newPhone), nil)
	if verifyResp.Code != http.StatusOK {
		t.Fatalf("verify status = %d", verifyResp.Code)
	}
	assertBodyContains(t, verifyResp.Body.String(), "já possui cadastro com", "Atualizar telefone deste cliente")

	// 2. Com phone_change_action="update" e telefone não duplicado -> resolvido (retorna vazio)
	verifyResp = performRequest(handler, http.MethodGet, "/clients/phone-change-confirmation?client_id="+strconv.FormatInt(cA.ID, 10)+"&client_name="+url.QueryEscape(cA.Name)+"&contact="+url.QueryEscape(newPhone)+"&phone_change_action=update", nil)
	if verifyResp.Code != http.StatusOK || strings.TrimSpace(verifyResp.Body.String()) != "" {
		t.Fatalf("expected empty response for non-duplicate updated phone, got: %s", verifyResp.Body.String())
	}

	// 3. POST final com phone_change_action="update" -> 200 OK e HX-Refresh: true
	form := url.Values{
		"client_id":           {strconv.FormatInt(cA.ID, 10)},
		"client_name":         {cA.Name},
		"client_resolution":   {ClientResolutionExisting},
		"contact":             {newPhone},
		"start_date":          {"2026-08-12"},
		"due_date":            {"2026-08-13"},
		"priority":            {PriorityHigh},
		"description":         {"Retorno com telefone novo não duplicado"},
		"phone_change_action": {"update"},
	}
	response := performRequest(handler, http.MethodPost, "/followups", form)
	if response.Code != http.StatusOK || response.Header().Get("HX-Refresh") != "true" {
		t.Fatalf("expected 200 OK with HX-Refresh, got %d (header: %q, body: %s)", response.Code, response.Header().Get("HX-Refresh"), response.Body.String())
	}
}

func TestDueDateValidationHTTP(t *testing.T) {
	store, _, _ := newTestStore(t)
	client, err := store.createClient("Cliente Datas", "(32) 99999-0001")
	if err != nil {
		t.Fatal(err)
	}
	app, err := newApplication(store, mustLocation(t))
	if err != nil {
		t.Fatal(err)
	}
	handler := app.routes()

	// Tentativa de POST /followups com due_date anterior a start_date -> 400 Bad Request
	form := url.Values{
		"client_id":         {strconv.FormatInt(client.ID, 10)},
		"client_name":       {client.Name},
		"client_resolution": {ClientResolutionExisting},
		"contact":           {client.Contact},
		"start_date":        {"2026-08-15"},
		"due_date":          {"2026-08-10"},
		"priority":          {PriorityHigh},
		"description":       {"Retorno inválido"},
	}
	response := performRequest(handler, http.MethodPost, "/followups", form)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", response.Code)
	}
	assertBodyContains(t, response.Body.String(), "a data limite não pode ser anterior à data de início")

	// Criação válida para teste de edição
	form.Set("due_date", "2026-08-20")
	response = performRequest(handler, http.MethodPost, "/followups", form)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", response.Code)
	}

	followUps, err := store.clientFollowUps(client.ID)
	if err != nil || len(followUps) != 1 {
		t.Fatalf("expected 1 followup, got %d", len(followUps))
	}
	followUpID := followUps[0].ID

	// Tentativa de POST /followups/{id}/edit com due_date anterior a start_date -> 400 Bad Request
	editForm := url.Values{
		"start_date":  {"2026-08-25"},
		"due_date":    {"2026-08-20"},
		"priority":    {PriorityHigh},
		"description": {"Edição com data inválida"},
	}
	response = performRequest(handler, http.MethodPost, "/followups/"+strconv.FormatInt(followUpID, 10)+"/edit", editForm)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request on edit, got %d", response.Code)
	}
	assertBodyContains(t, response.Body.String(), "a data limite não pode ser anterior à data de início")
}

func TestDashboardContentFragmentAndNoNotes(t *testing.T) {
	store, _, _ := newTestStore(t)
	client, err := store.createClient("Cliente Dashboard", "(32) 99999-0001")
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.createFollowUp(client.ID, client.Name, client.Contact, "2026-08-10", "2026-08-15", "Pendência com nota", "", PriorityMedium, "Nota que não deve aparecer na tabela principal", "", ClientResolutionExisting, "", "")
	if err != nil {
		t.Fatal(err)
	}
	app, err := newApplication(store, mustLocation(t))
	if err != nil {
		t.Fatal(err)
	}
	handler := app.routes()

	// 1. GET / (dashboard completo)
	respPage := performRequest(handler, http.MethodGet, "/", nil)
	if respPage.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for GET /, got %d", respPage.Code)
	}
	pageBody := respPage.Body.String()
	if strings.Contains(pageBody, "followup-notes") || strings.Contains(pageBody, "Nota que não deve aparecer na tabela principal") {
		t.Fatalf("dashboard table should not render notes (must stay compact): %s", pageBody)
	}

	// 2. GET /dashboard (fragmento)
	respFragment := performRequest(handler, http.MethodGet, "/dashboard", nil)
	if respFragment.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for GET /dashboard, got %d", respFragment.Code)
	}
	fragBody := respFragment.Body.String()
	if !strings.Contains(fragBody, `id="dashboard-content"`) {
		t.Fatalf("fragment must contain #dashboard-content: %s", fragBody)
	}
}

func TestFollowUpFormDateFieldsAndID(t *testing.T) {
	store, _, _ := newTestStore(t)
	app, err := newApplication(store, mustLocation(t))
	if err != nil {
		t.Fatal(err)
	}
	handler := app.routes()

	resp := performRequest(handler, http.MethodGet, "/followups/new", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /followups/new, got %d", resp.Code)
	}
	body := resp.Body.String()
	if !strings.Contains(body, `id="followup-start-date"`) {
		t.Fatalf("form must contain id=\"followup-start-date\": %s", body)
	}
	if !strings.Contains(body, `data-date-input`) {
		t.Fatalf("form must contain data-date-input: %s", body)
	}
	if !strings.Contains(body, `name="start_date"`) || !strings.Contains(body, `name="due_date"`) {
		t.Fatalf("form must contain hidden inputs with name start_date and due_date: %s", body)
	}
}

func TestReportsDateFieldsAndTriggers(t *testing.T) {
	store, _, _ := newTestStore(t)
	app, err := newApplication(store, mustLocation(t))
	if err != nil {
		t.Fatal(err)
	}
	handler := app.routes()

	resp := performRequest(handler, http.MethodGet, "/reports", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /reports, got %d", resp.Code)
	}
	body := resp.Body.String()
	if !strings.Contains(body, `data-date-input`) {
		t.Fatalf("reports must use data-date-input: %s", body)
	}
	if !strings.Contains(body, `name="date_from"`) || !strings.Contains(body, `name="date_to"`) {
		t.Fatalf("reports must contain hidden inputs date_from and date_to: %s", body)
	}
	if !strings.Contains(body, `from:[name=client]`) || !strings.Contains(body, `from:[name=forward_to]`) {
		t.Fatalf("reports form must use targeted input triggers: %s", body)
	}
}

func TestHXRefreshOnSuccessAndNoRefreshOnFailure(t *testing.T) {
	store, _, _ := newTestStore(t)
	client, err := store.createClient("Cliente Refresh", "(32) 99999-0099")
	if err != nil {
		t.Fatal(err)
	}
	id, err := store.createFollowUp(client.ID, client.Name, client.Contact, "2026-08-10", "2026-08-15", "Pendência", "", PriorityMedium, "", "", ClientResolutionExisting, "", "")
	if err != nil {
		t.Fatal(err)
	}
	app, err := newApplication(store, mustLocation(t))
	if err != nil {
		t.Fatal(err)
	}
	handler := app.routes()

	// 1. Sucesso no complete -> 200 OK com HX-Refresh: true
	res := performRequest(handler, http.MethodPost, "/followups/"+strconv.FormatInt(id, 10)+"/complete", nil)
	if res.Code != http.StatusOK || res.Header().Get("HX-Refresh") != "true" {
		t.Fatalf("complete: expected 200 OK with HX-Refresh: true, got %d header %q", res.Code, res.Header().Get("HX-Refresh"))
	}

	// 2. Falha ao tentar complete novamente (transição inválida) -> 409 Conflict SEM HX-Refresh
	resFail := performRequest(handler, http.MethodPost, "/followups/"+strconv.FormatInt(id, 10)+"/complete", nil)
	if resFail.Code != http.StatusConflict || resFail.Header().Get("HX-Refresh") != "" {
		t.Fatalf("complete failure: expected 409 Conflict without HX-Refresh, got %d header %q", resFail.Code, resFail.Header().Get("HX-Refresh"))
	}

	// 3. Sucesso no archive -> 200 OK com HX-Refresh: true
	res = performRequest(handler, http.MethodPost, "/followups/"+strconv.FormatInt(id, 10)+"/archive", nil)
	if res.Code != http.StatusOK || res.Header().Get("HX-Refresh") != "true" {
		t.Fatalf("archive: expected 200 OK with HX-Refresh: true, got %d header %q", res.Code, res.Header().Get("HX-Refresh"))
	}

	// 4. Falha ao tentar archive novamente -> 409 Conflict SEM HX-Refresh
	resFail = performRequest(handler, http.MethodPost, "/followups/"+strconv.FormatInt(id, 10)+"/archive", nil)
	if resFail.Code != http.StatusConflict || resFail.Header().Get("HX-Refresh") != "" {
		t.Fatalf("archive failure: expected 409 Conflict without HX-Refresh, got %d header %q", resFail.Code, resFail.Header().Get("HX-Refresh"))
	}

	// 5. Falha ao tentar delete em pendência arquivada -> 409 Conflict SEM HX-Refresh
	resFail = performRequest(handler, http.MethodPost, "/followups/"+strconv.FormatInt(id, 10)+"/delete", nil)
	if resFail.Code != http.StatusConflict || resFail.Header().Get("HX-Refresh") != "" {
		t.Fatalf("delete failure: expected 409 Conflict without HX-Refresh, got %d header %q", resFail.Code, resFail.Header().Get("HX-Refresh"))
	}

	// 6. Sucesso em createFollowUp -> 200 OK com HX-Refresh: true
	newForm := url.Values{
		"client_name":       {"Cliente Teste Refresh"},
		"client_resolution": {ClientResolutionNew},
		"contact":           {"32999991234"},
		"start_date":        {"2026-08-12"},
		"due_date":          {"2026-08-13"},
		"priority":          {PriorityHigh},
		"description":       {"Teste refresh na criação"},
	}
	res = performRequest(handler, http.MethodPost, "/followups", newForm)
	if res.Code != http.StatusOK || res.Header().Get("HX-Refresh") != "true" {
		t.Fatalf("create: expected 200 OK with HX-Refresh: true, got %d header %q", res.Code, res.Header().Get("HX-Refresh"))
	}

	// 7. Falha em createFollowUp (data limite anterior à data de início) -> 400 Bad Request SEM HX-Refresh
	invalidForm := url.Values{
		"client_name":       {"Cliente Teste Falha"},
		"client_resolution": {ClientResolutionNew},
		"contact":           {"32999991235"},
		"start_date":        {"2026-08-15"},
		"due_date":          {"2026-08-10"},
		"priority":          {PriorityHigh},
		"description":       {"Teste falha sem refresh"},
	}
	resFail = performRequest(handler, http.MethodPost, "/followups", invalidForm)
	if resFail.Code != http.StatusBadRequest || resFail.Header().Get("HX-Refresh") != "" {
		t.Fatalf("create failure: expected 400 Bad Request without HX-Refresh, got %d header %q", resFail.Code, resFail.Header().Get("HX-Refresh"))
	}
}

func extractTokenFromBody(t *testing.T, body string) string {
	t.Helper()
	marker := `data-duplicate-token="`
	idx := strings.Index(body, marker)
	if idx == -1 {
		t.Fatalf("token marker not found in body: %s", body)
	}
	token := body[idx+len(marker):]
	endIdx := strings.Index(token, `"`)
	if endIdx == -1 {
		t.Fatalf("closing quote for token not found: %s", token)
	}
	return token[:endIdx]
}
