package main

import (
	"database/sql"
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

type clientSuggestion struct {
	Client
	ShowID bool
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
	mux.HandleFunc("GET /followups/new", app.followUpForm)
	mux.HandleFunc("POST /followups", app.createFollowUp)
	mux.HandleFunc("POST /followups/{id}/complete", app.completeFollowUp)
	mux.HandleFunc("POST /followups/{id}/reopen", app.reopenFollowUp)
	mux.HandleFunc("POST /followups/{id}/archive", app.archiveFollowUp)
	mux.HandleFunc("GET /clients/search", app.searchClients)
	mux.HandleFunc("GET /clients/{id}", app.clientDetail)
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

func (app *Application) followUpForm(w http.ResponseWriter, _ *http.Request) {
	app.render(w, http.StatusOK, "followup-form.html", FollowUpFormView{
		Today: app.now().In(app.location).Format("2006-01-02"),
	})
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
			app.renderError(w, "Selecione novamente a cliente.", http.StatusBadRequest)
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
	)
	if err != nil {
		app.renderError(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("HX-Trigger", `{"followupsChanged":true,"closeModal":true}`)
	app.renderSuccess(w, "Pendência cadastrada com sucesso.")
}

func (app *Application) completeFollowUp(w http.ResponseWriter, r *http.Request) {
	app.changeFollowUpStatus(w, r, StatusPending, StatusCompleted, "Pendência finalizada.")
}

func (app *Application) reopenFollowUp(w http.ResponseWriter, r *http.Request) {
	app.changeFollowUpStatus(w, r, StatusCompleted, StatusPending, "Pendência reaberta.")
}

func (app *Application) archiveFollowUp(w http.ResponseWriter, r *http.Request) {
	app.changeFollowUpStatus(w, r, StatusCompleted, StatusArchived, "Pendência arquivada.")
}

func (app *Application) changeFollowUpStatus(w http.ResponseWriter, r *http.Request, fromStatus, toStatus, message string) {
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
	w.Header().Set("HX-Trigger", `{"followupsChanged":true}`)
	app.renderSuccess(w, message)
}

func (app *Application) searchClients(w http.ResponseWriter, r *http.Request) {
	query := normalizeText(r.URL.Query().Get("client_name"))
	if query == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	clients, err := app.store.searchClients(query)
	if err != nil {
		app.renderError(w, "Não foi possível pesquisar clientes.", http.StatusInternalServerError)
		return
	}
	app.render(w, http.StatusOK, "client-suggestions.html", clientSuggestions(clients))
}

func clientSuggestions(clients []Client) []clientSuggestion {
	nameCounts := make(map[string]int, len(clients))
	for _, client := range clients {
		nameCounts[strings.ToLower(client.Name)]++
	}

	suggestions := make([]clientSuggestion, 0, len(clients))
	for _, client := range clients {
		suggestions = append(suggestions, clientSuggestion{
			Client: client,
			ShowID: client.Contact == "" && nameCounts[strings.ToLower(client.Name)] > 1,
		})
	}
	return suggestions
}

func (app *Application) clientDetail(w http.ResponseWriter, r *http.Request) {
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
		app.renderError(w, "Não foi possível abrir a ficha da cliente.", http.StatusInternalServerError)
		return
	}
	items, err := app.store.clientFollowUps(id)
	if err != nil {
		app.renderError(w, "Não foi possível carregar as pendências da cliente.", http.StatusInternalServerError)
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

func (app *Application) updateClient(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		app.renderError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := app.store.updateClient(id, r.FormValue("name"), r.FormValue("contact")); err != nil {
		app.renderError(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("HX-Trigger", `{"followupsChanged":true}`)
	app.renderSuccess(w, "Dados da cliente atualizados.")
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
