package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	_ "github.com/mattn/go-sqlite3"
)

var errInvalidTransition = errors.New("transição de status inválida")
var errClientResolutionRequired = errors.New("escolha um cliente existente ou confirme o cadastro de um novo cliente com este nome")

var (
	digitsOnlyPhonePattern = regexp.MustCompile(`^[0-9]{11}$`)
	formattedPhonePattern  = regexp.MustCompile(`^\([0-9]{2}\) [0-9]{5}-[0-9]{4}$`)
)

type clientPhoneChangeRequiredError struct {
	Client         Client
	SubmittedPhone string
}

type clientEditPhoneChangeRequiredError struct {
	Client         Client
	SubmittedPhone string
}

type clientDuplicatePhoneRequiredError struct {
	SubmittedPhone  string
	ExistingClients []Client
	Token           string
}

type clientEditDuplicatePhoneRequiredError struct {
	Client          Client
	SubmittedPhone  string
	ExistingClients []Client
	Token           string
}

func (err *clientEditPhoneChangeRequiredError) Error() string {
	return "confirme a alteração do telefone do cliente"
}

func (err *clientPhoneChangeRequiredError) Error() string {
	return "confirme como o telefone diferente deve ser tratado"
}

func (err *clientDuplicatePhoneRequiredError) Error() string {
	return "confirme o uso do telefone já associado a outro cliente"
}

func (err *clientEditDuplicatePhoneRequiredError) Error() string {
	return "confirme a alteração para telefone já associado a outro cliente"
}

type Store struct {
	db          *sql.DB
	now         func() time.Time
	tokenSecret []byte
}

func (store *Store) phoneConfirmationToken(phone string, clientID int64, conflicts []Client) string {
	var ids []string
	for _, c := range conflicts {
		ids = append(ids, fmt.Sprintf("%d", c.ID))
	}
	sort.Strings(ids)
	mac := hmac.New(sha256.New, store.tokenSecret)
	fmt.Fprintf(mac, "phone-confirm-v1:%s:%d:%s", phone, clientID, strings.Join(ids, ","))
	return hex.EncodeToString(mac.Sum(nil))[:16]
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

	tokenSecret := make([]byte, 32)
	if _, err := rand.Read(tokenSecret); err != nil {
		database.Close()
		return nil, fmt.Errorf("inicializar secret de autenticação: %w", err)
	}

	store := &Store{db: database, now: now, tokenSecret: tokenSecret}
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
	name, err := validateClientName(name)
	if err != nil {
		return Client{}, err
	}
	contact, err = normalizePhone(contact)
	if err != nil {
		return Client{}, err
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

func (store *Store) findOrCreateClient(transaction *sql.Tx, clientID int64, name, contact, phoneChangeAction, clientResolution, duplicatePhoneDecision, duplicatePhoneToken string) (int64, error) {
	name, err := validateClientName(name)
	if err != nil {
		return 0, err
	}

	if clientID > 0 {
		var existing Client
		if err := transaction.QueryRow("SELECT id, name, contact, created_at, updated_at FROM clients WHERE id = ?", clientID).Scan(
			&existing.ID, &existing.Name, &existing.Contact, &existing.CreatedAt, &existing.UpdatedAt,
		); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return 0, errors.New("cliente selecionado não foi encontrado")
			}
			return 0, fmt.Errorf("consultar cliente: %w", err)
		}
		if normalizeClientName(existing.Name) != normalizeClientName(name) {
			return 0, errors.New("o nome não corresponde ao cliente selecionado; selecione novamente")
		}

		existingPhone, existingPhoneErr := normalizePhone(existing.Contact)
		if existingPhoneErr == nil && existingPhone == contact {
			return clientID, nil
		}

		switch phoneChangeAction {
		case PhoneChangeUpdate:
			conflicts, err := findClientsByPhoneQuery(transaction, contact, clientID)
			if err != nil {
				return 0, fmt.Errorf("verificar duplicidade de telefone: %w", err)
			}
			if len(conflicts) > 0 {
				expectedToken := store.phoneConfirmationToken(contact, clientID, conflicts)
				if duplicatePhoneDecision != "allow" || duplicatePhoneToken != expectedToken {
					return 0, &clientDuplicatePhoneRequiredError{SubmittedPhone: contact, ExistingClients: conflicts, Token: expectedToken}
				}
			}
			if _, err := transaction.Exec("UPDATE clients SET contact = ?, updated_at = ? WHERE id = ?", contact, store.now(), clientID); err != nil {
				return 0, fmt.Errorf("atualizar telefone do cliente: %w", err)
			}
			return clientID, nil
		case PhoneChangeNewClient:
			conflicts, err := findClientsByPhoneQuery(transaction, contact, 0)
			if err != nil {
				return 0, fmt.Errorf("verificar duplicidade de telefone: %w", err)
			}
			if len(conflicts) > 0 {
				expectedToken := store.phoneConfirmationToken(contact, 0, conflicts)
				if duplicatePhoneDecision != "allow" || duplicatePhoneToken != expectedToken {
					return 0, &clientDuplicatePhoneRequiredError{SubmittedPhone: contact, ExistingClients: conflicts, Token: expectedToken}
				}
			}
			return insertClient(transaction, name, contact, store.now())
		default:
			return 0, &clientPhoneChangeRequiredError{Client: existing, SubmittedPhone: contact}
		}
	}

	conflicts, err := findClientsByPhoneQuery(transaction, contact, 0)
	if err != nil {
		return 0, fmt.Errorf("verificar duplicidade de telefone: %w", err)
	}
	if len(conflicts) > 0 {
		expectedToken := store.phoneConfirmationToken(contact, 0, conflicts)
		if duplicatePhoneDecision != "allow" || duplicatePhoneToken != expectedToken {
			return 0, &clientDuplicatePhoneRequiredError{SubmittedPhone: contact, ExistingClients: conflicts, Token: expectedToken}
		}
	}

	matches, err := findClientsByExactNameQuery(transaction, name, 2)
	if err != nil {
		return 0, fmt.Errorf("consultar clientes homônimos: %w", err)
	}
	// A missing client ID is ambiguous when the normalized name already exists.
	// Requiring the explicit homonym choice prevents forged requests from silently
	// creating another record after the browser presented existing identities.
	if len(matches) > 0 && clientResolution != ClientResolutionNewHomonym {
		return 0, errClientResolutionRequired
	}
	if clientResolution == ClientResolutionNewHomonym && len(matches) == 0 {
		return 0, errors.New("não há clientes homônimos para esta decisão")
	}
	return insertClient(transaction, name, contact, store.now())
}

