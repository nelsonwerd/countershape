//go:build darwin

package scope

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
)

const (
	inventoryBatch         = 128
	inventoryMaxEntries    = 20_000
	inventoryMaxPathBytes  = 32 * 1024 * 1024
	inventoryMaxDepth      = 128
	inventoryMaxFileBytes  = 64 * 1024 * 1024
	inventoryMaxTotalBytes = 64 * 1024 * 1024
	darwinNoFollowAny      = 0x20000000
	inventoryForbiddenMode = os.ModeSetuid | os.ModeSetgid | os.ModeSticky
)

type walkItem struct {
	path     string
	relative string
	depth    int
}

type walkState struct {
	entries    []Entry
	pathBytes  int64
	totalBytes int64
}

func Snapshot(root string) (Inventory, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) != root {
		return Inventory{}, fail(CodeInventoryInvalid, nil)
	}
	rootInfo, err := os.Lstat(root)
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 ||
		inventoryModeForbidden(rootInfo.Mode()) {
		return Inventory{}, fail(CodeInventoryIdentity, err)
	}
	state := walkState{entries: make([]Entry, 0, 16)}
	queue := []walkItem{{path: root}}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		children, walkErr := readDirectory(root, current, &state)
		if walkErr != nil {
			return Inventory{}, walkErr
		}
		queue = append(queue, children...)
	}
	after, err := os.Lstat(root)
	if err != nil || !after.IsDir() || after.Mode() != rootInfo.Mode() || !os.SameFile(rootInfo, after) {
		return Inventory{}, fail(CodeInventoryChanged, err)
	}
	if len(state.entries) == 0 {
		return Inventory{}, fail(CodeInventoryInvalid, nil)
	}
	sort.Slice(state.entries, func(left, right int) bool {
		return state.entries[left].Path < state.entries[right].Path
	})
	digest := sha256.New()
	for _, entry := range state.entries {
		_, _ = digest.Write([]byte(entry.Path))
		_, _ = digest.Write([]byte{0})
		_, _ = digest.Write([]byte(entry.Mode))
		_, _ = digest.Write([]byte{0})
		_, _ = digest.Write([]byte(entry.SHA256))
		_, _ = digest.Write([]byte{0})
		var size [8]byte
		value := uint64(entry.Size)
		for index := 7; index >= 0; index-- {
			size[index] = byte(value)
			value >>= 8
		}
		_, _ = digest.Write(size[:])
	}
	return Inventory{entries: state.entries, digest: hex.EncodeToString(digest.Sum(nil))}, nil
}

