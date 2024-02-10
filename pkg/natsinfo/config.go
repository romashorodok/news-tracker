package natsinfo

import (
	"fmt"
	"log"
	"time"

	nats "github.com/nats-io/nats.go"
	"go.uber.org/fx"
)

type NatsConfig struct {
	Port string
	Host string
}

func (c *NatsConfig) GetURL() string {
	if c.Host == "" || c.Port == "" {
		return nats.DefaultURL
	}
	return fmt.Sprintf("nats://%s:%s", c.Host, c.Port)
}

func NewNatsConfig() *NatsConfig {
	return &NatsConfig{
		Host: "nats",
		Port: "4222",
	}
}

type NewNatsConnectionResult struct {
	fx.Out

	Conn *nats.Conn
	JS   nats.JetStreamContext
}

func NewNatsConnection(config *NatsConfig) (NewNatsConnectionResult, error) {
	conn, err := nats.Connect(config.GetURL(),
		nats.Timeout(time.Second*30),
		nats.RetryOnFailedConnect(true),
	)
	if err != nil {
		return NewNatsConnectionResult{}, err
	}

	js, err := conn.JetStream()
	if err != nil {
		return NewNatsConnectionResult{}, err
	}

	wait := make(chan struct{})
	go func() {
		defer close(wait)
		ticker := time.NewTicker(time.Millisecond * 10)
		done := time.NewTimer(time.Second * 30)
		for {
			log.Printf("NATS connection state: %s", conn.Status())
			select {
			case <-done.C:
				panic("Unable establish nats connection")
			case <-ticker.C:
				if nats.CONNECTED == conn.Status() {
					return
				}
			}
		}
	}()
	<-wait

	return NewNatsConnectionResult{
		Conn: conn,
		JS:   js,
	}, nil
}
