package backend

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegisterUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify authorization header
		auth := r.Header.Get("Authorization")
		if auth != "Bearer rn_test_key" {
			t.Errorf("Expected Auth header 'Bearer rn_test_key', got '%s'", auth)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		if r.URL.Path != "/v1/reseller/user" {
			t.Errorf("Expected path /v1/reseller/user, got %s", r.URL.Path)
		}

		var req UserRegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal("Failed to decode request body")
		}

		if req.TelegramID != 123456 {
			t.Errorf("Expected TelegramID 123456, got %d", req.TelegramID)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"user_id": 482, "balance": 50000, "lang": "fa", "created_new": true}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "rn_test_key")
	resp, err := client.RegisterUser(context.Background(), &UserRegisterRequest{
		TelegramID:   123456,
		FirstName:    "Ali",
		LastName:     "Reza",
		Username:     "alireza",
		LanguageCode: "fa",
	})

	if err != nil {
		t.Fatalf("RegisterUser failed: %v", err)
	}

	if resp.UserID != 482 {
		t.Errorf("Expected user_id 482, got %d", resp.UserID)
	}

	if resp.Balance != 50000 {
		t.Errorf("Expected balance 50000, got %d", resp.Balance)
	}

	if resp.Lang != "fa" {
		t.Errorf("Expected lang fa, got %s", resp.Lang)
	}
}

func TestUpdateUserBalance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("Expected PUT method, got %s", r.Method)
		}

		if r.URL.Path != "/v1/reseller/user/balance" {
			t.Errorf("Expected path /v1/reseller/user/balance, got %s", r.URL.Path)
		}

		var req BalanceUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal("Failed to decode request body")
		}

		if req.UserID != 482 || req.Amount != 50000 || req.Reason != "Test approve" {
			t.Errorf("Unexpected request data: %+v", req)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "rn_test_key")
	err := client.UpdateUserBalance(context.Background(), &BalanceUpdateRequest{
		UserID: 482,
		Amount: 50000,
		Reason: "Test approve",
	})

	if err != nil {
		t.Fatalf("UpdateUserBalance failed: %v", err)
	}
}
