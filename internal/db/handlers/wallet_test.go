package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
)

func TestChangeWallet(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	app := &App{DB: db}

	tests := []struct {
		name            string
		input           WalletOperation
		expectedCode    int
		expectedBalance float64
		mockBehavior    func(mock sqlmock.Sqlmock)
	}{
		{
			name: "Successful Deposit",
			input: WalletOperation{
				WalletID:      "123",
				OperationType: "DEPOSIT",
				Amount:        100.0,
			},
			expectedCode:    http.StatusOK,
			expectedBalance: 100.0,
			mockBehavior: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"balance"}).AddRow(100.0)
				mock.ExpectQuery("INSERT INTO wallets").
					WithArgs("123", 100.0).
					WillReturnRows(rows)
			},
		},
		{
			name: "Successful Withdraw",
			input: WalletOperation{
				WalletID:      "123",
				OperationType: "WITHDRAW",
				Amount:        50.0,
			},
			expectedCode:    http.StatusOK,
			expectedBalance: 50.0,
			mockBehavior: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"balance"}).AddRow(50.0)
				mock.ExpectQuery("INSERT INTO wallets").
					WithArgs("123", -50.0).
					WillReturnRows(rows)
			},
		},
		{
			name: "Insufficient Funds",
			input: WalletOperation{
				WalletID:      "123",
				OperationType: "WITHDRAW",
				Amount:        200.0,
			},
			expectedCode: http.StatusBadRequest,
			mockBehavior: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("INSERT INTO wallets").
					WithArgs("123", -200.0).
					WillReturnError(sql.ErrNoRows)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			tt.mockBehavior(mock)

			body, _ := json.Marshal(tt.input)
			req := httptest.NewRequest("POST", "/api/v1/wallet", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			app.ChangeWallet(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("Expected status code %d, got %d", tt.expectedCode, w.Code)
			}

			if tt.expectedCode == http.StatusOK {
				var response map[string]interface{}
				json.NewDecoder(w.Body).Decode(&response)

				if balance, ok := response["balance"].(float64); !ok || balance != tt.expectedBalance {
					t.Errorf("Expected balance %v, got %v", tt.expectedBalance, response["balance"])
				}
			}
		})
	}
}

func TestGetWallet(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	app := &App{DB: db}

	tests := []struct {
		name            string
		walletID        string
		expectedCode    int
		expectedBalance float64
		mockBehavior    func(mock sqlmock.Sqlmock)
	}{
		{
			name:            "Existing Wallet",
			walletID:        "123",
			expectedCode:    http.StatusOK,
			expectedBalance: 100.0,
			mockBehavior: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"balance"}).AddRow(100.0)
				mock.ExpectQuery("SELECT balance FROM wallets").
					WithArgs("123").
					WillReturnRows(rows)
			},
		},
		{
			name:         "Non-existing Wallet",
			walletID:     "999",
			expectedCode: http.StatusNotFound,
			mockBehavior: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT balance FROM wallets").
					WithArgs("999").
					WillReturnError(sql.ErrNoRows)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockBehavior(mock)

			req := httptest.NewRequest("GET", "/api/v1/wallets/"+tt.walletID, nil)
			w := httptest.NewRecorder()

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("walletId", tt.walletID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			app.GetWallet(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("Expected status code %d, got %d", tt.expectedCode, w.Code)
			}

			if tt.expectedCode == http.StatusOK {
				var response map[string]interface{}
				json.NewDecoder(w.Body).Decode(&response)

				if balance, ok := response["balance"].(float64); !ok || balance != tt.expectedBalance {
					t.Errorf("Expected balance %v, got %v", tt.expectedBalance, response["balance"])
				}
			}
		})
	}
}
