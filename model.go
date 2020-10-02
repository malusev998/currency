package currency_fetcher


type Currency struct {
	From string
	To string
	Provider string
	Rate float32
}


type CurrencyWithId struct {
	Currency
	Id interface{}
}