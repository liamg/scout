package wordlist

type Wordlist interface {
	Next() (string, error)
}
