package fetchers

import (
	"context"
	"encoding/json"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	currencyFetcher "github.com/malusev998/currency"
)

type FreeCurrConvFetcher struct {
	Ctx           context.Context
	URL           string
	APIKey        string
	MaxPerHour    int
	MaxPerRequest int
}

func (f FreeCurrConvFetcher) fetchCurrencies(
	client *http.Client,
	wg *sync.WaitGroup,
	currencies []string,
	channel currencyChannel,
	errorChannel chan<- error,
) {
	defer wg.Done()

	url := f.URL

	if url == "" {
		url = FreeConvFetchURL
	}

	ctx := f.Ctx

	if ctx == nil {
		ctx = context.Background()
	}

	req, formattedCurrencies, err := getData(ctx, url, currencies)

	if err != nil {
		errorChannel <- err
		return
	}

	q := req.URL.Query()
	q.Add("q", formattedCurrencies)
	q.Add("compact", "ultra")
	q.Add("apiKey", f.APIKey)

	req.URL.RawQuery = q.Encode()

	res, err := client.Do(req)

	if err != nil {
		errorChannel <- err
		return
	}

	defer res.Body.Close()

	var body []byte
	body, _ = ioutil.ReadAll(res.Body)

	if res.StatusCode == http.StatusOK {
		data := map[string]float32{}

		if err := json.Unmarshal(body, &data); err != nil {
			errorChannel <- err
			return
		}
		channel <- data

		return
	}

	if res.StatusCode == http.StatusBadRequest {
		errorRes := errorFreeConvResponse{}
		_ = json.Unmarshal(body, &errorRes)

		if strings.Contains(errorRes.Error, "required") {
			errorChannel <- ErrUnAuthorized
			return
		}

		if strings.Contains(errorRes.Error, "API limit reached") {
			errorChannel <- ErrAPILimitReached
			return
		}

		errorChannel <- errors.Wrap(ErrClient, errorRes.Error)

		return
	}

	if res.StatusCode >= http.StatusBadRequest && res.StatusCode < http.StatusInternalServerError {
		errorChannel <- ErrClient
		return
	}

	if res.StatusCode >= http.StatusInternalServerError {
		errorChannel <- ErrServer
		return
	}
	errorChannel <- ErrUnknown
}

func (f FreeCurrConvFetcher) Fetch(currenciesToFetch []string) ([]currencyFetcher.Currency, error) {
	var wg, appendWg sync.WaitGroup
	var numberOfRequests int

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if len(currenciesToFetch) < f.MaxPerRequest {
		numberOfRequests = len(currenciesToFetch)
	} else {
		numberOfRequests = len(currenciesToFetch) / f.MaxPerRequest
	}

	if numberOfRequests >= f.MaxPerHour {
		return nil, ErrNotEnoughRequests
	}

	channel := make(chan interface{})
	errorChannel := make(chan error)

	currencies := make([]currencyFetcher.Currency, 0, len(currenciesToFetch))

	client := &http.Client{}

	appendWg.Add(1)

	go appendToCurrencies(&appendWg, channel, &currencies, currencyFetcher.FreeConvProvider)

	idx := 0

	for i := 0; i < numberOfRequests; i++ {
		wg.Add(1)

		if numberOfRequests < f.MaxPerRequest {
			go f.fetchCurrencies(client, &wg, currenciesToFetch[idx:idx+1], channel, errorChannel)
		} else {
			go f.fetchCurrencies(client, &wg, currenciesToFetch[idx:idx+f.MaxPerRequest], channel, errorChannel)
		}

		idx += f.MaxPerRequest
	}

	go func(wg, appendWg *sync.WaitGroup) {
		wg.Wait()
		close(channel)
		appendWg.Wait()
		close(errorChannel)
	}(&wg, &appendWg)

	select {
	case err := <-errorChannel:
		cancel()

		if err != nil {
			return nil, err
		}

		return currencies, nil
	case <-ctx.Done():
		cancel()
		return currencies, nil
	}
}
