package storage

import (
	"context"
	"github.com/BrosSquad/currency-fetcher"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"math/rand"
	"testing"
	"time"
)

func TestStoreInMongo(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	asserts := require.New(t)
	client, err := mongo.NewClient(options.Client().ApplyURI("mongodb://localhost:27017"))

	asserts.Nil(err)
	asserts.NotNil(client)

	err = client.Connect(ctx)

	asserts.Nil(err)

	defer client.Disconnect(ctx)

	database := client.Database("currency_fetcher_store")
	defer database.Drop(ctx)
	collection := database.Collection("currency")
	storage := NewMongoStorage(ctx, collection)

	currency, err := storage.Store(currency_fetcher.Currency{
		From:     "EUR",
		To:       "USD",
		Provider: "TestProvider",
		Rate:     0.8,
	})

	asserts.Nil(err)
	asserts.NotNil(currency.Id)
	asserts.Equal("EUR", currency.From)
	asserts.Equal("USD", currency.To)
	asserts.Equal("TestProvider", currency.Provider)
	asserts.Equal(float32(0.8), currency.Rate)
}

func TestGetCurrenciesFromMongoDb(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	asserts := require.New(t)
	client, err := mongo.NewClient(options.Client().ApplyURI("mongodb://localhost:27017"))

	asserts.Nil(err)
	asserts.NotNil(client)

	err = client.Connect(ctx)

	asserts.Nil(err)

	defer client.Disconnect(ctx)

	database := client.Database("currency_fetcher_fetch")
	defer database.Drop(ctx)
	collection := database.Collection("currency")

	var currenciesToInsert []interface{}

	for i := 0; i < 10; i++ {
		currenciesToInsert = append(currenciesToInsert, map[string]interface{}{
			"currency":  "EUR_USD",
			"provider":  "TestProvider",
			"rate":      rand.Float32(),
			"createdAt": time.Now(),
		})
	}
	results, err := collection.InsertMany(ctx, currenciesToInsert)
	asserts.NotNil(results)
	asserts.Nil(err)

	t.Run("GetAllCurrenciesPaginated", func(t *testing.T) {
		storage := NewMongoStorage(ctx, collection)
		currencies, err := storage.Get("EUR", "USD", 1, 10)
		asserts.Nil(err)
		asserts.NotNil(currencies)
		asserts.Len(currencies, 10)

		for _, cur := range currencies {
			asserts.Equal("TestProvider", cur.Provider)
			asserts.Equal("EUR", cur.From)
			asserts.Equal("USD", cur.To)
		}
	})

	t.Run("GetWithProvider", func(t *testing.T) {
		storage := NewMongoStorage(ctx, collection)
		currencies, err := storage.GetByProvider("EUR", "USD", "TestProvider", 1, 10)
		asserts.Nil(err)
		asserts.NotNil(currencies)
		asserts.Len(currencies, 10)

		for _, cur := range currencies {
			asserts.Equal("TestProvider", cur.Provider)
			asserts.Equal("EUR", cur.From)
			asserts.Equal("USD", cur.To)
		}
	})

	t.Run("GetWithNonExistentProvider", func(t *testing.T) {
		storage := NewMongoStorage(ctx, collection)
		currencies, err := storage.GetByProvider("EUR", "USD", "NonExistentProvider", 1, 10)
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
				"currency":  "EUR_USD",
				"provider":  "OtherProvider",
				"rate":      rand.Float32(),
				"createdAt": now.Add(duration),
			})
		}

		results, err := collection.InsertMany(ctx, toInsert)
		asserts.NotNil(results)
		asserts.Nil(err)

		inFuture := now.Add(time.Duration(10) * time.Minute)
		storage := NewMongoStorage(ctx, collection)
		currencies, err := storage.GetByDate("EUR", "USD", inFuture.Add(time.Duration(-5)*time.Minute), inFuture, 1, 10)
		asserts.Nil(err)
		asserts.NotNil(currencies)
		asserts.Len(currencies, 5)
	})
}
