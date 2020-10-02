package currency_fetcher

type Fetcher interface {
	Fetch() []Currency
}
