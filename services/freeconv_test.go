package services

import (
	"errors"
	"math/rand"
	"strings"
	"testing"
	"time"

	currency_fetcher "github.com/malusev998/currency-fetcher"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type (
	MockFetcher struct {
		mock.Mock
	}

	MockStorage struct {
		mock.Mock
	}
)

func (m *MockStorage) Store(currencies []currency_fetcher.Currency) ([]currency_fetcher.CurrencyWithId, error) {
	args := m.Called(currencies)

	return1 := args.Get(0)

	if return1 == nil {
		return nil, args.Error(1)
	}
	return return1.([]currency_fetcher.CurrencyWithId), args.Error(1)
}

func (m *MockStorage) Get(from, to string, page, perPage int64) ([]currency_fetcher.CurrencyWithId, error) {
	args := m.Called(from, to, page, perPage)

	return args.Get(0).([]currency_fetcher.CurrencyWithId), args.Error(1)
}

func (m *MockStorage) GetByProvider(from, to, provider string, page, perPage int64) ([]currency_fetcher.CurrencyWithId, error) {
	args := m.Called(from, to, provider, page, perPage)

	return args.Get(0).([]currency_fetcher.CurrencyWithId), args.Error(1)
}

func (m *MockStorage) GetByDate(from, to string, start, end time.Time, page, perPage int64) ([]currency_fetcher.CurrencyWithId, error) {
	args := m.Called(from, to, start, end, page, perPage)

	return args.Get(0).([]currency_fetcher.CurrencyWithId), args.Error(1)
}

func (m *MockStorage) GetByDateAndProvider(from, to, provider string, start, end time.Time, page, perPage int64) ([]currency_fetcher.CurrencyWithId, error) {
	args := m.Called(from, to, provider, start, end, page, perPage)

	return args.Get(0).([]currency_fetcher.CurrencyWithId), args.Error(1)
}

func (m *MockStorage) GetStorageProviderName() string {
	return "MockStorage"
}

func (m *MockStorage) Migrate() error {
	return nil
}

func (m *MockFetcher) Fetch(currenciesToFetch []string) ([]currency_fetcher.Currency, error) {
	args := m.Called(currenciesToFetch)
	return1 := args.Get(0)

	if return1 == nil {
		return nil, args.Error(1)
	}

	return return1.([]currency_fetcher.Currency), args.Error(1)
}

func TestFreeConvService(t *testing.T) {
	t.Parallel()
	asserts := require.New(t)
	currenciesToFetch := []string{"EUR_USD", "USD_EUR"}
	currenciesWithId := make([]currency_fetcher.CurrencyWithId, 0, len(currenciesToFetch))
	currenciesFetched := make([]currency_fetcher.Currency, 0, len(currenciesToFetch))
	for i, c := range currenciesToFetch {
		isoCurrencies := strings.Split(c, "_")
		rate := rand.Float32()
		currenciesWithId = append(currenciesWithId, currency_fetcher.CurrencyWithId{
			Currency: currency_fetcher.Currency{
				From:      isoCurrencies[0],
				To:        isoCurrencies[1],
				Provider:  "MockProvider",
				Rate:      rate,
				CreatedAt: time.Now(),
			},
			Id: uint64(i),
		})
		currenciesFetched = append(currenciesFetched, currency_fetcher.Currency{
			From:     isoCurrencies[0],
			To:       isoCurrencies[1],
			Provider: "MockProvider",
			Rate:     rate,
		})
	}

	t.Run("SaveCorrectly", func(t *testing.T) {
		fetcher := &MockFetcher{}
		storage := &MockStorage{}
		service := FreeConvService{
			Fetcher: fetcher,
			Storage: []currency_fetcher.Storage{storage},
		}

		fetcher.On("Fetch", currenciesToFetch).Return(currenciesFetched, nil)
		storage.On("Store", currenciesFetched).Return(currenciesWithId, nil)

		savedCurrencies, err := service.Save(currenciesToFetch)

		asserts.Nil(err)
		asserts.NotNil(savedCurrencies)
		asserts.Contains(savedCurrencies, "MockStorage")

		for _, c := range savedCurrencies["MockStorage"] {
			_, ok := c.Id.(uint64)
			asserts.True(ok)
		}
	})

	t.Run("FetchReturnsError", func(t *testing.T) {
		fetcher := &MockFetcher{}
		storage := &MockStorage{}
		service := FreeConvService{
			Fetcher: fetcher,
			Storage: []currency_fetcher.Storage{storage},
		}

		fetcher.On("Fetch", currenciesToFetch).Return(nil, errors.New("an error has occurred"))
		savedCurrencies, err := service.Save(currenciesToFetch)
		asserts.Nil(savedCurrencies)
		asserts.NotNil(err)
	})

	t.Run("StorageReturnsError", func(t *testing.T) {
		fetcher := &MockFetcher{}
		storage := &MockStorage{}
		service := FreeConvService{
			Fetcher: fetcher,
			Storage: []currency_fetcher.Storage{storage},
		}
		fetcher.On("Fetch", currenciesToFetch).Return(currenciesFetched, nil)
		storage.On("Store", currenciesFetched).Return(nil, errors.New("error while inserting into storage"))

		savedCurrencies, err := service.Save(currenciesToFetch)
		asserts.Nil(savedCurrencies)
		asserts.NotNil(err)
	})
}
