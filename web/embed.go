// Package webassets embeds the exact production studio bundle into the Go
// binary. The bundle contains presentation code only; evidence and bearer
// authority are supplied by the authenticated loopback server at runtime.
package webassets

import (
	"embed"
	"io/fs"
)

//go:embed dist
var distribution embed.FS

var assets = func() fs.FS {
	result, err := fs.Sub(distribution, "dist")
	if err != nil {
		panic("embedded studio distribution is invalid")
	}
	return result
}()

// Assets returns the immutable embedded production distribution.
func Assets() fs.FS { return assets }
