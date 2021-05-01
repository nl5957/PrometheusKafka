package kafka

import (
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/common/model"
)

// Client allows sending batches of Prometheus samples to Graphite.
type Client struct {
	logger 		log.Logger
	address   string
	timeout   time.Duration
}

// NewClient creates a new Client.
func NewClient(logger log.Logger, address string, timeout time.Duration) *Client {
	if logger == nil {
		logger = log.NewNopLogger()
	}
	return &Client{
		logger:    logger,
		address:   address,
		timeout:   timeout,
	}
}

// Write sends a batch of samples to Graphite.
func (c *Client) Write(samples model.Samples) error {
	for _, s := range samples {
		t := float64(s.Timestamp.UnixNano()) / 1e9
		v := float64(s.Value)
		level.Debug(c.logger).Log("msg", "Cannot send value to kafka, skipping sample", "value", v, "sample", s, "time", t)
	}


	return nil
}

// Name identifies the client as a Graphite client.
func (c Client) Name() string {
	return "kafka"
}
