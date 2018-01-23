package pid

import (
	"errors"
	"fmt"
	"local/gintest/constants"
	"local/gintest/services/db"
	"log"
	"time"
)

const (
	debugging          = constants.Debugging
	debugWithTimeStamp = constants.DebugWithTimeStamp
)

var (
	pidIndexCounter int32
)

type pidsHub struct {
	subscribe       chan *DummyPIDTicker
	unsubscribe     chan *DummyPIDTicker
	incomingCommand chan constants.CommandRequest

	broadcastHandler constants.BroadcastHandle
}

func (h *pidsHub) log(v ...interface{}) {
	if debugging {
		text := fmt.Sprint(v...)
		prefix := fmt.Sprint("<< PID HUB >> ~ ")
		if debugWithTimeStamp {
			prefix = time.Now().Format(time.StampMicro) + " " + prefix
		}
		log.Println(prefix, text)
	}
}

func (h *pidsHub) logf(format string, v ...interface{}) {
	if debugging {
		prefix := fmt.Sprint("<< PID HUB >> ~ ")
		if debugWithTimeStamp {
			prefix = time.Now().Format(time.StampMicro) + " " + prefix
		}
		log.Printf(prefix+format, v...)
	}
}

var dHub = pidsHub{
	subscribe:       make(chan *DummyPIDTicker),
	unsubscribe:     make(chan *DummyPIDTicker),
	incomingCommand: make(chan constants.CommandRequest),
}

func Subscribe(pid *DummyPIDTicker) {
	dHub.subscribe <- pid
}

func Unsubscribe(pid *DummyPIDTicker) {
	dHub.unsubscribe <- pid
}

func (h *pidsHub) RequestCommand(request constants.CommandRequest) constants.RawCommandResponse {
	h.incomingCommand <- request
	return <-request.Response
}

// SetBroadcastHandle sets a broadcast handle to be used for this module
func SetBroadcastHandle(handle constants.BroadcastHandle) {
	dHub.broadcastHandler = handle
}

func RequestCommand(request constants.CommandRequest) constants.RawCommandResponse {
	dHub.incomingCommand <- request
	return <-request.Response
}

func broadcast(message []byte) error {
	if dHub.broadcastHandler == nil {
		return errors.New("The Broadcast handler is not set")
	}
	dHub.broadcastHandler(message)
	return nil
}

func (h *pidsHub) runHub() {
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

		case request := <-h.incomingCommand:
			h.log("Incoming Command id=", request.Command)
			switch request.Command {

			case constants.PIDListCommandRequest:
				h.log("Dispatching PID List Command")
				responseData, err := processPIDListCommand(dummyTickersMap)
				if err != nil {
					h.log("Error processing PID List Command: ", err)
				}
				request.Response <- responseData

			default:
				h.log("Invalid Command Id (", request.Command, ")")
				notSupportedResponse := constants.NewNotSupportedStatusCommandResponse(request.Command)
				response, _ := notSupportedResponse.Stringify()
				request.Response <- response
			}
		}
	}
}

func init() {

	go dHub.runHub()

	now := time.Now().UnixNano()

	for i := 0; i < 2; i++ {
		t := NewDummyPIDTicker(fmt.Sprint("Signal ", i), time.Second*10, standardTickHandler)
		Subscribe(t)
		t.Launch()
	}

	var err error
	dbase, err = db.Dial()
	if err != nil {
		log.Println("Error dialing to the DB: ", err)
	} else {
		SavePidsToDb(now)
	}

}
