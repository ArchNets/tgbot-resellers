package bot

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/url"
	"strings"
	"time"

	"reseller-bot/pkg/backend"
	"reseller-bot/pkg/config"
	"reseller-bot/pkg/db"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	qrcode "github.com/skip2/go-qrcode"
)

type Bot struct {
	cfg           *config.Config
	api           *tgbotapi.BotAPI
	client        *backend.Client
	db            *db.DB
	session       *SessionManager
	rateMgr       *RateManager
	siteConfigMgr *SiteConfigManager
}

func NewBot(cfg *config.Config, database *db.DB, client *backend.Client) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		return nil, err
	}

	return &Bot{
		cfg:           cfg,
		api:           api,
		client:        client,
		db:            database,
		session:       NewSessionManager(),
		rateMgr:       NewRateManager(),
		siteConfigMgr: NewSiteConfigManager(),
	}, nil
}

func (b *Bot) Start(ctx context.Context) {
	log.Printf("Authorized on account %s", b.api.Self.UserName)

	go b.siteConfigMgr.GetSiteConfig(ctx, b.client)
	go b.startPoller(ctx)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping bot...")
			return
		case update := <-updates:
			go b.handleUpdate(update)
		}
	}
}

func (b *Bot) startPoller(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	lastExpiryCheck := time.Time{}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			b.syncRechargeStatus(ctx)

			if time.Since(lastExpiryCheck) >= 24*time.Hour {
				b.checkExpiryReminders(ctx)
				lastExpiryCheck = time.Now()
			}
		}
	}
}

func (b *Bot) syncRechargeStatus(ctx context.Context) {
	resp, err := b.client.GetRechargeList(ctx, "customer_topup", "", 1, 100)
	if err != nil || resp == nil {
		return
	}

	for _, order := range resp.List {
		if order.Status == "approved" || order.Status == "rejected" {
			notified, err := b.db.IsRechargeNotified(order.ID, order.Status)
			if err != nil || notified {
				continue
			}

			user, err := b.db.GetUserByBackendID(order.CustomerUserID)
			if err != nil || user == nil {
				continue
			}

			userObj := &tgbotapi.User{ID: user.TelegramID}
			var msgText string
			if order.Status == "approved" {
				msgText = Tr(getLang(userObj), "balance_credited", FormatMoney(order.OriginalAmount))
			} else {
				reason := order.RejectReason
				if reason == "" {
					reason = "-"
				}
				msgText = Tr(getLang(userObj), "recharge_rejected_user", FormatMoney(order.OriginalAmount), reason)
			}

			b.sendSimpleMessage(user.TelegramID, msgText)
			_ = b.db.SaveRechargeNotified(order.ID, order.Status)
		}
	}
}

func (b *Bot) getSubscriptionConfigs(ctx context.Context, subID int64) []string {
	nodes, err := b.client.GetDownloadNodes(ctx, subID)
	if err != nil || len(nodes) == 0 {
		return nil
	}
	var configs []string
	for _, node := range nodes {
		prof, err := b.client.GetUserSubscribeProfile(ctx, subID, node.NodeID, "")
		if err == nil && prof != nil && len(prof.Content) > 0 {
			cfgStr := strings.TrimSpace(string(prof.Content))
			if cfgStr != "" {
				configs = append(configs, cfgStr)
			}
		}
	}
	return configs
}

func (b *Bot) checkExpiryReminders(ctx context.Context) {
	remEnabled, _ := b.db.GetSetting("reminders_enabled")
	if remEnabled == "off" {
		return
	}

	users, err := b.db.GetAllUsers()
	if err != nil || len(users) == 0 {
		return
	}

	for _, u := range users {
		subs, err := b.client.GetUserSubscriptions(ctx, u.UserID, 1, 100)
		if err != nil || subs == nil || len(subs.List) == 0 {
			continue
		}

		userObj := &tgbotapi.User{ID: u.TelegramID}
		for _, sub := range subs.List {
			if sub.ExpireTime <= 0 {
				continue
			}

			rem := time.Until(time.Unix(sub.ExpireTime/1000, 0))
			if rem > 0 && rem <= 24*time.Hour {
				sent, err := b.db.IsReminderSent(sub.ID, "1day")
				if err == nil && !sent {
					msgText := Tr(getLang(userObj), "expiry_reminder_1day", escapeMarkdown(sub.GetName()))
					b.sendSimpleMessage(u.TelegramID, msgText)
					_ = b.db.SaveReminderSent(sub.ID, "1day")
				}
			} else if rem > 0 && rem <= 72*time.Hour {
				sent, err := b.db.IsReminderSent(sub.ID, "3days")
				if err == nil && !sent {
					msgText := Tr(getLang(userObj), "expiry_reminder_3days", escapeMarkdown(sub.GetName()))
					b.sendSimpleMessage(u.TelegramID, msgText)
					_ = b.db.SaveReminderSent(sub.ID, "3days")
				}
			}
		}
	}
}

