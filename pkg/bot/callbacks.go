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
	isAdmin := b.isAdmin(chatID)

	// Channel Gate Recheck
	if data == "check_channel_join" {
		if b.checkChannelGate(chatID) {
			b.answerCallback(cb.ID, "✅ عضویت شما تایید شد!", false)
			b.sendSimpleMessage(chatID, "✅ عضویت شما تایید شد. اکنون می‌توانید از ربات استفاده کنید.")
		} else {
			b.answerCallback(cb.ID, "⚠️ هنوز در کانال عضو نشده‌اید.", true)
		}
		return
	}

	if data == "subs_noop" {
		b.answerCallback(cb.ID, "", false)
		return
	}

	if data == "back_to_tags" {
		b.answerCallback(cb.ID, "", false)
		b.renderTagsMenu(chatID, cb.Message.MessageID, cb.From)
		return
	}

	if data == "cb_buy_service" {
		b.answerCallback(cb.ID, "", false)
		b.handleBuyService(chatID)
		return
	}

	if strings.HasPrefix(data, "subs_page_") {
		pageStr := strings.TrimPrefix(data, "subs_page_")
		page, _ := strconv.Atoi(pageStr)
		b.answerCallback(cb.ID, "", false)
		b.renderSubscriptionsListPage(chatID, cb.Message.MessageID, page, cb.From)
		return
	}

	if strings.HasPrefix(data, "sub_detail_") {
		parts := strings.Split(data, "_") // sub_detail_<id> or sub_detail_<id>_<page>
		if len(parts) < 3 {
			b.answerCallback(cb.ID, "اشتراک نامعتبر است.", true)
			return
		}
		subID, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			b.answerCallback(cb.ID, "اشتراک نامعتبر است.", true)
			return
		}
		page := 1
		if len(parts) >= 4 {
			if p, err := strconv.Atoi(parts[3]); err == nil && p > 0 {
				page = p
			}
		}
		b.answerCallback(cb.ID, "", false)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		u, err := b.db.GetUser(chatID)
		if err != nil || u == nil {
			b.sendSimpleMessage(chatID, MsgGeneralError)
			return
		}

		var targetItem *backend.SubscriptionItem
		subs, err := b.client.GetUserSubscriptions(ctx, u.UserID, 1, 100)
		if err == nil && subs != nil {
			for i, item := range subs.List {
				if item.ID == subID {
					targetItem = &subs.List[i]
					break
				}
			}
		}

		if targetItem == nil {
			b.sendSimpleMessage(chatID, "اشتراک یافت نشد.")
			return
		}

		nodes, err := b.client.GetDownloadNodes(ctx, targetItem.ID)
		if err != nil {
			log.Printf("Failed to fetch download nodes for sub %d: %v", targetItem.ID, err)
		}
		ps := analyzeDownloadNodes(nodes)

		subLink := b.getSubscribeLink(targetItem.Token)
		text := BuildSubscriptionDetailText(b.db, targetItem, subLink, getLang(cb.From))

		kb := SubscriptionDetailKeyboard(targetItem.ID, ps.HasOpenVPN, ps.HasWireGuard, page)
		b.sendSubscriptionMessage(chatID, text, subLink, kb)
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
				if b.cfg.BotID > 0 && p.BotID > 0 && p.BotID != b.cfg.BotID {
					continue
				}
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
		editMsg := tgbotapi.NewEditMessageText(chatID, cb.Message.MessageID, fmt.Sprintf("🛒 *پلان‌های بخش %s:*", escapeMarkdown(disp)))
		editMsg.ParseMode = tgbotapi.ModeMarkdown
		markup := PlansInlineKeyboard(plans, tag)
		editMsg.ReplyMarkup = &markup
		b.api.Send(editMsg)
		return
	}

	// 2. Handle Plan Selection and Purchase flows
	if strings.HasPrefix(data, "plan_detail_") {
		parts := strings.Split(data, "_") // plan_detail_<id> or plan_detail_<id>_<tag>
		if len(parts) < 3 {
			b.answerCallback(cb.ID, "پلان نامعتبر است.", true)
			return
		}
		planID, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			b.answerCallback(cb.ID, "پلان نامعتبر است.", true)
			return
		}
		var tag string
		if len(parts) >= 4 {
			tag = parts[3]
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		apiResp, err := b.client.GetResellerSubscribeList(ctx, 1, 100)
		if err != nil {
			b.answerCallback(cb.ID, MsgGeneralError, true)
			return
		}

		var selectedPlan *backend.ResellerSubscribePlan
		for i, plan := range apiResp.List {
			if plan.ID == planID {
				selectedPlan = &apiResp.List[i]
				break
			}
		}

		if selectedPlan == nil {
			b.answerCallback(cb.ID, "پلان یافت نشد.", true)
			return
		}

		userRegisterResp, err := b.client.RegisterUser(ctx, &backend.UserRegisterRequest{
			TelegramID: chatID,
		})
		if err != nil {
			b.answerCallback(cb.ID, MsgGeneralError, true)
			return
		}

		rate := b.rateMgr.GetRate(ctx, b.client)
		b.answerCallback(cb.ID, "", false)
		text := fmt.Sprintf(MsgPlanDetail,
			escapeMarkdown(selectedPlan.Name),
			FormatMoney(selectedPlan.UnitPrice),
			escapeMarkdown(selectedPlan.Description),
			FormatUserBalance(userRegisterResp.Balance, rate),
		)

		editMsg := tgbotapi.NewEditMessageText(chatID, cb.Message.MessageID, text)
		editMsg.ParseMode = tgbotapi.ModeMarkdown
		markup := PlanDetailKeyboard(selectedPlan.ID, tag)
		editMsg.ReplyMarkup = &markup
		b.api.Send(editMsg)
		return
	}

	if strings.HasPrefix(data, "plan_buy_confirm_") {
		parts := strings.Split(data, "_") // plan_buy_confirm_<id> or plan_buy_confirm_<id>_<tag>
		if len(parts) < 4 {
			b.answerCallback(cb.ID, "پلان نامعتبر است.", true)
			return
		}
		planID, err := strconv.ParseInt(parts[3], 10, 64)
		if err != nil {
			b.answerCallback(cb.ID, "پلان نامعتبر است.", true)
			return
		}
		var tag string
		if len(parts) >= 5 {
			tag = parts[4]
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		apiResp, err := b.client.GetResellerSubscribeList(ctx, 1, 100)
		if err != nil {
			b.answerCallback(cb.ID, MsgGeneralError, true)
			return
		}

		var selectedPlan *backend.ResellerSubscribePlan
		for i, plan := range apiResp.List {
			if plan.ID == planID {
				selectedPlan = &apiResp.List[i]
				break
			}
		}

		if selectedPlan == nil {
			b.answerCallback(cb.ID, "پلان یافت نشد.", true)
			return
		}

		b.answerCallback(cb.ID, "", false)
		text := fmt.Sprintf("📋 *تأیید خرید اشتراک*\n\nآیا از خرید پلان *%s* به مبلغ *%s* تومان اطمینان دارید؟",
			escapeMarkdown(selectedPlan.Name),
			FormatMoney(selectedPlan.UnitPrice),
		)

		editMsg := tgbotapi.NewEditMessageText(chatID, cb.Message.MessageID, text)
		editMsg.ParseMode = tgbotapi.ModeMarkdown
		markup := PurchaseConfirmKeyboard(selectedPlan.ID, tag)
		editMsg.ReplyMarkup = &markup
		b.api.Send(editMsg)
		return
	}

	if data == "plan_cancel" {
		b.answerCallback(cb.ID, "خرید لغو شد.", false)
		editMsg := tgbotapi.NewEditMessageText(chatID, cb.Message.MessageID, "❌ خرید لغو شد.")
		b.api.Send(editMsg)
		return
	}

	if strings.HasPrefix(data, "plan_buy_") {
		if !b.session.TryLock(chatID) {
			b.answerCallback(cb.ID, "درخواست شما در حال پردازش است...", true)
			return
		}
		defer b.session.Unlock(chatID)

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

		var selectedPlan *backend.ResellerSubscribePlan
		for i, plan := range apiResp.List {
			if plan.ID == planID {
				selectedPlan = &apiResp.List[i]
				break
			}
		}

		if selectedPlan == nil {
			b.answerCallback(cb.ID, "پلان یافت نشد.", true)
			return
		}

		userRegisterResp, err := b.client.RegisterUser(ctx, &backend.UserRegisterRequest{
			TelegramID: chatID,
		})
		if err != nil {
			b.answerCallback(cb.ID, MsgGeneralError, true)
			return
		}

		rate := b.rateMgr.GetRate(ctx, b.client)
		var insufficient bool
		if rate > 0 {
			userBalanceToman := int64(float64(userRegisterResp.Balance) * rate / 100.0)
			insufficient = userBalanceToman < selectedPlan.UnitPrice
		} else {
			insufficient = userRegisterResp.Balance <= 0
		}

		if insufficient {
			b.answerCallback(cb.ID, "موجودی کافی نیست.", true)
			text := fmt.Sprintf(MsgInsufficientBalance,
				FormatUserBalance(userRegisterResp.Balance, rate),
				FormatMoney(selectedPlan.UnitPrice),
			)
			editMsg := tgbotapi.NewEditMessageText(chatID, cb.Message.MessageID, text)
			editMsg.ParseMode = tgbotapi.ModeMarkdown
			b.api.Send(editMsg)
			return
		}

		b.session.SetPurchasingPlan(chatID, selectedPlan.ID, selectedPlan.Name, selectedPlan.UnitTime)
		b.session.SetState(chatID, StateAwaitingSubCustomName)

		b.answerCallback(cb.ID, "", false)
		reply := tgbotapi.NewMessage(chatID, Tr(getLang(cb.From), "ask_sub_custom_name"))
		reply.ReplyMarkup = BackKeyboard()
		b.api.Send(reply)
		return
	}

	if strings.HasPrefix(data, "dl_ovpn_") || strings.HasPrefix(data, "dl_wg_") {
		parts := strings.Split(data, "_")
		if len(parts) < 3 {
			b.answerCallback(cb.ID, "درخواست نامعتبر است.", true)
			return
		}

		isOvpn := parts[1] == "ovpn"
		format := "openvpn"
		if !isOvpn {
			format = "wireguard"
		}

		subID, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			b.answerCallback(cb.ID, "شناسه اشتراک نامعتبر است.", true)
			return
		}

		page := 1
		var nodeID int64
		if len(parts) == 4 {
			if p, err := strconv.Atoi(parts[3]); err == nil && p > 0 {
				page = p
			}
		} else if len(parts) >= 5 {
			if nid, err := strconv.ParseInt(parts[3], 10, 64); err == nil {
				nodeID = nid
			}
			if p, err := strconv.Atoi(parts[4]); err == nil && p > 0 {
				page = p
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if nodeID > 0 {
			b.answerCallback(cb.ID, "در حال دریافت فایل تنظیمات...", false)
			b.sendProfileDocument(ctx, chatID, subID, nodeID, format)
			return
		}

		nodes, err := b.client.GetDownloadNodes(ctx, subID)
		if err != nil || len(nodes) == 0 {
			b.answerCallback(cb.ID, Tr(getLang(cb.From), "no_nodes_found"), true)
			return
		}

		ps := analyzeDownloadNodes(nodes)
		var matching []backend.DownloadNode
		if isOvpn {
			matching = ps.OpenVPNNodes
		} else {
			matching = ps.WireGuardNodes
		}

		if len(matching) == 0 {
			b.answerCallback(cb.ID, Tr(getLang(cb.From), "no_nodes_found"), true)
			return
		}

		if len(matching) == 1 {
			b.answerCallback(cb.ID, "در حال دریافت فایل تنظیمات...", false)
			b.sendProfileDocument(ctx, chatID, subID, matching[0].NodeID, format)
			return
		}

		b.answerCallback(cb.ID, "", false)
		prompt := Tr(getLang(cb.From), "select_node_prompt")
		replyMarkup := NodePickerKeyboard(subID, format, matching, page)
		reply := tgbotapi.NewMessage(chatID, prompt)
		reply.ReplyMarkup = replyMarkup
		b.api.Send(reply)
		return
	}

	// 3. Handle Admin Approval and Rejection flows
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

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		err = b.client.ReviewRecharge(ctx, &backend.RechargeReviewRequest{
			ID:     reqID,
			Action: "approve",
		})
		if err != nil {
			if backend.IsErrorCode(err, 62001) {
				b.answerCallback(cb.ID, Tr(getLang(cb.From), "already_reviewed"), true)
				origText := cb.Message.Caption
				if origText == "" {
					origText = cb.Message.Text
				}
				newText := origText + "\n\n" + Tr(getLang(cb.From), "already_reviewed")
				b.updateMessageTextOrCaption(chatID, cb.Message.MessageID, newText)
				return
			}
			log.Printf("Backend review recharge approve error for order %d: %v", reqID, err)
			b.answerCallback(cb.ID, MsgGeneralError, true)
			return
		}

		b.answerCallback(cb.ID, "تأیید شد.", false)

		adminUser := cb.From.FirstName
		if cb.From.LastName != "" {
			adminUser += " " + cb.From.LastName
		}
		if cb.From.UserName != "" {
			adminUser += " (@" + cb.From.UserName + ")"
		}

		origText := cb.Message.Caption
		if origText == "" {
			origText = cb.Message.Text
		}
		newText := origText + fmt.Sprintf("\n\n✅ *تأیید شد توسط:* %s\n*زمان:* %s", escapeMarkdown(adminUser), time.Now().Format("15:04:05"))
		b.updateMessageTextOrCaption(chatID, cb.Message.MessageID, newText)

		// Send customer DM immediately & mark notified in SQLite
		listResp, err := b.client.GetRechargeList(ctx, "customer_topup", "", 1, 100)
		if err == nil && listResp != nil {
			for _, order := range listResp.List {
				if order.ID == reqID {
					user, err := b.db.GetUserByBackendID(order.CustomerUserID)
					if err == nil && user != nil {
						msgText := Tr(getLang(&tgbotapi.User{ID: user.TelegramID}), "balance_credited", FormatMoney(order.OriginalAmount))
						b.sendSimpleMessage(user.TelegramID, msgText)
						_ = b.db.SaveRechargeNotified(order.ID, "approved")
					}
					break
				}
			}
		}
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

		b.answerCallback(cb.ID, "", false)
		b.session.SetState(chatID, StateAdminAwaitingRejectReason)
		b.session.SetRejectOrderID(chatID, reqID)

		reply := tgbotapi.NewMessage(chatID, Tr(getLang(cb.From), "reject_reason_prompt"))
		reply.ReplyMarkup = BackKeyboard()
		b.api.Send(reply)
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

func (b *Bot) updateMessageTextOrCaption(chatID int64, messageID int, text string) {
	editCaption := tgbotapi.NewEditMessageCaption(chatID, messageID, text)
	editCaption.ParseMode = tgbotapi.ModeMarkdown
	editCaption.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{}}
	if _, err := b.api.Send(editCaption); err != nil {
		editText := tgbotapi.NewEditMessageText(chatID, messageID, text)
		editText.ParseMode = tgbotapi.ModeMarkdown
		editText.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{}}
		b.api.Send(editText)
	}
}

