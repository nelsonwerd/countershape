package http

import (
	"path"
	"strings"
)

func validExpectedScratchRoot(root string) bool {
	return strings.HasPrefix(root, "/") && path.Clean(root) == root
}
