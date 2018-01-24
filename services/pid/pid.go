package pid

import (
	"fmt"
	"local/gintest/commons"
	"local/gintest/services/db"
	"local/gintest/wslogic"
	"log"
	"time"
)

const (
	debugging          = commons.Debugging
	debugWithTimeStamp = commons.DebugWithTimeStamp
)

type PidsHub struct {
	subscribe              chan *DummyPIDTicker
	unsubscribe            chan *DummyPIDTicker
	incomingPidListRequest chan commons.CommandRequest
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

	incomingPidListRequest: make(chan commons.CommandRequest),
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

	for {

		select {

		case pid := <-h.subscribe:
			h.log("Subscribing dummy ticker ", pid.name)
			dummyTickersMap[pid.index] = pid

		case pid := <-h.unsubscribe:
			h.log("Unsubscribing dummy ticker ", pid.name)
			delete(dummyTickersMap, pid.index)

		case request := <-h.incomingPidListRequest:

			h.log("Dispatching PID List Command")
			responseData, err := processPIDListCommand(dummyTickersMap)
			if err != nil {
				h.log("Error processing PID List Command: ", err)
			}
			request.Response <- responseData

		}
	}
}

func Init() {

	log.Println("INIT PID.GO >>> ", commons.GetInitCounter())
	wslogic.RegisterMessagesHandler(wslogic.RequestMessagesHandler{RequestType: commons.ApiPidListCommandRequest, Handler: RequestPidList})
	log.Println("INIT PID.GO >>> Back from registering messages handler")
	go pidsHub.runPidsHub()

	now := time.Now().UnixNano()

	for i := 0; i < 2; i++ {
		t := NewDummyPIDTicker(fmt.Sprint("Signal ", i), time.Second*10, standardTickHandler)
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
