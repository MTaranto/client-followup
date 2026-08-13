package main

import (
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	PriorityHigh   = "HIGH"
	PriorityMedium = "MEDIUM"
	PriorityLow    = "LOW"

	StatusPending   = "PENDING"
	StatusCompleted = "COMPLETED"
	StatusArchived  = "ARCHIVED"

	PhoneChangeUpdate    = "update"
	PhoneChangeNewClient = "new_client"
)

type Client struct {
	ID        int64
	Name      string
	Contact   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type FollowUp struct {
	ID          int64
	ClientID    int64
	ClientName  string
	StartDate   string
	DueDate     string
	Description string
	ForwardTo   string
	Priority    string
	Status      string
	Notes       string
	CompletedAt sql.NullTime
	ArchivedAt  sql.NullTime
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Alert       Reminder
}

type Reminder struct {
	Kind     string
	Label    string
	DaysLate int
}

type DashboardFilters struct {
	Query    string
	Priority string
	Status   string
	Due      string
}

func (filters DashboardFilters) Values() url.Values {
	values := url.Values{}
	values.Set("q", filters.Query)
	values.Set("priority", filters.Priority)
	values.Set("status", filters.Status)
	values.Set("due", filters.Due)
	return values
}

type ReportFilters struct {
	DateFrom  string
	DateTo    string
	Client    string
	ForwardTo string
	Priority  string
	Status    string
	Overdue   bool
}

func (filters ReportFilters) Values() url.Values {
	values := url.Values{}
	values.Set("date_from", filters.DateFrom)
	values.Set("date_to", filters.DateTo)
	values.Set("client", filters.Client)
	values.Set("forward_to", filters.ForwardTo)
	values.Set("priority", filters.Priority)
	values.Set("status", filters.Status)
	if filters.Overdue {
		values.Set("overdue", "1")
	}
	return values
}

type DashboardCounts struct {
	Overdue   int
	Today     int
	Tomorrow  int
	Pending   int
	Completed int
}

type DashboardView struct {
	Today     string
	Filters   DashboardFilters
	Counts    DashboardCounts
	FollowUps []FollowUp
}

type FollowUpFormView struct {
	Today string
}

type ClientPhoneConfirmationView struct {
	Name           string
	CurrentPhone   string
	SubmittedPhone string
}

type ClientDetailView struct {
	Client      Client
	FollowUps   []FollowUp
	HighlightID int64
}

type ReportsView struct {
	IssuedAt string
	Filters  ReportFilters
	Results  []FollowUp
}

type ReminderView struct {
	Items []FollowUp
}

func validPriority(priority string) bool {
	return priority == PriorityHigh || priority == PriorityMedium || priority == PriorityLow
}

func validStatus(status string) bool {
	return status == StatusPending || status == StatusCompleted || status == StatusArchived
}

func priorityLabel(priority string) string {
	switch priority {
	case PriorityHigh:
		return "Alta"
	case PriorityMedium:
		return "Média"
	case PriorityLow:
		return "Baixa"
	default:
		return priority
	}
}

func statusLabel(status string) string {
	switch status {
	case StatusPending:
		return "Pendente"
	case StatusCompleted:
		return "Finalizado"
	case StatusArchived:
		return "Arquivado"
	default:
		return status
	}
}

func parseID(value string) (int64, error) {
	var id int64
	if _, err := fmt.Sscan(value, &id); err != nil || id <= 0 {
		return 0, fmt.Errorf("identificador inválido")
	}
	return id, nil
}

func normalizeText(value string) string {
	return strings.TrimSpace(value)
}
