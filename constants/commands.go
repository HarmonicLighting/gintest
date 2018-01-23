package constants

import (
	"encoding/json"
	"errors"
)

type BroadcastHandle func([]byte)

type CommandType int

const (
	PIDListCommand CommandType = iota
	PIDUpdateCommand
	NCurrentClientsCommand
)

type CommandResponse struct {
	Response []byte
	Err      error
}

func NewCommandResponse(response []byte, err error) CommandResponse {
	return CommandResponse{Response: response, Err: err}
}

type CommandRequest struct {
	Command  CommandType
	Data     []byte
	Response chan CommandResponse
}

func (cr *CommandRequest) SendCommandResponse(response CommandResponse) {
	cr.Response <- response
}

func NewCommandRequest(command CommandType, data []byte) CommandRequest {
	return CommandRequest{Command: command, Data: data, Response: make(chan CommandResponse)}
}

type Command interface {
	Stringify() ([]byte, string)
	Parse(data []byte) error
}

type CommandHeader struct {
	Command CommandType `json:"command"`
}

// TO-IMPLEMENT
type CommandResponseHeader struct {
	Command CommandType `json:"command"`
	Status  int         `json:"status"`
	Error   string      `json:"error"`
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
	CommandHeader
	List []ApiPid `json:"pids"`
}

func NewPIDListResponse(list []ApiPid) PIDListResponse {
	return PIDListResponse{CommandHeader: CommandHeader{Command: PIDListCommand}, List: list}
}

func (r PIDListResponse) Stringify() ([]byte, error) {
	return json.Marshal(r)
}

func (r *PIDListResponse) Parse(data []byte) error {
	if !json.Valid(data) {
		return errors.New("The data to parse is not a valid JSON encoding")
	}
	return json.Unmarshal(data, r)
}

type ApiUpdate struct {
	Index     int     `json:"index"`
	Timestamp int64   `json:"timestamp"`
	Value     float32 `json:"value"`
}

type PIDUpdateResponse struct {
	CommandHeader
	ApiUpdate
}

func NewPIDUpdateResponse(index int, timestamp int64, value float32) PIDUpdateResponse {
	return PIDUpdateResponse{CommandHeader: CommandHeader{Command: PIDUpdateCommand}, ApiUpdate: ApiUpdate{Index: index, Timestamp: timestamp, Value: value}}
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
	CommandHeader
	ApiNClients
}

func NewNCurrentClientsResponse(nClients int) NCurrentClientsResponse {
	return NCurrentClientsResponse{CommandHeader: CommandHeader{Command: NCurrentClientsCommand}, ApiNClients: ApiNClients{Number: nClients}}
}

func (r *NCurrentClientsResponse) Stringify() ([]byte, error) {
	return json.Marshal(r)
}

func (r *NCurrentClientsResponse) Parse(data []byte) error {
	if !json.Valid(data) {
		return errors.New("The data to parse is not a valid JSON encoding")
	}
	return json.Unmarshal(data, r)
}
