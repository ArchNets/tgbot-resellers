package backend

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

type APIError struct {
	Code int
	Msg  string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("backend error code %d: %s", e.Code, e.Msg)
}

func IsErrorCode(err error, code int) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.Code == code
	}
	return false
}

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient(baseURL, apiKey string, hostMappings map[string]string, insecureSkipVerify bool) *Client {
	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err == nil && hostMappings != nil {
				if overrideIP, exists := hostMappings[host]; exists && overrideIP != "" {
					addr = net.JoinHostPort(overrideIP, port)
				}
			}
			return dialer.DialContext(ctx, network, addr)
		},
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecureSkipVerify,
		},
	}

	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout:   15 * time.Second,
			Transport: transport,
		},
	}
}

func (c *Client) newRequest(ctx context.Context, method, path string, body interface{}) (*http.Request, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	url := fmt.Sprintf("%s%s", c.baseURL, path)
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

type APIResponse struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

func (c *Client) do(req *http.Request, v interface{}) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("backend returned error status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("failed to decode response envelope: %w", err)
	}

	if apiResp.Code != 200 && apiResp.Code != 0 {
		return &APIError{Code: apiResp.Code, Msg: apiResp.Msg}
	}

	if v != nil && len(apiResp.Data) > 0 {
		if err := json.Unmarshal(apiResp.Data, v); err != nil {
			return fmt.Errorf("failed to decode response data: %w", err)
		}
	}

	return nil
}

