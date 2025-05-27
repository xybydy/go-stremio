package stremio

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	netpprof "net/http/pprof"
	"net/url"
	"os"
	"os/signal"
	"reflect"
	"runtime/pprof"
	"strconv"
	"syscall"

	"github.com/VictoriaMetrics/metrics"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/xybydy/go-stremio/pkg/cinemeta"
	"github.com/xybydy/go-stremio/types"
	"go.uber.org/zap"
)

// ManifestCallback is the callback for manifest requests, so mostly addon installations.
// You can use the callback for two things:
//  1. To *prevent* users from installing your addon in Stremio.
//     The userData parameter depends on whether you called `RegisterUserData()` before:
//     If not, a simple string will be passed. It's empty if the user didn't provide user data.
//     If yes, a pointer to an object you registered will be passed. It's nil if the user didn't provide user data.
//     Return an HTTP status code >= 400 to stop further processing and let the addon return that exact status code.
//     Any status code < 400 will lead to the manifest being returned with a 200 OK status code in the response.
//  2. To *alter* the manifest before it's returned.
//     This can be useful for example if you want to return some catalogs depending on the userData.
//     Note that the manifest is only returned if the first return value is < 400 (see point 1.).
type ManifestCallback func(ctx context.Context, manifest *types.Manifest, userData any) int

// CatalogHandler is the callback for catalog requests for a specific type (like "movie").
// The id parameter is the catalog ID that you specified yourself in the CatalogItem objects in the Manifest.
// The userData parameter depends on whether you called `RegisterUserData()` before:
// If not, a simple string will be passed. It's empty if the user didn't provide user data.
// If yes, a pointer to an object you registered will be passed. It's nil if the user didn't provide user data.
// Extra Parameters is optional
// search - set in the extra object; string to search for in the catalog
// genre - set in the extra object; a string to filter the feed or search results by genres
// skip - set in the extra object; used for catalog pagination, refers to the number of items skipped from the beginning of the catalog;
// the standard page size in Stremio is 100, so the skip value will be a multiple of 100; if you return less than 100 items,
// Stremio will consider this to be the end of the catalog.
type CatalogHandler func(ctx context.Context, id string, extra url.Values, userData any) ([]types.MetaPreviewItem, error)

// StreamHandler is the callback for stream requests for a specific type (like "movie").
// The context parameter contains a meta object under the key "meta" if PutMetaInContext was set to true in the addon options.
// The id parameter can be for example an IMDb ID if your addon handles the "movie" type.
// The userData parameter depends on whether you called `RegisterUserData()` before:
// If not, a simple string will be passed. It's empty if the user didn't provide user data.
// If yes, a pointer to an object you registered will be passed. It's nil if the user didn't provide user data.
type StreamHandler func(ctx context.Context, id string, userData any) ([]types.StreamItem, error)

// MetaHandler is the callback for metadata requests for a specific type (like "movie").
// The context parameter contains a meta object under the key "meta" if PutMetaInContext was set to true in the addon options.
// The id parameter can be for example an IMDb ID if your addon handles the "movie" type.
// The userData parameter depends on whether you called `RegisterUserData()` before:
// If not, a simple string will be passed. It's empty if the user didn't provide user data.
// If yes, a pointer to an object you registered will be passed. It's nil if the user didn't provide user data.
// The meta object is a pointer to a MetaItem object, which contains the metadata for the media.
// The metadata is returned in the form of a MetaItem object, which contains the metadata for the media.
type MetaHandler func(ctx context.Context, id string, userData any) (types.MetaItem, error)

// SubtitleHandler is the callback for subtitle requests for a specific type (like "movie").
// The context parameter contains a meta object under the key "meta" if PutMetaInContext was set to true in the addon options.
// The id parameter can be for example an "videoId" if your addon handles the "movie" type.
// The userData parameter depends on whether you called `RegisterUserData()` before:
// If not, a simple string will be passed. It's empty if the user didn't provide user data.
// If yes, a pointer to an object you registered will be passed. It's nil if the user didn't provide user data.
// Extra Parameters is optional
// videoHash - string OpenSubtitles file hash for the video
// videoSize - size of the video file in bytes
// filename - filename of the video file
// It returns array of SubtitleItem objects, which contain the subtitle URL and language.
type SubtitleHandler func(ctx context.Context, id string, extra url.Values, userData any) ([]types.SubtitleItem, error)

