package storage

import (
	"context"
	"fmt"
	"github.com/BrosSquad/currency-fetcher"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type mongoStorage struct {
	ctx        context.Context
	collection *mongo.Collection
}

func (m mongoStorage) Store(currency currency_fetcher.Currency) (currency_fetcher.CurrencyWithId, error) {
	result, err := m.collection.InsertOne(m.ctx, bson.D{
		{Key: "currency", Value: fmt.Sprintf("%s_%s", currency.From, currency.To)},
		{Key: "rate", Value: currency.Rate},
		{Key: "provider", Value: currency.Provider},
	})

	if err != nil {
		return currency_fetcher.CurrencyWithId{}, nil
	}

	return currency_fetcher.CurrencyWithId{
		Currency: currency,
		Id: result.InsertedID,
	}, nil
}

func (m mongoStorage) Get(s string, s2 string) currency_fetcher.CurrencyWithId {
	panic("implement me")
}

func NewMongoStorage(ctx context.Context, collection *mongo.Collection) currency_fetcher.Storage {
	return mongoStorage{
		ctx:        ctx,
		collection: collection,
	}
}
