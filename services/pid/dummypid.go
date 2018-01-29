package pid

import (
	"encoding/json"
	"errors"
	"fmt"
	"local/gintest/apicommands"
	"local/gintest/services/db"
	"local/gintest/services/dbheap"
	"local/gintest/wslogic"
	"log"
	"math"
	"math/rand"
	"time"
)

const (
	pidTickers              = 1000
	pidTickersMinDuration   = time.Millisecond * 100
	pidTickersMaxDuration   = time.Second * 2
	pidTickersRangeDuration = pidTickersMaxDuration - pidTickersMinDuration
)

var (
	pidIndexCounter int32
)

type ApiUpdate struct {
	Index     int      `json:"index"`
	Timestamp int64    `json:"timestamp"`
	Value     float32  `json:"value"`
	State     PidState `json:"state"`
}

type ApiPidUpdateResponse struct {
	wslogic.ApiResponseHeader
	ApiUpdate
}

func NewApiPidUpdateResponse(update ApiUpdate) ApiPidUpdateResponse {
	return ApiPidUpdateResponse{
		ApiResponseHeader: wslogic.ApiResponseHeader{
			Command: apicommands.ServerSignalUpdatePush,
		},
		ApiUpdate: update,
	}
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

type DummyPidTickerFunc func(PidData)

type DummyPIDTicker struct {
	pidData           PidStaticData
	onTick            DummyPidTickerFunc
	stop              chan struct{}
	reportCurrentData chan chan PidDynamicData
	valueUpdated      chan struct{}
	isRunning         bool
}

func NewDummyPIDTicker(pidData PidStaticData, onTick DummyPidTickerFunc) *DummyPIDTicker {
	return &DummyPIDTicker{
		pidData:           pidData,
		onTick:            onTick,
		stop:              make(chan struct{}),
		reportCurrentData: make(chan chan PidDynamicData),
		valueUpdated:      make(chan struct{}, 1),
		isRunning:         false,
	}
}

func (t *DummyPIDTicker) getValueAndState() (float32, PidState) {
	randfloat := rand.Float32() * 100
	var state PidState
	var randBool float32

	if randfloat < 50 {
		randBool = 0
		state = OkPidState
	} else {
		randBool = 1
		state = BadPidState
	}
	switch t.pidData.Type {
	case AnalogicalPidType:
		return randfloat, state
	case DiscretePidType:
		return float32(math.Ceil(float64(randfloat / 10))), state
	case DigitalPidType:
		return randBool, state
	default:
		return randfloat, state
	}
}

func (t *DummyPIDTicker) getCurrentData() PidDynamicData {
	if !t.isRunning {
		return PidDynamicData{State: InternalErrorPidState}
	}
	currentDataCh := make(chan PidDynamicData)
	t.reportCurrentData <- currentDataCh
	return <-currentDataCh
}

func (t *DummyPIDTicker) getCurrentDataIfUpdated() (PidDynamicData, bool) {
	if !t.isRunning {
		return PidDynamicData{State: InternalErrorPidState}, false
	}

	select {
	case <-t.valueUpdated: // There was already an updated value
		currentDataCh := make(chan PidDynamicData)
		t.reportCurrentData <- currentDataCh
		return <-currentDataCh, true
	default: // No new value
		return PidDynamicData{}, false
	}
}

func (t *DummyPIDTicker) log(v ...interface{}) {
	if debugging {
		text := fmt.Sprint(v...)
		prefix := fmt.Sprint("<Ticker ", t.pidData.Index, "> ~ ")
		if debugWithTimeStamp {
			prefix = time.Now().Format(time.StampMicro) + " " + prefix
		}
		log.Println(prefix, text)
	}
}

func (t *DummyPIDTicker) logf(format string, v ...interface{}) {
	if debugging {
		prefix := fmt.Sprint("<Ticker ", t.pidData.Index, "> ~ ")
		if debugWithTimeStamp {
			prefix = time.Now().Format(time.StampMicro) + " " + prefix
		}
		log.Printf(prefix+format, v...)
	}
}

func (t *DummyPIDTicker) Stop() {
	if t.stop != nil {
		t.stop <- struct{}{}
	}
}

func (t *DummyPIDTicker) Launch() {
	if t.isRunning {
		return
	}

	if t.stop == nil {
		t.stop = make(chan struct{})
	}

	t.isRunning = true

	ticker := time.NewTicker(t.pidData.SamplePeriod)
	logTicker := time.NewTicker(time.Second * 30)

	go func() {

		defer func() {
			ticker.Stop()
			logTicker.Stop()
			t.isRunning = false
		}()

		nTicks := 0
		nReports := 0
		var data PidDynamicData
		data.LastUpdated = time.Now().UnixNano()
		data.State = NeverUpdatedPidState
		for {
			select {

			// Stop signal received: exit the go routine
			case <-t.stop:
				t.log("Stopping the Dummy Ticker ", t.pidData.Name)
				return

				// Log time
			case <-logTicker.C:
				t.log("The Dummy PID has ticked ", nTicks, " times during this period.\n\t\t\t\t\t\tTotal data reports: ", nReports)
				nTicks = 0

				// Ticker signal, continue normal ticking
			case now := <-ticker.C:
				nTicks++
				//t.log("Got a tick for Dummy Ticker ", t.pidData.Name)
				data.LastUpdated = now.UnixNano()
				data.Value, data.State = t.getValueAndState()
				select {
				case t.valueUpdated <- struct{}{}:
					//t.log("Flagging an updated value for dummy pid ", t.pidData.Name)
					data.Updates = 1
				default:
					data.Updates++
				}
				t.onTick(PidData{PidStaticData: t.pidData, PidDynamicData: data})
			case channel := <-t.reportCurrentData:
				//t.log("Reporting current dynamic PID Data of ", t.pidData.Name)
				channel <- data
				nReports++
			}
		}
	}()
}

func standardTickHandler(data PidData) {
	d, err := dbheap.GetSession()
	if err != nil {
		log.Println("Error getting session ", err)
		return
	}
	defer d.Close()
	err = d.ClientSession.InsertSamples(&db.DBSample{Pid: data.Index, Value: data.Value, Timestamp: data.LastUpdated})
	if err != nil {
		log.Println("Error inserting sample ", err)
	}
}
