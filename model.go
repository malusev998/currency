package currency

import "time"

type (
	Currency struct {
		Rate      float32   `json:"rate,omitempty"`
		CreatedAt time.Time `json:"created_at,omitempty"`
		Provider  Provider  `json:"provider,omitempty"`
		From      string    `json:"from,omitempty"`
		To        string    `json:"to,omitempty"`
	}

	CurrencyWithID struct {
		ID interface{} `json:"id"`
		Currency
	}
)
