package wslogic

import (
	"container/list"
	"errors"
	"fmt"
	"local/gintest/commons"
	"log"
	"time"
)

// An implementation Idea to create different responses in a generic way providing a handle function operating on the hub shared resources
//type ConnectionsHubResponseFunction func(connectionsList *list.List) commons.RawCommandResponse
//type ConnectionsCommandRequest struct {
//	commons.CommandRequest
//	function ConnectionsHubResponseFunction
//}

// Hub maintains the set of active connections and broadcasts messages to the
// connections.
type ConnectionsHub struct {

	// Messages to broadcast to all the connections
	broadcast chan []byte

	// Messages to send to a specific client
	send chan clientMessage

	// Register requests from the connections.
	register chan *Conn

	// Unregister requests from connections.
	unregister chan *Conn

	// Attend Number of current Clients Command requests.
	incomingNCurrentClientsCommand chan commons.CommandRequest
}

var connectionsHub = ConnectionsHub{
	broadcast:                      make(chan []byte),
	send:                           make(chan clientMessage),
	register:                       make(chan *Conn),
	unregister:                     make(chan *Conn),
	incomingNCurrentClientsCommand: make(chan commons.CommandRequest),
}

func (h *ConnectionsHub) log(v ...interface{}) {
	if debugging {
		text := fmt.Sprint(v...)
		prefix := fmt.Sprint("< CONN HUB > ~ ")
		if debugWithTimeStamp {
			prefix = time.Now().Format(time.StampMicro) + " " + prefix
		}
		log.Println(prefix, text)
	}
}

func (h *ConnectionsHub) logf(format string, v ...interface{}) {
	if debugging {
		prefix := fmt.Sprint("< CONN HUB > ~ ")
		if debugWithTimeStamp {
			prefix = time.Now().Format(time.StampMicro) + " " + prefix
		}
		log.Printf(prefix+format, v...)
	}
}

func (h *ConnectionsHub) removeConnection(conn *Conn, connectionsList *list.List, connectionsMap map[connectionID]*Conn) {

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

	h.log("The connection was not found into the list")
}

func (h *ConnectionsHub) registerConnection(conn *Conn, connectionsList *list.List, connectionsMap map[connectionID]*Conn) {
	connectionsMap[conn.connID] = conn
	connectionsList.PushBack(conn)
	h.log("There are now ", connectionsList.Len(), " (", len(connectionsMap), " in map) active connections")
}

func Register(conn *Conn) {
	connectionsHub.register <- conn
}

func Unregister(conn *Conn) {
	connectionsHub.unregister <- conn
}

func Broadcast(message []byte) {
	connectionsHub.broadcast <- message
}

func Send(cmessage clientMessage) {
	connectionsHub.send <- cmessage
}

func (h *ConnectionsHub) broadcastMessage(message []byte, connectionsList *list.List, connectionsMap map[connectionID]*Conn) {
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
}

func (h *ConnectionsHub) sendMessage(cMessage clientMessage, connectionsList *list.List, connectionsMap map[connectionID]*Conn) error {
	for e := connectionsList.Front(); e != nil; e = e.Next() {
		conn := e.Value.(*Conn)
		if conn.connID == cMessage.connID {
			select {
			// The message was successfully queued up
			case conn.send <- cMessage.toMessage:
				h.log("A message focused towards the connection ", cMessage.connID, " was queued up")
				return nil
			default:
				h.log("Removing connection ", e.Value.(Conn).connID, ", unable to send message (client message queue full)")
				close(e.Value.(*Conn).send)
				h.removeConnection(e.Value.(*Conn), connectionsList, connectionsMap)
				return errors.New("The client connection was found, but it's message queue was full")
			}
		}

	}
	return errors.New(fmt.Sprint("The client with connection id ", cMessage.connID, " was not found in the connections list"))
}

func (h *ConnectionsHub) runConnectionsHub() {

	// The shared variables are declared in the beginning
	connectionsList := list.New()
	connectionsMap := make(map[connectionID]*Conn)

	staticsTicker := time.NewTicker(time.Minute)
	defer staticsTicker.Stop()
	var nRegistered, nUnregistered, nBroadcasts int
	for {
		select {

		// A new connection arrived to be registered
		case conn := <-h.register:
			nRegistered++
			h.log("Registering a connection")
			h.registerConnection(conn, connectionsList, connectionsMap)

			responseData := processNCurrentClientsCommand(connectionsList)
			h.broadcastMessage(responseData, connectionsList, connectionsMap)

			// A connection needs to be deleted
		case conn := <-h.unregister:
			nUnregistered++
			h.log("Unregistering a connection")

			h.removeConnection(conn, connectionsList, connectionsMap)
			responseData := processNCurrentClientsCommand(connectionsList)
			// Broadcast the updated Client connections count
			h.broadcastMessage(responseData, connectionsList, connectionsMap)

			// A message needs to be broadcasted
		case message := <-h.broadcast:
			nBroadcasts++
			nconn := connectionsList.Len()
			if nconn > 0 {
				//h.log("Broadcasting to ", nconn, " connections")
				h.broadcastMessage(message, connectionsList, connectionsMap)
			} else {
				//h.log("No clients to broadcast")
			}
		case cMessage := <-h.send:
			err := h.sendMessage(cMessage, connectionsList, connectionsMap)
			if err != nil {
				h.log(err)
			}
			// A command operating on shared resources arrived
		case request := <-h.incomingNCurrentClientsCommand:
			h.log("Dispatching NCurrentClients Command")
			// The generic way would be something like
			// request.Response <- request.ConnectionsHubResponseFunction(connectionsList)
			responseData := processNCurrentClientsCommand(connectionsList)
			request.Response <- responseData

		case <-staticsTicker.C:
			h.log("Connections Hub Statics of the last munute: ", nBroadcasts, " broadcasts handled\t\t\t\t", nRegistered, " connections registered\t\t\t\t", nUnregistered, " connections unregistered")
			nBroadcasts = 0
			nRegistered = 0
			nUnregistered = 0
		}
	}
}

func Init() {

	log.Println("INIT MessagesHUB.GO >>> ", commons.GetInitCounter())
	go messagesHub.runMessagesHub()

	log.Println("INIT ConnectionsHUB.GO >>> ", commons.GetInitCounter())

	RegisterMessagesHandler(RequestMessagesHandler{RequestType: commons.ApiNCurrentClientsCommandRequest, Handler: connectionsHub.requestNCurrentClientsCommand})
	log.Println("INIT ConnectionsHUB.GO >>> Back from registering messages handler")
	go connectionsHub.runConnectionsHub()

	//pid.SetBroadcastHandle(Broadcast)

}
