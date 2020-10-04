package services

import (
	"github.com/BrosSquad/currency-fetcher"
	"sync"
)

type FreeConvService struct {
	Fetcher currency_fetcher.Fetcher
	Storage []currency_fetcher.Storage
}

func (f FreeConvService) Save(currenciesToFetch []string) (map[string][]currency_fetcher.CurrencyWithId, error) {
	var wg sync.WaitGroup
	mutex := &sync.RWMutex{}
	fetchedCurrencies, err := f.Fetcher.Fetch(currenciesToFetch)
	errorChannel := make(chan error)
	defer close(errorChannel)
	if err != nil {
		return nil, err
	}

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

	go func(group *sync.WaitGroup, errorChannel chan<- error) {
		wg.Wait()
		errorChannel <- nil
	}(&wg, errorChannel)

	select {
	case err := <-errorChannel:
		if err == nil {
			return data, nil
		}
		return nil, err
	}

}
