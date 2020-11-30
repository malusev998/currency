package fetchers

import (
	"context"

	currencyFetcher "github.com/malusev998/currency-fetcher"
)

type (
	BaseConfig struct {
		Ctx context.Context
		URL string
	}
	FreeConvServiceConfig struct {
		BaseConfig
		APIKey             string
		MaxPerHourRequests int
		MaxPerRequest      int
	}
	ExchangeRatesAPIConfig struct {
		BaseConfig
	}
)

func NewCurrencyFetcher(provider currencyFetcher.Provider, config interface{}) currencyFetcher.Fetcher {
	switch provider {
	case currencyFetcher.FreeConvProvider:
		c := config.(FreeConvServiceConfig)

		return FreeCurrConvFetcher{
			Ctx:           c.Ctx,
			URL:           c.URL,
			APIKey:        c.APIKey,
			MaxPerHour:    c.MaxPerHourRequests,
			MaxPerRequest: c.MaxPerRequest,
		}
	case currencyFetcher.ExchangeRatesAPIProvider:
		c := config.(ExchangeRatesAPIConfig)

		return ExchangeRatesAPIFetcher{
			Ctx: c.Ctx,
			URL: c.URL,
		}
	}

	return nil
}
