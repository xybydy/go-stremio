package types

// StreamItem represents a stream for a MetaItem.
// See https://github.com/Stremio/stremio-addon-sdk/blob/f6f1f2a8b627b9d4f2c62b003b251d98adadbebe/docs/api/responses/stream.md
type StreamItem struct {
	// One of the following is required
	URL         string `json:"url,omitempty"` // URL
	YoutubeID   string `json:"ytId,omitempty"`
	InfoHash    string `json:"infoHash,omitempty"`
	FileIndex   uint8  `json:"fileIdx,omitempty"`     // Only when using InfoHash
	ExternalURL string `json:"externalUrl,omitempty"` // URL

	// Optional
	Name          string              `json:"name,omitempty"`
	Title         string              `json:"title,omitempty"` // Usually used for stream quality
	Description   string              `json:"description,omitempty"`
	Subtitles     []SubtitleItem      `json:"subtitles,omitempty"`
	Sources       []string            `json:"sources,omitempty"`
	BehaviorHints StreamBehaviorHints `json:"behaviorHints,omitempty"`
}

type StreamBehaviorHints struct {
	CountryWhitelist []string `json:"countryWhitelist,omitempty"` // array of ISO 3166-1 alpha-3 country codes in lowercase in which the stream is accessible
	NotWebReady      bool     `json:"notWebReady,omitempty"`
	BingeGroup       any      `json:"bingeGroup,omitempty"`
	ProxyHeaders     string   `json:"proxyHeaders,omitempty"`
	VideoHash        string   `json:"videoHash,omitempty"`
	VideoSize        int      `json:"videoSize,omitempty"`
	Filename         string   `json:"filename,omitempty"`
}
