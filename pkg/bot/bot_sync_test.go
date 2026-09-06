package bot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"reseller-bot/pkg/backend"
	"reseller-bot/pkg/config"
	"reseller-bot/pkg/db"
)

func TestBot_SyncBotConfigFromServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/reseller/hosting/bots/7/config" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"code": 200,
			"msg": "success",
			"data": {
				"bot_id": 7,
				"welcome_text": "Rehydrated Welcome Text",
				"welcome_image": "file_welcome_rehydrated",
				"support_text": "Rehydrated Support Text",
				"support_image": "file_support_rehydrated",
				"required_channel": "@rehydrated_channel",
				"qr_enabled": true,
				"reminders_enabled": false,
				"tag_mappings": {"GER": "🇩🇪 Germany Superfast", "US": "🇺🇸 USA"},
				"staff_list": [{"telegram_id": 112233, "display_name": "Admin John", "added_at": 1700000000}]
			}
		}`))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	sqliteDB, err := db.NewDB(filepath.Join(tmpDir, "bot_sync.db"))
	if err != nil {
		t.Fatalf("Failed to initialize SQLite: %v", err)
	}
	defer sqliteDB.Close()

	client := backend.NewClient(server.URL, "rn_test_key", nil, false, 7)
	bot := &Bot{
		db:     sqliteDB,
		client: client,
		cfg:    &config.Config{BotID: 7},
	}

	// Run sync on fresh SQLite DB
	bot.syncBotConfigFromServer(context.Background())

	// Verify all settings rehydrated in SQLite
	welcomeText, err := sqliteDB.GetSetting("welcome_text")
	if err != nil || welcomeText != "Rehydrated Welcome Text" {
		t.Errorf("Expected welcome_text 'Rehydrated Welcome Text', got %q", welcomeText)
	}
	welcomeImg, _ := sqliteDB.GetSetting("welcome_image")
	if welcomeImg != "file_welcome_rehydrated" {
		t.Errorf("Expected welcome_image 'file_welcome_rehydrated', got %q", welcomeImg)
	}
	supportText, _ := sqliteDB.GetSetting("support_text")
	if supportText != "Rehydrated Support Text" {
		t.Errorf("Expected support_text 'Rehydrated Support Text', got %q", supportText)
	}
	requiredChan, _ := sqliteDB.GetSetting("required_channel")
	if requiredChan != "@rehydrated_channel" {
		t.Errorf("Expected required_channel '@rehydrated_channel', got %q", requiredChan)
	}
	qrEnabled, _ := sqliteDB.GetSetting("qr_enabled")
	if qrEnabled != "on" {
		t.Errorf("Expected qr_enabled 'on', got %q", qrEnabled)
	}
	remEnabled, _ := sqliteDB.GetSetting("reminders_enabled")
	if remEnabled != "off" {
		t.Errorf("Expected reminders_enabled 'off', got %q", remEnabled)
	}

	// Verify tag mappings rehydrated in SQLite
	tags, err := sqliteDB.GetAllTagMappings()
	if err != nil || len(tags) != 2 {
		t.Fatalf("Expected 2 tag mappings, got %v (err=%v)", tags, err)
	}
	if tags["GER"] != "🇩🇪 Germany Superfast" {
		t.Errorf("Expected tag mapping for GER to be '🇩🇪 Germany Superfast', got %q", tags["GER"])
	}

	// Verify staff list rehydrated in SQLite
	staff, err := sqliteDB.GetStaffList()
	if err != nil || len(staff) != 1 {
		t.Fatalf("Expected 1 staff member, got %v (err=%v)", staff, err)
	}
	if staff[0].TelegramID != 112233 || staff[0].DisplayName != "Admin John" {
		t.Errorf("Unexpected staff member: %+v", staff[0])
	}
	isStaff, _ := sqliteDB.IsStaff(112233)
	if !isStaff {
		t.Errorf("Expected user 112233 to be recognized as staff")
	}
}

func TestBot_SyncBotUsersFromServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/reseller/hosting/bots/7/users" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"code": 200,
			"msg": "success",
			"data": {
				"list": [
					{"telegram_id": 555111, "user_id": 100, "created_at": 1700000000},
					{"telegram_id": 555222, "user_id": 101, "created_at": 1700000100},
					{"telegram_id": 555333, "user_id": 102, "created_at": 1700000200}
				]
			}
		}`))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	sqliteDB, err := db.NewDB(filepath.Join(tmpDir, "bot_users_sync.db"))
	if err != nil {
		t.Fatalf("Failed to initialize SQLite: %v", err)
	}
	defer sqliteDB.Close()

	client := backend.NewClient(server.URL, "rn_test_key", nil, false, 7)
	bot := &Bot{
		db:     sqliteDB,
		client: client,
		cfg:    &config.Config{BotID: 7},
	}

	// Verify database starts with zero users
	initialUsers, _ := sqliteDB.GetAllUsers()
	if len(initialUsers) != 0 {
		t.Fatalf("Expected 0 users initially, got %d", len(initialUsers))
	}

	// Run user rehydration
	bot.syncBotUsersFromServer(context.Background())

	// Verify all users are now present in SQLite
	rehydratedUsers, err := sqliteDB.GetAllUsers()
	if err != nil || len(rehydratedUsers) != 3 {
		t.Fatalf("Expected 3 rehydrated users, got %d (err=%v)", len(rehydratedUsers), err)
	}

	u1, err := sqliteDB.GetUser(555111)
	if err != nil || u1 == nil || u1.UserID != 100 {
		t.Errorf("Expected user 555111 to have UserID 100, got %+v", u1)
	}

	u3, err := sqliteDB.GetUser(555333)
	if err != nil || u3 == nil || u3.UserID != 102 {
		t.Errorf("Expected user 555333 to have UserID 102, got %+v", u3)
	}
}

func TestBot_PushBotConfigToBackend(t *testing.T) {
	var receivedBody backend.BotConfigUpdate
	var receivedMethod string
	var receivedPath string
	var wg sync.WaitGroup
	wg.Add(1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code": 200, "msg": "success", "data": {"bot_id": 7}}`))
		wg.Done()
	}))
	defer server.Close()

	client := backend.NewClient(server.URL, "rn_test_key", nil, false, 7)
	bot := &Bot{
		client: client,
		cfg:    &config.Config{BotID: 7},
	}

	newText := "Live Updated Welcome"
	bot.pushBotConfigToBackend(&backend.BotConfigUpdate{
		WelcomeText: &newText,
	})

	// Wait for async write-through goroutine to deliver
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for pushBotConfigToBackend to complete")
	}

	if receivedMethod != "PUT" {
		t.Errorf("Expected PUT method, got %s", receivedMethod)
	}
	if receivedPath != "/v1/reseller/hosting/bots/7/config" {
		t.Errorf("Expected path /v1/reseller/hosting/bots/7/config, got %s", receivedPath)
	}
	if receivedBody.WelcomeText == nil || *receivedBody.WelcomeText != "Live Updated Welcome" {
		t.Errorf("Unexpected payload received: %+v", receivedBody)
	}
}
