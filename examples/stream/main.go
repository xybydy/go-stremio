package main

import (
	"context"
	"time"

	"github.com/xybydy/go-stremio"
	`github.com/xybydy/go-stremio/types`
)

var (
	version = "0.1.0"

	manifest = types.Manifest{
		ID:          "com.example.blender-streams",
		Name:        "Blender movie streams",
		Description: "Stream addon for free movies that were made with Blender",
		Version:     version,

		ResourceItems: []types.ResourceItem{
			{
				Name:  "stream",
				Types: []string{"movie"},
			},
		},
		Types: []string{"movie"},
		// An empty slice is required for serializing to a JSON that Stremio expects
		Catalogs: []types.CatalogItem{},

		IDprefixes: []string{"tt"},
	}
)

func main() {
	// Let the movieHandler handle the "movie" type
	streamHandlers := map[string]stremio.StreamHandler{"movie": movieHandler}

	// We want clients and proxies to cache the response for 24 hours
	// and upon request with the same hash we only return a 304 Not Modified.
	options := stremio.Options{
		CacheAgeStreams:    24 * time.Hour,
		CachePublicStreams: true,
		HandleEtagStreams:  true,
	}

	addon, err := stremio.NewAddon(manifest, nil, streamHandlers, nil, nil, options)
	if err != nil {
		panic(err)
	}

	addon.Run(nil, nil)
}

func movieHandler(ctx context.Context, id string, userData interface{}) ([]types.StreamItem, error) {
	// We only serve Big Buck Bunny and Sintel
	if id == "tt1254207" {
		return []types.StreamItem{
			// Torrent stream
			{
				InfoHash: "dd8255ecdc7ca55fb0bbf81323d87062db1f6d1c",
				// Stremio recommends to set the quality as title, as the streams
				// are shown for a specific movie so the user knows the title.
				Title:     "1080p (torrent)",
				FileIndex: 1,
			},
			// HTTP stream
			{
				URL:   "https://ftp.halifax.rwth-aachen.de/blender/demo/movies/BBB/bbb_sunflower_1080p_30fps_normal.mp4",
				Title: "1080p (HTTP stream)",
			},
		}, nil
	} else if id == "tt1727587" {
		return []types.StreamItem{
			{
				InfoHash:  "08ada5a7a6183aae1e09d831df6748d566095a10",
				Title:     "480p (torrent)",
				FileIndex: 0,
			},
			{
				URL:   "https://ftp.halifax.rwth-aachen.de/blender/demo/movies/Sintel.2010.1080p.mkv",
				Title: "1080p (HTTP stream)",
			},
		}, nil
	}
	return nil, stremio.ErrNotFound
}
