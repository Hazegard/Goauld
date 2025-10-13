//nolint:revive
package types

import "encoding/json"

// HTTPResponse represent the HTTP responses used by the server on its client API.
type HTTPResponse struct {
	Message string `json:"message"`
	Success bool   `json:"status"`
}

// String encode as string json the http message.
func (h *HTTPResponse) String() string {
	b, err := json.Marshal(h)
	if err != nil {
		return ""
	}

	return string(b)
}

// Bytes encode as bytes json the http message.
func (h *HTTPResponse) Bytes() []byte {
	b, err := json.Marshal(h)
	if err != nil {
		return []byte{}
	}

	return b
}