func (c *Client) RegisterUser(ctx context.Context, req *UserRegisterRequest) (*UserRegisterResponse, error) {
	httpReq, err := c.newRequest(ctx, "POST", "/v1/reseller/user", req)
	if err != nil {
		return nil, err
	}
	var resp UserRegisterResponse
	if err := c.do(httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) UpdateUserBalance(ctx context.Context, req *BalanceUpdateRequest) error {
	httpReq, err := c.newRequest(ctx, "PUT", "/v1/reseller/user/balance", req)
	if err != nil {
		return err
	}
	return c.do(httpReq, nil)
}

func (c *Client) GetUserBalanceLogs(ctx context.Context, userID int64, page, size int) (*BalanceLogsResponse, error) {
	path := fmt.Sprintf("/v1/reseller/user/balance/logs?user_id=%d&page=%d&size=%d", userID, page, size)
	httpReq, err := c.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	var resp BalanceLogsResponse
	if err := c.do(httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) CreateSubscription(ctx context.Context, req *SubscribeRequest) (*SubscribeResponse, error) {
	httpReq, err := c.newRequest(ctx, "POST", "/v1/reseller/user/subscribe", req)
	if err != nil {
		return nil, err
	}
	var resp SubscribeResponse
	if err := c.do(httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetUserSubscriptions(ctx context.Context, userID int64, page, size int) (*SubscriptionListResponse, error) {
	path := fmt.Sprintf("/v1/reseller/user/subscribe?user_id=%d&page=%d&size=%d", userID, page, size)
	httpReq, err := c.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	var resp SubscriptionListResponse
	if err := c.do(httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetResellerSubscribeList(ctx context.Context, page, size int) (*GetResellerSubscribeListResponse, error) {
	path := fmt.Sprintf("/v1/reseller/subscribe/list?page=%d&size=%d", page, size)
	httpReq, err := c.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	var resp GetResellerSubscribeListResponse
	if err := c.do(httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) CreateResellerSubscribe(ctx context.Context, req *CreateResellerSubscribeRequest) error {
	httpReq, err := c.newRequest(ctx, "POST", "/v1/reseller/subscribe", req)
	if err != nil {
		return err
	}
	return c.do(httpReq, nil)
}

func (c *Client) UpdateResellerSubscribe(ctx context.Context, req *UpdateResellerSubscribeRequest) error {
	httpReq, err := c.newRequest(ctx, "PUT", "/v1/reseller/subscribe", req)
	if err != nil {
		return err
	}
	return c.do(httpReq, nil)
}

func (c *Client) DeleteResellerSubscribe(ctx context.Context, req *DeleteResellerSubscribeRequest) error {
	httpReq, err := c.newRequest(ctx, "DELETE", "/v1/reseller/subscribe", req)
	if err != nil {
		return err
	}
	return c.do(httpReq, nil)
}

func (c *Client) GetDownloadNodes(ctx context.Context, subscribeID int64) ([]DownloadNode, error) {
	path := fmt.Sprintf("/v1/reseller/user_subscribe/download_nodes?id=%d", subscribeID)
	httpReq, err := c.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	var resp DownloadNodesResponse
	if err := c.do(httpReq, &resp); err != nil {
		return nil, err
	}
	return resp.List, nil
}

func (c *Client) GetUserSubscribeProfile(ctx context.Context, subscribeID, nodeID int64, format string) (*Profile, error) {
	path := fmt.Sprintf("/v1/reseller/user_subscribe/profile?id=%d&node_id=%d&format=%s", subscribeID, nodeID, url.QueryEscape(format))
	httpReq, err := c.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	var resp ProfileResponse
	if err := c.do(httpReq, &resp); err != nil {
		return nil, err
	}

	decoded, err := base64.StdEncoding.DecodeString(resp.Content)
	if err != nil {
		decoded, err = base64.URLEncoding.DecodeString(resp.Content)
		if err != nil {
			return nil, fmt.Errorf("failed to base64 decode profile content: %w", err)
		}
	}

	return &Profile{
		Filename:    resp.Filename,
		ContentType: resp.ContentType,
		Content:     decoded,
	}, nil
}

func (c *Client) GetPaymentCard(ctx context.Context) (*PaymentCard, error) {
	httpReq, err := c.newRequest(ctx, "GET", "/v1/reseller/payment/card", nil)
	if err != nil {
		return nil, err
	}
	var resp PaymentCard
	if err := c.do(httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) UpsertPaymentCard(ctx context.Context, card *PaymentCard) error {
	httpReq, err := c.newRequest(ctx, "PUT", "/v1/reseller/payment/card", card)
	if err != nil {
		return err
	}
	return c.do(httpReq, nil)
}

func (c *Client) CreateRecharge(ctx context.Context, req *CreateRechargeRequest) (*RechargeOrder, error) {
	httpReq, err := c.newRequest(ctx, "POST", "/v1/reseller/recharge", req)
	if err != nil {
		return nil, err
	}
	var resp RechargeOrder
	if err := c.do(httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetRechargeList(ctx context.Context, tier, status string, page, size int) (*RechargeListResponse, error) {
	path := fmt.Sprintf("/v1/reseller/recharge/list?tier=%s&status=%s&page=%d&size=%d",
		url.QueryEscape(tier), url.QueryEscape(status), page, size)
	httpReq, err := c.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	var resp RechargeListResponse
	if err := c.do(httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetRechargeReceipt(ctx context.Context, id int64) (*RechargeReceiptResponse, error) {
	path := fmt.Sprintf("/v1/reseller/recharge/receipt?id=%d", id)
	httpReq, err := c.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	var resp RechargeReceiptResponse
	if err := c.do(httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) ReviewRecharge(ctx context.Context, req *RechargeReviewRequest) error {
	httpReq, err := c.newRequest(ctx, "PUT", "/v1/reseller/recharge/review", req)
	if err != nil {
		return err
	}
	return c.do(httpReq, nil)
}

func (c *Client) GetExchangeRate(ctx context.Context) (*GetResellerExchangeRateResponse, error) {
	httpReq, err := c.newRequest(ctx, "GET", "/v1/reseller/exchange_rate", nil)
	if err != nil {
		return nil, err
	}
	var resp GetResellerExchangeRateResponse
	if err := c.do(httpReq, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetBaseURL() string {
	return c.baseURL
}

func (c *Client) GetSiteConfig(ctx context.Context) (*SiteConfigData, error) {
	httpReq, err := c.newRequest(ctx, "GET", "/v1/common/site/config", nil)
	if err != nil {
		return nil, err
	}
	var data SiteConfigData
	if err := c.do(httpReq, &data); err != nil {
		return nil, err
	}
	return &data, nil
}


