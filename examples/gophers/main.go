package main

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/olahol/melody"
)

type GopherInfo struct {
	ID, X, Y string
}

func main() {
	m := melody.New()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		m.HandleRequest(w, r)
	})

	m.HandleConnect(func(s *melody.Session) {
		ss, _ := m.Sessions()

		for _, o := range ss {
			value, exists := o.Get("info")

			if !exists {
				continue
			}

			info := value.(*GopherInfo)

			s.Write([]byte("set " + info.ID + " " + info.X + " " + info.Y))
		}

		id := uuid.NewString()
		s.Set("info", &GopherInfo{id, "0", "0"})

		s.Write([]byte("iam " + id))
	})

	m.HandleDisconnect(func(s *melody.Session) {
		value, exists := s.Get("info")

		if !exists {
			return
		}

		info := value.(*GopherInfo)

		m.BroadcastOthers([]byte("dis "+info.ID), s)
	})

	m.HandleMessage(func(s *melody.Session, msg []byte) {
		p := strings.Split(string(msg), " ")
		value, exists := s.Get("info")

		if len(p) != 2 || !exists {
			return
		}

		info := value.(*GopherInfo)
		info.X = p[0]
		info.Y = p[1]

		m.BroadcastOthers([]byte("set "+info.ID+" "+info.X+" "+info.Y), s)
	})

	http.ListenAndServe(":5000", nil)
}
