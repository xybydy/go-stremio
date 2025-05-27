package stremio

import (
	"context"
	"fmt"
	"io/fs"
	"path"

	"github.com/xybydy/go-stremio/types"
)

// PrefixedFS is a wrapper around a fs.FS which adds a prefix before looking up the file.
// This is useful if you use the Go 1.16 embed feature and the directory name doesn't match the request URL.
// For example if your embed.FS contains a "web/index.html", but you want to serve it for "/configure" requests,
// the filesystem middleware already takes care of stripping the "/configure" prefix, but then we also need to
// add the "web" prefix. This wrapper does that.
type PrefixedFS struct {
	// Prefix for adding to the filename before looking it up in the FS.
	// Forward slashes are added before and after the prefix and then the file name is cleaned up (removing duplicate slashes).
	Prefix string
	// Regular fs.FS which you can create with `os.DirFS(folder)` for example.
	FS fs.FS
}

// Open adds a prefix to the name and then calls the wrapped FS' Open method.
func (fs *PrefixedFS) Open(name string) (fs.File, error) {
	name = path.Clean(fs.Prefix + "/" + name)
	return fs.FS.Open(name)
}

// GetMetaFromContext returns the Meta object that's stored in the context.
// It returns an error if no meta was found in the context or the value found isn't of type Meta.
// The former one is ErrNoMeta which acts as sentinel error so you can check for it.
func GetMetaFromContext(ctx context.Context) (types.MetaItem, error) {
	metaIface := ctx.Value("meta")
	if metaIface == nil {
		return types.MetaItem{}, ErrNoMeta
	} else if meta, ok := metaIface.(types.MetaItem); ok {
		return meta, nil
	}
	return types.MetaItem{}, fmt.Errorf("couldn't turn meta interface value to proper object: type is %T", metaIface)
}
