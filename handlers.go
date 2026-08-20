package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Application struct {
	store     *Store
	templates *template.Template
	location  *time.Location
	now       func() time.Time
}

type dashboardClientResult struct {
	Client
	ShowID bool
}

type clientMatch struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Contact string `json:"contact"`
}

func newApplication(store *Store, location *time.Location) (*Application, error) {
	functions := template.FuncMap{
		"formatDate":    formatDate,
		"priorityLabel": priorityLabel,
		"statusLabel":   statusLabel,
	}
	files, err := filepath.Glob("templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("listar templates: %w", err)
	}
	partials, err := filepath.Glob("templates/partials/*.html")
	if err != nil {
		return nil, fmt.Errorf("listar templates parciais: %w", err)
	}
	files = append(files, partials...)
	templates, err := template.New("client-followup").Funcs(functions).ParseFiles(files...)
	if err != nil {
		return nil, fmt.Errorf("carregar templates: %w", err)
	}
	return &Application{store: store, templates: templates, location: location, now: store.now}, nil
}

func (app *Application) routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("GET /health", app.health)
	mux.HandleFunc("GET /{$}", app.dashboardPage)
	mux.HandleFunc("GET /dashboard", app.dashboardContent)
	mux.HandleFunc("GET /dashboard/results", app.dashboardResults)
	mux.HandleFunc("GET /dashboard/metrics", app.dashboardMetrics)
	mux.HandleFunc("GET /followups/new", app.followUpForm)
	mux.HandleFunc("POST /followups", app.createFollowUp)
	mux.HandleFunc("GET /followups/{id}/edit", app.editFollowUpForm)
	mux.HandleFunc("POST /followups/{id}/edit", app.updateFollowUp)
	mux.HandleFunc("POST /followups/{id}/delete", app.deleteFollowUp)
	mux.HandleFunc("POST /followups/{id}/complete", app.completeFollowUp)
	mux.HandleFunc("POST /followups/{id}/reopen", app.reopenFollowUp)
	mux.HandleFunc("POST /followups/{id}/archive", app.archiveFollowUp)
	mux.HandleFunc("GET /clients/search", app.searchClients)
	mux.HandleFunc("GET /clients/exact", app.exactClients)
	mux.HandleFunc("GET /clients/phone-change-confirmation", app.phoneChangeConfirmation)
	mux.HandleFunc("GET /clients/{id}", app.clientDetail)
	mux.HandleFunc("GET /clients/{id}/edit", app.clientEditForm)
	mux.HandleFunc("POST /clients/{id}", app.updateClient)
	mux.HandleFunc("GET /reminders", app.reminders)
	mux.HandleFunc("GET /reports", app.reportsPage)
	mux.HandleFunc("GET /reports/results", app.reportResults)
	return app.recoverPanics(mux)
}

func (app *Application) recoverPanics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("erro inesperado em %s %s: %v", r.Method, r.URL.Path, recovered)
				http.Error(w, "Erro interno inesperado.", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (app *Application) render(w http.ResponseWriter, status int, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := app.templates.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("renderizar template %s: %v", name, err)
	}
}

func (app *Application) renderError(w http.ResponseWriter, message string, status int) {
	app.render(w, status, "message.html", struct {
		Kind    string
		Message string
	}{Kind: "error", Message: message})
}

func (app *Application) renderSuccess(w http.ResponseWriter, message string) {
	app.render(w, http.StatusOK, "message.html", struct {
		Kind    string
		Message string
	}{Kind: "success", Message: message})
}

