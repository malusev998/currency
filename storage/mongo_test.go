package storage

import (
	"context"
	"github.com/BrosSquad/currency-fetcher"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
	"testing"
)

func TestStoreInMongo(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	asserts := require.New(t)
	client, err := mongo.NewClient()

	asserts.Nil(err)
	asserts.NotNil(client)

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
	asserts.Equal(0.8, currency.Rate)
}
