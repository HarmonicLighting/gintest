package pid

import (
	"encoding/json"
	"local/gintest/constants"
	"local/gintest/services/db"
	"log"
	"time"
)

func getApiPidsList(dummyTickersMap map[int]*DummyPIDTicker) []constants.ApiPid {
	pids := make([]constants.ApiPid, len(dummyTickersMap))
	i := 0
	for _, pid := range dummyTickersMap {
		pids[i] = constants.NewApiPid(pid.name, pid.index, float32(pid.period))
		i++
	}
	return pids
}

func processPIDListCommand(dummyTickersMap map[int]*DummyPIDTicker) ([]byte, error) {
	pids := getApiPidsList(dummyTickersMap)
	responseStruct := constants.NewPIDListResponse(pids)
	return responseStruct.Stringify()
}

func RequestPIDListEventStruct() constants.PIDListResponse {
	request := constants.NewCommandRequest(constants.PIDListCommandRequest, []byte{})
	response := RequestCommand(request)
	var listResponse constants.PIDListResponse
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
