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

func main() {
	var mongoDbClient *mongo.Client
	var sqlDb *sql.DB
	var err error
	storages := make([]currency_fetcher.Storage, 0, 2)
	ctx := context.Background()

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error while reading in the config file: %v", err)
	}

	migrate := viper.GetBool("migrate")

	for _, st := range viper.GetStringSlice("storage") {
		switch st {
		case "mysql":
			config := viper.GetStringMapString("databases.mysql")
			sqlDb, err = sql.Open("mysql", fmt.Sprintf("%s:%s@%s:%s/%s", config["user"], config["password"], config["host"], config["port"], config["db"]))
			if err != nil {
				log.Fatalf("Error while connecting to mysql: %v", err)
			}

			mysqlStorage := storage.NewMySQLStorage(ctx, sqlDb, config["table"], nil)

			if migrate {
				if err := mysqlStorage.Migrate(); err != nil {
					log.Fatalf("Error while migrating mysql database: %v", err)
				}
			}

			storages = append(storages, mysqlStorage)
		case "mongodb":
			config := viper.GetStringMapString("databases.mongodb")
			mongoDbClient, err = mongo.NewClient(options.Client().ApplyURI(config["uri"]))

			if err != nil {
				log.Fatalf("Error in mongo configuration: %v", err)
			}

			if err := mongoDbClient.Connect(ctx); err != nil {
				log.Fatalf("Error while connecting to mongodb: %v", err)
			}
			db := mongoDbClient.Database(config["database"])

			mongoStorage := storage.NewMongoStorage(ctx, db.Collection(config["collection"]))

			if migrate {
				if err := db.CreateCollection(ctx, config["collection"]); err != nil {
					log.Fatalf("Error while creating mongodb collection: %v", err)
				}

				if err := mongoStorage.Migrate(); err != nil {
					log.Fatalf("Error while migrating mongodb collection: %v", err)
				}
			}

			storages = append(storages, mongoStorage)
		}
	}

	fetcherConfig := viper.GetStringMapString("fetchers.freecurrconv")
	maxPerHour, err := strconv.ParseUint(fetcherConfig["maxPerHour"], 10, 32)

	if err != nil {
		log.Fatalf("Error while parsing maxPerHour in fetchers.freecurrconv: %v", err)
	}

	maxPerRequest, err := strconv.ParseUint(fetcherConfig["maxPerRequest"], 10, 32)
	if err != nil {
		log.Fatalf("Error while parsing maxPerRequest in fetchers.freecurrconv: %v", err)
	}

	service := services.FreeConvService{
		Fetcher: currency.FreeCurrConvFetcher{
			ApiKey:        fetcherConfig["apiKey"],
			MaxPerHour:    int(maxPerHour),
			MaxPerRequest: int(maxPerRequest),
		},
		Storage: storages,
	}

	_, err = service.Save(viper.GetStringSlice("currencies"))

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
