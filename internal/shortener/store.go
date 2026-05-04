package shortener

type Store interface {
	NextID() uint64
	Save(code, longURL string)
	Get(code string) (string, bool)
}
