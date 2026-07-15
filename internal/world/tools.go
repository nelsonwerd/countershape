package world

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

var executedMajorPattern = regexp.MustCompile(`(?:^|[[:space:]])v?([1-9][0-9]*)\.[0-9]+(?:\.[0-9]+)?(?:$|[-+[:space:]])`)

const resolveToolsThroughAmbientPATH = false // MUTANT_U2_RESOLVE_TOOL_THROUGH_PATH

const maxExecutionToolBytes int64 = 512 << 20

// ToolSpec names one already-resolved executable and the direct argv needed to
// obtain its bounded version line. Admission executes this probe itself; a
// caller-authored version or satisfied Boolean is not authority.
type ToolSpec struct {
	Name              string
	AbsolutePath      string
	VersionConstraint string
	VersionArgs       []string
}

type resolvedTool struct {
	name              string
	absolutePath      string
	version           string
	major             int
	versionConstraint string
	executableDigest  domain.Digest
	executableBytes   int64
	executableMode    os.FileMode
}

func (t resolvedTool) revalidate() error {
	info, err := os.Lstat(t.absolutePath)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 ||
		info.Size() < 1 || info.Size() > maxExecutionToolBytes || info.Size() != t.executableBytes ||
		info.Mode().Perm() != t.executableMode.Perm() {
		return refuse(CodeToolRejected, "execution tool facts changed after admission", err)
	}
	data, err := os.ReadFile(t.absolutePath)
	if err != nil {
		return refuse(CodeToolRejected, "execution tool bytes cannot be re-read", err)
	}
	digest, err := canon.DigestBytes("ExecutionToolBytes", data)
	if err != nil {
		return err
	}
	parsed, err := domain.ParseDigest(digest.String())
	if err != nil || parsed != t.executableDigest {
		return refuse(CodeToolRejected, "execution tool byte digest changed after admission", err)
	}
	return nil
}

// ToolRegistry is an immutable, closed logical-name-to-absolute-capability
// registry. It never consults PATH.
type ToolRegistry struct{ byName map[string]resolvedTool }

func NewToolRegistry(ctx context.Context, rawScratchRoot string, specs ...ToolSpec) (ToolRegistry, error) {
	if len(specs) == 0 || len(specs) > 16 {
		return ToolRegistry{}, refuse(CodeToolRejected, "tool registry size is outside 1..16", nil)
	}
	scratch, err := privateCanonicalDirectory(rawScratchRoot)
	if err != nil {
		return ToolRegistry{}, refuse(CodeToolRejected, "tool-measurement scratch root is not private and canonical", err)
	}
	byName := make(map[string]resolvedTool, len(specs))
	for _, spec := range specs {
		if !validToolName(spec.Name) || spec.VersionConstraint != "executed-major-only" ||
			len(spec.VersionArgs) < 1 || len(spec.VersionArgs) > 4 {
			return ToolRegistry{}, refuse(CodeToolRejected, "tool declaration is outside the executed-major-only profile", nil)
		}
		if _, duplicate := byName[spec.Name]; duplicate {
			return ToolRegistry{}, refuse(CodeToolRejected, "duplicate tool name: "+spec.Name, nil)
		}
		for _, argument := range spec.VersionArgs {
			if !validBoundedText(argument, 128) || !strings.HasPrefix(argument, "--") || strings.ContainsAny(argument, `/\\=`) {
				return ToolRegistry{}, refuse(CodeToolRejected, "version probe argv is outside the closed long-flag profile", nil)
			}
		}
		tool, err := admitTool(ctx, scratch, spec)
		if err != nil {
			return ToolRegistry{}, err
		}
		byName[spec.Name] = tool
	}
	return ToolRegistry{byName: byName}, nil
}

