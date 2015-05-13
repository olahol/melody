# melody

[![GoDoc](https://godoc.org/github.com/olahol/melody?status.svg)](https://godoc.org/github.com/olahol/melody)
[![Build Status](https://travis-ci.org/olahol/melody.svg)](https://travis-ci.org/olahol/melody)

> :notes: Simple websocket framework for Go

## Install

```bash
go get github.com/olahol/melody
```

## Example

[Simple broadcasting chat server](https://github.com/olahol/melody/tree/master/examples/chat),
error handling left as en exercise for the developer.

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

	r.GET("/ws", func(c *gin.Context) {
		m.HandleRequest(c.Writer, c.Request)
	})

	m.HandleMessage(func(s *melody.Session, msg []byte) {
		m.Broadcast(msg)
	})

	r.Run(":5000")
}
```

[![Chat demo](https://cdn.rawgit.com/olahol/melody/master/examples/chat/demo.gif "Demo")](https://github.com/olahol/melody/tree/master/examples/chat)
