package currency

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sync"

	currencyFetcher "github.com/malusev998/currency-fetcher"
)

type (
	ExchangeRatesAPIFetcher struct {
		BaseCurrencies []string
		Ctx            context.Context
		URL            string
	}
)

func (e ExchangeRatesAPIFetcher) handleHTTPStatusCodeError(res *http.Response) error {
	if res.StatusCode != http.StatusOK {
		switch res.StatusCode {
		case http.StatusBadRequest:
			return ErrClient
		case http.StatusInternalServerError:
			return ErrServer
		default:
			return ErrUnknown
		}
	}

	return nil
}

func (e ExchangeRatesAPIFetcher) fetchCurrencies(
	ctx context.Context,
	client *http.Client,
	ratesCh currencyChannel,
	errCh chan<- error,
	url string,
	baseCurrency string,
	currenciesToFetch []string,

) {
	req, formattedCurrencies, err := getData(ctx, url, currenciesToFetch)

	if err != nil {
		errCh <- err
		return
	}

	q := req.URL.Query()
	q.Add("symbols", formattedCurrencies)
	q.Add("base", baseCurrency)

	req.URL.RawQuery = q.Encode()
	res, err := client.Do(req)

	if err != nil {
		errCh <- err
		return
	}

	if err := e.handleHTTPStatusCodeError(res); err != nil {
		errCh <- err
		return
	}

	var body []byte
	body, _ = ioutil.ReadAll(res.Body)

	var data exchangeRateAPIResponse

	if err := json.Unmarshal(body, &data); err != nil {
		errCh <- err
		return
	}
	ratesCh <- data

	if err := res.Body.Close(); err != nil {
		errCh <- err
		return
	}
}

func (e ExchangeRatesAPIFetcher) Fetch(currenciesToFetch []string) ([]currencyFetcher.Currency, error) {
	var wg sync.WaitGroup

	var appendWg sync.WaitGroup

	numberOfCurrenciesToFetch := len(currenciesToFetch) * len(e.BaseCurrencies)

	channel := make(currencyChannel, len(e.BaseCurrencies))
	errorChannel := make(chan error)
	currencies := make([]currencyFetcher.Currency, 0, numberOfCurrenciesToFetch)

	client := &http.Client{}

	appendWg.Add(1)

	go appendToCurrencies(&appendWg, channel, &currencies, currencyFetcher.ExchangeRatesAPIProvider)

	url := e.URL

	if url == "" {
		url = ExchangeRatesAPIURL
	}

	ctx := e.Ctx

	if ctx == nil {
		ctx = context.Background()
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, base := range e.BaseCurrencies {
		go e.fetchCurrencies(ctx, client, channel, errorChannel, url, base, currenciesToFetch)
	}

	go func(wg, appendWg *sync.WaitGroup, channel currencyChannel, errorChannel chan<- error) {
		wg.Wait()
		close(channel)
		appendWg.Wait()
		errorChannel <- nil
		close(errorChannel)
	}(&wg, &appendWg, channel, errorChannel)

	if err := <-errorChannel; err != nil {
		return nil, err
	}

	return currencies, nil
}
