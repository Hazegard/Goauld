package websocket

import "github.com/coder/websocket"

var expectedCloseCodes = []websocket.StatusCode{
	websocket.StatusNormalClosure,
	websocket.StatusGoingAway,
	websocket.StatusNoStatusRcvd,
	websocket.StatusAbnormalClosure,
}
