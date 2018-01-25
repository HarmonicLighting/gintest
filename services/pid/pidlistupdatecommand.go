package pid

import (
	"encoding/json"
	"errors"
	"local/gintest/commons"
)

type PidIndexedDynamicData struct {
	Index int `json:"index"`
	PidDynamicData
}

type ApiPidListUpdateResponse struct {
	commons.ApiResponseHeader
	List []PidIndexedDynamicData `json:"pids"`
}

func NewApiPidListUpdateResponse(list []PidIndexedDynamicData) ApiPidListUpdateResponse {
	return ApiPidListUpdateResponse{
		ApiResponseHeader: commons.ApiResponseHeader{
			Command: commons.PidListUpdateCommandResponse,
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
		pid := PidIndexedDynamicData{
			Index:          dummyTicker.pidData.Index,
			PidDynamicData: dummyTicker.getCurrentData(),
		}
		if pid.Updates > 0 {
			pids = append(pids, pid)
		}
	}
	return pids
}

func processPIDListUpdateCommand(dummyTickersMap map[int]*DummyPIDTicker) ([]byte, error) {
	pids := getPidIndexedDynamicDataList(dummyTickersMap)
	//log.Println("Updating list with ", len(pids), " signals")
	if len(pids) < 1 {
		return nil, errors.New("There are no updates to notify")
	}
	responseStruct := NewApiPidListUpdateResponse(pids)
	return responseStruct.Stringify()
}
