package melody

import (
	"bytes"
	"errors"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/quick"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

var TestMsg = []byte("test")

type TestServer struct {
	withKeys bool
	m        *Melody
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
	if s.withKeys {
		s.m.HandleRequestWithKeys(w, r, make(map[string]any))
	} else {
		s.m.HandleRequest(w, r)
	}
}

func NewDialer(url string) (*websocket.Conn, error) {
	dialer := &websocket.Dialer{}
	conn, _, err := dialer.Dial(strings.Replace(url, "http", "ws", 1), nil)
	return conn, err
}

func MustNewDialer(url string) *websocket.Conn {
	conn, err := NewDialer(url)

	if err != nil {
		panic("could not dail websocket")
	}

	return conn
}

func TestEcho(t *testing.T) {
	ws := NewTestServerHandler(func(session *Session, msg []byte) {
		session.Write(msg)
	})
	server := httptest.NewServer(ws)
	defer server.Close()

	fn := func(msg string) bool {
		conn := MustNewDialer(server.URL)
		defer conn.Close()

		conn.WriteMessage(websocket.TextMessage, []byte(msg))

		_, ret, err := conn.ReadMessage()

		assert.Nil(t, err)

		assert.Equal(t, msg, string(ret))

		return true
	}

	err := quick.Check(fn, nil)

	assert.Nil(t, err)
}

func TestEchoBinary(t *testing.T) {
	ws := NewTestServerHandler(func(session *Session, msg []byte) {
		session.WriteBinary(msg)
	})
	server := httptest.NewServer(ws)
	defer server.Close()

	fn := func(msg string) bool {
		conn := MustNewDialer(server.URL)
		defer conn.Close()

		conn.WriteMessage(websocket.TextMessage, []byte(msg))

		_, ret, err := conn.ReadMessage()

		assert.Nil(t, err)

		assert.True(t, bytes.Equal([]byte(msg), ret))

		return true
	}

	err := quick.Check(fn, nil)

	assert.Nil(t, err)
}

func TestWriteClosedServer(t *testing.T) {
	done := make(chan bool)

	ws := NewTestServer()

	server := httptest.NewServer(ws)
	defer server.Close()

	ws.m.HandleConnect(func(s *Session) {
		s.Close()
	})

	ws.m.HandleDisconnect(func(s *Session) {
		err := s.Write(TestMsg)

		assert.NotNil(t, err)
		close(done)
	})

	conn := MustNewDialer(server.URL)
	conn.ReadMessage()
	defer conn.Close()

	<-done
}

func TestWriteClosedClient(t *testing.T) {
	done := make(chan bool)

	ws := NewTestServer()

	server := httptest.NewServer(ws)
	defer server.Close()

	ws.m.HandleDisconnect(func(s *Session) {
		err := s.Write(TestMsg)

		assert.NotNil(t, err)
		close(done)
	})

	conn := MustNewDialer(server.URL)
	conn.Close()

	<-done
}

func TestUpgrader(t *testing.T) {
	ws := NewTestServer()
	ws.m.HandleMessage(func(session *Session, msg []byte) {
		session.Write(msg)
	})

	server := httptest.NewServer(ws)
	defer server.Close()

	ws.m.Upgrader = &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return false },
	}

	_, err := NewDialer(server.URL)

	assert.ErrorIs(t, err, websocket.ErrBadHandshake)
}