func readDirectory(root string, directory walkItem, state *walkState) ([]walkItem, error) {
	before, err := os.Lstat(directory.path)
	if err != nil || !before.IsDir() || before.Mode()&os.ModeSymlink != 0 {
		return nil, fail(CodeInventoryIdentity, err)
	}
	if inventoryModeForbidden(before.Mode()) {
		return nil, fail(CodeInventorySpecial, nil)
	}
	fd, err := syscall.Open(
		directory.path,
		syscall.O_RDONLY|syscall.O_DIRECTORY|darwinNoFollowAny|syscall.O_CLOEXEC|syscall.O_NONBLOCK,
		0,
	)
	if err != nil {
		return nil, fail(CodeInventoryRead, err)
	}
	handle := os.NewFile(uintptr(fd), directory.path)
	if handle == nil {
		_ = syscall.Close(fd)
		return nil, fail(CodeInventoryRead, nil)
	}
	opened, err := handle.Stat()
	if err != nil || opened.Mode() != before.Mode() || !os.SameFile(before, opened) {
		_ = handle.Close()
		return nil, fail(CodeInventoryIdentity, err)
	}
	children := make([]walkItem, 0)
	for {
		batch, readErr := handle.ReadDir(inventoryBatch)
		for _, listed := range batch {
			name := listed.Name()
			if name == "" || name == "." || name == ".." || strings.ContainsRune(name, filepath.Separator) {
				_ = handle.Close()
				return nil, fail(CodeInventoryInvalid, nil)
			}
			relative := name
			if directory.relative != "" {
				relative = filepath.Join(directory.relative, name)
			}
			relative = filepath.ToSlash(relative)
			depth := directory.depth + 1
			if depth > inventoryMaxDepth || len(state.entries) >= inventoryMaxEntries {
				_ = handle.Close()
				return nil, fail(CodeInventoryLimit, nil)
			}
			pathBytes := int64(len([]byte(relative)))
			if pathBytes > inventoryMaxPathBytes-state.pathBytes {
				_ = handle.Close()
				return nil, fail(CodeInventoryLimit, nil)
			}
			path := filepath.Join(root, filepath.FromSlash(relative))
			listedInfo, listErr := listed.Info()
			info, statErr := os.Lstat(path)
			if listErr != nil || statErr != nil || info.Mode()&os.ModeSymlink != 0 ||
				!os.SameFile(listedInfo, info) {
				_ = handle.Close()
				return nil, fail(CodeInventoryChanged, errors.Join(listErr, statErr))
			}
			if inventoryModeForbidden(listedInfo.Mode()) || inventoryModeForbidden(info.Mode()) {
				_ = handle.Close()
				return nil, fail(CodeInventorySpecial, nil)
			}
			entry := Entry{Path: relative}
			switch {
			case info.IsDir():
				entry.Mode = "040" + modeText(info.Mode().Perm())
				children = append(children, walkItem{path: path, relative: relative, depth: depth})
			case info.Mode().IsRegular():
				if info.Size() < 0 || info.Size() > inventoryMaxFileBytes ||
					info.Size() > inventoryMaxTotalBytes-state.totalBytes {
					_ = handle.Close()
					return nil, fail(CodeInventoryLimit, nil)
				}
				entry.Mode = "100" + modeText(info.Mode().Perm())
				entry.Size = info.Size()
				entry.SHA256, err = digestFile(path, info)
				if err != nil {
					_ = handle.Close()
					return nil, err
				}
				state.totalBytes += info.Size()
			default:
				_ = handle.Close()
				return nil, fail(CodeInventorySpecial, nil)
			}
			state.pathBytes += pathBytes
			state.entries = append(state.entries, entry)
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil || len(batch) == 0 {
			_ = handle.Close()
			return nil, fail(CodeInventoryRead, readErr)
		}
	}
	afterDescriptor, descriptorErr := handle.Stat()
	closeErr := handle.Close()
	afterPath, pathErr := os.Lstat(directory.path)
	if descriptorErr != nil || closeErr != nil || pathErr != nil ||
		afterDescriptor.Mode() != opened.Mode() || afterPath.Mode() != opened.Mode() ||
		!os.SameFile(opened, afterDescriptor) || !os.SameFile(opened, afterPath) {
		return nil, fail(CodeInventoryChanged, errors.Join(descriptorErr, closeErr, pathErr))
	}
	return children, nil
}

func digestFile(path string, expected os.FileInfo) (string, error) {
	fd, err := syscall.Open(
		path,
		syscall.O_RDONLY|syscall.O_CLOEXEC|darwinNoFollowAny|syscall.O_NONBLOCK,
		0,
	)
	if err != nil {
		return "", fail(CodeInventoryRead, err)
	}
	handle := os.NewFile(uintptr(fd), path)
	if handle == nil {
		_ = syscall.Close(fd)
		return "", fail(CodeInventoryRead, nil)
	}
	opened, err := handle.Stat()
	if err != nil || !opened.Mode().IsRegular() || opened.Mode() != expected.Mode() ||
		opened.Size() != expected.Size() || !os.SameFile(expected, opened) {
		_ = handle.Close()
		return "", fail(CodeInventoryIdentity, err)
	}
	digest := sha256.New()
	count, readErr := io.Copy(digest, io.LimitReader(handle, opened.Size()+1))
	afterDescriptor, descriptorErr := handle.Stat()
	closeErr := handle.Close()
	afterPath, pathErr := os.Lstat(path)
	if readErr != nil || descriptorErr != nil || closeErr != nil || pathErr != nil ||
		count != opened.Size() || afterDescriptor.Mode() != opened.Mode() ||
		afterPath.Mode() != opened.Mode() || afterDescriptor.Size() != opened.Size() ||
		afterPath.Size() != opened.Size() || !os.SameFile(opened, afterDescriptor) ||
		!os.SameFile(opened, afterPath) {
		return "", fail(CodeInventoryChanged, errors.Join(readErr, descriptorErr, closeErr, pathErr))
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func inventoryModeForbidden(mode os.FileMode) bool {
	return mode&inventoryForbiddenMode != 0
}

func modeText(mode os.FileMode) string {
	const digits = "01234567"
	return string([]byte{
		digits[(mode>>6)&7],
		digits[(mode>>3)&7],
		digits[mode&7],
	})
}
