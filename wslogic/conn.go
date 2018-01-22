package wslogic

import (
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"local/gintest/constants"

	"github.com/gorilla/websocket"
)

const (

	// The buffer of the channel handling the messages to be sent to its respective client (send channel)
	sizeMsgChanBuffer = 256

	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512

	debugging          = constants.Debugging
	debugWithTimeStamp = constants.DebugWithTimeStamp
)

var (
	sessionCounter int32
)

// Conn is an middleman between the websocket connection and the hub.
type Conn struct {
	// The websocket connection.
	ws *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	connID int32
}

type clientMessage struct {
	conn    *Conn
	message []byte
}

func newClientMessage(conn *Conn, message []byte) clientMessage {
	return clientMessage{conn: conn, message: message}
}

// NewConn returns a new Connection to work with a session
func NewConn(ws *websocket.Conn) *Conn {
	return &Conn{ws: ws, send: make(chan []byte, sizeMsgChanBuffer), connID: atomic.AddInt32(&sessionCounter, 1) - 1}
}

func (c *Conn) log(v ...interface{}) {
	if debugging {
		prefix := fmt.Sprint("<Connection ", c.connID, "> ~ ")
		if debugWithTimeStamp {
			prefix = time.Now().Format(time.StampMicro) + " " + prefix
		}
		text := fmt.Sprint(v...)
		log.Println(prefix, text)
	}
}

func (c *Conn) logf(format string, v ...interface{}) {
	if debugging {
		prefix := fmt.Sprint("<Connection ", c.connID, "> ~ ")
		if debugWithTimeStamp {
			prefix = time.Now().Format(time.StampMicro) + " " + prefix
		}
		log.Printf(prefix+format, v...)
	}
}

// ReadPump pumps messages from the websocket connection to the hub.
func (c *Conn) ReadPump() {
	defer func() {
		c.log("exiting readPump()")
		Unregister(c)
		c.ws.Close()
	}()

	c.ws.SetReadLimit(maxMessageSize)
	c.ws.SetReadDeadline(time.Now().Add(pongWait))
	c.ws.SetPongHandler(
		func(string) error {
			c.ws.SetReadDeadline(time.Now().Add(pongWait))
			return nil
		})
	for {
		c.log("waiting for message from client")
		typ, message, err := c.ws.ReadMessage()
		c.log("Got a message ", typ, " ", string(message), " ", err)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				c.logf("error: %v", err)
			}
			break
		}
		//log.Println("Broadcasting the message received")
		//hub.broadcast <- message
		c.log("Sending the incoming message to be handled")
		processClientMessage(newClientMessage(c, message))
	}
}

// write writes a message with the given message type and payload.
func (c *Conn) write(mt int, payload []byte) error {
	c.ws.SetWriteDeadline(time.Now().Add(writeWait))
	return c.ws.WriteMessage(mt, payload)
}

// WritePump pumps messages from the hub to the websocket connection.
func (c *Conn) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		c.log("exiting writePump()")
		ticker.Stop()
		c.ws.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.log("message to be sent arrived")
			if !ok {
				// The hub closed the channel.
				c.log("The hub closed this connection")
				c.write(websocket.CloseMessage, []byte{})
				return
			}

			c.ws.SetWriteDeadline(time.Now().Add(writeWait))
			//w, err := c.ws.NextWriter(websocket.TextMessage)
			//if err != nil {
			//	return
			//}
			err := c.ws.WriteMessage(websocket.TextMessage, message)

			// Add queued chat messages to the current websocket message.
			n := len(c.send)
			c.log("There are ", n, " messages yet to send")
			var i int
			for i = 0; i < n; i++ {
				if err = c.ws.WriteMessage(websocket.TextMessage, <-c.send); err != nil {
					c.log("Error writing queued message: ", err.Error())
					return
				}
			}
			if i > 0 {
				c.log(i, " additional messages were sent!")
			}

		case <-ticker.C:
			c.log("Ping time...")
			if err := c.write(websocket.PingMessage, []byte{}); err != nil {
				log.Println("Error on ping: ", err.Error())
				return
			}
			c.log("Ping Ok!")
		}
	}
}
