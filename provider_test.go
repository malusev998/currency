package currency_test

import (
	"errors"
	"testing"

	"github.com/malusev998/currency"
	"github.com/stretchr/testify/require"
)

func TestConvertToProvidersFromStringSlice(t *testing.T) {
	assert := require.New(t)

	values := []struct {
		value    []string
		expected interface{}
		err      error
	}{
		{[]string{"freecurrconversion", "exchangeratesapi"}, []currency.Provider{currency.FreeConvProvider, currency.ExchangeRatesAPIProvider}, nil},
		{[]string{"not-valid-value"}, []currency.Provider([]currency.Provider(nil)), errors.New("value not-valid-value is not valid Provider")},
	}
	for _, value := range values {
		providers, err := currency.ConvertToProvidersFromStringSlice(value.value)
		assert.Equal(providers, value.expected)
		assert.Equal(value.err, err)
	}
}

func TestConvertToProviderFromString(t *testing.T) {
	assert := require.New(t)
	values := []struct {
		value    string
		expected interface{}
		err      error
	}{
		{"freecurrconversion", currency.FreeConvProvider, nil},
		{"exchangeratesapi", currency.ExchangeRatesAPIProvider, nil},
		{"", currency.Provider(""), errors.New("value  is not valid Provider")},
		{"not-valid-value", currency.Provider(""), errors.New("value not-valid-value is not valid Provider")},
	}

	for _, value := range values {
		provider, err := currency.ConvertToProviderFromString(value.value)
		assert.Equal(provider, value.expected)
		assert.Equal(value.err, err)
	}
}
