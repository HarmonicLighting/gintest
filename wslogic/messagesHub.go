package wslogic

import (
	"encoding/json"
	"fmt"
	"local/gintest/commons"
	"log"
	"time"
)

type MessageHandler func(commons.CommandRequest) commons.RawResponseData

type RequestMessagesHandler struct {
	RequestType commons.CommandRequestType
	Handler     MessageHandler
}

type MessagesHub struct {

	// Incoming messages from the connections
	incomingMessage chan clientMessage

	registerHandler   chan RequestMessagesHandler
	unregisterHandler chan RequestMessagesHandler
}

var messagesHub = MessagesHub{
	incomingMessage:   make(chan clientMessage),
	registerHandler:   make(chan RequestMessagesHandler),
	unregisterHandler: make(chan RequestMessagesHandler),
}

func (h *MessagesHub) log(v ...interface{}) {
	if debugging {
		text := fmt.Sprint(v...)
		prefix := fmt.Sprint("< MSGS HUB > ~ ")
		if debugWithTimeStamp {
			prefix = time.Now().Format(time.StampMicro) + " " + prefix
		}
		log.Println(prefix, text)
	}
}

func (h *MessagesHub) logf(format string, v ...interface{}) {
	if debugging {
		prefix := fmt.Sprint("< MSGS HUB > ~ ")
		if debugWithTimeStamp {
			prefix = time.Now().Format(time.StampMicro) + " " + prefix
		}
		log.Printf(prefix+format, v...)
	}
}

func processClientMessage(cm clientMessage) {
	messagesHub.incomingMessage <- cm
}

func RegisterMessagesHandler(handler RequestMessagesHandler) {
	messagesHub.registerHandler <- handler
}

func UnregisterMessagesHandler(handler RequestMessagesHandler) {
	messagesHub.unregisterHandler <- handler
}

func (h *MessagesHub) runMessagesHub() {

	requestHandlersMap := make(map[commons.CommandRequestType]MessageHandler)

	for {
		select {
		// Register a new handler
		case messageHandler := <-h.registerHandler:
			_, ok := requestHandlersMap[messageHandler.RequestType]
			if ok { // The is already a handler with the same command request type!
				h.log("REGISTER MESSAGE HANDLER ERROR! >>>> the request handler ", messageHandler.RequestType, " is already registered!")
			} else {
				requestHandlersMap[messageHandler.RequestType] = messageHandler.Handler
				h.log("New message handler with id ", messageHandler.RequestType, " has been registered. Now there are ", len(requestHandlersMap))
			}
			// Unregister a handler
		case messageHandler := <-h.unregisterHandler:
			_, ok := requestHandlersMap[messageHandler.RequestType]
			if ok { // a handler with the ID to remove was found
				delete(requestHandlersMap, messageHandler.RequestType)
				h.log("Message handler with id ", messageHandler.RequestType, " was unregistered. Now there are ", len(requestHandlersMap))
			} else {
				h.log("UNREGISTER MESSAGE HANDLER ERROR! >>>> the request handler ", messageHandler.RequestType, " is not registered!")
			}
			// handle an incoming message depending on it's request type
		case clientMessage := <-h.incomingMessage:
			var cmm commons.ApiRequestHeader
			err := json.Unmarshal(clientMessage.message, &cmm)
			if err != nil {
				h.log("Error unmarshalling the event command ", string(clientMessage.message), ": ", err)
				badRequestResponse := commons.NewBadRequestApiResponse()
				response, _ := badRequestResponse.Stringify()
				clientMessage.conn.send <- response
			} else {

				handler, ok := requestHandlersMap[cmm.Command]
				if ok {
					h.log("The request command ", cmm.Command, " will be processed.")
					rc := commons.NewCommandRequest(cmm.Command, clientMessage.message)
					response := handler(rc)
					clientMessage.conn.send <- response
				} else {
					// No handler with the command id has been registered
					h.log("The request command ", cmm.Command, " is not supported.")
					notSupportedResponse := commons.NewNotSupportedStatusApiResponse(cmm.Command)
					response, _ := notSupportedResponse.Stringify()
					clientMessage.conn.send <- response
				}
			}
		}
	}
}
