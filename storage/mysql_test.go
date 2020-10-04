package storage

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"github.com/BrosSquad/currency-fetcher"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type (
	IdGeneratorMock struct {
		mock.Mock
	}
	IdGeneratorWithNullBytes struct {
		mock.Mock
	}
	IdGeneratorWithLessBytes struct {
		mock.Mock
	}
)

func (i *IdGeneratorWithNullBytes) Generate() []byte {
	return nil
}

func (i *IdGeneratorWithLessBytes) Generate() []byte {
	bytes := make([]byte, 10)
	_, _ = rand.Read(bytes)
	return bytes
}

func (i *IdGeneratorMock) Generate() []byte {
	bytes, _ := uuid.MustParse("8f8fa9e1-f854-4c94-9316-cdc561c8f536").MarshalBinary()
	return bytes
}

func TestMysqlStorage_Store(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	asserts := require.New(t)
	db, err := sql.Open("mysql", "currency:currency@/currencydb")

	asserts.NotNil(db)
	asserts.Nil(err)

	defer db.ExecContext(ctx, "DROP TABLE currency_store_test;")

	_, err = db.ExecContext(ctx, `CREATE TABLE currency_store_test(
		id binary(36) PRIMARY KEY,
		currency varchar(20) NOT NULL,
		provider varchar(30) NOT NULL,
		rate float(8,4) NOT NULL,
		created_at timestamp DEFAULT CURRENT_TIMESTAMP 
	);`)
	asserts.Nil(err)
	_, err = db.ExecContext(ctx, `CREATE INDEX search_index ON currency_store_test(currency, provider, created_at);`)
	asserts.Nil(err)

	t.Run("InsertOne", func(t *testing.T) {
		generator := &IdGeneratorMock{}
		storage := NewMySQLStorage(ctx, db, "currency_store_test", generator)
		currencies, err := storage.Store([]currency_fetcher.Currency{
			{
				From:      "EUR",
				To:        "USD",
				Provider:  "TestProvider",
				Rate:      0.021,
				CreatedAt: time.Now(),
			},
		})

		asserts.Nil(err)
		asserts.NotNil(currencies)
		asserts.Len(currencies, 1)
		asserts.IsType(uuid.UUID{}, currencies[0].Id)
	})
	t.Run("InsertMany", func(t *testing.T) {
		storage := NewMySQLStorage(ctx, db, "currency_store_test", nil)
		currencies, err := storage.Store([]currency_fetcher.Currency{
			{
				From:      "EUR",
				To:        "USD",
				Provider:  "TestProvider",
				Rate:      0.021,
				CreatedAt: time.Now(),
			},
			{
				From:      "USD",
				To:        "EUR",
				Provider:  "TestProvider",
				Rate:      1.1,
				CreatedAt: time.Now(),
			},
			{
				From:      "PHP",
				To:        "EUR",
				Provider:  "TestProvider",
				Rate:      0.5,
				CreatedAt: time.Now(),
			},
		})

		asserts.Nil(err)
		asserts.NotNil(currencies)
		asserts.Len(currencies, 3)

		for _, cur := range currencies {
			asserts.IsType(uuid.UUID{}, cur.Id)
		}
	})

	t.Run("Not enough bytes in id generator", func(t *testing.T) {
		generators := []IdGenerator{
			&IdGeneratorWithNullBytes{},
			&IdGeneratorWithLessBytes{},
		}

		for _, gen := range generators {
			storage := NewMySQLStorage(ctx, db, "currency_store_test", gen)
			currencies, err := storage.Store([]currency_fetcher.Currency{
				{
					From:      "EUR",
					To:        "USD",
					Provider:  "TestProvider",
					Rate:      0.021,
					CreatedAt: time.Now(),
				},
			})

			asserts.Nil(currencies)
			asserts.NotNil(err)
			asserts.True(errors.Is(err, ErrNotEnoughBytesInGenerator))
		}
	})
}
