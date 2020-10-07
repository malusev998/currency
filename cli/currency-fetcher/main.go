package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/BrosSquad/currency-fetcher"
	"github.com/BrosSquad/currency-fetcher/currency"
	"github.com/BrosSquad/currency-fetcher/services"
	"github.com/BrosSquad/currency-fetcher/storage"
	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"strconv"
)

type Config struct {
	Storages              []currency_fetcher.Storage
	FreeConvServiceConfig struct {
		ApiKey             string
		MaxPerHourRequests int
		MaxPerRequest      int
	}
	CurrenciesToFetch []string
}

func getConfig(ctx context.Context, viperConfig *viper.Viper, sqlDb **sql.DB, mongoDbClient **mongo.Client) Config {
	var config Config
	var err error
	viperConfig.SetConfigName("config")
	viperConfig.SetConfigType("yaml")
	viperConfig.AddConfigPath(".")

	config.Storages = make([]currency_fetcher.Storage, 0, 2)

	if err := viperConfig.ReadInConfig(); err != nil {
		log.Fatalf("Error while reading in the config file: %v", err)
	}

	migrate := viperConfig.GetBool("migrate")

	for _, st := range viperConfig.GetStringSlice("storage") {
		switch st {
		case "mysql":
			mysqlConfig := viperConfig.GetStringMapString("databases.mysql")
			*sqlDb, err = sql.Open("mysql", fmt.Sprintf("%s:%s@%s:%s/%s", mysqlConfig["user"], mysqlConfig["password"], mysqlConfig["host"], mysqlConfig["port"], mysqlConfig["db"]))
			if err != nil {
				log.Fatalf("Error while connecting to mysql: %v", err)
			}

			mysqlStorage := storage.NewMySQLStorage(ctx, *sqlDb, mysqlConfig["table"], nil)

			if migrate {
				if err := mysqlStorage.Migrate(); err != nil {
					log.Fatalf("Error while migrating mysql database: %v", err)
				}
			}

			config.Storages = append(config.Storages, mysqlStorage)
		case "mongodb":
			mongodbConfig := viperConfig.GetStringMapString("databases.mongodb")
			*mongoDbClient, err = mongo.NewClient(options.Client().ApplyURI(mongodbConfig["uri"]))

			if err != nil {
				log.Fatalf("Error in mongo mongodbConfiguration: %v", err)
			}

			if err := (*mongoDbClient).Connect(ctx); err != nil {
				log.Fatalf("Error while connecting to mongodb: %v", err)
			}
			db := (*mongoDbClient).Database(mongodbConfig["database"])

			mongoStorage := storage.NewMongoStorage(ctx, db.Collection(mongodbConfig["collection"]))

			if migrate {
				if err := db.CreateCollection(ctx, mongodbConfig["collection"]); err != nil {
					log.Fatalf("Error while creating mongodb collection: %v", err)
				}

				if err := mongoStorage.Migrate(); err != nil {
					log.Fatalf("Error while migrating mongodb collection: %v", err)
				}
			}

			config.Storages = append(config.Storages, mongoStorage)
		}
	}

	fetcherConfig := viperConfig.GetStringMapString("fetchers.freecurrconv")
	maxPerHour, err := strconv.ParseUint(fetcherConfig["maxPerHour"], 10, 32)

	if err != nil {
		log.Fatalf("Error while parsing maxPerHour in fetchers.freecurrconv: %v", err)
	}

	config.FreeConvServiceConfig.MaxPerHourRequests = int(maxPerHour)

	maxPerRequest, err := strconv.ParseUint(fetcherConfig["maxPerRequest"], 10, 32)
	if err != nil {
		log.Fatalf("Error while parsing maxPerRequest in fetchers.freecurrconv: %v", err)
	}

	config.FreeConvServiceConfig.MaxPerRequest = int(maxPerRequest)
	config.FreeConvServiceConfig.ApiKey = fetcherConfig["apiKey"]
	config.CurrenciesToFetch = viperConfig.GetStringSlice("currencies")
	return config
}

func main() {
	var mongoDbClient *mongo.Client
	var sqlDb *sql.DB
	var err error
	storages := make([]currency_fetcher.Storage, 0, 2)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	config := getConfig(ctx, viper.New(), &sqlDb, &mongoDbClient)

	service := services.FreeConvService{
		Fetcher: currency.FreeCurrConvFetcher{
			ApiKey:        config.FreeConvServiceConfig.ApiKey,
			MaxPerHour:    config.FreeConvServiceConfig.MaxPerHourRequests,
			MaxPerRequest: config.FreeConvServiceConfig.MaxPerRequest,
		},
		Storage: storages,
	}

	_, err = service.Save(config.CurrenciesToFetch)

	if err != nil {
		log.Fatalf("Error while saving currencies: %v", err)
	}

	if mongoDbClient != nil && mongoDbClient.Disconnect(ctx) == nil {
		log.Println("Disconnecting from mongodb")
	}

	if sqlDb != nil && sqlDb.Close() == nil {
		log.Println("Disconnecting from SQL Database")
	}
}
