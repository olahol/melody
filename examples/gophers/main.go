package main

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/olahol/melody"
)

type GopherInfo struct {
	ID, X, Y string
}

func main() {
	router := gin.Default()
	mrouter := melody.New()

	router.GET("/", func(c *gin.Context) {
		http.ServeFile(c.Writer, c.Request, "index.html")
	})

	router.GET("/ws", func(c *gin.Context) {
		mrouter.HandleRequest(c.Writer, c.Request)
	})

	mrouter.HandleConnect(func(s *melody.Session) {
		ss, _ := mrouter.Sessions()

		for _, o := range ss {
			info := o.MustGet("info").(*GopherInfo)
			s.Write([]byte("set " + info.ID + " " + info.X + " " + info.Y))
		}

		id := uuid.NewString()
		s.Set("info", &GopherInfo{id, "0", "0"})

		s.Write([]byte("iam " + id))
	})

	mrouter.HandleDisconnect(func(s *melody.Session) {
		info := s.MustGet("info").(*GopherInfo)
		mrouter.BroadcastOthers([]byte("dis "+info.ID), s)
	})

	mrouter.HandleMessage(func(s *melody.Session, msg []byte) {
		p := strings.Split(string(msg), " ")
		if len(p) == 2 {
			info := s.MustGet("info").(*GopherInfo)
			info.X = p[0]
			info.Y = p[1]
			mrouter.BroadcastOthers([]byte("set "+info.ID+" "+info.X+" "+info.Y), s)
		}
	})

	router.Run(":5000")
}
