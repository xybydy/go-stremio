package main

import (
	"context"

	"github.com/xybydy/go-stremio"
	"github.com/xybydy/go-stremio/types"
)

var (
	manifest = types.Manifest{
		ID:      "org.myexampleaddon",
		Version: "1.0.0",

		Description: "simple example",
		Name:        "simple example",

		Catalogs: []types.CatalogItem{},
		ResourceItems: []types.ResourceItem{
			{
				Name:  "stream",
				Types: []string{"movie"},
			},
		},
		Types:      []string{"movie"},
		IDprefixes: []string{"tt"},
	}
)

func streamHandler(_ context.Context, id string, _ any) ([]types.StreamItem, error) {
	if id == "tt1254207" {
		return []types.StreamItem{{URL: "http://distribution.bbb3d.renderfarming.net/video/mp4/bbb_sunflower_1080p_30fps_normal.mp4"}}, nil
	}
	return nil, stremio.ErrNotFound
}

func main() {
	streamHandlers := map[string]stremio.StreamHandler{"movie": streamHandler}

	addon, err := stremio.NewAddon(manifest, nil, streamHandlers, nil, nil, stremio.Options{BindAddr: "0.0.0.0", Port: 7000, DisableRequestLogging: true})
	if err != nil {
		panic(err)
	}

	addon.Run(nil, nil)
}
