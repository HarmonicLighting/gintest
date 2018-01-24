package commons

import "log"

type CommandRequestType int

const (
	ApiPidListCommandRequest CommandRequestType = iota
	ApiPidUpdateCommandRequest
	ApiNCurrentClientsCommandRequest
)

type RawResponseData []byte
type RawRequestData []byte

type CommandRequest struct {
	Command  CommandRequestType
	Data     RawRequestData
	Response chan RawResponseData
}

func NewCommandRequest(command CommandRequestType, data []byte) CommandRequest {
	return CommandRequest{Command: command, Data: data, Response: make(chan RawResponseData)}
}

func (cr *CommandRequest) SendCommandResponse(response RawResponseData) {
	if cr.Response == nil {
		log.Println(">> COMMAND REQUEST ERROR: Response channel is Nil")
		return
	}
	cr.Response <- response
}

type ApiRequest interface {
	Parse(data []byte) error
}

type ApiRequestHeader struct {
	Command CommandRequestType `json:"command"`
}
