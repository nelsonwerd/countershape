//go:build !darwin || !arm64 || !cgo

package contractmaterialize

import (
	"errors"
	"os"
)

func nativePublicationSupported() bool { return false }

func trustedPublicationParent(os.FileInfo) bool { return false }

func ownedByEffectiveUser(os.FileInfo) bool { return false }

func openDirectoryPathNoFollow(string) (durableDirectory, error) {
	return nil, errors.New("Darwin/arm64 no-follow directory open is unreceipted on this platform/toolchain")
}

func openDirectoryAtNoFollow(durableDirectory, string) (durableDirectory, error) {
	return nil, errors.New("Darwin/arm64 descriptor-relative directory open is unreceipted on this platform/toolchain")
}

func openFileAtNoFollow(durableDirectory, string, int, os.FileMode) (durableFile, error) {
	return nil, errors.New("Darwin/arm64 descriptor-relative file open is unreceipted on this platform/toolchain")
}

func mkdirDirectoryAt(durableDirectory, string, os.FileMode) error {
	return errors.New("Darwin/arm64 descriptor-relative directory creation is unreceipted on this platform/toolchain")
}

func unlinkEntryAt(durableDirectory, string, bool) error {
	return errors.New("Darwin/arm64 descriptor-relative cleanup is unreceipted on this platform/toolchain")
}

func renameExclusiveDirectoryAt(durableDirectory, string, string) error {
	return errors.New("Darwin/arm64 exclusive no-follow publication is unreceipted on this platform/toolchain")
}

func physicalLinkCount(os.FileInfo) (uint64, bool) { return 0, false }
