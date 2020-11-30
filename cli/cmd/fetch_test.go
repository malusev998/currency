package cmd

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"

	currencyFetcher "github.com/malusev998/currency-fetcher"
	"github.com/malusev998/currency-fetcher/currency"
	"github.com/malusev998/currency-fetcher/services"
	"github.com/malusev998/currency-fetcher/storage"
)

const mysqlTableName = "currency_integration_test"

type (
	httpMock  struct{}
	mysqlData struct {
		ID        string
		Currency  string
		Provider  currencyFetcher.Provider
		Rate      float32
		CreatedAt string
	}
)

func (h httpMock) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	payload, _ := json.Marshal(map[string]float32{
		"EUR_USD": 1.2,
		"USD_EUR": 0.8,
	})

	writer.WriteHeader(200)
	writer.Write(payload)
}

func storages(t *testing.T, ctx context.Context) ([]currencyFetcher.Storage, *sql.DB) {
	st := make([]currencyFetcher.Storage, 0)
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
		mysqlDriverConfig.Addr = "sqlStorage:3306"
	} else {
		mysqlDriverConfig.Addr = "localhost:3306"
	}

	connectionString := mysqlDriverConfig.FormatDSN()

	db, err := sql.Open("mysql", connectionString)

	if err != nil {
		t.FailNow()
	}

	sqlStorage, _ := storage.NewMySQLStorage(storage.MySQLConfig{
		BaseConfig: storage.BaseConfig{
			Cxt:     ctx,
			Migrate: true,
		},
		ConnectionString: connectionString,
		TableName:        mysqlTableName,
		IDGenerator:      nil,
	})

	if err := sqlStorage.Migrate(); err != nil {
		t.FailNow()
	}

	st = append(st, sqlStorage)

	return st, db
}

func testMySQLDataSet(asserts *require.Assertions, rows *sql.Rows) {
	mysqlDataSet := make([]mysqlData, 0)

	for rows.Next() {
		set := mysqlData{}
		rows.Scan(&set.ID, &set.Currency, &set.Provider, &set.Rate, &set.CreatedAt)
		mysqlDataSet = append(mysqlDataSet, set)
	}

	asserts.Len(mysqlDataSet, 2)

	for _, set := range mysqlDataSet {
		asserts.Contains([]string{"EUR_USD", "USD_EUR"}, set.Currency)
		asserts.Equal(currencyFetcher.FreeConvProvider, set.Provider)
		asserts.Contains([]float32{1.2, 0.8}, set.Rate)
	}
}

func TestFetchCommand(t *testing.T) {
	t.Parallel()
	asserts := require.New(t)
	debug := false
	ctx := context.Background()
	server := httptest.NewServer(httpMock{})

	defer server.Close()

	st, mysqlDb := storages(t, ctx)

	for _, s := range st {
		defer s.Drop()
	}

	currencyService := []currencyFetcher.Service{
		services.Service{
			Fetcher: currency.NewCurrencyFetcher(currencyFetcher.FreeConvProvider, currency.FreeConvServiceConfig{
				BaseConfig: currency.BaseConfig{
					Ctx: ctx,
					URL: server.URL,
				},
				APIKey:             "123456",
				MaxPerHourRequests: 100,
				MaxPerRequest:      2,
			}),
			Storage: st,
		},
	}

	config := Config{
		Ctx:               ctx,
		debug:             &debug,
		CurrenciesToFetch: []string{"EUR_USD", "USD_EUR"},
		CurrencyService:   currencyService,
	}

	t.Run("Without Debug", func(t *testing.T) {
		cmd := fetch(&config)
		asserts.Nil(cmd.Execute())

		rows, err := mysqlDb.Query(fmt.Sprintf("SELECT * FROM %s;", mysqlTableName))

		asserts.Nil(err)

		defer rows.Close()

		testMySQLDataSet(asserts, rows)
	})
}
