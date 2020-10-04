package currency_fetcher

type Service interface {
	Save(currenciesToFetch []string) (map[string][]CurrencyWithId, error)
}
