package aimoderation

import "context"

type Client struct {
	provider AIClient
}

func New(provider AIClient) *Client {
	return &Client{provider: provider}
}

func (c *Client) Provider() string {
	return c.provider.Provider()
}

func (c *Client) Moderate(ctx context.Context, input AIModerationInput) (*AIModerationResult, error) {
	return c.provider.Moderate(ctx, input)
}
func (c *Client) HealthCheck(ctx context.Context) error {
	return c.provider.HealthCheck(ctx)
}
