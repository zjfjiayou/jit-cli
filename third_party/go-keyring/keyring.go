package keyring

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

var ErrNotFound = errors.New("keyring entry not found")

func Get(service, user string) (string, error) {
	output, _, err := runSecurityCommand("find-generic-password", "-s", service, "-a", user, "-w")
	if err != nil {
		return "", wrapSecurityError("find-generic-password", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func Set(service, user, password string) error {
	if _, _, err := runSecurityCommand("add-generic-password", "-U", "-s", service, "-a", user, "-w", password); err != nil {
		return wrapSecurityError("add-generic-password", err)
	}
	return nil
}

func Delete(service, user string) error {
	if runtime.GOOS != "darwin" {
		return nil
	}
	if _, _, err := runSecurityCommand("delete-generic-password", "-s", service, "-a", user); err != nil {
		return wrapSecurityError("delete-generic-password", err)
	}
	return nil
}

func runSecurityCommand(args ...string) ([]byte, string, error) {
	if runtime.GOOS != "darwin" {
		return nil, "", errors.New("keyring unavailable")
	}

	cmd := exec.Command("security", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err == nil {
		return output, stderr.String(), nil
	}
	if isSecurityNotFound(stderr.String()) {
		return nil, stderr.String(), ErrNotFound
	}
	return nil, stderr.String(), err
}

func isSecurityNotFound(stderr string) bool {
	lower := strings.ToLower(stderr)
	return strings.Contains(lower, "could not be found") || strings.Contains(lower, "item could not be found")
}

func wrapSecurityError(action string, err error) error {
	if errors.Is(err, ErrNotFound) {
		return ErrNotFound
	}
	return fmt.Errorf("security %s failed: %w", action, err)
}
