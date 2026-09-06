package bot

import (
	"testing"
	"time"

	"reseller-bot/pkg/backend"
	"reseller-bot/pkg/config"
)

func TestIsSubscriptionEligibleForReminder(t *testing.T) {
	nowMs := time.Now().Add(12 * time.Hour).UnixMilli()

	// Bot A: Reseller Bot with BotID = 10
	botA := &Bot{
		cfg: &config.Config{
			BotID: 10,
		},
	}

	// Bot B: Main Bot with BotID = 0
	botMain := &Bot{
		cfg: &config.Config{
			BotID: 0,
		},
	}

	// Sub 1: Direct Main Platform Subscription (reseller_subscription_id = 0, bot_id = 0)
	subMain := &backend.SubscriptionItem{
		ID:                     1,
		ExpireTime:             nowMs,
		ResellerSubscriptionID: 0,
		Subscribe: backend.SubscriptionPlan{
			Name:                   "Main Plan",
			BotID:                  0,
			ResellerSubscriptionID: 0,
		},
	}

	// Sub 2: Reseller Subscription for Bot 10 (reseller_subscription_id = 50, bot_id = 10)
	subBot10 := &backend.SubscriptionItem{
		ID:                     2,
		ExpireTime:             nowMs,
		ResellerSubscriptionID: 50,
		Subscribe: backend.SubscriptionPlan{
			Name:                   "Bot 10 Plan",
			BotID:                  10,
			ResellerSubscriptionID: 50,
		},
	}

	// Sub 3: Reseller Subscription for Bot 12 (reseller_subscription_id = 50, bot_id = 12)
	subBot12 := &backend.SubscriptionItem{
		ID:                     3,
		ExpireTime:             nowMs,
		ResellerSubscriptionID: 50,
		Subscribe: backend.SubscriptionPlan{
			Name:                   "Bot 12 Plan",
			BotID:                  12,
			ResellerSubscriptionID: 50,
		},
	}

	// Sub 4: Reseller Shared Subscription (reseller_subscription_id = 50, bot_id = 0)
	subSharedReseller := &backend.SubscriptionItem{
		ID:                     4,
		ExpireTime:             nowMs,
		ResellerSubscriptionID: 50,
		Subscribe: backend.SubscriptionPlan{
			Name:                   "Shared Reseller Plan",
			BotID:                  0,
			ResellerSubscriptionID: 50,
		},
	}

	// Sub 5: Expired Subscription (ExpireTime = 0)
	subExpired := &backend.SubscriptionItem{
		ID:                     5,
		ExpireTime:             0,
		ResellerSubscriptionID: 50,
		Subscribe: backend.SubscriptionPlan{
			Name:                   "Expired Plan",
			BotID:                  10,
			ResellerSubscriptionID: 50,
		},
	}

	// Assertions for Reseller Bot A (BotID = 10)
	if botA.isSubscriptionEligibleForReminder(subMain) {
		t.Errorf("Reseller Bot A MUST NOT be eligible for direct Main Platform subscription")
	}
	if !botA.isSubscriptionEligibleForReminder(subBot10) {
		t.Errorf("Reseller Bot A MUST be eligible for its own Bot 10 subscription")
	}
	if botA.isSubscriptionEligibleForReminder(subBot12) {
		t.Errorf("Reseller Bot A MUST NOT be eligible for Bot 12 subscription")
	}
	if !botA.isSubscriptionEligibleForReminder(subSharedReseller) {
		t.Errorf("Reseller Bot A MUST be eligible for shared reseller subscription")
	}
	if botA.isSubscriptionEligibleForReminder(subExpired) {
		t.Errorf("Reseller Bot A MUST NOT be eligible for expired subscription with ExpireTime <= 0")
	}

	// Assertions for Main Bot (BotID = 0)
	if !botMain.isSubscriptionEligibleForReminder(subMain) {
		t.Errorf("Main Bot MUST be eligible for direct Main Platform subscription")
	}
	if botMain.isSubscriptionEligibleForReminder(subBot10) {
		t.Errorf("Main Bot MUST NOT be eligible for reseller Bot 10 subscription")
	}
	if botMain.isSubscriptionEligibleForReminder(subBot12) {
		t.Errorf("Main Bot MUST NOT be eligible for reseller Bot 12 subscription")
	}
	if botMain.isSubscriptionEligibleForReminder(subSharedReseller) {
		t.Errorf("Main Bot MUST NOT be eligible for shared reseller subscription")
	}
	if botMain.isSubscriptionEligibleForReminder(subExpired) {
		t.Errorf("Main Bot MUST NOT be eligible for expired subscription")
	}
}
