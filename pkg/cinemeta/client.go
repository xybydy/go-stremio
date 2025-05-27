package cinemeta

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/xybydy/go-stremio/types"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ClientOptions are the options for the Cinemeta client.
type ClientOptions struct {
	// The base URL for Cinemeta.
	// Default "https://v3-cinemeta.strem.io".
	BaseURL string
	// Timeout for requests.
	// A more customizable cancellation can be achieved with the context,
	// but it can never be *longer* than this timeout.
	// Default 2 seconds.
	Timeout time.Duration
	// Max age of items in the cache.
	// Default 30 days.
	TTL time.Duration
}

// DefaultClientOpts is an options object with sensible defaults.
var DefaultClientOpts = ClientOptions{
	BaseURL: "https://v3-cinemeta.strem.io",
	// HTTP client timeout
	Timeout: 2 * time.Second,
	TTL:     30 * 24 * time.Hour, // 30 days
}

// Client is the Cinemeta client.
type Client struct {
	baseURL    string
	httpClient *http.Client
	cache      Cache
	logger     *zap.Logger
	ttl        time.Duration
}

// NewClient creates a new Cinemeta client.
func NewClient(opts ClientOptions, cache Cache, logger *zap.Logger) *Client {
	// Set defaults if necessary.
	// A TTL of 0 is allowed.
	if opts.BaseURL == "" {
		opts.BaseURL = DefaultClientOpts.BaseURL
	}
	if opts.Timeout == 0 {
		opts.Timeout = DefaultClientOpts.Timeout
	}
	if opts.TTL == 0 {
		opts.TTL = DefaultClientOpts.TTL
	}

	return &Client{
		baseURL: opts.BaseURL,
		httpClient: &http.Client{
			Timeout: opts.Timeout,
		},
		cache:  cache,
		logger: logger,
		ttl:    opts.TTL,
	}
}

// GetMovie returns the meta object either from the cache or from Cinemeta.
// It automatically fills the cache with new Cinemeta responses.
// The context can control the lifetime of the request, and if for example the timeout is shorter
// than the HTTP client's configured timeout then it takes precedence.
// If no timeout is set in the context, the HTTP client's timeout takes effect.
func (c *Client) GetMovie(ctx context.Context, imdbID string) (types.MetaItem, error) {
	return c.getMeta(ctx, movie, imdbID, 0, 0)
}

// GetSeries returns the meta object either from the cache or from Cinemeta.
// It automatically fills the cache with new Cinemeta responses.
// The context can control the lifetime of the request, and if for example the timeout is shorter
// than the HTTP client's configured timeout then it takes precedence.
// If no timeout is set in the context, the HTTP client's timeout takes effect.
func (c *Client) GetSeries(ctx context.Context, imdbID string, season int, episode int) (types.MetaItem, error) {
	return c.getMeta(ctx, tvShow, imdbID, season, episode)
}

// GetMeta returns the meta object either from the cache or from Cinemeta.
// It automatically fills the cache with new Cinemeta responses.
// The context can control the lifetime of the request, and if for example the timeout is shorter
// than the HTTP client's configured timeout then it takes precedence.
// If no timeout is set in the context, the HTTP client's timeout takes effect.
func (c *Client) getMeta(ctx context.Context, t mediaType, imdbID string, season int, episode int) (types.MetaItem, error) {
	var zapFieldIMDbID zapcore.Field
	switch t {
	case movie:
		zapFieldIMDbID = zap.String("imdbID", imdbID)
	case tvShow:
		zapFieldIMDbID = zap.String("imdbID", fmt.Sprintf("%v:%v:%v", imdbID, season, episode))
	}

	// Check cache first
	meta, created, found, err := c.cache.Get(imdbID)
	if err != nil {
		c.logger.Error("Couldn't decode meta", zap.Error(err), zapFieldIMDbID)
	} else if !found {
		c.logger.Debug("Meta not found in cache", zapFieldIMDbID)
	} else if time.Since(created) > c.ttl {
		expiredSince := time.Since(created.Add(c.ttl))
		c.logger.Debug("Hit cache for meta, but item is expired", zap.Duration("expiredSince", expiredSince), zapFieldIMDbID)
	} else {
		c.logger.Debug("Hit cache for meta, returning result")
		convMeta := meta.(types.MetaItem)
		return convMeta, nil
	}

	var reqURL string
	switch t {
	case movie:
		reqURL = c.baseURL + "/meta/movie/" + imdbID + ".json"
	case tvShow:
		reqURL = c.baseURL + "/meta/series/" + imdbID + ".json"
	}

	// Then check web service
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return types.MetaItem{}, fmt.Errorf("couldn't create request: %w", err)
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		return types.MetaItem{}, fmt.Errorf("couldn't GET %v: %w", reqURL, err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return types.MetaItem{}, fmt.Errorf("bad GET response: %v", res.StatusCode)
	}
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return types.MetaItem{}, fmt.Errorf("couldn't read response body: %w", err)
	}
	cineRes := types.MetaItem{}
	if err := json.Unmarshal(resBody, &cineRes); err != nil {
		return types.MetaItem{}, fmt.Errorf("couldn't unmarshal response body: %w", err)
	}
	if cineRes.Name == "" {
		return types.MetaItem{}, fmt.Errorf("couldn't find %v name in Cinemeta response", t)
	}

	// Fill cache
	if err = c.cache.Set(imdbID, types.MetaItem{}); err != nil {
		c.logger.Error("Couldn't cache meta", zap.Error(err), zap.String("meta", fmt.Sprintf("%+v", cineRes)), zapFieldIMDbID)
	}

	return cineRes, nil
}
