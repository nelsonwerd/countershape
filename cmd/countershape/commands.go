//go:build darwin

package main

import (
	"fmt"
	"io"
	"os"

	"github.com/nelsonwerd/countershape/internal/assets"
	"github.com/nelsonwerd/countershape/internal/reference/app"
	_ "github.com/nelsonwerd/countershape/internal/reference/clistudy"
	_ "github.com/nelsonwerd/countershape/internal/reference/httpstudy"
	"github.com/nelsonwerd/countershape/internal/reference/reproduce"
	"github.com/nelsonwerd/countershape/internal/report"
	"github.com/nelsonwerd/countershape/internal/server"
)

var (
	version = "dev"
	commit  = "unknown"
)

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) > 0 {
		switch args[0] {
		case "studio":
			embedded, err := assets.Production()
			if err != nil {
				fmt.Fprintln(stderr, "countershape studio: embedded asset closure refused")
				return 1
			}
			return server.RunCLI(args[1:], embedded, stdout, stderr)
		case "export", "report":
			return report.RunCLI(args[1:], stdout, stderr)
		case "version", "about", "--version":
			fmt.Fprintf(stdout, "countershape %s (%s)\n", version, commit)
			return 0
		case "--help", "-h":
			printPackageHelp(stdout)
			return 0
		case "help":
			printPackageHelp(stdout)
			return 0
		}
	}
	return app.Run(args, app.Runtime{Stdin: stdin, Stdout: stdout, Stderr: stderr, Columns: app.TerminalColumns(os.Getenv("COLUMNS")), Studies: reproduce.Studies()})
}

func printPackageHelp(output io.Writer) {
	fmt.Fprintln(output, `Countershape local reference package

Usage:
  countershape studio [--no-open] [--seed-state STATE]
  countershape export --output REPORT.html --acknowledge-confidentiality-not-established
  countershape export --output RAW.html --acknowledge-confidentiality-not-established --include-raw --i-understand-raw-may-contain-secrets
  countershape study <http|cli> --json   (frozen harness route)
  countershape validate|preflight ...
  countershape version

Warning: Countershape executes trusted repository code with full user permissions and host network access.
Exports are local evidence projections. CONFIDENTIALITY NOT ESTABLISHED.`)
}
