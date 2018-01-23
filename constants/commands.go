package constants

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
)

type BroadcastHandle func([]byte)

type StatusType int

const (
	BadRequestStatus          StatusType = -2
	RequestNotSupportedStatus StatusType = -1
)

func NewNotSupportedStatusCommandResponse(command CommandRequestType) CommandResponseHeader {
	return NewCommandResponseHeader(NotSupportedCommandResponse, RequestNotSupportedStatus, fmt.Sprint("The Command Request ", command, " is not supported"))
}

func NewBadRequestCommandResponse() CommandResponseHeader {
	return NewCommandResponseHeader(NotSupportedCommandResponse, BadRequestStatus, fmt.Sprint("The Command Request is unrecognizable"))
}

type CommandResponseType int

const NotSupportedCommandResponse CommandResponseType = -1
const (
	PIDListCommandResponse CommandResponseType = iota
	PIDUpdateCommandResponse
	NCurrentClientsCommandResponse
)

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

type Request interface {
	Parse(data []byte) error
}

type Response interface {
	Stringify() ([]byte, error)
}

type CommandRequestHeader struct {
	Command CommandRequestType `json:"command"`
}

type CommandResponseHeader struct {
	Command CommandResponseType `json:"command"`
	Status  StatusType          `json:"status"`
	Error   string              `json:"error,omitempty"`
}

func NewCommandResponseHeader(responseType CommandResponseType, status StatusType, err string) CommandResponseHeader {
	return CommandResponseHeader{Command: responseType, Status: status, Error: err}
}

func (r *CommandResponseHeader) Stringify() ([]byte, error) {
	return json.Marshal(r)
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
