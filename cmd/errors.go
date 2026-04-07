package cmd

import (
	"encoding/json"
	"fmt"
)

type CLIError struct {
	Code    int    `json:"-"`
	Key     string `json:"error"`
	Message string `json:"message"`
}

func (e *CLIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Key, e.Message)
}

func NewCLIError(key string, message string) *CLIError {
	return &CLIError{
		Code:    ExitCLIError,
		Key:     key,
		Message: message,
	}
}

type ExitCodeOnlyError struct {
	Code int
}

func (e *ExitCodeOnlyError) Error() string {
	return fmt.Sprintf("exit with code %d", e.Code)
}

func writeCLIErrorJSON(writable WriterFlusher, cliErr *CLIError) {
	if cliErr == nil {
		return
	}
	data, err := json.Marshal(cliErr)
	if err != nil {
		_, _ = writable.Write([]byte(`{"error":"internal_error","message":"failed to marshal error"}` + "\n"))
		return
	}
	_, _ = writable.Write(append(data, '\n'))
}

type WriterFlusher interface {
	Write([]byte) (int, error)
}
