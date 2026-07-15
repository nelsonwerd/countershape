// Package gitrepo builds isolated Git object databases for hostile materializer tests.
// It is test infrastructure only: production source selection must use internal/gitobj.
package gitrepo

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type ObjectFormat string

const (
	SHA1   ObjectFormat = "sha1"
	SHA256 ObjectFormat = "sha256"
)

var ErrUnsupportedObjectFormat = errors.New("Git object format is unsupported")

type Repository struct {
	GitExecutable string
	Root          string
	ObjectFormat  ObjectFormat
	privateHome   string
}

type File struct {
	Path    string
	Mode    string
	Content []byte
}

type RawTreeEntry struct {
	Mode string
	Name []byte
	OID  string
}

func Init(ctx context.Context, gitExecutable, parent string, format ObjectFormat) (Repository, error) {
	if !filepath.IsAbs(gitExecutable) {
		return Repository{}, fmt.Errorf("Git executable must be absolute")
	}
	if format != SHA1 && format != SHA256 {
		return Repository{}, fmt.Errorf("unknown object format %q", format)
	}
	root, err := os.MkdirTemp(parent, "countershape-gitrepo-")
	if err != nil {
		return Repository{}, err
	}
	if err := os.Chmod(root, 0o700); err != nil {
		return Repository{}, err
	}
	home := filepath.Join(root, "fixture-home")
	repository := filepath.Join(root, "repository.git")
	if err := os.Mkdir(home, 0o700); err != nil {
		return Repository{}, err
	}
	fixture := Repository{GitExecutable: gitExecutable, Root: repository, ObjectFormat: format, privateHome: home}
	args := []string{"init", "--bare", "--object-format=" + string(format), repository}
	if _, err := fixture.run(ctx, false, nil, args...); err != nil {
		if format == SHA256 && isUnsupportedSHA256Init(err) {
			return Repository{}, fmt.Errorf("%w: %v", ErrUnsupportedObjectFormat, err)
		}
		return Repository{}, err
	}
	return fixture, nil
}

func isUnsupportedSHA256Init(err error) bool {
	text := strings.ToLower(err.Error())
	for _, marker := range []string{
		"unknown option `object-format", "unknown option 'object-format", "unknown option: --object-format",
		"unknown hash algorithm 'sha256'", "unknown hash algorithm sha256", "unsupported object format",
	} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func (r Repository) run(ctx context.Context, inRepository bool, stdin []byte, args ...string) ([]byte, error) {
	closed := make([]string, 0, len(args)+8)
	closed = append(closed, "--no-pager", "--no-replace-objects", "-c", "protocol.allow=never", "-c", "core.hooksPath=/dev/null")
	if inRepository {
		closed = append(closed, "--git-dir="+r.Root)
	}
	closed = append(closed, args...)
	command := exec.CommandContext(ctx, r.GitExecutable, closed...)
	command.Dir = r.privateHome
	command.Env = []string{
		"HOME=" + r.privateHome,
		"TMPDIR=" + r.privateHome,
		"LANG=C",
		"LC_ALL=C",
		"TZ=UTC",
		"NO_COLOR=1",
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_TERMINAL_PROMPT=0",
		"GIT_NO_REPLACE_OBJECTS=1",
		"GIT_NO_LAZY_FETCH=1",
	}
	command.Stdin = bytes.NewReader(stdin)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
	}
	return append([]byte(nil), stdout.Bytes()...), nil
}

func oneLine(raw []byte) (string, error) {
	line := strings.TrimSuffix(string(raw), "\n")
	if line == "" || strings.ContainsAny(line, "\r\n\x00") {
		return "", fmt.Errorf("expected one Git output line")
	}
	return line, nil
}

func (r Repository) WriteBlob(ctx context.Context, content []byte) (string, error) {
	output, err := r.run(ctx, true, content, "hash-object", "-w", "--stdin")
	if err != nil {
		return "", err
	}
	return oneLine(output)
}

