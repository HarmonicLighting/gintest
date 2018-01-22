package pid

import (
	"fmt"
	"local/gintest/constants"
	"local/gintest/services/db"
	"log"
	"math/rand"
	"sync/atomic"
	"time"
)

var (
	dbase *db.DB
)

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

func standardTickHandler(t *DummyPIDTicker, time time.Time) {
	randfloat := rand.Float32() * 100
	event := constants.NewPIDUpdateResponse(t.GetIndex(), time.UnixNano(), randfloat)
	message, err := event.Stringify()
	if err != nil {
		log.Println("Error stringifying: ", err)
		return
	}
	log.Println("Broadcasting event ", string(message), " by Dummy Ticker ", t.GetName())
	broadcast(message)
	log.Println("Saving sample to DB")
	d, err := dbase.Copy()
	if err != nil {
		log.Println("Error copying the db session: ", err)
		return
	}
	defer d.Close()
	d.InsertSamples(&db.DBSample{Pid: t.GetIndex(), Value: event.Value, Timestamp: event.Timestamp})
}
