package bot

import (
	"context"
	"sync"
	"time"

	"reseller-bot/pkg/backend"
)

type RateManager struct {
	mu          sync.RWMutex
	rate        float64
	lastFetched time.Time
}

func NewRateManager() *RateManager {
	return &RateManager{}
}

func (rm *RateManager) GetRate(ctx context.Context, client *backend.Client) float64 {
	rm.mu.RLock()
	if time.Since(rm.lastFetched) < 15*time.Minute && rm.rate > 0 {
		r := rm.rate
		rm.mu.RUnlock()
		return r
	}
	rm.mu.RUnlock()

	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Double check inside write lock
	if time.Since(rm.lastFetched) < 15*time.Minute && rm.rate > 0 {
		return rm.rate
	}

	resp, err := client.GetExchangeRate(ctx)
	if err == nil && resp != nil && resp.UsdToIrt > 0 {
		rm.rate = resp.UsdToIrt
		rm.lastFetched = time.Now()
	}

	return rm.rate
}
