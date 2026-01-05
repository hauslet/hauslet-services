package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	cloudtaskspb "cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"google.golang.org/protobuf/types/known/durationpb"
)

type Config struct {
	ProjectID           string
	Location            string
	WorkerBaseURL       string
	ServiceAccountEmail string
	Environment         string
}

// Client wraps a Cloud Tasks client for publishing jobs.
type Client struct {
	client *cloudtasks.Client
	cfg    Config
	log    *slog.Logger
	routes map[string]QueueRoute
}

// New initializes a Cloud Tasks client with queue routing config.
func New(ctx context.Context, cfg Config, queueNames map[string]string, log *slog.Logger) (*Client, error) {
	if cfg.ProjectID == "" || cfg.Location == "" || cfg.WorkerBaseURL == "" {
		return nil, fmt.Errorf("cloud tasks config incomplete")
	}

	workerURL := strings.TrimRight(cfg.WorkerBaseURL, "/")
	cfg.WorkerBaseURL = workerURL

	c, err := cloudtasks.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("cloud tasks client: %w", err)
	}

	routes := BuildQueueRoutes(queueNames)
	if len(routes) == 0 {
		_ = c.Close()
		return nil, fmt.Errorf("no queue routes configured")
	}

	return &Client{
		client: c,
		cfg:    cfg,
		log:    log,
		routes: routes,
	}, nil
}

// Close closes the underlying Cloud Tasks client.
func (c *Client) Close() {
	if c == nil || c.client == nil {
		return
	}
	_ = c.client.Close()
}

// Publish marshals payload (any JSON-able value) and enqueues it on the queue name.
func (c *Client) Publish(ctx context.Context, queueName string, payload any) error {
	if c == nil {
		return fmt.Errorf("queue client is nil")
	}

	route, ok := c.routes[queueName]
	if !ok {
		return fmt.Errorf("queue route not configured for %s", queueName)
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	parent := fmt.Sprintf("projects/%s/locations/%s/queues/%s", c.cfg.ProjectID, c.cfg.Location, route.Name)
	url := c.cfg.WorkerBaseURL + route.Path

	req := &cloudtaskspb.CreateTaskRequest{
		Parent: parent,
		Task: &cloudtaskspb.Task{
			MessageType: &cloudtaskspb.Task_HttpRequest{
				HttpRequest: &cloudtaskspb.HttpRequest{
					HttpMethod: cloudtaskspb.HttpMethod_POST,
					Url:        url,
					Headers: map[string]string{
						"Content-Type": "application/json",
					},
					Body: body,
				},
			},
			DispatchDeadline: durationpb.New(route.Timeout),
		},
	}

	if c.cfg.ServiceAccountEmail != "" {
		// You must use the "AuthorizationHeader" field with the specific wrapper type
		req.Task.GetHttpRequest().AuthorizationHeader = &cloudtaskspb.HttpRequest_OidcToken{
			OidcToken: &cloudtaskspb.OidcToken{
				ServiceAccountEmail: c.cfg.ServiceAccountEmail,
			},
		}
	}

	if _, err := c.client.CreateTask(ctx, req); err != nil {
		if c.log != nil {
			c.log.Error("failed to publish task", "queue", queueName, "error", err)
		}
		return fmt.Errorf("publish: %w", err)
	}

	return nil
}

// AllowFallback returns true when direct fallback is allowed for queue failures.
func (c *Client) AllowFallback() bool {
	if c == nil {
		return true
	}
	return c.cfg.Environment != "production"
}

// QueueRoutes returns the resolved queue routes.
func (c *Client) QueueRoutes() map[string]QueueRoute {
	if c == nil {
		return nil
	}
	return c.routes
}
