package wslogic

import (
	"encoding/json"
	"fmt"
	"local/gintest/apicommands"
	"log"
	"time"
)

type ResponseStatusType int

const notSupportedCommandResponse = -1
const requestNotSupportedStatus ResponseStatusType = -1

const badRequestStatus ResponseStatusType = -1

func newBadRequestApiResponse() ApiResponseHeader {
	return NewApiResponseHeader(notSupportedCommandResponse, badRequestStatus, fmt.Sprint("The Command Request is unrecognizable"))
}

func newNotSupportedStatusAPIResponse(command apicommands.CommandType) ApiResponseHeader {
	return NewApiResponseHeader(command, requestNotSupportedStatus, fmt.Sprint("The Command Request ", command, " is not supported"))
}

type RawResponseData []byte
type RawRequestData []byte

type CommandRequest struct {
	command  apicommands.CommandType
	data     RawRequestData
	response chan RawResponseData
}

func NewCommandRequest(command apicommands.CommandType, data []byte) CommandRequest {
	return CommandRequest{
		command:  command,
		data:     data,
		response: make(chan RawResponseData),
	}
}

func (cr *CommandRequest) SendCommandResponse(response RawResponseData) {
	if cr.response == nil {
		log.Println(">> COMMAND REQUEST ERROR: Response channel is Nil")
		return
	}
	cr.response <- response
}

func (cr *CommandRequest) ReceiveCommandResponse() RawResponseData {
	return <-cr.response
}

type messageHandler func(CommandRequest) RawResponseData

type RequestMessagesHandler struct {
	requestType apicommands.CommandType //commons.CommandRequestType
	handler     messageHandler
}

func NewRequestMessageHandler(requestType apicommands.CommandType, handler messageHandler) RequestMessagesHandler {
	return RequestMessagesHandler{
		requestType: requestType,
		handler:     handler,
	}
}

type ApiRequestHeader struct {
	Command apicommands.CommandType `json:"command"`
}

type ApiResponseHeader struct {
	Command apicommands.CommandType `json:"command"`
	Status  ResponseStatusType      `json:"status"`
	Error   string                  `json:"error,omitempty"`
}

func NewApiResponseHeader(responseType apicommands.CommandType, status ResponseStatusType, err string) ApiResponseHeader {
	return ApiResponseHeader{Command: responseType, Status: status, Error: err}
}

func (r *ApiResponseHeader) Stringify() ([]byte, error) {
	return json.Marshal(r)
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

func safeSend(send chan []byte, msg []byte) error {
	var err error
	defer func() {
		if x := recover(); x != nil {
			err = fmt.Errorf("Unable to send msg: %v", x)
		}
	}()
	send <- msg
	return err
}

func (h *MessagesHub) runMessagesHub() {

	requestHandlersMap := make(map[apicommands.CommandType]messageHandler)

	for {
		select {
		// Register a new handler
		case messageHandler := <-h.registerHandler:
			_, ok := requestHandlersMap[messageHandler.requestType]
			if ok { // The is already a handler with the same command request type!
				h.log("REGISTER MESSAGE HANDLER ERROR! >>>> the request handler ", messageHandler.requestType, " is already registered!")
			} else {
				requestHandlersMap[messageHandler.requestType] = messageHandler.handler
				h.log("New message handler with id ", messageHandler.requestType, " has been registered. Now there are ", len(requestHandlersMap))
			}
			// Unregister a handler
		case messageHandler := <-h.unregisterHandler:
			_, ok := requestHandlersMap[messageHandler.requestType]
			if ok { // a handler with the ID to remove was found
				delete(requestHandlersMap, messageHandler.requestType)
				h.log("Message handler with id ", messageHandler.requestType, " was unregistered. Now there are ", len(requestHandlersMap))
			} else {
				h.log("UNREGISTER MESSAGE HANDLER ERROR! >>>> the request handler ", messageHandler.requestType, " is not registered!")
			}
			// handle an incoming message depending on it's request type
		case clientMessage := <-h.incomingMessage:
			var cmm ApiRequestHeader
			err := json.Unmarshal(clientMessage.fromMessage, &cmm)
			if err != nil {
				h.log("Error unmarshalling the event command ", string(clientMessage.fromMessage), ": ", err)
				badRequestResponse := newBadRequestApiResponse()
				response, _ := badRequestResponse.Stringify()
				clientMessage.setResponseMessage(response)
				Send(clientMessage)
			} else {

				handler, ok := requestHandlersMap[cmm.Command]
				if ok {
					h.log("The request command ", cmm.Command, " will be processed.")
					rc := NewCommandRequest(cmm.Command, clientMessage.fromMessage)
					response := handler(rc)
					clientMessage.setResponseMessage(response)
					Send(clientMessage)
				} else {
					// No handler with the command id has been registered
					h.log("The request command ", cmm.Command, " is not supported.")
					notSupportedResponse := newNotSupportedStatusAPIResponse(cmm.Command)
					response, _ := notSupportedResponse.Stringify()
					clientMessage.setResponseMessage(response)
					Send(clientMessage)
				}
			}
		}
	}
}