// MetaFetcher returns metadata for movies and TV shows.
// It's used when you configure that the media name should be logged or that metadata should be put into the context.
type MetaFetcher interface {
	GetMovie(ctx context.Context, imdbID string) (types.MetaItem, error)
	GetSeries(ctx context.Context, imdbID string, season int, episode int) (types.MetaItem, error)
}

// Addon represents a remote addon.
// You can create one with NewAddon() and then run it with Run().
type Addon struct {
	manifest          types.Manifest
	catalogHandlers   map[string]CatalogHandler
	streamHandlers    map[string]StreamHandler
	metaHandlers      map[string]MetaHandler
	subtitleHandlers  map[string]SubtitleHandler
	opts              Options
	logger            *zap.Logger
	customMiddlewares []customMiddleware
	customEndpoints   []customEndpoint
	manifestCallback  ManifestCallback
	userDataType      reflect.Type
	metaClient        MetaFetcher
}

// NewAddon creates a new Addon object that can be started with Run().
// A proper manifest must be supplied, but manifestCallback and all but one handler can be nil in case you only want to handle specific requests and opts can be the zero value of Options.
func NewAddon(manifest types.Manifest, catalogHandlers map[string]CatalogHandler, streamHandlers map[string]StreamHandler, metaHandlers map[string]MetaHandler, subtitleHandlers map[string]SubtitleHandler, opts Options) (*Addon, error) {
	// Precondition checks
	switch {
	case manifest.ID == "" || manifest.Name == "" || manifest.Description == "" || manifest.Version == "":
		return nil, errors.New("an empty manifest was passed")
	case catalogHandlers == nil && streamHandlers == nil && metaHandlers == nil && subtitleHandlers == nil:
		return nil, errors.New("no handler was passed")
	case (opts.CachePublicCatalogs && opts.CacheAgeCatalogs == 0) ||
		(opts.CachePublicStreams && opts.CacheAgeStreams == 0) ||
		(opts.CachePublicMeta && opts.CacheAgeMeta == 0):
		return nil, errors.New("enabling public caching only makes sense when also setting a cache age")
	case (opts.StaleRevalidateCatalogs != 0 && opts.CacheAgeCatalogs == 0) ||
		(opts.StaleRevalidateStreams != 0 && opts.CacheAgeStreams == 0):
		return nil, errors.New("to enable stale-while-revalidate you must also set cache age")
	case (opts.StaleErrorCatalogs != 0 && opts.CacheAgeCatalogs == 0) ||
		(opts.StaleErrorStreams != 0 && opts.CacheAgeStreams == 0):
		return nil, errors.New("to enable stale-if-error you must also set cache age")
	case (opts.HandleEtagCatalogs && opts.CacheAgeCatalogs == 0) ||
		(opts.HandleEtagStreams && opts.CacheAgeStreams == 0):
		return nil, errors.New(`ETag handling only makes sense when also setting a cache age`)
	case opts.DisableRequestLogging && (opts.LogIPs || opts.LogUserAgent):
		return nil, errors.New("enabling IP or user agent logging doesn't make sense when disabling request logging")
	case opts.Logger != nil && opts.LoggingLevel != "":
		return nil, errors.New("setting a logging level in the options doesn't make sense when you already set a custom logger")
	case opts.DisableRequestLogging && opts.LogMediaName:
		return nil, errors.New("enabling media name logging doesn't make sense when disabling request logging")
	case opts.MetaClient != nil && !opts.LogMediaName && !opts.PutMetaInContext:
		return nil, errors.New("setting a meta client when neither logging the media name nor putting it in the context doesn't make sense")
	case opts.MetaClient != nil && opts.MetaTimeout != 0:
		return nil, errors.New("setting a MetaClient timeout doesn't make sense when you already set a meta client")
	case manifest.BehaviorHints.ConfigurationRequired && !manifest.BehaviorHints.Configurable:
		return nil, errors.New("requiring a configuration only makes sense when also making the addon configurable")
	case opts.ConfigureHTMLfs != nil && !manifest.BehaviorHints.Configurable:
		return nil, errors.New("setting a ConfigureHTMLfs only makes sense when also making the addon configurable")
	}

	// Set default values
	if opts.BindAddr == "" {
		opts.BindAddr = DefaultOptions.BindAddr
	}
	if opts.Port == 0 {
		opts.Port = DefaultOptions.Port
	}
	if opts.LoggingLevel == "" {
		opts.LoggingLevel = DefaultOptions.LoggingLevel
	}
	if opts.LogEncoding == "" {
		opts.LogEncoding = DefaultOptions.LogEncoding
	}
	if opts.MetaTimeout == 0 {
		opts.MetaTimeout = DefaultOptions.MetaTimeout
	}

	// Configure logger if no custom one is set
	if opts.Logger == nil {
		var err error
		if opts.Logger, err = NewLogger(opts.LoggingLevel, opts.LogEncoding); err != nil {
			return nil, fmt.Errorf("couldn't create new logger: %w", err)
		}
	}
	// Configure Cinemeta client if no custom MetaFetcher is set
	if opts.MetaClient == nil && (opts.LogMediaName || opts.PutMetaInContext) {
		cinemetaCache := cinemeta.NewInMemoryCache()
		cinemetaOpts := cinemeta.ClientOptions{
			Timeout: opts.MetaTimeout,
		}
		opts.MetaClient = cinemeta.NewClient(cinemetaOpts, cinemetaCache, opts.Logger)
	}

	// Create and return addon
	return &Addon{
		manifest:         manifest,
		catalogHandlers:  catalogHandlers,
		streamHandlers:   streamHandlers,
		metaHandlers:     metaHandlers,
		subtitleHandlers: subtitleHandlers,
		opts:             opts,
		logger:           opts.Logger,
		metaClient:       opts.MetaClient,
	}, nil
}

