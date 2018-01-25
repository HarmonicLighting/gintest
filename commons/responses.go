package commons

import (
	"encoding/json"
	"fmt"
)

type StatusType int

const (
	BadRequestStatus          StatusType = -2
	RequestNotSupportedStatus StatusType = -1
)

type CommandResponseType int

const NotSupportedCommandResponse CommandResponseType = -1
const (
	PidListCommandResponse CommandResponseType = iota
	PidUpdateCommandResponse
	NCurrentClientsCommandResponse
	PidListUpdateCommandResponse
)

type ApiResponse interface {
	Stringify() ([]byte, error)
}

type ApiResponseHeader struct {
	Command CommandResponseType `json:"command"`
	Status  StatusType          `json:"status"`
	Error   string              `json:"error,omitempty"`
}

func NewNotSupportedStatusApiResponse(command CommandRequestType) ApiResponseHeader {
	return NewApiResponseHeader(NotSupportedCommandResponse, RequestNotSupportedStatus, fmt.Sprint("The Command Request ", command, " is not supported"))
}

func NewBadRequestApiResponse() ApiResponseHeader {
	return NewApiResponseHeader(NotSupportedCommandResponse, BadRequestStatus, fmt.Sprint("The Command Request is unrecognizable"))
}

func NewApiResponseHeader(responseType CommandResponseType, status StatusType, err string) ApiResponseHeader {
	return ApiResponseHeader{Command: responseType, Status: status, Error: err}
}

func (r *ApiResponseHeader) Stringify() ([]byte, error) {
	return json.Marshal(r)
}
