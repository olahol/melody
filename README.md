# melody

[![Build Status](https://travis-ci.org/olahol/melody.svg)](https://travis-ci.org/olahol/melody)
[![Coverage Status](https://img.shields.io/coveralls/olahol/melody.svg?style=flat)](https://coveralls.io/r/olahol/melody)
[![GoDoc](https://godoc.org/github.com/olahol/melody?status.svg)](https://godoc.org/github.com/olahol/melody)

> :notes: Minimalist websocket framework for Go.

Melody is websocket framework based on [github.com/gorilla/websocket](https://github.com/gorilla/websocket)
that abstracts away the tedious parts of handling websockets. It gets out of
your way so you can write real-time apps. Features include:

* [x] Clear and easy interface similar to `net/http` or Gin.
* [x] A simple way to broadcast to all or selected connected sessions.
* [x] Message buffers making concurrent writing safe.
* [x] Automatic handling of ping/pong and session timeouts.

## Install

```bash
go get github.com/olahol/melody
```

## [Example](https://github.com/olahol/melody/tree/master/examples)

[Multi channel chat server](https://github.com/olahol/melody/tree/master/examples/multichat),
error handling left as en exercise for the developer.

[![Chat demo](https://cdn.rawgit.com/olahol/melody/master/examples/chat/demo.gif "Demo")](https://github.com/olahol/melody/tree/master/examples/multichat)

```go
package main

import (
	"github.com/olahol/melody"
	"github.com/gin-gonic/gin"
	"net/http"
)

func main() {
	r := gin.Default()
	m := melody.New()

	r.GET("/", func(c *gin.Context) {
		http.ServeFile(c.Writer, c.Request, "index.html")
	})

	r.GET("/channel/:name", func(c *gin.Context) {
		http.ServeFile(c.Writer, c.Request, "chan.html")
	})

	r.GET("/channel/:name/ws", func(c *gin.Context) {
		m.HandleRequest(c.Writer, c.Request)
	})

	m.HandleMessage(func(s *melody.Session, msg []byte) {
		m.BroadcastFilter(msg, func(q *melody.Session) bool {
			return q.Request.URL.Path == s.Request.URL.Path
		})
	})

	r.Run(":5000")
}
```

### [More examples](https://github.com/olahol/melody/tree/master/examples)

## [Documentation](https://godoc.org/github.com/olahol/melody)

## Contributors

* Ola Holmström (@olahol)
* Shogo Iwano (@shiwano)

## FAQ

If you are getting a `403` when trying  to connect to your websocket you can [change allow all origin hosts](http://godoc.org/github.com/gorilla/websocket#hdr-Origin_Considerations):

```go
m := melody.New()
m.Upgrader.CheckOrigin = func(r *http.Request) bool { return true }
```
