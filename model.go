package currency_fetcher

import "time"

type Currency struct {
	From     string  `json:"from,omitempty"`
	To       string  `json:"to,omitempty"`
	Provider string  `json:"provider,omitempty"`
	Rate     float32 `json:"rate,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

type CurrencyWithId struct {
	Currency
	Id interface{} `json:"id"`
}
