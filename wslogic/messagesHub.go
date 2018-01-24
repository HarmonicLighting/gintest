package wslogic

import (
	"encoding/json"
	"fmt"
	"local/gintest/commons"
	"local/gintest/services/pid"
	"log"
	"time"
)

type MessagesHub struct {

	// Incoming messages from the connections
	incomingMessage chan clientMessage
}

var messagesHub = MessagesHub{
	incomingMessage: make(chan clientMessage),
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

func (h *MessagesHub) runMessagesHub() {

	for clientMessage := range h.incomingMessage {

		var cmm commons.ApiRequestHeader
		err := json.Unmarshal(clientMessage.message, &cmm)
		if err != nil {
			h.log("Error unmarshalling the event command ", string(clientMessage.message), ": ", err)
			badRequestResponse := commons.NewBadRequestApiResponse()
			response, _ := badRequestResponse.Stringify()
			clientMessage.conn.send <- response
		} else {

			switch cmm.Command {

			case commons.ApiPidListCommandRequest:
				rc := commons.NewCommandRequest(cmm.Command, clientMessage.message)
				response := pid.RequestPidList(rc)
				clientMessage.conn.send <- response

			case commons.ApiNCurrentClientsCommandRequest:
				rc := commons.NewCommandRequest(cmm.Command, clientMessage.message)
				response := connectionsHub.requestNCurrentClientsCommand(rc)
				clientMessage.conn.send <- response

			default:
				h.log("The request command ", cmm.Command, " is not supported.")
				notSupportedResponse := commons.NewNotSupportedStatusApiResponse(cmm.Command)
				response, _ := notSupportedResponse.Stringify()
				clientMessage.conn.send <- response
			}
		}
	}
}

func init() {
	log.Println("INIT MessagesHUB.GO >>> ", commons.GetInitCounter())
	go messagesHub.runMessagesHub()
}