func (app *Application) health(w http.ResponseWriter, _ *http.Request) {
	if err := app.store.db.Ping(); err != nil {
		http.Error(w, "database unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintln(w, "ok")
}

func (app *Application) dashboardPage(w http.ResponseWriter, r *http.Request) {
	view, err := app.dashboardView(r)
	if err != nil {
		app.renderError(w, "Não foi possível carregar o painel.", http.StatusInternalServerError)
		return
	}
	app.render(w, http.StatusOK, "dashboard.html", view)
}

func (app *Application) dashboardContent(w http.ResponseWriter, r *http.Request) {
	view, err := app.dashboardView(r)
	if err != nil {
		app.renderError(w, "Não foi possível atualizar o painel.", http.StatusInternalServerError)
		return
	}
	app.render(w, http.StatusOK, "dashboard-content.html", view)
}

func (app *Application) dashboardResults(w http.ResponseWriter, r *http.Request) {
	view, err := app.dashboardView(r)
	if err != nil {
		app.renderError(w, "Não foi possível atualizar o painel.", http.StatusInternalServerError)
		return
	}
	app.render(w, http.StatusOK, "dashboard-results.html", view)
}

func (app *Application) dashboardMetrics(w http.ResponseWriter, r *http.Request) {
	view, err := app.dashboardView(r)
	if err != nil {
		app.renderError(w, "Não foi possível atualizar os indicadores.", http.StatusInternalServerError)
		return
	}
	app.render(w, http.StatusOK, "dashboard-metrics.html", view)
}

func (app *Application) dashboardView(r *http.Request) (DashboardView, error) {
	filters := DashboardFilters{
		Query:    normalizeText(r.URL.Query().Get("q")),
		Priority: r.URL.Query().Get("priority"),
		Status:   r.URL.Query().Get("status"),
		Due:      r.URL.Query().Get("due"),
	}
	todayTime := dateOnly(app.now().In(app.location))
	today := todayTime.Format("2006-01-02")
	items, err := app.store.dashboardFollowUps(filters, today)
	if err != nil {
		return DashboardView{}, err
	}
	for index := range items {
		items[index].Alert = reminderFor(items[index], todayTime)
	}
	counts, err := app.store.dashboardCounts(today)
	if err != nil {
		return DashboardView{}, err
	}
	return DashboardView{Today: today, Filters: filters, Counts: counts, FollowUps: items}, nil
}

func (app *Application) followUpForm(w http.ResponseWriter, r *http.Request) {
	view := FollowUpFormView{
		Today:      app.now().In(app.location).Format("2006-01-02"),
		ClientName: normalizeText(r.URL.Query().Get("client_name")),
	}
	if value := r.URL.Query().Get("client_id"); value != "" {
		clientID, err := parseID(value)
		if err != nil {
			app.renderError(w, "Selecione novamente o cliente.", http.StatusBadRequest)
			return
		}
		client, err := app.store.getClient(clientID)
		if errors.Is(err, sql.ErrNoRows) {
			app.renderError(w, "Cliente não encontrado.", http.StatusNotFound)
			return
		}
		if err != nil {
			app.renderError(w, "Não foi possível abrir o cadastro da pendência.", http.StatusInternalServerError)
			return
		}
		view.ClientID = client.ID
		view.ClientName = client.Name
		view.Contact = client.Contact
		view.Resolved = true
	}
	app.render(w, http.StatusOK, "followup-form.html", view)
}

func (app *Application) createFollowUp(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		app.renderError(w, "Dados do formulário inválidos.", http.StatusBadRequest)
		return
	}
	clientID := int64(0)
	if value := r.FormValue("client_id"); value != "" {
		parsedID, err := parseID(value)
		if err != nil {
			app.renderError(w, "Selecione novamente o cliente.", http.StatusBadRequest)
			return
		}
		clientID = parsedID
	}

	_, err := app.store.createFollowUp(
		clientID,
		r.FormValue("client_name"),
		r.FormValue("contact"),
		r.FormValue("start_date"),
		r.FormValue("due_date"),
		r.FormValue("description"),
		r.FormValue("forward_to"),
		r.FormValue("priority"),
		r.FormValue("notes"),
		r.FormValue("phone_change_action"),
		r.FormValue("client_resolution"),
		r.FormValue("duplicate_phone_decision"),
		r.FormValue("duplicate_phone_token"),
	)
	if err != nil {
		var phoneDuplicate *clientDuplicatePhoneRequiredError
		if errors.As(err, &phoneDuplicate) {
			app.render(w, http.StatusConflict, "client-duplicate-phone-confirmation.html", ClientDuplicatePhoneConfirmationView{
				Name:              r.FormValue("client_name"),
				SubmittedPhone:    phoneDuplicate.SubmittedPhone,
				ExistingClients:   phoneDuplicate.ExistingClients,
				ConfirmationToken: phoneDuplicate.Token,
			})
			return
		}
		var phoneChange *clientPhoneChangeRequiredError
		if errors.As(err, &phoneChange) {
			app.render(w, http.StatusConflict, "client-phone-confirmation.html", ClientPhoneConfirmationView{
				Name:           phoneChange.Client.Name,
				CurrentPhone:   phoneChange.Client.Contact,
				SubmittedPhone: phoneChange.SubmittedPhone,
			})
			return
		}
		app.renderError(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("HX-Refresh", "true")
	w.WriteHeader(http.StatusOK)
}

func (app *Application) editFollowUpForm(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		app.renderError(w, err.Error(), http.StatusBadRequest)
		return
	}
	item, err := app.store.getFollowUp(id)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		app.renderError(w, "Não foi possível abrir a pendência.", http.StatusInternalServerError)
		return
	}
	if item.Status != StatusPending {
		app.renderError(w, "Somente pendências abertas podem ser editadas.", http.StatusConflict)
		return
	}
	app.render(w, http.StatusOK, "followup-edit.html", FollowUpEditView{FollowUp: item})
}

func (app *Application) updateFollowUp(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		app.renderError(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = app.store.updatePendingFollowUp(
		id,
		r.FormValue("start_date"),
		r.FormValue("due_date"),
		r.FormValue("description"),
		r.FormValue("forward_to"),
		r.FormValue("priority"),
		r.FormValue("notes"),
	)
	if errors.Is(err, errInvalidTransition) {
		app.renderError(w, "Somente pendências abertas podem ser editadas.", http.StatusConflict)
		return
	}
	if err != nil {
		app.renderError(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("HX-Refresh", "true")
	w.WriteHeader(http.StatusOK)
}

func (app *Application) deleteFollowUp(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		app.renderError(w, err.Error(), http.StatusBadRequest)
		return
	}
	_, err = app.store.deletePendingFollowUp(id)
	if errors.Is(err, errInvalidTransition) {
		app.renderError(w, "Somente pendências abertas podem ser excluídas.", http.StatusConflict)
		return
	}
	if err != nil {
		app.renderError(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("HX-Refresh", "true")
	w.WriteHeader(http.StatusOK)
}

func (app *Application) completeFollowUp(w http.ResponseWriter, r *http.Request) {
	app.changeFollowUpStatus(w, r, StatusPending, StatusCompleted)
}

func (app *Application) reopenFollowUp(w http.ResponseWriter, r *http.Request) {
	app.changeFollowUpStatus(w, r, StatusCompleted, StatusPending)
}

func (app *Application) archiveFollowUp(w http.ResponseWriter, r *http.Request) {
	app.changeFollowUpStatus(w, r, StatusCompleted, StatusArchived)
}

func (app *Application) changeFollowUpStatus(w http.ResponseWriter, r *http.Request, fromStatus, toStatus string) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		app.renderError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := app.store.transitionFollowUp(id, fromStatus, toStatus); err != nil {
		if errors.Is(err, errInvalidTransition) {
			app.renderError(w, "A pendência mudou de estado. Atualize a tela e tente novamente.", http.StatusConflict)
			return
		}
		app.renderError(w, "Não foi possível atualizar a pendência.", http.StatusInternalServerError)
		return
	}
	w.Header().Set("HX-Refresh", "true")
	w.WriteHeader(http.StatusOK)
}

func (app *Application) searchClients(w http.ResponseWriter, r *http.Request) {
	query := normalizeText(r.URL.Query().Get("client_name"))
	if query == "" {
		app.render(w, http.StatusOK, "client-search-results.html", []dashboardClientResult{})
		return
	}
	if _, err := validateClientName(query); err != nil {
		app.renderError(w, err.Error(), http.StatusBadRequest)
		return
	}
	clients, err := app.store.searchClients(query)
	if err != nil {
		app.renderError(w, "Não foi possível pesquisar clientes.", http.StatusInternalServerError)
		return
	}
	app.render(w, http.StatusOK, "client-search-results.html", dashboardClientResults(clients))
}

func (app *Application) exactClients(w http.ResponseWriter, r *http.Request) {
	name, err := validateClientName(r.URL.Query().Get("client_name"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	clients, err := app.store.findClientsByExactName(name)
	if err != nil {
		http.Error(w, "Não foi possível verificar o cliente.", http.StatusInternalServerError)
		return
	}
	matches := make([]clientMatch, 0, len(clients))
	for _, client := range clients {
		matches = append(matches, clientMatch{ID: client.ID, Name: client.Name, Contact: client.Contact})
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(struct {
		Clients []clientMatch `json:"clients"`
	}{Clients: matches}); err != nil {
		log.Printf("renderizar correspondências exatas: %v", err)
	}
}

func (app *Application) phoneChangeConfirmation(w http.ResponseWriter, r *http.Request) {
	clientID, _ := parseID(r.URL.Query().Get("client_id"))
	clientName := normalizeText(r.URL.Query().Get("client_name"))
	submittedPhone, err := normalizePhone(r.URL.Query().Get("contact"))
	if err != nil {
		w.WriteHeader(http.StatusOK)
		return
	}
	phoneChangeAction := r.URL.Query().Get("phone_change_action")
	duplicatePhoneDecision := r.URL.Query().Get("duplicate_phone_decision")
	duplicatePhoneToken := r.URL.Query().Get("duplicate_phone_token")

	if clientID > 0 {
		client, err := app.store.getClient(clientID)
		if err != nil {
			w.WriteHeader(http.StatusOK)
			return
		}
		if clientName == "" {
			clientName = client.Name
		}
		currentPhone, err := normalizePhone(client.Contact)
		if err == nil && currentPhone == submittedPhone {
			w.WriteHeader(http.StatusOK)
			return
		}

		switch phoneChangeAction {
		case PhoneChangeUpdate:
			conflicts, err := app.store.findClientsByPhone(submittedPhone, clientID)
			if err == nil && len(conflicts) > 0 {
				token := app.store.phoneConfirmationToken(submittedPhone, clientID, conflicts)
				if duplicatePhoneDecision != "allow" || duplicatePhoneToken != token {
					app.render(w, http.StatusOK, "client-duplicate-phone-confirmation.html", ClientDuplicatePhoneConfirmationView{
						Name:              clientName,
						SubmittedPhone:    submittedPhone,
						ExistingClients:   conflicts,
						ConfirmationToken: token,
						OriginalPhone:     client.Contact,
						IsClientEdit:      false,
					})
					return
				}
			}
			w.WriteHeader(http.StatusOK)
			return
		case PhoneChangeNewClient:
			conflicts, err := app.store.findClientsByPhone(submittedPhone, 0)
			if err == nil && len(conflicts) > 0 {
				token := app.store.phoneConfirmationToken(submittedPhone, 0, conflicts)
				if duplicatePhoneDecision != "allow" || duplicatePhoneToken != token {
					app.render(w, http.StatusOK, "client-duplicate-phone-confirmation.html", ClientDuplicatePhoneConfirmationView{
						Name:              clientName,
						SubmittedPhone:    submittedPhone,
						ExistingClients:   conflicts,
						ConfirmationToken: token,
						OriginalPhone:     client.Contact,
						IsClientEdit:      false,
					})
					return
				}
			}
			w.WriteHeader(http.StatusOK)
			return
		default:
			app.render(w, http.StatusOK, "client-phone-confirmation.html", ClientPhoneConfirmationView{
				Name:           client.Name,
				CurrentPhone:   client.Contact,
				SubmittedPhone: submittedPhone,
			})
			return
		}
	}

	conflicts, err := app.store.findClientsByPhone(submittedPhone, 0)
	if err == nil && len(conflicts) > 0 {
		token := app.store.phoneConfirmationToken(submittedPhone, 0, conflicts)
		if duplicatePhoneDecision != "allow" || duplicatePhoneToken != token {
			app.render(w, http.StatusOK, "client-duplicate-phone-confirmation.html", ClientDuplicatePhoneConfirmationView{
				Name:              clientName,
				SubmittedPhone:    submittedPhone,
				ExistingClients:   conflicts,
				ConfirmationToken: token,
				OriginalPhone:     "",
				IsClientEdit:      false,
			})
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

func dashboardClientResults(clients []Client) []dashboardClientResult {
	nameCounts := make(map[string]int, len(clients))
	for _, client := range clients {
		nameCounts[strings.ToLower(client.Name)]++
	}

	results := make([]dashboardClientResult, 0, len(clients))
	for _, client := range clients {
		results = append(results, dashboardClientResult{
			Client: client,
			ShowID: client.Contact == "" && nameCounts[strings.ToLower(client.Name)] > 1,
		})
	}
	return results
}

func (app *Application) clientDetail(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		app.renderError(w, err.Error(), http.StatusBadRequest)
		return
	}
	client, err := app.store.getClient(id)
	if errors.Is(err, sql.ErrNoRows) {
		w.Header().Set("HX-Trigger", `{"closeModal":true}`)
		w.WriteHeader(http.StatusOK)
		return
	}
	if err != nil {
		app.renderError(w, "Não foi possível abrir a ficha do cliente.", http.StatusInternalServerError)
		return
	}
	items, err := app.store.clientFollowUps(id)
	if err != nil {
		app.renderError(w, "Não foi possível carregar as pendências do cliente.", http.StatusInternalServerError)
		return
	}
	today := dateOnly(app.now().In(app.location))
	for index := range items {
		items[index].Alert = reminderFor(items[index], today)
	}
	highlightID, _ := strconv.ParseInt(r.URL.Query().Get("highlight"), 10, 64)
	app.render(w, http.StatusOK, "client-detail.html", ClientDetailView{
		Client: client, FollowUps: items, HighlightID: highlightID,
	})
}

func (app *Application) clientEditForm(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		app.renderError(w, err.Error(), http.StatusBadRequest)
		return
	}
	client, err := app.store.getClient(id)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		app.renderError(w, "Não foi possível editar o cliente.", http.StatusInternalServerError)
		return
	}
	app.render(w, http.StatusOK, "client-edit-form.html", client)
}

func (app *Application) updateClient(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		app.renderError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := app.store.updateClient(id, r.FormValue("name"), r.FormValue("contact"), r.FormValue("phone_change_confirmation"), r.FormValue("duplicate_phone_decision"), r.FormValue("duplicate_phone_token")); err != nil {
		var phoneDuplicate *clientEditDuplicatePhoneRequiredError
		if errors.As(err, &phoneDuplicate) {
			app.render(w, http.StatusConflict, "client-duplicate-phone-confirmation.html", ClientDuplicatePhoneConfirmationView{
				Name:              phoneDuplicate.Client.Name,
				SubmittedPhone:    phoneDuplicate.SubmittedPhone,
				ExistingClients:   phoneDuplicate.ExistingClients,
				ConfirmationToken: phoneDuplicate.Token,
				OriginalPhone:     phoneDuplicate.Client.Contact,
				IsClientEdit:      true,
			})
			return
		}
		var phoneChange *clientEditPhoneChangeRequiredError
		if errors.As(err, &phoneChange) {
			app.render(w, http.StatusConflict, "client-edit-phone-confirmation.html", ClientEditPhoneConfirmationView{
				Name:           phoneChange.Client.Name,
				CurrentPhone:   phoneChange.Client.Contact,
				SubmittedPhone: phoneChange.SubmittedPhone,
			})
			return
		}
		app.renderError(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("HX-Trigger", `{"followupsChanged":true,"clientChanged":true}`)
	app.renderSuccess(w, "Dados do cliente atualizados.")
}

func (app *Application) reminders(w http.ResponseWriter, _ *http.Request) {
	items, err := app.store.reminders()
	if err != nil {
		app.renderError(w, "Não foi possível carregar os alertas.", http.StatusInternalServerError)
		return
	}
	items = addReminders(items, app.now().In(app.location))
	if len(items) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	app.render(w, http.StatusOK, "reminders.html", ReminderView{Items: items})
}

func (app *Application) reportsPage(w http.ResponseWriter, r *http.Request) {
	view, err := app.reportsView(r)
	if err != nil {
		app.renderError(w, "Não foi possível gerar a consulta.", http.StatusInternalServerError)
		return
	}
	app.render(w, http.StatusOK, "reports.html", view)
}

func (app *Application) reportResults(w http.ResponseWriter, r *http.Request) {
	view, err := app.reportsView(r)
	if err != nil {
		app.renderError(w, "Não foi possível atualizar a consulta.", http.StatusInternalServerError)
		return
	}
	app.render(w, http.StatusOK, "report-results.html", view)
}

func (app *Application) reportsView(r *http.Request) (ReportsView, error) {
	filters := ReportFilters{
		DateFrom:  r.URL.Query().Get("date_from"),
		DateTo:    r.URL.Query().Get("date_to"),
		Client:    normalizeText(r.URL.Query().Get("client")),
		ForwardTo: normalizeText(r.URL.Query().Get("forward_to")),
		Priority:  r.URL.Query().Get("priority"),
		Status:    r.URL.Query().Get("status"),
		Overdue:   r.URL.Query().Get("overdue") == "1",
	}
	now := app.now().In(app.location)
	results, err := app.store.reportFollowUps(filters, now.Format("2006-01-02"))
	if err != nil {
		return ReportsView{}, err
	}
	return ReportsView{IssuedAt: now.Format("02/01/2006 15:04"), Filters: filters, Results: results}, nil
}

func formatDate(value string) string {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return value
	}
	return parsed.Format("02/01/2006")
}
