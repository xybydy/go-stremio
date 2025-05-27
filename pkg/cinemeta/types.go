package cinemeta

type mediaType int

const (
	movie mediaType = iota + 1
	tvShow
)

func (mt mediaType) String() string {
	return [...]string{"movie", "TV show"}[mt-1]
}
