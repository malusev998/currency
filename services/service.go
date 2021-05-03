package services

import (
	"sync"

	currencyFetcher "github.com/malusev998/currency"
)

type Service struct {
	Fetcher currencyFetcher.Fetcher
	Storage []currencyFetcher.Storage
}


type currencyCh struct {
	StorageName string
	Currency []currencyFetcher.CurrencyWithID
}

func saveToStorage(
	wg *sync.WaitGroup,
	currencies []currencyFetcher.Currency,
	storage currencyFetcher.Storage,
	cs chan<- currencyCh,
	errorChannel chan<- error,
) {
	defer wg.Done()
	c, err := storage.Store(currencies)

	if err != nil {
		errorChannel <- err
		return
	}

	cs <- currencyCh{
		StorageName: storage.GetStorageProviderName(),
		Currency: c,
	}
}

func (f Service) Save(currenciesToFetch []string) (map[string][]currencyFetcher.CurrencyWithID, error) {
	var wg sync.WaitGroup
	fetchedCurrencies, err := f.Fetcher.Fetch(currenciesToFetch)
	if err != nil {
		return nil, err
	}

	cs := make(chan currencyCh, len(currenciesToFetch))
	errorChannel := make(chan error)
	data := make(map[string][]currencyFetcher.CurrencyWithID)

	wg.Add(len(f.Storage))

	for _, storage := range f.Storage {
		go saveToStorage(&wg, fetchedCurrencies, storage, cs, errorChannel)
	}

	go func(wg *sync.WaitGroup, cs chan currencyCh, errorChannel chan error) {
		wg.Wait()
		close(errorChannel)
		close(cs)
	}(&wg, cs, errorChannel)


	for item := range cs {
		data[item.StorageName] = item.Currency
	}

	if err, more := <-errorChannel; more {
		return nil, err
	}

	return data, nil
}
