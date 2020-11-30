package main

import (
	"fmt"

	currencyFetcher "github.com/malusev998/currency-fetcher"
	"github.com/malusev998/currency-fetcher/fetchers"
	service "github.com/malusev998/currency-fetcher/services"
	"github.com/malusev998/currency-fetcher/storage"
)

func createStorages(config *Config) ([]currencyFetcher.Storage, error) {
	storages := make([]currencyFetcher.Storage, 0, len(config.Storage))
	for _, s := range config.Storage {
		c, ok := config.StorageConfig[s]
		if !ok {
			return nil, fmt.Errorf("storage %s does not exist", s)
		}

		st, err := storage.NewStorage(s, c)

		if err != nil {
			return nil, err
		}

		storages = append(storages, st)
	}

	return storages, nil
}

func createCurrencyService(config *Config, storages []currencyFetcher.Storage) ([]currencyFetcher.Service, error) {
	services := make([]currencyFetcher.Service, 0, len(config.Fetchers))

	for _, f := range config.Fetchers {
		c, ok := config.FetchersConfig[f]

		if !ok {
			return nil, fmt.Errorf("fetcher %s does not exist", f)
		}

		services = append(services, service.Service{
			Fetcher: fetchers.NewCurrencyFetcher(f, c),
			Storage: storages,
		})
	}

	return services, nil
}
