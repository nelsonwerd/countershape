package report

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nelsonwerd/countershape/internal/server"
)

func TestDefaultReportIsDeterministicInertAndMinimized(t *testing.T) {
	snapshot, err := server.BuildSeedExportSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	first, err := Render(snapshot, false)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Render(snapshot, false)
	if err != nil || !bytes.Equal(first, second) {
		t.Fatal("report bytes are not deterministic")
	}
	want := []string{Warning, "content-security-policy", "default-src 'none'", "UNRECEIPTED", "Captured → Projection", "cli.stdout.json.mode", "cli.exit.code", "redaction=countershape/report-preview/v1"}
	lower := strings.ToLower(string(first))
	for _, value := range want {
		if !strings.Contains(lower, strings.ToLower(value)) {
			t.Fatalf("missing %q", value)
		}
	}
	for _, forbidden := range []string{"<script", "http://", "https://", "access_token", "csrf", "outer deterministic CLI precedence fixture", "candidate:", `\"value\":\"env\"`} {
		if strings.Contains(string(first), forbidden) {
			t.Fatalf("default report leaked %q", forbidden)
		}
	}
}

func TestRawReportRequiresExplicitCallerAndRemainsInert(t *testing.T) {
	snapshot, _ := server.BuildSeedExportSnapshot()
	body, err := Render(snapshot, true)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(body, []byte("DANGEROUS RAW OPT-IN")) || !bytes.Contains(body, []byte("outer deterministic CLI precedence fixture")) || bytes.Contains(body, []byte("<script")) {
		t.Fatal("raw report boundary is not explicit or inert")
	}
}

func TestVisibleEncodingAndKnownPatternRedaction(t *testing.T) {
	hostile := "</script><svg/onload=alert(1)>\x1b]8;;https://bad.example\aBAD\u202eheading"
	encoded := visible(hostile)
	for _, want := range []string{"</script>", "<svg/onload=alert(1)>", "⟦U+001B⟧", "⟦U+0007⟧", "⟦U+202E⟧"} {
		if !strings.Contains(encoded, want) {
			t.Fatalf("visible encoding lost %q", want)
		}
	}
	secret := "Bearer abcdefghijklmnopqrstuvwxyz"
	redacted := redactSelected(secret)
	if strings.Contains(redacted, secret) || !strings.HasPrefix(redacted, "REDACTED_KNOWN_PATTERN digest=sha256:") {
		t.Fatal("known secret-shaped selected value was not minimized")
	}
	long := visible(strings.Repeat("x", maxVisibleText+200))
	if !strings.Contains(long, "⟦TRUNCATED sha256=") || len(long) > maxVisibleText+100 {
		t.Fatal("bounded visible value was not explicitly truncated")
	}
}

func TestUnknownReceiptGradeRendersVerbatimButInert(t *testing.T) {
	value := view{Warning: Warning, Receipts: []receiptView{{Authority: "didrun", Grade: `FUTURE</td><script>alert(1)</script>`, Commit: "abc", Command: "sha256:def"}}}
	var output bytes.Buffer
	if err := reportTemplate.Execute(&output, value); err != nil {
		t.Fatal(err)
	}
	text := output.String()
	if !strings.Contains(text, "FUTURE&lt;/td&gt;&lt;script&gt;alert(1)&lt;/script&gt;") || strings.Contains(text, "<script>") {
		t.Fatal("unknown receipt grade was upgraded or activated")
	}
}

func TestExclusiveWriterRefusesOverwriteAndSymlinkParent(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "report.html")
	if err := WriteExclusive(output, []byte("first")); err != nil {
		t.Fatal(err)
	}
	if err := WriteExclusive(output, []byte("second")); err == nil {
		t.Fatal("overwrite succeeded")
	}
	info, err := os.Lstat(output)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
		t.Fatal("terminal output mode refused")
	}
	real := filepath.Join(root, "real")
	if err := os.Mkdir(real, 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	if err := WriteExclusive(filepath.Join(link, "escape.html"), []byte("x")); err == nil {
		t.Fatal("symlink parent accepted")
	}
}

func TestCLIRequiresBothConfidentialityAndRawDangerAcknowledgement(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "report.html")
	var stdout, stderr bytes.Buffer
	if code := RunCLI([]string{"--output", output}, &stdout, &stderr); code != 2 || !strings.Contains(stderr.String(), Warning) {
		t.Fatal("missing confidentiality acknowledgement accepted")
	}
	stdout.Reset()
	stderr.Reset()
	if code := RunCLI([]string{"--output", output, "--acknowledge-confidentiality-not-established", "--include-raw"}, &stdout, &stderr); code != 2 {
		t.Fatal("one-sided raw acknowledgement accepted")
	}
	stdout.Reset()
	stderr.Reset()
	if code := RunCLI([]string{"--output", output, "--acknowledge-confidentiality-not-established"}, &stdout, &stderr); code != 0 {
		t.Fatalf("valid export failed: %s", stderr.String())
	}
}
