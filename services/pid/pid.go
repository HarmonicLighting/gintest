package pid

import (
	"fmt"
	"local/gintest/commons"
	"local/gintest/services/db"
	"local/gintest/wslogic"
	"log"
	"math/rand"
	"sync/atomic"
	"time"
)

const (
	debugging          = commons.Debugging
	debugWithTimeStamp = commons.DebugWithTimeStamp

	pidTickers              = 10
	pidTickersMinDuration   = time.Millisecond * 100
	pidTickersMaxDuration   = time.Second * 5
	pidTickersRangeDuration = pidTickersMaxDuration - pidTickersMinDuration
)

type PidType int

const (
	AnalogicalPidType PidType = iota
	DiscretePidType
	DigitalPidType
)

type PidStaticData struct {
	Name         string        `json:"name"`
	Index        int           `json:"index"`
	Type         PidType       `json:"type"`
	SamplePeriod time.Duration `json:"period"`
}

func NewPidStaticData(name string, index int, typ PidType, period time.Duration) PidStaticData {
	return PidStaticData{
		Name:         name,
		Index:        index,
		Type:         typ,
		SamplePeriod: period,
	}
}

type PidState int

const InternalErrorPidState = -1
const (
	NeverUpdatedPidState PidState = iota
	OkPidState
	BadPidState
)

type PidDynamicData struct {
	Value       float32  `json:"value"`
	State       PidState `json:"state"`
	Updates     int      `json:"-"`
	LastUpdated int64    `json:"timestamp"`
}

type PidData struct {
	PidStaticData
	PidDynamicData
}

type PidsHub struct {
	subscribe   chan *DummyPIDTicker
	unsubscribe chan *DummyPIDTicker

	incomingPidListRequest       chan commons.CommandRequest
	incomingPidListUpdateRequest chan commons.CommandRequest
}

func (h *PidsHub) log(v ...interface{}) {
	if debugging {
		text := fmt.Sprint(v...)
		prefix := fmt.Sprint("<< PID HUB >> ~ ")
		if debugWithTimeStamp {
			prefix = time.Now().Format(time.StampMicro) + " " + prefix
		}
		log.Println(prefix, text)
	}
}

func (h *PidsHub) logf(format string, v ...interface{}) {
	if debugging {
		prefix := fmt.Sprint("<< PID HUB >> ~ ")
		if debugWithTimeStamp {
			prefix = time.Now().Format(time.StampMicro) + " " + prefix
		}
		log.Printf(prefix+format, v...)
	}
}

var pidsHub = PidsHub{
	subscribe:   make(chan *DummyPIDTicker),
	unsubscribe: make(chan *DummyPIDTicker),

	incomingPidListRequest:       make(chan commons.CommandRequest),
	incomingPidListUpdateRequest: make(chan commons.CommandRequest),
}

func Subscribe(pid *DummyPIDTicker) {
	pidsHub.subscribe <- pid
}

func Unsubscribe(pid *DummyPIDTicker) {
	pidsHub.unsubscribe <- pid
}

func RequestPidList(request commons.CommandRequest) commons.RawResponseData {
	pidsHub.incomingPidListRequest <- request
	return <-request.Response
}

func (h *PidsHub) runPidsHub() {
	h.log("Runing Dummy PIDs Hub")
	defer h.log("Exiting Dummy Hub")

	dummyTickersMap := make(map[int]*DummyPIDTicker)
	//ticker := time.NewTicker(time.Second)
	for {

		select {

		//case tick := <-ticker.C:
		//	h.log("Ticker sent tick ", tick)

		case pid := <-h.subscribe:
			h.log("Subscribing dummy ticker ", pid.pidData.Name)
			dummyTickersMap[pid.pidData.Index] = pid

		case pid := <-h.unsubscribe:
			h.log("Unsubscribing dummy ticker ", pid.pidData.Name)
			delete(dummyTickersMap, pid.pidData.Index)

		case request := <-h.incomingPidListRequest:

			h.log("Dispatching PID List Command")
			responseData, err := processPIDListCommand(dummyTickersMap)
			if err != nil {
				h.log("Error processing PID List Command: ", err)
			}
			request.Response <- responseData

		case request := <-h.incomingPidListUpdateRequest:
			request.Response <- []byte{}
		}
	}
}

func Init() {

	log.Println("INIT PID.GO >>> ", commons.GetInitCounter())
	wslogic.RegisterMessagesHandler(wslogic.RequestMessagesHandler{RequestType: commons.ApiPidListCommandRequest, Handler: RequestPidList})
	log.Println("INIT PID.GO >>> Back from registering messages handler")
	go pidsHub.runPidsHub()

	now := time.Now().UnixNano()

	for i := 0; i < pidTickers; i++ {
		period := pidTickersMinDuration + time.Duration(rand.Int63n(int64(pidTickersRangeDuration)))
		ty := rand.Intn(3)
		staticData := NewPidStaticData(fmt.Sprint("Sig", i), int(atomic.AddInt32(&pidIndexCounter, 1)-1), PidType(ty), period)
		t := NewDummyPIDTicker(staticData, standardTickHandler)
		Subscribe(t)
		t.Launch()
	}

	var err error
	dbase, err = db.Dial()
	if err != nil {
		log.Println(">>>>>>>    Error dialing to the DB: ", err)
	} else {
		SavePidsToDb(now)
	}

}
