package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx"

	currency_fetcher "github.com/malusev998/currency-fetcher"
)

const MongoDBProviderName = "mongodb"

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
		Sort: bson.M{
			"createdAt": -1,
		},
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

func (m mongoStorage) Store(currency []currency_fetcher.Currency) ([]currency_fetcher.CurrencyWithId, error) {
	currenciesToInsert := make([]interface{}, 0, len(currency))

	for _, cur := range currency {
		createdAt := cur.CreatedAt
		if createdAt.IsZero() {
			createdAt = time.Now()
		}

		currenciesToInsert = append(currenciesToInsert, bson.M{
			"currency":  fmt.Sprintf("%s_%s", cur.From, cur.To),
			"rate":      cur.Rate,
			"provider":  cur.Provider,
			"createdAt": createdAt,
		})
	}

	results, err := m.collection.InsertMany(m.ctx, currenciesToInsert)

	if err != nil {
		return nil, err
	}

	data := make([]currency_fetcher.CurrencyWithId, 0, len(results.InsertedIDs))
	for i, result := range results.InsertedIDs {
		data = append(data, currency_fetcher.CurrencyWithId{
			Currency: currency[i],
			Id:       result,
		})
	}

	return data, nil
}

func (m mongoStorage) Migrate() error {
	_, err := m.collection.Indexes().CreateOne(m.ctx, mongo.IndexModel{
		Keys: bsonx.Doc{
			{
				Key:   "currency",
				Value: bsonx.Int32(1),
			},
			{
				Key:   "createdAt",
				Value: bsonx.Int32(-1),
			},
		},
		Options: nil,
	})

	if err != nil {
		return err
	}

	_, err = m.collection.Indexes().CreateOne(m.ctx, mongo.IndexModel{
		Keys: bsonx.Doc{
			{
				Key:   "currency",
				Value: bsonx.Int32(1),
			},
			{
				Key:   "provider",
				Value: bsonx.Int32(1),
			},
			{
				Key:   "createdAt",
				Value: bsonx.Int32(-1),
			},
		},
		Options: nil,
	})

	return err

}

func (mongoStorage) GetStorageProviderName() string {
	return MongoDBProviderName
}

func NewMongoStorage(ctx context.Context, collection *mongo.Collection) currency_fetcher.Storage {
	return mongoStorage{
		ctx:        ctx,
		collection: collection,
	}
}
