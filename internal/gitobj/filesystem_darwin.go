//go:build darwin

package gitobj

import (
	"fmt"
	"os"
	"syscall"
)

func filesystemIdentityAt(path string) (filesystemIdentity, error) {
	info, err := os.Stat(path)
	if err != nil {
		return filesystemIdentity{}, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return filesystemIdentity{}, fmt.Errorf("darwin stat payload unavailable")
	}
	return filesystemIdentity{Device: fmt.Sprintf("%d", stat.Dev), Inode: fmt.Sprintf("%d", stat.Ino)}, nil
}