type ProtocolSupport struct {
	HasOpenVPN     bool
	HasWireGuard   bool
	OpenVPNNodes   []backend.DownloadNode
	WireGuardNodes []backend.DownloadNode
}

func analyzeDownloadNodes(nodes []backend.DownloadNode) ProtocolSupport {
	var ps ProtocolSupport
	for _, n := range nodes {
		proto := strings.ToLower(n.Protocol)
		var isOvpn, isWg bool

		if proto == "openvpn" {
			isOvpn = true
		}
		if proto == "wireguard" || proto == "awg" {
			isWg = true
		}

		for _, f := range n.Formats {
			fl := strings.ToLower(f)
			if fl == "openvpn" {
				isOvpn = true
			}
			if fl == "wireguard" || fl == "awg" {
				isWg = true
			}
		}

		if isOvpn {
			ps.HasOpenVPN = true
			ps.OpenVPNNodes = append(ps.OpenVPNNodes, n)
		}
		if isWg {
			ps.HasWireGuard = true
			ps.WireGuardNodes = append(ps.WireGuardNodes, n)
		}
	}
	return ps
}

func parseFirstDomain(domainStr string) string {
	domainStr = strings.TrimSpace(domainStr)
	if domainStr == "" {
		return ""
	}
	fields := strings.FieldsFunc(domainStr, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n'
	})
	if len(fields) > 0 {
		d := strings.TrimSpace(fields[0])
		d = strings.TrimPrefix(d, "https://")
		d = strings.TrimPrefix(d, "http://")
		d = strings.TrimSuffix(d, "/")
		return d
	}
	return ""
}

func (b *Bot) getSubscribeLink(token string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var domain, path string
	if b.siteConfigMgr != nil {
		siteCfg := b.siteConfigMgr.GetSiteConfig(ctx, b.client)
		if siteCfg != nil {
			domain = parseFirstDomain(siteCfg.Subscribe.SubscribeDomain)
			if domain == "" {
				domain = parseFirstDomain(siteCfg.Site.Host)
			}
			path = strings.TrimSpace(siteCfg.Subscribe.SubscribePath)
		}
	}

	if domain == "" && b.client != nil {
		u, err := url.Parse(b.client.GetBaseURL())
		if err == nil && u.Host != "" {
			domain = u.Host
		}
	}
	if domain == "" && b.cfg != nil {
		u, err := url.Parse(b.cfg.BackendURL)
		if err == nil && u.Host != "" {
			domain = u.Host
		}
	}
	if domain == "" {
		domain = "panel.archnets.com"
	}

	if path == "" {
		path = "/v1/subscribe/config"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return fmt.Sprintf("https://%s%s?token=%s", domain, path, url.QueryEscape(token))
}

func (b *Bot) sendSubscriptionMessage(chatID int64, text string, subLink string, replyMarkup *tgbotapi.InlineKeyboardMarkup) {
	qrEnabled, _ := b.db.GetSetting("qr_enabled")
	if qrEnabled == "" {
		qrEnabled = "on"
	}

	hasKeyboard := replyMarkup != nil && len(replyMarkup.InlineKeyboard) > 0

	if qrEnabled == "on" && strings.TrimSpace(subLink) != "" {
		pngBytes, err := qrcode.Encode(subLink, qrcode.Medium, 256)
		if err != nil {
			log.Printf("sendSubscriptionMessage QR encode failed for %d: %v", chatID, err)
		} else if len(pngBytes) > 0 {
			photoUpload := tgbotapi.NewPhoto(chatID, tgbotapi.FileBytes{
				Name:  "qrcode.png",
				Bytes: pngBytes,
			})
			photoUpload.Caption = text
			photoUpload.ParseMode = tgbotapi.ModeMarkdown
			if hasKeyboard {
				photoUpload.ReplyMarkup = *replyMarkup
			}
			if _, sendErr := b.api.Send(photoUpload); sendErr == nil {
				return
			} else {
				log.Printf("sendSubscriptionMessage photo send failed for %d: %v", chatID, sendErr)
			}
		}
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	if hasKeyboard {
		msg.ReplyMarkup = *replyMarkup
	}
	if _, err := b.api.Send(msg); err != nil {
		log.Printf("sendSubscriptionMessage text message send failed for %d: %v", chatID, err)
	}
}

func (b *Bot) handleUpdate(update tgbotapi.Update) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in update handler: %v", r)
		}
	}()

	if update.Message != nil {
		b.handleMessage(update.Message)
	} else if update.CallbackQuery != nil {
		b.handleCallbackQuery(update.CallbackQuery)
	}
}

