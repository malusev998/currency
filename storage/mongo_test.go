package storage_test

import (
	"context"
	"github.com/malusev998/currency/storage"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	currencyFetcher "github.com/malusev998/currency"
)

func TestStoreInMongo(t *testing.T) {
	t.Parallel()
	runningInDocker := false

	if os.Getenv("RUNNING_IN_DOCKER") != "" {
		runningInDocker = true
	}

	ctx := context.Background()
	asserts := require.New(t)

	uri := "mongodb://localhost:27017"

	if runningInDocker {
		uri = "mongodb://mongo:27017"
	}

	client, err := mongo.NewClient(options.Client().ApplyURI(uri))

	asserts.Nil(err)
	asserts.NotNil(client)

	err = client.Connect(ctx)

	asserts.Nil(err)

	defer client.Disconnect(ctx)

	t.Run("StoreOne", func(t *testing.T) {
		database := client.Database("currency_fetcher_store_one")
		defer database.Drop(ctx)

		storage, _ := storage.NewMongoStorage(storage.MongoDBConfig{
			BaseConfig: storage.BaseConfig{
				Cxt:     ctx,
				Migrate: true,
			},
			ConnectionString: uri,
			Database:         "currency_fetcher_store_one",
			Collection:       "fetchers",
		})
		defer storage.Drop()

		provider := currencyFetcher.Provider("TestProvider")

		currencies, err := storage.Store([]currencyFetcher.Currency{
			{
				From:     "EUR",
				To:       "USD",
				Provider: provider,
				Rate:     0.8,
			},
		})

		asserts.Nil(err)
		asserts.Len(currencies, 1)
		asserts.NotNil(currencies[0].ID)
		asserts.Equal("EUR", currencies[0].From)
		asserts.Equal("USD", currencies[0].To)
		asserts.Equal(provider, currencies[0].Provider)
		asserts.Equal(float32(0.8), currencies[0].Rate)
	})

	t.Run("StoreMany", func(t *testing.T) {
		database := client.Database("currency_fetcher_store_many")
		defer database.Drop(ctx)

		storage, _ := storage.NewMongoStorage(storage.MongoDBConfig{
			BaseConfig: storage.BaseConfig{
				Cxt:     ctx,
				Migrate: true,
			},
			ConnectionString: uri,
			Database:         "currency_fetcher_store_many",
			Collection:       "fetchers",
		})
		defer storage.Drop()

		currencies, err := storage.Store([]currencyFetcher.Currency{
			{
				From:      "EUR",
				To:        "USD",
				Provider:  "TestProvider",
				Rate:      0.8,
				CreatedAt: time.Now().Add(time.Duration(10) * time.Minute),
			},
			{
				From:     "EUR",
				To:       "USD",
				Provider: "TestProvider",
				Rate:     0.8,
			},
		})

		asserts.Nil(err)
		asserts.Len(currencies, 2)
		for _, cur := range currencies {
			asserts.NotNil(cur.ID)
			asserts.Equal("EUR", cur.From)
			asserts.Equal("USD", cur.To)
			asserts.Equal(currencyFetcher.Provider("TestProvider"), cur.Provider)
			asserts.Equal(float32(0.8), cur.Rate)
		}
	})
}

