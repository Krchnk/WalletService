package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"wallet/internal/cache"

	"github.com/go-chi/chi/v5"
)

type WalletOperation struct {
	WalletID      string  `json:"walletId"`
	OperationType string  `json:"operationType"`
	Amount        float64 `json:"amount"`
}

type App struct {
	DB    *sql.DB
	Cache *cache.WalletCache
}

func NewApp(db *sql.DB, cache *cache.WalletCache) *App {

	db.SetMaxOpenConns(500)
	db.SetMaxIdleConns(100)
	db.SetConnMaxLifetime(2 * time.Minute)

	go func() {
		for {
			stats := db.Stats()
			log.Printf(
				"DB Stats: Open=%d, Idle=%d, InUse=%d, WaitCount=%d",
				stats.OpenConnections,
				stats.Idle,
				stats.InUse,
				stats.WaitCount,
			)
			time.Sleep(5 * time.Second)
		}
	}()

	return &App{
		DB:    db,
		Cache: cache,
	}
}

func (a *App) ChangeWallet(w http.ResponseWriter, r *http.Request) {
	var op WalletOperation
	if err := json.NewDecoder(r.Body).Decode(&op); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if op.OperationType != "DEPOSIT" && op.OperationType != "WITHDRAW" {
		http.Error(w, "invalid operation type", http.StatusBadRequest)
		return
	}

	amount := op.Amount
	if op.OperationType == "WITHDRAW" {
		amount = -amount
	}

	query := `
		INSERT INTO wallets (id, balance) VALUES ($1, $2)
		ON CONFLICT (id) DO UPDATE
		SET balance = wallets.balance + $2
		WHERE wallets.balance + $2 >= 0 RETURNING balance`

	var balance float64

	err := a.DB.QueryRowContext(r.Context(), query, op.WalletID, amount).Scan(&balance)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "insufficient funds", http.StatusBadRequest)
			return
		}
		log.Printf("Error executing query: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if err := a.Cache.InvalidateWallet(r.Context(), op.WalletID); err != nil {
		log.Printf("Failed to invalidate cache: %v", err)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{"walletId": op.WalletID, "balance": balance})
}

func (a *App) GetWallet(w http.ResponseWriter, r *http.Request) {
	walletID := chi.URLParam(r, "walletId")

	if wallet, err := a.Cache.GetWallet(r.Context(), walletID); err == nil && wallet != nil {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"walletId": walletID,
			"balance":  wallet.Balance,
		})
		return
	}

	var balance float64
	err := a.DB.QueryRowContext(r.Context(), "SELECT balance FROM wallets WHERE id = $1", walletID).Scan(&balance)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "wallet not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	wallet := &cache.WalletData{
		Balance:   balance,
		UpdatedAt: time.Now(),
	}
	if err := a.Cache.SetWallet(r.Context(), walletID, wallet, 5*time.Minute); err != nil {
		log.Printf("Failed to cache wallet data: %v", err)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{"walletId": walletID, "balance": balance})
}
