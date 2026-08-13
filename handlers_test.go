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
	if _, err := store.createFollowUp(client.ID, client.Name, client.Contact, "2026-08-12", "2026-08-13", `<img src=x onerror=alert("description")>`, "", PriorityMedium, "", ""); err != nil {
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
		"client_name": {"Cliente integração"},
		"contact":     {"32999990005"},
		"start_date":  {"2026-08-12"},
		"due_date":    {"2026-08-13"},
		"priority":    {PriorityHigh},
		"description": {"Confirmar retorno"},
		"forward_to":  {"Equipe A"},
		"notes":       {"Administrativo"},
	}
	response := performRequest(handler, http.MethodPost, "/followups", form)
	if response.Code != http.StatusOK || !strings.Contains(response.Header().Get("HX-Trigger"), "followupsChanged") {
		t.Fatalf("create response = %d, trigger %q, body %s", response.Code, response.Header().Get("HX-Trigger"), response.Body.String())
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
	assertResponseContains(t, response, "finalizada")
	response = performRequest(handler, http.MethodPost, "/followups/"+strconv.FormatInt(followUpID, 10)+"/reopen", nil)
	assertResponseContains(t, response, "reaberta")
	performRequest(handler, http.MethodPost, "/followups/"+strconv.FormatInt(followUpID, 10)+"/complete", nil)
	response = performRequest(handler, http.MethodPost, "/followups/"+strconv.FormatInt(followUpID, 10)+"/archive", nil)
	assertResponseContains(t, response, "arquivada")

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
		`data-client-id="`+strconv.FormatInt(withContact.ID, 10)+`"`,
		"(32) 99999-1111",
		`data-client-id="`+strconv.FormatInt(withoutContact.ID, 10)+`"`,
		"Cadastro #"+strconv.FormatInt(withoutContact.ID, 10),
	)
}

func TestDashboardSearchUpdatesOnlyResults(t *testing.T) {
	store, _, _ := newTestStore(t)
	app, err := newApplication(store, mustLocation(t))
	if err != nil {
		t.Fatal(err)
	}

	response := performRequest(app.routes(), http.MethodGet, "/", nil)
	assertResponseContains(
		t,
		response,
		`id="dashboard-filters"`,
		`hx-get="/dashboard/results"`,
		`hx-target="#dashboard-results"`,
		`hx-trigger="input, search, change"`,
	)

	response = performRequest(app.routes(), http.MethodGet, "/dashboard/results?q=Ana", nil)
	if strings.Contains(response.Body.String(), `id="dashboard-filters"`) {
		t.Fatalf("results response replaced the fixed filter form: %s", response.Body.String())
	}
	assertResponseContains(t, response, `id="dashboard-results"`, `q=Ana`)
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
			`name="phone_change_action" value="update"`,
			`name="phone_change_action" value="new_client"`,
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

func followUpFormValues(client Client, phone string) url.Values {
	values := url.Values{
		"client_name": {"Nova Cliente"},
		"contact":     {phone},
		"start_date":  {"2026-08-12"},
		"due_date":    {"2026-08-13"},
		"priority":    {PriorityMedium},
		"description": {"Confirmar retorno"},
	}
	if client.ID > 0 {
		values.Set("client_id", strconv.FormatInt(client.ID, 10))
		values.Set("client_name", client.Name)
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
