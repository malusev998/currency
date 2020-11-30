package fetchers

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"

	currencyFetcher "github.com/malusev998/currency"
)

const (
	FreeConvFetchURL    = "https://free.currconv.com/api/v7/convert"
	ExchangeRatesAPIURL = "https://api.exchangeratesapi.io/latest"
)

type (
	errorFreeConvResponse struct {
		Status int    `json:"status"`
		Error  string `json:"error"`
	}

	exchangeRateAPIResponse struct {
		Base  string             `json:"base,omitempty"`
		Rates map[string]float32 `json:"rates,omitempty"`
		Date  string             `json:"date,omitempty"`
	}

	currencyChannel chan interface{}
)

var (
	ErrUnAuthorized      = errors.New("unauthorized, API key is not provided")
	ErrNotEnoughRequests = errors.New("not enough requests per hour")
	ErrClient            = errors.New("client error")
	ErrServer            = errors.New("server error")
	ErrUnknown           = errors.New("unknown error")
	ErrAPILimitReached   = errors.New("API limit reached")
)

func getData(ctx context.Context, url string, currencies []string) (*http.Request, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)

	if err != nil {
		return nil, "", err
	}

	req.Header.Add("Accept", "application/json")

	var builder strings.Builder

	for _, c := range currencies {
		builder.WriteString(c)
		builder.WriteRune(',')
	}

	return req, strings.TrimRight(builder.String(), ","), nil
}

func appendToCurrencies(
	wg *sync.WaitGroup,
	c currencyChannel,
	currencies *[]currencyFetcher.Currency,
	provider currencyFetcher.Provider,
) {
	defer wg.Done()

	for data := range c {
		switch casted := data.(type) {
		case map[string]float32:
			for key, cur := range casted {
				isoCurrencies := strings.Split(key, "_")

				*currencies = append(*currencies, currencyFetcher.Currency{
					From:     isoCurrencies[0],
					To:       isoCurrencies[1],
					Provider: provider,
					Rate:     cur,
				})
			}
		case exchangeRateAPIResponse:
			for to, rate := range casted.Rates {
				*currencies = append(*currencies, currencyFetcher.Currency{
					From:     casted.Base,
					To:       to,
					Provider: provider,
					Rate:     rate,
				})
			}
		}
	}
}
