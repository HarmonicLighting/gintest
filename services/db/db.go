package db

import (
	"errors"
	"io"
	"log"
	"time"

	"github.com/globalsign/mgo"
)

const (
	mongoURL     = "localhost"
	dbName       = "gintest"
	samplesCName = "samples"
	pidsCName    = "pids"
)

type DBSample struct {
	Pid       int
	Value     float32
	Timestamp int64
}

type DBPid struct {
	Name   string
	Pid    int
	Period time.Duration
}

type DBPids struct {
	Timestamp int64
	Pids      []*DBPid
}

type DB struct {
	session  *mgo.Session
	db       *mgo.Database
	samplesC *mgo.Collection
	pidsC    *mgo.Collection
	logger   *log.Logger
	ok       bool
}

func (d *DB) Copy() (*DB, error) {
	if !d.ok {
		return nil, errors.New("The original DB instance is not ready")
	}
	session := d.session.Copy()
	return d.setup(session)
}

func (d *DB) Clone() (*DB, error) {
	if !d.ok {
		return nil, errors.New("The original DB instance is not ready")
	}
	session := d.session.Clone()
	return d.setup(session)
}

func (d *DB) Close() error {
	if !d.ok {
		return errors.New("The original DB instance is not open")
	}

	d.session.Close()
	d.ok = false
	return nil
}

func (d *DB) setup(session *mgo.Session) (*DB, error) {
	db := session.DB(dbName)
	samplesC := db.C(samplesCName)
	pidsC := db.C(pidsCName)
	logger := d.logger
	return &DB{session: session, db: db, samplesC: samplesC, pidsC: pidsC, logger: logger, ok: true}, nil
}

func (d *DB) InsertSamples(samples ...*DBSample) error {
	if !d.ok {
		return errors.New("This DB instance is not ready.")
	}

	isamples := make([]interface{}, len(samples))
	for i, s := range samples {
		isamples[i] = s
	}

	err := d.samplesC.Insert(isamples...)
	if err != nil {
		d.logger.Println("Error inserting Samples: ", err)
	}
	return err
}

func (d *DB) InsertPids(pids DBPids) error {
	err := d.pidsC.Insert(&pids)
	if err != nil {
		d.logger.Println("Error inserting PIDS: ", err)
	}
	return err
}

func Dial() (*DB, error) {
	session, err := mgo.Dial(mongoURL)
	if err != nil {
		return nil, err
	}
	session.SetMode(mgo.Monotonic, true)
	db := session.DB(dbName)

	samplesC := db.C(samplesCName)
	pidsC := db.C(pidsCName)
	var infoHandle io.Writer
	logger := log.New(infoHandle, "<<DB Struct>> ", log.Ldate|log.Ltime|log.Lshortfile)
	return &DB{session: session, db: db, samplesC: samplesC, pidsC: pidsC, logger: logger, ok: true}, nil
}

func init() {
	//var infoHandle io.Writer
	//mgo.SetLogger(log.New(infoHandle, "MGO DB ~ ", log.Ldate|log.Ltime|log.Lshortfile))
}
