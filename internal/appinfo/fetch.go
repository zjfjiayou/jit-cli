package appinfo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"jit-cli/internal/client"
)

type rawAppInfo struct {
	OrgID      string                   `json:"orgId"`
	AppID      string                   `json:"appId"`
	Name       string                   `json:"name"`
	Title      string                   `json:"title"`
	Version    string                   `json:"version"`
	Elements   map[string]ElementDefine `json:"elements"`
	ExtendApps []rawAppInfo             `json:"extendApps"`
}

func Fetch(ctx context.Context, server, org, app, token string) (*AppInfo, error) {
	urlValue, err := buildAppInfoURL(server, org, app)
	if err != nil {
		return nil, err
	}

	token = strings.TrimSpace(token)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlValue, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch appInfo.js (status %d): %s", resp.StatusCode, responsePreview(rawBody))
	}

	decrypted := xorDecrypt(rawBody, org+"."+app)
	jsonText, err := extractAppInfoJSON(decrypted)
	if err != nil {
		return nil, err
	}

	var raw rawAppInfo
	if err := json.Unmarshal([]byte(jsonText), &raw); err != nil {
		return nil, fmt.Errorf("parse app info json: %w", err)
	}

	info := sanitizeAppInfo(raw, normalizeAppID(org, app, raw.AppID))
	return &info, nil
}

func buildAppInfoURL(server, org, app string) (string, error) {
	normalizedServer, err := client.NormalizeServer(server)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s/%s/appInfo.js", normalizedServer, strings.TrimSpace(org), strings.TrimSpace(app)), nil
}

func xorDecrypt(encrypted []byte, key string) string {
	keyBytes := []byte(key)
	if len(keyBytes) == 0 {
		return string(encrypted)
	}

	result := make([]byte, len(encrypted))
	for i, b := range encrypted {
		result[i] = b ^ keyBytes[i%len(keyBytes)]
	}
	return string(result)
}

func extractAppInfoJSON(source string) (string, error) {
	marker := "JIT_APP_INFO"
	idx := strings.Index(source, marker)
	if idx < 0 {
		return "", fmt.Errorf("JIT_APP_INFO assignment not found")
	}

	assignment := source[idx:]
	start := strings.Index(assignment, "{")
	end := strings.LastIndex(assignment, "}")
	if start < 0 || end < start {
		return "", fmt.Errorf("JIT_APP_INFO json block not found")
	}
	return assignment[start : end+1], nil
}

func sanitizeAppInfo(raw rawAppInfo, appID string) AppInfo {
	info := AppInfo{
		Name:     raw.Name,
		Title:    raw.Title,
		AppID:    appID,
		Version:  raw.Version,
		Elements: make(map[string]ElementDefine, len(raw.Elements)),
	}

	for path, element := range raw.Elements {
		info.Elements[path] = sanitizeElement(path, element)
	}
	for _, child := range raw.ExtendApps {
		info.ExtendApps = append(info.ExtendApps, sanitizeAppInfo(child, normalizeAppID(child.OrgID, child.Name, child.AppID)))
	}
	return info
}

func sanitizeElement(path string, element ElementDefine) ElementDefine {
	if strings.TrimSpace(element.FullName) == "" {
		element.FullName = strings.TrimSpace(path)
	}
	return element
}

func normalizeAppID(org, app, rawAppID string) string {
	org = strings.TrimSpace(org)
	app = strings.TrimSpace(app)
	if org != "" && app != "" {
		return org + "/" + app
	}

	clean := strings.TrimSpace(rawAppID)
	if strings.Contains(clean, "/") {
		return clean
	}
	if idx := strings.Index(clean, "."); idx > 0 && idx < len(clean)-1 {
		return clean[:idx] + "/" + clean[idx+1:]
	}
	return clean
}

func responsePreview(rawBody []byte) string {
	const maxPreview = 200

	trimmed := strings.TrimSpace(string(rawBody))
	if len(trimmed) <= maxPreview {
		return trimmed
	}
	return trimmed[:maxPreview] + "...(truncated)"
}
