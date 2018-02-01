package ws

import (
	"log"

	"local/gintest/wslogic"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// ServeWs handles websocket requests from the peer.
func ServeWs(c *gin.Context) {
	uid, _ := c.Get("userID")
	userID := uid.(string)
	log.Println("User ID: ", userID)
	w := c.Writer
	r := c.Request
	token := r.URL.Query().Get("token")
	log.Println("WS Token: ", token)
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	conn := wslogic.NewConn(ws)
	wslogic.Register(conn)
	go conn.WritePump()
	go conn.ReadPump()
}
