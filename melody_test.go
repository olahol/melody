package melody

import (
	"github.com/gorilla/websocket"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/quick"
	"time"
)

type TestServer struct {
	m *Melody
}

func NewTestServerHandler(handler handleMessageFunc) *TestServer {
	m := New()
	m.HandleMessage(handler)
	return &TestServer{
		m: m,
	}
}

func NewTestServer() *TestServer {
	m := New()
	return &TestServer{
		m: m,
	}
}

func (s *TestServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.m.HandleRequest(w, r)
}

func NewDialer(url string) (*websocket.Conn, error) {
	dialer := &websocket.Dialer{}
	conn, _, err := dialer.Dial(strings.Replace(url, "http", "ws", 1), nil)
	return conn, err
}

func TestEcho(t *testing.T) {
	echo := NewTestServerHandler(func(session *Session, msg []byte) {
		session.Write(msg)
	})
	server := httptest.NewServer(echo)
	defer server.Close()

	fn := func(msg string) bool {
		conn, err := NewDialer(server.URL)
		defer conn.Close()

		if err != nil {
			t.Error(err)
			return false
		}

		conn.WriteMessage(websocket.TextMessage, []byte(msg))

		_, ret, err := conn.ReadMessage()

		if err != nil {
			t.Error(err)
			return false
		}

		if msg != string(ret) {
			t.Errorf("%s should equal %s", msg, string(ret))
			return false
		}

		return true
	}

	if err := quick.Check(fn, nil); err != nil {
		t.Error(err)
	}
}

func TestUpgrader(t *testing.T) {
	broadcast := NewTestServer()
	broadcast.m.HandleMessage(func(session *Session, msg []byte) {
		session.Write(msg)
	})
	server := httptest.NewServer(broadcast)
	defer server.Close()

	broadcast.m.Upgrader = &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return false },
	}

	broadcast.m.HandleError(func(session *Session, err error) {
		if err == nil || err.Error() != "websocket: origin not allowed" {
			t.Error("there should be a origin error")
		}
	})

	_, err := NewDialer(server.URL)

	if err == nil || err.Error() != "websocket: bad handshake" {
		t.Error("there should be a badhandshake error")
	}
}

func TestBroadcast(t *testing.T) {
	broadcast := NewTestServer()
	broadcast.m.HandleMessage(func(session *Session, msg []byte) {
		broadcast.m.Broadcast(msg)
	})
	server := httptest.NewServer(broadcast)
	defer server.Close()

	n := 10

	fn := func(msg string) bool {
		conn, _ := NewDialer(server.URL)
		defer conn.Close()

		listeners := make([]*websocket.Conn, n)
		for i := 0; i < n; i++ {
			listener, _ := NewDialer(server.URL)
			listeners[i] = listener
			defer listeners[i].Close()
		}

		conn.WriteMessage(websocket.TextMessage, []byte(msg))

		for i := 0; i < n; i++ {
			_, ret, err := listeners[i].ReadMessage()

			if err != nil {
				t.Error(err)
				return false
			}

			if msg != string(ret) {
				t.Errorf("%s should equal %s", msg, string(ret))
				return false
			}
		}

		return true
	}

	if err := quick.Check(fn, nil); err != nil {
		t.Error(err)
	}
}

func TestPingPong(t *testing.T) {
	noecho := NewTestServer()
	noecho.m.Config.PongWait = time.Second
	noecho.m.Config.PingPeriod = time.Second * 9 / 10
	server := httptest.NewServer(noecho)
	defer server.Close()

	conn, err := NewDialer(server.URL)
	conn.SetPingHandler(func(string) error {
		return nil
	})
	defer conn.Close()

	if err != nil {
		t.Error(err)
	}

	conn.WriteMessage(websocket.TextMessage, []byte("test"))

	_, _, err = conn.ReadMessage()

	if err == nil {
		t.Error("there should be an error")
	}
}

/*
func TestBroadcastFilter(t *testing.T) {
	echo := NewTestServer()
	echo.m.HandleMessage(func(session *Session, msg []byte) {
		echo.m.BroadcastFilter(func(s *Session) bool {
			//return s == session
			return false
		}, msg)
	})
	server := httptest.NewServer(echo)
	defer server.Close()

	fn := func(msg string) bool {
		conn, err := NewDialer(server.URL)
		conn.SetPingHandler(func(string) error {
			return nil
		})
		defer conn.Close()

		if err != nil {
			t.Error(err)
			return false
		}

		conn.WriteMessage(websocket.TextMessage, []byte(msg))

		_, ret, err := conn.ReadMessage()

		if err != nil {
			t.Error(err)
			return false
		}

		if msg != string(ret) {
			t.Errorf("%s should equal %s", msg, string(ret))
			return false
		}

		return true
	}

	if err := quick.Check(fn, nil); err != nil {
		t.Error(err)
	}
}
*/

func BenchmarkEcho(b *testing.B) {
	echo := NewTestServerHandler(func(session *Session, msg []byte) {
		session.Write(msg)
	})
	server := httptest.NewServer(echo)
	defer server.Close()

	conn, _ := NewDialer(server.URL)
	defer conn.Close()

	for i := 0; i < b.N; i++ {
		conn.WriteMessage(websocket.TextMessage, []byte("test"))
		conn.ReadMessage()
	}
}

func BenchmarkBroadcast(b *testing.B) {
	broadcast := NewTestServer()
	broadcast.m.HandleMessage(func(session *Session, msg []byte) {
		broadcast.m.Broadcast(msg)
	})
	server := httptest.NewServer(broadcast)
	defer server.Close()

	conn, _ := NewDialer(server.URL)
	defer conn.Close()

	n := 10
	listeners := make([]*websocket.Conn, n)
	for i := 0; i < n; i++ {
		listener, _ := NewDialer(server.URL)
		listeners[i] = listener
		defer listeners[i].Close()
	}

	for i := 0; i < b.N; i++ {
		conn.WriteMessage(websocket.TextMessage, []byte("test"))
		for i := 0; i < n; i++ {
			listeners[i].ReadMessage()
		}
	}
}
