package pid

import (
	"encoding/json"
	"local/gintest/commons"
	"local/gintest/services/db"
	"log"
	"time"
)

type ApiPidListResponse struct {
	commons.ApiResponseHeader
	List []PidData `json:"pids"`
}

func NewApiPidListResponse(list []PidData) ApiPidListResponse {
	return ApiPidListResponse{
		ApiResponseHeader: commons.ApiResponseHeader{
			Command: commons.PidListCommandResponse,
		},
		List: list}
}

func (r ApiPidListResponse) Stringify() ([]byte, error) {
	return json.Marshal(r)
}

func getApiPidsList(dummyTickersMap map[int]*DummyPIDTicker) []PidData {
	pids := make([]PidData, len(dummyTickersMap))
	i := 0
	for _, dummyTicker := range dummyTickersMap {
		pids[i] = PidData{
			PidStaticData:  dummyTicker.pidData,
			PidDynamicData: dummyTicker.getCurrentData(),
		}
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
	response := RequestPidList(request)
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
		pids[i] = &db.DBPid{
			Name:   pid.Name,
			Pid:    pid.Index,
			Period: time.Duration(pid.SamplePeriod),
		}
	}
	dbpids.Timestamp = t
	dbpids.Pids = pids
	dbase.InsertPids(dbpids)
}
