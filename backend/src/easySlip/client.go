package easyslip

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type EasySlipClient interface {
	CheckSlip(ctx context.Context, slipUrl string) (*EasySlipResponse, error)
}

type easySlipClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient(baseURL, apiKey string) EasySlipClient {
	return &easySlipClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *easySlipClient) CheckSlip(ctx context.Context, slipUrl string) (*EasySlipResponse, error) {
    reqBody := CheckSlipRequest{
        Url: slipUrl,
        // CheckDuplicate: true,
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
    req.Header.Set("Authorization", "Bearer "+c.apiKey)

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
        if resp.StatusCode == 400 {
            return nil, ErrDupeSlip
        }

        return nil, ErrInternal
    }

    var result EasySlipResponse
    if err := json.Unmarshal(bodyBytes, &result); err != nil {
        return nil, fmt.Errorf("failed to parse response: %v", err)
    }

    return &result, nil
}