package services

import (
	"errors"
	"github.com/BrosSquad/currency-fetcher"
	"github.com/shopspring/decimal"
	"math"
	"time"
)

var ErrCurrencyNotFound = errors.New("rate for the currency is not found in storage")

type ConversionService struct {
	Storage currency_fetcher.Storage
}

func (c ConversionService) Convert(from, to, provider string, value float32, date time.Time) (float32, error) {
	decimalValue := decimal.NewFromFloat32(value)
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	currencies, err := c.Storage.GetByDateAndProvider(from, to, provider, startOfDay, date, 1, 1)

	if err != nil {
		return 0.0, err
	}

	if len(currencies) == 0 {
		return 0, ErrCurrencyNotFound
	}

	rateDecimal := decimal.NewFromFloat32(currencies[0].Rate)

	floatValue, _ := decimalValue.Mul(rateDecimal).Float64()
	return float32(math.Round(floatValue*1_000_000) / 1_000_000), nil
}
