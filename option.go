package melody

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

type Option func(*websocket.Upgrader)

func WithReadBufferSize(size int) Option {
	return func(u *websocket.Upgrader) {
		u.ReadBufferSize = size
	}
}

func WithWriteBufferSize(size int) Option {
	return func(u *websocket.Upgrader) {
		u.WriteBufferSize = size
	}
}

func WithHandshakeTimeout(d time.Duration) Option {
	return func(u *websocket.Upgrader) {
		u.HandshakeTimeout = d
	}
}

func WithEnableCompression() Option {
	return func(u *websocket.Upgrader) {
		u.EnableCompression = true
	}
}

func WithSubprotocols(protocols []string) Option {
	return func(u *websocket.Upgrader) {
		u.Subprotocols = protocols
	}
}

func WithCheckOrigin(fn func(r *http.Request) bool) Option {
	return func(u *websocket.Upgrader) {
		u.CheckOrigin = fn
	}
}
