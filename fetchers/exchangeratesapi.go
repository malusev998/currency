package fetchers

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	"github.com/malusev998/currency"
)

type (
	ExchangeRatesAPIFetcher struct {
		Ctx context.Context
		URL string
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
	wg *sync.WaitGroup,
	client *http.Client,
	ratesCh currencyChannel,
	errCh chan<- error,
	url string,
	baseCurrency string,
	currenciesToFetch []string,
) {
	defer wg.Done()
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

func (e ExchangeRatesAPIFetcher) PrepareISOCurrencies(currencies []string) map[string][]string {
	var cs []string
	var exists bool

	mappedCurrencies := make(map[string][]string)

	for _, c := range currencies {
		isoCurrency := strings.Split(c, "_")
		if cs, exists = mappedCurrencies[isoCurrency[0]]; exists && len(cs) != 0 {
			cs = append(cs, isoCurrency[1])
		} else {
			cs = []string{isoCurrency[1]}
		}

		mappedCurrencies[isoCurrency[0]] = cs
	}

	return mappedCurrencies
}

func (e ExchangeRatesAPIFetcher) Fetch(currenciesToFetch []string) ([]currency.Currency, error) {
	var wg, appendWg sync.WaitGroup
	currencies := e.PrepareISOCurrencies(currenciesToFetch)

	channel := make(currencyChannel, len(currencies))
	errorChannel := make(chan error)

	result := make([]currency.Currency, 0)
	client := &http.Client{}

	appendWg.Add(1)
	go appendToCurrencies(&appendWg, channel, &result, currency.ExchangeRatesAPIProvider)
	url := e.URL

	if url == "" {
		url = ExchangeRatesAPIURL
	}

	ctx := e.Ctx

	if ctx == nil {
		ctx = context.Background()
	}

	ctx, cancel := context.WithCancel(ctx)
	for base, curs := range currencies {
		wg.Add(1)
		go e.fetchCurrencies(ctx, &wg, client, channel, errorChannel, url, base, curs)
	}

	go func(wg, appendWg *sync.WaitGroup) {
		wg.Wait()
		close(channel)
		appendWg.Wait()
		close(errorChannel)
		cancel()
	}(&wg, &appendWg)

	select {
	case err := <-errorChannel:
		cancel()

		if err != nil {
			return nil, err
		}

		return result, err
	case <-ctx.Done():
		return result, nil
	}
}