func TestBroadcast(t *testing.T) {
	n := 10
	msg := "test"

	test := func(h func(*TestServer), w func(*websocket.Conn)) {

		ws := NewTestServer()

		h(ws)

		server := httptest.NewServer(ws)
		defer server.Close()

		conn := MustNewDialer(server.URL)
		defer conn.Close()

		listeners := make([]*websocket.Conn, n)
		for i := range listeners {
			listener := MustNewDialer(server.URL)
			listeners[i] = listener
			defer listeners[i].Close()
		}

		w(conn)

		for _, listener := range listeners {
			_, ret, err := listener.ReadMessage()

			assert.Nil(t, err)

			assert.Equal(t, msg, string(ret))
		}
	}

	test(func(ws *TestServer) {
		ws.m.HandleMessage(func(s *Session, msg []byte) {
			ws.m.Broadcast(msg)
		})
	}, func(conn *websocket.Conn) {
		conn.WriteMessage(websocket.TextMessage, []byte(msg))
	})

	test(func(ws *TestServer) {
		ws.m.HandleMessageBinary(func(s *Session, msg []byte) {
			ws.m.BroadcastBinary(msg)
		})
	}, func(conn *websocket.Conn) {
		conn.WriteMessage(websocket.BinaryMessage, []byte(msg))
	})

	test(func(ws *TestServer) {
		ws.m.HandleMessage(func(s *Session, msg []byte) {
			ws.m.BroadcastFilter(msg, func(s *Session) bool {
				return true
			})
		})
	}, func(conn *websocket.Conn) {
		conn.WriteMessage(websocket.TextMessage, []byte(msg))
	})

	test(func(ws *TestServer) {
		ws.m.HandleMessageBinary(func(s *Session, msg []byte) {
			ws.m.BroadcastBinaryFilter(msg, func(s *Session) bool {
				return true
			})
		})
	}, func(conn *websocket.Conn) {
		conn.WriteMessage(websocket.BinaryMessage, []byte(msg))
	})

	test(func(ws *TestServer) {
		ws.m.HandleMessage(func(s *Session, msg []byte) {
			ws.m.BroadcastOthers(msg, s)
		})
	}, func(conn *websocket.Conn) {
		conn.WriteMessage(websocket.TextMessage, []byte(msg))
	})

	test(func(ws *TestServer) {
		ws.m.HandleMessageBinary(func(s *Session, msg []byte) {
			ws.m.BroadcastBinaryOthers(msg, s)
		})
	}, func(conn *websocket.Conn) {
		conn.WriteMessage(websocket.BinaryMessage, []byte(msg))
	})

	test(func(ws *TestServer) {
		ws.m.HandleMessage(func(s *Session, msg []byte) {
			ss, _ := ws.m.Sessions()
			ws.m.BroadcastMultiple(msg, ss)
		})
	}, func(conn *websocket.Conn) {
		conn.WriteMessage(websocket.TextMessage, []byte(msg))
	})
}

func TestClose(t *testing.T) {
	ws := NewTestServer()

	server := httptest.NewServer(ws)
	defer server.Close()

	n := 10

	conns := make([]*websocket.Conn, n)
	for i := range conns {
		conn := MustNewDialer(server.URL)
		conns[i] = conn
		defer conns[i].Close()
	}

	q := make(chan bool)
	ws.m.HandleDisconnect(func(s *Session) {
		q <- true
	})

	ws.m.Close()

	for _, conn := range conns {
		conn.ReadMessage()
	}

	assert.Zero(t, ws.m.Len())

	m := 0
	for range q {
		m += 1
		if m == n {
			break
		}
	}
}

func TestLen(t *testing.T) {
	rand.Seed(time.Now().UnixNano())

	connect := int(rand.Int31n(100))
	disconnect := rand.Float32()
	conns := make([]*websocket.Conn, connect)

	defer func() {
		for _, conn := range conns {
			if conn != nil {
				conn.Close()
			}
		}
	}()

	ws := NewTestServer()

	server := httptest.NewServer(ws)
	defer server.Close()

	disconnected := 0
	for i := 0; i < connect; i++ {
		conn := MustNewDialer(server.URL)

		if rand.Float32() < disconnect {
			conns[i] = nil
			disconnected++
			conn.Close()
			continue
		}

		conns[i] = conn
	}

	time.Sleep(time.Millisecond)

	connected := connect - disconnected

	assert.Equal(t, ws.m.Len(), connected)
}

