package bot

import (
	"testing"
)

func TestFormatUserBalance(t *testing.T) {
	// Rate > 0: Convert 500 cents ($5.00) at 105,000 IRR/USD
	// 500 cents * 105000 / 100 = 525,000 Toman
	rate := 105000.0
	cents := int64(500)
	got := FormatUserBalance(cents, rate)
	expected := "۵۲۵٬۰۰۰ تومان"
	if got != expected {
		t.Errorf("FormatUserBalance(%d, %f) = %q; want %q", cents, rate, got, expected)
	}

	// Rate == 0: Fallback to $X.XX
	rateZero := 0.0
	gotFallback := FormatUserBalance(cents, rateZero)
	expectedFallback := "$5.00"
	if gotFallback != expectedFallback {
		t.Errorf("FormatUserBalance(%d, %f) = %q; want %q", cents, rateZero, gotFallback, expectedFallback)
	}
}
