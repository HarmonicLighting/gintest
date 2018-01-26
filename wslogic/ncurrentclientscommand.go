package wslogic

import (
	"container/list"
	"encoding/json"
	"local/gintest/apicommands"
	"log"
)

type ApiNClients struct {
	Number int `json:"number"`
}

type NCurrentClientsResponse struct {
	ApiResponseHeader
	ApiNClients
}

const (
	errorNCurrentClientsStatus ResponseStatusType = -1
)

func NewNCurrentClientsResponse(nClients int) NCurrentClientsResponse {
	return NCurrentClientsResponse{
		ApiResponseHeader: ApiResponseHeader{
			Command: apicommands.ServerNConnectionsPush,
		},
		ApiNClients: ApiNClients{Number: nClients},
	}
}

func (r *NCurrentClientsResponse) Stringify() ([]byte, error) {
	return json.Marshal(r)
}

func (h *ConnectionsHub) requestNCurrentClientsCommand(request CommandRequest) RawResponseData {
	h.incomingNCurrentClientsCommand <- request
	return <-request.response
}

func processNCurrentClientsCommand(connectionsList *list.List) RawResponseData {
	responseStruct := NewNCurrentClientsResponse(connectionsList.Len())
	bytes, err := responseStruct.Stringify()
	if err != nil {
		log.Println("ERROR processNCurrentClientsCommand >>>> Couldn't stringify the response structure!")
	}
	return bytes
}
