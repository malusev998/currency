package currency_fetcher

import (
	"io"
	"time"
)

type Storage interface {
	io.Closer
	Store([]Currency) ([]CurrencyWithID, error)
	Get(from, to string, page, perPage int64) ([]CurrencyWithID, error)
	GetByProvider(from, to string, provider Provider, page, perPage int64) ([]CurrencyWithID, error)
	GetByDate(from, to string, start, end time.Time, page, perPage int64) ([]CurrencyWithID, error)
	GetByDateAndProvider(from, to string, provider Provider, start, end time.Time, page, perPage int64) ([]CurrencyWithID, error)
	GetStorageProviderName() string
	Migrate() error
	Drop() error
}
