package services

import (
	"sync"

	currencyFetcher "github.com/malusev998/currency"
)

type Service struct {
	Fetcher currencyFetcher.Fetcher
	Storage []currencyFetcher.Storage
}

func saveToStorage(
	wg *sync.WaitGroup,
	currencies []currencyFetcher.Currency,
	data *map[string][]currencyFetcher.CurrencyWithID,
	storage currencyFetcher.Storage,
	errorChannel chan<- error,
	mutex sync.Locker,
) {
	defer wg.Done()
	c, err := storage.Store(currencies)

	if err != nil {
		errorChannel <- err
		return
	}

	mutex.Lock()
	(*data)[storage.GetStorageProviderName()] = c
	mutex.Unlock()
}

func (f Service) Save(currenciesToFetch []string) (map[string][]currencyFetcher.CurrencyWithID, error) {
	var wg sync.WaitGroup
	mutex := &sync.RWMutex{}

	fetchedCurrencies, err := f.Fetcher.Fetch(currenciesToFetch)
	if err != nil {
		return nil, err
	}

	errorChannel := make(chan error)
	data := make(map[string][]currencyFetcher.CurrencyWithID)

	wg.Add(len(f.Storage))
	for _, storage := range f.Storage {
		go saveToStorage(&wg, fetchedCurrencies, &data, storage, errorChannel, mutex)
	}

	go func(wg *sync.WaitGroup, errorChannel chan error) {
		wg.Wait()
		close(errorChannel)
	}(&wg, errorChannel)

	if err, more := <-errorChannel; more {
		return nil, err
	}

	return data, nil
}
