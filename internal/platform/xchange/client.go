package xchange

// Client is a high-level wrapper that allows swapping FX providers.
type Client struct {
	provider XChange
}

// New creates a new FX client with the given provider.
func New(provider XChange) *Client {
	return &Client{provider: provider}
}

// ConvertCurrency delegates to the configured provider.
func (c *Client) ConvertCurrency(amount float64, fromCurrency, toCurrency string) (float64, error) {
	return c.provider.ConvertCurrency(amount, fromCurrency, toCurrency)
}

// GetExchangeRate delegates to the configured provider.
func (c *Client) GetExchangeRate(fromCurrency, toCurrency string) (float64, error) {
	return c.provider.GetExchangeRate(fromCurrency, toCurrency)
}
