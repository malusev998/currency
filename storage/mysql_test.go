package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bxcodec/faker/v3"
	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	currency_fetcher "github.com/malusev998/currency-fetcher"
)

type (
	IDGeneratorMock struct {
		mock.Mock
	}
	IDGeneratorWithNullBytes struct {
		mock.Mock
	}
	IDGeneratorWithLessBytes struct {
		mock.Mock
	}
)

func (i *IDGeneratorWithNullBytes) Generate() []byte {
	return nil
}

func (i *IDGeneratorWithLessBytes) Generate() []byte {
	bytes := make([]byte, 10)
	_, _ = rand.Read(bytes)

	return bytes
}

func (i *IDGeneratorMock) Generate() []byte {
	bytes, _ := uuid.MustParse("8f8fa9e1-f854-4c94-9316-cdc561c8f536").MarshalBinary()
	return bytes
}

func mysqlConnectionString() string {
	runningInDocker := false

	if os.Getenv("RUNNING_IN_DOCKER") != "" {
		runningInDocker = true
	}

	mysqlDriverConfig := mysql.NewConfig()
	mysqlDriverConfig.User = "currency"
	mysqlDriverConfig.Passwd = "currency"
	mysqlDriverConfig.DBName = "currencydb"
	mysqlDriverConfig.Net = "tcp"

	if runningInDocker {
		mysqlDriverConfig.Addr = "mysql:3306"
	} else {
		mysqlDriverConfig.Addr = "localhost:3306"
	}
	return mysqlDriverConfig.FormatDSN()
}

func connectToMysql() (*sql.DB, error) {
	return sql.Open("mysql", mysqlConnectionString())
}

func seedMysql(ctx context.Context, db *sql.DB) error {
	var builder strings.Builder

	builder.WriteString("INSERT INTO currency_get_test(id, currency, provider, rate, created_at) VALUES ")

	for i := 0; i < 100; i++ {
		builder.WriteString(fmt.Sprintf("('%s','%s_%s','%s',%f,'%s'),", faker.UUIDHyphenated(), faker.Currency(), faker.Currency(), "TestProvider", rand.Float32(), time.Now().Add(-time.Duration(i)*time.Minute).Format(MySQLTimeFormat)))
	}

	builder.WriteString("('" + faker.UUIDHyphenated() + "',")
	builder.WriteString("'EUR_USD', 'TestProvider', 1.8,")
	builder.WriteString("'" + time.Now().Format(MySQLTimeFormat) + "'")
	builder.WriteString(");")

	_, err := db.ExecContext(ctx, builder.String())

	return err
}

//func TestMysqlStorage_Get(t *testing.T) {
//	t.Parallel()
//	asserts := require.New(t)
//	ctx := context.Background()
//	db, err := connectToMysql()
//	asserts.NotNil(db)
//	asserts.Nil(err)
//
//	defer db.ExecContext(ctx, "DROP TABLE currency_get_test;")
//
//	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS currency_get_test(
//		id binary(36) PRIMARY KEY,
//		currency varchar(20) NOT NULL,
//		provider varchar(30) NOT NULL,
//		rate float(8,4) NOT NULL,
//		created_at timestamp DEFAULT CURRENT_TIMESTAMP
//	);`)
//	asserts.Nil(err)
//	_, err = db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS search_index ON currency_store_test(currency, provider, created_at);`)
//	asserts.Nil(err)
//	asserts.Nil(seedMysql(ctx, db))
//
//}

func TestMysqlStorage_Store(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	asserts := require.New(t)

	t.Run("InsertOne", func(t *testing.T) {
		generator := &IDGeneratorMock{}
		storage, _ := NewMySQLStorage(MySQLConfig{
			BaseConfig: BaseConfig{
				Cxt:     ctx,
				Migrate: true,
			},
			ConnectionString: mysqlConnectionString(),
			TableName:        "currency_store_test_insert",
			IDGenerator:      generator,
		})
		defer storage.Drop()

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
		asserts.IsType(uuid.UUID{}, currencies[0].ID)
	})

	t.Run("InsertMany", func(t *testing.T) {
		storage, _ := NewMySQLStorage(MySQLConfig{
			BaseConfig: BaseConfig{
				Cxt:     ctx,
				Migrate: true,
			},
			ConnectionString: mysqlConnectionString(),
			TableName:        "currency_store_test_insert_many",
			IDGenerator:      nil,
		})
		defer storage.Drop()

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
			asserts.IsType(uuid.UUID{}, cur.ID)
		}

	})

	t.Run("Not enough bytes in id generator", func(t *testing.T) {
		generators := []IDGenerator{
			&IDGeneratorWithNullBytes{},
			&IDGeneratorWithLessBytes{},
		}

		for _, gen := range generators {
			storage, _ := NewMySQLStorage(MySQLConfig{
				BaseConfig: BaseConfig{
					Cxt:     ctx,
					Migrate: true,
				},
				ConnectionString: mysqlConnectionString(),
				TableName:        "currency_store_test",
				IDGenerator:      gen,
			})
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
			storage.Drop()
		}
	})
}
