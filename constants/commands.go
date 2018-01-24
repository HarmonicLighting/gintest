package constants

import (
	"encoding/json"
	"errors"
)

type BroadcastHandle func([]byte)

type ApiPid struct {
	Name   string  `json:"name"`
	Index  int     `json:"index"`
	Period float32 `json:"period"`
}

func NewApiPid(name string, index int, period float32) ApiPid {
	return ApiPid{Name: name, Index: index, Period: period}
}

type PIDListResponse struct {
	ApiResponseHeader
	List []ApiPid `json:"pids"`
}

func NewPIDListResponse(list []ApiPid) PIDListResponse {
	return PIDListResponse{ApiResponseHeader: ApiResponseHeader{Command: PIDListCommandResponse}, List: list}
}

func (r PIDListResponse) Stringify() ([]byte, error) {
	return json.Marshal(r)
}

type ApiUpdate struct {
	Index     int     `json:"index"`
	Timestamp int64   `json:"timestamp"`
	Value     float32 `json:"value"`
}

type ApiPidUpdateResponse struct {
	ApiResponseHeader
	ApiUpdate
}

func NewPIDUpdateResponse(index int, timestamp int64, value float32) ApiPidUpdateResponse {
	return ApiPidUpdateResponse{ApiResponseHeader: ApiResponseHeader{Command: PIDUpdateCommandResponse}, ApiUpdate: ApiUpdate{Index: index, Timestamp: timestamp, Value: value}}
}

func (r *ApiPidUpdateResponse) Stringify() ([]byte, error) {
	return json.Marshal(r)
}

func (r *ApiPidUpdateResponse) Parse(data []byte) error {
	if !json.Valid(data) {
		return errors.New("The data to parse is not a valid JSON encoding")
	}
	return json.Unmarshal(data, r)
}
