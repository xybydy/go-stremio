package types

type SubtitleItem struct {
	ID   string `json:"id,omitempty"`
	URL  string `json:"url,omitempty"`
	Lang string `json:"lang,omitempty"` // language code for the subtitle, if a valid ISO 639-2 code is not sent, the text of this value will be used instead
}
