//go:build darwin && arm64 && cgo

package noderuntime

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

const nodeProbeProgram = `process.stdout.write(JSON.stringify({architecture:process.arch,measured_process_exec_path:process.execPath,platform:process.platform,version:process.version})+"\n");`

type boundedBuffer struct {
	buffer   bytes.Buffer
	limit    int
	overflow bool
}

func (writer *boundedBuffer) Write(value []byte) (int, error) {
	remaining := writer.limit - writer.buffer.Len()
	if remaining > 0 {
		if remaining > len(value) {
			remaining = len(value)
		}
		_, _ = writer.buffer.Write(value[:remaining])
	}
	if len(value) > remaining {
		writer.overflow = true
	}
	return len(value), nil
}

func nodeProbeDigest() domain.Digest {
	digest, err := canon.DigestBytes("NodeRuntimeProbeProgram", []byte(nodeProbeProgram))
	if err != nil {
		return ""
	}
	parsed, err := domain.ParseDigest(digest.String())
	if err != nil {
		return ""
	}
	return parsed
}

func resolvePrivateProbeParent(raw string) (string, probeParentIdentity, error) {
	if raw == "" || !filepath.IsAbs(raw) || filepath.Clean(raw) != raw {
		return "", probeParentIdentity{}, refuse(CodeInvalidRuntime, "probe parent must be clean and absolute", nil)
	}
	canonical, err := filepath.EvalSymlinks(raw)
	if err != nil || canonical != raw {
		return "", probeParentIdentity{}, refuse(CodeInvalidRuntime, "probe parent must be symlink-free", err)
	}
	info, err := os.Lstat(raw)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0o700 {
		return "", probeParentIdentity{}, refuse(CodeInvalidRuntime, "probe parent must be a private directory", err)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || int(stat.Uid) != os.Geteuid() {
		return "", probeParentIdentity{}, refuse(CodeInvalidRuntime, "probe parent must be owned by the effective user", nil)
	}
	identity := probeParentIdentity{
		device: uint64(stat.Dev), inode: stat.Ino, owner: uint64(stat.Uid), permissions: uint32(info.Mode().Perm()),
	}
	if !identity.valid() {
		return "", probeParentIdentity{}, refuse(CodeInvalidRuntime, "probe parent identity is invalid", nil)
	}
	return raw, identity, nil
}

func runOwnedProbe(ctx context.Context, executable, parent string) (result probeResult, resultErr error) {
	retainedParent, err := os.Lstat(parent)
	if err != nil {
		return probeResult{}, err
	}
	root, err := os.MkdirTemp(parent, ".countershape-node-probe-")
	if err != nil {
		return probeResult{}, err
	}
	if err := os.Chmod(root, 0o700); err != nil {
		_ = os.RemoveAll(root)
		return probeResult{}, err
	}
	defer func() {
		cleanupErr := os.RemoveAll(root)
		_, absenceErr := os.Lstat(root)
		if cleanupErr != nil || !errors.Is(absenceErr, os.ErrNotExist) {
			result = probeResult{}
			resultErr = refuse(CodeProbeFailed, "private probe cleanup did not prove absence", errors.Join(resultErr, cleanupErr, absenceErr))
		}
	}()
	home := filepath.Join(root, "home")
	tmp := filepath.Join(root, "tmp")
	if err := os.Mkdir(home, 0o700); err != nil {
		return probeResult{}, err
	}
	if err := os.Mkdir(tmp, 0o700); err != nil {
		return probeResult{}, err
	}
	probeContext, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	command := exec.CommandContext(probeContext, executable, "--eval", nodeProbeProgram)
	command.Dir = root
	command.Env = []string{"HOME=" + home, "TMPDIR=" + tmp, "LANG=C", "LC_ALL=C", "TZ=UTC", "NO_COLOR=1"}
	command.Stdin = bytes.NewReader(nil)
	stdout := &boundedBuffer{limit: 8192}
	stderr := &boundedBuffer{limit: 4096}
	command.Stdout = stdout
	command.Stderr = stderr
	command.WaitDelay = time.Second
	err = command.Run()
	if probeContext.Err() != nil || err != nil || stdout.overflow || stderr.overflow || stderr.buffer.Len() != 0 {
		return probeResult{}, errors.Join(errors.New("Node probe process or bounded output failed"), err, probeContext.Err())
	}
	parentAfter, err := os.Lstat(parent)
	if err != nil || !os.SameFile(retainedParent, parentAfter) {
		return probeResult{}, errors.Join(errors.New("probe parent identity changed"), err)
	}
	return parseProbeOutput(stdout.buffer.Bytes(), executable)
}

func parseProbeOutput(output []byte, executable string) (probeResult, error) {
	if len(output) < 2 || output[len(output)-1] != '\n' || bytes.IndexByte(output[:len(output)-1], '\n') >= 0 ||
		bytes.IndexByte(output, '\r') >= 0 || bytes.IndexByte(output, 0) >= 0 {
		return probeResult{}, errors.New("probe output is not one LF-terminated JSON line")
	}
	body := output[:len(output)-1]
	value, err := canon.Parse(body)
	if err != nil {
		return probeResult{}, err
	}
	canonical, err := value.CanonicalChecked()
	if err != nil || !bytes.Equal(canonical, body) {
		return probeResult{}, errors.Join(errors.New("probe JSON is not exact canonical form"), err)
	}
	members, ok := value.Members()
	wantNames := [...]string{"architecture", "measured_process_exec_path", "platform", "version"}
	if !ok || len(members) != len(wantNames) {
		return probeResult{}, errors.New("probe JSON roster differs")
	}
	texts := make(map[string]string, len(members))
	for index, member := range members {
		text, textOK := member.Value.Text()
		if !textOK || member.Name != wantNames[index] {
			return probeResult{}, errors.New("probe JSON name or value type differs")
		}
		texts[member.Name] = text
	}
	version := texts["version"]
	major, versionOK := parseNodeVersion(version)
	if !versionOK || texts["measured_process_exec_path"] != executable ||
		texts["platform"] != "darwin" || texts["architecture"] != "arm64" {
		return probeResult{}, errors.New("probe tuple is outside the Node Darwin arm64 profile")
	}
	return probeResult{
		path: executable, version: version, major: major, platform: texts["platform"],
		architecture: texts["architecture"], programDigest: nodeProbeDigest(),
	}, nil
}
