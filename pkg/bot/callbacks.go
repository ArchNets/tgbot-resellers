package bot

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"reseller-bot/pkg/backend"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) handleCallbackQuery(cb *tgbotapi.CallbackQuery) {
	chatID := cb.Message.Chat.ID
	data := cb.Data
	log.Printf("Received callback query from %d: %s", chatID, data)

	// Check if user is admin
	isAdmin := false
	for _, adminID := range b.cfg.AdminChatIDs {
		if chatID == adminID {
			isAdmin = true
			break
		}
	}

	// Fetch user details from database
	u, err := b.db.GetUser(chatID)
	if err != nil {
		log.Printf("DB error fetching user: %v", err)
		b.answerCallback(cb.ID, MsgGeneralError, true)
		return
	}

	// 1. Select Tag/Category Flow
	if strings.HasPrefix(data, "select_tag_") {
		tag := strings.TrimPrefix(data, "select_tag_")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		apiResp, err := b.client.GetResellerSubscribeList(ctx, 1, 100)
		if err != nil {
			b.answerCallback(cb.ID, MsgGeneralError, true)
			return
		}

		var plans []backend.ResellerSubscribePlan
		for _, p := range apiResp.List {
			if p.Show {
				hasTag := false
				for _, t := range p.NodeTags {
					if strings.TrimSpace(t) == tag {
						hasTag = true
						break
					}
				}
				if hasTag {
					plans = append(plans, p)
				}
			}
		}

		// Sort by Traffic ascending, then DeviceLimit
		sort.Slice(plans, func(i, j int) bool {
			if plans[i].Traffic != plans[j].Traffic {
				return plans[i].Traffic < plans[j].Traffic
			}
			return plans[i].DeviceLimit < plans[j].DeviceLimit
		})

		disp, err := b.db.GetTagMapping(tag)
		if err != nil || disp == "" {
			disp = tag
		}

		b.answerCallback(cb.ID, "", false)
		editMsg := tgbotapi.NewEditMessageText(chatID, cb.Message.MessageID, fmt.Sprintf("🛒 *پلان‌های بخش %s:*", disp))
		editMsg.ParseMode = tgbotapi.ModeMarkdown
		markup := PlansInlineKeyboard(plans)
		editMsg.ReplyMarkup = &markup
		b.api.Send(editMsg)
		return
	}

	// 2. Handle Plan Selection and Purchase flows
	if strings.HasPrefix(data, "plan_detail_") {
		idStr := strings.TrimPrefix(data, "plan_detail_")
		planID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			b.answerCallback(cb.ID, "پلان نامعتبر است.", true)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		apiResp, err := b.client.GetResellerSubscribeList(ctx, 1, 100)
		if err != nil {
			b.answerCallback(cb.ID, MsgGeneralError, true)
			return
		}

		var plan *backend.ResellerSubscribePlan
		for i, p := range apiResp.List {
			if p.ID == planID {
				plan = &apiResp.List[i]
				break
			}
		}

		if plan == nil {
			b.answerCallback(cb.ID, "پلان نامعتبر است یا یافت نشد.", true)
			return
		}

		// Fetch current balance from backend
		resp, err := b.client.RegisterUser(ctx, &backend.UserRegisterRequest{
			TelegramID: chatID,
		})
		if err != nil {
			log.Printf("Backend balance fetch error: %v", err)
			b.answerCallback(cb.ID, MsgGeneralError, true)
			return
		}

		b.answerCallback(cb.ID, "", false)

		text := fmt.Sprintf(MsgPlanDetail,
			plan.Name,
			FormatMoney(plan.UnitPrice),
			plan.Description,
			FormatMoney(resp.Balance),
		)

		editMsg := tgbotapi.NewEditMessageText(chatID, cb.Message.MessageID, text)
		editMsg.ParseMode = tgbotapi.ModeMarkdown
		markup := PurchaseConfirmKeyboard(plan.ID)
		editMsg.ReplyMarkup = &markup
		b.api.Send(editMsg)
		return
	}

	if data == "plan_cancel" {
		b.answerCallback(cb.ID, "خرید انصراف داده شد.", false)
		editMsg := tgbotapi.NewEditMessageText(chatID, cb.Message.MessageID, "❌ فرآیند خرید لغو شد.")
		b.api.Send(editMsg)
		return
	}

	if strings.HasPrefix(data, "plan_buy_") {
		idStr := strings.TrimPrefix(data, "plan_buy_")
		planID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			b.answerCallback(cb.ID, "پلان نامعتبر است.", true)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		apiResp, err := b.client.GetResellerSubscribeList(ctx, 1, 100)
		if err != nil {
			b.answerCallback(cb.ID, MsgGeneralError, true)
			return
		}

		var plan *backend.ResellerSubscribePlan
		for i, p := range apiResp.List {
			if p.ID == planID {
				plan = &apiResp.List[i]
				break
			}
		}

		if plan == nil {
			b.answerCallback(cb.ID, "پلان یافت نشد.", true)
			return
		}

		if u == nil {
			b.answerCallback(cb.ID, "کاربر یافت نشد.", true)
			return
		}

		// Retrieve latest balance
		profile, err := b.client.RegisterUser(ctx, &backend.UserRegisterRequest{
			TelegramID: chatID,
		})
		if err != nil {
			log.Printf("Backend register profile check failed: %v", err)
			b.answerCallback(cb.ID, MsgGeneralError, true)
			return
		}

		if profile.Balance < plan.UnitPrice {
			b.answerCallback(cb.ID, "موجودی کافی نیست.", true)
			text := fmt.Sprintf(MsgInsufficientBalance, FormatMoney(profile.Balance), FormatMoney(plan.UnitPrice))
			editMsg := tgbotapi.NewEditMessageText(chatID, cb.Message.MessageID, text)
			editMsg.ParseMode = tgbotapi.ModeMarkdown
			b.api.Send(editMsg)
			return
		}

		// Deduct user wallet balance first
		err = b.client.UpdateUserBalance(ctx, &backend.BalanceUpdateRequest{
			UserID: u.UserID,
			Amount: -plan.UnitPrice,
			Reason: fmt.Sprintf("خرید پلان: %s", plan.Name),
		})
		if err != nil {
			log.Printf("Backend update balance deduction failed: %v", err)
			b.answerCallback(cb.ID, "خطا در کسر موجودی از کیف پول شما.", true)
			return
		}

		// Call backend to provision subscription
		subResp, err := b.client.CreateSubscription(ctx, &backend.SubscribeRequest{
			UserID:      u.UserID,
			SubscribeID: int(plan.ID),
		})
		if err != nil {
			log.Printf("Backend subscription creation failed: %v", err)
			// Refund user balance
			_ = b.client.UpdateUserBalance(ctx, &backend.BalanceUpdateRequest{
				UserID: u.UserID,
				Amount: plan.UnitPrice,
				Reason: fmt.Sprintf("استرداد خرید ناموفق: %s", plan.Name),
			})
			b.answerCallback(cb.ID, "خطا در فعال‌سازی سرویس. موجودی شما بازگردانده شد.", true)
			return
		}

		b.answerCallback(cb.ID, "خرید با موفقیت انجام شد!", false)

		// Success response
		editMsg := tgbotapi.NewEditMessageText(chatID, cb.Message.MessageID, MsgPurchaseSuccess)
		b.api.Send(editMsg)

		// Fetch the newly created subscription configuration to display it
		subs, err := b.client.GetUserSubscriptions(ctx, u.UserID, 1, 10)
		if err == nil && len(subs.List) > 0 {
			var targetItem *backend.SubscriptionItem
			for i, item := range subs.List {
				if item.UUID == subResp.UUID {
					targetItem = &subs.List[i]
					break
				}
			}
			if targetItem == nil {
				targetItem = &subs.List[0]
			}

			expStr := "بدون انقضا"
			if targetItem.ExpireTime > 0 {
				expStr = time.Unix(targetItem.ExpireTime, 0).Format("2006-01-02 15:04:05")
			}

			configsBlock := ""
			if len(targetItem.Configs) > 0 {
				configsBlock = "```\n" + strings.Join(targetItem.Configs, "\n\n") + "\n```"
			} else {
				configsBlock = "_کانفیگی یافت نشد. لطفاً از بخش اشتراک‌های من مجدداً تلاش کنید._"
			}

			text := fmt.Sprintf(MsgSubDetail,
				targetItem.Name,
				targetItem.UUID,
				FormatTraffic(targetItem.TotalTraffic),
				FormatTraffic(targetItem.UsedTraffic),
				expStr,
			) + configsBlock

			msg := tgbotapi.NewMessage(chatID, text)
			msg.ParseMode = tgbotapi.ModeMarkdown
			b.api.Send(msg)
		}
		return
	}

	// 2. Handle Admin Approval and Rejection flows
	if strings.HasPrefix(data, "admin_approve_") {
		if !isAdmin {
			b.answerCallback(cb.ID, "شما دسترسی به این بخش را ندارید.", true)
			return
		}

		reqIDStr := strings.TrimPrefix(data, "admin_approve_")
		reqID, err := strconv.ParseInt(reqIDStr, 10, 64)
		if err != nil {
			b.answerCallback(cb.ID, "شناسه درخواست نامعتبر است.", true)
			return
		}

		req, err := b.db.GetRechargeRequest(reqID)
		if err != nil {
			log.Printf("DB error fetching request: %v", err)
			b.answerCallback(cb.ID, "خطا در دیتابیس.", true)
			return
		}

		if req == nil {
			b.answerCallback(cb.ID, "درخواست یافت نشد.", true)
			return
		}

		if req.Status != "pending" {
			b.answerCallback(cb.ID, "این درخواست قبلاً تعیین تکلیف شده است.", true)
			b.removeInlineKeyboard(chatID, cb.Message.MessageID, cb.Message.Text)
			return
		}

		if err := b.db.UpdateRechargeStatus(reqID, "approved"); err != nil {
			log.Printf("Failed to update status in DB: %v", err)
			b.answerCallback(cb.ID, "خطا در بروزرسانی دیتابیس.", true)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		err = b.client.UpdateUserBalance(ctx, &backend.BalanceUpdateRequest{
			UserID: req.UserID,
			Amount: req.Amount,
			Reason: "Card-to-card recharge approval",
		})
		if err != nil {
			log.Printf("Backend balance update failed: %v", err)
			_ = b.db.UpdateRechargeStatus(reqID, "pending")
			b.answerCallback(cb.ID, "خطا در کسر/افزایش اعتبار در سرور اصلی.", true)
			return
		}

		b.answerCallback(cb.ID, "تأیید شد.", false)

		notifyUserText := fmt.Sprintf(MsgRequestApproved, FormatMoney(req.Amount))
		b.sendSimpleMessage(req.TelegramID, notifyUserText)

		adminUser := cb.From.FirstName
		if cb.From.LastName != "" {
			adminUser += " " + cb.From.LastName
		}
		newCaption := cb.Message.Caption + fmt.Sprintf("\n\n✅ *تأیید شد توسط:* %s\n*زمان:* %s", adminUser, time.Now().Format("15:04:05"))
		b.updateMessageCaption(chatID, cb.Message.MessageID, newCaption)
		return
	}

	if strings.HasPrefix(data, "admin_reject_") {
		if !isAdmin {
			b.answerCallback(cb.ID, "شما دسترسی به این بخش را ندارید.", true)
			return
		}

		reqIDStr := strings.TrimPrefix(data, "admin_reject_")
		reqID, err := strconv.ParseInt(reqIDStr, 10, 64)
		if err != nil {
			b.answerCallback(cb.ID, "شناسه درخواست نامعتبر است.", true)
			return
		}

		req, err := b.db.GetRechargeRequest(reqID)
		if err != nil {
			log.Printf("DB error fetching request: %v", err)
			b.answerCallback(cb.ID, "خطا در دیتابیس.", true)
			return
		}

		if req == nil {
			b.answerCallback(cb.ID, "درخواست یافت نشد.", true)
			return
		}

		if req.Status != "pending" {
			b.answerCallback(cb.ID, "این درخواست قبلاً تعیین تکلیف شده است.", true)
			b.removeInlineKeyboard(chatID, cb.Message.MessageID, cb.Message.Text)
			return
		}

		if err := b.db.UpdateRechargeStatus(reqID, "rejected"); err != nil {
			log.Printf("Failed to update status in DB: %v", err)
			b.answerCallback(cb.ID, "خطا در بروزرسانی دیتابیس.", true)
			return
		}

		b.answerCallback(cb.ID, "رد شد.", false)

		notifyUserText := fmt.Sprintf(MsgRequestRejected, FormatMoney(req.Amount))
		b.sendSimpleMessage(req.TelegramID, notifyUserText)

		adminUser := cb.From.FirstName
		if cb.From.LastName != "" {
			adminUser += " " + cb.From.LastName
		}
		newCaption := cb.Message.Caption + fmt.Sprintf("\n\n❌ *رد شد توسط:* %s\n*زمان:* %s", adminUser, time.Now().Format("15:04:05"))
		b.updateMessageCaption(chatID, cb.Message.MessageID, newCaption)
		return
	}

	// 3. Handle Admin Plan Panel actions
	if data == "admin_plan_noop" {
		b.answerCallback(cb.ID, "", false)
		return
	}

	if data == "admin_plan_add" {
		if !isAdmin {
			b.answerCallback(cb.ID, "شما دسترسی به این بخش را ندارید.", true)
			return
		}
		b.answerCallback(cb.ID, "", false)

		// Ask admin to send subscribe_id
		reply := tgbotapi.NewMessage(chatID, MsgAdminAwaitingPlanSubscribeID)
		reply.ReplyMarkup = BackKeyboard()
		b.api.Send(reply)

		b.session.SetState(chatID, StateAdminAwaitingPlanSubscribeID)
		return
	}

	if strings.HasPrefix(data, "admin_plan_del_") {
		if !isAdmin {
			b.answerCallback(cb.ID, "شما دسترسی به این بخش را ندارید.", true)
			return
		}
		idStr := strings.TrimPrefix(data, "admin_plan_del_")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			b.answerCallback(cb.ID, "شناسه پلان معتبر نیست.", true)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		err = b.client.DeleteResellerSubscribe(ctx, &backend.DeleteResellerSubscribeRequest{ID: id})
		if err != nil {
			log.Printf("Failed to delete plan %d from backend: %v", id, err)
			b.answerCallback(cb.ID, "خطا در حذف پلان از سرور.", true)
			return
		}

		b.answerCallback(cb.ID, MsgAdminPlanDeleted, false)

		// Reload plans list
		apiResp, _ := b.client.GetResellerSubscribeList(ctx, 1, 100)
		if apiResp != nil {
			markup := AdminPlansInlineKeyboard(apiResp.List)
			editMsg := tgbotapi.NewEditMessageReplyMarkup(chatID, cb.Message.MessageID, markup)
			b.api.Send(editMsg)
		}
		return
	}

	if strings.HasPrefix(data, "admin_tag_edit_") {
		if !isAdmin {
			b.answerCallback(cb.ID, "شما دسترسی به این بخش را ندارید.", true)
			return
		}
		b.answerCallback(cb.ID, "", false)
		tag := strings.TrimPrefix(data, "admin_tag_edit_")

		reply := tgbotapi.NewMessage(chatID, fmt.Sprintf("✏️ لطفاً نام نمایشی جدید برای دسته '%s' را ارسال کنید:", tag))
		reply.ReplyMarkup = BackKeyboard()
		b.api.Send(reply)

		b.session.SetState(chatID, StateAdminAwaitingTagDisplayName)
		b.session.SetTempPlanName(chatID, tag) // TempPlanName stores original tag name
		return
	}
}

func (b *Bot) answerCallback(callbackQueryID string, text string, showAlert bool) {
	msg := tgbotapi.NewCallbackWithAlert(callbackQueryID, text)
	msg.ShowAlert = showAlert
	if _, err := b.api.Request(msg); err != nil {
		log.Printf("Failed to answer callback query: %v", err)
	}
}

func (b *Bot) removeInlineKeyboard(chatID int64, messageID int, text string) {
	editMsg := tgbotapi.NewEditMessageReplyMarkup(chatID, messageID, tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{}})
	b.api.Send(editMsg)
}

func (b *Bot) updateMessageCaption(chatID int64, messageID int, caption string) {
	editMsg := tgbotapi.NewEditMessageCaption(chatID, messageID, caption)
	editMsg.ParseMode = tgbotapi.ModeMarkdown
	editMsg.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{}}
	b.api.Send(editMsg)
}