func insertClient(transaction *sql.Tx, name, contact string, now time.Time) (int64, error) {
	result, err := transaction.Exec(
		"INSERT INTO clients (name, contact, created_at, updated_at) VALUES (?, ?, ?, ?)",
		name, contact, now, now,
	)
	if err != nil {
		return 0, fmt.Errorf("cadastrar cliente: %w", err)
	}
	return result.LastInsertId()
}

func (store *Store) createFollowUp(clientID int64, clientName, contact, startDate, dueDate, description, forwardTo, priority, notes, phoneChangeAction, clientResolution, duplicatePhoneDecision, duplicatePhoneToken string) (int64, error) {
	contact, err := normalizePhone(contact)
	if err != nil {
		return 0, err
	}
	parsedStart, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return 0, errors.New("a data de início é inválida")
	}
	parsedDue, err := time.Parse("2006-01-02", dueDate)
	if err != nil {
		return 0, errors.New("a data limite é obrigatória e deve ser válida")
	}
	if parsedDue.Before(parsedStart) {
		return 0, errors.New("a data limite não pode ser anterior à data de início")
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

	resolvedClientID, err := store.findOrCreateClient(transaction, clientID, clientName, contact, phoneChangeAction, clientResolution, duplicatePhoneDecision, duplicatePhoneToken)
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

func (store *Store) updateClient(id int64, name, contact, phoneChangeConfirmation, duplicatePhoneDecision, duplicatePhoneToken string) error {
	name, err := validateClientName(name)
	if err != nil {
		return err
	}
	contact, err = normalizePhone(contact)
	if err != nil {
		return err
	}
	existing, err := store.getClient(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("cliente não encontrado")
		}
		return fmt.Errorf("consultar cliente: %w", err)
	}
	currentPhone, currentPhoneErr := normalizePhone(existing.Contact)
	if currentPhoneErr != nil || currentPhone != contact {
		conflicts, err := findClientsByPhoneQuery(store.db, contact, id)
		if err != nil {
			return fmt.Errorf("verificar duplicidade de telefone: %w", err)
		}
		if len(conflicts) > 0 {
			expectedToken := store.phoneConfirmationToken(contact, id, conflicts)
			if duplicatePhoneDecision != "allow" || duplicatePhoneToken != expectedToken {
				return &clientEditDuplicatePhoneRequiredError{Client: existing, ExistingClients: conflicts, SubmittedPhone: contact, Token: expectedToken}
			}
		}
		if phoneChangeConfirmation != ClientPhoneChangeConfirmation {
			return &clientEditPhoneChangeRequiredError{Client: existing, SubmittedPhone: contact}
		}
	}
	result, err := store.db.Exec("UPDATE clients SET name = ?, contact = ?, updated_at = ? WHERE id = ?", name, contact, store.now(), id)
	if err != nil {
		return fmt.Errorf("atualizar cliente: %w", err)
	}
	return requireChangedRow(result, "cliente não encontrado")
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

func (store *Store) getFollowUp(id int64) (FollowUp, error) {
	var item FollowUp
	err := store.db.QueryRow(`SELECT `+followUpColumns+` FROM followups f JOIN clients c ON c.id = f.client_id WHERE f.id = ?`, id).Scan(
		&item.ID, &item.ClientID, &item.ClientName, &item.StartDate, &item.DueDate,
		&item.Description, &item.ForwardTo, &item.Priority, &item.Status, &item.Notes,
		&item.CompletedAt, &item.ArchivedAt, &item.CreatedAt, &item.UpdatedAt,
	)
	return item, err
}

func (store *Store) updatePendingFollowUp(id int64, startDate, dueDate, description, forwardTo, priority, notes string) error {
	parsedStart, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return errors.New("a data de início é inválida")
	}
	parsedDue, err := time.Parse("2006-01-02", dueDate)
	if err != nil {
		return errors.New("a data limite é obrigatória e deve ser válida")
	}
	if parsedDue.Before(parsedStart) {
		return errors.New("a data limite não pode ser anterior à data de início")
	}
	description = normalizeText(description)
	if description == "" {
		return errors.New("a descrição é obrigatória")
	}
	if !validPriority(priority) {
		return errors.New("a prioridade é inválida")
	}

	// Keeping the current status in the write predicate closes the race between
	// opening the editor and another request completing or archiving the item.
	result, err := store.db.Exec(`UPDATE followups SET start_date = ?, due_date = ?, description = ?,
		forward_to = ?, priority = ?, notes = ?, updated_at = ? WHERE id = ? AND status = ?`,
		startDate, dueDate, description, normalizeText(forwardTo), priority, normalizeText(notes), store.now(), id, StatusPending)
	if err != nil {
		return fmt.Errorf("atualizar pendência: %w", err)
	}
	if err := requireChangedRow(result, errInvalidTransition.Error()); err != nil {
		return errInvalidTransition
	}
	return nil
}

