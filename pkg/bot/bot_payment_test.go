package bot

import (
	"testing"

	"reseller-bot/pkg/backend"
)

func TestPaymentMethodSelectionKeyboard(t *testing.T) {
	methods := []backend.PaymentMethodItem{
		{
			ID:       1,
			Name:     "Card Transfer",
			Platform: "CardToCard",
			Enable:   true,
		},
		{
			ID:       2,
			Name:     "Tether USDT",
			Platform: "Tronado",
			Enable:   true,
		},
		{
			ID:       3,
			Name:     "Disabled Method",
			Platform: "Stripe",
			Enable:   false,
		},
	}

	kb := PaymentMethodSelectionKeyboard(methods)

	// Should only have 2 buttons because method 3 is disabled
	if len(kb.InlineKeyboard) != 2 {
		t.Fatalf("Expected 2 keyboard rows, got %d", len(kb.InlineKeyboard))
	}

	row0 := kb.InlineKeyboard[0]
	if len(row0) != 1 {
		t.Fatalf("Expected 1 button in row 0, got %d", len(row0))
	}
	if *row0[0].CallbackData != "topup_c2c_1" {
		t.Errorf("Expected callback topup_c2c_1, got %s", *row0[0].CallbackData)
	}

	row1 := kb.InlineKeyboard[1]
	if len(row1) != 1 {
		t.Fatalf("Expected 1 button in row 1, got %d", len(row1))
	}
	if *row1[0].CallbackData != "topup_gw_2" {
		t.Errorf("Expected callback topup_gw_2, got %s", *row1[0].CallbackData)
	}
}
