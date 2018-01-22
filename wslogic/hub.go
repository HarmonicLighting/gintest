package wslogic

import (
	"container/list"
	"encoding/json"
	"errors"
	"fmt"
	"local/gintest/constants"
	"local/gintest/services/pid"
	"log"
	"time"
)

// Hub maintains the set of active connections and broadcasts messages to the
// connections.
type Hub struct {
	// Registered connections.
	//connectionsMap  map[int32]*Conn
	//connectionsPool *list.List

	// Messages to broadcast to all the connections
	broadcast chan []byte

	// Incoming messages from the connections
	incomingMessage chan clientMessage

	// Register requests from the connections.
	register chan *Conn

	// Unregister requests from connections.
	unregister chan *Conn
}

var hub = Hub{
	incomingMessage: make(chan clientMessage),

	broadcast:  make(chan []byte),
	register:   make(chan *Conn),
	unregister: make(chan *Conn),
	//connectionsMap:  make(map[int32]*Conn),
	//connectionsPool: list.New(),
}

func (h *Hub) log(v ...interface{}) {
	if debugging {
		text := fmt.Sprint(v...)
		prefix := fmt.Sprint("<<< HUB >>> ~ ")
		if debugWithTimeStamp {
			prefix = time.Now().Format(time.StampMicro) + " " + prefix
		}
		log.Println(prefix, text)
	}
}

func (h *Hub) logf(format string, v ...interface{}) {
	if debugging {
		prefix := fmt.Sprint("<<< HUB >>> ~ ")
		if debugWithTimeStamp {
			prefix = time.Now().Format(time.StampMicro) + " " + prefix
		}
		log.Printf(prefix+format, v...)
	}
}

func (h *Hub) removeConnection(conn *Conn, connectionsList *list.List, connectionsMap map[int32]*Conn) error {

	delete(connectionsMap, conn.connID)

	for e := connectionsList.Front(); e != nil; e = e.Next() {
		if e.Value.(*Conn) == conn {
			h.log("Found  the connection to be unregistered (id ", conn.connID, ")")
			connectionsList.Remove(e)
			close(conn.send)
			h.log("There are now ", connectionsList.Len(), " (", len(connectionsMap), " in map) active connections")
			return nil
		}
	}

	return errors.New("The connection was not found into the list")
}

func (h *Hub) registerConnection(conn *Conn, connectionsList *list.List, connectionsMap map[int32]*Conn) {
	connectionsMap[conn.connID] = conn
	connectionsList.PushBack(conn)
	h.log("There are now ", connectionsList.Len(), " (", len(connectionsMap), " in map) active connections")
}

func Register(conn *Conn) {
	hub.register <- conn
}

func Unregister(conn *Conn) {
	hub.unregister <- conn
}

func Broadcast(message []byte) {
	hub.broadcast <- message
}

func processClientMessage(cm clientMessage) {
	hub.incomingMessage <- cm
}

func (h *Hub) BroadcastMessage(message []byte, connectionsList *list.List, connectionsMap map[int32]*Conn) {
	h.log("There are ", connectionsList.Len(), " connections to broadcast to")
	i := 1
	for e := connectionsList.Front(); e != nil; e = e.Next() {
		h.log("broadcast to conn in place", i)
		i++
		select {

		// If the channel can not proceed inmediately its because its buffer is full,
		// so we presume that the connection with the client was lost
		case e.Value.(*Conn).send <- message:

		default:
			h.log("Removing a connection inside default (client message queue full)")
			close(e.Value.(*Conn).send)
			err := h.removeConnection(e.Value.(*Conn), connectionsList, connectionsMap)
			if err != nil {
				h.log("Error removing connection: ", err.Error())
			}
		}
	}
}

func (h *Hub) runClientMessageHandler() {
	for {
		clientMessage := <-h.incomingMessage
		// Handle the client message Here
		var cmm constants.CommandHeader
		err := json.Unmarshal(clientMessage.message, &cmm)
		if err != nil {
			h.log("Error unmarshalling the event command ", string(clientMessage.message), ": ", err)
		} else {
			switch cmm.Command {
			case constants.PIDListCommand:
				rc := constants.NewCommandRequest(constants.PIDListCommand, clientMessage.message)
				response := pid.RequestCommand(rc)
				if response.Err != nil {
					h.log("Error processing the command ", string(clientMessage.message), " for client ", clientMessage.conn.connID, ": ", response.Err)
					//clientMessage.conn.send
				} else {
					log.Println("Sending the processed command response to client ", clientMessage.conn.connID)
					clientMessage.conn.send <- response.Response
				}
			default:
				h.log("The command ", cmm.Command, " is not supported.")
			}
		}

		// for now we will just broadcast the arriving messages
		//h.log("Broadcasting message comming from the connection ", clientMessage.conn.connID)
		//Broadcast(clientMessage.message)

	}
}

func (h *Hub) runHub() {

	// The shared variables are declared in the beginning
	connectionsList := list.New()
	connectionsMap := make(map[int32]*Conn)

	for {
		select {
		// A new connection arrived to be registered
		case conn := <-h.register:
			h.log("Registering a connection")
			h.registerConnection(conn, connectionsList, connectionsMap)

			// A connection needs to be deleted
		case conn := <-h.unregister:
			h.log("Unregistering a connection")

			err := h.removeConnection(conn, connectionsList, connectionsMap)
			if err != nil {
				h.log("Error removing connection: ", err.Error())
			}

			// A message needs to be broadcasted
		case message := <-h.broadcast:
			h.log("Broadcasting")
			h.BroadcastMessage(message, connectionsList, connectionsMap)
		}
	}
}

func init() {
	//now := time.Now().UnixNano()
	hub.log("Launching hub init")
	go hub.runClientMessageHandler()
	go hub.runHub()

	pid.SetBroadcastHandle(Broadcast)

}
