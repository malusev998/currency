package storage_test

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

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/bxcodec/faker/v3"
	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/malusev998/currency"
	"github.com/malusev998/currency/storage"
)

type (
	IDGeneratorMock struct {
		mock.Mock
	}
)

func (i *IDGeneratorMock) Generate() []byte {
	args := i.Called()
	if value, ok := args.Get(0).([]byte); ok {
		return value
	}
	return nil
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
		builder.WriteString(fmt.Sprintf("('%s','%s_%s','%s',%f,'%s'),", faker.UUIDHyphenated(), faker.Currency(), faker.Currency(), "TestProvider", rand.Float32(), time.Now().Add(-time.Duration(i)*time.Minute).Format(storage.MySQLTimeFormat)))
	}

	builder.WriteString("('" + faker.UUIDHyphenated() + "',")
	builder.WriteString("'EUR_USD', 'TestProvider', 1.8,")
	builder.WriteString("'" + time.Now().Format(storage.MySQLTimeFormat) + "'")
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

	st, _ := storage.NewMySQLStorage(storage.MySQLConfig{
		BaseConfig: storage.BaseConfig{
			Cxt:     ctx,
			Migrate: true,
		},
		ConnectionString: mysqlConnectionString(),
		TableName:        "currency_get_test",
		IDGenerator:      nil,
	})
	defer st.Drop()
	asserts.Nil(seedMysql(ctx, db))

	result, err := st.Get("EUR", "USD", 1, 10)
	asserts.Nil(err)
	asserts.Len(result, 1)
}

func TestMySQL_InsertOne(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	asserts := require.New(t)
	st, _ := storage.NewMySQLStorage(storage.MySQLConfig{
		BaseConfig: storage.BaseConfig{
			Cxt:     ctx,
			Migrate: true,
		},
		ConnectionString: mysqlConnectionString(),
		TableName:        "currency_store_test_insert",
	})
	defer st.Drop()

	currencies, err := st.Store([]currency.Currency{
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
}

func TestMySQL_InsertMany(t *testing.T) {
	t.Parallel()

	asserts := require.New(t)
	st, _ := storage.NewMySQLStorage(storage.MySQLConfig{
		BaseConfig: storage.BaseConfig{
			Cxt:     context.Background(),
			Migrate: true,
		},
		ConnectionString: mysqlConnectionString(),
		TableName:        "currency_store_test_insert_many",
		IDGenerator:      nil,
	})
	defer st.Drop()

	currencies, err := st.Store([]currency.Currency{
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

}

func TestMysqlStorage_Store(t *testing.T) {
	t.Parallel()
	asserts := require.New(t)
	idNullBytes := &IDGeneratorMock{}
	idLessBytes := &IDGeneratorMock{}

	idNullBytes.On("Generate").Return(nil)
	idLessBytes.On("Generate").Return(make([]byte, 10))
	generators := []storage.IDGenerator{
		idNullBytes,
		idLessBytes,
	}

	for _, gen := range generators {
		st, _ := storage.NewMySQLStorage(storage.MySQLConfig{
			BaseConfig: storage.BaseConfig{
				Cxt:     context.Background(),
				Migrate: true,
			},
			ConnectionString: mysqlConnectionString(),
			TableName:        "currency_store_test",
			IDGenerator:      gen,
		})
		currencies, err := st.Store([]currency.Currency{
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
		asserts.True(errors.Is(err, storage.ErrNotEnoughBytesInGenerator))
		_ = st.Drop()
	}
}

func TestMysqlStorage_StoreUnit(t *testing.T) {
	t.Parallel()
	db, m, _ := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	defer db.Close()
	assert := require.New(t)
	ctx := context.Background()
	st, _ := storage.NewSQLStorage(ctx, db, nil, "currency_store_test_unit", false)

	currencies := []currency.Currency{
		{
			From:      "EUR",
			To:        "USD",
			Provider:  "TestProvider",
			Rate:      0.021,
			CreatedAt: time.Now(),
		},
	}

	t.Run("Transaction_Not_Started", func(t *testing.T) {
		m.ExpectBegin().WillReturnError(errors.New("error while starting transaction"))
		_, err := st.Store(currencies)
		assert.Error(err)
		assert.Nil(m.ExpectationsWereMet())
		assert.Equal("error while starting transaction", err.Error())
	})

	t.Run("Prepare_SQL_WithError", func(t *testing.T) {
		m.ExpectBegin()
		m.ExpectPrepare("INSERT INTO currency_store_test_unit(id, currency, provider, rate, created_at) VALUES (?,?,?,?,?);").
			WillReturnError(errors.New("cannot create prepare statement"))
		m.ExpectRollback()

		_, err := st.Store(currencies)
		assert.Nil(m.ExpectationsWereMet())
		assert.Error(err)
		assert.Equal("cannot create prepare statement", err.Error())
	})

}
