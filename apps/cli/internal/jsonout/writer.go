package jsonout

import (
	"encoding/json"
	"io"
)

// WriteJSON writes a value as pretty-printed JSON to the writer.
func WriteJSON(w io.Writer, v any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}
