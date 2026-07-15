//go:build !darwin || !cgo

package gitobj

import "fmt"

func renameExclusive(from, to string) error {
	return fmt.Errorf("exclusive directory publication is unreceipted on this platform/toolchain")
}
