package currency_fetcher

import (
	"fmt"
	"strings"
)

type Provider string

const (
	FreeConvProvider         Provider = "FreeCurrConversion"
	ExchangeRatesAPIProvider Provider = "ExchangeRatesAPI"
	EmptyProvider            Provider = ""
)

func ConvertToProvidersFromStringSlice(strings []string) ([]Provider, error) {
	providers := make([]Provider, 0, len(strings))

	for _, str := range strings {
		provider, err := ConvertToProviderFromString(str)
		if err != nil {
			return nil, err
		}

		providers = append(providers, provider)
	}

	return providers, nil
}

func ConvertToProviderFromString(str string) (Provider, error) {
	switch strings.ToLower(str) {
	case "freecurrconversion":
		return FreeConvProvider, nil
	case "exchangeratesapi":
		return ExchangeRatesAPIProvider, nil
	}

	return "", fmt.Errorf("value %s is not valid Provider", str)
}

func (p *Provider) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string
	if err := unmarshal(&str); err != nil {
		return err
	}

	provider, err := ConvertToProviderFromString(str)

	if err != nil {
		return err
	}

	*p = provider

	return nil
}

func (p Provider) MarshalYAML() (interface{}, error) {
	return string(p), nil
}
