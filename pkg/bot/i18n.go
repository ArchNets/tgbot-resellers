package bot

import (
	"fmt"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// Use localized printing for numbers (e.g. 50,000 instead of 50000)
var printer = message.NewPrinter(language.Persian)

func FormatMoney(amount int64) string {
	return printer.Sprintf("%d", amount)
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
• موجودی کیف پول: *%s* تومان`

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

• موجودی فعلی شما: *%s* تومان

آیا از خرید این پلان اطمینان دارید؟`

	MsgInsufficientBalance = `❌ موجودی کیف پول شما کافی نیست!
• موجودی فعلی: *%s* تومان
• قیمت پلان: *%s* تومان

لطفاً ابتدا کیف پول خود را افزایش موجودی دهید.`

	MsgPurchaseSuccess = `✅ خرید شما با موفقیت انجام شد!
کانفیگ‌های شما در زیر آماده استفاده هستند.`

	MsgNoSubscriptions = `📭 شما در حال حاضر هیچ اشتراک فعالی ندارید.`

	MsgSubscriptionsList = `🔑 *لیست اشتراک‌های شما:*`

	MsgSubDetail = `📦 *اشتراک: %s*
• شناسه: %s
• حجم کل: %s
• حجم مصرفی: %s
• تاریخ انقضا: %s

*کانفیگ‌های اتصال:*
`

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
	BtnAdminPanel        = "⚙️ پنل مدیریت"
	BtnAdminCardSettings = "💳 تنظیمات کارت بانکی"
	BtnAdminPlansSettings = "📦 مدیریت پلان‌ها"
	BtnAdminTagSettings   = "🏷️ مدیریت نام دسته‌ها"

	// Admin Messages
	MsgAdminPanelWelcome = `⚙️ *به پنل مدیریت ربات خوش آمدید.*

لطفاً یک گزینه را انتخاب کنید:`

	MsgAdminAwaitingCardNumber = `💳 لطفاً شماره کارت بانکی جدید را وارد کنید:
(مثال: ۶۰۳۷-۹۹۱۱-۲۲۳۳-۴۴۵۵)`

	MsgAdminAwaitingCardOwner = `👤 شماره کارت ثبت شد.
حالا لطفاً نام صاحب کارت جدید را وارد کنید:`

	MsgAdminCardUpdated = `✅ اطلاعات کارت بانکی با موفقیت در دیتابیس ربات بروزرسانی شد.`

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
