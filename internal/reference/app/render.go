package app

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
)

const (
	minimumTerminalColumns = 40
	defaultTerminalColumns = 80
	maximumTerminalColumns = 240
)

// TerminalColumns admits a bounded presentation width. Width only affects
// human layout; the canonical JSON protocol is deliberately width-independent.
func TerminalColumns(raw string) int {
	columns, err := strconv.Atoi(raw)
	if err != nil || columns < minimumTerminalColumns || columns > maximumTerminalColumns {
		return defaultTerminalColumns
	}
	return columns
}

func renderJSON(writer io.Writer, response ResponseEnvelope) error {
	exact, err := json.Marshal(response)
	if err != nil {
		return err
	}
	exact = append(exact, '\n')
	_, err = writer.Write(exact)
	return err
}

func renderStudyDomainResult(writer io.Writer, domain string, ordinal int) error {
	exact, err := json.Marshal(StudyDomainResult{
		SchemaVersion: StudyDomainResultSchemaVersion,
		Domain:        domain,
		Ordinal:       ordinal,
		Status:        "GREEN",
	})
	if err != nil {
		return err
	}
	exact = append(exact, '\n')
	_, err = writer.Write(exact)
	return err
}

func renderHuman(writer io.Writer, response ResponseEnvelope, columns int) error {
	columns = admittedColumns(columns)
	if response.Command == "help" {
		_, err := io.WriteString(writer, humanHelp(columns))
		return err
	}
	var lines []string
	if response.Status == "REFUSED" {
		lines = append(lines,
			"Countershape refused the request.",
			"",
			"Status: REFUSED",
			"Code: "+humanSafeText(response.Failure.Code),
		)
		if response.Failure.Path != "" {
			lines = appendField(lines, "Path", response.Failure.Path, columns)
		}
		if response.Failure.Offset != nil {
			lines = append(lines, fmt.Sprintf("Byte offset: %d", *response.Failure.Offset))
		}
		if response.Failure.Detail != "" {
			lines = appendField(lines, "Detail", response.Failure.Detail, columns)
		}
	} else {
		lines = append(lines,
			"Countershape reference CLI",
			"",
			"Status: "+humanSafeText(response.Status),
		)
		lines = appendWrapped(lines, response.Message, columns, 0)
		if response.Source != nil {
			lines = append(lines, "")
			lines = appendLiteralField(lines, "Source", response.Source.Input, columns)
			lines = appendField(lines, "Adapter", response.Source.AdapterDomain+" / "+response.Source.ExecutionShape, columns)
			lines = appendLiteralField(lines, "Source digest", response.Source.SourceDigest, columns)
			lines = appendField(lines, "Candidate ceiling", fmt.Sprintf("%d", response.Source.Budgets.CandidateCount), columns)
			lines = appendField(lines, "Trial ceiling", fmt.Sprintf("%d", response.Source.Budgets.TotalCandidateTrials), columns)
		}
		if response.Preflight != nil {
			unresolved := unresolvedAuthorities(*response.Preflight)
			lines = append(lines,
				"",
				"Execution started: "+yesNo(response.Preflight.ExecutionStarted),
				"Execution authorized: "+yesNo(response.Preflight.ExecutionAuthorized),
				"Unresolved: "+strings.Join(unresolved, ", "),
			)
		}
	}
	if response.Warning != nil {
		lines = append(lines, "", "Execution warning:")
		lines = appendWrapped(lines, *response.Warning, columns, 2)
	}
	if response.NextAction != "" {
		lines = append(lines, "", "Next action:")
		lines = appendWrapped(lines, response.NextAction, columns, 2)
	}
	_, err := io.WriteString(writer, strings.Join(lines, "\n")+"\n")
	return err
}

func humanHelp(columns int) string {
	lines := []string{
		"Countershape reference CLI",
		"",
		"Usage:",
		"  countershape help",
		"  countershape validate --spec <path|-> [--json]",
		"  countershape preflight --spec <path|-> [--json]",
		"  countershape study <domain> --json  (frozen harness only)",
		"",
	}
	lines = appendWrapped(lines, "U7A validates inert source and reports preflight prerequisites.", columns, 0)
	lines = appendWrapped(lines, "It does not run candidates, resume processes, compare outcomes, or emit contracts.", columns, 0)
	lines = appendWrapped(lines, "Installed study handlers are the sole exception and execute only through the frozen machine harness route; interactive study requests are refused.", columns, 0)
	return strings.Join(lines, "\n") + "\n"
}

func admittedColumns(columns int) int {
	if columns < minimumTerminalColumns || columns > maximumTerminalColumns {
		return defaultTerminalColumns
	}
	return columns
}

func appendField(lines []string, label, value string, columns int) []string {
	value = humanSafeText(value)
	prefix := label + ": "
	if len(prefix)+len(value) <= columns {
		return append(lines, prefix+value)
	}
	lines = append(lines, label+":")
	return appendWrapped(lines, value, columns, 2)
}

func appendLiteralField(lines []string, label, value string, columns int) []string {
	value = humanSafeText(value)
	prefix := label + ": "
	if len([]rune(prefix))+len([]rune(value)) <= columns {
		return append(lines, prefix+value)
	}
	lines = append(lines, label+":")
	available := columns - 2
	if available < 1 {
		available = 1
	}
	remaining := []rune(value)
	for len(remaining) > available {
		lines = append(lines, "  "+string(remaining[:available]))
		remaining = remaining[available:]
	}
	return append(lines, "  "+string(remaining))
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func unresolvedAuthorities(view PreflightView) []string {
	result := make([]string, 0, 3)
	if view.ProjectionAuthority == "UNRESOLVED" {
		result = append(result, "projection")
	}
	if view.CandidateAuthority == "UNRESOLVED" {
		result = append(result, "candidates")
	}
	if view.RuntimeAuthority == "UNRESOLVED" {
		result = append(result, "runtime")
	}
	if len(result) == 0 {
		return []string{"none"}
	}
	return result
}

func appendWrapped(lines []string, value string, columns, indent int) []string {
	value = humanSafeText(value)
	prefix := strings.Repeat(" ", indent)
	available := columns - indent
	if available < 1 {
		available = 1
	}
	words := strings.Fields(value)
	if len(words) == 0 {
		return append(lines, prefix)
	}
	current := ""
	flush := func() {
		if current != "" {
			lines = append(lines, prefix+current)
			current = ""
		}
	}
	for _, word := range words {
		if len(word) > available {
			flush()
			for len(word) > available {
				lines = append(lines, prefix+word[:available])
				word = word[available:]
			}
			current = word
			continue
		}
		if current == "" {
			current = word
			continue
		}
		if len(current)+1+len(word) <= available {
			current += " " + word
			continue
		}
		flush()
		current = word
	}
	flush()
	return lines
}

// humanSafeText makes presentation-only text inert before it reaches a
// terminal. Machine JSON retains the original typed values and relies on the
// JSON encoder's escaping; human output visibly escapes control and format
// runes so an attacker-controlled diagnostic path cannot become CSI or OSC.
func humanSafeText(value string) string {
	var result strings.Builder
	for _, current := range value {
		if !unicode.IsControl(current) && !unicode.Is(unicode.Cf, current) && current != '\u2028' && current != '\u2029' {
			result.WriteRune(current)
			continue
		}
		if current <= 0xffff {
			result.WriteString(fmt.Sprintf("\\u%04x", current))
			continue
		}
		result.WriteString(fmt.Sprintf("\\U%08x", current))
	}
	return result.String()
}
