package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/BrosSquad/currency-fetcher"
	"github.com/google/uuid"
	"strings"
	"time"
)

const (
	MySQLStorageProviderName = "mysql"
	MySQLTimeFormat  = "2006-01-02 15:04:05"
)

var ErrNotEnoughBytesInGenerator = errors.New("id generator must return byte slice with 16 bytes in it")

type (
	IdGenerator interface {
		Generate() []byte
	}
	mysqlStorage struct {
		idGenerator IdGenerator
		ctx         context.Context
		db          *sql.DB
		tableName   string
	}
)

func NewMySQLStorage(ctx context.Context, db *sql.DB, tableName string, generator IdGenerator) currency_fetcher.Storage {
	return mysqlStorage{
		idGenerator: generator,
		ctx:         ctx,
		db:          db,
		tableName:   tableName,
	}
}

func (m mysqlStorage) Store(currency []currency_fetcher.Currency) ([]currency_fetcher.CurrencyWithId, error) {
	tx, err := m.db.Begin()

	if err != nil {
		return nil, err
	}

	var builder strings.Builder
	bind := make([]interface{}, 0, 5*len(currency))
	data := make([]currency_fetcher.CurrencyWithId, 0, len(currency))
	for _, cur := range currency {
		var id uuid.UUID
		if m.idGenerator == nil {
			id = uuid.New()
		} else {
			bytes := m.idGenerator.Generate()
			if bytes == nil || len(bytes) != 16 {
				return nil, ErrNotEnoughBytesInGenerator
			}
			id, err = uuid.FromBytes(m.idGenerator.Generate())
			if err != nil {
				return nil, err
			}
		}
		createdAt := cur.CreatedAt
		if createdAt.IsZero() {
			createdAt = time.Now()
		}
		builder.WriteString("(?,?,?,?,?),")
		bind = append(bind, id, fmt.Sprintf("%s_%s", cur.From, cur.To), cur.Provider, cur.Rate, createdAt)
		data = append(data, currency_fetcher.CurrencyWithId{
			Currency: cur,
			Id:       id,
		})
	}

	stmt, err := tx.PrepareContext(m.ctx, fmt.Sprintf("INSERT INTO %s(id, currency, provider, rate, created_at) VALUES %s;", m.tableName, strings.TrimRight(builder.String(), ", ")))

	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	_, err = stmt.ExecContext(m.ctx, bind...)

	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	if err := stmt.Close(); err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	return data, nil
}

func (m mysqlStorage) Get(from, to string, page, perPage int64) ([]currency_fetcher.CurrencyWithId, error) {
	return m.GetByProvider(from, to, "", page, perPage)
}

func (m mysqlStorage) GetByProvider(from, to, provider string, page, perPage int64) ([]currency_fetcher.CurrencyWithId, error) {
	return m.GetByDateAndProvider(from, to, provider, time.Time{}, time.Now(), page, perPage)
}

func (m mysqlStorage) GetByDate(from, to string, start, end time.Time, page, perPage int64) ([]currency_fetcher.CurrencyWithId, error) {
	return m.GetByDateAndProvider(from, to, "", start, end, page, perPage)
}

func (m mysqlStorage) GetByDateAndProvider(from, to, provider string, start, end time.Time, page, perPage int64) ([]currency_fetcher.CurrencyWithId, error) {
	if start.After(end) {
		return nil, errors.New("start time cannot be after end time")
	}

	var builder strings.Builder

	builder.WriteString("SELECT id,currency,provider,rate,created_at FROM ")
	builder.WriteString(m.tableName)
	builder.WriteString(" WHERE currency = ? AND created_at BETWEEN ? AND ?")

	if provider != "" {
		builder.WriteString(" AND provider = ?")
	}

	builder.WriteString(" ORDER BY created_at LIMIT ?, ?")

	stmt, err := m.db.PrepareContext(m.ctx, builder.String())

	if err != nil {
		return nil, err
	}

	rows, err := stmt.Query(fmt.Sprintf("%s_%s", from, to), start.Format(MySQLTimeFormat), end.Format(MySQLTimeFormat), perPage, (page-1)*perPage)

	if err != nil {
		return nil, err
	}

	defer rows.Close()
	result := make([]currency_fetcher.CurrencyWithId, 0, perPage)
	for rows.Next() {
		var currency string
		var createdAt string
		currencyWithId := currency_fetcher.CurrencyWithId{}
		if err := rows.Scan(&currencyWithId.Id, currency, &currencyWithId.Provider, &currencyWithId.Rate, &createdAt); err != nil {
			return nil, err
		}

		currencyWithId.CreatedAt, _ = time.Parse(MySQLTimeFormat, createdAt)
		isoCurrencies := strings.Split(currency, "_")
		currencyWithId.From = isoCurrencies[0]
		currencyWithId.To = isoCurrencies[1]
		result = append(result, currencyWithId)
	}

	return result, nil
}

func (m mysqlStorage) Migrate() error {
	_, err := m.db.ExecContext(m.ctx, `CREATE TABLE currency_store_test(
		id binary(36) PRIMARY KEY,
		currency varchar(20) NOT NULL,
		provider varchar(30) NOT NULL,
		rate float(8,4) NOT NULL,
		created_at timestamp DEFAULT CURRENT_TIMESTAMP 
	);`)

	if err != nil {
		return err
	}

	_, err = m.db.ExecContext(m.ctx, `CREATE INDEX search_index ON currency_store_test(currency, provider, created_at);`)
	return err
}

func (mysqlStorage) GetStorageProviderName() string {
	return MySQLStorageProviderName
}
