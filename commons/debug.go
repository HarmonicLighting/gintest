package commons

import "sync/atomic"

const (
	Debugging          = true
	DebugWithTimeStamp = true
)

var (
	initCounter int32
)

func GetInitCounter() int32 {
	return atomic.AddInt32(&initCounter, 1) - 1
}
