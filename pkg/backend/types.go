package backend

type UserRegisterRequest struct {
	TelegramID   int64  `json:"telegram_id"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Username     string `json:"username"`
	LanguageCode string `json:"language_code"`
	BotID        int64  `json:"bot_id,omitempty"`
}

type UserRegisterResponse struct {
	UserID     int64  `json:"user_id"`
	Balance    int64  `json:"balance"`
	Lang       string `json:"lang"`
	CreatedNew bool   `json:"created_new"`
}

type BalanceUpdateRequest struct {
	UserID int64  `json:"user_id"`
	Amount int64  `json:"amount"`
	Reason string `json:"reason"`
}

type BalanceLog struct {
	UserID    int64 `json:"user_id"`
	Amount    int64 `json:"amount"`
	Type      int   `json:"type"`
	Balance   int64 `json:"balance"`
	Timestamp int64 `json:"timestamp"`
}

type BalanceLogsResponse struct {
	Total int          `json:"total"`
	List  []BalanceLog `json:"list"`
}

type SubscribeRequest struct {
	UserID            int64  `json:"user_id"`
	SubscribeID       int    `json:"subscribe_id"`
	CustomName        string `json:"custom_name,omitempty"`
	ExpiredAt         int64  `json:"expired_at,omitempty"`
	ChargeFromBalance bool   `json:"charge_from_balance"`
}

type SubscribeResponse struct {
	UserSubscribeID int64 `json:"user_subscribe_id"`
}

type SubscriptionPlan struct {
	Name        string   `json:"name"`
	DeviceLimit int      `json:"device_limit"`
	SpeedLimit  int64    `json:"speed_limit"`
	NodeTags    []string `json:"node_tags"`
}

type SubscriptionItem struct {
	ID          int64            `json:"id"`
	SubscribeID int64            `json:"subscribe_id"`
	CustomName  string           `json:"custom_name"`
	Token       string           `json:"token"`
	Short       string           `json:"short"`
	Status      int              `json:"status"`
	Traffic     int64            `json:"traffic"`
	Upload      int64            `json:"upload"`
	Download    int64            `json:"download"`
	StartTime   int64            `json:"start_time"`
	ExpireTime  int64            `json:"expire_time"`
	OnlineCount int              `json:"online_count"`
	Subscribe   SubscriptionPlan `json:"subscribe"`
}

func (s *SubscriptionItem) GetName() string {
	if s.CustomName != "" {
		return s.CustomName
	}
	return s.Subscribe.Name
}

type SubscriptionListResponse struct {
	Total int                `json:"total"`
	List  []SubscriptionItem `json:"list"`
}

type ResellerDiscount struct {
	Quantity int   `json:"quantity"`
	Discount float64 `json:"discount"`
}

type ResellerSubscribePlan struct {
	ID                     int64              `json:"id"`
	Name                   string             `json:"name"`
	Language               string             `json:"language"`
	Description            string             `json:"description"`
	UnitPrice              int64              `json:"unit_price"`
	UnitTime               string             `json:"unit_time"`
	Discount               []ResellerDiscount `json:"discount"`
	Traffic                int64              `json:"traffic"`
	SpeedLimit             int64              `json:"speed_limit"`
	DeviceLimit            int64              `json:"device_limit"`
	Nodes                  []int64            `json:"nodes"`
	NodeTags               []string           `json:"node_tags"`
	Show                   bool               `json:"show"`
	Sell                   bool               `json:"sell"`
	Sort                   int                `json:"sort"`
	ResellerSubscriptionID int64              `json:"reseller_subscription_id"`
	BotID                  int64              `json:"bot_id,omitempty"`
	CreatedTime            int64              `json:"created_at"`
	UpdatedTime            int64              `json:"updated_at"`
}

type GetResellerSubscribeListResponse struct {
	Total int                     `json:"total"`
	List  []ResellerSubscribePlan `json:"list"`
}

type CreateResellerSubscribeRequest struct {
	Name                   string             `json:"name"`
	Language               string             `json:"language,omitempty"`
	Description            string             `json:"description,omitempty"`
	UnitPrice              int64              `json:"unit_price"`
	UnitTime               string             `json:"unit_time"`
	Discount               []ResellerDiscount `json:"discount,omitempty"`
	Traffic                int64              `json:"traffic"`
	SpeedLimit             int64              `json:"speed_limit"`
	DeviceLimit            int64              `json:"device_limit"`
	Nodes                  []int64            `json:"nodes,omitempty"`
	ResellerSubscriptionID int64              `json:"reseller_subscription_id"`
	BotID                  int64              `json:"bot_id,omitempty"`
	Show                   bool               `json:"show"`
	Sell                   bool               `json:"sell"`
}

type UpdateResellerSubscribeRequest struct {
	ID                     int64              `json:"id"`
	Name                   string             `json:"name"`
	Language               string             `json:"language,omitempty"`
	Description            string             `json:"description,omitempty"`
	UnitPrice              int64              `json:"unit_price"`
	UnitTime               string             `json:"unit_time"`
	Discount               []ResellerDiscount `json:"discount,omitempty"`
	Traffic                int64              `json:"traffic"`
	SpeedLimit             int64              `json:"speed_limit"`
	DeviceLimit            int64              `json:"device_limit"`
	Nodes                  []int64            `json:"nodes,omitempty"`
	ResellerSubscriptionID int64              `json:"reseller_subscription_id"`
	BotID                  int64              `json:"bot_id,omitempty"`
	Show                   bool               `json:"show"`
	Sell                   bool               `json:"sell"`
}

type DeleteResellerSubscribeRequest struct {
	ID int64 `json:"id"`
}

type AdminPaymentCard struct {
	CardNumber   string `json:"card_number"`
	CardOwner    string `json:"card_owner"`
	BankName     string `json:"bank_name"`
	Instructions string `json:"instructions"`
}

type ResellerPaymentCard struct {
	CardNumber   string `json:"card_number"`
	CardOwner    string `json:"card_owner"`
	BankName     string `json:"bank_name"`
	Enabled      bool   `json:"enabled"`
	Instructions string `json:"instructions"`
}

type DownloadNode struct {
	NodeID   int64    `json:"node_id"`
	Name     string   `json:"name"`
	Protocol string   `json:"protocol"`
	Formats  []string `json:"formats"`
}

type DownloadNodesResponse struct {
	List []DownloadNode `json:"list"`
}

type ProfileResponse struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Content     string `json:"content"`
}

type Profile struct {
	Filename    string
	ContentType string
	Content     []byte
}

type PaymentCard struct {
	CardNumber   string `json:"card_number"`
	CardOwner    string `json:"card_owner"`
	BankName     string `json:"bank_name"`
	Enabled      bool   `json:"enabled"`
	Instructions string `json:"instructions"`
}

type CreateRechargeRequest struct {
	Tier           string `json:"tier"`
	CustomerUserID int64  `json:"customer_user_id"`
	BotID          int64  `json:"bot_id,omitempty"`
	Amount         int64  `json:"amount"`
	Currency       string `json:"currency"`
	ReceiptType    string `json:"receipt_type"`
	ReceiptData    string `json:"receipt_data"`
	Note           string `json:"note,omitempty"`
	Source         string `json:"source"`
}

type RechargeOrder struct {
	ID               int64  `json:"id"`
	OrderNo          string `json:"order_no"`
	Tier             string `json:"tier"`
	CustomerUserID   int64  `json:"customer_user_id"`
	BotID            int64  `json:"bot_id,omitempty"`
	AmountUSDCents   int64  `json:"amount_usd_cents"`
	OriginalAmount   int64  `json:"original_amount"`
	OriginalCurrency string `json:"original_currency"`
	Status           string `json:"status"`
	Note             string `json:"note"`
	Source           string `json:"source"`
	ReviewedAt       int64  `json:"reviewed_at"`
	RejectReason     string `json:"reject_reason"`
	CreatedAt        int64  `json:"created_at"`
	UpdatedAt        int64  `json:"updated_at"`
}

type RechargeListResponse struct {
	List  []RechargeOrder `json:"list"`
	Total int             `json:"total"`
}

type RechargeReceiptResponse struct {
	ReceiptType string `json:"receipt_type"`
	ReceiptData string `json:"receipt_data"`
}

type RechargeReviewRequest struct {
	ID     int64  `json:"id"`
	Action string `json:"action"`
	Reason string `json:"reason,omitempty"`
}

type GetResellerExchangeRateResponse struct {
	UsdToIrt float64 `json:"usd_to_irt"`
}

type SiteConfigData struct {
	Subscribe struct {
		SubscribeDomain string `json:"subscribe_domain"`
		SubscribePath   string `json:"subscribe_path"`
		PanDomain       bool   `json:"pan_domain"`
	} `json:"subscribe"`
	Site struct {
		Host string `json:"host"`
	} `json:"site"`
}


