package currency

import "time"

type (
	Currency struct {
		From      string    `json:"from,omitempty"`
		To        string    `json:"to,omitempty"`
		Provider  Provider  `json:"provider,omitempty"`
		Rate      float32   `json:"rate,omitempty"`
		CreatedAt time.Time `json:"created_at,omitempty"`
	}

	CurrencyWithID struct {
		Currency
		ID interface{} `json:"id"`
	}
)
