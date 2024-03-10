package main

import (
	"net/http"
	"os"

	"github.com/fsnotify/fsnotify"
	"github.com/olahol/melody"
)

func main() {
	file := "file.txt"

	m := melody.New()
	w, _ := fsnotify.NewWatcher()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		m.HandleRequest(w, r)
	})

	m.HandleConnect(func(s *melody.Session) {
		content, _ := os.ReadFile(file)
		s.Write(content)
	})

	go func() {
		for {
			ev := <-w.Events
			if ev.Op == fsnotify.Write {
				content, _ := os.ReadFile(ev.Name)
				m.Broadcast(content)
			}
		}
	}()

	w.Add(file)

	http.ListenAndServe(":5000", nil)
}
