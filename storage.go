package currency_fetcher

type Storage interface {
	Store(Currency) (CurrencyWithId, error)
	Get(string, string) CurrencyWithId
}
