package main

import (
	"database/sql"

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

	// Инициализация Redis
	cache, err := cache.NewWalletCache(
		os.Getenv("REDIS_ADDR"),
		os.Getenv("REDIS_PASSWORD"),
		0, // номер базы данных
	)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	app := handlers.NewApp(db, cache)

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