func (r Repository) WriteRawTree(ctx context.Context, entries []RawTreeEntry) (string, error) {
	var object bytes.Buffer
	for _, entry := range entries {
		if entry.Mode == "" || len(entry.Name) == 0 || bytes.IndexByte(entry.Name, 0) >= 0 {
			return "", fmt.Errorf("invalid raw tree entry")
		}
		oid, err := hex.DecodeString(entry.OID)
		if err != nil {
			return "", err
		}
		want := 20
		if r.ObjectFormat == SHA256 {
			want = 32
		}
		if len(oid) != want {
			return "", fmt.Errorf("OID has %d bytes, want %d", len(oid), want)
		}
		object.WriteString(entry.Mode)
		object.WriteByte(' ')
		object.Write(entry.Name)
		object.WriteByte(0)
		object.Write(oid)
	}
	output, err := r.run(ctx, true, object.Bytes(), "hash-object", "--literally", "-t", "tree", "-w", "--stdin")
	if err != nil {
		return "", err
	}
	return oneLine(output)
}

type treeNode struct {
	files    map[string]File
	children map[string]*treeNode
}

func newTreeNode() *treeNode {
	return &treeNode{files: map[string]File{}, children: map[string]*treeNode{}}
}

func (r Repository) WriteFilesTree(ctx context.Context, files []File) (string, error) {
	root := newTreeNode()
	for _, file := range files {
		if file.Mode != "100644" && file.Mode != "100755" && file.Mode != "120000" {
			return "", fmt.Errorf("unsupported fixture mode %q", file.Mode)
		}
		components := strings.Split(file.Path, "/")
		if len(components) == 0 {
			return "", fmt.Errorf("empty fixture path")
		}
		node := root
		for _, component := range components[:len(components)-1] {
			if component == "" {
				return "", fmt.Errorf("empty fixture path component")
			}
			child := node.children[component]
			if child == nil {
				child = newTreeNode()
				node.children[component] = child
			}
			node = child
		}
		leaf := components[len(components)-1]
		if leaf == "" {
			return "", fmt.Errorf("empty fixture leaf")
		}
		if _, exists := node.files[leaf]; exists {
			return "", fmt.Errorf("duplicate fixture path %q", file.Path)
		}
		node.files[leaf] = file
	}
	return r.writeNode(ctx, root)
}

func (r Repository) writeNode(ctx context.Context, node *treeNode) (string, error) {
	entries := make([]RawTreeEntry, 0, len(node.files)+len(node.children))
	for name, child := range node.children {
		oid, err := r.writeNode(ctx, child)
		if err != nil {
			return "", err
		}
		entries = append(entries, RawTreeEntry{Mode: "40000", Name: []byte(name), OID: oid})
	}
	for name, file := range node.files {
		oid, err := r.WriteBlob(ctx, file.Content)
		if err != nil {
			return "", err
		}
		entries = append(entries, RawTreeEntry{Mode: file.Mode, Name: []byte(name), OID: oid})
	}
	sort.Slice(entries, func(i, j int) bool {
		left := append(append([]byte(nil), entries[i].Name...), treeSortSuffix(entries[i].Mode)...)
		right := append(append([]byte(nil), entries[j].Name...), treeSortSuffix(entries[j].Mode)...)
		return bytes.Compare(left, right) < 0
	})
	return r.WriteRawTree(ctx, entries)
}

func treeSortSuffix(mode string) []byte {
	if mode == "40000" || mode == "040000" {
		return []byte{'/'}
	}
	return nil
}