func TestGetCurrenciesFromMongoDb(t *testing.T) {
	t.Parallel()
	runningInDocker := false
	provider := currencyFetcher.Provider("TestProvider")
	otherProvider := currencyFetcher.Provider("OtherProvider")

	if os.Getenv("RUNNING_IN_DOCKER") != "" {
		runningInDocker = true
	}

	ctx := context.Background()
	asserts := require.New(t)

	uri := "mongodb://localhost:27017"

	if runningInDocker {
		uri = "mongodb://mongo:27017"
	}
	client, err := mongo.NewClient(options.Client().ApplyURI(uri))

	asserts.Nil(err)
	asserts.NotNil(client)

	err = client.Connect(ctx)

	asserts.Nil(err)

	defer client.Disconnect(ctx)

	database := client.Database("currency_fetcher_fetch")
	defer database.Drop(ctx)
	collection := database.Collection("fetchers")

	var currenciesToInsert []interface{}

	for i := 0; i < 10; i++ {
		currenciesToInsert = append(currenciesToInsert, map[string]interface{}{
			"fetchers":  "EUR_USD",
			"provider":  provider,
			"rate":      rand.Float32(),
			"createdAt": time.Now(),
		})
	}
	results, err := collection.InsertMany(ctx, currenciesToInsert)
	asserts.NotNil(results)
	asserts.Nil(err)

	defer collection.Drop(ctx)

	t.Run("GetAllCurrenciesPaginated", func(t *testing.T) {
		storage, _ := storage.NewMongoStorage(storage.MongoDBConfig{
			BaseConfig: storage.BaseConfig{
				Cxt:     ctx,
				Migrate: false,
			},
			ConnectionString: uri,
			Database:         "currency_fetcher_fetch",
			Collection:       "fetchers",
		})
		currencies, err := storage.Get("EUR", "USD", 1, 10)
		asserts.Nil(err)
		asserts.NotNil(currencies)
		asserts.Len(currencies, 10)

		for _, cur := range currencies {
			asserts.NotNil(cur.ID)
			asserts.IsType(primitive.ObjectID{}, cur.ID)
			asserts.Equal(provider, cur.Provider)
			asserts.Equal("EUR", cur.From)
			asserts.Equal("USD", cur.To)
		}
	})

	t.Run("GetWithProvider", func(t *testing.T) {
		st, _ := storage.NewMongoStorage(storage.MongoDBConfig{
			BaseConfig: storage.BaseConfig{
				Cxt:     ctx,
				Migrate: false,
			},
			ConnectionString: uri,
			Database:         "currency_fetcher_fetch",
			Collection:       "fetchers",
		})
		currencies, err := st.GetByProvider("EUR", "USD", provider, 1, 10)
		asserts.Nil(err)
		asserts.NotNil(currencies)
		asserts.Len(currencies, 10)

		for _, cur := range currencies {
			asserts.Equal(provider, cur.Provider)
			asserts.Equal("EUR", cur.From)
			asserts.Equal("USD", cur.To)
		}
	})

	t.Run("GetWithNonExistentProvider", func(t *testing.T) {
		st, _ := storage.NewMongoStorage(storage.MongoDBConfig{
			BaseConfig: storage.BaseConfig{
				Cxt:     ctx,
				Migrate: false,
			},
			ConnectionString: uri,
			Database:         "currency_fetcher_fetch",
			Collection:       "fetchers",
		})

		currencies, err := st.GetByProvider("EUR", "USD", "NonExistentProvider", 1, 10)
		asserts.Nil(err)
		asserts.NotNil(currencies)
		asserts.Len(currencies, 0)
	})

	t.Run("GetWithDate", func(t *testing.T) {
		now := time.Now()
		var toInsert []interface{}
		for i := 0; i < 10; i++ {
			duration := time.Minute * time.Duration(i)
			toInsert = append(toInsert, map[string]interface{}{
				"fetchers":  "EUR_USD",
				"provider":  otherProvider,
				"rate":      rand.Float32(),
				"createdAt": now.Add(duration),
			})
		}

		results, err := collection.InsertMany(ctx, toInsert)
		asserts.NotNil(results)
		asserts.Nil(err)

		inFuture := now.Add(time.Duration(10) * time.Minute)
		st, _ := storage.NewMongoStorage(storage.MongoDBConfig{
			BaseConfig: storage.BaseConfig{
				Cxt:     ctx,
				Migrate: false,
			},
			ConnectionString: uri,
			Database:         "currency_fetcher_fetch",
			Collection:       "fetchers",
		})
		currencies, err := st.GetByDate("EUR", "USD", inFuture.Add(time.Duration(-5)*time.Minute), inFuture, 1, 10)
		asserts.Nil(err)
		asserts.NotNil(currencies)
		asserts.Len(currencies, 5)
	})
}
