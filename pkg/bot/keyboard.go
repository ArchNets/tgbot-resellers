package bot

import (
	"fmt"
	"strings"
	"time"

	"reseller-bot/pkg/backend"
	"reseller-bot/pkg/db"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func MainMenuKeyboard(isAdmin bool) tgbotapi.ReplyKeyboardMarkup {
	rows := [][]tgbotapi.KeyboardButton{
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(BtnBuyService),
			tgbotapi.NewKeyboardButton(BtnMySubscriptions),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(BtnAccountInfo),
			tgbotapi.NewKeyboardButton(BtnTopUpBalance),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(BtnContactSupport),
		),
	}

	if isAdmin {
		rows = append(rows, tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(BtnAdminPanel),
		))
	}

	return tgbotapi.NewReplyKeyboard(rows...)
}

func AdminMenuKeyboard(isOwner bool) tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(BtnAdminCardSettings),
			tgbotapi.NewKeyboardButton(BtnAdminPlansSettings),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(BtnAdminWelcomeSettings),
			tgbotapi.NewKeyboardButton(BtnAdminSupportSettings),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(BtnAdminTagSettings),
			tgbotapi.NewKeyboardButton(BtnAdminStaffSettings),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(BtnAdminChannelGate),
			tgbotapi.NewKeyboardButton(BtnAdminQRToggle),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(BtnAdminReminderToggle),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(BtnBack),
		),
	)
}

func StaffInlineKeyboard(staffList []db.Staff) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, s := range staffList {
		labelBtn := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("👤 %s (%d)", s.DisplayName, s.TelegramID),
			"staff_noop",
		)
		delBtn := tgbotapi.NewInlineKeyboardButtonData(
			"❌ حذف",
			fmt.Sprintf("staff_remove_%d", s.TelegramID),
		)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(labelBtn, delBtn))
	}

	addBtn := tgbotapi.NewInlineKeyboardButtonData(
		"➕ افزودن همکار جدید",
		"staff_add",
	)
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(addBtn))

	return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func ChannelGateInlineKeyboard(channelUsername string) tgbotapi.InlineKeyboardMarkup {
	username := strings.TrimPrefix(channelUsername, "@")
	joinURL := fmt.Sprintf("https://t.me/%s", username)

	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("🔗 عضویت در کانال", joinURL),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ عضو شدم", "check_channel_join"),
		),
	)
}

func AdminWelcomeSettingsKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(BtnAdminEditWelcomeText),
			tgbotapi.NewKeyboardButton(BtnAdminChangeWelcomeImg),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(BtnAdminDelWelcomeImg),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(BtnBack),
		),
	)
}

func AdminSupportSettingsKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(BtnAdminEditSupportText),
			tgbotapi.NewKeyboardButton(BtnAdminChangeSupportImg),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(BtnAdminDelSupportImg),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(BtnBack),
		),
	)
}

func BackKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(BtnBack),
		),
	)
}

func PlansInlineKeyboard(plans []backend.ResellerSubscribePlan, tag string) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, plan := range plans {
		btnData := fmt.Sprintf("plan_detail_%d", plan.ID)
		if tag != "" {
			btnData = fmt.Sprintf("plan_detail_%d_%s", plan.ID, tag)
		}
		btn := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("%s - %s تومان", plan.Name, FormatMoney(plan.UnitPrice)),
			btnData,
		)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(btn))
	}

	backBtn := tgbotapi.NewInlineKeyboardButtonData(Tr("fa", "back_to_list"), "back_to_tags")
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(backBtn))

	return tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: rows,
	}
}

func PlanDetailKeyboard(planID int64, tag string) tgbotapi.InlineKeyboardMarkup {
	buyData := fmt.Sprintf("plan_buy_confirm_%d", planID)
	backData := "back_to_tags"
	if tag != "" {
		buyData = fmt.Sprintf("plan_buy_confirm_%d_%s", planID, tag)
		backData = fmt.Sprintf("select_tag_%s", tag)
	}

	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ خرید اشتراک", buyData),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 بازگشت", backData),
		),
	)
}

func PurchaseConfirmKeyboard(planID int64, tag string) tgbotapi.InlineKeyboardMarkup {
	cancelData := fmt.Sprintf("plan_detail_%d", planID)
	if tag != "" {
		cancelData = fmt.Sprintf("plan_detail_%d_%s", planID, tag)
	}
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ بله، خرید", fmt.Sprintf("plan_buy_%d", planID)),
			tgbotapi.NewInlineKeyboardButtonData("❌ انصراف", cancelData),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 بازگشت", cancelData),
		),
	)
}

type TagItem struct {
	Original string
	Display  string
}

func TagsInlineKeyboard(tags []TagItem) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, tag := range tags {
		btn := tgbotapi.NewInlineKeyboardButtonData(
			tag.Display,
			fmt.Sprintf("select_tag_%s", tag.Original),
		)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(btn))
	}
	return tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: rows,
	}
}

func AdminApprovalKeyboard(rechargeRequestID int64) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ تأیید", fmt.Sprintf("admin_approve_%d", rechargeRequestID)),
			tgbotapi.NewInlineKeyboardButtonData("❌ رد", fmt.Sprintf("admin_reject_%d", rechargeRequestID)),
		),
	)
}

