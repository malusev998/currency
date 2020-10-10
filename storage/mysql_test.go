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

	currency_fetcher "github.com/BrosSquad/currency-fetcher"
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

func connectToMysql() (*sql.DB, error) {
	runningInDocker := false

	if os.Getenv("RUNNING_IN_DOCKER") != "" {
		runningInDocker = true
	}

	mysqlDriverConfig := mysql.NewConfig()
	mysqlDriverConfig.User = "currency"
	mysqlDriverConfig.Passwd = "currency"
	mysqlDriverConfig.DBName = "currencydb"
	connectionString := mysqlDriverConfig.FormatDSN()

	if runningInDocker {
		mysqlDriverConfig.Addr = "mysql:3306"
	} else {
		mysqlDriverConfig.Addr = "localhost:3306"
	}

	return sql.Open("mysql", connectionString)
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

func TestMysqlStorage_Get(t *testing.T) {
	t.Parallel()
	asserts := require.New(t)
	ctx := context.Background()
	db, err := connectToMysql()
	asserts.NotNil(db)
	asserts.Nil(err)

	defer db.ExecContext(ctx, "DROP TABLE currency_get_test;")

	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS currency_get_test(
		id binary(36) PRIMARY KEY,
		currency varchar(20) NOT NULL,
		provider varchar(30) NOT NULL,
		rate float(8,4) NOT NULL,
		created_at timestamp DEFAULT CURRENT_TIMESTAMP 
	);`)
	asserts.Nil(err)
	_, err = db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS search_index ON currency_store_test(currency, provider, created_at);`)
	asserts.Nil(err)
	asserts.Nil(seedMysql(ctx, db))

}

func TestMysqlStorage_Store(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	asserts := require.New(t)

	db, err := connectToMysql()
	asserts.NotNil(db)
	asserts.Nil(err)

	defer db.ExecContext(ctx, "DROP TABLE currency_store_test;")

	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS currency_store_test(
		id binary(36) PRIMARY KEY,
		currency varchar(20) NOT NULL,
		provider varchar(30) NOT NULL,
		rate float(8,4) NOT NULL,
		created_at timestamp DEFAULT CURRENT_TIMESTAMP 
	);`)
	asserts.Nil(err)
	_, err = db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS search_index ON currency_store_test(currency, provider, created_at);`)
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
