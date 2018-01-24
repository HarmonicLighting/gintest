package pid

import (
	"encoding/json"
	"local/gintest/commons"
	"local/gintest/services/db"
	"log"
	"time"
)

type ApiPid struct {
	Name   string  `json:"name"`
	Index  int     `json:"index"`
	Period float32 `json:"period"`
}

func NewApiPid(name string, index int, period float32) ApiPid {
	return ApiPid{Name: name, Index: index, Period: period}
}

type ApiPidListResponse struct {
	commons.ApiResponseHeader
	List []ApiPid `json:"pids"`
}

func NewApiPidListResponse(list []ApiPid) ApiPidListResponse {
	return ApiPidListResponse{ApiResponseHeader: commons.ApiResponseHeader{Command: commons.PidListCommandResponse}, List: list}
}

func (r ApiPidListResponse) Stringify() ([]byte, error) {
	return json.Marshal(r)
}

func getApiPidsList(dummyTickersMap map[int]*DummyPIDTicker) []ApiPid {
	pids := make([]ApiPid, len(dummyTickersMap))
	i := 0
	for _, pid := range dummyTickersMap {
		pids[i] = NewApiPid(pid.name, pid.index, float32(pid.period))
		i++
	}
	return pids
}

func processPIDListCommand(dummyTickersMap map[int]*DummyPIDTicker) ([]byte, error) {
	pids := getApiPidsList(dummyTickersMap)
	responseStruct := NewApiPidListResponse(pids)
	return responseStruct.Stringify()
}

func RequestPIDListEventStruct() ApiPidListResponse {
	request := commons.NewCommandRequest(commons.ApiPidListCommandRequest, []byte{})
	response := RequestCommand(request)
	var listResponse ApiPidListResponse
	err := json.Unmarshal(response, &listResponse)
	if err != nil {
		log.Println("On pid RequestApiPids: Error parsing the List Event : ", err)
	}
	return listResponse
}

func SavePidsToDb(t int64) {

	d, err := dbase.Copy()
	if err != nil {
		log.Println("Error copying session")
		return
	}
	defer d.Close()

	listEvent := RequestPIDListEventStruct()
	log.Println("Obtained ", len(listEvent.List), " pids to insert to the DB")
	var dbpids db.DBPids
	pids := make([]*db.DBPid, len(listEvent.List))
	for i, pid := range listEvent.List {
		pids[i] = &db.DBPid{Name: pid.Name, Pid: pid.Index, Period: time.Duration(pid.Period)}
	}
	dbpids.Timestamp = t
	dbpids.Pids = pids
	dbase.InsertPids(dbpids)
}