func (r Repository) CommitTree(ctx context.Context, treeOID, parentOID, message string) (string, error) {
	args := []string{"commit-tree", treeOID}
	if parentOID != "" {
		args = append(args, "-p", parentOID)
	}
	args = append(args, "-m", message)
	command := exec.CommandContext(ctx, r.GitExecutable, append([]string{"--no-pager", "--no-replace-objects", "--git-dir=" + r.Root}, args...)...)
	command.Dir = r.privateHome
	command.Env = []string{
		"HOME=" + r.privateHome,
		"TMPDIR=" + r.privateHome,
		"LANG=C",
		"LC_ALL=C",
		"TZ=UTC",
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_AUTHOR_NAME=Countershape Fixture",
		"GIT_AUTHOR_EMAIL=fixture@countershape.invalid",
		"GIT_AUTHOR_DATE=2000-01-01T00:00:00Z",
		"GIT_COMMITTER_NAME=Countershape Fixture",
		"GIT_COMMITTER_EMAIL=fixture@countershape.invalid",
		"GIT_COMMITTER_DATE=2000-01-01T00:00:00Z",
		"GIT_NO_REPLACE_OBJECTS=1",
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return "", fmt.Errorf("git commit-tree: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return oneLine(stdout.Bytes())
}

// WriteCommitLiteral creates a deterministic commit object without asking Git
// to validate the referenced tree. It is reserved for malformed-tree fixtures.
func (r Repository) WriteCommitLiteral(ctx context.Context, treeOID string) (string, error) {
	body := []byte("tree " + treeOID + "\n" +
		"author Countershape Fixture <fixture@countershape.invalid> 946684800 +0000\n" +
		"committer Countershape Fixture <fixture@countershape.invalid> 946684800 +0000\n\n" +
		"hostile tree fixture\n")
	output, err := r.run(ctx, true, body, "hash-object", "--literally", "-t", "commit", "-w", "--stdin")
	if err != nil {
		return "", err
	}
	return oneLine(output)
}

func (r Repository) CommitFiles(ctx context.Context, files []File, parentOID, message string) (string, error) {
	tree, err := r.WriteFilesTree(ctx, files)
	if err != nil {
		return "", err
	}
	return r.CommitTree(ctx, tree, parentOID, message)
}

func (r Repository) UpdateRef(ctx context.Context, ref, oid string) error {
	_, err := r.run(ctx, true, nil, "update-ref", ref, oid)
	return err
}

func (r Repository) InstallReplacement(ctx context.Context, original, replacement string) error {
	return r.UpdateRef(ctx, "refs/replace/"+original, replacement)
}

func (r Repository) SetConfig(ctx context.Context, key, value string) error {
	_, err := r.run(ctx, true, nil, "config", "--local", key, value)
	return err
}

func (r Repository) ConfigurePromisor(ctx context.Context, remoteURL string) error {
	for _, setting := range [][2]string{
		{"core.repositoryFormatVersion", "1"},
		{"extensions.partialClone", "origin"},
		{"remote.origin.promisor", "true"},
		{"remote.origin.partialCloneFilter", "blob:none"},
		{"remote.origin.url", remoteURL},
	} {
		if err := r.SetConfig(ctx, setting[0], setting[1]); err != nil {
			return err
		}
	}
	return nil
}

func (r Repository) InstallAlternates(path string) error {
	directory := filepath.Join(r.Root, "objects", "info")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(directory, "alternates"), []byte(path+"\n"), 0o600)
}

func (r Repository) InstallPromisorMarker() error {
	directory := filepath.Join(r.Root, "objects", "pack")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(directory, "fixture.promisor"), []byte("promisor\n"), 0o600)
}

func (r Repository) InstallPolicySentinels(ctx context.Context, sentinel string) error {
	script := filepath.Join(filepath.Dir(sentinel), "git-policy-sentinel.sh")
	scriptBody := "#!/bin/sh\n: > " + shellQuote(sentinel) + "\ncat\n"
	if err := os.WriteFile(script, []byte(scriptBody), 0o700); err != nil {
		return err
	}
	hook := filepath.Join(r.Root, "hooks", "post-checkout")
	if err := os.WriteFile(hook, []byte("#!/bin/sh\n: > "+shellQuote(sentinel)+"\n"), 0o700); err != nil {
		return err
	}
	settings := [][2]string{
		{"core.hooksPath", filepath.Join(r.Root, "hooks")},
		{"filter.countershape.smudge", script},
		{"filter.countershape.clean", script},
		{"remote.origin.url", "ext::" + script},
	}
	for _, setting := range settings {
		if err := r.SetConfig(ctx, setting[0], setting[1]); err != nil {
			return err
		}
	}
	return nil
}