func (store *Store) deletePendingFollowUp(id int64) (bool, error) {
	transaction, err := store.db.Begin()
	if err != nil {
		return false, fmt.Errorf("iniciar exclusão: %w", err)
	}
	defer transaction.Rollback()

	var clientID int64
	var status string
	if err := transaction.QueryRow("SELECT client_id, status FROM followups WHERE id = ?", id).Scan(&clientID, &status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, errors.New("pendência não encontrada")
		}
		return false, fmt.Errorf("consultar pendência: %w", err)
	}
	if status != StatusPending {
		return false, errInvalidTransition
	}
	result, err := transaction.Exec("DELETE FROM followups WHERE id = ? AND status = ?", id, StatusPending)
	if err != nil {
		return false, fmt.Errorf("excluir pendência: %w", err)
	}
	if err := requireChangedRow(result, errInvalidTransition.Error()); err != nil {
		return false, errInvalidTransition
	}

	var remaining int
	if err := transaction.QueryRow("SELECT COUNT(*) FROM followups WHERE client_id = ?", clientID).Scan(&remaining); err != nil {
		return false, fmt.Errorf("verificar histórico da cliente: %w", err)
	}
	clientDeleted := remaining == 0
	if clientDeleted {
		// The orphan check and both deletions share this transaction so a failure
		// cannot leave a client without its only follow-up or delete one with history.
		if _, err := transaction.Exec("DELETE FROM clients WHERE id = ?", clientID); err != nil {
			return false, fmt.Errorf("excluir cliente sem pendências: %w", err)
		}
	}
	if err := transaction.Commit(); err != nil {
		return false, fmt.Errorf("confirmar exclusão: %w", err)
	}
	return clientDeleted, nil
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
	query = normalizeClientName(query)
	if query == "" {
		return []Client{}, nil
	}
	rows, err := store.db.Query(`SELECT id, name, contact, created_at, updated_at FROM clients ORDER BY name COLLATE NOCASE, id`)
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
		if strings.Contains(normalizeClientName(client.Name), query) {
			clients = append(clients, client)
			if len(clients) == 8 {
				break
			}
		}
	}
	return clients, rows.Err()
}

