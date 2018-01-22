package constants

import (
	"encoding/json"
	"errors"
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

func NewCommandRequest(command CommandType, data []byte) CommandRequest {
	return CommandRequest{Command: command, Data: data, Response: make(chan CommandResponse)}
}

type CommandType int

const (
	PIDListCommand CommandType = iota
	PIDUpdateCommand
)

type Command interface {
	Stringify() ([]byte, string)
	Parse(data []byte) error
}

type EventCommand struct {
	Command CommandType `json:"command"`
}

type ApiPid struct {
	Name   string  `json:"name"`
	Index  int     `json:"index"`
	Period float32 `json:"period"`
}

func NewApiPID(name string, index int, period float32) ApiPid {
	return ApiPid{Name: name, Index: index, Period: period}
}

type PIDListEvent struct {
	EventCommand
	List []ApiPid `json:"pids"`
}

func NewPIDListEvent(list []ApiPid) PIDListEvent {
	return PIDListEvent{EventCommand: EventCommand{Command: PIDListCommand}, List: list}
}

func (e PIDListEvent) Stringify() ([]byte, error) {
	return json.Marshal(e)
}

func (e *PIDListEvent) Parse(data []byte) error {
	if !json.Valid(data) {
		return errors.New("The data to parse is not a valid JSON encoding")
	}
	return json.Unmarshal(data, e)
}

type ApiUpdate struct {
	Index     int     `json:"index"`
	Timestamp int64   `json:"timestamp"`
	Value     float32 `json:"value"`
}

type PIDUpdateEvent struct {
	EventCommand
	ApiUpdate
}

func NewPIDUpdateEvent(index int, timestamp int64, value float32) PIDUpdateEvent {
	return PIDUpdateEvent{EventCommand: EventCommand{Command: PIDUpdateCommand}, ApiUpdate: ApiUpdate{Index: index, Timestamp: timestamp, Value: value}}
}

func (e *PIDUpdateEvent) Stringify() ([]byte, error) {
	return json.Marshal(e)
}

func (e *PIDUpdateEvent) Parse(data []byte) error {
	if !json.Valid(data) {
		return errors.New("The data to parse is not a valid JSON encoding")
	}
	return json.Unmarshal(data, e)
}
