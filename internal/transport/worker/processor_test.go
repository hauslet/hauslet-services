package worker

import (
	"context"
	"sync"
	"testing"
	"time"

	"hauslet/config"
	"hauslet/internal/platform/email"
	"hauslet/internal/platform/logger"
	platformQueue "hauslet/internal/platform/queue"
	"hauslet/internal/queue"
	jobs "hauslet/internal/queue/jobs/emails"
	handlers "hauslet/internal/transport/worker/handlers/emails"

	"github.com/nats-io/nats-server/v2/test"
)

type stubSender struct {
	mu    sync.Mutex
	calls int
}

func (s *stubSender) SendHtml(_ context.Context, _, _, _ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	return nil
}

func (s *stubSender) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func TestProcessorEmailsFlow(t *testing.T) {
	opts := test.DefaultTestOptions
	opts.Port = -1
	opts.JetStream = true
	srv := test.RunServer(&opts)
	defer srv.Shutdown()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream := "EMAILS_TEST"
	subject := "email.test"

	qc, err := platformQueue.New(ctx, srv.ClientURL(), stream, []string{subject})
	if err != nil {
		t.Fatalf("queue init failed: %v", err)
	}
	defer qc.Close()

	sender := &stubSender{}
	mailClient := email.New(sender)
	log := logger.New()

	reg := queue.NewRegistry()
	reg.Register(handlers.NewEmailHandler(mailClient, log, subject))

	processor := NewProcessor(qc, reg, log, &config.GlobalConfig{})
	if err := processor.Start(ctx); err != nil {
		t.Fatalf("processor start failed: %v", err)
	}

	job := jobs.EmailJob{
		To:      "user@example.com",
		Subject: "hello",
		HTML:    "<p>hi</p>",
	}
	if err := qc.Publish(ctx, subject, job); err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for {
		if sender.Count() > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("email job was not processed in time")
		}
		time.Sleep(50 * time.Millisecond)
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer stopCancel()
	if err := processor.Stop(stopCtx); err != nil {
		t.Fatalf("processor stop failed: %v", err)
	}
}
