package currency_fetcher

import "time"

type Service interface {
	Save(currenciesToFetch []string) (map[string][]CurrencyWithId, error)
}


type Conversion interface {
	Convert(from, to, provider string, value float32 ,date time.Time) (float32, error)
}