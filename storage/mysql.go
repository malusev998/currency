package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	currencyFetcher "github.com/malusev998/currency-fetcher"
)

const (
	MySQLStorageProviderName = "mysql"
	MySQLTimeFormat          = "2006-01-02 15:04:05"
)

var ErrNotEnoughBytesInGenerator = errors.New("id generator must return byte slice with 16 bytes in it")

type (
	IDGenerator interface {
		Generate() []byte
	}
	mysqlStorage struct {
		idGenerator IDGenerator
		ctx         context.Context
		db          *sql.DB
		tableName   string
	}
)

func (m mysqlStorage) Store(currency []currencyFetcher.Currency) ([]currencyFetcher.CurrencyWithID, error) {
	tx, err := m.db.Begin()

	if err != nil {
		return nil, err
	}

	var builder strings.Builder
	bind := make([]interface{}, 0, 5*len(currency))
	data := make([]currencyFetcher.CurrencyWithID, 0, len(currency))
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
		data = append(data, currencyFetcher.CurrencyWithID{
			Currency: cur,
			ID:       id,
		})
	}

	stmt, err := tx.PrepareContext(m.ctx, fmt.Sprintf("INSERT INTO %s(id, fetchers, provider, rate, created_at) VALUES %s;", m.tableName, strings.TrimRight(builder.String(), ", ")))

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

func (m mysqlStorage) Get(from, to string, page, perPage int64) ([]currencyFetcher.CurrencyWithID, error) {
	return m.GetByProvider(from, to, "", page, perPage)
}

func (m mysqlStorage) GetByProvider(from, to string, provider currencyFetcher.Provider, page, perPage int64) ([]currencyFetcher.CurrencyWithID, error) {
	return m.GetByDateAndProvider(from, to, provider, time.Time{}, time.Now(), page, perPage)
}

func (m mysqlStorage) GetByDate(from, to string, start, end time.Time, page, perPage int64) ([]currencyFetcher.CurrencyWithID, error) {
	return m.GetByDateAndProvider(from, to, "", start, end, page, perPage)
}

func (m mysqlStorage) GetByDateAndProvider(from, to string, provider currencyFetcher.Provider, start, end time.Time, page, perPage int64) ([]currencyFetcher.CurrencyWithID, error) {
	if start.After(end) {
		return nil, errors.New("start time cannot be after end time")
	}

	var builder strings.Builder

	builder.WriteString("SELECT id,fetchers,provider,rate,created_at FROM ")
	builder.WriteString(m.tableName)
	builder.WriteString(" WHERE fetchers = ? AND created_at BETWEEN ? AND ?")

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

	result := make([]currencyFetcher.CurrencyWithID, 0, perPage)
	for rows.Next() {
		var currency string
		var createdAt string
		currencyWithId := currencyFetcher.CurrencyWithID{}
		if err := rows.Scan(&currencyWithId.ID, currency, &currencyWithId.Provider, &currencyWithId.Rate, &createdAt); err != nil {
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
	_, err := m.db.ExecContext(m.ctx, fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s(
		id binary(36) PRIMARY KEY,
		fetchers varchar(20) NOT NULL,
		provider varchar(30) NOT NULL,
		rate float(8,4) NOT NULL,
		created_at timestamp DEFAULT CURRENT_TIMESTAMP 
	);`, m.tableName))

	if err != nil {
		return err
	}

	_, err = m.db.ExecContext(m.ctx, fmt.Sprintf(`CREATE INDEX IF NOT EXISTS search_index ON %s(fetchers, provider, created_at);`, m.tableName))
	return err
}

func (mysqlStorage) GetStorageProviderName() string {
	return MySQLStorageProviderName
}

func (m *mysqlStorage) Close() error {
	if err := m.db.Close(); err != nil {
		return fmt.Errorf("disconnecting from SQL Database failed: %v", err)
	}
	return nil
}

func (m mysqlStorage) Drop() error {
	_, err := m.db.ExecContext(m.ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", m.tableName))
	return err
}

func NewMySQLStorage(c MySQLConfig) (currencyFetcher.Storage, error) {
	db, err := sql.Open("mysql", c.ConnectionString)

	if err != nil {
		return nil, fmt.Errorf("Error while connecting to MySQL database: %v", err)
	}

	storage := &mysqlStorage{
		idGenerator: c.IDGenerator,
		ctx:         c.Cxt,
		db:          db,
		tableName:   c.TableName,
	}

	if c.Migrate {
		if err := storage.Migrate(); err != nil {
			return nil, fmt.Errorf("error while migrating mysql database: %v", err)
		}
	}

	return storage, nil
}