func admitTool(ctx context.Context, scratch string, spec ToolSpec) (resolvedTool, error) {
	candidatePath := spec.AbsolutePath
	if resolveToolsThroughAmbientPATH && !filepath.IsAbs(candidatePath) {
		for _, directory := range filepath.SplitList(os.Getenv("PATH")) {
			candidate := filepath.Join(directory, candidatePath)
			if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0o111 != 0 {
				candidatePath = candidate
				break
			}
		}
	}
	if !filepath.IsAbs(candidatePath) || filepath.Clean(candidatePath) != candidatePath {
		return resolvedTool{}, refuse(CodeToolRejected, "tool path must be clean and absolute", nil)
	}
	resolved, err := filepath.EvalSymlinks(candidatePath)
	if err != nil || resolved != candidatePath {
		return resolvedTool{}, refuse(CodeToolRejected, "tool path must already be symlink-resolved; PATH lookup is forbidden", err)
	}
	info, err := os.Lstat(resolved)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 ||
		info.Size() < 1 || info.Size() > maxExecutionToolBytes {
		return resolvedTool{}, refuse(CodeToolRejected, "tool path is not an executable regular file", err)
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return resolvedTool{}, refuse(CodeToolRejected, "tool bytes cannot be measured", err)
	}
	digest, err := canon.DigestBytes("ExecutionToolBytes", data)
	if err != nil {
		return resolvedTool{}, err
	}
	parsedDigest, err := domain.ParseDigest(digest.String())
	if err != nil {
		return resolvedTool{}, err
	}
	probeRoot, err := os.MkdirTemp(scratch, "countershape-tool-probe-")
	if err != nil {
		return resolvedTool{}, refuse(CodeToolRejected, "tool version-probe root cannot be allocated", err)
	}
	defer os.RemoveAll(probeRoot)
	if err := os.Chmod(probeRoot, 0o700); err != nil {
		return resolvedTool{}, refuse(CodeToolRejected, "tool version-probe root is not private", err)
	}
	tool := resolvedTool{
		name: spec.Name, absolutePath: resolved, versionConstraint: spec.VersionConstraint, executableDigest: parsedDigest,
		executableBytes: info.Size(), executableMode: info.Mode().Perm(),
	}
	logicalArgv := append([]string{spec.Name}, spec.VersionArgs...)
	probe := runProcess(ctx, processRequest{
		tool: tool, logicalArgv: logicalArgv,
		environment: []string{"HOME=" + probeRoot, "TMPDIR=" + probeRoot, "LANG=C", "LC_ALL=C", "TZ=UTC", "NO_COLOR=1"},
		cwd:         probeRoot, stdoutLimit: 4096, stderrLimit: 4096,
		executionBudgetMS: 5000, teardownBudgetMS: 1000,
	})
	if probe.primary != "" || probe.exitCode != 0 || probe.stdoutOverflow || probe.stderrOverflow ||
		len(probe.stderr) != 0 || probe.teardownError || probe.orphanRisk || !probe.directChildWaited ||
		!probe.stdoutDrained || !probe.stderrDrained || !probe.finalProbeClean {
		return resolvedTool{}, refuse(CodeToolRejected, "direct tool version probe failed or exceeded its closed lifecycle profile", nil)
	}
	version := strings.TrimSuffix(string(probe.stdout), "\n")
	if !validBoundedText(version, 256) || strings.ContainsAny(version, "\r\n") {
		return resolvedTool{}, refuse(CodeToolRejected, "tool version is not one bounded UTF-8 line", nil)
	}
	match := executedMajorPattern.FindStringSubmatch(version)
	if len(match) != 2 {
		return resolvedTool{}, refuse(CodeToolRejected, "tool version has no executed major", nil)
	}
	major, err := strconv.Atoi(match[1])
	if err != nil || major < 1 {
		return resolvedTool{}, refuse(CodeToolRejected, "tool major version is invalid", err)
	}
	tool.version = version
	tool.major = major
	if err := tool.revalidate(); err != nil {
		return resolvedTool{}, err
	}
	return tool, nil
}

func (r ToolRegistry) resolvePlan(plan domain.WorldPlan) (resolvedTool, error) {
	requirements := plan.RequiredTools()
	if len(requirements) == 0 || len(requirements) != len(r.byName) {
		return resolvedTool{}, refuse(CodeToolRejected, "registry does not exactly cover required_tools", nil)
	}
	for _, requirement := range requirements {
		tool, present := r.byName[requirement.Name]
		if !present || tool.versionConstraint != requirement.VersionConstraint {
			return resolvedTool{}, refuse(CodeToolRejected, "required tool measurement mismatch: "+requirement.Name, nil)
		}
		if err := tool.revalidate(); err != nil {
			return resolvedTool{}, err
		}
	}
	argv := plan.StartArgv()
	if len(argv) == 0 {
		return resolvedTool{}, refuse(CodeToolRejected, "start argv is empty", nil)
	}
	tool, present := r.byName[argv[0]]
	if !present {
		return resolvedTool{}, refuse(CodeToolRejected, "argv[0] is absent from the closed registry", nil)
	}
	return tool, nil
}

func (r ToolRegistry) Names() []string {
	names := make([]string, 0, len(r.byName))
	for name := range r.byName {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func validToolName(name string) bool {
	if len(name) < 1 || len(name) > 128 || !utf8.ValidString(name) {
		return false
	}
	for index := 0; index < len(name); index++ {
		character := name[index]
		alphaNumeric := character >= 'a' && character <= 'z' ||
			character >= 'A' && character <= 'Z' || character >= '0' && character <= '9'
		if alphaNumeric || (index > 0 && strings.ContainsRune("._+-", rune(character))) {
			continue
		}
		return false
	}
	return true
}

func validBoundedText(value string, limit int) bool {
	if len(value) < 1 || len(value) > limit || !utf8.ValidString(value) || strings.TrimSpace(value) != value {
		return false
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return false
		}
	}
	return true
}

func (t resolvedTool) display() string {
	return fmt.Sprintf("%s %s (major %d) at %s", t.name, t.version, t.major, t.absolutePath)
}

func (t resolvedTool) measurementDigest() domain.Digest { return t.executableDigest }

func isToolRejection(err error) bool {
	var refusal *Refusal
	return errors.As(err, &refusal) && refusal.Code == CodeToolRejected
}
