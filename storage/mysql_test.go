package storage

import (
	"context"
	"database/sql"
	"github.com/BrosSquad/currency-fetcher"
	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestMysqlStorage_Store(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	asserts := require.New(t)
	db, err := sql.Open("mysql", "root:password@/currencydb")

	asserts.NotNil(db)
	asserts.Nil(err)

	defer db.ExecContext(ctx, "DROP TABLE currency_store_test;")

	_, err = db.ExecContext(ctx, `CREATE TABLE currency_store_test(
		id int(11) PRIMARY KEY AUTO_INCREMENT,
		currency varchar(20) NOT NULL,
		provider varchar(30) NOT NULL,
		rate float(8,4) NOT NULL,
		created_at timestamp DEFAULT CURRENT_TIMESTAMP 
	);`)
	asserts.Nil(err)

	_, err = db.ExecContext(ctx, `CREATE INDEX all_index ON currency_store_test(currency, provider, rate, created_at);`)
	asserts.Nil(err)
	_, err = db.ExecContext(ctx, `CREATE INDEX search_index ON currency_store_test(currency, provider, created_at);`)
	asserts.Nil(err)


	storage := NewMySQLStorage(ctx, db, "currency_store_test")

	currency, err := storage.Store(currency_fetcher.Currency{
		From:      "EUR",
		To:       "USD",
		Provider:  "TestProvider",
		Rate:      0.021,
		CreatedAt: time.Now(),
	})

	asserts.Nil(err)
	asserts.Equal(uint64(1), currency.Id)
}
