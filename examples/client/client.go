package main

import (
	"github.com/olahol/melody"
	"log"
	"net/url"
	"time"
)

func main() {
	u := url.URL{Scheme: "ws", Host: "localhost:5000", Path: "/ws"}
	cl := melody.NewClient(u)
	cl.HandleConnect(func() {
		log.Println("connected, sending my name packet")
		cl.Send([]byte("my name is client"))
	})

	cl.HandleError(func(err error) {
		log.Println("Encountered error from melody: ", err)
	})

	cl.HandleMessage(func(bytes []byte) {
		log.Println("Received '", string(bytes), "' from server")
		time.Sleep(time.Second * 2)
		cl.Send([]byte(time.Now().String()))
	})

	cl.HandleMessageBinary(func(bytes []byte) {
		log.Println("Received '", string(bytes), "' from server over binary")
	})

	cl.Connect()
}
