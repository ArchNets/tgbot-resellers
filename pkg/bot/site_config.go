package bot

import (
	"context"
	"log"
	"sync"
	"time"

	"reseller-bot/pkg/backend"
)

type SiteConfigManager struct {
	mu          sync.RWMutex
	data        *backend.SiteConfigData
	lastFetched time.Time
}

func NewSiteConfigManager() *SiteConfigManager {
	return &SiteConfigManager{}
}

func (scm *SiteConfigManager) GetSiteConfig(ctx context.Context, client *backend.Client) *backend.SiteConfigData {
	scm.mu.RLock()
	if time.Since(scm.lastFetched) < 15*time.Minute && scm.data != nil {
		d := scm.data
		scm.mu.RUnlock()
		return d
	}
	scm.mu.RUnlock()

	scm.mu.Lock()
	defer scm.mu.Unlock()

	if time.Since(scm.lastFetched) < 15*time.Minute && scm.data != nil {
		return scm.data
	}

	if client != nil {
		data, err := client.GetSiteConfig(ctx)
		if err == nil && data != nil {
			scm.data = data
			scm.lastFetched = time.Now()
		} else if err != nil {
			log.Printf("Failed to fetch site config: %v", err)
		}
	}

	return scm.data
}