func TestSessions(t *testing.T) {
	rand.Seed(time.Now().UnixNano())

	connect := int(rand.Int31n(100))
	disconnect := rand.Float32()
	conns := make([]*websocket.Conn, connect)
	defer func() {
		for _, conn := range conns {
			if conn != nil {
				conn.Close()
			}
		}
	}()

	ws := NewTestServer()
	server := httptest.NewServer(ws)
	defer server.Close()

	disconnected := 0
	for i := 0; i < connect; i++ {
		conn, err := NewDialer(server.URL)

		if err != nil {
			t.Error(err)
		}

		if rand.Float32() < disconnect {
			conns[i] = nil
			disconnected++
			conn.Close()
			continue
		}

		conns[i] = conn
	}

	time.Sleep(time.Millisecond)

	connected := connect - disconnected

	ss, err := ws.m.Sessions()

	assert.Nil(t, err)

	assert.Equal(t, len(ss), connected)
}

func TestPingPong(t *testing.T) {
	done := make(chan bool)

	ws := NewTestServer()
	ws.m.Config.PingPeriod = time.Millisecond

	ws.m.HandlePong(func(s *Session) {
		close(done)
	})

	server := httptest.NewServer(ws)
	defer server.Close()

	conn := MustNewDialer(server.URL)
	defer conn.Close()

	go conn.NextReader()

	<-done
}

func TestHandleClose(t *testing.T) {
	done := make(chan bool)

	ws := NewTestServer()
	ws.m.Config.PingPeriod = time.Millisecond

	ws.m.HandleClose(func(s *Session, code int, text string) error {
		close(done)
		return nil
	})

	server := httptest.NewServer(ws)
	defer server.Close()

	conn := MustNewDialer(server.URL)

	conn.WriteMessage(websocket.CloseMessage, nil)

	<-done
}

func TestHandleError(t *testing.T) {
	done := make(chan bool)

	ws := NewTestServer()

	ws.m.HandleError(func(s *Session, err error) {
		var closeError *websocket.CloseError
		assert.ErrorAs(t, err, &closeError)
		close(done)
	})

	server := httptest.NewServer(ws)
	defer server.Close()

	conn := MustNewDialer(server.URL)

	conn.Close()

	<-done
}

func TestHandleErrorWrite(t *testing.T) {
	writeError := make(chan struct{})
	disconnect := make(chan struct{})

	ws := NewTestServer()
	ws.m.Config.WriteWait = 0

	ws.m.HandleConnect(func(s *Session) {
		err := s.Write(TestMsg)
		assert.Nil(t, err)
	})

	ws.m.HandleError(func(s *Session, err error) {
		assert.NotNil(t, err)

		if os.IsTimeout(err) {
			close(writeError)
		}
	})

	ws.m.HandleDisconnect(func(s *Session) {
		close(disconnect)
	})

	server := httptest.NewServer(ws)
	defer server.Close()

	conn := MustNewDialer(server.URL)
	defer conn.Close()

	go conn.NextReader()

	<-writeError
	<-disconnect
}

func TestErrClosed(t *testing.T) {
	res := make(chan *Session)

	ws := NewTestServer()

	ws.m.HandleConnect(func(s *Session) {
		ws.m.CloseWithMsg(TestMsg)
	})

	ws.m.HandleDisconnect(func(s *Session) {
		res <- s
	})

	server := httptest.NewServer(ws)
	defer server.Close()

	conn := MustNewDialer(server.URL)
	defer conn.Close()

	go conn.ReadMessage()

	s := <-res

	assert.True(t, s.IsClosed())
	assert.True(t, ws.m.IsClosed())
	_, err := ws.m.Sessions()
	assert.ErrorIs(t, err, ErrClosed)
	assert.ErrorIs(t, ws.m.Close(), ErrClosed)
	assert.ErrorIs(t, ws.m.CloseWithMsg(TestMsg), ErrClosed)

	assert.ErrorIs(t, ws.m.Broadcast(TestMsg), ErrClosed)
	assert.ErrorIs(t, ws.m.BroadcastBinary(TestMsg), ErrClosed)
	assert.ErrorIs(t, ws.m.BroadcastFilter(TestMsg, func(s *Session) bool { return true }), ErrClosed)
	assert.ErrorIs(t, ws.m.BroadcastBinaryFilter(TestMsg, func(s *Session) bool { return true }), ErrClosed)
	assert.ErrorIs(t, ws.m.HandleRequest(nil, nil), ErrClosed)
}

