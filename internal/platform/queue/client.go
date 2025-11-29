package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const defaultStreamDescription = "Hauslet work queue stream"

// Client wraps a JetStream connection for publishing jobs.
type Client struct {
	js       jetstream.JetStream
	nc       *nats.Conn
	stream   string
	subjects []string
}

// New connects to NATS, ensures the stream exists, and returns a client.
// subjects are the subjects bound to the stream (e.g., []string{"email.send", "moderation.*"}).
func New(ctx context.Context, url, stream string, subjects []string) (*Client, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		_ = nc.Drain()
		return nil, fmt.Errorf("jetstream init: %w", err)
	}

	streamCfg := jetstream.StreamConfig{
		Name:        stream,
		Subjects:    subjects,
		Retention:   jetstream.WorkQueuePolicy,
		Description: defaultStreamDescription,
	}

	streamCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if _, err := js.CreateOrUpdateStream(streamCtx, streamCfg); err != nil {
		_ = nc.Drain()
		return nil, fmt.Errorf("ensure stream: %w", err)
	}

	return &Client{
		js:       js,
		nc:       nc,
		stream:   stream,
		subjects: subjects,
	}, nil
}

// Close drains and closes the underlying NATS connection.
func (c *Client) Close() {
	if c == nil || c.nc == nil {
		return
	}
	_ = c.nc.Drain()
}

// Publish marshals payload (any JSON-able value) and enqueues it on the subject.
func (c *Client) Publish(ctx context.Context, subject string, payload interface{}) error {
	if c == nil {
		return fmt.Errorf("queue client is nil")
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	if _, err := c.js.Publish(ctx, subject, data); err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	return nil
}

// JetStream exposes the underlying JetStream instance for consumers.
func (c *Client) JetStream() jetstream.JetStream {
	return c.js
}

// StreamName returns the stream name.
func (c *Client) StreamName() string {
	return c.stream
}
