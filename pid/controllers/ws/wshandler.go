package controllers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var wsupgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Demo struct {
	Field1 string `json:"field1"`
	Field2 int    `json:"field2"`
}

type sub struct {
	ch  chan<- error
	msg []byte
	tp  int
}

func merge(channels ...<-chan sub) <-chan sub {
	out := make(chan sub)
	var wg sync.WaitGroup
	wg.Add(len(channels))
	for _, c := range channels {
		go func(channel <-chan sub) {
			for v := range channel {
				out <- v
			}
			log.Println("merged channel closed")
			wg.Done()
		}(c)
	}

	go func() {
		wg.Wait()
		log.Println("all merged channels closed")
		close(out)
	}()
	return out
}

func launchGreetings(conn *websocket.Conn) <-chan sub {
	c := make(chan sub)
	e := make(chan error)

	go func() {
		i := 0
		for {

			mydemo := Demo{"hey", i}
			i++
			marshalled, err := json.Marshal(mydemo)
			if err != nil {
				log.Println("Error marshalling mydemo: ", err)
			}

			thisSub := sub{ch: e, msg: marshalled, tp: websocket.TextMessage}
			c <- thisSub
			err = <-e
			if err != nil {
				log.Println("Error sending a greeting: ", err)
				close(c)
				close(e)
				return
			}
			time.Sleep(1 * time.Second)
		}
	}()
	return c
}

func launchWriteToSocket(conn *websocket.Conn, c <-chan sub) {
	for s := range c {
		err := conn.WriteMessage(s.tp, s.msg)
		if err != nil {
			log.Println("Error sending message <", string(s.msg), ">: ", err)
		}
		s.ch <- err
	}
	log.Println("Exiting the writer to socket")
}

// WsHandler handles the ws communication
func WsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Beginning ws connection")
	conn, err := wsupgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("Failed to upgrade ws: %+v\n", err)
		return
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(time.Second * 10))

	echoCh := make(chan sub)
	errCh := make(chan error)

	greetCh := launchGreetings(conn)

	launchWriteToSocket(conn, merge(greetCh, echoCh))

	for {
		t, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error reading socket: ", err)
			close(echoCh)
			close(errCh)
			break
		}
		var mydemo Demo
		err = json.Unmarshal(msg, &mydemo)
		log.Println("Got message from ", conn.RemoteAddr(), ": <", t, "> ", msg, "|", mydemo, " - ", err)
		//conn.WriteMessage(t, msg)
		echoCh <- sub{ch: errCh, msg: msg, tp: websocket.TextMessage}
		err = <-errCh
		if err != nil {
			log.Println("Error sending an the echo <", string(msg), ">: ", err)
			close(echoCh)
			close(errCh)
			break
		} else {
			log.Println("Message <", string(msg), "> successfully echoed")
		}
	}

	log.Println(">>>> Exiting this session")
}
