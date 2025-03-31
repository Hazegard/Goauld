package types

import "encoding/json"

type HttpResponse struct {
	Message string `json:"message"`
	Success bool   `json:"status"`
}

func (h *HttpResponse) String() string {
	b, err := json.Marshal(h)
	if err != nil {
		return ""
	}
	return string(b)
}

func (h *HttpResponse) Bytes() []byte {
	b, err := json.Marshal(h)
	if err != nil {
		return []byte{}
	}
	return b
}

func NewErrHttpResponse(err error) *HttpResponse {
	return &HttpResponse{
		Message: err.Error(),
		Success: false,
	}
}
