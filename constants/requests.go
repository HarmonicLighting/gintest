package constants

import "log"

type CommandRequestType int

const (
	PIDListCommandRequest CommandRequestType = iota
	PIDUpdateCommandRequest
	NCurrentClientsCommandRequest
)

type RawCommandResponse []byte

type CommandRequest struct {
	Command  CommandRequestType
	Data     []byte
	Response chan RawCommandResponse
}

func NewCommandRequest(command CommandRequestType, data []byte) CommandRequest {
	return CommandRequest{Command: command, Data: data, Response: make(chan RawCommandResponse)}
}

func (cr *CommandRequest) SendCommandResponse(response RawCommandResponse) {
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
