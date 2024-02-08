package natsinfo

import (
	"encoding/json"
	"errors"

	nats "github.com/nats-io/nats.go"
)

func CreateStreamIfNotExists(js nats.JetStreamContext, config *nats.StreamConfig) (*nats.StreamInfo, error) {
	info, err := js.StreamInfo(config.Name)

	switch {
	case errors.Is(err, nats.ErrStreamNotFound):
		info, err = js.AddStream(config)
	}

	return info, err
}

func JsPublishJson(js nats.JetStreamContext, subject string, payload interface{}) (*nats.PubAck, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return js.Publish(subject, data)
}
