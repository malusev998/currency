package currency_fetcher

import "time"

type Storage interface {
	Store([]Currency) ([]CurrencyWithId, error)
	Get(from, to string, page, perPage int64) ([]CurrencyWithId, error)
	GetByProvider(from, to, provider string, page, perPage int64) ([]CurrencyWithId, error)
	GetByDate(from, to string, start, end time.Time, page, perPage int64) ([]CurrencyWithId, error)
	GetByDateAndProvider(from, to, provider string, start, end time.Time, page, perPage int64) ([]CurrencyWithId, error)
	GetStorageProviderName() string
	Migrate() error
}
