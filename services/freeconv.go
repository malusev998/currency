package services

import (
	"sync"

	currency_fetcher "github.com/BrosSquad/currency-fetcher"
)

type FreeConvService struct {
	Fetcher currency_fetcher.Fetcher
	Storage []currency_fetcher.Storage
}

func (f FreeConvService) Save(currenciesToFetch []string) (map[string][]currency_fetcher.CurrencyWithId, error) {
	var wg sync.WaitGroup
	mutex := &sync.RWMutex{}
	fetchedCurrencies, err := f.Fetcher.Fetch(currenciesToFetch)
	if err != nil {
		return nil, err
	}
	errorChannel := make(chan error)
	data := make(map[string][]currency_fetcher.CurrencyWithId)
	wg.Add(len(f.Storage))
	for _, storage := range f.Storage {
		go func(wg *sync.WaitGroup, data *map[string][]currency_fetcher.CurrencyWithId, storage currency_fetcher.Storage, errorChannel chan<- error, mutex *sync.RWMutex) {
			defer wg.Done()
			currencies, err := storage.Store(fetchedCurrencies)
			if err != nil {
				errorChannel <- err
				return
			}
			mutex.Lock()
			(*data)[storage.GetStorageProviderName()] = currencies
			mutex.Unlock()
		}(&wg, &data, storage, errorChannel, mutex)
	}

	go func(group *sync.WaitGroup, errorChannel chan error) {
		wg.Wait()
		close(errorChannel)
	}(&wg, errorChannel)

	select {
	case err, more := <-errorChannel:
		if !more {
			return data, nil
		}
		return nil, err
	}

}
