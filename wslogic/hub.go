package wslogic

import (
	"container/list"
	"encoding/json"
	"errors"
	"fmt"
	"local/gintest/constants"
	"local/gintest/services/db"
	"local/gintest/services/pid"
	"log"
	"math/rand"
	"time"
)

var (
	dbase *db.DB
)

// Hub maintains the set of active connections and broadcasts messages to the
// connections.
type Hub struct {
	// Registered connections.
	connectionsMap  map[int32]*Conn
	connectionsPool *list.List

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
	broadcast:       make(chan []byte),
	incomingMessage: make(chan clientMessage),
	register:        make(chan *Conn),
	unregister:      make(chan *Conn),
	connectionsMap:  make(map[int32]*Conn),
	connectionsPool: list.New(),
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

func (h *Hub) removeConnection(conn *Conn) error {

	delete(h.connectionsMap, conn.connID)

	for e := h.connectionsPool.Front(); e != nil; e = e.Next() {
		if e.Value.(*Conn) == conn {
			h.log("Found  the connection to be unregistered (id ", conn.connID, ")")
			h.connectionsPool.Remove(e)
			close(conn.send)
			h.log("There are now ", h.connectionsPool.Len(), " (", len(h.connectionsMap), " in map) active connections")
			return nil
		}
	}

	return errors.New("The connection was not found into the list")
}

func (h *Hub) registerConnection(conn *Conn) {
	h.connectionsMap[conn.connID] = conn
	h.connectionsPool.PushBack(conn)
	h.log("There are now ", h.connectionsPool.Len(), " (", len(h.connectionsMap), " in map) active connections")
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

func (h *Hub) BroadcastMessage(message []byte) {
	h.log("There are ", h.connectionsPool.Len(), " connections to broadcast to")
	i := 1
	for e := h.connectionsPool.Front(); e != nil; e = e.Next() {
		h.log("broadcast to conn in place", i)
		i++
		select {
		// If the channel can not proceed inmediately its because its buffer is full,
		// so we presume that the connection with the client was lost
		case e.Value.(*Conn).send <- message:
		default:
			h.log("Removing a connection inside default (client message queue full)")
			close(e.Value.(*Conn).send)
			err := h.removeConnection(e.Value.(*Conn))
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
		var cmm constants.EventCommand
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
	for {
		select {
		// A new connection arrived to be registered
		case conn := <-h.register:
			h.log("Registering a connection")
			h.registerConnection(conn)

			// A connection needs to be deleted
		case conn := <-h.unregister:
			h.log("Unregistering a connection")

			//if _, ok := h.connections[conn]; ok {
			//	delete(h.connections, conn)
			//	close(conn.send)
			//}

			err := h.removeConnection(conn)
			if err != nil {
				h.log("Error removing connection: ", err.Error())
			}

			// A message needs to be broadcasted
		case message := <-h.broadcast:
			h.log("Broadcasting")
			h.BroadcastMessage(message)
		}
	}
}

func standardTickHandler(t *pid.DummyPIDTicker, time time.Time) {
	randfloat := rand.Float32() * 100
	event := constants.NewPIDUpdateEvent(t.GetIndex(), time.UnixNano(), randfloat)
	message, err := event.Stringify()
	if err != nil {
		log.Println("Error stringifying: ", err)
		return
	}
	log.Println("Broadcasting event ", string(message), " by Dummy Ticker ", t.GetName())
	Broadcast(message)
	log.Println("Saving sample to DB")
	d, err := dbase.Copy()
	if err != nil {
		log.Println("Error copying the db session: ", err)
		return
	}
	defer d.Close()
	d.InsertSamples(&db.DBSample{Pid: t.GetIndex(), Value: event.Value, Timestamp: event.Timestamp})
}

func init() {
	now := time.Now().UnixNano()
	hub.log("Launching hub init")
	go hub.runClientMessageHandler()
	go hub.runHub()

	for i := 0; i < 2; i++ {
		t := pid.NewDummyPIDTicker(fmt.Sprint("Signal ", i), time.Second*10, standardTickHandler)
		pid.Subscribe(t)
		t.Launch()
	}

	dab, err := db.Dial()
	if err != nil {
		hub.log("Error dialing to the DB: ", err)
	} else {
		dbase = dab
		listEvent := pid.RequestPIDListEventStruct()
		hub.log("Obtained ", len(listEvent.List), " pids to insert to the DB")
		var dbpids db.DBPids
		pids := make([]*db.DBPid, len(listEvent.List))
		for i, pid := range listEvent.List {
			pids[i] = &db.DBPid{Name: pid.Name, Pid: pid.Index, Period: time.Duration(pid.Period)}
		}
		dbpids.Timestamp = now
		dbpids.Pids = pids
		dbase.InsertPids(dbpids)
	}
}
