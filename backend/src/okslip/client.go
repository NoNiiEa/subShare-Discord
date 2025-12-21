package okslip

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type OkSlipClient interface {
	CheckSlip(ctx context.Context, slipUrl string) (*Response, error)
}

type okSlipClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient(baseURL string, apiKey string) OkSlipClient {
	return &okSlipClient{
		baseURL: baseURL,
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *okSlipClient) CheckSlip(ctx context.Context, slipUrl string) (*Response, error) {
	reqBody := CheckSlipRequest{
		Url: slipUrl,
		Log: false,
	}

	jsonBytes, err := json.Marshal(reqBody)
    if err != nil {
        return nil, err
    }

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewBuffer(jsonBytes))
    if err != nil {
        return nil, err
    }

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-authorization", c.apiKey)

	resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

	if resp.StatusCode != http.StatusOK {
        return nil, ErrInternal
    }

	var result Response
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
        return nil, fmt.Errorf("failed to parse response: %v", err)
    }

	return &result, nil
}