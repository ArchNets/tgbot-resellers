package bot

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
	"unicode"

	"reseller-bot/pkg/backend"
	"reseller-bot/pkg/db"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) handleMessage(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	log.Printf("Received message from %d: %s", chatID, msg.Text)

	// Check if user is admin or staff
	isAdmin := b.isAdmin(chatID)
	isOwner := b.isOwner(chatID)

	// 1. Get or register user
	u, err := b.db.GetUser(chatID)
	if err != nil {
		log.Printf("DB error fetching user: %v", err)
		b.sendSimpleMessage(chatID, MsgGeneralError)
		return
	}

	if u == nil {
		// Register user on Core Backend
		firstName := msg.From.FirstName
		lastName := msg.From.LastName
		username := msg.From.UserName
		langCode := msg.From.LanguageCode
		if langCode == "" {
			langCode = "fa"
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		resp, err := b.client.RegisterUser(ctx, &backend.UserRegisterRequest{
			TelegramID:   chatID,
			FirstName:    firstName,
			LastName:     lastName,
			Username:     username,
			LanguageCode: langCode,
		})
		if err != nil {
			log.Printf("Backend register error for %d: %v", chatID, err)
			b.sendSimpleMessage(chatID, MsgGeneralError)
			return
		}

		u = &db.User{
			TelegramID: chatID,
			UserID:     resp.UserID,
			CreatedAt:  time.Now().Unix(),
		}
		if err := b.db.SaveUser(u); err != nil {
			log.Printf("DB error saving user %d: %v", chatID, err)
		}
	}

	// 2. Handle state-based inputs
	sess := b.session.Get(chatID)
	if sess.State != StateNone {
		b.handleStateMessage(msg, u, sess, isAdmin)
		return
	}

	// 3. Handle commands & standard menu buttons
	switch msg.Text {
	case "/start", BtnBack:
		b.session.Clear(chatID)
		args := strings.TrimSpace(msg.CommandArguments())

		// First-user auto-bind fallback if no admin configured yet
		if len(b.cfg.AdminChatIDs) == 0 && chatID > 0 {
			b.addAdminChatID(chatID)
			isAdmin = true
			b.sendSimpleMessage(chatID, "👑 شما به عنوان مدیر اصلی این ربات ثبت شدید.")
		}

		// Handle pair/login code authentication: /start login_123456
		if strings.HasPrefix(args, "login_") || strings.HasPrefix(args, "pair_") {
			codeDigits := strings.TrimPrefix(strings.TrimPrefix(args, "login_"), "pair_")
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			verifyResp, err := b.client.VerifyPairCode(ctx, codeDigits, chatID)
			if err == nil && verifyResp != nil && verifyResp.Status {
				b.addAdminChatID(chatID)
				isAdmin = true
				b.sendSimpleMessage(chatID, "✅ حساب تلگرام شما با موفقیت به عنوان مدیر ربات فعال شد!")
			} else {
				b.sendSimpleMessage(chatID, "❌ کد ورود منقضی شده یا نامعتبر است.")
			}
		}

		name := msg.From.FirstName
		if name == "" {
			name = "Co Worker"
		} else if msg.From.LastName != "" {
			name += " " + msg.From.LastName
		}

		welcomeText, _ := b.db.GetSetting("welcome_text")
		if welcomeText == "" {
			welcomeText = MsgWelcome
		}
		welcomeText = strings.ReplaceAll(welcomeText, "{name}", escapeMarkdown(name))

		welcomeImage, _ := b.db.GetSetting("welcome_image")
		if welcomeImage != "" {
			reply := tgbotapi.NewPhoto(chatID, tgbotapi.FileID(welcomeImage))
			reply.Caption = welcomeText
			reply.ParseMode = tgbotapi.ModeMarkdown
			reply.ReplyMarkup = MainMenuKeyboard(isAdmin)
			b.api.Send(reply)
		} else {
			reply := tgbotapi.NewMessage(chatID, welcomeText)
			reply.ParseMode = tgbotapi.ModeMarkdown
			reply.ReplyMarkup = MainMenuKeyboard(isAdmin)
			b.api.Send(reply)
		}

	case BtnAccountInfo:
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		resp, err := b.client.RegisterUser(ctx, &backend.UserRegisterRequest{
			TelegramID: chatID,
		})
		if err != nil {
			log.Printf("Backend info fetch error: %v", err)
			b.sendSimpleMessage(chatID, MsgGeneralError)
			return
		}

		rate := b.rateMgr.GetRate(ctx, b.client)
		formattedBalance := FormatUserBalance(resp.Balance, rate)
		text := fmt.Sprintf(MsgAccountInfo, chatID, resp.UserID, formattedBalance)
		reply := tgbotapi.NewMessage(chatID, text)
		reply.ParseMode = tgbotapi.ModeMarkdown
		reply.ReplyMarkup = MainMenuKeyboard(isAdmin)
		b.api.Send(reply)

	case BtnBuyService:
		b.handleBuyService(chatID)

	case BtnMySubscriptions:
		if !b.checkChannelGate(chatID) {
			return
		}
		b.renderSubscriptionsListPage(chatID, 0, 1, msg.From)

	case BtnTopUpBalance:
		if !b.checkChannelGate(chatID) {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		card, err := b.client.GetPaymentCard(ctx)
		if err != nil || card == nil || !card.Enabled || strings.TrimSpace(card.CardNumber) == "" {
			b.sendSimpleMessage(chatID, Tr(getLang(msg.From), "payments_unavailable"))
			return
		}

		text := fmt.Sprintf(MsgTopUpCardInfo, card.CardNumber, card.CardOwner)
		reply := tgbotapi.NewMessage(chatID, text)
		reply.ParseMode = tgbotapi.ModeMarkdown
		reply.ReplyMarkup = BackKeyboard()
		b.api.Send(reply)
		b.session.SetState(chatID, StateAwaitingAmount)

	case BtnContactSupport:
		supportText, _ := b.db.GetSetting("support_text")
		if supportText == "" {
			supportText = MsgContactSupportText
		}
		supportImage, _ := b.db.GetSetting("support_image")
		if supportImage != "" {
			reply := tgbotapi.NewPhoto(chatID, tgbotapi.FileID(supportImage))
			reply.Caption = supportText
			reply.ParseMode = tgbotapi.ModeMarkdown
			reply.ReplyMarkup = MainMenuKeyboard(isAdmin)
			b.api.Send(reply)
		} else {
			reply := tgbotapi.NewMessage(chatID, supportText)
			reply.ParseMode = tgbotapi.ModeMarkdown
			reply.ReplyMarkup = MainMenuKeyboard(isAdmin)
			b.api.Send(reply)
		}

	case BtnAdminPanel, "/admin":
		if !isAdmin {
			b.sendSimpleMessage(chatID, "⚠️ شما دسترسی به این بخش را ندارید.")
			return
		}
		b.session.Clear(chatID)
		reply := tgbotapi.NewMessage(chatID, MsgAdminPanelWelcome)
		reply.ParseMode = tgbotapi.ModeMarkdown
		reply.ReplyMarkup = AdminMenuKeyboard(isOwner)
		b.api.Send(reply)

	case BtnAdminCardSettings:
		if !isOwner {
			b.sendSimpleMessage(chatID, Tr(getLang(msg.From), "staff_owner_only"))
			return
		}
		reply := tgbotapi.NewMessage(chatID, MsgAdminAwaitingCardNumber)
		reply.ReplyMarkup = BackKeyboard()
		b.api.Send(reply)
		b.session.SetState(chatID, StateAdminAwaitingCardNumber)

	case BtnAdminStaffSettings:
		if !isOwner {
			b.sendSimpleMessage(chatID, Tr(getLang(msg.From), "staff_owner_only"))
			return
		}
		staffList, _ := b.db.GetStaffList()
		reply := tgbotapi.NewMessage(chatID, Tr(getLang(msg.From), "staff_menu"))
		reply.ParseMode = tgbotapi.ModeMarkdown
		reply.ReplyMarkup = StaffInlineKeyboard(staffList)
		b.api.Send(reply)

	case BtnAdminChannelGate:
		if !isOwner {
			b.sendSimpleMessage(chatID, Tr(getLang(msg.From), "staff_owner_only"))
			return
		}
		currentChannel, _ := b.db.GetSetting("required_channel")
		if currentChannel == "" {
			currentChannel = "غیرفعال"
		}
		reply := tgbotapi.NewMessage(chatID, Tr(getLang(msg.From), "channel_gate_settings", currentChannel))
		reply.ParseMode = tgbotapi.ModeMarkdown
		reply.ReplyMarkup = BackKeyboard()
		b.api.Send(reply)
		b.session.SetState(chatID, StateAdminAwaitingChannel)

	case BtnAdminQRToggle:
		if !isOwner {
			b.sendSimpleMessage(chatID, Tr(getLang(msg.From), "staff_owner_only"))
			return
		}
		currentQR, _ := b.db.GetSetting("qr_enabled")
		if currentQR == "off" {
			_ = b.db.SetSetting("qr_enabled", "on")
			b.sendSimpleMessage(chatID, Tr(getLang(msg.From), "qr_toggled_on"))
		} else {
			_ = b.db.SetSetting("qr_enabled", "off")
			b.sendSimpleMessage(chatID, Tr(getLang(msg.From), "qr_toggled_off"))
		}

	case BtnAdminReminderToggle:
		if !isOwner {
			b.sendSimpleMessage(chatID, Tr(getLang(msg.From), "staff_owner_only"))
			return
		}
		currentRem, _ := b.db.GetSetting("reminders_enabled")
		if currentRem == "off" {
			_ = b.db.SetSetting("reminders_enabled", "on")
			b.sendSimpleMessage(chatID, Tr(getLang(msg.From), "reminders_toggled_on"))
		} else {
			_ = b.db.SetSetting("reminders_enabled", "off")
			b.sendSimpleMessage(chatID, Tr(getLang(msg.From), "reminders_toggled_off"))
		}

	case BtnAdminPlansSettings, "📦 مدیریت پلانها", "📦 مدیریت پلان ها":
		if !isAdmin {
			b.sendSimpleMessage(chatID, "⚠️ شما دسترسی به این بخش را ندارید.")
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		apiResp, err := b.client.GetResellerSubscribeList(ctx, 1, 100)
		if err != nil || apiResp == nil {
			log.Printf("[BtnAdminPlansSettings] GetResellerSubscribeList error for %d: %v", chatID, err)
			b.sendSimpleMessage(chatID, fmt.Sprintf("❌ خطای ارتباط با سرور backend:\n`%v`", err))
			return
		}
		reply := tgbotapi.NewMessage(chatID, MsgAdminPlansList)
		reply.ParseMode = tgbotapi.ModeMarkdown
		reply.ReplyMarkup = AdminPlansInlineKeyboard(apiResp.List)
		b.api.Send(reply)

	case BtnAdminWelcomeSettings, "📝 مدیریت پیام خوشآمد", "📝 مدیریت پیام خوش آمد":
		if !isAdmin {
			b.sendSimpleMessage(chatID, "⚠️ شما دسترسی به این بخش را ندارید.")
			return
		}
		reply := tgbotapi.NewMessage(chatID, MsgAdminWelcomeSettingsMenu)
		reply.ParseMode = tgbotapi.ModeMarkdown
		reply.ReplyMarkup = AdminWelcomeSettingsKeyboard()
		b.api.Send(reply)

	case BtnAdminEditWelcomeText, "✏️ ویرایش متن خوشآمد", "✏️ ویرایش متن خوش آمد":
		if !isAdmin {
			b.sendSimpleMessage(chatID, "⚠️ شما دسترسی به این بخش را ندارید.")
			return
		}
		reply := tgbotapi.NewMessage(chatID, MsgAdminAwaitingWelcomeText)
		reply.ReplyMarkup = BackKeyboard()
		b.api.Send(reply)
		b.session.SetState(chatID, StateAdminAwaitingWelcomeText)

	case BtnAdminChangeWelcomeImg, "🖼️ تغییر عکس خوشآمد", "🖼️ تغییر عکس خوش آمد":
		if !isAdmin {
			b.sendSimpleMessage(chatID, "⚠️ شما دسترسی به این بخش را ندارید.")
			return
		}
		reply := tgbotapi.NewMessage(chatID, "🖼️ لطفاً تصویر جدید پیام خوش‌آمدگویی را ارسال نمایید:")
		reply.ReplyMarkup = BackKeyboard()
		b.api.Send(reply)
		b.session.SetState(chatID, StateAdminAwaitingWelcomeImage)

	case BtnAdminDelWelcomeImg, "❌ حذف عکس خوشآمد", "❌ حذف عکس خوش آمد":
		if !isAdmin {
			b.sendSimpleMessage(chatID, "⚠️ شما دسترسی به این بخش را ندارید.")
			return
		}
		_ = b.db.SetSetting("welcome_image", "")
		reply := tgbotapi.NewMessage(chatID, "✅ تصویر پیام خوش‌آمدگویی با موفقیت حذف شد.")
		reply.ReplyMarkup = AdminWelcomeSettingsKeyboard()
		b.api.Send(reply)

	case BtnAdminSupportSettings:
		if !isAdmin {
			b.sendSimpleMessage(chatID, "⚠️ شما دسترسی به این بخش را ندارید.")
			return
		}
		reply := tgbotapi.NewMessage(chatID, MsgAdminSupportSettingsMenu)
		reply.ParseMode = tgbotapi.ModeMarkdown
		reply.ReplyMarkup = AdminSupportSettingsKeyboard()
		b.api.Send(reply)

	case BtnAdminEditSupportText:
		if !isAdmin {
			b.sendSimpleMessage(chatID, "⚠️ شما دسترسی به این بخش را ندارید.")
			return
		}
		reply := tgbotapi.NewMessage(chatID, MsgAdminAwaitingSupportText)
		reply.ReplyMarkup = BackKeyboard()
		b.api.Send(reply)
		b.session.SetState(chatID, StateAdminAwaitingSupportText)

	case BtnAdminChangeSupportImg:
		if !isAdmin {
			b.sendSimpleMessage(chatID, "⚠️ شما دسترسی به این بخش را ندارید.")
			return
		}
		reply := tgbotapi.NewMessage(chatID, MsgAdminAwaitingSupportImg)
		reply.ReplyMarkup = BackKeyboard()
		b.api.Send(reply)
		b.session.SetState(chatID, StateAdminAwaitingSupportImage)

	case BtnAdminDelSupportImg:
		if !isAdmin {
			b.sendSimpleMessage(chatID, "⚠️ شما دسترسی به این بخش را ندارید.")
			return
		}
		_ = b.db.SetSetting("support_image", "")
		reply := tgbotapi.NewMessage(chatID, MsgAdminSupportImgDeleted)
		reply.ReplyMarkup = AdminSupportSettingsKeyboard()
		b.api.Send(reply)

	case BtnAdminTagSettings, "🏷️ مدیریت نام دسته ها":
		if !isAdmin {
			b.sendSimpleMessage(chatID, "⚠️ شما دسترسی به این بخش را ندارید.")
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		apiResp, err := b.client.GetResellerSubscribeList(ctx, 1, 100)
		if err != nil || apiResp == nil {
			log.Printf("[BtnAdminTagSettings] GetResellerSubscribeList error for %d: %v", chatID, err)
			b.sendSimpleMessage(chatID, fmt.Sprintf("❌ خطای ارتباط با سرور backend:\n`%v`", err))
			return
		}
		tagMap := make(map[string]bool)
		var tagItems []TagItem
		for _, plan := range apiResp.List {
			tags := plan.NodeTags
			if len(tags) == 0 {
				tags = []string{"عمومی"}
			}
			for _, tag := range tags {
				tag = strings.TrimSpace(tag)
				if tag != "" && !tagMap[tag] {
					tagMap[tag] = true
					display, _ := b.db.GetTagMapping(tag)
					if display == "" {
						display = tag
					}
					tagItems = append(tagItems, TagItem{Original: tag, Display: display})
				}
			}
		}
		reply := tgbotapi.NewMessage(chatID, "🏷️ *مدیریت نام دسته‌ها*\n\nبرای تغییر نام نمایشی هر دسته، روی آن کلیک کنید:")
		reply.ParseMode = tgbotapi.ModeMarkdown
		reply.ReplyMarkup = AdminTagsInlineKeyboard(tagItems)
		b.api.Send(reply)

	default:
		reply := tgbotapi.NewMessage(chatID, "لطفاً یک گزینه از منوی زیر را انتخاب کنید:")
		reply.ReplyMarkup = MainMenuKeyboard(isAdmin)
		b.api.Send(reply)
	}
}

func (b *Bot) checkChannelGate(chatID int64) bool {
	if b.isAdmin(chatID) {
		return true
	}

	requiredChannel, err := b.db.GetSetting("required_channel")
	if err != nil || strings.TrimSpace(requiredChannel) == "" {
		return true
	}

	requiredChannel = strings.TrimSpace(requiredChannel)
	chatMemberConfig := tgbotapi.GetChatMemberConfig{
		ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
			SuperGroupUsername: requiredChannel,
			UserID:             chatID,
		},
	}

	member, err := b.api.GetChatMember(chatMemberConfig)
	if err != nil {
		log.Printf("Telegram API warning during channel membership check for %d in %s: %v", chatID, requiredChannel, err)
		return true
	}

	switch member.Status {
	case "creator", "administrator", "member":
		return true
	default:
		u, _ := b.db.GetUser(chatID)
		var userObj *tgbotapi.User
		if u != nil {
			userObj = &tgbotapi.User{ID: chatID}
		}
		msgText := Tr(getLang(userObj), "channel_gate_required", requiredChannel)
		reply := tgbotapi.NewMessage(chatID, msgText)
		reply.ParseMode = tgbotapi.ModeMarkdown
		reply.ReplyMarkup = ChannelGateInlineKeyboard(requiredChannel)
		b.api.Send(reply)
		return false
	}
}

func (b *Bot) handleStateMessage(msg *tgbotapi.Message, u *db.User, sess *Session, isAdmin bool) {
	chatID := msg.Chat.ID

	// Check if user decided to go back
	if msg.Text == BtnBack {
		previousState := sess.State
		b.session.Clear(chatID)
		welcomeMsg := tgbotapi.NewMessage(chatID, "بازگشت:")
		if isAdmin {
			if previousState == StateAdminAwaitingWelcomeText || previousState == StateAdminAwaitingWelcomeImage {
				welcomeMsg.ReplyMarkup = AdminWelcomeSettingsKeyboard()
			} else if previousState == StateAdminAwaitingSupportText || previousState == StateAdminAwaitingSupportImage {
				welcomeMsg.ReplyMarkup = AdminSupportSettingsKeyboard()
			} else {
				welcomeMsg.ReplyMarkup = AdminMenuKeyboard(b.isOwner(chatID))
			}
		} else {
			welcomeMsg.ReplyMarkup = MainMenuKeyboard(isAdmin)
		}
		b.api.Send(welcomeMsg)
		return
	}

	switch sess.State {
	// Client States
	case StateAwaitingSubCustomName:
		if !b.session.TryLock(chatID) {
			return
		}
		defer b.session.Unlock(chatID)

		customName := strings.TrimSpace(msg.Text)
		if customName == "-" || customName == "" {
			customName = fmt.Sprintf("%s %s", sess.PurchasingPlanName, time.Now().Format("2006-01-02"))
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		expiredAt := calculateExpiryMs(sess.PurchasingUnitTime)
		subResp, err := b.client.CreateSubscription(ctx, &backend.SubscribeRequest{
			UserID:            u.UserID,
			SubscribeID:       int(sess.PurchasingPlanID),
			CustomName:        customName,
			ExpiredAt:         expiredAt,
			ChargeFromBalance: true,
		})
		if err != nil {
			log.Printf("CreateSubscription backend error: %v", err)
			purchasingPlanID := sess.PurchasingPlanID
			b.session.Clear(chatID)

			if backend.IsErrorCode(err, 20005) {
				rate := b.rateMgr.GetRate(ctx, b.client)
				userRegisterResp, _ := b.client.RegisterUser(ctx, &backend.UserRegisterRequest{TelegramID: chatID})
				var userBal int64
				if userRegisterResp != nil {
					userBal = userRegisterResp.Balance
				}
				apiResp, _ := b.client.GetResellerSubscribeList(ctx, 1, 100)
				var planPrice int64
				if apiResp != nil {
					for _, p := range apiResp.List {
						if p.ID == purchasingPlanID {
							planPrice = p.UnitPrice
							break
						}
					}
				}
				text := fmt.Sprintf(MsgInsufficientBalance, FormatUserBalance(userBal, rate), FormatMoney(planPrice))
				b.sendSimpleMessage(chatID, text)
				return
			}

			b.sendSimpleMessage(chatID, Tr(getLang(msg.From), "purchase_failed_try_again"))
			return
		}

		b.session.Clear(chatID)

		successMsg := tgbotapi.NewMessage(chatID, MsgPurchaseSuccess)
		successMsg.ParseMode = tgbotapi.ModeMarkdown
		successMsg.ReplyMarkup = MainMenuKeyboard(isAdmin)
		b.api.Send(successMsg)

		var targetItem *backend.SubscriptionItem
		var fetchErr error

		for attempt := 1; attempt <= 2; attempt++ {
			if attempt > 1 {
				time.Sleep(500 * time.Millisecond)
			}
			subs, err := b.client.GetUserSubscriptions(ctx, u.UserID, 1, 10)
			if err == nil && subs != nil && len(subs.List) > 0 {
				for i, item := range subs.List {
					if item.ID == subResp.UserSubscribeID {
						targetItem = &subs.List[i]
						break
					}
				}
				if targetItem == nil {
					targetItem = &subs.List[0]
				}
				fetchErr = nil
				break
			} else {
				fetchErr = err
			}
		}

		if fetchErr != nil || targetItem == nil {
			log.Printf("Failed to fetch subscription details after purchase for user %d (subID: %d): %v", u.UserID, subResp.UserSubscribeID, fetchErr)
			b.sendSimpleMessage(chatID, Tr(getLang(msg.From), "purchase_success_fallback"))
			return
		}

		nodes, err := b.client.GetDownloadNodes(ctx, targetItem.ID)
		if err != nil {
			log.Printf("Failed to fetch download nodes for post-purchase sub %d: %v", targetItem.ID, err)
		}
		ps := analyzeDownloadNodes(nodes)

		subLink := b.getSubscribeLink(targetItem.Token)
		text := BuildSubscriptionDetailText(b.db, targetItem, subLink, getLang(msg.From))

		b.sendSubscriptionMessage(chatID, text, subLink, SubscriptionDetailKeyboard(targetItem.ID, ps.HasOpenVPN, ps.HasWireGuard, 1))

	case StateAwaitingAmount:
		amountStr := strings.TrimSpace(msg.Text)
		amount, err := strconv.ParseInt(amountStr, 10, 64)
		if err != nil || amount <= 0 {
			b.sendSimpleMessage(chatID, MsgInvalidAmount)
			return
		}

		b.session.SetPendingAmount(chatID, amount)
		b.session.SetState(chatID, StateAwaitingReceipt)

		text := fmt.Sprintf(MsgSendReceipt, FormatMoney(amount))
		reply := tgbotapi.NewMessage(chatID, text)
		reply.ParseMode = tgbotapi.ModeMarkdown
		reply.ReplyMarkup = BackKeyboard()
		b.api.Send(reply)

	case StateAwaitingReceipt:
		if len(msg.Photo) == 0 {
			b.sendSimpleMessage(chatID, MsgInvalidReceipt)
			return
		}

		photo := msg.Photo[len(msg.Photo)-1]

		base64Receipt, err := b.processReceiptPhoto(photo.FileID)
		if err != nil {
			log.Printf("Failed to process receipt photo: %v", err)
			b.sendSimpleMessage(chatID, MsgGeneralError)
			b.session.Clear(chatID)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		order, err := b.client.CreateRecharge(ctx, &backend.CreateRechargeRequest{
			Tier:           "customer_topup",
			CustomerUserID: u.UserID,
			Amount:         sess.PendingAmount,
			Currency:       "IRT",
			ReceiptType:    "base64",
			ReceiptData:    base64Receipt,
			Source:         "bot",
		})
		if err != nil {
			if backend.IsErrorCode(err, 62002) {
				b.sendSimpleMessage(chatID, Tr(getLang(msg.From), "pending_limit_exceeded"))
				b.session.Clear(chatID)
				return
			}
			log.Printf("Failed to create recharge order on backend: %v", err)
			b.sendSimpleMessage(chatID, MsgGeneralError)
			b.session.Clear(chatID)
			return
		}

		b.session.Clear(chatID)

		var orderNoLine string
		if order != nil && strings.TrimSpace(order.OrderNo) != "" {
			orderNoLine = fmt.Sprintf("- شماره سفارش: %s\n", escapeMarkdown(order.OrderNo))
		}

		userText := fmt.Sprintf("✅ رسید شما دریافت شد!\n- مبلغ: %s تومان\n%sوضعیت: در انتظار بررسی توسط پشتیبانی\n\nپس از تأیید، موجودی کیف پول شما به‌صورت خودکار افزایش می‌یابد و به شما اطلاع داده می‌شود.",
			escapeMarkdown(FormatMoney(sess.PendingAmount)),
			orderNoLine,
		)

		successMsg := tgbotapi.NewMessage(chatID, userText)
		successMsg.ParseMode = tgbotapi.ModeMarkdown
		successMsg.ReplyMarkup = MainMenuKeyboard(isAdmin)
		b.api.Send(successMsg)

		username := msg.From.UserName
		if username == "" {
			username = "NoUsername"
		}
		displayName := msg.From.FirstName
		if msg.From.LastName != "" {
			displayName += " " + msg.From.LastName
		}

		adminText := fmt.Sprintf(MsgAdminNewRequest, escapeMarkdown(username), escapeMarkdown(displayName), chatID, u.UserID, FormatMoney(sess.PendingAmount))
		if order != nil && order.OrderNo != "" {
			adminText += fmt.Sprintf("\n• شماره سفارش: `%s`", order.OrderNo)
		}

		orderID := order.ID
		allAdminIDs := b.getAllAdminIDs()
		for _, adminID := range allAdminIDs {
			adminPhoto := tgbotapi.NewPhoto(adminID, tgbotapi.FileID(photo.FileID))
			adminPhoto.Caption = adminText
			adminPhoto.ParseMode = tgbotapi.ModeMarkdown
			adminPhoto.ReplyMarkup = AdminApprovalKeyboard(orderID)
			if _, err := b.api.Send(adminPhoto); err != nil {
				log.Printf("Failed to send admin notification to %d: %v", adminID, err)
			}
		}

	case StateAdminAwaitingRejectReason:
		if !isAdmin {
			b.session.Clear(chatID)
			return
		}
		reason := strings.TrimSpace(msg.Text)
		if reason == "" {
			b.sendSimpleMessage(chatID, Tr(getLang(msg.From), "reject_reason_required"))
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		err := b.client.ReviewRecharge(ctx, &backend.RechargeReviewRequest{
			ID:     sess.RejectOrderID,
			Action: "reject",
			Reason: reason,
		})

		if err != nil {
			if backend.IsErrorCode(err, 62001) {
				b.sendSimpleMessage(chatID, Tr(getLang(msg.From), "already_reviewed"))
				b.session.Clear(chatID)
				return
			}
			log.Printf("Failed to reject recharge order %d: %v", sess.RejectOrderID, err)
			b.sendSimpleMessage(chatID, MsgGeneralError)
			b.session.Clear(chatID)
			return
		}

		adminUser := msg.From.FirstName
		if msg.From.LastName != "" {
			adminUser += " " + msg.From.LastName
		}
		if msg.From.UserName != "" {
			adminUser += " (@" + msg.From.UserName + ")"
		}

		b.sendSimpleMessage(chatID, fmt.Sprintf("❌ درخواست شارژ رد شد.\nعلت: %s\nتوسط: %s", reason, adminUser))

		// Send customer DM immediately & mark notified in SQLite
		listResp, err := b.client.GetRechargeList(ctx, "customer_topup", "", 1, 100)
		if err == nil && listResp != nil {
			for _, order := range listResp.List {
				if order.ID == sess.RejectOrderID {
					user, err := b.db.GetUserByBackendID(order.CustomerUserID)
					if err == nil && user != nil {
						msgText := Tr(getLang(&tgbotapi.User{ID: user.TelegramID}), "recharge_rejected_user", FormatMoney(order.OriginalAmount), reason)
						b.sendSimpleMessage(user.TelegramID, msgText)
						_ = b.db.SaveRechargeNotified(order.ID, "rejected")
					}
					break
				}
			}
		}

		b.session.Clear(chatID)

	// Admin Settings States
	case StateAdminAwaitingCardNumber:
		if !b.isOwner(chatID) {
			b.session.Clear(chatID)
			return
		}

		var digitsOnly strings.Builder
		for _, r := range msg.Text {
			if unicode.IsDigit(r) {
				digitsOnly.WriteRune(r)
			}
		}
		num := digitsOnly.String()
		if len(num) < 13 || len(num) > 19 {
			b.sendSimpleMessage(chatID, Tr(getLang(msg.From), "card_number_invalid"))
			return
		}

		b.session.SetTempCardNumber(chatID, num)
		b.session.SetState(chatID, StateAdminAwaitingCardOwner)
		b.sendSimpleMessage(chatID, MsgAdminAwaitingCardOwner)

	case StateAdminAwaitingCardOwner:
		if !b.isOwner(chatID) {
			b.session.Clear(chatID)
			return
		}
		owner := strings.TrimSpace(msg.Text)
		if owner == "" {
			b.sendSimpleMessage(chatID, "⚠️ نام صاحب کارت نمی‌تواند خالی باشد. مجدداً وارد کنید:")
			return
		}

		b.session.SetTempCardOwner(chatID, owner)
		b.session.SetState(chatID, StateAdminAwaitingCardBank)
		b.sendSimpleMessage(chatID, MsgAdminAwaitingCardBank)

	case StateAdminAwaitingCardBank:
		if !b.isOwner(chatID) {
			b.session.Clear(chatID)
			return
		}
		bank := strings.TrimSpace(msg.Text)
		b.session.SetTempBankName(chatID, bank)
		b.session.SetState(chatID, StateAdminAwaitingCardInstructions)
		b.sendSimpleMessage(chatID, MsgAdminAwaitingCardInstructions)

	case StateAdminAwaitingCardInstructions:
		if !b.isOwner(chatID) {
			b.session.Clear(chatID)
			return
		}
		inst := strings.TrimSpace(msg.Text)
		if inst == "-" {
			inst = ""
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		err := b.client.UpsertPaymentCard(ctx, &backend.PaymentCard{
			CardNumber:   sess.TempCardNumber,
			CardOwner:    sess.TempCardOwner,
			BankName:     sess.TempBankName,
			Enabled:      true,
			Instructions: inst,
		})
		if err != nil {
			log.Printf("UpsertPaymentCard backend error: %v", err)
			var apiErr *backend.APIError
			if errors.As(err, &apiErr) && apiErr.Msg != "" {
				b.sendSimpleMessage(chatID, fmt.Sprintf("⚠️ خطا: %s", apiErr.Msg))
			} else {
				b.sendSimpleMessage(chatID, MsgGeneralError)
			}
			b.session.SetState(chatID, StateAdminAwaitingCardNumber)
			b.sendSimpleMessage(chatID, MsgAdminAwaitingCardNumber)
			return
		}

		reply := tgbotapi.NewMessage(chatID, MsgAdminCardUpdated)
		reply.ReplyMarkup = AdminMenuKeyboard(true)
		b.api.Send(reply)
		b.session.Clear(chatID)

	case StateAdminAwaitingStaffAdd:
		if !b.isOwner(chatID) {
			b.session.Clear(chatID)
			return
		}

		var targetID int64
		var targetName string

		if msg.ForwardFrom != nil {
			targetID = msg.ForwardFrom.ID
			targetName = msg.ForwardFrom.FirstName
			if msg.ForwardFrom.LastName != "" {
				targetName += " " + msg.ForwardFrom.LastName
			}
			if msg.ForwardFrom.UserName != "" {
				targetName += " (@" + msg.ForwardFrom.UserName + ")"
			}
		} else {
			txt := strings.TrimSpace(msg.Text)
			parsedID, err := strconv.ParseInt(txt, 10, 64)
			if err != nil || parsedID <= 0 {
				b.sendSimpleMessage(chatID, Tr(getLang(msg.From), "staff_invalid_id"))
				return
			}
			targetID = parsedID
			targetName = fmt.Sprintf("Staff_%d", targetID)
		}

		if err := b.db.AddStaff(targetID, targetName); err != nil {
			log.Printf("Failed to add staff: %v", err)
			b.sendSimpleMessage(chatID, MsgGeneralError)
		} else {
			b.sendSimpleMessage(chatID, Tr(getLang(msg.From), "staff_added_success"))
		}
		b.session.Clear(chatID)

	case StateAdminAwaitingChannel:
		if !b.isOwner(chatID) {
			b.session.Clear(chatID)
			return
		}

		input := strings.TrimSpace(msg.Text)
		if strings.EqualFold(input, "off") || input == "0" || input == "none" {
			_ = b.db.SetSetting("required_channel", "")
			b.sendSimpleMessage(chatID, Tr(getLang(msg.From), "channel_cleared_success"))
			b.session.Clear(chatID)
			return
		}

		if !strings.HasPrefix(input, "@") {
			input = "@" + input
		}

		chatMemberConfig := tgbotapi.GetChatMemberConfig{
			ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
				SuperGroupUsername: input,
				UserID:             b.api.Self.ID,
			},
		}
		member, err := b.api.GetChatMember(chatMemberConfig)
		if err != nil || (member.Status != "administrator" && member.Status != "creator") {
			log.Printf("Warning: bot is not admin in channel %s: %v", input, err)
			b.sendSimpleMessage(chatID, Tr(getLang(msg.From), "channel_warning_not_admin", input))
		}

		_ = b.db.SetSetting("required_channel", input)
		b.sendSimpleMessage(chatID, Tr(getLang(msg.From), "channel_updated_success"))
		b.session.Clear(chatID)

	case StateAdminAwaitingPlanSubscribeID:
		if !isAdmin {
			b.session.Clear(chatID)
			return
		}
		subIDStr := strings.TrimSpace(msg.Text)
		subID, err := strconv.Atoi(subIDStr)
		if err != nil || subID <= 0 {
			b.sendSimpleMessage(chatID, MsgAdminInvalidSubscribeID)
			return
		}

		b.session.SetTempPlanSubscribeID(chatID, subID)
		b.session.SetState(chatID, StateAdminAwaitingPlanName)
		b.sendSimpleMessage(chatID, MsgAdminAwaitingPlanName)

	case StateAdminAwaitingPlanName:
		if !isAdmin {
			b.session.Clear(chatID)
			return
		}
		name := strings.TrimSpace(msg.Text)
		if name == "" {
			b.sendSimpleMessage(chatID, "⚠️ نام پلان نمی‌تواند خالی باشد. مجدداً وارد کنید:")
			return
		}

		b.session.SetTempPlanName(chatID, name)
		b.session.SetState(chatID, StateAdminAwaitingPlanPrice)
		b.sendSimpleMessage(chatID, MsgAdminAwaitingPlanPrice)

	case StateAdminAwaitingPlanPrice:
		if !isAdmin {
			b.session.Clear(chatID)
			return
		}
		priceStr := strings.TrimSpace(msg.Text)
		price, err := strconv.ParseInt(priceStr, 10, 64)
		if err != nil || price <= 0 {
			b.sendSimpleMessage(chatID, MsgAdminInvalidPrice)
			return
		}

		b.session.SetTempPlanPrice(chatID, price)
		b.session.SetState(chatID, StateAdminAwaitingPlanDescription)
		b.sendSimpleMessage(chatID, MsgAdminAwaitingPlanDescription)

	case StateAdminAwaitingPlanDescription:
		if !isAdmin {
			b.session.Clear(chatID)
			return
		}
		desc := strings.TrimSpace(msg.Text)
		if desc == "" {
			b.sendSimpleMessage(chatID, "⚠️ توضیحات نمی‌تواند خالی باشد. مجدداً وارد کنید:")
			return
		}

		req := &backend.CreateResellerSubscribeRequest{
			Name:                   sess.TempPlanName,
			UnitPrice:              sess.TempPlanPrice,
			Description:            desc,
			ResellerSubscriptionID: int64(sess.TempPlanSubscribeID),
			UnitTime:               "month",
			Traffic:                10 * 1024 * 1024 * 1024, // 10GB default
			DeviceLimit:            1,
			Show:                   true,
			Sell:                   true,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		err := b.client.CreateResellerSubscribe(ctx, req)
		if err != nil {
			log.Printf("Failed to save new plan to backend: %v", err)
			b.sendSimpleMessage(chatID, MsgGeneralError)
		} else {
			b.sendSimpleMessage(chatID, MsgAdminPlanAdded)
		}

		b.session.Clear(chatID)

		// List updated plans
		apiResp, _ := b.client.GetResellerSubscribeList(ctx, 1, 100)
		reply := tgbotapi.NewMessage(chatID, MsgAdminPlansList)
		reply.ParseMode = tgbotapi.ModeMarkdown
		if apiResp != nil {
			reply.ReplyMarkup = AdminPlansInlineKeyboard(apiResp.List)
		}
		b.api.Send(reply)

	case StateAdminAwaitingWelcomeText:
		if !isAdmin {
			b.session.Clear(chatID)
			return
		}
		txt := strings.TrimSpace(msg.Text)
		if txt == "" {
			b.sendSimpleMessage(chatID, "⚠️ متن پیام خوش‌آمدگویی نمی‌تواند خالی باشد. مجدداً وارد کنید:")
			return
		}

		_ = b.db.SetSetting("welcome_text", txt)

		b.session.Clear(chatID)
		reply := tgbotapi.NewMessage(chatID, MsgAdminWelcomeTextUpdated)
		reply.ReplyMarkup = AdminWelcomeSettingsKeyboard()
		b.api.Send(reply)

	case StateAdminAwaitingWelcomeImage:
		if !isAdmin {
			b.session.Clear(chatID)
			return
		}
		if len(msg.Photo) == 0 {
			b.sendSimpleMessage(chatID, "⚠️ خطا! لطفاً تصویر جدید پیام خوش‌آمدگویی را ارسال نمایید:")
			return
		}

		photo := msg.Photo[len(msg.Photo)-1]
		_ = b.db.SetSetting("welcome_image", photo.FileID)

		b.session.Clear(chatID)
		reply := tgbotapi.NewMessage(chatID, MsgAdminWelcomeImgUpdated)
		reply.ReplyMarkup = AdminWelcomeSettingsKeyboard()
		b.api.Send(reply)

	case StateAdminAwaitingSupportText:
		if !isAdmin {
			b.session.Clear(chatID)
			return
		}
		txt := strings.TrimSpace(msg.Text)
		if txt == "" {
			b.sendSimpleMessage(chatID, "⚠️ متن پیام پشتیبانی نمی‌تواند خالی باشد. مجدداً وارد کنید:")
			return
		}

		_ = b.db.SetSetting("support_text", txt)

		b.session.Clear(chatID)
		reply := tgbotapi.NewMessage(chatID, MsgAdminSupportTextUpdated)
		reply.ReplyMarkup = AdminSupportSettingsKeyboard()
		b.api.Send(reply)

	case StateAdminAwaitingSupportImage:
		if !isAdmin {
			b.session.Clear(chatID)
			return
		}
		if len(msg.Photo) == 0 {
			b.sendSimpleMessage(chatID, "⚠️ خطا! لطفاً تصویر جدید برای پیام پشتیبانی را ارسال نمایید:")
			return
		}

		photo := msg.Photo[len(msg.Photo)-1]
		_ = b.db.SetSetting("support_image", photo.FileID)

		b.session.Clear(chatID)
		reply := tgbotapi.NewMessage(chatID, MsgAdminSupportImgUpdated)
		reply.ReplyMarkup = AdminSupportSettingsKeyboard()
		b.api.Send(reply)

	case StateAdminAwaitingTagDisplayName:
		if !isAdmin {
			b.session.Clear(chatID)
			return
		}
		newName := strings.TrimSpace(msg.Text)
		if newName == "" {
			b.sendSimpleMessage(chatID, "⚠️ نام نمایشی دسته نمی‌تواند خالی باشد. مجدداً وارد کنید:")
			return
		}

		err := b.db.SetTagMapping(sess.TempPlanName, newName) // TempPlanName stores original tag name
		if err != nil {
			log.Printf("Failed to save tag mapping: %v", err)
			b.sendSimpleMessage(chatID, MsgGeneralError)
		} else {
			b.sendSimpleMessage(chatID, "✅ نام نمایشی با موفقیت ذخیره شد.")
		}

		b.session.Clear(chatID)
		reply := tgbotapi.NewMessage(chatID, MsgAdminPanelWelcome)
		reply.ReplyMarkup = AdminMenuKeyboard(b.isOwner(chatID))
		b.api.Send(reply)
	}
}

func (b *Bot) sendSimpleMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	b.api.Send(msg)
}

func (b *Bot) renderSubscriptionsListPage(chatID int64, messageID int, page int, from *tgbotapi.User) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	const pageSize = 8
	if page < 1 {
		page = 1
	}

	u, err := b.db.GetUser(chatID)
	if err != nil || u == nil {
		b.sendSimpleMessage(chatID, MsgGeneralError)
		return
	}

	resp, err := b.client.GetUserSubscriptions(ctx, u.UserID, page, pageSize)
	if err != nil {
		log.Printf("Failed to get user subscriptions for %d (page %d): %v", u.UserID, page, err)
		b.sendSimpleMessage(chatID, MsgGeneralError)
		return
	}

	lang := getLang(from)

	if resp == nil || resp.Total == 0 || len(resp.List) == 0 {
		text := Tr(lang, "no_subscriptions_text")
		kb := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(Tr(lang, "buy_service_btn"), "cb_buy_service"),
			),
		)

		if messageID > 0 {
			editMsg := tgbotapi.NewEditMessageText(chatID, messageID, text)
			editMsg.ParseMode = tgbotapi.ModeMarkdown
			editMsg.ReplyMarkup = &kb
			b.api.Send(editMsg)
		} else {
			msg := tgbotapi.NewMessage(chatID, text)
			msg.ParseMode = tgbotapi.ModeMarkdown
			msg.ReplyMarkup = kb
			b.api.Send(msg)
		}
		return
	}

	totalPages := (resp.Total + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}

	text := fmt.Sprintf(Tr(lang, "subscriptions_list_title"), resp.Total)
	kb := SubscriptionsListKeyboard(resp.List, page, totalPages, lang)

	if messageID > 0 {
		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, text)
		editMsg.ParseMode = tgbotapi.ModeMarkdown
		editMsg.ReplyMarkup = &kb
		b.api.Send(editMsg)
	} else {
		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = tgbotapi.ModeMarkdown
		msg.ReplyMarkup = kb
		b.api.Send(msg)
	}
}

func (b *Bot) handleBuyService(chatID int64) {
	if !b.checkChannelGate(chatID) {
		return
	}
	b.renderTagsMenu(chatID, 0, nil)
}

func (b *Bot) renderTagsMenu(chatID int64, messageID int, from *tgbotapi.User) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	apiResp, err := b.client.GetResellerSubscribeList(ctx, 1, 100)
	if err != nil {
		log.Printf("Failed to get plans from backend: %v", err)
		b.sendSimpleMessage(chatID, MsgGeneralError)
		return
	}

	tagMap := make(map[string]bool)
	var plans []backend.ResellerSubscribePlan
	for _, p := range apiResp.List {
		if p.Show {
			plans = append(plans, p)
			for _, t := range p.NodeTags {
				tag := strings.TrimSpace(t)
				if tag != "" {
					tagMap[tag] = true
				}
			}
		}
	}

	if len(plans) == 0 {
		b.sendSimpleMessage(chatID, "⚠️ در حال حاضر هیچ پلانی تعریف نشده است. لطفاً بعداً تلاش کنید.")
		return
	}

	var uniqueTags []string
	for tag := range tagMap {
		uniqueTags = append(uniqueTags, tag)
	}

	var tagItems []TagItem
	for _, tag := range uniqueTags {
		disp, err := b.db.GetTagMapping(tag)
		if err != nil || disp == "" {
			disp = tag
		}
		tagItems = append(tagItems, TagItem{
			Original: tag,
			Display:  disp,
		})
	}

	if len(tagItems) > 1 {
		text := "🏷️ لطفاً دسته مورد نظر خود را انتخاب کنید:"
		kb := TagsInlineKeyboard(tagItems)
		if messageID > 0 {
			editMsg := tgbotapi.NewEditMessageText(chatID, messageID, text)
			editMsg.ParseMode = tgbotapi.ModeMarkdown
			editMsg.ReplyMarkup = &kb
			b.api.Send(editMsg)
		} else {
			msg := tgbotapi.NewMessage(chatID, text)
			msg.ParseMode = tgbotapi.ModeMarkdown
			msg.ReplyMarkup = kb
			b.api.Send(msg)
		}
		return
	}

	text := MsgPlansList
	kb := PlansInlineKeyboard(plans, "")
	if messageID > 0 {
		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, text)
		editMsg.ParseMode = tgbotapi.ModeMarkdown
		editMsg.ReplyMarkup = &kb
		b.api.Send(editMsg)
	} else {
		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = tgbotapi.ModeMarkdown
		msg.ReplyMarkup = kb
		b.api.Send(msg)
	}
}
