package storage

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/BrosSquad/currency-fetcher"
	"time"
)

type mysqlStorage struct {
	ctx       context.Context
	db        *sql.DB
	tableName string
}

func NewMySQLStorage(ctx context.Context, db *sql.DB, tableName string) currency_fetcher.Storage {
	return mysqlStorage{
		ctx:       ctx,
		db:        db,
		tableName: tableName,
	}
}

func (m mysqlStorage) Store(currency currency_fetcher.Currency) (currency_fetcher.CurrencyWithId, error) {
	if currency.CreatedAt.IsZero() {
		currency.CreatedAt = time.Now()
	}

	combinedCurrency := fmt.Sprintf("%s_%s", currency.From, currency.To)

	tx, err := m.db.Begin()

	if err != nil {
		return currency_fetcher.CurrencyWithId{}, err
	}

	stmt, err := tx.PrepareContext(m.ctx, fmt.Sprintf("INSERT INTO %s(currency, provider, rate, created_at) VALUES(?,?,?,?);", m.tableName))

	if err != nil {
		_ = tx.Rollback()
		return currency_fetcher.CurrencyWithId{}, err
	}

	_, err = stmt.ExecContext(m.ctx, combinedCurrency, currency.Provider, currency.Rate, currency.CreatedAt)

	if err != nil {
		_ = tx.Rollback()
		return currency_fetcher.CurrencyWithId{}, err
	}

	stmt, err = tx.PrepareContext(m.ctx, fmt.Sprintf("SELECT id FROM %s WHERE currency = ? AND provider = ? AND rate = ? AND created_at = ? LIMIT 1;", m.tableName))

	if err != nil {
		return currency_fetcher.CurrencyWithId{}, err
	}

	var id uint64
	row := stmt.QueryRowContext(m.ctx, combinedCurrency, currency.Provider, currency.Rate, currency.CreatedAt)

	if err := row.Scan(&id); err != nil {
		_ = tx.Rollback()
		return currency_fetcher.CurrencyWithId{}, err
	}

	if err := stmt.Close(); err != nil {
		_ = tx.Rollback()
		return currency_fetcher.CurrencyWithId{}, err
	}

	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		return currency_fetcher.CurrencyWithId{}, err
	}

	return currency_fetcher.CurrencyWithId{
		Currency: currency,
		Id: id,
	}, nil
}

func (m mysqlStorage) Get(from, to string, page, perPage int64) ([]currency_fetcher.CurrencyWithId, error) {
	panic("implement me")
}

func (m mysqlStorage) GetByProvider(from, to, provider string, page, perPage int64) ([]currency_fetcher.CurrencyWithId, error) {
	panic("implement me")
}

func (m mysqlStorage) GetByDate(from, to string, start, end time.Time, page, perPage int64) ([]currency_fetcher.CurrencyWithId, error) {
	panic("implement me")
}

func (m mysqlStorage) GetByDateAndProvider(from, to, provider string, start, end time.Time, page, perPage int64) ([]currency_fetcher.CurrencyWithId, error) {
	panic("implement me")
}