func (store *Store) findClientsByPhone(phone string, excludeClientID int64) ([]Client, error) {
	return findClientsByPhoneQuery(store.db, phone, excludeClientID)
}

func findClientsByPhoneQuery(queryer clientQueryer, phone string, excludeClientID int64) ([]Client, error) {
	phone, err := normalizePhone(phone)
	if err != nil {
		return []Client{}, nil
	}
	rows, err := queryer.Query(`SELECT id, name, contact, created_at, updated_at FROM clients WHERE id != ? ORDER BY id`, excludeClientID)
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
		cPhone, err := normalizePhone(client.Contact)
		if err == nil && cPhone == phone {
			clients = append(clients, client)
		}
	}
	return clients, rows.Err()
}

func (store *Store) findClientsByExactName(name string) ([]Client, error) {
	return findClientsByExactNameQuery(store.db, name, 8)
}

type clientQueryer interface {
	Query(query string, args ...any) (*sql.Rows, error)
}

func findClientsByExactNameQuery(queryer clientQueryer, name string, limit int) ([]Client, error) {
	name = normalizeClientName(name)
	if name == "" {
		return []Client{}, nil
	}
	rows, err := queryer.Query(`SELECT id, name, contact, created_at, updated_at FROM clients ORDER BY id`)
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
		if normalizeClientName(client.Name) == name {
			clients = append(clients, client)
			if len(clients) == limit {
				break
			}
		}
	}
	return clients, rows.Err()
}

func normalizeClientName(value string) string {
	// SQLite NOCASE only handles ASCII. Folding the Portuguese characters in Go
	// keeps the original stored text untouched and avoids a schema or dependency
	// solely for the small local client dataset.
	value = strings.Join(strings.Fields(value), " ")
	var normalized strings.Builder
	for _, character := range strings.ToLower(value) {
		switch character {
		case 'á', 'à', 'â', 'ã', 'ä':
			character = 'a'
		case 'é', 'è', 'ê', 'ë':
			character = 'e'
		case 'í', 'ì', 'î', 'ï':
			character = 'i'
		case 'ó', 'ò', 'ô', 'õ', 'ö':
			character = 'o'
		case 'ú', 'ù', 'û', 'ü':
			character = 'u'
		case 'ç':
			character = 'c'
		case 'ñ':
			character = 'n'
		}
		if !unicode.Is(unicode.Mn, character) {
			normalized.WriteRune(character)
		}
	}
	return normalized.String()
}

func validateClientName(value string) (string, error) {
	value = normalizeText(value)
	if value == "" {
		return "", errors.New("o nome do cliente é obrigatório")
	}
	for _, character := range value {
		if unicode.IsDigit(character) {
			return "", errors.New("o nome do cliente não pode conter números")
		}
	}
	return value, nil
}

func normalizePhone(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("o telefone é obrigatório")
	}

	var digits strings.Builder
	for _, character := range value {
		if character >= '0' && character <= '9' {
			digits.WriteRune(character)
		}
	}
	phoneDigits := digits.String()
	if len(phoneDigits) < 11 {
		return "", errors.New("o telefone deve conter exatamente 11 dígitos")
	}
	if len(phoneDigits) > 11 {
		return "", errors.New("o telefone não pode conter mais de 11 dígitos")
	}
	if !digitsOnlyPhonePattern.MatchString(value) && !formattedPhonePattern.MatchString(value) {
		return "", errors.New("o telefone contém caracteres ou formatação inválidos")
	}

	return fmt.Sprintf("(%s) %s-%s", phoneDigits[:2], phoneDigits[2:7], phoneDigits[7:]), nil
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

	items, err := scanFollowUps(rows)
	if err != nil {
		return nil, err
	}

	clientQuery := normalizeClientName(filters.Client)
	forwardQuery := normalizeClientName(filters.ForwardTo)

	if clientQuery == "" && forwardQuery == "" {
		return items, nil
	}

	filtered := make([]FollowUp, 0, len(items))
	for _, item := range items {
		if clientQuery != "" && !strings.Contains(normalizeClientName(item.ClientName), clientQuery) {
			continue
		}
		if forwardQuery != "" && !strings.Contains(normalizeClientName(item.ForwardTo), forwardQuery) {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered, nil
}
