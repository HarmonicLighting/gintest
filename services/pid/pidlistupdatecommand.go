package pid

import (
	"encoding/json"
	"errors"
	"local/gintest/apicommands"
	"local/gintest/wslogic"
)

type PidIndexedDynamicData struct {
	Index int `json:"index"`
	PidDynamicData
}

type ApiPidListUpdateResponse struct {
	wslogic.ApiResponseHeader
	List []PidIndexedDynamicData `json:"pids"`
}

func NewApiPidListUpdateResponse(list []PidIndexedDynamicData) ApiPidListUpdateResponse {
	return ApiPidListUpdateResponse{
		ApiResponseHeader: wslogic.ApiResponseHeader{
			Command: apicommands.ServerSignalUpdateListPush,
		},
		List: list}
}

func (r ApiPidListUpdateResponse) Stringify() ([]byte, error) {
	return json.Marshal(r)
}

func getPidIndexedDynamicDataList(dummyTickersMap map[int]*DummyPIDTicker) []PidIndexedDynamicData {
	var pids []PidIndexedDynamicData
	//:= make([]PidIndexedDynamicData, len(dummyTickersMap))
	for _, dummyTicker := range dummyTickersMap {
		if data, ok := dummyTicker.getCurrentDataIfUpdated(); ok {
			pid := PidIndexedDynamicData{
				Index:          dummyTicker.pidData.Index,
				PidDynamicData: data,
			}
			pids = append(pids, pid)
		}
	}
	return pids
}

func processPIDListUpdateCommand(dummyTickersMap map[int]*DummyPIDTicker) ([]byte, error) {
	pids := getPidIndexedDynamicDataList(dummyTickersMap)
	//log.Println("Updating list with ", len(pids), " signals")
	npids := len(pids)
	responseStruct := NewApiPidListUpdateResponse(pids)
	data, _ := responseStruct.Stringify()
	if npids < 1 {
		return data, errors.New("There are no updates to notify")
	}
	//log.Println("There are ", npids, " pids to update")
	return data, nil
}
