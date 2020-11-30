package main

import (
	"context"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"github.com/spf13/viper"
	"strconv"

	currencyFetcher "github.com/malusev998/currency-fetcher"
	"github.com/malusev998/currency-fetcher/currency"
	"github.com/malusev998/currency-fetcher/storage"
)

type (
	FetchersConfig map[currencyFetcher.Provider]interface{}
	StorageConfig  map[storage.Provider]interface{}
	Config         struct {
		Fetchers          []currencyFetcher.Provider
		Storage           []storage.Provider
		FetchersConfig    FetchersConfig
		StorageConfig     StorageConfig
		CurrenciesToFetch []string
	}
)

func getMysqlDSN(config map[string]string) string {
	mysqlDriverConfig := mysql.NewConfig()
	mysqlDriverConfig.User = config["user"]
	mysqlDriverConfig.Passwd = config["password"]
	mysqlDriverConfig.Addr = config["addr"]
	mysqlDriverConfig.Net = "tcp"
	mysqlDriverConfig.DBName = config["db"]

	return mysqlDriverConfig.FormatDSN()
}

func getConfig(ctx context.Context) (*Config, error) {
	mysqlConfig := viper.GetStringMapString("databases.mysql")
	mongodbConfig := viper.GetStringMapString("databases.mongo")

	fetcherConfig := viper.GetStringMapString("fetchers.freecurrconv")
	maxPerHour, err := strconv.ParseUint(fetcherConfig["maxperhour"], 10, 32)

	if err != nil {
		return nil, fmt.Errorf("error while parsing maxPerHour in fetchers.freecurrconv: %v", err)
	}

	maxPerRequest, err := strconv.ParseUint(fetcherConfig["maxperrequest"], 10, 32)
	if err != nil {
		return nil, fmt.Errorf("error while parsing maxPerRequest in fetchers.freecurrconv: %v", err)
	}

	fetchers, err := currencyFetcher.ConvertToProvidersFromStringSlice(viper.GetStringSlice("fetchers.fetch"))

	if err != nil {
		return nil, err
	}

	storages, err := storage.ConvertToProvidersFromStringSlice(viper.GetStringSlice("storage"))

	storageBaseConfig := storage.BaseConfig{
		Cxt:     ctx,
		Migrate: viper.GetBool("migrate"),
	}

	return &Config{
		Fetchers: fetchers,
		Storage:  storages,
		StorageConfig: StorageConfig{
			storage.MySQL: storage.MySQLConfig{
				BaseConfig:       storageBaseConfig,
				ConnectionString: getMysqlDSN(mysqlConfig),
				TableName:        mysqlConfig["table"],
				IDGenerator:      nil,
			},
			storage.MongoDB: storage.MongoDBConfig{
				BaseConfig:       storageBaseConfig,
				ConnectionString: mongodbConfig["uri"],
				Database:         mongodbConfig["db"],
				Collection:       mongodbConfig["collection"],
			},
		},
		FetchersConfig: map[currencyFetcher.Provider]interface{}{
			currencyFetcher.ExchangeRatesAPIProvider: currency.ExchangeRatesAPIConfig{
				BaseConfig: currency.BaseConfig{
					Ctx: ctx,
					URL: viper.GetString("fetchers.exchangeratesapi"),
				},
			},
			currencyFetcher.FreeConvProvider: currency.FreeConvServiceConfig{
				BaseConfig: currency.BaseConfig{
					Ctx: ctx,
					URL: fetcherConfig["url"],
				},
				APIKey:             fetcherConfig["apikey"],
				MaxPerHourRequests: int(maxPerHour),
				MaxPerRequest:      int(maxPerRequest),
			},
		},
		CurrenciesToFetch: viper.GetStringSlice("currencies"),
	}, nil
}

