package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (store *Store) initializeBackups(directory string, today time.Time) (string, error) {
	if directory == "" {
		return "", nil
	}
	store.recoveryMu.Lock()
	defer store.recoveryMu.Unlock()

	store.backupDirectory = directory
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", fmt.Errorf("criar diretório de backup: %w", err)
	}

	cleanTemporarySnapshots(directory)

	baselinePath, err := store.createDailyBaseline(directory, today)
	if err != nil {
		return "", err
	}

	if err := cleanDailyBaselines(directory, today); err != nil {
		return "", fmt.Errorf("limpar baselines diários anteriores: %w", err)
	}

	return baselinePath, nil
}

func (store *Store) createDailyBaseline(directory string, today time.Time) (string, error) {
	baselinePath := filepath.Join(directory, "client-followup-"+today.Format("2006-01-02")+".db")
	if _, err := os.Stat(baselinePath); err == nil {
		return baselinePath, nil
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("verificar baseline diário: %w", err)
	}

	if _, err := store.db.Exec("VACUUM INTO ?", baselinePath); err != nil {
		return "", fmt.Errorf("criar baseline diário: %w", err)
	}
	if err := os.Chmod(baselinePath, 0o600); err != nil {
		return "", fmt.Errorf("proteger permissões do baseline diário: %w", err)
	}
	return baselinePath, nil
}

func cleanDailyBaselines(directory string, today time.Time) error {
	entries, err := os.ReadDir(directory)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	todayName := "client-followup-" + today.Format("2006-01-02") + ".db"
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "client-followup-") || !strings.HasSuffix(entry.Name(), ".db") {
			continue
		}
		if entry.Name() != todayName {
			if err := os.Remove(filepath.Join(directory, entry.Name())); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("remover baseline antigo %s: %w", entry.Name(), err)
			}
		}
	}
	return nil
}

func cleanTemporarySnapshots(directory string) {
	_ = os.Remove(filepath.Join(directory, "recent-tmp.db"))
}

func (store *Store) prepareRecoverySnapshot() (string, error) {
	if store.backupDirectory == "" {
		return "", nil
	}

	tempPath := filepath.Join(store.backupDirectory, "recent-tmp.db")
	_ = os.Remove(tempPath)

	if _, err := store.db.Exec("VACUUM INTO ?", tempPath); err != nil {
		return "", fmt.Errorf("criar snapshot de recuperação: %w", err)
	}
	if err := os.Chmod(tempPath, 0o600); err != nil {
		_ = os.Remove(tempPath)
		return "", fmt.Errorf("proteger permissões do snapshot: %w", err)
	}
	return tempPath, nil
}

func (store *Store) discardRecoverySnapshot(tempPath string) {
	if tempPath == "" {
		return
	}
	_ = os.Remove(tempPath)
}

func (store *Store) promoteRecoverySnapshot(tempPath string) {
	if tempPath == "" || store.backupDirectory == "" {
		return
	}

	recent3 := filepath.Join(store.backupDirectory, "recent-3.db")
	recent2 := filepath.Join(store.backupDirectory, "recent-2.db")
	recent1 := filepath.Join(store.backupDirectory, "recent-1.db")

	if err := os.Remove(recent3); err != nil && !os.IsNotExist(err) {
		log.Printf("aviso: falha ao remover recent-3.db na promoção de recovery snapshot: %v", err)
		_ = os.Remove(tempPath)
		return
	}

	if _, err := os.Stat(recent2); err == nil {
		if err := os.Rename(recent2, recent3); err != nil {
			log.Printf("aviso: falha ao rotacionar recent-2 para recent-3 na promoção de recovery snapshot: %v", err)
			_ = os.Remove(tempPath)
			return
		}
	} else if !os.IsNotExist(err) {
		log.Printf("aviso: falha ao verificar recent-2.db na promoção de recovery snapshot: %v", err)
		_ = os.Remove(tempPath)
		return
	}

	if _, err := os.Stat(recent1); err == nil {
		if err := os.Rename(recent1, recent2); err != nil {
			log.Printf("aviso: falha ao rotacionar recent-1 para recent-2 na promoção de recovery snapshot: %v", err)
			_ = os.Remove(tempPath)
			return
		}
	} else if !os.IsNotExist(err) {
		log.Printf("aviso: falha ao verificar recent-1.db na promoção de recovery snapshot: %v", err)
		_ = os.Remove(tempPath)
		return
	}

	if err := os.Rename(tempPath, recent1); err != nil {
		log.Printf("aviso: falha ao promover snapshot temporário para recent-1.db: %v", err)
		_ = os.Remove(tempPath)
		return
	}
}
