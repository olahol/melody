package main

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/olahol/melody"
	"net/http"
	"time"
)

func main() {
	e := echo.New()
	m := melody.New()

	e.HideBanner = true
	//e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/", func(c echo.Context) error {
		http.ServeFile(c.Response().Writer, c.Request(), "index.html")
		return nil
	})

	e.GET("/ws", func(c echo.Context) error {
		m.HandleRequest(c.Response().Writer, c.Request())
		return nil
	})

	m.HandleConnect(func(session *melody.Session) {
		log.Info("Client connected")
		time.Sleep(time.Second * 2)
		session.Write([]byte("you have connected"))
	})

	m.HandleDisconnect(func(session *melody.Session) {
		log.Info("Client disconnected")
	})

	m.HandlePong(func(session *melody.Session) {
		log.Info("Receive ping")
	})

	m.HandleMessage(func(s *melody.Session, msg []byte) {
		log.Info("Received client message: ", string(msg))
		m.Broadcast(msg)
	})

	e.Logger.Fatal(e.Start(":5000"))
}
