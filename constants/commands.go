package constants

import (
	"encoding/json"
	"errors"
	"log"
)

type BroadcastHandle func([]byte)

type CommandResponseType int
type CommandRequestType int

const (
	PIDListCommandRequest CommandRequestType = iota
	PIDUpdateCommandRequest
	NCurrentClientsCommandRequest
)

const (
	PIDListCommandResponse CommandResponseType = iota
	PIDUpdateCommandResponse
	NCurrentClientsCommandResponse
)

type CommandResponse struct {
	Response []byte
	Err      error
}

func NewCommandResponse(response []byte, err error) CommandResponse {
	return CommandResponse{Response: response, Err: err}
}

type CommandRequest struct {
	Command  CommandRequestType
	Data     []byte
	Response chan CommandResponse
}

func NewCommandRequest(command CommandRequestType, data []byte) CommandRequest {
	return CommandRequest{Command: command, Data: data, Response: make(chan CommandResponse)}
}

func (cr *CommandRequest) SendCommandResponse(response CommandResponse) {
	if cr.Response == nil {
		log.Println(">> COMMAND REQUEST ERROR: Response channel is Nil")
		return
	}
	cr.Response <- response
}

type Request interface {
	Parse(data []byte) error
}

type Response interface {
	Stringify() ([]byte, error)
}

type CommandRequestHeader struct {
	Command CommandRequestType `json:"command"`
}

// TO-IMPLEMENT
type CommandResponseHeader struct {
	Command CommandResponseType `json:"command"`
	Status  int                 `json:"status"`
	Error   string              `json:"error"`
}

type ApiPid struct {
	Name   string  `json:"name"`
	Index  int     `json:"index"`
	Period float32 `json:"period"`
}

func NewApiPID(name string, index int, period float32) ApiPid {
	return ApiPid{Name: name, Index: index, Period: period}
}

type PIDListResponse struct {
	CommandResponseHeader
	List []ApiPid `json:"pids"`
}

func NewPIDListResponse(list []ApiPid) PIDListResponse {
	return PIDListResponse{CommandResponseHeader: CommandResponseHeader{Command: PIDListCommandResponse}, List: list}
}

func (r PIDListResponse) Stringify() ([]byte, error) {
	return json.Marshal(r)
}

type ApiUpdate struct {
	Index     int     `json:"index"`
	Timestamp int64   `json:"timestamp"`
	Value     float32 `json:"value"`
}

type PIDUpdateResponse struct {
	CommandResponseHeader
	ApiUpdate
}

func NewPIDUpdateResponse(index int, timestamp int64, value float32) PIDUpdateResponse {
	return PIDUpdateResponse{CommandResponseHeader: CommandResponseHeader{Command: PIDUpdateCommandResponse}, ApiUpdate: ApiUpdate{Index: index, Timestamp: timestamp, Value: value}}
}

func (r *PIDUpdateResponse) Stringify() ([]byte, error) {
	return json.Marshal(r)
}

func (r *PIDUpdateResponse) Parse(data []byte) error {
	if !json.Valid(data) {
		return errors.New("The data to parse is not a valid JSON encoding")
	}
	return json.Unmarshal(data, r)
}

type ApiNClients struct {
	Number int `json:"number"`
}

type NCurrentClientsResponse struct {
	CommandResponseHeader
	ApiNClients
}

func NewNCurrentClientsResponse(nClients int) NCurrentClientsResponse {
	return NCurrentClientsResponse{CommandResponseHeader: CommandResponseHeader{Command: NCurrentClientsCommandResponse}, ApiNClients: ApiNClients{Number: nClients}}
}

func (r *NCurrentClientsResponse) Stringify() ([]byte, error) {
	return json.Marshal(r)
}
