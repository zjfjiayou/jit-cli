package output

import "io"

type CLIError struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func WriteCLIError(w io.Writer, code, message string, details any) error {
	return WriteJSON(w, CLIError{
		Error:   code,
		Message: message,
		Details: details,
	}, false)
}
