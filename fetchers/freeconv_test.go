package fetchers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	currency_fetcher "github.com/malusev998/currency"
)

type (
	httpHandler             struct{}
	httpHandlerLimitReached struct{}
	httpClientError         struct{}
	httpServerError         struct{}
)

func (h httpServerError) ServeHTTP(writer http.ResponseWriter, _ *http.Request) {
	writer.WriteHeader(503)
	_, _ = writer.Write([]byte("{\"error\": \"On vacation\", \"status\": 503}"))
}

func (h httpClientError) ServeHTTP(writer http.ResponseWriter, _ *http.Request) {
	writer.WriteHeader(422)
	_, _ = writer.Write([]byte("{\"error\": \"Validation error.\", \"status\": 422}"))
}

func (h httpHandlerLimitReached) ServeHTTP(writer http.ResponseWriter, _ *http.Request) {
	writer.WriteHeader(400)
	_, _ = writer.Write([]byte("{\"error\": \"Free API limit reached.\", \"status\": 400}"))
}

var Currencies = map[string]float32{"EUR_RSD": 117.4, "USD_EUR": 1.2, "EUR_USD": 0.8, "RSD_EUR": 0.001}

func (h httpHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	apiKey := request.URL.Query().Get("apiKey")
	if apiKey == "" {
		writer.WriteHeader(400)
		_, _ = writer.Write([]byte("{\"error\": \"API Key is required.\", \"status\": 400}"))
		return
	}

	query := strings.Split(request.URL.Query().Get("q"), ",")
	data := make(map[string]float32)

	for _, q := range query {
		data[q] = Currencies[q]
	}

	bytes, _ := json.Marshal(data)

	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(bytes)
}

func TestFreeCurrConvFetcher_Fetch(t *testing.T) {
	t.Parallel()
	server := httptest.NewUnstartedServer(httpHandler{})
	server.Start()
	defer server.Close()
	t.Run("Retrieves data from API", func(t *testing.T) {
		asserts := require.New(t)
		required := []string{"USD_EUR", "EUR_USD", "EUR_RSD", "RSD_EUR"}
		fetcher := FreeCurrConvFetcher{
			URL:           server.URL,
			APIKey:        "1234566789",
			MaxPerHour:    300,
			MaxPerRequest: 2,
		}

		currencies, err := fetcher.Fetch(required)

		asserts.Nilf(err, "Error while fetching currencies: %v", err)
		asserts.Lenf(currencies, 4, "Not enough currencies returned: %d", len(currencies))
		for i := 0; i < 4; i++ {
			pair := fmt.Sprintf("%s_%s", currencies[i].From, currencies[i].To)
			asserts.Contains(Currencies, pair)
			asserts.Equal(currency_fetcher.FreeConvProvider, currencies[i].Provider)
			asserts.Equal(Currencies[pair], currencies[i].Rate)
		}
	})

	t.Run("API key not found", func(t *testing.T) {
		asserts := require.New(t)
		fetcher := FreeCurrConvFetcher{
			URL:           server.URL,
			APIKey:        "",
			MaxPerHour:    300,
			MaxPerRequest: 2,
		}

		currencies, err := fetcher.Fetch([]string{"USD_EUR", "EUR_USD", "EUR_RSD", "RSD_EUR"})

		asserts.Nil(currencies)
		asserts.NotNil(err)
		asserts.True(errors.Is(err, ErrUnAuthorized))
	})

	t.Run("Not enough requests", func(t *testing.T) {
		asserts := require.New(t)
		fetcher := FreeCurrConvFetcher{
			URL:           server.URL,
			APIKey:        "",
			MaxPerHour:    1,
			MaxPerRequest: 2,
		}
		currencies, err := fetcher.Fetch([]string{"USD_EUR", "EUR_USD", "EUR_RSD", "RSD_EUR"})

		asserts.Nil(currencies)
		asserts.NotNil(err)
		asserts.True(errors.Is(err, ErrNotEnoughRequests))
	})
}

func TestApiLimitReached(t *testing.T) {
	t.Parallel()
	server := httptest.NewUnstartedServer(httpHandlerLimitReached{})
	server.Start()
	defer server.Close()

	asserts := require.New(t)
	fetcher := FreeCurrConvFetcher{
		URL:           server.URL,
		APIKey:        "1234567890",
		MaxPerHour:    300,
		MaxPerRequest: 2,
	}

	currencies, err := fetcher.Fetch([]string{"USD_EUR", "EUR_USD", "EUR_RSD", "RSD_EUR"})

	asserts.Nil(currencies)
	asserts.NotNil(err)
	asserts.True(errors.Is(err, ErrAPILimitReached))
}

func TestClientError(t *testing.T) {
	t.Parallel()
	server := httptest.NewUnstartedServer(httpClientError{})
	server.Start()
	defer server.Close()

	asserts := require.New(t)
	fetcher := FreeCurrConvFetcher{
		URL:           server.URL,
		APIKey:        "1234567890",
		MaxPerHour:    300,
		MaxPerRequest: 2,
	}

	currencies, err := fetcher.Fetch([]string{"USD_EUR", "EUR_USD", "EUR_RSD", "RSD_EUR"})

	asserts.Nil(currencies)
	asserts.NotNil(err)
	asserts.True(errors.Is(err, ErrClient))
}

func TestServerError(t *testing.T) {
	t.Parallel()
	server := httptest.NewUnstartedServer(httpServerError{})
	server.Start()
	defer server.Close()

	asserts := require.New(t)
	fetcher := FreeCurrConvFetcher{
		URL:           server.URL,
		APIKey:        "1234567890",
		MaxPerHour:    300,
		MaxPerRequest: 2,
	}

	currencies, err := fetcher.Fetch([]string{"USD_EUR", "EUR_USD", "EUR_RSD", "RSD_EUR"})

	asserts.Nil(currencies)
	asserts.NotNil(err)
	asserts.True(errors.Is(err, ErrServer))
}
