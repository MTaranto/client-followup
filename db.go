package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var errInvalidTransition = errors.New("transição de status inválida")

type Store struct {
	db  *sql.DB
	now func() time.Time
}

func openStore(databasePath string, now func() time.Time) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(databasePath), 0o700); err != nil {
		return nil, fmt.Errorf("criar diretório de dados: %w", err)
	}

	dsn := databasePath + "?_foreign_keys=on&_busy_timeout=5000&_journal_mode=WAL"
	database, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("abrir banco de dados: %w", err)
	}
	database.SetMaxOpenConns(1)

	store := &Store{db: database, now: now}
	if err := store.migrate(); err != nil {
		database.Close()
		return nil, err
	}
	if err := os.Chmod(databasePath, 0o600); err != nil {
		database.Close()
		return nil, fmt.Errorf("proteger permissões do banco de dados: %w", err)
	}
	return store, nil
}

func (store *Store) Close() error {
	return store.db.Close()
}

func (store *Store) migrate() error {
	var version int
	if err := store.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("ler versão do schema: %w", err)
	}

	migrations := []string{
		`CREATE TABLE clients (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL COLLATE NOCASE,
			contact TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		);
		CREATE TABLE followups (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			client_id INTEGER NOT NULL REFERENCES clients(id),
			start_date TEXT NOT NULL,
			due_date TEXT NOT NULL,
			description TEXT NOT NULL,
			forward_to TEXT NOT NULL DEFAULT '',
			priority TEXT NOT NULL DEFAULT 'MEDIUM' CHECK (priority IN ('HIGH', 'MEDIUM', 'LOW')),
			status TEXT NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'COMPLETED', 'ARCHIVED')),
			notes TEXT NOT NULL DEFAULT '',
			completed_at DATETIME,
			archived_at DATETIME,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		);
		CREATE INDEX followups_client_id_idx ON followups(client_id);
		CREATE INDEX followups_status_due_date_idx ON followups(status, due_date);
		CREATE INDEX clients_name_idx ON clients(name);`,
	}

	for index := version; index < len(migrations); index++ {
		transaction, err := store.db.Begin()
		if err != nil {
			return fmt.Errorf("iniciar migração %d: %w", index+1, err)
		}
		if _, err := transaction.Exec(migrations[index]); err != nil {
			transaction.Rollback()
			return fmt.Errorf("executar migração %d: %w", index+1, err)
		}
		if _, err := transaction.Exec(fmt.Sprintf("PRAGMA user_version = %d", index+1)); err != nil {
			transaction.Rollback()
			return fmt.Errorf("registrar migração %d: %w", index+1, err)
		}
		if err := transaction.Commit(); err != nil {
			return fmt.Errorf("confirmar migração %d: %w", index+1, err)
		}
	}
	return nil
}

func (store *Store) createClient(name, contact string) (Client, error) {
	name = normalizeText(name)
	contact = normalizeText(contact)
	if name == "" {
		return Client{}, errors.New("o nome da cliente é obrigatório")
	}

	now := store.now()
	result, err := store.db.Exec(
		"INSERT INTO clients (name, contact, created_at, updated_at) VALUES (?, ?, ?, ?)",
		name, contact, now, now,
	)
	if err != nil {
		return Client{}, fmt.Errorf("cadastrar cliente: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Client{}, fmt.Errorf("obter cliente cadastrada: %w", err)
	}
	return Client{ID: id, Name: name, Contact: contact, CreatedAt: now, UpdatedAt: now}, nil
}

func (store *Store) findOrCreateClient(transaction *sql.Tx, clientID int64, name, contact string) (int64, error) {
	name = normalizeText(name)
	contact = normalizeText(contact)
	if name == "" {
		return 0, errors.New("o nome da cliente é obrigatório")
	}

	if clientID > 0 {
		var existingID int64
		if err := transaction.QueryRow("SELECT id FROM clients WHERE id = ?", clientID).Scan(&existingID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return 0, errors.New("cliente selecionada não foi encontrada")
			}
			return 0, fmt.Errorf("consultar cliente: %w", err)
		}
		if _, err := transaction.Exec("UPDATE clients SET contact = ?, updated_at = ? WHERE id = ?", contact, store.now(), clientID); err != nil {
			return 0, fmt.Errorf("atualizar contato da cliente: %w", err)
		}
		return clientID, nil
	}

	now := store.now()
	result, err := transaction.Exec(
		"INSERT INTO clients (name, contact, created_at, updated_at) VALUES (?, ?, ?, ?)",
		name, contact, now, now,
	)
	if err != nil {
		return 0, fmt.Errorf("cadastrar cliente: %w", err)
	}
	return result.LastInsertId()
}

