package melody

import "time"

type Option func(config *Config)

// Config melody configuration struct.
type Config struct {
	WriteWait         time.Duration // Milliseconds until write times out.
	PongWait          time.Duration // Timeout for waiting on pong.
	PingPeriod        time.Duration // Milliseconds between pings.
	MaxMessageSize    int64         // Maximum size in bytes of a message.
	MessageBufferSize int           // The max amount of messages that can be in a sessions buffer before it starts dropping them.
}

func newConfig() *Config {
	return &Config{
		WriteWait:         10 * time.Second,
		PongWait:          60 * time.Second,
		PingPeriod:        (60 * time.Second * 9) / 10,
		MaxMessageSize:    512,
		MessageBufferSize: 256,
	}
}

// WithMaxMessageSize sets the write wait time.
func WithMaxMessageSize(maxMessageSize int64) Option {
	return func(c *Config) {
		c.MaxMessageSize = maxMessageSize
	}
}

// WithMessageBufferSize sets the message buffer size.
func WithMessageBufferSize(messageBufferSize int) Option {
	return func(c *Config) {
		c.MessageBufferSize = messageBufferSize
	}
}

// WithWriteWait sets the write wait time.
func WithWriteWait(writeWait time.Duration) Option {
	return func(c *Config) {
		c.WriteWait = writeWait
	}
}

// WithPongWait sets the pong wait time.
func WithPongWait(pongWait time.Duration) Option {
	return func(c *Config) {
		c.PongWait = pongWait
	}
}

// WithPingPeriod sets the ping period time.
func WithPingPeriod(pingPeriod time.Duration) Option {
	return func(c *Config) {
		c.PingPeriod = pingPeriod
	}
}
