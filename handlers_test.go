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
	client, err := store.createClient(`<script>alert("client")</script>`, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.createFollowUp(client.ID, client.Name, "", "2026-08-12", "2026-08-13", `<img src=x onerror=alert("description")>`, "", PriorityMedium, ""); err != nil {
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
		"contact":     {"contato@example.com"},
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
	assertResponseContains(t, response, "Cliente integração", "contato@example.com")
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
	withContact, err := store.createClient("Ana Silva", "ana@example.com")
	if err != nil {
		t.Fatal(err)
	}
	withoutContact, err := store.createClient("Ana Silva", "")
	if err != nil {
		t.Fatal(err)
	}
	app, err := newApplication(store, mustLocation(t))
	if err != nil {
		t.Fatal(err)
	}

	response := performRequest(app.routes(), http.MethodGet, "/clients/search?client_name=Ana", nil)
	assertResponseContains(
		t,
		response,
		`data-client-id="`+strconv.FormatInt(withContact.ID, 10)+`"`,
		"ana@example.com",
		`data-client-id="`+strconv.FormatInt(withoutContact.ID, 10)+`"`,
		"Cadastro #"+strconv.FormatInt(withoutContact.ID, 10),
	)
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
