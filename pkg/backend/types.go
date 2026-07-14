package backend

type UserRegisterRequest struct {
	TelegramID   int64  `json:"telegram_id"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Username     string `json:"username"`
	LanguageCode string `json:"language_code"`
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
	UserID      int64 `json:"user_id"`
	SubscribeID int   `json:"subscribe_id"`
}

type SubscribeResponse struct {
	UUID string `json:"uuid"`
}

type SubscriptionItem struct {
	ID           int64    `json:"id"`
	UUID         string   `json:"uuid"`
	Name         string   `json:"name"`
	Configs      []string `json:"configs"`
	TotalTraffic int64    `json:"total_traffic"`
	UsedTraffic  int64    `json:"used_traffic"`
	ExpireTime   int64    `json:"expire_time"`
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
	Show                   bool               `json:"show"`
	Sell                   bool               `json:"sell"`
}

type DeleteResellerSubscribeRequest struct {
	ID int64 `json:"id"`
}
