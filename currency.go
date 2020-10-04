package currency_fetcher

type Fetcher interface {
	Fetch(currenciesToFetch []string) ([]Currency, error)
}