// WriteGitWrapper creates an absolute executable that delegates to realGit,
// records the lazy-fetch environment, and optionally changes the first byte of
// ASCII blob output without changing its length.
func WriteGitWrapper(parent, realGit, environmentLog string, tamperBlob bool) (string, error) {
	objectType := ""
	if tamperBlob {
		objectType = "blob"
	}
	return WriteGitObjectTamperWrapper(parent, realGit, environmentLog, objectType)
}

// WriteGitObjectTamperWrapper delegates to real Git but can change the first
// streamed byte of one cat-file object type without changing output length.
// It handles binary tree bytes without shell command substitution.
func WriteGitObjectTamperWrapper(parent, realGit, environmentLog, tamperObjectType string) (string, error) {
	if tamperObjectType != "" && tamperObjectType != "blob" && tamperObjectType != "tree" && tamperObjectType != "commit" {
		return "", fmt.Errorf("unsupported tamper object type %q", tamperObjectType)
	}
	path := filepath.Join(parent, "git-wrapper.sh")
	body := "#!/bin/sh\n" +
		"printf '%s\\n' \"${GIT_NO_LAZY_FETCH-UNSET}\" >> " + shellQuote(environmentLog) + "\n" +
		"tamper_type=" + shellQuote(tamperObjectType) + "\n" +
		"previous=''\n" +
		"for argument in \"$@\"; do\n" +
		"  if [ \"$previous\" = 'cat-file' ] && [ -n \"$tamper_type\" ] && [ \"$argument\" = \"$tamper_type\" ]; then\n" +
		"    temporary=$(/usr/bin/mktemp \"${TMPDIR}/countershape-git-wrapper.XXXXXX\") || exit 91\n" +
		"    " + shellQuote(realGit) + " \"$@\" > \"$temporary\" || { status=$?; /bin/rm -f \"$temporary\"; exit $status; }\n" +
		"    /usr/bin/python3 -c 'import pathlib,sys; p=pathlib.Path(sys.argv[1]); d=bytearray(p.read_bytes()); d[0] ^= 1 if d else 0; sys.stdout.buffer.write(d)' \"$temporary\"\n" +
		"    status=$?\n" +
		"    /bin/rm -f \"$temporary\"\n" +
		"    exit $status\n" +
		"  fi\n" +
		"  previous=$argument\n" +
		"done\n" +
		"exec " + shellQuote(realGit) + " \"$@\"\n"
	if err := os.WriteFile(path, []byte(body), 0o700); err != nil {
		return "", err
	}
	return path, nil
}

// WriteGitCommandLogWrapper creates an absolute executable that delegates to
// realGit and records one space-separated argv line per invocation. Fixture
// arguments are controlled by Countershape's tests, so the log is intended for
// command-boundary assertions rather than general shell transcript parsing.
func WriteGitCommandLogWrapper(parent, realGit, commandLog string) (string, error) {
	path := filepath.Join(parent, "git-command-log-wrapper.sh")
	body := "#!/bin/sh\n" +
		"printf '%s\\n' \"$*\" >> " + shellQuote(commandLog) + "\n" +
		"exec " + shellQuote(realGit) + " \"$@\"\n"
	if err := os.WriteFile(path, []byte(body), 0o700); err != nil {
		return "", err
	}
	return path, nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func (r Repository) RemoveLooseObject(oid string) error {
	if len(oid) < 3 {
		return fmt.Errorf("invalid loose object OID")
	}
	return os.Remove(filepath.Join(r.Root, "objects", oid[:2], oid[2:]))
}

func (r Repository) ObjectPath(oid string) string {
	if len(oid) < 3 {
		return ""
	}
	return filepath.Join(r.Root, "objects", oid[:2], oid[2:])
}

func LFSPointer(oid string, size int64) []byte {
	return []byte(fmt.Sprintf("version https://git-lfs.github.com/spec/v1\noid sha256:%s\nsize %d\n", oid, size))
}
