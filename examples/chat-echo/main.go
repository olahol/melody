package main

import (
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"gopkg.in/olahol/melody.v1"
	"net/http"
)

func main() {
	e := echo.New()
	m := melody.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/", func(c echo.Context) error {
		http.ServeFile(c.Response().Writer, c.Request(), "index.html")
		return nil
	})

	e.GET("/ws", func(c echo.Context) error {
		m.HandleRequest(c.Response().Writer, c.Request())
		return nil
	})

	m.HandleMessage(func(s *melody.Session, msg []byte) {
		m.Broadcast(msg)
	})
	e.Logger.Fatal(e.Start(":5000"))
}
