# melody

![Build Status](https://github.com/olahol/melody/actions/workflows/test.yml/badge.svg)
[![Codecov](https://img.shields.io/codecov/c/github/olahol/melody)](https://app.codecov.io/github/olahol/melody)
[![Go Report Card](https://goreportcard.com/badge/github.com/olahol/melody)](https://goreportcard.com/report/github.com/olahol/melody)
[![GoDoc](https://godoc.org/github.com/olahol/melody?status.svg)](https://godoc.org/github.com/olahol/melody)

> :notes: Minimalist websocket framework for Go.

Melody is websocket framework based on [github.com/gorilla/websocket](https://github.com/gorilla/websocket)
that abstracts away the tedious parts of handling websockets. It gets out of
your way so you can write real-time apps. Features include:

* [x] Clear and easy interface similar to `net/http` or Gin.
* [x] A simple way to broadcast to all or selected connected sessions.
* [x] Message buffers making concurrent writing safe.
* [x] Automatic handling of sending ping/pong heartbeats that timeout broken sessions.
* [x] Store data on sessions.

## Install

```bash
go get github.com/olahol/melody
```

## [Example: chat](https://github.com/olahol/melody/tree/master/examples/chat)

[![Chat](https://cdn.rawgit.com/olahol/melody/master/examples/chat/demo.gif "Demo")](https://github.com/olahol/melody/tree/master/examples/chat)

```go
package main

import (
	"net/http"

	"github.com/olahol/melody"
)

func main() {
	m := melody.New()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		m.HandleRequest(w, r)
	})

	m.HandleMessage(func(s *melody.Session, msg []byte) {
		m.Broadcast(msg)
	})

	http.ListenAndServe(":5000", nil)
}
```

## [Example: gophers](https://github.com/olahol/melody/tree/master/examples/gophers)

[![Gophers](https://cdn.rawgit.com/olahol/melody/master/examples/gophers/demo.gif "Demo")](https://github.com/olahol/melody/tree/master/examples/gophers)

```go
package main

import (
	"fmt"
	"net/http"
	"sync/atomic"

	"github.com/olahol/melody"
)

var idCounter atomic.Int64

func main() {
	m := melody.New()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		m.HandleRequest(w, r)
	})

	m.HandleConnect(func(s *melody.Session) {
		id := idCounter.Add(1)

		s.Set("id", id)

		s.Write([]byte(fmt.Sprintf("iam %d", id)))
	})

	m.HandleDisconnect(func(s *melody.Session) {
		if id, ok := s.Get("id"); ok {
			m.BroadcastOthers([]byte(fmt.Sprintf("dis %d", id)), s)
		}
	})

	m.HandleMessage(func(s *melody.Session, msg []byte) {
		if id, ok := s.Get("id"); ok {
			m.BroadcastOthers([]byte(fmt.Sprintf("set %d %s", id, msg)), s)
		}
	})

	http.ListenAndServe(":5000", nil)
}
```

### [More examples](https://github.com/olahol/melody/tree/master/examples)

## [Documentation](https://godoc.org/github.com/olahol/melody)

## Contributors

<a href="https://github.com/olahol/melody/graphs/contributors">
	<img src="https://contrib.rocks/image?repo=olahol/melody" />
</a>

## FAQ

If you are getting a `403` when trying  to connect to your websocket you can [change allow all origin hosts](http://godoc.org/github.com/gorilla/websocket#hdr-Origin_Considerations):

```go
m := melody.New()
m.Upgrader.CheckOrigin = func(r *http.Request) bool { return true }
```
