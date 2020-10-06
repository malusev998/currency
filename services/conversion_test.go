package services

import (
	"github.com/BrosSquad/currency-fetcher"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type mockStorage struct {
	mock.Mock
}

func (m *mockStorage) Store(currencies []currency_fetcher.Currency) ([]currency_fetcher.CurrencyWithId, error) {
	panic("implement me")
}

func (m *mockStorage) Get(from, to string, page, perPage int64) ([]currency_fetcher.CurrencyWithId, error) {
	panic("implement me")
}

func (m *mockStorage) GetByProvider(from, to, provider string, page, perPage int64) ([]currency_fetcher.CurrencyWithId, error) {
	panic("implement me")
}

func (m *mockStorage) GetByDate(from, to string, start, end time.Time, page, perPage int64) ([]currency_fetcher.CurrencyWithId, error) {
	panic("implement me")
}

func (m *mockStorage) GetByDateAndProvider(from, to, provider string, start, end time.Time, page, perPage int64) ([]currency_fetcher.CurrencyWithId, error) {
	args := m.Called(from, to, provider, start, end, page, perPage)
	return args.Get(0).([]currency_fetcher.CurrencyWithId), args.Error(1)
}

func (m *mockStorage) GetStorageProviderName() string {
	panic("implement me")
}

func (m *mockStorage) Migrate() error {
	panic("implement me")
}

func TestConversionService_Convert(t *testing.T) {
	asserts := require.New(t)
	storage := &mockStorage{}
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	storage.On("GetByDateAndProvider", "EUR", "USD", "TestProvider", startOfDay, now, int64(1), int64(1)).
		Return([]currency_fetcher.CurrencyWithId{
			{
				Currency: currency_fetcher.Currency{
					From:      "EUR",
					To:        "USD",
					Provider:  "TestProvider",
					Rate:      1.2564421,
					CreatedAt: time.Time{},
				},
				Id: 1,
			},
		}, nil)

	service := ConversionService{Storage: storage}

	value, err := service.Convert("EUR", "USD", "TestProvider", 1.531454, now)
	asserts.Nil(err)
	asserts.Equal(float32(1.924183), value)
}
