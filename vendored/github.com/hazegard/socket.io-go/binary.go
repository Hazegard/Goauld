package sio

import jsonparser "github.com/hazegard/socket.io-go/parser/json"

type Binary []byte

func (b Binary) MarshalJSON() ([]byte, error) {
	return (jsonparser.Binary)(b).MarshalJSON()
}

func (b *Binary) UnmarshalJSON(data []byte) error {
	return (*jsonparser.Binary)(b).UnmarshalJSON(data)
}

func (b Binary) SocketIOBinary() bool {
	return true
}
