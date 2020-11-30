package fetchers_test

import (
	"context"
	"github.com/malusev998/currency-fetcher/fetchers"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestExchangeRatesAPIFetcher_PrepareISOCurrencies(t *testing.T) {
	assert := require.New(t)
	api := fetchers.ExchangeRatesAPIFetcher{Ctx: context.Background()}

	result := api.PrepareISOCurrencies([]string{"EUR_USD", "EUR_RSD", "RSD_EUR", "RSD_USD", "USD_EUR", "USD_RSD"})

	assert.NotEmpty(result)
	assert.Contains(result, "RSD")
	assert.Contains(result, "EUR")
	assert.Contains(result, "USD")
	assert.EqualValues(result["RSD"], []string{"EUR", "USD"})
	assert.EqualValues(result["EUR"], []string{"USD", "RSD"})
	assert.EqualValues(result["USD"], []string{"EUR", "RSD"})
}
