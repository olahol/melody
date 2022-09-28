package melody

import (
	"net/url"
	"time"
)

type clientHub struct {
	connectionUrl url.URL

	session *ClientSession
	client  *MelodyClient

	sendToServer chan *envelope
	exit         chan *envelope
	open         bool
}

func newClientHub(connectionUrl url.URL, client *MelodyClient) *clientHub {
	return &clientHub{
		client:        client,
		connectionUrl: connectionUrl,
		sendToServer:  make(chan *envelope),
		exit:          make(chan *envelope),
		open:          true,
	}
}

func (h *clientHub) run() {
	h.session = &ClientSession{
		output:     make(chan *envelope, h.client.Config.MessageBufferSize),
		outputDone: make(chan struct{}),
		melody:     h.client,
		open:       true,
	}

	go h.session.writePump()
	go h.writePump()

	h.session.connect(h.connectionUrl)

	h.client.disconnectHandler()
}

func (h *clientHub) writePump() {
	ticker := time.NewTicker(h.client.Config.PingPeriod)
	defer ticker.Stop()

loop:
	for {
		select {
		case m := <-h.sendToServer:
			h.session.writeMessage(m)
		case _ = <-h.exit:
			h.session.Close()
			h.open = false
			break loop

		case <-ticker.C:
			h.session.ping()
		}
	}
}

func (h *clientHub) closed() bool {
	return !h.open
}
