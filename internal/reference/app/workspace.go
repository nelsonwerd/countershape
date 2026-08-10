package app

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
)

const maximumSourceBytes = 1 << 20

type InputError struct {
	Code   string
	Detail string
}

func (e *InputError) Error() string { return e.Code + ": " + e.Detail }

type sourceInput struct {
	displayPath string
	argument    string
	exact       []byte
}

func readSourceInput(path string, stdin io.Reader, workingDirectory string) (sourceInput, error) {
	if !validInputPath(path) {
		return sourceInput{}, &InputError{Code: "INPUT_PATH_INVALID", Detail: "--spec must name one bounded path or '-' for stdin"}
	}
	if path == "-" {
		if stdin == nil {
			return sourceInput{}, &InputError{Code: "INPUT_READ_FAILED", Detail: "stdin is unavailable"}
		}
		exact, err := readBounded(stdin)
		if err != nil {
			return sourceInput{}, err
		}
		return sourceInput{displayPath: "stdin", argument: path, exact: exact}, nil
	}

	root := workingDirectory
	if root == "" {
		var err error
		root, err = os.Getwd()
		if err != nil {
			return sourceInput{}, &InputError{Code: "WORKSPACE_UNAVAILABLE", Detail: "the current working directory is unavailable"}
		}
	}
	if !filepath.IsAbs(root) || filepath.Clean(root) != root {
		return sourceInput{}, &InputError{Code: "WORKSPACE_INVALID", Detail: "the working directory must be clean and absolute"}
	}
	absolute := path
	if !filepath.IsAbs(absolute) {
		absolute = filepath.Join(root, path)
	}
	absolute = filepath.Clean(absolute)

	before, err := os.Lstat(absolute)
	if err != nil {
		return sourceInput{}, &InputError{Code: "INPUT_OPEN_FAILED", Detail: "the source path could not be opened"}
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return sourceInput{}, &InputError{Code: "INPUT_NOT_REGULAR", Detail: "the source path must be a regular file, not a link or special file"}
	}
	if before.Size() < 1 || before.Size() > maximumSourceBytes {
		return sourceInput{}, &InputError{Code: "INPUT_SIZE_INVALID", Detail: "the source must contain 1..1048576 bytes"}
	}

	handle, err := os.Open(absolute)
	if err != nil {
		return sourceInput{}, &InputError{Code: "INPUT_OPEN_FAILED", Detail: "the source path could not be opened"}
	}
	defer handle.Close()
	opened, err := handle.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(before, opened) ||
		before.Size() != opened.Size() || before.ModTime() != opened.ModTime() {
		return sourceInput{}, &InputError{Code: "INPUT_CHANGED", Detail: "the source identity changed while it was opened"}
	}
	exact, err := readBounded(handle)
	if err != nil {
		return sourceInput{}, err
	}
	openedAfter, statErr := handle.Stat()
	pathAfter, pathErr := os.Lstat(absolute)
	if statErr != nil || pathErr != nil || !pathAfter.Mode().IsRegular() || pathAfter.Mode()&os.ModeSymlink != 0 ||
		!os.SameFile(opened, openedAfter) || !os.SameFile(openedAfter, pathAfter) ||
		opened.Size() != openedAfter.Size() || opened.ModTime() != openedAfter.ModTime() {
		return sourceInput{}, &InputError{Code: "INPUT_CHANGED", Detail: "the source identity changed while it was read"}
	}
	return sourceInput{displayPath: path, argument: path, exact: exact}, nil
}

func readBounded(reader io.Reader) ([]byte, error) {
	exact, err := io.ReadAll(io.LimitReader(reader, maximumSourceBytes+1))
	if err != nil {
		return nil, &InputError{Code: "INPUT_READ_FAILED", Detail: "the source bytes could not be read"}
	}
	if len(exact) < 1 || len(exact) > maximumSourceBytes {
		return nil, &InputError{Code: "INPUT_SIZE_INVALID", Detail: "the source must contain 1..1048576 bytes"}
	}
	return exact, nil
}

func validInputPath(path string) bool {
	if path == "-" {
		return true
	}
	if path == "" || len(path) > 4096 || !utf8.ValidString(path) || strings.IndexByte(path, 0) >= 0 {
		return false
	}
	for _, character := range path {
		if unicode.IsControl(character) || unicode.Is(unicode.Cf, character) || unicode.Is(unicode.Zl, character) || unicode.Is(unicode.Zp, character) {
			return false
		}
	}
	return true
}

func isInputError(err error) bool {
	var target *InputError
	return errors.As(err, &target)
}
