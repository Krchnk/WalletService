package main

import (
	"database/sql"
	"io/ioutil"
	"os/signal"
	"syscall"

	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"wallet/internal/cache"
	"wallet/internal/db/handlers"
)

func main() {
	err := godotenv.Load("/app/config.env")
	if err != nil {
		log.Fatalf("Error loading config.env file")
	}

	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := applyMigrations(db); err != nil {
		log.Printf("Warning: Failed to apply migrations: %v", err)
	}

	cache, err := cache.NewWalletCache(
		os.Getenv("REDIS_ADDR"),
		os.Getenv("REDIS_PASSWORD"),
		0,
	)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	app := handlers.NewApp(db, cache)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		app.Shutdown()
		os.Exit(0)
	}()

	r := chi.NewRouter()
	r.Post("/api/v1/wallet", app.ChangeWallet)
	r.Get("/api/v1/wallets/{walletId}", app.GetWallet)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server running on port %s", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), r))
}

func applyMigrations(db *sql.DB) error {
	migrationFile := "/app/internal/db/migrations/001_create_wallets_table.sql"
	content, err := ioutil.ReadFile(migrationFile)
	if err != nil {
		return fmt.Errorf("error reading migration file: %v", err)
	}

	_, err = db.Exec(string(content))
	if err != nil {
		return fmt.Errorf("error applying migration: %v", err)
	}

	return nil
}
