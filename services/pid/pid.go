package pid

import (
	"errors"
	"fmt"
	"local/gintest/constants"
	"log"
	"sync/atomic"
	"time"
)

const (
	debugging          = constants.Debugging
	debugWithTimeStamp = constants.DebugWithTimeStamp
)

var (
	pidIndexCounter int32
)

type dummyPidsHub struct {
	subscribe       chan *DummyPIDTicker
	unsubscribe     chan *DummyPIDTicker
	incomingCommand chan constants.CommandRequest
	dummyTickersMap map[int]*DummyPIDTicker
}

func (h *dummyPidsHub) log(v ...interface{}) {
	if debugging {
		text := fmt.Sprint(v...)
		prefix := fmt.Sprint("<< PID HUB >> ~ ")
		if debugWithTimeStamp {
			prefix = time.Now().Format(time.StampMicro) + " " + prefix
		}
		log.Println(prefix, text)
	}
}

func (h *dummyPidsHub) logf(format string, v ...interface{}) {
	if debugging {
		prefix := fmt.Sprint("<< PID HUB >> ~ ")
		if debugWithTimeStamp {
			prefix = time.Now().Format(time.StampMicro) + " " + prefix
		}
		log.Printf(prefix+format, v...)
	}
}

var dHub = dummyPidsHub{
	subscribe:       make(chan *DummyPIDTicker),
	unsubscribe:     make(chan *DummyPIDTicker),
	incomingCommand: make(chan constants.CommandRequest),
	dummyTickersMap: make(map[int]*DummyPIDTicker),
}

func Subscribe(pid *DummyPIDTicker) {
	dHub.subscribe <- pid
}

func Unsubscribe(pid *DummyPIDTicker) {
	dHub.unsubscribe <- pid
}

func (h *dummyPidsHub) RequestCommand(request constants.CommandRequest) constants.CommandResponse {
	h.incomingCommand <- request
	return <-request.Response
}

func (h *dummyPidsHub) getApiPidsList() []constants.ApiPid {
	pids := make([]constants.ApiPid, len(h.dummyTickersMap))
	i := 0
	for _, pid := range h.dummyTickersMap {
		pids[i] = constants.NewApiPID(pid.name, pid.index, float32(pid.period))
		i++
	}
	return pids
}

func (h *dummyPidsHub) processPIDListCommand() ([]byte, error) {
	pids := h.getApiPidsList()
	responseStruct := constants.NewPIDListEvent(pids)
	return responseStruct.Stringify()
}

func RequestCommand(request constants.CommandRequest) constants.CommandResponse {
	dHub.incomingCommand <- request
	return <-request.Response
}

func RequestPIDListEventStruct() constants.PIDListEvent {
	request := constants.CommandRequest{Command: constants.PIDListCommand, Response: make(chan constants.CommandResponse)}
	response := RequestCommand(request)
	var listEvent constants.PIDListEvent
	err := listEvent.Parse(response.Response)
	if err != nil {
		log.Println("On pid RequestApiPids: Error parsing the List Event : ", err)
	}
	return listEvent
}

func (h *dummyPidsHub) runHub() {
	h.log("Runing Dummy PIDs Hub")
	defer h.log("Exiting Dummy Hub")
	for {
		select {
		case pid := <-h.subscribe:
			h.log("Subscribing dummy ticker ", pid.name)
			h.dummyTickersMap[pid.index] = pid
		case pid := <-h.unsubscribe:
			h.log("Unsubscribing dummy ticker ", pid.name)
			delete(h.dummyTickersMap, pid.index)
		case request := <-h.incomingCommand:
			h.log("Incoming Command id=", request.Command)
			switch request.Command {
			case constants.PIDListCommand:
				h.log("Dispatching PID List Command")
				responseData, err := h.processPIDListCommand()
				if err != nil {
					h.log("Error processing PID List Command: ", err)
				}
				request.Response <- constants.NewCommandResponse(responseData, err)
			default:
				h.log("Invalid Command Id (", request.Command, ")")
				request.Response <- constants.NewCommandResponse([]byte{}, errors.New("Invalid Command ID"))
			}
		}
	}
}

type DummyPidTickerFunc func(*DummyPIDTicker, time.Time)

type DummyPIDTicker struct {
	index     int
	name      string
	period    time.Duration
	ticker    *time.Ticker
	onTick    DummyPidTickerFunc
	stop      chan struct{}
	isRunning bool
}

func (t *DummyPIDTicker) log(v ...interface{}) {
	if debugging {
		text := fmt.Sprint(v...)
		prefix := fmt.Sprint("<Ticker ", t.index, "> ~ ")
		if debugWithTimeStamp {
			prefix = time.Now().Format(time.StampMicro) + " " + prefix
		}
		log.Println(prefix, text)
	}
}

func (t *DummyPIDTicker) logf(format string, v ...interface{}) {
	if debugging {
		prefix := fmt.Sprint("<Ticker ", t.index, "> ~ ")
		if debugWithTimeStamp {
			prefix = time.Now().Format(time.StampMicro) + " " + prefix
		}
		log.Printf(prefix+format, v...)
	}
}

func (t *DummyPIDTicker) GetIndex() int {
	return t.index
}

func (t *DummyPIDTicker) GetName() string {
	return t.name
}

func NewDummyPIDTicker(pidname string, period time.Duration, onTick DummyPidTickerFunc) *DummyPIDTicker {
	ticker := &DummyPIDTicker{index: int(atomic.AddInt32(&pidIndexCounter, 1) - 1), name: pidname, period: period, ticker: nil, onTick: onTick, stop: make(chan struct{}), isRunning: false}
	return ticker
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

	t.isRunning = true

	t.ticker = time.NewTicker(t.period)

	go func() {

		defer func() {
			t.ticker.Stop()
			t.ticker = nil
			t.isRunning = false
		}()

		for {
			select {

			// Stop signal received: exit the go routine
			case <-t.stop:
				t.log("Stopping the Dummy Ticker ", t.name)
				return

				// Ticker signal, continue normal ticking
			case now := <-t.ticker.C:
				t.log("Got a tick for Dummy Ticker ", t.name)
				t.onTick(t, now)
			}
		}
	}()
}

func init() {
	go dHub.runHub()
}
