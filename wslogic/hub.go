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

// An implementation Idea to create different responses in a generic way providing a handle function operating on the hub shared resources
//type ConnectionsHubResponseFunction func(connectionsList *list.List) constants.RawCommandResponse
//type ConnectionsCommandRequest struct {
//	constants.CommandRequest
//	function ConnectionsHubResponseFunction
//}

// Hub maintains the set of active connections and broadcasts messages to the
// connections.
type Hub struct {

	// Messages to broadcast to all the connections
	broadcast chan []byte

	// Incoming messages from the connections
	incomingMessage chan clientMessage

	// Register requests from the connections.
	register chan *Conn

	// Unregister requests from connections.
	unregister chan *Conn

	// Attend Number of current Clients Command requests.
	incomingNCurrentClientsCommand chan constants.CommandRequest
}

var hub = Hub{
	incomingMessage: make(chan clientMessage),

	broadcast:                      make(chan []byte),
	register:                       make(chan *Conn),
	unregister:                     make(chan *Conn),
	incomingNCurrentClientsCommand: make(chan constants.CommandRequest),
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

func (h *Hub) removeConnection(conn *Conn, connectionsList *list.List, connectionsMap map[int32]*Conn) {

	delete(connectionsMap, conn.connID)

	for e := connectionsList.Front(); e != nil; e = e.Next() {
		if e.Value.(*Conn) == conn {
			h.log("Found  the connection to be unregistered (id ", conn.connID, ")")
			connectionsList.Remove(e)
			close(conn.send)
			h.log("There are now ", connectionsList.Len(), " (", len(connectionsMap), " in map) active connections")
			return
		}
	}

	errors.New("The connection was not found into the list")
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
	for e := connectionsList.Front(); e != nil; e = e.Next() {
		select {

		// If the channel can not proceed inmediately its because its buffer is full,
		// so we presume that the connection with the client was lost
		case e.Value.(*Conn).send <- message:

		default:
			h.log("Removing connection ", e.Value.(Conn).connID, ", unable to broadcast (client message queue full)")
			close(e.Value.(*Conn).send)
			h.removeConnection(e.Value.(*Conn), connectionsList, connectionsMap)
		}
	}

	if connectionsList.Len() != 0 {
		h.log("Broadcasting done...")
	}
}

func (h *Hub) runClientsMessageHandler() {

	for clientMessage := range h.incomingMessage {

		var cmm constants.CommandRequestHeader
		err := json.Unmarshal(clientMessage.message, &cmm)
		if err != nil {
			h.log("Error unmarshalling the event command ", string(clientMessage.message), ": ", err)
			badRequestResponse := constants.NewBadRequestCommandResponse()
			response, _ := badRequestResponse.Stringify()
			clientMessage.conn.send <- response
		} else {

			switch cmm.Command {

			case constants.PIDListCommandRequest:
				rc := constants.NewCommandRequest(cmm.Command, clientMessage.message)
				response := pid.RequestCommand(rc)
				clientMessage.conn.send <- response

			case constants.NCurrentClientsCommandRequest:
				rc := constants.NewCommandRequest(cmm.Command, clientMessage.message)
				response := h.requestNCurrentClientsCommand(rc)
				clientMessage.conn.send <- response

			default:
				h.log("The request command ", cmm.Command, " is not supported.")
				notSupportedResponse := constants.NewNotSupportedStatusCommandResponse(cmm.Command)
				response, _ := notSupportedResponse.Stringify()
				clientMessage.conn.send <- response
			}
		}
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

			responseData := processNCurrentClientsCommand(connectionsList)
			h.BroadcastMessage(responseData, connectionsList, connectionsMap)

			// A connection needs to be deleted
		case conn := <-h.unregister:
			h.log("Unregistering a connection")

			h.removeConnection(conn, connectionsList, connectionsMap)
			responseData := processNCurrentClientsCommand(connectionsList)
			// Broadcast the updated Client connections count
			h.BroadcastMessage(responseData, connectionsList, connectionsMap)

			// A message needs to be broadcasted
		case message := <-h.broadcast:
			h.log("Broadcasting")
			h.BroadcastMessage(message, connectionsList, connectionsMap)

			// A command operating on shared resources arrived
		case request := <-h.incomingNCurrentClientsCommand:
			h.log("Dispatching NCurrentClients Command")
			// The generic way would be something like
			// request.Response <- request.ConnectionsHubResponseFunction(connectionsList)
			responseData := processNCurrentClientsCommand(connectionsList)
			request.Response <- responseData
		}
	}
}

func init() {
	hub.log("Launching hub init")
	go hub.runClientsMessageHandler()
	go hub.runHub()

	pid.SetBroadcastHandle(Broadcast)

}
