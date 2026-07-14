package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"reseller-bot/pkg/backend"
	"reseller-bot/pkg/db"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) handleMessage(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	log.Printf("Received message from %d: %s", chatID, msg.Text)

	// Check if user is admin
	isAdmin := false
	for _, adminID := range b.cfg.AdminChatIDs {
		if chatID == adminID {
			isAdmin = true
			break
		}
	}

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
		welcomeText = strings.ReplaceAll(welcomeText, "{name}", name)

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

		// Refresh user profile/balance from backend
		resp, err := b.client.RegisterUser(ctx, &backend.UserRegisterRequest{
			TelegramID: chatID,
		})
		if err != nil {
			log.Printf("Backend info fetch error: %v", err)
			b.sendSimpleMessage(chatID, MsgGeneralError)
			return
		}

		text := fmt.Sprintf(MsgAccountInfo, chatID, resp.UserID, FormatMoney(resp.Balance))
		reply := tgbotapi.NewMessage(chatID, text)
		reply.ParseMode = tgbotapi.ModeMarkdown
		reply.ReplyMarkup = MainMenuKeyboard(isAdmin)
		b.api.Send(reply)

	case BtnBuyService:
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		apiResp, err := b.client.GetResellerSubscribeList(ctx, 1, 100)
		if err != nil {
			log.Printf("Failed to get plans from backend: %v", err)
			b.sendSimpleMessage(chatID, MsgGeneralError)
			return
		}

		// Filter plans where Show is true
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
			b.sendSimpleMessage(chatID, "⚠️ در حال حاضر هیچ پلانی تعریف نشده است. لطفا بعدا تلاش کنید.")
			return
		}

		var uniqueTags []string
		for tag := range tagMap {
			uniqueTags = append(uniqueTags, tag)
		}

		// Look up displaying names from local Sqlite DB
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

		// If multiple unique tags exist, present category menu first
		if len(tagItems) > 1 {
			reply := tgbotapi.NewMessage(chatID, "🏷️ لطفاً دسته مورد نظر خود را انتخاب کنید:")
			reply.ReplyMarkup = TagsInlineKeyboard(tagItems)
			b.api.Send(reply)
			return
		}

		// Default: Show all plans
		reply := tgbotapi.NewMessage(chatID, MsgPlansList)
		reply.ReplyMarkup = PlansInlineKeyboard(plans)
		b.api.Send(reply)

	case BtnMySubscriptions:
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		resp, err := b.client.GetUserSubscriptions(ctx, u.UserID, 1, 100)
		if err != nil {
			log.Printf("Backend subscriptions fetch error: %v", err)
			b.sendSimpleMessage(chatID, MsgGeneralError)
			return
		}

		if resp.Total == 0 || len(resp.List) == 0 {
			reply := tgbotapi.NewMessage(chatID, MsgNoSubscriptions)
			reply.ReplyMarkup = MainMenuKeyboard(isAdmin)
			b.api.Send(reply)
			return
		}

		b.sendSimpleMessage(chatID, MsgSubscriptionsList)

		for _, item := range resp.List {
			expStr := "بدون انقضا"
			if item.ExpireTime > 0 {
				expStr = time.Unix(item.ExpireTime, 0).Format("2006-01-02 15:04:05")
			}

			configsBlock := ""
			if len(item.Configs) > 0 {
				configsBlock = "```\n" + strings.Join(item.Configs, "\n\n") + "\n```"
			} else {
				configsBlock = "_کانفیگی در دسترس نیست_"
			}

			text := fmt.Sprintf(MsgSubDetail,
				item.Name,
				item.UUID,
				FormatTraffic(item.TotalTraffic),
				FormatTraffic(item.UsedTraffic),
				expStr,
			) + configsBlock

			reply := tgbotapi.NewMessage(chatID, text)
			reply.ParseMode = tgbotapi.ModeMarkdown
			reply.ReplyMarkup = MainMenuKeyboard(isAdmin)
			b.api.Send(reply)
		}

	case BtnTopUpBalance:
		// Fetch card details dynamically from SQLite settings table
		cardNumber, _ := b.db.GetSetting("card_number")
		cardOwner, _ := b.db.GetSetting("card_owner")

		if cardNumber == "" || cardOwner == "" {
			b.sendSimpleMessage(chatID, "⚠️ متاسفانه در حال حاضر امکان واریز کارت به کارت وجود ندارد (اطلاعات حساب مدیریت تعریف نشده است).")
			return
		}

		text := fmt.Sprintf(MsgTopUpCardInfo, cardNumber, cardOwner)
		reply := tgbotapi.NewMessage(chatID, text)
		reply.ParseMode = tgbotapi.ModeMarkdown
		reply.ReplyMarkup = BackKeyboard()
		b.api.Send(reply)
		b.session.SetState(chatID, StateAwaitingAmount)

	case BtnContactSupport:
		reply := tgbotapi.NewMessage(chatID, MsgContactSupportText)
		reply.ParseMode = tgbotapi.ModeMarkdown
		reply.ReplyMarkup = MainMenuKeyboard(isAdmin)
		b.api.Send(reply)

	case BtnAdminPanel, "/admin":
		if !isAdmin {
			b.sendSimpleMessage(chatID, "⚠️ شما دسترسی به این بخش را ندارید.")
			return
		}
		b.session.Clear(chatID)
		reply := tgbotapi.NewMessage(chatID, MsgAdminPanelWelcome)
		reply.ParseMode = tgbotapi.ModeMarkdown
		reply.ReplyMarkup = AdminMenuKeyboard()
		b.api.Send(reply)

	case BtnAdminCardSettings:
		if !isAdmin {
			return
		}
		reply := tgbotapi.NewMessage(chatID, MsgAdminAwaitingCardNumber)
		reply.ReplyMarkup = BackKeyboard()
		b.api.Send(reply)
		b.session.SetState(chatID, StateAdminAwaitingCardNumber)

	case BtnAdminPlansSettings:
		if !isAdmin {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		apiResp, err := b.client.GetResellerSubscribeList(ctx, 1, 100)
		if err != nil {
			log.Printf("Failed to get plans for admin: %v", err)
			b.sendSimpleMessage(chatID, MsgGeneralError)
			return
		}
		reply := tgbotapi.NewMessage(chatID, MsgAdminPlansList)
		reply.ParseMode = tgbotapi.ModeMarkdown
		reply.ReplyMarkup = AdminPlansInlineKeyboard(apiResp.List)
		b.api.Send(reply)

	case BtnAdminTagSettings:
		if !isAdmin {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		apiResp, err := b.client.GetResellerSubscribeList(ctx, 1, 100)
		if err != nil {
			log.Printf("Failed to get plans for admin tags: %v", err)
			b.sendSimpleMessage(chatID, MsgGeneralError)
			return
		}

		tagMap := make(map[string]bool)
		for _, p := range apiResp.List {
			if p.Show {
				for _, t := range p.NodeTags {
					tag := strings.TrimSpace(t)
					if tag != "" {
						tagMap[tag] = true
					}
				}
			}
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

		if len(tagItems) == 0 {
			b.sendSimpleMessage(chatID, "⚠️ هیچ دسته‌ای از پنل اصلی یافت نشد.")
			return
		}

		reply := tgbotapi.NewMessage(chatID, "🏷️ *مدیریت نام نمایشی دسته‌ها:* \n\nبرای تغییر نام نمایشی هر دسته روی آن کلیک کنید:")
		reply.ParseMode = tgbotapi.ModeMarkdown
		reply.ReplyMarkup = AdminTagsInlineKeyboard(tagItems)
		b.api.Send(reply)

	case BtnAdminWelcomeSettings:
		if !isAdmin {
			return
		}
		b.session.Clear(chatID)
		reply := tgbotapi.NewMessage(chatID, MsgAdminWelcomeSettingsMenu)
		reply.ParseMode = tgbotapi.ModeMarkdown
		reply.ReplyMarkup = AdminWelcomeSettingsKeyboard()
		b.api.Send(reply)

	case BtnAdminEditWelcomeText:
		if !isAdmin {
			return
		}
		reply := tgbotapi.NewMessage(chatID, MsgAdminAwaitingWelcomeText)
		reply.ParseMode = tgbotapi.ModeMarkdown
		reply.ReplyMarkup = BackKeyboard()
		b.api.Send(reply)
		b.session.SetState(chatID, StateAdminAwaitingWelcomeText)

	case BtnAdminChangeWelcomeImg:
		if !isAdmin {
			return
		}
		reply := tgbotapi.NewMessage(chatID, MsgAdminAwaitingWelcomeImg)
		reply.ParseMode = tgbotapi.ModeMarkdown
		reply.ReplyMarkup = BackKeyboard()
		b.api.Send(reply)
		b.session.SetState(chatID, StateAdminAwaitingWelcomeImage)

	case BtnAdminDelWelcomeImg:
		if !isAdmin {
			return
		}
		_ = b.db.SetSetting("welcome_image", "")
		reply := tgbotapi.NewMessage(chatID, MsgAdminWelcomeImgDeleted)
		reply.ReplyMarkup = AdminWelcomeSettingsKeyboard()
		b.api.Send(reply)

	default:
		reply := tgbotapi.NewMessage(chatID, "لطفاً یک گزینه از منوی زیر را انتخاب کنید:")
		reply.ReplyMarkup = MainMenuKeyboard(isAdmin)
		b.api.Send(reply)
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
			} else {
				welcomeMsg.ReplyMarkup = AdminMenuKeyboard()
			}
		} else {
			welcomeMsg.ReplyMarkup = MainMenuKeyboard(isAdmin)
		}
		b.api.Send(welcomeMsg)
		return
	}

	switch sess.State {
	// Client States
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

		req := &db.RechargeRequest{
			TelegramID:    chatID,
			UserID:        u.UserID,
			Amount:        sess.PendingAmount,
			Status:        "pending",
			ReceiptFileID: photo.FileID,
			CreatedAt:     time.Now().Unix(),
		}

		reqID, err := b.db.CreateRechargeRequest(req)
		if err != nil {
			log.Printf("Failed to save recharge request: %v", err)
			b.sendSimpleMessage(chatID, MsgGeneralError)
			b.session.Clear(chatID)
			return
		}

		b.session.Clear(chatID)

		successMsg := tgbotapi.NewMessage(chatID, MsgReceiptReceived)
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

		adminText := fmt.Sprintf(MsgAdminNewRequest, username, displayName, chatID, u.UserID, FormatMoney(req.Amount))

		for _, adminID := range b.cfg.AdminChatIDs {
			adminPhoto := tgbotapi.NewPhoto(adminID, tgbotapi.FileID(photo.FileID))
			adminPhoto.Caption = adminText
			adminPhoto.ParseMode = tgbotapi.ModeMarkdown
			adminPhoto.ReplyMarkup = AdminApprovalKeyboard(reqID)
			if _, err := b.api.Send(adminPhoto); err != nil {
				log.Printf("Failed to send admin notification to %d: %v", adminID, err)
			}
		}

	// Admin Settings States
	case StateAdminAwaitingCardNumber:
		if !isAdmin {
			b.session.Clear(chatID)
			return
		}
		num := strings.TrimSpace(msg.Text)
		if num == "" {
			b.sendSimpleMessage(chatID, "⚠️ شماره کارت نمی‌تواند خالی باشد. مجدداً وارد کنید:")
			return
		}

		b.session.SetTempCardNumber(chatID, num)
		b.session.SetState(chatID, StateAdminAwaitingCardOwner)
		b.sendSimpleMessage(chatID, MsgAdminAwaitingCardOwner)

	case StateAdminAwaitingCardOwner:
		if !isAdmin {
			b.session.Clear(chatID)
			return
		}
		owner := strings.TrimSpace(msg.Text)
		if owner == "" {
			b.sendSimpleMessage(chatID, "⚠️ نام صاحب کارت نمی‌تواند خالی باشد. مجدداً وارد کنید:")
			return
		}

		_ = b.db.SetSetting("card_number", sess.TempCardNumber)
		_ = b.db.SetSetting("card_owner", owner)

		b.session.Clear(chatID)
		reply := tgbotapi.NewMessage(chatID, MsgAdminCardUpdated)
		reply.ReplyMarkup = AdminMenuKeyboard()
		b.api.Send(reply)

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
		reply.ReplyMarkup = AdminMenuKeyboard()
		b.api.Send(reply)
	}
}

func (b *Bot) sendSimpleMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	b.api.Send(msg)
}
