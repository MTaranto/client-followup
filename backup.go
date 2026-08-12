package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const backupRetention = 14

func (store *Store) createDailyBackup(directory string, today time.Time) (string, error) {
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", fmt.Errorf("criar diretório de backup: %w", err)
	}

	backupPath := filepath.Join(directory, "client-followup-"+today.Format("2006-01-02")+".db")
	if _, err := os.Stat(backupPath); err == nil {
		return backupPath, nil
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("verificar backup diário: %w", err)
	}

	if _, err := store.db.Exec("VACUUM INTO ?", backupPath); err != nil {
		return "", fmt.Errorf("criar backup diário: %w", err)
	}
	if err := os.Chmod(backupPath, 0o600); err != nil {
		return "", fmt.Errorf("proteger permissões do backup diário: %w", err)
	}
	if err := removeOldBackups(directory, backupRetention); err != nil {
		return "", err
	}
	return backupPath, nil
}

func removeOldBackups(directory string, retention int) error {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return fmt.Errorf("listar backups: %w", err)
	}

	paths := []string{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "client-followup-") || !strings.HasSuffix(entry.Name(), ".db") {
			continue
		}
		paths = append(paths, filepath.Join(directory, entry.Name()))
	}
	sort.Strings(paths)
	if len(paths) <= retention {
		return nil
	}
	for _, path := range paths[:len(paths)-retention] {
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("remover backup antigo %s: %w", filepath.Base(path), err)
		}
	}
	return nil
}
