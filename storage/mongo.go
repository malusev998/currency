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

	currencyFetcher "github.com/malusev998/currency"
)

const MongoDBProviderName = "mongodb"

type mongoStorage struct {
	client         *mongo.Client
	db             *mongo.Database
	ctx            context.Context
	collection     *mongo.Collection
	collectionName string
}

func (m mongoStorage) Get(from, to string, page, perPage int64) ([]currencyFetcher.CurrencyWithID, error) {
	return m.GetByProvider(from, to, currencyFetcher.EmptyProvider, page, perPage)
}

func (m mongoStorage) GetByProvider(from, to string, provider currencyFetcher.Provider, page, perPage int64) ([]currencyFetcher.CurrencyWithID, error) {
	return m.GetByDateAndProvider(from, to, provider, time.Time{}, time.Now(), page, perPage)
}

func (m mongoStorage) GetByDate(from, to string, start, end time.Time, page, perPage int64) ([]currencyFetcher.CurrencyWithID, error) {
	return m.GetByDateAndProvider(from, to, "", start, end, page, perPage)
}

func (m mongoStorage) GetByDateAndProvider(from, to string, provider currencyFetcher.Provider, start, end time.Time, page, perPage int64) ([]currencyFetcher.CurrencyWithID, error) {
	filter := bson.M{
		"fetchers": fmt.Sprintf("%s_%s", from, to),
		"createdAt": bson.M{
			"$gte": start,
			"$lt":  end,
		},
	}

	if provider != currencyFetcher.EmptyProvider {
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

	currencies := make([]currencyFetcher.CurrencyWithID, 0, perPage)

	for cursor.Next(m.ctx) {
		current := cursor.Current
		isoCurrencies := strings.Split(current.Lookup("fetchers").StringValue(), "_")
		currencies = append(currencies, currencyFetcher.CurrencyWithID{
			Currency: currencyFetcher.Currency{
				From:      isoCurrencies[0],
				To:        isoCurrencies[1],
				Provider:  currencyFetcher.Provider(current.Lookup("provider").StringValue()),
				Rate:      float32(current.Lookup("rate").Double()),
				CreatedAt: current.Lookup("createdAt").Time(),
			},
			ID: current.Lookup("_id").ObjectID(),
		})
	}

	return currencies, nil
}

func (m mongoStorage) Store(currency []currencyFetcher.Currency) ([]currencyFetcher.CurrencyWithID, error) {
	currenciesToInsert := make([]interface{}, 0, len(currency))

	for _, cur := range currency {
		createdAt := cur.CreatedAt
		if createdAt.IsZero() {
			createdAt = time.Now()
		}

		currenciesToInsert = append(currenciesToInsert, bson.M{
			"fetchers":  fmt.Sprintf("%s_%s", cur.From, cur.To),
			"rate":      cur.Rate,
			"provider":  cur.Provider,
			"createdAt": createdAt,
		})
	}

	results, err := m.collection.InsertMany(m.ctx, currenciesToInsert)

	if err != nil {
		return nil, err
	}

	data := make([]currencyFetcher.CurrencyWithID, 0, len(results.InsertedIDs))
	for i, result := range results.InsertedIDs {
		data = append(data, currencyFetcher.CurrencyWithID{
			Currency: currency[i],
			ID:       result,
		})
	}

	return data, nil
}

func (m mongoStorage) Migrate() error {
	if err := m.db.CreateCollection(m.ctx, m.collectionName); err != nil {
		if _, ok := err.(mongo.CommandError); !ok {
			return fmt.Errorf("error while creating mongodb collection: %v", err)
		}
	}

	_, err := m.collection.Indexes().CreateOne(m.ctx, mongo.IndexModel{
		Keys: bsonx.Doc{
			{
				Key:   "fetchers",
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
				Key:   "fetchers",
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

func (m mongoStorage) Drop() error {
	return m.collection.Drop(m.ctx)
}

func (m *mongoStorage) Close() error {
	if err := m.client.Disconnect(m.ctx); err != nil {
		return fmt.Errorf("disconnecting from mongodb failed: %v", err)
	}

	return nil
}

func NewMongoStorage(c MongoDBConfig) (currencyFetcher.Storage, error) {
	mongoDbClient, err := mongo.NewClient(options.Client().ApplyURI(c.ConnectionString))

	if err != nil {
		return nil, fmt.Errorf("error while connecting to MongoDB database: %v", err)
	}

	if err := mongoDbClient.Connect(c.Cxt); err != nil {
		return nil, fmt.Errorf("rrror while connecting to mongodb: %v", err)
	}

	db := (*mongoDbClient).Database(c.Database)
	collection := db.Collection(c.Collection)

	storage := &mongoStorage{
		db:             db,
		client:         mongoDbClient,
		ctx:            c.Cxt,
		collection:     collection,
		collectionName: c.Collection,
	}

	if c.Migrate {
		if err := storage.Migrate(); err != nil {
			return nil, fmt.Errorf("error while migrating mongodb collection: %v", err)
		}
	}

	return storage, nil
}