func calculateExpiryMs(unitTime string) int64 {
	now := time.Now()
	switch strings.ToLower(strings.TrimSpace(unitTime)) {
	case "month":
		return now.AddDate(0, 1, 0).UnixMilli()
	case "quarter":
		return now.AddDate(0, 3, 0).UnixMilli()
	case "half_year":
		return now.AddDate(0, 6, 0).UnixMilli()
	case "year":
		return now.AddDate(1, 0, 0).UnixMilli()
	default:
		return 0
	}
}

func (b *Bot) sendProfileDocument(ctx context.Context, chatID, subID, nodeID int64, format string) {
	prof, err := b.client.GetUserSubscribeProfile(ctx, subID, nodeID, format)
	if err != nil || prof == nil || len(prof.Content) == 0 {
		b.sendSimpleMessage(chatID, Tr(getLang(nil), "download_profile_error"))
		return
	}

	filename := prof.Filename
	if filename == "" {
		if format == "openvpn" {
			filename = "config.ovpn"
		} else {
			filename = "config.conf"
		}
	}

	doc := tgbotapi.NewDocument(chatID, tgbotapi.FileBytes{
		Name:  filename,
		Bytes: prof.Content,
	})
	if _, err := b.api.Send(doc); err != nil {
		log.Printf("Failed to send profile document to %d: %v", chatID, err)
	}
}

