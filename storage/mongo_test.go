package storage_test

import (
	"context"
	"github.com/malusev998/currency"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/malusev998/currency/storage"
)

func getMongoURI() string {
	uri := "mongodb://localhost:27017"

	if os.Getenv("RUNNING_IN_DOCKER") != "" {
		uri = "mongodb://mongo:27017"
	}

	return uri
}

func getMongoDB(dbName string) (*mongo.Client, *mongo.Database) {
	client, _ := mongo.NewClient(options.Client().ApplyURI(getMongoURI()))
	db := client.Database(dbName)
	client.Connect(context.Background())
	return client, db
}

func TestGetStorageProviderName(t *testing.T) {
	assert := require.New(t)
	st, _ := storage.NewMongoStorage(storage.MongoDBConfig{
		BaseConfig: storage.BaseConfig{
			Cxt:     context.Background(),
			Migrate: true,
		},
		ConnectionString: getMongoURI(),
		Database:         "currency_fetcher_store_one",
		Collection:       "fetchers",
	})

	assert.Equal(storage.MongoDBProviderName, st.GetStorageProviderName())
}

func TestStoreOne(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	assert := require.New(t)

	st, _ := storage.NewMongoStorage(storage.MongoDBConfig{
		BaseConfig: storage.BaseConfig{
			Cxt:     ctx,
			Migrate: true,
		},
		ConnectionString: getMongoURI(),
		Database:         "currency_fetcher_store_one",
		Collection:       "fetchers",
	})
	defer st.Drop()
	defer st.Close()

	provider := currency.Provider("TestProvider")

	currencies, err := st.Store([]currency.Currency{
		{
			From:     "EUR",
			To:       "USD",
			Provider: provider,
			Rate:     0.8,
		},
	})

	assert.Nil(err)
	assert.Len(currencies, 1)
	assert.NotNil(currencies[0].ID)
	assert.Equal("EUR", currencies[0].From)
	assert.Equal("USD", currencies[0].To)
	assert.Equal(provider, currencies[0].Provider)
	assert.Equal(float32(0.8), currencies[0].Rate)
}

func TestStoreMany(t *testing.T) {
	t.Parallel()
	assert := require.New(t)
	st, _ := storage.NewMongoStorage(storage.MongoDBConfig{
		BaseConfig: storage.BaseConfig{
			Cxt:     context.Background(),
			Migrate: true,
		},
		ConnectionString: getMongoURI(),
		Database:         "currency_fetcher_store_many",
		Collection:       "fetchers",
	})
	defer st.Drop()
	defer st.Close()

	currencies, err := st.Store([]currency.Currency{
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

	assert.Nil(err)
	assert.Len(currencies, 2)
	for _, cur := range currencies {
		assert.NotNil(cur.ID)
		assert.Equal("EUR", cur.From)
		assert.Equal("USD", cur.To)
		assert.Equal(currency.Provider("TestProvider"), cur.Provider)
		assert.Equal(float32(0.8), cur.Rate)
	}
}

func TestGetCurrenciesFromMongoDb(t *testing.T) {
	t.Parallel()
	provider := currency.Provider("TestProvider")
	otherProvider := currency.Provider("OtherProvider")

	ctx := context.Background()
	assert := require.New(t)

	_, db := getMongoDB("currency_fetcher_fetch")

	collection := db.Collection("fetchers")
	defer db.Drop(ctx)

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
	assert.NotNil(results)
	assert.Nil(err)

	t.Run("GetAllCurrenciesPaginated", func(t *testing.T) {
		st, _ := storage.NewMongoStorage(storage.MongoDBConfig{
			BaseConfig: storage.BaseConfig{
				Cxt:     context.Background(),
				Migrate: false,
			},
			ConnectionString: getMongoURI(),
			Database:         "currency_fetcher_fetch",
			Collection:       "fetchers",
		})
		currencies, err := st.Get("EUR", "USD", 1, 10)
		assert.Nil(err)
		assert.NotNil(currencies)
		assert.Len(currencies, 10)

		for _, cur := range currencies {
			assert.NotNil(cur.ID)
			assert.IsType(primitive.ObjectID{}, cur.ID)
			assert.Equal(provider, cur.Provider)
			assert.Equal("EUR", cur.From)
			assert.Equal("USD", cur.To)
		}
	})

	t.Run("GetWithProvider", func(t *testing.T) {
		st, _ := storage.NewMongoStorage(storage.MongoDBConfig{
			BaseConfig: storage.BaseConfig{
				Cxt:     ctx,
				Migrate: false,
			},
			ConnectionString: getMongoURI(),
			Database:         "currency_fetcher_fetch",
			Collection:       "fetchers",
		})
		currencies, err := st.GetByProvider("EUR", "USD", provider, 1, 10)
		assert.Nil(err)
		assert.NotNil(currencies)
		assert.Len(currencies, 10)

		for _, cur := range currencies {
			assert.Equal(provider, cur.Provider)
			assert.Equal("EUR", cur.From)
			assert.Equal("USD", cur.To)
		}
	})

	t.Run("GetWithNonExistentProvider", func(t *testing.T) {
		st, _ := storage.NewMongoStorage(storage.MongoDBConfig{
			BaseConfig: storage.BaseConfig{
				Cxt:     ctx,
				Migrate: false,
			},
			ConnectionString: getMongoURI(),
			Database:         "currency_fetcher_fetch",
			Collection:       "fetchers",
		})

		currencies, err := st.GetByProvider("EUR", "USD", "NonExistentProvider", 1, 10)
		assert.Nil(err)
		assert.NotNil(currencies)
		assert.Len(currencies, 0)
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
		assert.NotNil(results)
		assert.Nil(err)

		inFuture := now.Add(time.Duration(10) * time.Minute)
		st, _ := storage.NewMongoStorage(storage.MongoDBConfig{
			BaseConfig: storage.BaseConfig{
				Cxt:     ctx,
				Migrate: false,
			},
			ConnectionString: getMongoURI(),
			Database:         "currency_fetcher_fetch",
			Collection:       "fetchers",
		})
		currencies, err := st.GetByDate("EUR", "USD", inFuture.Add(time.Duration(-5)*time.Minute), inFuture, 1, 10)
		assert.Nil(err)
		assert.NotNil(currencies)
		assert.Len(currencies, 5)
	})
}