// RegisterUserData registers the type of userData, so the addon can automatically unmarshal user data into an object of this type
// and pass the object into the manifest callback or catalog and stream handlers.
func (a *Addon) RegisterUserData(userDataObject any) {
	t := reflect.TypeOf(userDataObject)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	a.userDataType = t
}

// DecodeUserData decodes the request's user data and returns the result.
// It's useful when you add custom endpoints to the addon that don't have a userData parameter
// like the ManifestCallback, CatalogHandler and StreamHandler have.
// The param value must match the URL parameter you used when creating the custom endpoint,
// for example when using `AddEndpoint("GET", "/:userData/ping", customEndpoint)` you must pass "userData".
func (a *Addon) DecodeUserData(param string, c fiber.Ctx) (any, error) {
	data := c.Params(param, "")
	return decodeUserData(data, a.userDataType, a.logger, a.opts.UserDataIsBase64)
}

// AddMiddleware appends a custom middleware to the chain of existing middlewares.
// Set path to an empty string or "/" to let the middleware apply to all routes.
// Don't forget to call c.Next() on the Fiber context!
func (a *Addon) AddMiddleware(path string, middleware fiber.Handler) {
	customMW := customMiddleware{
		path: path,
		mw:   middleware,
	}
	a.customMiddlewares = append(a.customMiddlewares, customMW)
}

// AddEndpoint adds a custom endpoint (a route and its handler).
// If you want to be able to access custom user data, you can use a path like this:
// "/:userData/foo" and then either deal with the data yourself
// by using `c.Params("userData", "")` in the handler,
// or use the convenience method `DecodeUserData("userData", c)`.
func (a *Addon) AddEndpoint(method, path string, handler fiber.Handler) {
	customEndpoint := customEndpoint{
		method:  method,
		path:    path,
		handler: handler,
	}
	a.customEndpoints = append(a.customEndpoints, customEndpoint)
}

// SetManifestCallback sets the manifest callback.
func (a *Addon) SetManifestCallback(callback ManifestCallback) {
	a.manifestCallback = callback
}