func (store *Store) createFollowUp(clientID int64, clientName, contact, startDate, dueDate, description, forwardTo, priority, notes string) (int64, error) {
	if _, err := time.Parse("2006-01-02", startDate); err != nil {
		return 0, errors.New("a data de início é inválida")
	}
	if _, err := time.Parse("2006-01-02", dueDate); err != nil {
		return 0, errors.New("a data limite é obrigatória e deve ser válida")
	}
	description = normalizeText(description)
	if description == "" {
		return 0, errors.New("a descrição é obrigatória")
	}
	if !validPriority(priority) {
		return 0, errors.New("a prioridade é inválida")
	}

	transaction, err := store.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("iniciar cadastro: %w", err)
	}
	defer transaction.Rollback()

	resolvedClientID, err := store.findOrCreateClient(transaction, clientID, clientName, contact)
	if err != nil {
		return 0, err
	}
	now := store.now()
	result, err := transaction.Exec(`INSERT INTO followups
		(client_id, start_date, due_date, description, forward_to, priority, status, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		resolvedClientID, startDate, dueDate, description, normalizeText(forwardTo), priority, StatusPending, normalizeText(notes), now, now,
	)
	if err != nil {
		return 0, fmt.Errorf("cadastrar pendência: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("obter pendência cadastrada: %w", err)
	}
	if err := transaction.Commit(); err != nil {
		return 0, fmt.Errorf("confirmar cadastro: %w", err)
	}
	return id, nil
}

func (store *Store) updateClient(id int64, name, contact string) error {
	name = normalizeText(name)
	if name == "" {
		return errors.New("o nome da cliente é obrigatório")
	}
	result, err := store.db.Exec("UPDATE clients SET name = ?, contact = ?, updated_at = ? WHERE id = ?", name, normalizeText(contact), store.now(), id)
	if err != nil {
		return fmt.Errorf("atualizar cliente: %w", err)
	}
	return requireChangedRow(result, "cliente não encontrada")
}

func (store *Store) transitionFollowUp(id int64, fromStatus, toStatus string) error {
	if !validStatus(fromStatus) || !validStatus(toStatus) {
		return errInvalidTransition
	}

	now := store.now()
	var query string
	switch {
	case fromStatus == StatusPending && toStatus == StatusCompleted:
		query = "UPDATE followups SET status = ?, completed_at = ?, archived_at = NULL, updated_at = ? WHERE id = ? AND status = ?"
	case fromStatus == StatusCompleted && toStatus == StatusPending:
		query = "UPDATE followups SET status = ?, completed_at = NULL, archived_at = NULL, updated_at = ? WHERE id = ? AND status = ?"
		result, err := store.db.Exec(query, toStatus, now, id, fromStatus)
		if err != nil {
			return fmt.Errorf("reabrir pendência: %w", err)
		}
		if err := requireChangedRow(result, errInvalidTransition.Error()); err != nil {
			return errInvalidTransition
		}
		return nil
	case fromStatus == StatusCompleted && toStatus == StatusArchived:
		query = "UPDATE followups SET status = ?, archived_at = ?, updated_at = ? WHERE id = ? AND status = ?"
	default:
		return errInvalidTransition
	}

	result, err := store.db.Exec(query, toStatus, now, now, id, fromStatus)
	if err != nil {
		return fmt.Errorf("atualizar status: %w", err)
	}
	if err := requireChangedRow(result, errInvalidTransition.Error()); err != nil {
		return errInvalidTransition
	}
	return nil
}

func requireChangedRow(result sql.Result, message string) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return errors.New(message)
	}
	return nil
}

func (store *Store) getClient(id int64) (Client, error) {
	var client Client
	err := store.db.QueryRow("SELECT id, name, contact, created_at, updated_at FROM clients WHERE id = ?", id).
		Scan(&client.ID, &client.Name, &client.Contact, &client.CreatedAt, &client.UpdatedAt)
	if err != nil {
		return Client{}, err
	}
	return client, nil
}

func (store *Store) searchClients(query string) ([]Client, error) {
	term := "%" + normalizeText(query) + "%"
	rows, err := store.db.Query(`SELECT id, name, contact, created_at, updated_at FROM clients
		WHERE name LIKE ? COLLATE NOCASE OR contact LIKE ? COLLATE NOCASE ORDER BY name LIMIT 8`, term, term)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	clients := []Client{}
	for rows.Next() {
		var client Client
		if err := rows.Scan(&client.ID, &client.Name, &client.Contact, &client.CreatedAt, &client.UpdatedAt); err != nil {
			return nil, err
		}
		clients = append(clients, client)
	}
	return clients, rows.Err()
}

func scanFollowUps(rows *sql.Rows) ([]FollowUp, error) {
	items := []FollowUp{}
	for rows.Next() {
		var item FollowUp
		err := rows.Scan(
			&item.ID, &item.ClientID, &item.ClientName, &item.StartDate, &item.DueDate,
			&item.Description, &item.ForwardTo, &item.Priority, &item.Status, &item.Notes,
			&item.CompletedAt, &item.ArchivedAt, &item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

const followUpColumns = `f.id, f.client_id, c.name, f.start_date, f.due_date, f.description,
	f.forward_to, f.priority, f.status, f.notes, f.completed_at, f.archived_at, f.created_at, f.updated_at`

func (store *Store) dashboardFollowUps(filters DashboardFilters, today string) ([]FollowUp, error) {
	where := []string{"f.status != 'ARCHIVED'"}
	arguments := []any{}
	if filters.Query != "" {
		term := "%" + filters.Query + "%"
		where = append(where, "(c.name LIKE ? COLLATE NOCASE OR f.description LIKE ? COLLATE NOCASE OR f.forward_to LIKE ? COLLATE NOCASE)")
		arguments = append(arguments, term, term, term)
	}
	if validPriority(filters.Priority) {
		where = append(where, "f.priority = ?")
		arguments = append(arguments, filters.Priority)
	}
	if filters.Status == StatusPending || filters.Status == StatusCompleted {
		where = append(where, "f.status = ?")
		arguments = append(arguments, filters.Status)
	}
	switch filters.Due {
	case "OVERDUE":
		where = append(where, "f.status = 'PENDING' AND f.due_date < ?")
		arguments = append(arguments, today)
	case "TODAY":
		where = append(where, "f.status = 'PENDING' AND f.due_date = ?")
		arguments = append(arguments, today)
	case "TOMORROW":
		tomorrow, _ := time.Parse("2006-01-02", today)
		where = append(where, "f.status = 'PENDING' AND f.due_date = ?")
		arguments = append(arguments, tomorrow.AddDate(0, 0, 1).Format("2006-01-02"))
	}

	query := `SELECT ` + followUpColumns + ` FROM followups f JOIN clients c ON c.id = f.client_id
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY CASE WHEN f.status = 'PENDING' AND f.due_date < ? THEN 0 ELSE 1 END,
		f.due_date ASC, CASE f.priority WHEN 'HIGH' THEN 0 WHEN 'MEDIUM' THEN 1 ELSE 2 END, f.id ASC`
	arguments = append(arguments, today)
	rows, err := store.db.Query(query, arguments...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanFollowUps(rows)
}

func (store *Store) clientFollowUps(clientID int64) ([]FollowUp, error) {
	rows, err := store.db.Query(`SELECT `+followUpColumns+` FROM followups f JOIN clients c ON c.id = f.client_id
		WHERE f.client_id = ? AND f.status != 'ARCHIVED'
		ORDER BY CASE f.status WHEN 'PENDING' THEN 0 ELSE 1 END, f.due_date ASC,
		CASE f.priority WHEN 'HIGH' THEN 0 WHEN 'MEDIUM' THEN 1 ELSE 2 END`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanFollowUps(rows)
}

func (store *Store) reminders() ([]FollowUp, error) {
	rows, err := store.db.Query(`SELECT ` + followUpColumns + ` FROM followups f JOIN clients c ON c.id = f.client_id
		WHERE f.status = 'PENDING' ORDER BY f.due_date ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanFollowUps(rows)
}

