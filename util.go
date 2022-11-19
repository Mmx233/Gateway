package gateway

import (
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"net/http"
	"time"
)

var Upper = websocket.Upgrader{
	HandshakeTimeout: time.Minute,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func UpgradeWs(c *gin.Context) (*websocket.Conn, error) {
	return Upper.Upgrade(c.Writer, c.Request, map[string][]string{
		"Sec-WebSocket-Protocol": {c.GetHeader("Sec-WebSocket-Protocol")},
	})
}
