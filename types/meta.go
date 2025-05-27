package types

// MetaPreviewItem represents a meta preview item and is meant to be used within catalog responses.
// See https://github.com/Stremio/stremio-addon-sdk/blob/f6f1f2a8b627b9d4f2c62b003b251d98adadbebe/docs/api/responses/meta.md#meta-preview-object
type MetaPreviewItem struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Name   string `json:"name"`
	Poster string `json:"poster"` // URL

	// Optional
	PosterShape string `json:"posterShape,omitempty"`

	// Optional, used for the "Discover" page sidebar
	Genres      []string       `json:"genres,omitempty"` // Will be replaced by Links at some point
	IMDbRating  string         `json:"imdbRating,omitempty"`
	ReleaseInfo string         `json:"releaseInfo,omitempty"` // E.g. "2000" for movies and "2000-2014" or "2000-" for TV shows
	Director    []string       `json:"director,omitempty"`    // Will be replaced by Links at some point
	Cast        []string       `json:"cast,omitempty"`        // Will be replaced by Links at some point
	Links       []MetaLinkItem `json:"links,omitempty"`       // For genres, director, cast and potentially more. Not fully supported by Stremio yet!
	Description string         `json:"description,omitempty"`
	Trailers    []StreamItem   `json:"trailers,omitempty"`
}

// MetaItem represents a meta item and is meant to be used when info for a specific item was requested.
// See https://github.com/Stremio/stremio-addon-sdk/blob/f6f1f2a8b627b9d4f2c62b003b251d98adadbebe/docs/api/responses/meta.md
type MetaItem struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`

	// Optional
	Genres             []string          `json:"genres,omitempty"` // Will be replaced by Links at some point
	Poster             string            `json:"poster,omitempty"` // URL
	PosterShape        string            `json:"posterShape,omitempty"`
	Background         string            `json:"background,omitempty"` // URL
	Logo               string            `json:"logo,omitempty"`       // URL
	Description        string            `json:"description,omitempty"`
	ReleaseInfo        string            `json:"releaseInfo,omitempty"` // E.g. "2000" for movies and "2000-2014" or "2000-" for TV shows
	Director           []string          `json:"director,omitempty"`    // Will be replaced by Links at some point
	Cast               []string          `json:"cast,omitempty"`        // Will be replaced by Links at some point
	IMDbRating         string            `json:"imdbRating,omitempty"`
	Released           string            `json:"released,omitempty"` // Must be ISO 8601, e.g. "2010-12-06T05:00:00.000Z"
	Trailers           []StreamItem      `json:"trailers,omitempty"`
	Links              []MetaLinkItem    `json:"links,omitempty"` // For genres, director, cast and potentially more. Not fully supported by Stremio yet!
	Videos             []VideoItem       `json:"videos,omitempty"`
	Runtime            string            `json:"runtime,omitempty"`
	Language           string            `json:"language,omitempty"`
	Country            string            `json:"country,omitempty"`
	Awards             string            `json:"awards,omitempty"`
	Website            string            `json:"website,omitempty"` // URL
	BehaviorHints      MetaBehaviorHints `json:"behaviorHints,omitempty"`
	ContainerExtension string            `json:"-"`
}

type MetaBehaviorHints struct {
	DefaultVideoID string `json:"defaultVideoId,omitempty"` // The ID of the default video to play when the user clicks on the item in the catalog
}

// MetaLinkItem links to a page within Stremio.
// It will at some point replace the usage of `genres`, `director` and `cast`.
// Note: It's not fully supported by Stremio yet (not fully on PC and not at all on Android)!
type MetaLinkItem struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	URL      string `json:"url"` //  // URL. Can be "Meta Links" (see https://github.com/Stremio/stremio-addon-sdk/blob/f6f1f2a8b627b9d4f2c62b003b251d98adadbebe/docs/api/responses/meta.links.md)
}

type VideoItem struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Released string `json:"released,omitempty"` // Must be ISO 8601, e.g. "2010-12-06T05:00:00.000Z"

	// Optional
	Thumbnail string       `json:"thumbnail,omitempty"` // URL
	Streams   []StreamItem `json:"streams,omitempty"`
	Available bool         `json:"available,omitempty"`
	Episode   int          `json:"episode,omitempty"`
	Season    int          `json:"season,omitempty"`
	Trailer   string       `json:"trailer,omitempty"` // Youtube ID
	Overview  string       `json:"overview,omitempty"`
}