func (b *Bot) isOwner(chatID int64) bool {
	if len(b.cfg.AdminChatIDs) > 0 {
		return chatID == b.cfg.AdminChatIDs[0]
	}
	return false
}

func (b *Bot) isAdmin(chatID int64) bool {
	for _, id := range b.cfg.AdminChatIDs {
		if chatID == id {
			return true
		}
	}
	isStaff, err := b.db.IsStaff(chatID)
	if err == nil && isStaff {
		return true
	}
	return false
}

func (b *Bot) getAllAdminIDs() []int64 {
	adminMap := make(map[int64]bool)
	for _, id := range b.cfg.AdminChatIDs {
		adminMap[id] = true
	}
	staff, err := b.db.GetStaffList()
	if err == nil {
		for _, s := range staff {
			adminMap[s.TelegramID] = true
		}
	}
	var result []int64
	for id := range adminMap {
		result = append(result, id)
	}
	return result
}

func FormatProgressBar(used int64, total int64) string {
	if total <= 0 {
		return "[░░░░░░░░░░] 0%"
	}
	pct := (float64(used) / float64(total)) * 100.0
	if pct > 100.0 {
		pct = 100.0
	}
	if pct < 0.0 {
		pct = 0.0
	}
	filled := int(math.Round(pct / 10.0))
	if filled > 10 {
		filled = 10
	}
	if filled < 0 {
		filled = 0
	}
	bar := strings.Repeat("▓", filled) + strings.Repeat("░", 10-filled)
	return fmt.Sprintf("[%s] %.0f%%", bar, pct)
}

func BuildSubscriptionDetailText(database *db.DB, item *backend.SubscriptionItem, subLink string, lang string) string {
	usedTraffic := item.Upload + item.Download
	remTraffic := item.Traffic - usedTraffic
	if remTraffic < 0 {
		remTraffic = 0
	}
	remTrafficStr := Tr(lang, "remaining_traffic", FormatTraffic(remTraffic))
	titleWithRem := fmt.Sprintf("%s (%s)", item.GetName(), remTrafficStr)

	planType := strings.TrimSpace(item.Subscribe.Name)
	if planType == "" {
		planType = "—"
	}

	categoryLine := ""
	if len(item.Subscribe.NodeTags) > 0 {
		var resolved []string
		for _, t := range item.Subscribe.NodeTags {
			tag := strings.TrimSpace(t)
			if tag == "" {
				continue
			}
			if database != nil {
				disp, err := database.GetTagMapping(tag)
				if err == nil && disp != "" {
					resolved = append(resolved, disp)
					continue
				}
			}
			resolved = append(resolved, tag)
		}
		if len(resolved) > 0 {
			categoryLine = fmt.Sprintf("- دسته: %s\n", escapeMarkdown(strings.Join(resolved, "، ")))
		}
	}

	var devicesStr string
	if item.Subscribe.DeviceLimit > 0 {
		devicesStr = fmt.Sprintf("%d از %d متصل", item.OnlineCount, item.Subscribe.DeviceLimit)
	} else {
		devicesStr = fmt.Sprintf("%d متصل (بدون محدودیت)", item.OnlineCount)
	}

	startDateStr := "—"
	if item.StartTime > 0 {
		startDateStr = time.Unix(item.StartTime/1000, 0).Format("2006-01-02")
	}

	totalTrafficStr := FormatTraffic(item.Traffic)
	usedTrafficStr := FormatTraffic(usedTraffic)
	progressBarStr := FormatProgressBar(usedTraffic, item.Traffic)

	expStr := "بدون انقضا"
	if item.ExpireTime > 0 {
		expDateStr := time.Unix(item.ExpireTime/1000, 0).Format("2006-01-02")
		nowMs := time.Now().UnixMilli()
		daysRem := int(math.Ceil(float64(item.ExpireTime-nowMs) / float64(86400*1000)))
		if daysRem > 0 {
			expStr = fmt.Sprintf("%s (%d روز مانده)", expDateStr, daysRem)
		} else {
			expStr = fmt.Sprintf("%s (منقضی شده)", expDateStr)
		}
	}

	return fmt.Sprintf(MsgSubDetail,
		escapeMarkdown(titleWithRem),
		escapeMarkdown(planType),
		categoryLine,
		escapeMarkdown(devicesStr),
		escapeMarkdown(startDateStr),
		escapeMarkdown(totalTrafficStr),
		escapeMarkdown(usedTrafficStr),
		escapeMarkdown(progressBarStr),
		escapeMarkdown(expStr),
		subLink,
	)
}

