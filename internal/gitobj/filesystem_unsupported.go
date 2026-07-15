//go:build !darwin

package gitobj

import "fmt"

func filesystemIdentityAt(path string) (filesystemIdentity, error) {
	return filesystemIdentity{}, fmt.Errorf("filesystem identity is unreceipted outside Darwin: %s", path)
}