func (store *Store) dashboardCounts(today string) (DashboardCounts, error) {
	tomorrow, _ := time.Parse("2006-01-02", today)
	var counts DashboardCounts
	err := store.db.QueryRow(`SELECT
		COALESCE(SUM(CASE WHEN status = 'PENDING' AND due_date < ? THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN status = 'PENDING' AND due_date = ? THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN status = 'PENDING' AND due_date = ? THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN status = 'PENDING' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN status = 'COMPLETED' THEN 1 ELSE 0 END), 0)
		FROM followups`, today, today, tomorrow.AddDate(0, 0, 1).Format("2006-01-02")).
		Scan(&counts.Overdue, &counts.Today, &counts.Tomorrow, &counts.Pending, &counts.Completed)
	return counts, err
}

func (store *Store) reportFollowUps(filters ReportFilters, today string) ([]FollowUp, error) {
	where := []string{"1 = 1"}
	arguments := []any{}
	if filters.DateFrom != "" {
		where = append(where, "f.due_date >= ?")
		arguments = append(arguments, filters.DateFrom)
	}
	if filters.DateTo != "" {
		where = append(where, "f.due_date <= ?")
		arguments = append(arguments, filters.DateTo)
	}
	if filters.Client != "" {
		where = append(where, "c.name LIKE ? COLLATE NOCASE")
		arguments = append(arguments, "%"+filters.Client+"%")
	}
	if filters.ForwardTo != "" {
		where = append(where, "f.forward_to LIKE ? COLLATE NOCASE")
		arguments = append(arguments, "%"+filters.ForwardTo+"%")
	}
	if validPriority(filters.Priority) {
		where = append(where, "f.priority = ?")
		arguments = append(arguments, filters.Priority)
	}
	if validStatus(filters.Status) {
		where = append(where, "f.status = ?")
		arguments = append(arguments, filters.Status)
	}
	if filters.Overdue {
		where = append(where, "f.due_date < ? AND f.status = 'PENDING'")
		arguments = append(arguments, today)
	}

	rows, err := store.db.Query(`SELECT `+followUpColumns+` FROM followups f JOIN clients c ON c.id = f.client_id
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY f.due_date ASC, CASE f.priority WHEN 'HIGH' THEN 0 WHEN 'MEDIUM' THEN 1 ELSE 2 END`, arguments...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanFollowUps(rows)
}
