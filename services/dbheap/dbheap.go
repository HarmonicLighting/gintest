package dbheap

import (
	"container/heap"
	"errors"
	"fmt"
	"local/gintest/services/db"
)

const nConcurrentSessions = 10

var globalDBHeap *DbHeap

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
func (pq *priorityQueue) update(item *heapItem) {
	item.timesCloned--
	heap.Fix(pq, item.index)
}

type ClientHeapSession struct {
	copiedSession *db.DB
	ClientSession *db.DB
	closeChannel  chan<- *ClientHeapSession
}

type DbHeap struct {
	maxConcurrentSessions int
	isRunning             bool
	masterDbSession       *db.DB

	newDBSession   chan chan *ClientHeapSession
	closeDBSession chan *ClientHeapSession
}

func NewDbHeap(maxConcurrentSessions int, masterSession *db.DB) *DbHeap {
	return &DbHeap{
		maxConcurrentSessions: maxConcurrentSessions,
		isRunning:             false,
		masterDbSession:       masterSession,
		newDBSession:          make(chan chan *ClientHeapSession),
		closeDBSession:        make(chan *ClientHeapSession),
	}
}

func (dbh *DbHeap) GetSession() (*ClientHeapSession, error) {
	if !dbh.isRunning {
		return nil, errors.New("The DB Heap is not running")
	}
	ch := make(chan *ClientHeapSession)
	dbh.newDBSession <- ch
	return <-ch, nil
}

func GetSession() (*ClientHeapSession, error) {
	return globalDBHeap.GetSession()
}

func (i *ClientHeapSession) Close() {
	if i.closeChannel == nil {
		return
	}
	i.closeChannel <- i
}

func (dbh *DbHeap) runHub() {
	heapMap := make(map[*db.DB]*heapItem)
	queue := make(priorityQueue, dbh.maxConcurrentSessions)
	for i := 0; i < dbh.maxConcurrentSessions; i++ {
		newCopiedSession, err := dbh.masterDbSession.Copy()
		if err != nil {
			panic(err)
		}
		queue[i] = &heapItem{
			copiedSession: newCopiedSession,
			timesCloned:   0,
			index:         i,
		}
		heapMap[newCopiedSession] = queue[i]
	}
	heap.Init(&queue)
	for {
		select {
		case sessionChan := <-dbh.newDBSession:

			item := heap.Pop(&queue).(*heapItem)
			clonnedSession, err := item.copiedSession.Clone()
			if err != nil {
				panic(err)
			}
			item.timesCloned++
			heap.Push(&queue, item)
			sessionChan <- &ClientHeapSession{
				copiedSession: item.copiedSession,
				ClientSession: clonnedSession,
				closeChannel:  dbh.closeDBSession,
			}
		case clientClose := <-dbh.closeDBSession:

			if clientClose.ClientSession != nil {
				clientClose.ClientSession.Close()
			}

			hItem, ok := heapMap[clientClose.copiedSession]
			if !ok {
				fmt.Println("the copied session wasn't found!")
			} else {
				queue.update(hItem)
				if hItem.timesCloned < 0 {
					fmt.Println("Error after update: Index:", hItem.index, ", times clonned:", hItem.timesCloned)
				}
			}
			clientClose.ClientSession = nil
			clientClose.copiedSession = nil
			clientClose.closeChannel = nil
		}
	}
}

func (dbh *DbHeap) Run() {

	go dbh.runHub()
	dbh.isRunning = true
}

func init() {
	masterSession, err := db.Dial()
	if err != nil {
		panic(err)
	}
	globalDBHeap = NewDbHeap(nConcurrentSessions, masterSession)
	globalDBHeap.Run()
}
