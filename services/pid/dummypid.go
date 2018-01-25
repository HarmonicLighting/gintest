package pid

import (
	"encoding/json"
	"errors"
	"fmt"
	"local/gintest/commons"
	"local/gintest/services/db"
	"local/gintest/wslogic"
	"log"
	"math"
	"math/rand"
	"time"
)

var (
	pidIndexCounter int32
)

var (
	dbase *db.DB
)

type ApiUpdate struct {
	Index     int      `json:"index"`
	Timestamp int64    `json:"timestamp"`
	Value     float32  `json:"value"`
	State     PidState `json:"state"`
}

type ApiPidUpdateResponse struct {
	commons.ApiResponseHeader
	ApiUpdate
}

func NewApiPidUpdateResponse(update ApiUpdate) ApiPidUpdateResponse {
	return ApiPidUpdateResponse{
		ApiResponseHeader: commons.ApiResponseHeader{
			Command: commons.PidUpdateCommandResponse,
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
	ticker            *time.Ticker
	onTick            DummyPidTickerFunc
	stop              chan struct{}
	reportCurrentData chan chan PidDynamicData
	isRunning         bool
}

func NewDummyPIDTicker(pidData PidStaticData, onTick DummyPidTickerFunc) *DummyPIDTicker {
	ticker := &DummyPIDTicker{
		pidData:           pidData,
		ticker:            nil,
		onTick:            onTick,
		stop:              make(chan struct{}),
		reportCurrentData: make(chan chan PidDynamicData),
		isRunning:         false,
	}
	return ticker
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

	t.ticker = time.NewTicker(t.pidData.SamplePeriod)

	go func() {

		defer func() {
			t.ticker.Stop()
			t.ticker = nil
			t.isRunning = false
		}()

		var data PidDynamicData
		data.LastUpdated = time.Now().UnixNano()
		data.State = NeverUpdatedPidState
		for {
			select {

			// Stop signal received: exit the go routine
			case <-t.stop:
				t.log("Stopping the Dummy Ticker ", t.pidData.Name)
				return

				// Ticker signal, continue normal ticking
			case now := <-t.ticker.C:
				t.log("Got a tick for Dummy Ticker ", t.pidData.Name)
				data.LastUpdated = now.UnixNano()
				data.Updates++
				data.Value, data.State = t.getValueAndState()
				t.onTick(PidData{PidStaticData: t.pidData, PidDynamicData: data})
			case channel := <-t.reportCurrentData:
				t.log("Reporting current dynamic PID Data of ", t.pidData.Name)
				channel <- data
				data.Updates = 0
			}
		}
	}()
}

func standardTickHandler(data PidData) {
	update := ApiUpdate{Index: data.Index, Value: data.Value, Timestamp: data.LastUpdated, State: data.State}
	event := NewApiPidUpdateResponse(update)
	message, err := event.Stringify()
	if err != nil {
		log.Println("Error stringifying: ", err)
		return
	}
	log.Println("Broadcasting event ", string(message), " by Dummy Ticker ", data.Name)
	wslogic.Broadcast(message)
	log.Println("Saving sample to DB")
	d, err := dbase.Copy()
	if err != nil {
		log.Println("Error copying the db session: ", err)
		return
	}
	defer d.Close()
	d.InsertSamples(&db.DBSample{Pid: data.Index, Value: data.Value, Timestamp: data.LastUpdated})
}
