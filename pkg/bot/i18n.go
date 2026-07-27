package bot

import (
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// Use localized printing for numbers (e.g. 50,000 instead of 50000)
var printer = message.NewPrinter(language.Persian)

func FormatMoney(amount int64) string {
	return printer.Sprintf("%d", amount)
}

func FormatUserBalance(cents int64, rate float64) string {
	if rate <= 0 {
		return fmt.Sprintf("$%.2f", float64(cents)/100.0)
	}
	toman := int64(float64(cents) * rate / 100.0)
	return printer.Sprintf("%d تومان", toman)
}

const (
	// Keyboard Buttons
	BtnAccountInfo      = "👤 اطلاعات حساب"
	BtnBuyService       = "🛒 خرید سرویس"
	BtnMySubscriptions  = "🔑 اشتراک‌های من"
	BtnTopUpBalance     = "💳 افزایش موجودی"
	BtnContactSupport   = "📞 پشتیبانی"
	BtnBack             = "🔙 بازگشت به منوی اصلی"

	// Messages
	MsgWelcome = `سلام {name} عزیز! 
به ربات تلگرام نمایندگی فروش خوش آمدید.
لطفاً از دکمه‌های زیر جهت مدیریت حساب و خرید سرویس استفاده کنید.`

	MsgAccountInfo = `👤 *پروفایل کاربری شما:*

• شناسه تلگرام: %d
• شناسه کاربری سیستم: %d
• موجودی کیف پول: *%s*`

	MsgTopUpCardInfo = `💳 *افزایش موجودی کیف پول*

لطفاً مبلغ مورد نظر خود را به کارت زیر واریز نمایید:

• *شماره کارت:* %s
• *به نام:* %s

*نکته:* پس از واریز وجه، لطفاً مبلغ واریز شده را *به تومان* وارد کنید (مثال: ۵۰۰۰۰):`

	MsgInvalidAmount = `⚠️ لطفاً فقط یک عدد معتبر بزرگتر از صفر (مبلغ به تومان) وارد کنید.`

	MsgSendReceipt = `💵 مبلغ واریزی شما: *%s* تومان ثبت شد.

حالا لطفاً تصویر یا اسکرین‌شات رسید پرداخت خود را ارسال کنید:`

	MsgInvalidReceipt = `⚠️ خطا! لطفاً تصویر یا اسکرین‌شات رسید واریز را ارسال نمایید.`

	MsgReceiptReceived = `✅ رسید پرداخت شما دریافت شد و برای بررسی و تأیید به مدیریت ارسال گردید.
پس از تأیید، موجودی کیف پول شما افزایش خواهد یافت و به شما اطلاع‌رسانی می‌شود.`

	MsgPlansList = `🛒 *پلان‌های اشتراک در دسترس:*

لطفاً پلان مورد نظر خود را برای خرید انتخاب کنید:`

	MsgPlanDetail = `🛒 *پیش‌خرید اشتراک*

• پلان: *%s*
• قیمت: *%s* تومان
• توضیحات: %s

• موجودی فعلی شما: *%s*

آیا از خرید این پلان اطمینان دارید؟`

	MsgInsufficientBalance = `❌ موجودی کیف پول شما کافی نیست!
• موجودی فعلی: *%s*
• قیمت پلان: *%s* تومان

لطفاً ابتدا کیف پول خود را افزایش موجودی دهید.`

	MsgPurchaseSuccess = `✅ خرید شما با موفقیت انجام شد!
کانفیگ‌های شما در زیر آماده استفاده هستند.`

	MsgNoSubscriptions = `📭 شما در حال حاضر هیچ اشتراک فعالی ندارید.`

	MsgSubscriptionsList = `🔑 *لیست اشتراک‌های شما:*`

	MsgSubDetail = `📦 *اشتراک: %s*
- نوع پلان: %s
%s- دستگاه‌ها: %s
- تاریخ خرید: %s
- حجم کل: %s
- حجم مصرفی: %s
%s
- تاریخ انقضا: %s

لینک اشتراک:
` + "```\n%s\n```"

	MsgContactSupportText = `📞 *پشتیبانی فنی*

در صورت بروز هرگونه مشکل یا سوال، می‌توانید با ما در ارتباط باشید:
آیدی مدیریت: @admin`

	MsgAdminNewRequest = `🚨 *درخواست افزایش موجودی جدید*

• کاربر: @%s (%s)
• شناسه تلگرام: %d
• شناسه سیستم: %d
• مبلغ: *%s* تومان`

	MsgRequestApproved = `✅ درخواست افزایش موجودی شما به مبلغ *%s* تومان توسط مدیریت تأیید شد.`
	MsgRequestRejected = `❌ درخواست افزایش موجودی شما به مبلغ *%s* تومان توسط مدیریت رد شد.`

	// Admin Keyboard Buttons
	BtnAdminPanel          = "⚙️ پنل مدیریت"
	BtnAdminCardSettings   = "💳 تنظیمات کارت بانکی"
	BtnAdminPlansSettings  = "📦 مدیریت پلان‌ها"
	BtnAdminTagSettings    = "🏷️ مدیریت نام دسته‌ها"
	BtnAdminStaffSettings  = "👥 همکاران (Staff)"
	BtnAdminChannelGate    = "📢 کانال اجباری"
	BtnAdminQRToggle       = "📱 تنظیمات QR کد"
	BtnAdminReminderToggle = "⏰ یادآور انقضا"

	// Payments & Recharge
	MsgPaymentsUnavailable  = "payments_unavailable"
	MsgPendingLimitExceeded = "pending_limit_exceeded"
	MsgAlreadyReviewed      = "already_reviewed"

	// Admin Messages
	MsgAdminPanelWelcome = `⚙️ *به پنل مدیریت ربات خوش آمدید.*

لطفاً یک گزینه را انتخاب کنید:`

	MsgAdminAwaitingCardNumber = `💳 لطفاً شماره کارت بانکی جدید را وارد کنید:
(مثال: ۶۰۳۷-۹۹۱۱-۲۲۳۳-۴۴۵۵)`

	MsgAdminAwaitingCardOwner = `👤 شماره کارت ثبت شد.
حالا لطفاً نام صاحب کارت جدید را وارد کنید:`

	MsgAdminAwaitingCardBank = `🏦 نام صاحب کارت ثبت شد.
حالا لطفاً نام بانک را وارد کنید (مثال: بانک ملی):`

	MsgAdminAwaitingCardInstructions = `📝 نام بانک ثبت شد.
حالا توضیحات و راهنمای واریز را وارد کنید (یا '-' برای خالی گذاشتن):`

	MsgAdminCardUpdated = `✅ اطلاعات کارت بانکی با موفقیت در سیستم ثبت شد.`

	MsgAdminPlansList = `📦 *لیست پلان‌های تعریف شده:*

برای حذف یک پلان روی دکمه مربوطه کلیک کنید. جهت افزودن پلان جدید، دکمه افزودن پلان را بزنید.`

	MsgAdminPlanDeleted = `✅ پلان با موفقیت حذف شد.`

	MsgAdminAwaitingPlanSubscribeID = `➕ *افزودن پلان جدید*

لطفاً شناسه پلان (subscribe_id مربوط به پنل اصلی) را وارد کنید:`

	MsgAdminInvalidSubscribeID = `⚠️ خطا! شناسه پلان باید یک عدد معتبر باشد. مجدداً تلاش کنید:`

	MsgAdminAwaitingPlanName = `✅ شناسه ثبت شد.
حالا نام پلان را وارد کنید (مثال: پلان برنزی - ۵۰ گیگ):`

	MsgAdminAwaitingPlanPrice = `✅ نام ثبت شد.
حالا قیمت پلان را *به تومان* وارد کنید (مثال: ۱۵۰۰۰۰):`

	MsgAdminInvalidPrice = `⚠️ خطا! قیمت پلان باید یک عدد معتبر باشد. مجدداً تلاش کنید:`

	MsgAdminAwaitingPlanDescription = `✅ قیمت ثبت شد.
حالا توضیحات پلان را وارد کنید (مثال: حجم ۵۰ گیگابایت - اعتبار ۱ ماه):`

	MsgAdminPlanAdded = `✅ پلان جدید با موفقیت به دیتابیس اضافه شد.`

	// Welcome Message Admin Settings
	BtnAdminWelcomeSettings  = "📝 مدیریت پیام خوش‌آمد"
	BtnAdminEditWelcomeText  = "✏️ ویرایش متن خوش‌آمد"
	BtnAdminChangeWelcomeImg = "🖼️ تغییر عکس خوش‌آمد"
	BtnAdminDelWelcomeImg    = "❌ حذف عکس خوش‌آمد"

	MsgAdminWelcomeSettingsMenu = `📝 *مدیریت پیام و تصویر خوش‌آمدگویی*

در این بخش می‌توانید متن خوش‌آمدگویی و تصویر بالای آن را مدیریت کنید. لطفاً یک گزینه را انتخاب کنید:`

	MsgAdminAwaitingWelcomeText = `📝 لطفاً متن جدید پیام خوش‌آمدگویی را ارسال کنید.

*نکته:* می‌توانید از عبارت '{name}' در متن خود استفاده کنید تا نام کاربر به‌صورت خودکار جایگزین شود.`

	MsgAdminWelcomeTextUpdated = `✅ متن پیام خوش‌آمدگویی با موفقیت بروزرسانی شد.`

	MsgAdminAwaitingWelcomeImg = `🖼️ لطفاً تصویر یا اسکرین‌شات جدید برای بالای پیام خوش‌آمدگویی ارسال کنید:`

	MsgAdminWelcomeImgUpdated = `✅ تصویر پیام خوش‌آمدگویی با موفقیت در دیتابیس ثبت شد.`

	MsgAdminWelcomeImgDeleted = `✅ تصویر پیام خوش‌آمدگویی با موفقیت حذف شد و از این پس پیام به‌صورت متنی ارسال خواهد شد.`

	MsgGeneralError = `❌ متأسفانه خطایی در ارتباط با سرور رخ داده است. لطفاً بعداً تلاش کنید.`
)

var i18nFa = map[string]string{
	"payments_unavailable":  "⚠️ در حال حاضر امکان پرداخت آنلاین و کارت به کارت فعال نیست. لطفاً بعداً تلاش کنید.",
	"pending_limit_exceeded": "⚠️ شما دارای حداکثر (۳) درخواست شارژ در انتظار بررسی هستید. لطفاً تا تعیین تکلیف درخواست‌های قبلی صبور باشید.",
	"already_reviewed":      "ℹ️ این درخواست قبلاً توسط مدیریت یا از طریق پنل وب بررسی شده است.",
	"topup_card_info": `💳 *افزایش موجودی کیف پول*

لطفاً مبلغ مورد نظر خود را به کارت زیر واریز نمایید:

• *شماره کارت:* %s
• *به نام:* %s
• *بانک:* %s

*راهنما:* %s

*نکته:* پس از واریز وجه، لطفاً مبلغ واریز شده را *به %s* وارد کنید (مثال: ۵۰۰۰۰):`,
	"staff_menu":                "👥 *مدیریت همکاران (Staff)*\n\nهمکاران می‌توانند درخواست‌های شارژ را مشاهده، تأیید یا رد کنند و به پیام‌های پشتیبانی پاسخ دهند.",
	"staff_list_header":         "📋 *لیست همکاران فعال:*",
	"awaiting_staff_add":        "➕ لطفاً یک پیام از کاربر مورد نظر را به اینجای چت فوروارد کنید یا شناسه عددی تلگرام کاربر را ارسال نمایید:",
	"staff_added_success":       "✅ همکار جدید با موفقیت اضافه شد.",
	"staff_removed_success":     "✅ همکار با موفقیت حذف شد.",
	"staff_invalid_id":          "⚠️ شناسه عددی تلگرام نامعتبر است.",
	"staff_owner_only":          "⛔ این بخش فقط برای مالک اصلی ربات قابل دسترسی است.",
	"channel_gate_required":     "📢 *عضویت در کانال اجباری*\n\nبرای استفاده از امکانات ربات، لطفاً ابتدا در کانال زیر عضو شوید و سپس دکمه '✅ عضو شدم' را بزنید:\n\n%s",
	"channel_gate_settings":     "📢 *تنظیمات کانال اجباری*\n\nکانال فعلی: %s\n\nبرای تغییر آیدی کانال را ارسال کنید (مثال: @mychannel) یا برای غیرفعال‌سازی عبارت 'off' را بفرستید:",
	"channel_updated_success":   "✅ کانال اجباری با موفقیت تنظیم شد.",
	"channel_cleared_success":   "✅ کانال اجباری غیرفعال شد.",
	"channel_warning_not_admin": "⚠️ هشدار: ربات در کانال %s دسترسی مدیریت ندارد یا کانال یافت نشد.",
	"qr_toggled_on":             "📱 ارسال QR کد فعال شد.",
	"qr_toggled_off":            "📱 ارسال QR کد غیرفعال شد.",
	"reminders_toggled_on":      "⏰ یادآور انقضا فعال شد.",
	"reminders_toggled_off":     "⏰ یادآور انقضا غیرفعال شد.",
	"reject_reason_prompt":      "❌ لطفاً علت رد درخواست را وارد کنید (علت الزامی است):",
	"reject_reason_required":    "⚠️ وارد کردن علت رد درخواست الزامی است. لطفاً علت را وارد کنید:",
	"expiry_reminder_3days":     "⏳ *یادآور انقضای اشتراک*\n\nکاربر گرامی، اشتراک شما (*%s*) تا ۳ روز دیگر منقضی می‌شود. جهت تمدید اقدام کنید.",
	"expiry_reminder_1day":      "⚠️ *هشدار انقضای اشتراک*\n\nکاربر گرامی، اشتراک شما (*%s*) تا ۲۴ ساعت دیگر منقضی می‌شود. جهت جلوگیری از قطع سرویس نسبت به تمدید اقدام نمایید.",
	"balance_credited":          "✅ *افزایش موجودی*\n\nدرخواست شارژ شما به مبلغ *%s* تأیید شد و موجودی حساب شما افزایش یافت.",
	"recharge_rejected_user":    "❌ *رد درخواست شارژ*\n\nدرخواست شارژ شما به مبلغ *%s* رد شد.\nعلت رد: %s",
	"card_number_invalid":       "⚠️ شماره کارت باید بین ۱۳ تا ۱۹ رقم باشد. لطفاً مجدداً وارد کنید:",
	"select_node_prompt":        "📥 لطفاً سرور مورد نظر برای دریافت فایل تنظیمات را انتخاب کنید:",
	"no_nodes_found":            "⚠️ سروری با این مشخصات یا پروتکل یافت نشد.",
	"download_profile_error":    "❌ خطا در دریافت فایل تنظیمات از سرور.",
	"ask_sub_custom_name":       "✏️ لطفاً یک نام برای این اشتراک وارد کنید (یا - برای نام پیش‌فرض):",
	"remaining_traffic":         "%s باقی‌مانده",
	"purchase_failed_try_again": "❌ خرید با خطا مواجه شد. لطفاً مجدداً تلاش کنید.",
	"purchase_success_fallback": "خرید انجام شد ✅ — لینک اشتراک را از بخش «اشتراک‌های من» دریافت کنید.",
	"subscriptions_list_title": "🔑 *اشتراک‌های شما (%d اشتراک):*",
	"no_subscriptions_text":   "📭 شما هنوز هیچ اشتراکی ندارید.",
	"buy_service_btn":         "🛒 خرید سرویس",
	"sub_status_expired":      "منقضی شده",
	"back_to_list":            "🔙 بازگشت به لیست",
}

var i18nEn = map[string]string{
	"payments_unavailable":  "⚠️ Online payment is currently unavailable. Please try again later.",
	"pending_limit_exceeded": "⚠️ You have reached the maximum limit of (3) pending recharge requests. Please wait for previous requests to be reviewed.",
	"already_reviewed":      "ℹ️ This request has already been reviewed (via web or another admin).",
	"topup_card_info": `💳 *Top-Up Wallet Balance*

Please transfer your desired amount to the following card:

• *Card Number:* %s
• *Account Owner:* %s

*Note:* After payment, enter the deposited amount in Toman (e.g. 50000):`,
	"staff_menu":                "👥 *Staff Management*\n\nStaff admins can review, approve, or reject top-up requests and reply to support messages.",
	"staff_list_header":         "📋 *Active Staff Members:*",
	"awaiting_staff_add":        "➕ Forward a message from the user or send their numerical Telegram User ID:",
	"staff_added_success":       "✅ New staff member added successfully.",
	"staff_removed_success":     "✅ Staff member removed successfully.",
	"staff_invalid_id":          "⚠️ Invalid Telegram User ID.",
	"staff_owner_only":          "⛔ This feature is restricted to the bot owner only.",
	"channel_gate_required":     "📢 *Channel Membership Required*\n\nTo use the bot, please join our channel first and click '✅ I joined':\n\n%s",
	"channel_gate_settings":     "📢 *Required Channel Settings*\n\nCurrent channel: %s\n\nSend a channel username (e.g. @mychannel) to update, or 'off' to disable:",
	"channel_updated_success":   "✅ Required channel updated successfully.",
	"channel_cleared_success":   "✅ Required channel disabled.",
	"channel_warning_not_admin": "⚠️ Warning: The bot is not an admin in channel %s or channel was not found.",
	"qr_toggled_on":             "📱 QR code generation turned ON.",
	"qr_toggled_off":            "📱 QR code generation turned OFF.",
	"reminders_toggled_on":      "⏰ Expiry reminders turned ON.",
	"reminders_toggled_off":     "⏰ Expiry reminders turned OFF.",
	"reject_reason_prompt":      "❌ Please provide a rejection reason (reason is required):",
	"reject_reason_required":    "⚠️ Rejection reason is required. Please enter a valid reason:",
	"expiry_reminder_3days":     "⏳ *Subscription Expiry Reminder*\n\nDear user, your subscription (*%s*) will expire in 3 days. Please renew to keep your service active.",
	"expiry_reminder_1day":      "⚠️ *Subscription Expiry Warning*\n\nDear user, your subscription (*%s*) will expire in 24 hours. Please renew now to avoid service interruption.",
	"balance_credited":          "✅ *Balance Credited*\n\nYour recharge request of *%s* has been approved and credited to your wallet.",
	"recharge_rejected_user":    "❌ *Recharge Rejected*\n\nYour recharge request of *%s* was rejected.\nReason: %s",
	"card_number_invalid":       "⚠️ Card number must be between 13 and 19 digits. Please try again:",
	"select_node_prompt":        "📥 Please select the server to download the configuration file:",
	"no_nodes_found":            "⚠️ No servers found for this protocol.",
	"download_profile_error":    "❌ Error downloading configuration file from server.",
	"ask_sub_custom_name":       "✏️ لطفاً یک نام برای این اشتراک وارد کنید (یا - برای نام پیش‌فرض):",
	"remaining_traffic":         "%s باقی‌مانده",
	"purchase_failed_try_again": "❌ خرید با خطا مواجه شد. لطفاً مجدداً تلاش کنید.",
	"purchase_success_fallback": "خرید انجام شد ✅ — لینک اشتراک را از بخش «اشتراک‌های من» دریافت کنید.",
	"subscriptions_list_title": "🔑 *اشتراک‌های شما (%d اشتراک):*",
	"no_subscriptions_text":   "📭 شما هنوز هیچ اشتراکی ندارید.",
	"buy_service_btn":         "🛒 خرید سرویس",
	"sub_status_expired":      "منقضی شده",
	"back_to_list":            "🔙 بازگشت به لیست",
}

func getLang(from *tgbotapi.User) string {
	if from != nil && from.LanguageCode != "" {
		if strings.HasPrefix(strings.ToLower(from.LanguageCode), "en") {
			return "en"
		}
	}
	return "fa"
}

func Tr(lang string, key string, args ...interface{}) string {
	dict := i18nFa
	if strings.HasPrefix(strings.ToLower(lang), "en") {
		dict = i18nEn
	}
	tpl, exists := dict[key]
	if !exists {
		tpl = key
	}
	if len(args) > 0 {
		return fmt.Sprintf(tpl, args...)
	}
	return tpl
}

func FormatTraffic(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func escapeMarkdown(s string) string {
	replacer := strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"`", "\\`",
		"[", "\\[",
	)
	return replacer.Replace(s)
}

