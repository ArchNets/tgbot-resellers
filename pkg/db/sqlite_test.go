package db

import (
	"testing"
	"time"
)

func TestDBOperations(t *testing.T) {
	// Initialize in-memory database
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create in-memory database: %v", err)
	}
	defer db.Close()

	// 1. Test User Operations
	telegramID := int64(987654321)
	userID := int64(101)

	// Check non-existent user
	u, err := db.GetUser(telegramID)
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	if u != nil {
		t.Errorf("Expected nil user, got %+v", u)
	}

	// Save user
	now := time.Now().Unix()
	uSave := &User{
		TelegramID: telegramID,
		UserID:     userID,
		CreatedAt:  now,
	}
	err = db.SaveUser(uSave)
	if err != nil {
		t.Fatalf("SaveUser failed: %v", err)
	}

	// Retrieve user
	u, err = db.GetUser(telegramID)
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	if u == nil {
		t.Fatal("Expected user to be retrieved, got nil")
	}
	if u.TelegramID != telegramID || u.UserID != userID {
		t.Errorf("Retrieved user data mismatch. Expected TG:%d User:%d, got TG:%d User:%d",
			telegramID, userID, u.TelegramID, u.UserID)
	}

	// 2. Test Recharge Request Operations
	req := &RechargeRequest{
		TelegramID:    telegramID,
		UserID:        userID,
		Amount:        150000,
		Status:        "pending",
		ReceiptFileID: "file_xyz123",
		CreatedAt:     now,
	}

	id, err := db.CreateRechargeRequest(req)
	if err != nil {
		t.Fatalf("CreateRechargeRequest failed: %v", err)
	}
	if id <= 0 {
		t.Errorf("Expected positive insert ID, got %d", id)
	}

	// Retrieve Recharge Request
	retrievedReq, err := db.GetRechargeRequest(id)
	if err != nil {
		t.Fatalf("GetRechargeRequest failed: %v", err)
	}
	if retrievedReq == nil {
		t.Fatal("Expected recharge request to be retrieved, got nil")
	}
	if retrievedReq.Amount != 150000 || retrievedReq.Status != "pending" || retrievedReq.ReceiptFileID != "file_xyz123" {
		t.Errorf("Retrieved request mismatch: %+v", retrievedReq)
	}

	// Update Recharge Request Status
	err = db.UpdateRechargeStatus(id, "approved")
	if err != nil {
		t.Fatalf("UpdateRechargeStatus failed: %v", err)
	}

	// Verify update
	retrievedReq, err = db.GetRechargeRequest(id)
	if err != nil {
		t.Fatalf("GetRechargeRequest failed: %v", err)
	}
	if retrievedReq.Status != "approved" {
		t.Errorf("Expected status to be approved, got %s", retrievedReq.Status)
	}

	// 3. Test Settings CRUD
	err = db.SetSetting("card_number", "6037-1111-2222-3333")
	if err != nil {
		t.Fatalf("SetSetting failed: %v", err)
	}

	val, err := db.GetSetting("card_number")
	if err != nil {
		t.Fatalf("GetSetting failed: %v", err)
	}
	if val != "6037-1111-2222-3333" {
		t.Errorf("Expected card_number value '6037-1111-2222-3333', got '%s'", val)
	}

	err = db.SetSetting("welcome_text", "Welcome to our bot {name}")
	if err != nil {
		t.Fatalf("SetSetting welcome_text failed: %v", err)
	}
	val, err = db.GetSetting("welcome_text")
	if err != nil {
		t.Fatalf("GetSetting welcome_text failed: %v", err)
	}
	if val != "Welcome to our bot {name}" {
		t.Errorf("Expected welcome_text value, got '%s'", val)
	}

	err = db.SetSetting("welcome_image", "file_abc_image_id")
	if err != nil {
		t.Fatalf("SetSetting welcome_image failed: %v", err)
	}
	val, err = db.GetSetting("welcome_image")
	if err != nil {
		t.Fatalf("GetSetting welcome_image failed: %v", err)
	}
	if val != "file_abc_image_id" {
		t.Errorf("Expected welcome_image value, got '%s'", val)
	}

	// 4. Test Plans CRUD
	plan := &Plan{
		SubscribeID: 99,
		Name:        "Test Plan",
		Price:       99000,
		Description: "For Testing Only",
	}

	planID, err := db.SavePlan(plan)
	if err != nil {
		t.Fatalf("SavePlan failed: %v", err)
	}

	plans, err := db.GetPlans()
	if err != nil {
		t.Fatalf("GetPlans failed: %v", err)
	}
	if len(plans) != 1 {
		t.Errorf("Expected 1 plan, got %d", len(plans))
	} else {
		if plans[0].ID != planID || plans[0].Name != "Test Plan" || plans[0].Price != 99000 || plans[0].SubscribeID != 99 {
			t.Errorf("Retrieved plan mismatch: %+v", plans[0])
		}
	}

	// Delete Plan
	err = db.DeletePlan(planID)
	if err != nil {
		t.Fatalf("DeletePlan failed: %v", err)
	}

	plans, err = db.GetPlans()
	if err != nil {
		t.Fatalf("GetPlans failed: %v", err)
	}
	if len(plans) != 0 {
		t.Errorf("Expected 0 plans after deletion, got %d", len(plans))
	}
}
