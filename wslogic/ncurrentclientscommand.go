package wslogic

import (
	"container/list"
	"encoding/json"
	"local/gintest/constants"
	"log"
)

type ApiNClients struct {
	Number int `json:"number"`
}

type NCurrentClientsResponse struct {
	constants.CommandResponseHeader
	ApiNClients
}

func NewNCurrentClientsResponse(nClients int) NCurrentClientsResponse {
	return NCurrentClientsResponse{CommandResponseHeader: constants.CommandResponseHeader{Command: constants.NCurrentClientsCommandResponse}, ApiNClients: ApiNClients{Number: nClients}}
}

func (r *NCurrentClientsResponse) Stringify() ([]byte, error) {
	return json.Marshal(r)
}

const (
	errorNCurrentClientsStatus constants.StatusType = -1
)

func (h *Hub) requestNCurrentClientsCommand(request constants.CommandRequest) constants.RawCommandResponse {
	h.incomingNCurrentClientsCommand <- request
	return <-request.Response
}

func processNCurrentClientsCommand(connectionsList *list.List) constants.RawCommandResponse {
	responseStruct := NewNCurrentClientsResponse(connectionsList.Len())
	bytes, err := responseStruct.Stringify()
	if err != nil {
		log.Println("ERROR processPIDListCommand >>>> Couldn't stringify the response structure!")
	}
	return bytes
}
