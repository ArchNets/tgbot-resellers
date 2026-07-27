package bot

import (
	"strings"
	"testing"
	"time"

	"reseller-bot/pkg/backend"
	"reseller-bot/pkg/config"
)

func TestCalculateExpiryMs(t *testing.T) {
	tests := []struct {
		unitTime string
		expectGT bool
	}{
		{"month", true},
		{"quarter", true},
		{"half_year", true},
		{"year", true},
		{"", false},
		{"unknown", false},
	}

	nowMs := time.Now().UnixMilli()
	for _, tt := range tests {
		got := calculateExpiryMs(tt.unitTime)
		if tt.expectGT {
			if got <= nowMs {
				t.Errorf("calculateExpiryMs(%q) = %d; expected > %d", tt.unitTime, got, nowMs)
			}
		} else {
			if got != 0 {
				t.Errorf("calculateExpiryMs(%q) = %d; expected 0", tt.unitTime, got)
			}
		}
	}
}

func TestSubscriptionItemGetName(t *testing.T) {
	sub1 := backend.SubscriptionItem{
		CustomName: "My Custom",
		Subscribe: backend.SubscriptionPlan{
			Name: "Default Plan",
		},
	}
	if sub1.GetName() != "My Custom" {
		t.Errorf("expected 'My Custom', got %q", sub1.GetName())
	}

	sub2 := backend.SubscriptionItem{
		CustomName: "",
		Subscribe: backend.SubscriptionPlan{
			Name: "Default Plan",
		},
	}
	if sub2.GetName() != "Default Plan" {
		t.Errorf("expected 'Default Plan', got %q", sub2.GetName())
	}
}

func TestAnalyzeDownloadNodes(t *testing.T) {
	nodes := []backend.DownloadNode{
		{NodeID: 1, Name: "Node 1", Protocol: "openvpn"},
		{NodeID: 2, Name: "Node 2", Protocol: "vless", Formats: []string{"wireguard"}},
		{NodeID: 3, Name: "Node 3", Protocol: "awg"},
	}

	ps := analyzeDownloadNodes(nodes)
	if !ps.HasOpenVPN || len(ps.OpenVPNNodes) != 1 {
		t.Errorf("expected 1 OpenVPN node, got %d", len(ps.OpenVPNNodes))
	}
	if !ps.HasWireGuard || len(ps.WireGuardNodes) != 2 {
		t.Errorf("expected 2 WireGuard nodes, got %d", len(ps.WireGuardNodes))
	}
}

func TestGetSubscribeLink(t *testing.T) {
	// Case 1: Site config with multiple comma-separated domains
	scm := NewSiteConfigManager()
	scm.data = &backend.SiteConfigData{}
	scm.data.Subscribe.SubscribeDomain = "sub1.example.com, sub2.example.com"
	scm.data.Subscribe.SubscribePath = "/sub/config"
	scm.lastFetched = time.Now()

	b1 := &Bot{
		cfg:           &config.Config{BackendURL: "https://backend.example.com"},
		siteConfigMgr: scm,
	}

	link1 := b1.getSubscribeLink("token123")
	expected1 := "https://sub1.example.com/sub/config?token=token123"
	if link1 != expected1 {
		t.Errorf("case 1 got %q, want %q", link1, expected1)
	}

	// Case 2: Fallback to Site.Host when SubscribeDomain is empty
	scm2 := NewSiteConfigManager()
	scm2.data = &backend.SiteConfigData{}
	scm2.data.Site.Host = "panel.site.com"
	scm2.lastFetched = time.Now()

	b2 := &Bot{
		cfg:           &config.Config{BackendURL: "https://backend.example.com"},
		siteConfigMgr: scm2,
	}

	link2 := b2.getSubscribeLink("token123")
	expected2 := "https://panel.site.com/v1/subscribe/config?token=token123"
	if link2 != expected2 {
		t.Errorf("case 2 got %q, want %q", link2, expected2)
	}

	// Case 3: Fallback to backend_url host when site config has no domain
	scm3 := NewSiteConfigManager()
	b3 := &Bot{
		cfg:           &config.Config{BackendURL: "https://api.backend.com:8443"},
		siteConfigMgr: scm3,
	}

	link3 := b3.getSubscribeLink("token123")
	expected3 := "https://api.backend.com:8443/v1/subscribe/config?token=token123"
	if link3 != expected3 {
		t.Errorf("case 3 got %q, want %q", link3, expected3)
	}
}

