package dbheap

import (
	"container/heap"
	"fmt"
	"local/gintest/services/db"
)

type heapItem struct {
	copiedSession *db.DB
	timesCloned   int // Priority
	index         int
}

type priorityQueue []*heapItem

func (pq priorityQueue) Len() int {
	return len(pq)
}

func (pq priorityQueue) Less(i, j int) bool {
	// We want Pop to give us the lowest priority so we use greater than here.
	return pq[i].timesCloned < pq[j].timesCloned
}

func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *priorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*heapItem)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	item.index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

// update modifies the priority and value of an Item in the queue.
func (pq *priorityQueue) update(item *heapItem, value *db.DB, timesCloned int) {
	item.copiedSession = value
	item.timesCloned = timesCloned
	heap.Fix(pq, item.index)
}

type clientHeapItem struct {
	copiedSession *db.DB
	clientSession *db.DB
}

type DbHeap struct {
	maxConcurrentSessions int
	masterDbSession       *db.DB

	newDBSession   chan chan clientHeapItem
	closeDBSession chan clientHeapItem
}

func (dbh *DbHeap) NewDbHeap(maxConcurrentSessions int, masterSession *db.DB) *DbHeap {
	return &DbHeap{
		maxConcurrentSessions: maxConcurrentSessions,
		masterDbSession:       masterSession,
		newDBSession:          make(chan chan clientHeapItem),
		closeDBSession:        make(chan clientHeapItem),
	}
}

func (dbh *DbHeap) runHub() {
	queue := priorityQueue{}
	heap.Init(&queue)
	heapMap := make(map[*db.DB]int)
	for {
		select {
		case sessionChan := <-dbh.newDBSession:
			if len(heapMap) < dbh.maxConcurrentSessions {
				newCopiedSession, err := dbh.masterDbSession.Copy()
				if err != nil {
					panic(err)
				}
				heapMap[newCopiedSession] = 0
				item := &heapItem{
					copiedSession: newCopiedSession,
					timesCloned:   0,
					index:         len(heapMap),
				}
				heap.Push(&queue, item)
				sessionChan <- clientHeapItem{copiedSession: newCopiedSession, clientSession: newCopiedSession}
			} else {
				item := heap.Pop(&queue).(heapItem)
				clonnedSession, err := item.copiedSession.Clone()
				if err != nil {
					panic(err)
				}
				item.timesCloned++
				heapMap[item.copiedSession]++
				heap.Push(&queue, clonnedSession)
				sessionChan <- clientHeapItem{copiedSession: item.copiedSession, clientSession: clonnedSession}
			}
		case clientClose := <-dbh.closeDBSession:
			// TO-DO
			fmt.Println(clientClose)
		}
	}
}

func (dbh *DbHeap) run() {

	go func() {

	}()

}
