package currency

type (
	Fetcher interface {
		Fetch(currenciesToFetch []string) ([]Currency, error)
	}
)
