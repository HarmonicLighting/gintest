package pid

import (
	"fmt"
	"local/gintest/apicommands"
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

	pidListUpdateTimePeriod = time.Millisecond * 250
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

	incomingPidListRequest       chan wslogic.CommandRequest
	incomingPidListUpdateRequest chan wslogic.CommandRequest
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

	incomingPidListRequest:       make(chan wslogic.CommandRequest),
	incomingPidListUpdateRequest: make(chan wslogic.CommandRequest),
}

func Subscribe(pid *DummyPIDTicker) {
	pidsHub.subscribe <- pid
}

func Unsubscribe(pid *DummyPIDTicker) {
	pidsHub.unsubscribe <- pid
}

func RequestPidList(request wslogic.CommandRequest) wslogic.RawResponseData {
	pidsHub.incomingPidListRequest <- request
	return request.ReceiveCommandResponse()
}

func (h *PidsHub) runPidsHub() {
	h.log("Runing Dummy PIDs Hub")
	defer h.log("Exiting Dummy Hub")

	dummyTickersMap := make(map[int]*DummyPIDTicker)
	ticker := time.NewTicker(pidListUpdateTimePeriod)
	for {

		select {

		case <-ticker.C:
			//h.log("Ticker sent tick ", tick)
			responseData, err := processPIDListUpdateCommand(dummyTickersMap)
			if err != nil {
				//h.log("Error processing PID List Update Command: ", err)
				continue
			}
			wslogic.Broadcast(responseData)

		case pid := <-h.subscribe:
			//h.log("Subscribing dummy ticker ", pid.pidData.Name)
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
			h.log("Dispatched!-----------------------------------------------------")
			request.SendCommandResponse(responseData)
			//request.Response <- responseData

		case request := <-h.incomingPidListUpdateRequest:
			request.SendCommandResponse([]byte{})
			h.log(">>>>>>>>> DISPATCHED AN EMPTY COMMAND RESPONSE")
			//request.Response <- []byte{}
		}
	}
}

func Init() {

	log.Println("INIT PID.GO >>> ", commons.GetInitCounter())
	wslogic.RegisterMessagesHandler(
		wslogic.NewRequestMessageHandler(apicommands.ServerCompleteSignalList, RequestPidList))
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

	log.Println("A total of ", pidTickers, " dummy tickers have been launched")

	var err error
	dbase, err = db.Dial()
	if err != nil {
		log.Println(">>>>>>>    Error dialing to the DB: ", err)
	} else {
		SavePidsToDb(now)
	}

}
