package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
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

	if v != nil {
		if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
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
