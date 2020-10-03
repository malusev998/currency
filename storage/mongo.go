package storage

import (
	"context"
	"fmt"
	"github.com/BrosSquad/currency-fetcher"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"strings"
	"time"
)

type mongoStorage struct {
	ctx        context.Context
	collection *mongo.Collection
}

func (m mongoStorage) Get(from, to string, page, perPage int64) ([]currency_fetcher.CurrencyWithId, error) {
	return m.GetByProvider(from, to, "", page, perPage)
}

func (m mongoStorage) GetByProvider(from, to, provider string, page, perPage int64) ([]currency_fetcher.CurrencyWithId, error) {
	return m.GetByDateAndProvider(from, to, provider, time.Time{}, time.Now(), page, perPage)
}

func (m mongoStorage) GetByDate(from, to string, start, end time.Time, page, perPage int64) ([]currency_fetcher.CurrencyWithId, error) {
	return m.GetByDateAndProvider(from, to, "", start, end, page, perPage)
}

func (m mongoStorage) GetByDateAndProvider(from, to, provider string, start, end time.Time, page, perPage int64) ([]currency_fetcher.CurrencyWithId, error) {
	filter := bson.M{
		"currency": fmt.Sprintf("%s_%s", from, to),
		"createdAt": bson.M{
			"$gte": start,
			"$lt":  end,
		},
	}

	if provider != "" {
		filter["provider"] = provider
	}

	skip := (page - 1) * perPage
	cursor, err := m.collection.Find(m.ctx, filter, &options.FindOptions{
		Limit: &perPage,
		Skip:  &skip,
	})

	if err != nil {
		return nil, err
	}

	defer cursor.Close(m.ctx)

	currencies := make([]currency_fetcher.CurrencyWithId, 0, perPage)

	for cursor.Next(m.ctx) {
		current := cursor.Current
		isoCurrencies := strings.Split(current.Lookup("currency").StringValue(), "_")
		currencies = append(currencies, currency_fetcher.CurrencyWithId{
			Currency: currency_fetcher.Currency{
				From:      isoCurrencies[0],
				To:        isoCurrencies[1],
				Provider:  current.Lookup("provider").StringValue(),
				Rate:      float32(current.Lookup("rate").Double()),
				CreatedAt: current.Lookup("createdAt").Time(),
			},
			Id: current.Lookup("_id").ObjectID(),
		})
	}

	return currencies, nil
}

func (m mongoStorage) Store(currency currency_fetcher.Currency) (currency_fetcher.CurrencyWithId, error) {
	if currency.CreatedAt.IsZero() {
		currency.CreatedAt = time.Now()
	}

	result, err := m.collection.InsertOne(m.ctx, bson.M{
		"currency":  fmt.Sprintf("%s_%s", currency.From, currency.To),
		"rate":      currency.Rate,
		"provider":  currency.Provider,
		"createdAt": currency.CreatedAt,
	})

	if err != nil {
		return currency_fetcher.CurrencyWithId{}, err
	}

	return currency_fetcher.CurrencyWithId{
		Currency: currency,
		Id:       result.InsertedID,
	}, nil
}

func NewMongoStorage(ctx context.Context, collection *mongo.Collection) currency_fetcher.Storage {
	return mongoStorage{
		ctx:        ctx,
		collection: collection,
	}
}