func AdminPlansInlineKeyboard(plans []backend.ResellerSubscribePlan) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, plan := range plans {
		labelBtn := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("%s (%s تومان)", plan.Name, FormatMoney(plan.UnitPrice)),
			"admin_plan_noop",
		)
		delBtn := tgbotapi.NewInlineKeyboardButtonData(
			"❌ حذف",
			fmt.Sprintf("admin_plan_del_%d", plan.ID),
		)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(labelBtn, delBtn))
	}

	addBtn := tgbotapi.NewInlineKeyboardButtonData(
		"➕ افزودن پلان جدید",
		"admin_plan_add",
	)
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(addBtn))

	return tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: rows,
	}
}

func getSubStatusEmojiAndLabel(item backend.SubscriptionItem, lang string) string {
	nowMs := time.Now().UnixMilli()
	usedTraffic := item.Upload + item.Download
	remTraffic := item.Traffic - usedTraffic
	if remTraffic < 0 {
		remTraffic = 0
	}

	isExpired := (item.ExpireTime > 0 && nowMs >= item.ExpireTime) || remTraffic <= 0
	isWarning := false
	if !isExpired {
		if item.Traffic > 0 && remTraffic <= item.Traffic/10 {
			isWarning = true
		}
		if item.ExpireTime > 0 && (item.ExpireTime-nowMs) <= 3*24*3600*1000 {
			isWarning = true
		}
	}

	if isExpired {
		return fmt.Sprintf("🔴 %s — %s", item.GetName(), Tr(lang, "sub_status_expired"))
	}
	if isWarning {
		remStr := Tr(lang, "remaining_traffic", FormatTraffic(remTraffic))
		return fmt.Sprintf("🟠 %s — %s", item.GetName(), remStr)
	}
	remStr := Tr(lang, "remaining_traffic", FormatTraffic(remTraffic))
	return fmt.Sprintf("🟢 %s — %s", item.GetName(), remStr)
}

func SubscriptionsListKeyboard(items []backend.SubscriptionItem, page int, totalPages int, lang string) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, item := range items {
		label := getSubStatusEmojiAndLabel(item, lang)
		btn := tgbotapi.NewInlineKeyboardButtonData(label, fmt.Sprintf("sub_detail_%d_%d", item.ID, page))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(btn))
	}

	if totalPages > 1 {
		var navRow []tgbotapi.InlineKeyboardButton

		prevBtn := tgbotapi.NewInlineKeyboardButtonData("⛔", "subs_noop")
		if page > 1 {
			prevBtn = tgbotapi.NewInlineKeyboardButtonData("◀️", fmt.Sprintf("subs_page_%d", page-1))
		}

		pageBtn := tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("صفحه %d/%d", page, totalPages), "subs_noop")

		nextBtn := tgbotapi.NewInlineKeyboardButtonData("⛔", "subs_noop")
		if page < totalPages {
			nextBtn = tgbotapi.NewInlineKeyboardButtonData("▶️", fmt.Sprintf("subs_page_%d", page+1))
		}

		navRow = append(navRow, prevBtn, pageBtn, nextBtn)
		rows = append(rows, navRow)
	}

	return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func SubscriptionDetailKeyboard(subID int64, hasOpenVPN, hasWireGuard bool, page int) *tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	var protoRow []tgbotapi.InlineKeyboardButton

	backPage := page
	if backPage < 1 {
		backPage = 1
	}

	if hasOpenVPN {
		protoRow = append(protoRow, tgbotapi.NewInlineKeyboardButtonData("📥 OpenVPN", fmt.Sprintf("dl_ovpn_%d_%d", subID, backPage)))
	}
	if hasWireGuard {
		protoRow = append(protoRow, tgbotapi.NewInlineKeyboardButtonData("📥 WireGuard", fmt.Sprintf("dl_wg_%d_%d", subID, backPage)))
	}

	if len(protoRow) > 0 {
		rows = append(rows, protoRow)
	}

	backBtn := tgbotapi.NewInlineKeyboardButtonData(Tr("fa", "back_to_list"), fmt.Sprintf("subs_page_%d", backPage))
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(backBtn))

	return &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func NodePickerKeyboard(subID int64, protocol string, nodes []backend.DownloadNode, page int) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	prefix := "dl_ovpn"
	if protocol == "wireguard" {
		prefix = "dl_wg"
	}
	backPage := page
	if backPage < 1 {
		backPage = 1
	}
	for _, n := range nodes {
		cbData := fmt.Sprintf("%s_%d_%d_%d", prefix, subID, n.NodeID, backPage)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(n.Name, cbData),
		))
	}
	backBtn := tgbotapi.NewInlineKeyboardButtonData("🔙 بازگشت", fmt.Sprintf("sub_detail_%d_%d", subID, backPage))
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(backBtn))

	return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func FormatsInlineKeyboard(subID, nodeID int64, formats []string) tgbotapi.InlineKeyboardMarkup {
	var row []tgbotapi.InlineKeyboardButton
	for _, fmtStr := range formats {
		btn := tgbotapi.NewInlineKeyboardButtonData(
			fmtStr,
			fmt.Sprintf("cfg:%d:%d:%s", subID, nodeID, fmtStr),
		)
		row = append(row, btn)
	}
	return tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{row},
	}
}


func AdminTagsInlineKeyboard(tags []TagItem) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, tag := range tags {
		btn := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("✏️ %s ➔ %s", tag.Original, tag.Display),
			fmt.Sprintf("admin_tag_edit_%s", tag.Original),
		)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(btn))
	}
	return tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: rows,
	}
}
