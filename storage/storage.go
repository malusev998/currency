package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"

	currencyFetcher "github.com/malusev998/currency-fetcher"
)

type (
	Provider   string
	BaseConfig struct {
		Cxt     context.Context
		Migrate bool
	}
	MySQLConfig struct {
		BaseConfig
		ConnectionString string
		TableName        string
		IDGenerator      IDGenerator
	}
	MongoDBConfig struct {
		BaseConfig
		ConnectionString string
		Database         string
		Collection       string
	}
)

const (
	MySQL   Provider = "mysql"
	MongoDB Provider = "mongodb"
)

var (
	ErrStorageNotFound = errors.New("storage is not found")
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
	case "mysql":
		return MySQL, nil
	case "mongodb":
		return MongoDB, nil
	}

	return "", fmt.Errorf("value %s is not valid Provider", str)
}

func NewStorage(provider Provider, config interface{}) (currencyFetcher.Storage, error) {
	switch provider {
	case MySQL:
		return NewMySQLStorage(config.(MySQLConfig))
	case MongoDB:
		c := config.(MongoDBConfig)
		return NewMongoStorage(c)
	}

	return nil, ErrStorageNotFound
}
