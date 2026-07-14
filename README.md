# Standalone Reseller Telegram Bot (Golang)

This is a self-hosted, standalone Telegram bot for resellers, built in Go. It integrates with the core backend API using Reseller API endpoints (`Authorization: Bearer rn_...`).

## Features

1.  **Direct Register/Login**: Auto-registers users based on their Telegram ID (`POST /v1/reseller/user`).
2.  **Wallet Top-Up (Card-to-Card)**:
    *   Guides users to pay by displaying the reseller's card number and cardholder name.
    *   Instructs users to enter the payment amount and upload the transaction screenshot/receipt.
    *   Automatically forwards payment screenshots to reseller admins (specified chat IDs) with dynamic Approve/Reject inline buttons.
    *   Updates user wallet balances on approval via `PUT /v1/reseller/user/balance`.
3.  **Buy Subscription**:
    *   Displays configured plans inline.
    *   Validates user wallet balance.
    *   Calls backend `POST /v1/reseller/user/subscribe` to provision the VPN subscription.
    *   Deducts subscription price from user wallet.
    *   Returns connection configs (VLESS, Trojan, etc.) immediately to the user.
4.  **My Subscriptions**:
    *   Lists active subscriptions, including traffic quotas, remaining traffic, and expiration dates.
    *   Displays formatted configuration lines in markdown copy-paste code blocks.
5.  **Persistency**:
    *   Uses a pure Go SQLite engine (`modernc.org/sqlite`) for zero-dependency database storage of users and recharge requests, protecting against state losses during reboots.
6.  **Persian Language Interface**: Fully localized in Persian (Farsi) out-of-the-box.

---

## Folder Structure

```
.
├── cmd/
│   └── bot/
│       └── main.go           # Application entry point & graceful shutdown
├── pkg/
│   ├── config/
│   │   └── config.go         # Parses yaml application configurations
│   ├── backend/
│   │   ├── client.go         # HTTP Reseller API Client
│   │   └── types.go          # Mapped API request/response structures
│   ├── db/
│   │   └── sqlite.go         # SQLite user and transaction persistence
│   ├── bot/
│   │   ├── bot.go            # Updates loop and dispatcher
│   │   ├── state.go          # User wizard interaction sessions
│   │   ├── keyboard.go       # Keyboard buttons and inline layouts
│   │   ├── handlers.go       # Text messages and state actions
│   │   ├── callbacks.go      # Inline button actions (approval/rejection/plans)
│   │   └── i18n.go           # String dict mapping in Persian
└── config.yaml.template      # Settings templates
```

---

## Configuration (`config.yaml`)

Create a `config.yaml` file in the root directory:

```yaml
bot_token: "YOUR_TELEGRAM_BOT_TOKEN"
backend_url: "https://your-core-backend.com" # Core backend url
reseller_api_key: "rn_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" # Reseller private API Key
admin_chat_ids:
  - 123456789 # Telegram ID of managers who approve receipts
card_number: "6037-9911-2233-4455" # Card details for wallet topup
card_owner: "مدیریت سیستم"

plans:
  - id: 1
    subscribe_id: 12
    name: "🚀 پلان برنزی - ۵۰ گیگابایت"
    price: 150000 # Price in Tomans
    description: "حجم ۵۰ گیگابایت - اعتبار ۱ ماه"
```

---

## Building and Running

### Prerequisites
*   Go 1.20 or newer

### Setup
1.  Clone this repository or move to its workspace path.
2.  Install dependencies:
    ```bash
    go mod tidy
    ```
3.  Copy and customize configuration:
    ```bash
    cp config.yaml.template config.yaml
    ```
4.  Run unit tests:
    ```bash
    go test -v ./...
    ```
5.  Start the bot:
    ```bash
    go run cmd/bot/main.go
    ```
