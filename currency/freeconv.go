package currency

import (
	"encoding/json"
	"errors"
	"github.com/BrosSquad/currency-fetcher"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
)

const (
	FreeConvFetchUrl = "https://free.currconv.com/api/v7/convert"
	FreeConvProvider = "FreeCurrConversion"
)

var (
	ErrUnAuthorized      = errors.New("unauthorized, API key is not provided")
	ErrNotEnoughRequests = errors.New("not enough requests per hour")
	ErrClient            = errors.New("client error")
	ErrServer            = errors.New("server error")
	ErrUnknown           = errors.New("unknown error")
	ErrApiLimitReached   = errors.New("API limit reached")
)

type FreeCurrConvFetcher struct {
	Url           string
	ApiKey        string
	MaxPerHour    int
	MaxPerRequest int
}

func (f FreeCurrConvFetcher) getData(url string, currencies []string) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)

	if err != nil {
		return nil, err
	}

	req.Header.Add("Accept", "application/json")
	q := req.URL.Query()

	var builder strings.Builder

	for _, currency := range currencies {
		builder.WriteString(currency)
		builder.WriteRune(',')
	}

	q.Add("q", strings.TrimRight(builder.String(), ","))
	q.Add("compact", "ultra")
	q.Add("apiKey", f.ApiKey)

	req.URL.RawQuery = q.Encode()

	return req, nil
}

func appendToCurrencies(wg *sync.WaitGroup, c <-chan map[string]float32, currencies *[]currency_fetcher.Currency) {
	defer wg.Done()
	handle := func(data map[string]float32, currencies *[]currency_fetcher.Currency) {
		for key, cur := range data {
			isoCurrencies := strings.Split(key, "_")
			*currencies = append(*currencies, currency_fetcher.Currency{
				From:     isoCurrencies[0],
				To:       isoCurrencies[1],
				Provider: FreeConvProvider,
				Rate:     cur,
			})
		}
	}

	for {
		select {
		case data, more := <-c:
			handle(data, currencies)
			if !more {
				return
			}
		}
	}
}

func (f FreeCurrConvFetcher) fetchCurrencies(client *http.Client, wg *sync.WaitGroup, currencies []string, channel chan<- map[string]float32, errorChannel chan<- error) {
	defer wg.Done()
	url := f.Url

	if url == "" {
		url = FreeConvFetchUrl
	}

	req, err := f.getData(url, currencies)

	if err != nil {
		errorChannel <- err
		return
	}

	res, err := client.Do(req)

	if err != nil {
		errorChannel <- err
		return
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)

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
		errorResponse := struct {
			Status int    `json:"status"`
			Error  string `json:"error"`
		}{}
		_ = json.Unmarshal(body, &errorResponse)

		if strings.Contains(errorResponse.Error, "required") {
			errorChannel <- ErrUnAuthorized
			return
		}

		if strings.Contains(errorResponse.Error, "API limit reached") {
			errorChannel <- ErrApiLimitReached
		}
		return
	}

	if res.StatusCode >= 400 && res.StatusCode < 500 {
		errorChannel <- ErrClient
		return
	}

	if res.StatusCode >= 500 {
		errorChannel <- ErrServer
		return
	}
	errorChannel <- ErrUnknown
}

func (f FreeCurrConvFetcher) Fetch(currenciesToFetch []string) ([]currency_fetcher.Currency, error) {
	numberOfRequests := len(currenciesToFetch) / f.MaxPerRequest
	if numberOfRequests >= f.MaxPerHour {
		return nil, ErrNotEnoughRequests
	}

	channel := make(chan map[string]float32, numberOfRequests)
	errorChannel := make(chan error)
	currencies := make([]currency_fetcher.Currency, 0, len(currenciesToFetch))
	var wg sync.WaitGroup
	var appendWg sync.WaitGroup
	client := &http.Client{}

	appendWg.Add(1)
	go appendToCurrencies(&appendWg, channel, &currencies)

	idx := 0
	for i := 0; i < numberOfRequests; i++ {
		wg.Add(1)
		go f.fetchCurrencies(client, &wg, currenciesToFetch[idx:idx+f.MaxPerRequest], channel, errorChannel)
		idx += f.MaxPerRequest
	}

	go func(wg, appendWg *sync.WaitGroup, channel chan map[string]float32, errorChannel chan<- error) {
		wg.Wait()
		close(channel)
		appendWg.Wait()
		errorChannel <- nil
		close(errorChannel)
	}(&wg, &appendWg, channel, errorChannel)

	select {
	case err := <-errorChannel:
		if err != nil {
			return nil, err
		}
		return currencies, nil
	}

}
