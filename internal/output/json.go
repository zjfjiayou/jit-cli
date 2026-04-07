package output

import (
	"bytes"
	"encoding/json"
	"io"
)

func WriteJSON(w io.Writer, payload any, pretty bool) error {
	var (
		data []byte
		err  error
	)
	if pretty {
		data, err = json.MarshalIndent(payload, "", "  ")
	} else {
		data, err = json.Marshal(payload)
	}
	if err != nil {
		return err
	}
	_, err = w.Write(append(data, '\n'))
	return err
}

func WriteRawJSON(w io.Writer, raw []byte, pretty bool) error {
	if !pretty {
		if len(raw) == 0 || raw[len(raw)-1] != '\n' {
			raw = append(raw, '\n')
		}
		_, err := w.Write(raw)
		return err
	}

	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "", "  "); err != nil {
		if len(raw) == 0 || raw[len(raw)-1] != '\n' {
			raw = append(raw, '\n')
		}
		_, writeErr := w.Write(raw)
		return writeErr
	}
	buf.WriteByte('\n')
	_, err := w.Write(buf.Bytes())
	return err
}
