package xchange

type XChange interface {
	ConvertCurrency(amount float64, fromCurrency string, toCurrency string) (float64, error)
	GetExchangeRate(fromCurrency string, toCurrency string) (float64, error)
}