func TestCustomNameDefaulting(t *testing.T) {
	planName := "VIP Plan"
	inputDash := "-"
	inputEmpty := ""
	inputCustom := "My Custom Subscription"

	formatDefault := func(inp string) string {
		if inp == "-" || inp == "" {
			return planName + " " + time.Now().Format("2006-01-02")
		}
		return inp
	}

	expectedDefault := planName + " " + time.Now().Format("2006-01-02")
	if got := formatDefault(inputDash); got != expectedDefault {
		t.Errorf("dash input got %q, want %q", got, expectedDefault)
	}
	if got := formatDefault(inputEmpty); got != expectedDefault {
		t.Errorf("empty input got %q, want %q", got, expectedDefault)
	}
	if got := formatDefault(inputCustom); got != inputCustom {
		t.Errorf("custom input got %q, want %q", got, inputCustom)
	}
}

func TestSubscriptionDetailKeyboard(t *testing.T) {
	kbOnlyBack := SubscriptionDetailKeyboard(123, false, false, 1)
	if kbOnlyBack == nil || len(kbOnlyBack.InlineKeyboard) != 1 {
		t.Errorf("expected 1 row (back button), got %+v", kbOnlyBack)
	}

	kbWithOvpn := SubscriptionDetailKeyboard(123, true, false, 1)
	if kbWithOvpn == nil || len(kbWithOvpn.InlineKeyboard) != 2 {
		t.Errorf("expected 2 rows (OpenVPN + back button), got %+v", kbWithOvpn)
	}
}

func TestFormatProgressBar(t *testing.T) {
	if got := FormatProgressBar(40, 100); got != "[▓▓▓▓░░░░░░] 40%" {
		t.Errorf("expected [▓▓▓▓░░░░░░] 40%%, got %q", got)
	}
	if got := FormatProgressBar(0, 100); got != "[░░░░░░░░░░] 0%" {
		t.Errorf("expected [░░░░░░░░░░] 0%%, got %q", got)
	}
	if got := FormatProgressBar(120, 100); got != "[▓▓▓▓▓▓▓▓▓▓] 100%" {
		t.Errorf("expected [▓▓▓▓▓▓▓▓▓▓] 100%%, got %q", got)
	}
}

func TestBuildSubscriptionDetailText(t *testing.T) {
	itemZeroLimit := &backend.SubscriptionItem{
		ID:          1,
		CustomName:  "Unlimited Device Sub",
		Traffic:     107374182400, // 100 GB
		Upload:      10737418240,  // 10 GB
		Download:    32212254720,  // 30 GB (Total 40 GB = 40%)
		OnlineCount: 2,
		StartTime:   1770000000000,
		Subscribe: backend.SubscriptionPlan{
			Name:        "1 Month VIP",
			DeviceLimit: 0,
			NodeTags:    []string{"VIP"},
		},
	}

	subLink := "https://panel.example.com/v1/subscribe/config?token=abc"

	txtZero := BuildSubscriptionDetailText(nil, itemZeroLimit, subLink, "fa")
	if !testing.Short() {
		t.Logf("Rendered text (DeviceLimit=0):\n%s", txtZero)
	}

	if !strings.Contains(txtZero, "📦 *اشتراک: Unlimited Device Sub") {
		t.Errorf("missing title with box emoji in output: %s", txtZero)
	}
	if !strings.Contains(txtZero, "- نوع پلان: 1 Month VIP") {
		t.Errorf("missing plan type in output: %s", txtZero)
	}
	if !strings.Contains(txtZero, "- دسته: VIP") {
		t.Errorf("missing category line in output: %s", txtZero)
	}
	if !strings.Contains(txtZero, "- دستگاه‌ها: 2 متصل (بدون محدودیت)") {
		t.Errorf("missing zero device limit string in output: %s", txtZero)
	}
	if strings.Contains(txtZero, "سرعت") {
		t.Errorf("speed line should be removed: %s", txtZero)
	}
	if strings.Contains(txtZero, "پروتکل") {
		t.Errorf("protocol line should be removed: %s", txtZero)
	}
	if !strings.Contains(txtZero, "[▓▓▓▓░░░░░░] 40%") {
		t.Errorf("missing progress bar in output: %s", txtZero)
	}

	itemWithLimit := &backend.SubscriptionItem{
		ID:          2,
		CustomName:  "Limited Sub",
		Traffic:     50 * 1024 * 1024 * 1024,
		Upload:      5 * 1024 * 1024 * 1024,
		Download:    5 * 1024 * 1024 * 1024,
		OnlineCount: 1,
		Subscribe: backend.SubscriptionPlan{
			Name:        "Pro Plan",
			DeviceLimit: 3,
			NodeTags:    nil, // No tags
		},
	}

	txtLimit := BuildSubscriptionDetailText(nil, itemWithLimit, subLink, "fa")
	if !strings.Contains(txtLimit, "- دستگاه‌ها: 1 از 3 متصل") {
		t.Errorf("missing device limit string in output: %s", txtLimit)
	}
	if strings.Contains(txtLimit, "- دسته:") {
		t.Errorf("category line should be omitted when no tags: %s", txtLimit)
	}
}