func TestErrSessionClosed(t *testing.T) {
	res := make(chan *Session)

	ws := NewTestServer()

	ws.m.HandleConnect(func(s *Session) {
		s.CloseWithMsg(TestMsg)
	})

	ws.m.HandleDisconnect(func(s *Session) {
		res <- s
	})

	server := httptest.NewServer(ws)
	defer server.Close()

	conn := MustNewDialer(server.URL)
	defer conn.Close()

	go conn.ReadMessage()

	s := <-res

	assert.True(t, s.IsClosed())
	assert.ErrorIs(t, s.Write(TestMsg), ErrSessionClosed)
	assert.ErrorIs(t, s.WriteBinary(TestMsg), ErrSessionClosed)
	assert.ErrorIs(t, s.CloseWithMsg(TestMsg), ErrSessionClosed)
	assert.ErrorIs(t, s.Close(), ErrSessionClosed)
	assert.ErrorIs(t, ws.m.BroadcastMultiple(TestMsg, []*Session{s}), ErrSessionClosed)

	assert.ErrorIs(t, s.writeRaw(nil), ErrWriteClosed)
	s.writeMessage(nil)
}

func TestErrMessageBufferFull(t *testing.T) {
	done := make(chan bool)

	ws := NewTestServerHandler(func(session *Session, msg []byte) {
		session.Write(msg)
		session.Write(msg)
	})
	ws.m.Config.MessageBufferSize = 0
	ws.m.HandleError(func(s *Session, err error) {
		if errors.Is(err, ErrMessageBufferFull) {
			close(done)
		}
	})
	server := httptest.NewServer(ws)
	defer server.Close()

	conn := MustNewDialer(server.URL)
	defer conn.Close()

	conn.WriteMessage(websocket.TextMessage, TestMsg)

	<-done
}

func TestSessionKeys(t *testing.T) {
	ws := NewTestServer()

	ws.m.HandleConnect(func(session *Session) {
		session.Set("stamp", time.Now().UnixNano())
	})
	ws.m.HandleMessage(func(session *Session, msg []byte) {
		stamp := session.MustGet("stamp").(int64)
		session.Write([]byte(strconv.Itoa(int(stamp))))
	})
	server := httptest.NewServer(ws)
	defer server.Close()

	fn := func(msg string) bool {
		conn := MustNewDialer(server.URL)
		defer conn.Close()

		conn.WriteMessage(websocket.TextMessage, []byte(msg))

		_, ret, err := conn.ReadMessage()

		assert.Nil(t, err)

		stamp, err := strconv.Atoi(string(ret))

		assert.Nil(t, err)

		diff := int(time.Now().UnixNano()) - stamp

		assert.Greater(t, diff, 0)

		return true
	}

	assert.Nil(t, quick.Check(fn, nil))
}

func TestSessionKeysConcurrent(t *testing.T) {
	ss := make(chan *Session)

	ws := NewTestServer()

	ws.m.HandleConnect(func(s *Session) {
		ss <- s
	})

	server := httptest.NewServer(ws)
	defer server.Close()

	conn := MustNewDialer(server.URL)
	defer conn.Close()

	s := <-ss

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)

		go func() {
			s.Set("test", TestMsg)

			v1, exists := s.Get("test")

			assert.True(t, exists)
			assert.Equal(t, v1, TestMsg)

			v2 := s.MustGet("test")

			assert.Equal(t, v1, v2)

			wg.Done()
		}()
	}

	wg.Wait()

	for i := 0; i < 100; i++ {
		wg.Add(1)

		go func() {
			s.UnSet("test")

			_, exists := s.Get("test")

			assert.False(t, exists)

			wg.Done()
		}()
	}

	wg.Wait()
}

