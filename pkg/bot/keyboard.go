package bot

import (
	"fmt"
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

func AdminMenuKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(BtnAdminCardSettings),
			tgbotapi.NewKeyboardButton(BtnAdminPlansSettings),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(BtnAdminWelcomeSettings),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(BtnBack),
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

func BackKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(BtnBack),
		),
	)
}

func PlansInlineKeyboard(plans []db.Plan) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for i, plan := range plans {
		btn := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("%s - %s تومان", plan.Name, FormatMoney(plan.Price)),
			fmt.Sprintf("plan_detail_%d", i),
		)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(btn))
	}
	return tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: rows,
	}
}

func PurchaseConfirmKeyboard(planIndex int) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ بله، خرید", fmt.Sprintf("plan_buy_%d", planIndex)),
			tgbotapi.NewInlineKeyboardButtonData("❌ انصراف", "plan_cancel"),
		),
	)
}

func AdminApprovalKeyboard(rechargeRequestID int64) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ تأیید", fmt.Sprintf("admin_approve_%d", rechargeRequestID)),
			tgbotapi.NewInlineKeyboardButtonData("❌ رد", fmt.Sprintf("admin_reject_%d", rechargeRequestID)),
		),
	)
}

func AdminPlansInlineKeyboard(plans []db.Plan) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, plan := range plans {
		// Show plan name and price as a status button
		labelBtn := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("%s (%s تومان)", plan.Name, FormatMoney(plan.Price)),
			"admin_plan_noop",
		)
		// Delete button next to it
		delBtn := tgbotapi.NewInlineKeyboardButtonData(
			"❌ حذف",
			fmt.Sprintf("admin_plan_del_%d", plan.ID),
		)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(labelBtn, delBtn))
	}

	// Add plan button at the bottom
	addBtn := tgbotapi.NewInlineKeyboardButtonData(
		"➕ افزودن پلان جدید",
		"admin_plan_add",
	)
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(addBtn))

	return tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: rows,
	}
}

func SubscriptionConfigsKeyboard(subID int64) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔑 مشاهده کانفیگ‌ها", fmt.Sprintf("sub_configs_%d", subID)),
		),
	)
}
