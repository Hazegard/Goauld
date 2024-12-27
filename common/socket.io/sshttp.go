package socket_io

type SsHttp struct {
	Data []byte `json:"data,omitempty"`
	Id   string `json:"id,omitempty"`
	Num  int64  `json:"num,omitempty"`
}

const (
	SSHTTPEvent = "SSHttpEvent"
)