func TestMisc(t *testing.T) {
	res := make(chan *Session)

	ws := NewTestServer()

	ws.m.HandleConnect(func(s *Session) {
		res <- s
	})

	server := httptest.NewServer(ws)
	defer server.Close()

	conn := MustNewDialer(server.URL)
	defer conn.Close()

	go conn.ReadMessage()

	s := <-res

	assert.Contains(t, s.LocalAddr().String(), "127.0.0.1")
	assert.Contains(t, s.RemoteAddr().String(), "127.0.0.1")
	assert.Equal(t, FormatCloseMessage(websocket.CloseMessage, "test"), websocket.FormatCloseMessage(websocket.CloseMessage, "test"))
	assert.Panics(t, func() {
		s.MustGet("test")
	})
}

func TestHandleSentMessage(t *testing.T) {
	test := func(h func(*TestServer, chan bool), w func(*websocket.Conn)) {
		done := make(chan bool)

		ws := NewTestServer()
		server := httptest.NewServer(ws)
		defer server.Close()

		h(ws, done)

		conn := MustNewDialer(server.URL)
		defer conn.Close()

		w(conn)

		<-done
	}

	test(func(ws *TestServer, done chan bool) {
		ws.m.HandleMessage(func(s *Session, msg []byte) {
			s.Write(msg)
		})

		ws.m.HandleSentMessage(func(s *Session, msg []byte) {
			assert.Equal(t, TestMsg, msg)
			close(done)
		})
	}, func(conn *websocket.Conn) {
		conn.WriteMessage(websocket.TextMessage, TestMsg)
	})

	test(func(ws *TestServer, done chan bool) {
		ws.m.HandleMessageBinary(func(s *Session, msg []byte) {
			s.WriteBinary(msg)
		})

		ws.m.HandleSentMessageBinary(func(s *Session, msg []byte) {
			assert.Equal(t, TestMsg, msg)
			close(done)
		})
	}, func(conn *websocket.Conn) {
		conn.WriteMessage(websocket.BinaryMessage, TestMsg)
	})
}

func TestConcurrentMessageHandling(t *testing.T) {
	testTimeout := func(cmh bool, msgType int) bool {
		base := time.Millisecond * 100
		done := make(chan struct{})

		handler := func(s *Session, msg []byte) {
			if len(msg) == 0 {
				done <- struct{}{}
				return
			}

			time.Sleep(base * 2)
		}

		ws := NewTestServerHandler(func(session *Session, msg []byte) {})
		if msgType == websocket.TextMessage {
			ws.m.HandleMessage(handler)
		} else {
			ws.m.HandleMessageBinary(handler)
		}

		ws.m.Config.ConcurrentMessageHandling = cmh
		ws.m.Config.PongWait = base

		var errorSet atomic.Bool
		ws.m.HandleError(func(s *Session, err error) {
			errorSet.Store(true)
			done <- struct{}{}
		})

		server := httptest.NewServer(ws)
		defer server.Close()

		conn := MustNewDialer(server.URL)
		defer conn.Close()

		conn.WriteMessage(msgType, TestMsg)
		conn.WriteMessage(msgType, TestMsg)

		time.Sleep(base / 4)

		conn.WriteMessage(msgType, nil)

		<-done

		return errorSet.Load()
	}

	t.Run("text should error", func(t *testing.T) {
		errorSet := testTimeout(false, websocket.TextMessage)

		if !errorSet {
			t.FailNow()
		}
	})

	t.Run("text should not error", func(t *testing.T) {
		errorSet := testTimeout(true, websocket.TextMessage)

		if errorSet {
			t.FailNow()
		}
	})

	t.Run("binary should error", func(t *testing.T) {
		errorSet := testTimeout(false, websocket.BinaryMessage)

		if !errorSet {
			t.FailNow()
		}
	})

	t.Run("binary should not error", func(t *testing.T) {
		errorSet := testTimeout(true, websocket.BinaryMessage)

		if errorSet {
			t.FailNow()
		}
	})
}
