package socket_io

type SioError struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}
