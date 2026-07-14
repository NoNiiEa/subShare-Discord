package okslip

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

type APIErrorResponse struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
}

func (e *APIErrorResponse) Error() string {
    return fmt.Sprintf("OkSlip Error [%d]: %s", e.Code, e.Message)
}

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
        if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
            return nil, fmt.Errorf("%w: %v", ErrTimeout, err)
        }
        if errors.Is(err, context.DeadlineExceeded) {
            return nil, fmt.Errorf("%w: %v", ErrTimeout, err)
        }
        return nil, err
    }
    defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

	var result Response
    if err := json.Unmarshal(bodyBytes, &result); err == nil {
        // If the API returns 'success: false' but a 200 OK
        if !result.Success {
            // Check if there is an error code in the body
            var apiErr APIErrorResponse
            if json.Unmarshal(bodyBytes, &apiErr) == nil && apiErr.Code != 0 {
                return nil, &apiErr
            }
        }
        
        if resp.StatusCode == http.StatusOK {
            return &result, nil
        }
    }

    // If we reach here, it's a non-200 response or a failed success flag
    var apiErr APIErrorResponse
    if err := json.Unmarshal(bodyBytes, &apiErr); err == nil && apiErr.Code != 0 {
        return nil, &apiErr
    }

    // Fallback for generic HTTP errors
    return nil, fmt.Errorf("request failed status %d: %s", resp.StatusCode, string(bodyBytes))
}