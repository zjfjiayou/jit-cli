package client

import (
	"fmt"
	"net/url"
	"strings"
)

// AppRef is the parsed {org}/{app} tuple.
type AppRef struct {
	Org string
	App string
}

// EndpointRef is the parsed {elementPath}/{functionName} tuple.
type EndpointRef struct {
	ElementPath  string
	FunctionName string
}

// ParseAppRef parses "org/app".
func ParseAppRef(value string) (AppRef, error) {
	parts := strings.Split(strings.TrimSpace(value), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return AppRef{}, fmt.Errorf("invalid app format %q, expected org/app", value)
	}
	return AppRef{Org: parts[0], App: parts[1]}, nil
}

// ParseEndpointRef parses "elementPath/functionName".
func ParseEndpointRef(value string) (EndpointRef, error) {
	clean := strings.Trim(value, "/")
	if clean == "" {
		return EndpointRef{}, fmt.Errorf("endpoint is empty")
	}

	idx := strings.LastIndex(clean, "/")
	if idx <= 0 || idx == len(clean)-1 {
		return EndpointRef{}, fmt.Errorf(
			"invalid endpoint format %q, expected elementPath/functionName",
			value,
		)
	}

	elementPath := clean[:idx]
	functionName := clean[idx+1:]
	if elementPath == "" || functionName == "" {
		return EndpointRef{}, fmt.Errorf(
			"invalid endpoint format %q, expected elementPath/functionName",
			value,
		)
	}

	return EndpointRef{
		ElementPath:  elementPath,
		FunctionName: functionName,
	}, nil
}

// NormalizeServer validates and normalizes server URL.
func NormalizeServer(server string) (string, error) {
	raw := strings.TrimSpace(server)
	if raw == "" {
		return "", fmt.Errorf("server is empty")
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid server %q: %w", server, err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid server %q, expected absolute URL", server)
	}

	return strings.TrimRight(parsed.String(), "/"), nil
}

// BuildURL builds /api/{org}/{app}/{elementPath}/{functionName}.
func BuildURL(server, app, endpoint string) (string, error) {
	normalizedServer, err := NormalizeServer(server)
	if err != nil {
		return "", err
	}

	appRef, err := ParseAppRef(app)
	if err != nil {
		return "", err
	}

	endpointRef, err := ParseEndpointRef(endpoint)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(
		"%s/api/%s/%s/%s/%s",
		normalizedServer,
		appRef.Org,
		appRef.App,
		strings.Trim(endpointRef.ElementPath, "/"),
		endpointRef.FunctionName,
	), nil
}
