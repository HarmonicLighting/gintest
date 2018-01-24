package wslogic

import (
	"container/list"
	"encoding/json"
	"local/gintest/commons"
	"log"
)

type ApiNClients struct {
	Number int `json:"number"`
}

type NCurrentClientsResponse struct {
	commons.ApiResponseHeader
	ApiNClients
}

const (
	errorNCurrentClientsStatus commons.StatusType = -1
)

func NewNCurrentClientsResponse(nClients int) NCurrentClientsResponse {
	return NCurrentClientsResponse{ApiResponseHeader: commons.ApiResponseHeader{Command: commons.NCurrentClientsCommandResponse}, ApiNClients: ApiNClients{Number: nClients}}
}

func (r *NCurrentClientsResponse) Stringify() ([]byte, error) {
	return json.Marshal(r)
}

func (h *ConnectionsHub) requestNCurrentClientsCommand(request commons.CommandRequest) commons.RawResponseData {
	h.incomingNCurrentClientsCommand <- request
	return <-request.Response
}

func processNCurrentClientsCommand(connectionsList *list.List) commons.RawResponseData {
	responseStruct := NewNCurrentClientsResponse(connectionsList.Len())
	bytes, err := responseStruct.Stringify()
	if err != nil {
		log.Println("ERROR processNCurrentClientsCommand >>>> Couldn't stringify the response structure!")
	}
	return bytes
}
