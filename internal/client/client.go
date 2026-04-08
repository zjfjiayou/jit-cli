package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Request struct {
	Server   string
	App      string
	Endpoint string
	Token    string
	Body     any
	Method   string
	DryRun   bool
}

type DryRunRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    any               `json:"body"`
}

type Response struct {
	StatusCode int             `json:"statusCode"`
	RawBody    json.RawMessage `json:"rawBody"`
	JSON       any             `json:"json"`
	ErrCode    int             `json:"errCode"`
	HasErrCode bool            `json:"hasErrCode"`
}

func (r *Response) IsBusinessError() bool {
	return r != nil && r.HasErrCode && r.ErrCode != 0
}

type Result struct {
	DryRun   *DryRunRequest `json:"dryRun,omitempty"`
	Response *Response      `json:"response,omitempty"`
}

type Client struct {
	httpClient *http.Client
}

func New(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{httpClient: httpClient}
}

func (c *Client) Call(ctx context.Context, req Request) (*Result, error) {
	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = http.MethodPost
	}
	urlValue, err := BuildURL(req.Server, req.App, req.Endpoint)
	if err != nil {
		return nil, err
	}
	bodyValue, bodyBytes, err := normalizeBody(req.Body)
	if err != nil {
		return nil, err
	}

	if req.DryRun {
		headers := requestHeaders(req.Token)
		return &Result{
			DryRun: &DryRunRequest{
				Method:  method,
				URL:     urlValue,
				Headers: headers,
				Body:    bodyValue,
			},
		}, nil
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, urlValue, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	for key, value := range requestHeaders(req.Token) {
		httpReq.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var payload any
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		return nil, fmt.Errorf(
			"decode response json (status %d): %w; response preview: %q",
			resp.StatusCode,
			err,
			responsePreview(rawBody),
		)
	}

	errCode, hasErrCode := extractErrCode(payload)
	return &Result{
		Response: &Response{
			StatusCode: resp.StatusCode,
			RawBody:    append(json.RawMessage(nil), rawBody...),
			JSON:       payload,
			ErrCode:    errCode,
			HasErrCode: hasErrCode,
		},
	}, nil
}

func normalizeBody(body any) (any, []byte, error) {
	if body == nil {
		body = map[string]any{}
	}

	switch typed := body.(type) {
	case json.RawMessage:
		return decodeBodyBytes([]byte(typed))
	case []byte:
		return decodeBodyBytes(typed)
	case string:
		return decodeBodyBytes([]byte(typed))
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal request body: %w", err)
	}
	decoded, err := decodeJSONBytes(bodyBytes, "decode request body")
	if err != nil {
		return nil, nil, err
	}
	return decoded, bodyBytes, nil
}

func decodeBodyBytes(raw []byte) (any, []byte, error) {
	decoded, err := decodeJSONBytes(raw, "invalid request body")
	if err != nil {
		return nil, nil, err
	}
	return decoded, raw, nil
}

func decodeJSONBytes(raw []byte, message string) (any, error) {
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, fmt.Errorf("%s: %w", message, err)
	}
	return decoded, nil
}

func extractErrCode(payload any) (int, bool) {
	payloadMap, ok := payload.(map[string]any)
	if !ok {
		return 0, false
	}
	value, ok := payloadMap["errcode"]
	if !ok {
		return 0, false
	}
	return extractInt(value)
}

func extractInt(value any) (int, bool) {
	switch typed := value.(type) {
	case float64:
		return int(typed), true
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case json.Number:
		n, err := typed.Int64()
		if err != nil {
			return 0, false
		}
		return int(n), true
	default:
		return 0, false
	}
}

func responsePreview(rawBody []byte) string {
	const maxPreview = 200

	trimmed := bytes.TrimSpace(rawBody)
	if len(trimmed) == 0 {
		return ""
	}
	if len(trimmed) <= maxPreview {
		return string(trimmed)
	}
	return string(trimmed[:maxPreview]) + "...(truncated)"
}

func requestHeaders(token string) map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + strings.TrimSpace(token),
		"Content-Type":  "application/json",
	}
}
