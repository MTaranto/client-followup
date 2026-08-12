package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	address := flag.String("addr", "127.0.0.1:8080", "endereço local do servidor")
	databasePath := flag.String("db", "data/client-followup.db", "caminho do banco SQLite")
	backupDirectory := flag.String("backups", "backups", "diretório de backups diários")
	flag.Parse()

	location, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		log.Fatalf("carregar fuso America/Sao_Paulo: %v", err)
	}
	now := func() time.Time { return time.Now().In(location) }

	store, err := openStore(*databasePath, now)
	if err != nil {
		log.Fatalf("inicializar banco de dados: %v", err)
	}
	defer store.Close()

	backupPath, err := store.createDailyBackup(*backupDirectory, now())
	if err != nil {
		log.Fatalf("proteger dados com backup diário: %v", err)
	}
	log.Printf("backup diário disponível em %s", backupPath)

	app, err := newApplication(store, location)
	if err != nil {
		log.Fatalf("inicializar aplicação: %v", err)
	}
	server := &http.Server{
		Addr:              *address,
		Handler:           app.routes(),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("client-followup disponível em http://%s", *address)
		serverErrors <- server.ListenAndServe()
	}()

	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, syscall.SIGINT, syscall.SIGTERM)
	select {
	case signalValue := <-shutdownSignals:
		log.Printf("encerrando após sinal %s", signalValue)
	case err := <-serverErrors:
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("servidor HTTP: %v", err)
		}
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownContext); err != nil {
		fmt.Fprintf(os.Stderr, "encerrar servidor: %v\n", err)
	}
}
