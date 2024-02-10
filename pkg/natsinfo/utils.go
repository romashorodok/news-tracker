package natsinfo

import (
	"errors"

	nats "github.com/nats-io/nats.go"
)

func CreateOrUpdateStream(js nats.JetStreamContext, config *nats.StreamConfig) (*nats.StreamInfo, error) {
	info, err := js.AddStream(config)

	switch {
	case errors.Is(err, nats.ErrStreamNameAlreadyInUse):
		info, err = js.UpdateStream(config)
	}

	return info, err
}

func CreateOrUpdateConsumer(js nats.JetStreamContext, stream string, config *nats.ConsumerConfig, opts ...nats.JSOpt) (*nats.ConsumerInfo, error) {
	info, err := js.AddConsumer(stream, config, opts...)

	switch {
	case errors.Is(err, nats.ErrConsumerNameAlreadyInUse):
		info, err = js.UpdateConsumer(stream, config, opts...)
	}

	return info, err
}
