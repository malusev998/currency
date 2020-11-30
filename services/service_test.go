package services

import (
	"errors"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	currencyFetcher "github.com/malusev998/currency"
)

type (
	MockFetcher struct {
		mock.Mock
	}

	MockStorage struct {
		mock.Mock
	}
)

func (m *MockStorage) Store(currencies []currencyFetcher.Currency) ([]currencyFetcher.CurrencyWithID, error) {
	args := m.Called(currencies)

	return1 := args.Get(0)

	if return1 == nil {
		return nil, args.Error(1)
	}
	return return1.([]currencyFetcher.CurrencyWithID), args.Error(1)
}

func (m *MockStorage) Get(from, to string, page, perPage int64) ([]currencyFetcher.CurrencyWithID, error) {
	args := m.Called(from, to, page, perPage)

	return args.Get(0).([]currencyFetcher.CurrencyWithID), args.Error(1)
}

func (m *MockStorage) GetByProvider(from, to string, provider currencyFetcher.Provider, page, perPage int64) ([]currencyFetcher.CurrencyWithID, error) {
	args := m.Called(from, to, provider, page, perPage)

	return args.Get(0).([]currencyFetcher.CurrencyWithID), args.Error(1)
}

func (m *MockStorage) GetByDate(from, to string, start, end time.Time, page, perPage int64) ([]currencyFetcher.CurrencyWithID, error) {
	args := m.Called(from, to, start, end, page, perPage)

	return args.Get(0).([]currencyFetcher.CurrencyWithID), args.Error(1)
}

func (m *MockStorage) GetByDateAndProvider(from, to string, provider currencyFetcher.Provider, start, end time.Time, page, perPage int64) ([]currencyFetcher.CurrencyWithID, error) {
	args := m.Called(from, to, provider, start, end, page, perPage)

	return args.Get(0).([]currencyFetcher.CurrencyWithID), args.Error(1)
}

func (m *MockStorage) GetStorageProviderName() string {
	return "MockStorage"
}

func (m *MockStorage) Migrate() error {
	return nil
}

func (m *MockStorage) Close() error {
	return nil
}

func (m *MockStorage) Drop() error {
	return nil
}

func (m *MockFetcher) Fetch(currenciesToFetch []string) ([]currencyFetcher.Currency, error) {
	args := m.Called(currenciesToFetch)
	return1 := args.Get(0)

	if return1 == nil {
		return nil, args.Error(1)
	}

	return return1.([]currencyFetcher.Currency), args.Error(1)
}

func TestFreeConvService(t *testing.T) {
	t.Parallel()
	asserts := require.New(t)
	currenciesToFetch := []string{"EUR_USD", "USD_EUR"}
	currenciesWithId := make([]currencyFetcher.CurrencyWithID, 0, len(currenciesToFetch))
	currenciesFetched := make([]currencyFetcher.Currency, 0, len(currenciesToFetch))
	for i, c := range currenciesToFetch {
		isoCurrencies := strings.Split(c, "_")
		rate := rand.Float32()
		currenciesWithId = append(currenciesWithId, currencyFetcher.CurrencyWithID{
			Currency: currencyFetcher.Currency{
				From:      isoCurrencies[0],
				To:        isoCurrencies[1],
				Provider:  "MockProvider",
				Rate:      rate,
				CreatedAt: time.Now(),
			},
			ID: uint64(i),
		})
		currenciesFetched = append(currenciesFetched, currencyFetcher.Currency{
			From:     isoCurrencies[0],
			To:       isoCurrencies[1],
			Provider: "MockProvider",
			Rate:     rate,
		})
	}

	t.Run("SaveCorrectly", func(t *testing.T) {
		fetcher := &MockFetcher{}
		storage := &MockStorage{}
		service := Service{
			Fetcher: fetcher,
			Storage: []currencyFetcher.Storage{storage},
		}

		fetcher.On("Fetch", currenciesToFetch).Return(currenciesFetched, nil)
		storage.On("Store", currenciesFetched).Return(currenciesWithId, nil)

		savedCurrencies, err := service.Save(currenciesToFetch)

		asserts.Nil(err)
		asserts.NotNil(savedCurrencies)
		asserts.Contains(savedCurrencies, "MockStorage")

		for _, c := range savedCurrencies["MockStorage"] {
			_, ok := c.ID.(uint64)
			asserts.True(ok)
		}
	})

	t.Run("FetchReturnsError", func(t *testing.T) {
		fetcher := &MockFetcher{}
		storage := &MockStorage{}
		service := Service{
			Fetcher: fetcher,
			Storage: []currencyFetcher.Storage{storage},
		}

		fetcher.On("Fetch", currenciesToFetch).Return(nil, errors.New("an error has occurred"))
		savedCurrencies, err := service.Save(currenciesToFetch)
		asserts.Nil(savedCurrencies)
		asserts.NotNil(err)
	})

	t.Run("StorageReturnsError", func(t *testing.T) {
		fetcher := &MockFetcher{}
		storage := &MockStorage{}
		service := Service{
			Fetcher: fetcher,
			Storage: []currencyFetcher.Storage{storage},
		}
		fetcher.On("Fetch", currenciesToFetch).Return(currenciesFetched, nil)
		storage.On("Store", currenciesFetched).Return(nil, errors.New("error while inserting into storage"))

		savedCurrencies, err := service.Save(currenciesToFetch)
		asserts.Nil(savedCurrencies)
		asserts.NotNil(err)
	})
}
