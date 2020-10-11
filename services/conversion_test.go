package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	currency_fetcher "github.com/malusev998/currency-fetcher"
)

type mockStorage struct {
	mock.Mock
}

type mockTimeoutStorage struct {
	mock.Mock
}

func (m *mockTimeoutStorage) Store(currencies []currency_fetcher.Currency) ([]currency_fetcher.CurrencyWithId, error) {
	return nil, nil
}

func (m *mockTimeoutStorage) Get(from, to string, page, perPage int64) ([]currency_fetcher.CurrencyWithId, error) {
	panic("implement me")
}

func (m *mockTimeoutStorage) GetByProvider(from, to, provider string, page, perPage int64) ([]currency_fetcher.CurrencyWithId, error) {
	panic("implement me")
}

func (m *mockTimeoutStorage) GetByDate(from, to string, start, end time.Time, page, perPage int64) ([]currency_fetcher.CurrencyWithId, error) {
	panic("implement me")
}

func (m *mockTimeoutStorage) GetByDateAndProvider(from, to, provider string, start, end time.Time, page, perPage int64) ([]currency_fetcher.CurrencyWithId, error) {
	time.Sleep(time.Duration(10) * time.Second)
	return nil, nil
}

func (m *mockTimeoutStorage) GetStorageProviderName() string {
	return "mockTimeOutStorage"
}

func (m *mockTimeoutStorage) Migrate() error {
	panic("implement me")
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
	t.Parallel()
	asserts := require.New(t)
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	t.Run("SuccessfulConversion_ONE_STORAGE_PROVIDER", func(t *testing.T) {
		storage := &mockStorage{}
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

		service := ConversionService{Ctx: context.Background(), Storages: []currency_fetcher.Storage{storage}}
		value, err := service.Convert("EUR", "USD", "TestProvider", 1.531454, now)
		asserts.Nil(err)
		asserts.Equal(float32(1.924183), value)
	})

	t.Run("NoStorageProvider", func(t *testing.T) {
		service := ConversionService{Ctx: context.Background()}
		value, err := service.Convert("EUR", "USD", "TestProvider", 1.531454, now)

		asserts.NotNil(err)
		asserts.True(errors.Is(err, ErrNoStorageProvided))
		asserts.Equal(float32(0.0), value)
	})

	t.Run("StorageTimeOut", func(t *testing.T) {
		storage1 := &mockTimeoutStorage{}
		storage2 := &mockTimeoutStorage{}
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(5)*time.Second)
		go func() {
			select {
			case <-time.After(time.Duration(5) * time.Millisecond):
				cancel()
			}
		}()
		service := ConversionService{Ctx: ctx, Storages: []currency_fetcher.Storage{storage1, storage2}}
		value, err := service.Convert("EUR", "USD", "TestProvider", 1.531454, now)
		asserts.NotNil(err)
		asserts.True(errors.Is(err, ErrTimeRanOut))
		asserts.Equal(float32(0.0), value)
	})
}
