package controllers

import (
	"log"
	"net/http"

	"local/gintest/wslogic"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// ServeWs handles websocket requests from the peer.
func ServeWs(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	conn := wslogic.NewConn(ws) //&Conn{send: make(chan []byte, 256), ws: ws}
	wslogic.Register(conn)      //hub.register <- conn
	go conn.WritePump()
	go conn.ReadPump()
}
