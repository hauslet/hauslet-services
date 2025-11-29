package workers

import (
	"context"
	"fmt"
	"time"

	"hauslet/config"
	platformQueue "hauslet/internal/platform/queue"
	"hauslet/internal/queue"

	"github.com/go-pkgz/lgr"
	"github.com/nats-io/nats.go/jetstream"
)

// Processor handles message processing from NATS JetStream
type Processor struct {
	queue    *platformQueue.Client
	registry *queue.Registry
	log      *lgr.Logger
	cfg      *config.GlobalConfig
	consumer jetstream.Consumer
	done     chan struct{}
}

// NewProcessor creates a new message processor
func NewProcessor(q *platformQueue.Client, reg *queue.Registry, log *lgr.Logger, cfg *config.GlobalConfig) *Processor {
	return &Processor{
		queue:    q,
		registry: reg,
		log:      log,
		cfg:      cfg,
		done:     make(chan struct{}),
	}
}

// Start begins processing messages
func (p *Processor) Start(ctx context.Context) error {
	js := p.queue.JetStream()

	// Create consumer config for all registered subjects
	consumerCfg := jetstream.ConsumerConfig{
		Name:           "multi-worker",
		Durable:        "multi-worker",
		FilterSubjects: p.registry.Subjects(),
		AckPolicy:      jetstream.AckExplicitPolicy,
		MaxDeliver:     3, // Retry failed jobs 3 times
	}

	consumer, err := js.CreateOrUpdateConsumer(ctx, p.queue.StreamName(), consumerCfg)
	if err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
	}

	p.consumer = consumer
	p.log.Logf("INFO ✅ Consumer created for subjects: %v", p.registry.Subjects())

	// Start consuming
	go p.consume(ctx)

	return nil
}

// consume processes messages from the queue
func (p *Processor) consume(ctx context.Context) {
	msgs, err := p.consumer.Messages()
	if err != nil {
		p.log.Logf("ERROR Failed to get messages: %v", err)
		return
	}
	defer msgs.Stop()

	for {
		select {
		case <-ctx.Done():
			p.log.Logf("INFO Context cancelled, stopping consumer")
			close(p.done)
			return
		case <-p.done:
			p.log.Logf("INFO Processor stopped")
			return
		default:
			msg, err := msgs.Next()
			if err != nil {
				// Timeout or context expiration, continue
				time.Sleep(time.Second)
				continue
			}

			p.processMessage(ctx, msg)
		}
	}
}

// processMessage handles a single message
func (p *Processor) processMessage(ctx context.Context, msg jetstream.Msg) {
	subject := msg.Subject()

	p.log.Logf("INFO Processing job from subject: %s", subject)

	// Create context with timeout for job processing
	jobCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Route to appropriate handler
	if err := p.registry.Handle(jobCtx, subject, msg.Data()); err != nil {
		p.log.Logf("ERROR Job processing failed: %v", err)

		// Check if we should retry or terminate
		metadata, _ := msg.Metadata()
		if metadata.NumDelivered >= 3 {
			p.log.Logf("WARN Job failed after 3 attempts, terminating")
			msg.Term()
		} else {
			p.log.Logf("INFO Retrying job (attempt %d/3)", metadata.NumDelivered)
			msg.NakWithDelay(60 * time.Second)
		}
		return
	}

	// Success - acknowledge
	msg.Ack()
	p.log.Logf("INFO ✅ Job completed successfully")
}

// Stop gracefully stops the processor
func (p *Processor) Stop(ctx context.Context) error {
	p.log.Logf("INFO Stopping processor")
	close(p.done)

	// Wait for processing to finish or timeout
	select {
	case <-p.done:
		p.log.Logf("INFO Processor stopped cleanly")
	case <-ctx.Done():
		p.log.Logf("WARN Processor stop timeout")
	}

	return nil
}