// Run starts the remote addon. It sets up an HTTP server that handles requests to "/manifest.json" etc. and gracefully handles shutdowns.
// The call is *blocking*, so use the stoppingChan param if you want to be notified when the addon is about to shut down
// because of a system signal like Ctrl+C or `docker stop`. It should be a buffered channel with a capacity of 1.
func (a *Addon) Run(stoppingChan chan bool, fiberConf *fiber.Config) {
	logger := a.logger

	defer func() {
		err := logger.Sync()
		if err != nil {
			logger.Error("Failed to sync logger", zap.Error(err))
		}
	}()

	// Make sure the passed channel is buffered, so we can send a message before shutting down and not be blocked by the channel.
	if stoppingChan != nil && cap(stoppingChan) < 1 {
		logger.Fatal("The passed stopping channel isn't buffered")
	}

	if fiberConf == nil {
		fiberConf = &fiber.Config{
			ErrorHandler: func(c fiber.Ctx, err error) error {
				code := fiber.StatusInternalServerError
				var e *fiber.Error
				if errors.As(err, &e) {
					code = e.Code
					logger.Error("Fiber's error handler was called", zap.Error(e), zap.String("url", c.OriginalURL()))
				}
				c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
				return c.Status(code).SendString("An internal server error occurred")
			},
			BodyLimit: 0,
			//ReadTimeout: 5 * time.Second,
			// Docker stop only gives us 10s. We want to close all connections before that.
			//WriteTimeout: 9 * time.Second,
			//IdleTimeout:  9 * time.Second,
		}
	}

	// Fiber app

	logger.Info("Setting up server...")
	app := fiber.New(*fiberConf)

	// Middlewares

	app.Use(recover.New())
	if !a.opts.DisableRequestLogging {
		app.Use(createLoggingMiddleware(logger, a.opts.LogIPs, a.opts.LogUserAgent, a.opts.LogMediaName))
	}
	if a.opts.Metrics {
		app.Use(createMetricsMiddleware())
	}
	app.Use(corsMiddleware()) // Stremio doesn't show stream responses when no CORS middleware is used!
	// Filter some requests (like for requests without user data when the addon requires configuration, or for missing type or id URL parameters) and put some request info in the context
	addRouteMatcherMiddleware(app, a.manifest.BehaviorHints.ConfigurationRequired, a.opts.StreamIDregex, logger)
	metaMw := createMetaMiddleware(a.metaClient, a.opts.PutMetaInContext, a.opts.LogMediaName, logger)
	// Meta middleware only works for stream requests.
	if !a.manifest.BehaviorHints.ConfigurationRequired {
		app.Use("/stream/:type/:id.json", metaMw)
	}
	app.Use("/:userData/stream/:type/:id.json", metaMw)
	// Custom middlewares
	for _, customMW := range a.customMiddlewares {
		app.Use(customMW.path, customMW.mw)
	}

	// Extra endpoints

	app.Get("/health", createHealthHandler(logger))
	// Optional profiling
	if a.opts.Profiling {
		group := app.Group("/debug/pprof")

		group.Get("/", func(c fiber.Ctx) error {
			c.Set(fiber.HeaderContentType, fiber.MIMETextHTML)
			return adaptor.HTTPHandlerFunc(netpprof.Index)(c)
		})
		for _, p := range pprof.Profiles() {
			group.Get("/"+p.Name(), adaptor.HTTPHandler(netpprof.Handler(p.Name())))
		}
		group.Get("/cmdline", adaptor.HTTPHandlerFunc(netpprof.Cmdline))
		group.Get("/profile", adaptor.HTTPHandlerFunc(netpprof.Profile))
		group.Get("/trace", adaptor.HTTPHandlerFunc(netpprof.Trace))
	}
	// Optional metrics
	if a.opts.Metrics {
		app.Get("/metrics", adaptor.HTTPHandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			metrics.WritePrometheus(w, true)
		}))
	}

	// Stremio endpoints

	// In Fiber optional parameters don't work at the beginning of the URL, so we have to register two routes each
	manifestHandler := createManifestHandler(a.manifest, logger, a.manifestCallback, a.userDataType, a.opts.UserDataIsBase64)
	// We always register this route, because even if BehaviorHints.ConfigurationRequired is true, this endpoint is required for the addon to be listed in Stremio's community addons.
	app.Get("/manifest.json", manifestHandler)
	app.Get("/:userData/manifest.json", manifestHandler)
	if a.catalogHandlers != nil {
		catalogHandler := createCatalogHandler(a.catalogHandlers, a.opts.CacheAgeCatalogs, a.opts.StaleRevalidateCatalogs, a.opts.StaleErrorCatalogs, a.opts.CachePublicCatalogs, a.opts.HandleEtagCatalogs, logger, a.userDataType, a.opts.UserDataIsBase64)
		if !a.manifest.BehaviorHints.ConfigurationRequired {
			app.Get("/catalog/:type/:id.json", catalogHandler)
			app.Get("/catalog/:type/:id/:extras", catalogHandler)
		}
		// We always register this route, because we don't know if the addon developer wants to use user data or not, as BehaviorHints.Configurable only indicates the configurability *via Stremio*
		app.Get("/:userData/catalog/:type/:id.json", catalogHandler)
		app.Get("/:userData/catalog/:type/:id/:extras", catalogHandler)
	}

	if a.streamHandlers != nil {
		streamHandler := createStreamHandler(a.streamHandlers, a.opts.CacheAgeStreams, a.opts.StaleRevalidateStreams, a.opts.StaleErrorStreams, a.opts.CachePublicStreams, a.opts.HandleEtagStreams, logger, a.userDataType, a.opts.UserDataIsBase64)
		if !a.manifest.BehaviorHints.ConfigurationRequired {
			app.Get("/stream/:type/:id.json", streamHandler)
		}
		// We always register this route, because we don't know if the addon developer wants to use user data or not, as BehaviorHints.Configurable only indicates the configurability *via Stremio*
		app.Get("/:userData/stream/:type/:id.json", streamHandler)
	}

	if a.metaHandlers != nil {
		metaHandler := createMetaHandler(a.metaHandlers, a.opts.CacheAgeMeta, a.opts.StaleRevalidateMeta, a.opts.StaleErrorMeta, a.opts.CachePublicMeta, a.opts.HandleEtagMeta, logger, a.userDataType, a.opts.UserDataIsBase64)
		if !a.manifest.BehaviorHints.ConfigurationRequired {
			app.Get("/meta/:type/:id.json", metaHandler)
		}
		// We always register this route, because we don't know if the addon developer wants to use user data or not, as BehaviorHints.Configurable only indicates the configurability *via Stremio*
		app.Get("/:userData/meta/:type/:id.json", metaHandler)
	}

	if a.subtitleHandlers != nil {
		subtitleHandler := createSubtitleHandler(a.subtitleHandlers, a.opts.CacheAgeStreams, a.opts.StaleRevalidateStreams, a.opts.StaleErrorStreams, a.opts.CachePublicStreams, a.opts.HandleEtagStreams, logger, a.userDataType, a.opts.UserDataIsBase64)
		if !a.manifest.BehaviorHints.ConfigurationRequired {
			app.Get("/subtitles/:type/:id.json", subtitleHandler)
		}
		app.Get("/:userData/subtitles/:type/:id.json", subtitleHandler)
	}

	if a.opts.ConfigureHTMLfs != nil {
		fsConfig := static.Config{
			FS: a.opts.ConfigureHTMLfs,
		}
		app.Use("/configure", static.New("", fsConfig))
		//fmt.Printf("%s", a.opts.ConfigureHTMLfs)
		// When a Stremio user has the addon already installed and configures it again, this endpoint is called,
		// theoretically enabling the addon to deliver a website with the configuration fields populated with the currently configured values.
		// The Fiber filesystem middleware currently doesn't work with parameters in the route (see https://github.com/gofiber/fiber/issues/834),
		// so we'll just redirect to the original one, as we don't use the existing configuration anyway.
		// FIXME: this is a workaround to fill form, might be a better way for fiber.static. On this scenario data filling must be on client-side.
		app.Get("/:userData/configure*", func(c fiber.Ctx) error {
			c.Set("Location", c.BaseURL()+"/configure?data="+c.Params("userData"))
			return c.SendStatus(fiber.StatusMovedPermanently)
		})
	}

	// Additional endpoints

	// Root redirects to website
	if a.opts.RedirectURL != "" {
		app.Get("/", createRootHandler(a.opts.RedirectURL, logger))
	}

	// Custom endpoints
	for _, customEndpoint := range a.customEndpoints {
		app.Add([]string{customEndpoint.method}, customEndpoint.path, customEndpoint.handler)
	}

	logger.Info("Finished setting up server")

	stopping := false
	stoppingPtr := &stopping

	addr := a.opts.BindAddr + ":" + strconv.Itoa(a.opts.Port)
	logger.Info("Starting server", zap.String("address", addr))
	go func() {
		if err := app.Listen(addr, fiber.ListenConfig{DisableStartupMessage: true}); err != nil {
			if !*stoppingPtr {
				logger.Fatal("Couldn't start server", zap.Error(err))
			} else {
				logger.Fatal("Error in srv.ListenAndServe() during server shutdown (probably context deadline expired before the server could shutdown cleanly)", zap.Error(err))
			}
		}
	}()

	// Graceful shutdown

	c := make(chan os.Signal, 1)
	// Accept SIGINT (Ctrl+C) and SIGTERM (`docker stop`)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	sig := <-c
	logger.Info("Received signal, shutting down server...", zap.Stringer("signal", sig))
	*stoppingPtr = true
	if stoppingChan != nil {
		stoppingChan <- true
	}
	// Graceful shutdown, waiting for all current requests to finish without accepting new ones.
	if err := app.Shutdown(); err != nil {
		logger.Fatal("Error shutting down server", zap.Error(err))
	}
	logger.Info("Finished shutting down server")
}
