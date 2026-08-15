// Package assets verifies the production web asset closure embedded in the
// Countershape binary.
package assets

import (
	"bytes"
	"errors"
	"io/fs"
	"path"
	"sort"
	"strings"

	webassets "github.com/nelsonwerd/countershape/web"
)

var exactRoster = []string{"assets/studio.css", "assets/studio.js", "index.html"}

// Production returns the checked-in embedded distribution only after its
// exact, bounded, no-source-map roster has been verified.
func Production() (fs.FS, error) {
	value := webassets.Assets()
	var names []string
	var total int64
	err := fs.WalkDir(value, ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 || path.Ext(name) == ".map" {
			return errors.New("unsafe embedded asset")
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 2<<20 {
			return errors.New("invalid embedded asset")
		}
		total += info.Size()
		names = append(names, name)
		body, err := fs.ReadFile(value, name)
		if err != nil || bytes.Contains(body, []byte("localhost:")) || bytes.Contains(body, []byte("127.0.0.1:")) || bytes.Contains(body, []byte("sourceMappingURL=")) {
			return errors.New("development marker in embedded asset")
		}
		return nil
	})
	if err != nil || total > 8<<20 {
		return nil, errors.New("embedded asset closure refused")
	}
	sort.Strings(names)
	if strings.Join(names, "\n") != strings.Join(exactRoster, "\n") {
		return nil, errors.New("embedded asset roster refused")
	}
	index, err := fs.ReadFile(value, "index.html")
	if err != nil || !bytes.Contains(index, []byte("/assets/studio.css")) || !bytes.Contains(index, []byte("/assets/studio.js")) {
		return nil, errors.New("embedded asset joins refused")
	}
	return value, nil
}
