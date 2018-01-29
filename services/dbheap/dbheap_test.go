package dbheap_test

import (
	"local/gintest/services/db"
	"local/gintest/services/dbheap"
	"testing"
)

func TestDbHeap(t *testing.T) {
	session, err := db.Dial()
	if err != nil {
		t.Error("The DB session didn't open up")
	}
	defer session.Close()

	dbh := dbheap.NewDbHeap(5, session)
	_, err = dbh.GetSession()
	if err == nil {
		t.Fail()
	}
	dbh.Run()
	clientSessions := make([]*dbheap.ClientHeapSession, 20)
	sample := &db.DBSample{
		Pid:       0,
		Value:     0,
		Timestamp: 0,
	}
	for i := 0; i < 10; i++ {
		clientSessions[i], err = dbh.GetSession()
		clientSessions[i].ClientSession.InsertSamples(sample)
	}

	for i := 0; i < 10; i++ {
		clientSessions[i].Close()
	}

	for i := 10; i < 20; i++ {
		clientSessions[i], err = dbh.GetSession()
		clientSessions[i].ClientSession.InsertSamples(sample)
	}

	for i := 10; i < 20; i++ {
		clientSessions[i].Close()
	}

	clientSessions[0].Close()
}
