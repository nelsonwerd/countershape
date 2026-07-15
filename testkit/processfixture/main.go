package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	stateEnvironment         = "COUNTERSHAPE_STATE_ROOT"
	evidenceEnvironment      = "COUNTERSHAPE_EVIDENCE_ROOT"
	attemptEnvironment       = "COUNTERSHAPE_ATTEMPT_ID"
	originalGroupEnvironment = "COUNTERSHAPE_FIXTURE_ORIGINAL_PGID"
)

type options struct {
	mode        string
	stdoutBytes int
	stderrBytes int
	exitCode    int
}

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		_, _ = fmt.Fprintln(os.Stdout, "countershape-process-fixture v1.0.0")
		return
	}
	if len(os.Args) == 2 && os.Args[1] == "--version-fork" {
		spawnVersionPipeHolder()
		_, _ = fmt.Fprintln(os.Stdout, "countershape-process-fixture v1.0.0")
		return
	}
	if len(os.Args) == 2 && os.Args[1] == "--version-holder" {
		waitForever()
	}
	options, err := parseOptions(os.Args[1:])
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(64)
	}
	markerPresent, err := recordInvocation(options.mode)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(65)
	}
	switch options.mode {
	case "report":
		report(markerPresent)
	case "emit":
		emit(os.Stdout, options.stdoutBytes, 'o')
		emit(os.Stderr, options.stderrBytes, 'e')
	case "echo-stdin":
		if _, err := io.Copy(os.Stdout, os.Stdin); err != nil {
			fatal(err)
		}
	case "emit-both-ignore-term":
		signal.Ignore(syscall.SIGTERM)
		done := make(chan struct{}, 2)
		go func() {
			emit(os.Stdout, options.stdoutBytes, 'o')
			done <- struct{}{}
		}()
		go func() {
			emit(os.Stderr, options.stderrBytes, 'e')
			done <- struct{}{}
		}()
		<-done
		<-done
		waitForever()
	case "exit":
		os.Exit(options.exitCode)
	case "signal-self":
		_ = syscall.Kill(os.Getpid(), syscall.SIGTERM)
		waitForever()
	case "hang":
		waitForever()
	case "ignore-term":
		signal.Ignore(syscall.SIGTERM)
		waitForever()
	case "descendant":
		spawnAndWait("child")
	case "parent-exits":
		spawnWithoutWait("grandchild")
		waitForStateFile("grandchild.pid")
	case "child":
		spawnAndWait("grandchild")
	case "grandchild":
		writePID("grandchild.pid")
		waitForever()
	case "setsid-escape":
		spawnEscapeAndWait("escaped-child")
	case "setpgid-escape":
		spawnEscapeAndWait("escaped-group-child")
	case "escaped-child":
		if _, err := syscall.Setsid(); err != nil {
			fatal(err)
		}
		writePID("escaped.pid")
		writeEscapeIdentity("setsid", "escaped.identity.json")
		waitForever()
	case "escaped-group-child":
		if err := syscall.Setpgid(0, 0); err != nil {
			fatal(err)
		}
		writePID("escaped-group.pid")
		writeEscapeIdentity("setpgid", "escaped-group.identity.json")
		waitForever()
	default:
		fatal(fmt.Errorf("unsupported mode %q", options.mode))
	}
}

func parseOptions(arguments []string) (options, error) {
	result := options{mode: "report"}
	if len(arguments)%2 != 0 {
		return options{}, errors.New("fixture arguments must be exact flag/value pairs")
	}
	seen := map[string]bool{}
	for index := 0; index < len(arguments); index += 2 {
		name, value := arguments[index], arguments[index+1]
		if seen[name] {
			return options{}, fmt.Errorf("duplicate fixture flag %s", name)
		}
		seen[name] = true
		switch name {
		case "--mode":
			result.mode = value
		case "--stdout-bytes":
			parsed, err := boundedInteger(value, 0, 64<<20)
			if err != nil {
				return options{}, err
			}
			result.stdoutBytes = parsed
		case "--stderr-bytes":
			parsed, err := boundedInteger(value, 0, 64<<20)
			if err != nil {
				return options{}, err
			}
			result.stderrBytes = parsed
		case "--exit-code":
			parsed, err := boundedInteger(value, 0, 125)
			if err != nil {
				return options{}, err
			}
			result.exitCode = parsed
		default:
			return options{}, fmt.Errorf("unknown fixture flag %s", name)
		}
	}
	return result, nil
}

func boundedInteger(raw string, minimum, maximum int) (int, error) {
	value, err := strconv.Atoi(raw)
	if err != nil || value < minimum || value > maximum || strconv.Itoa(value) != raw {
		return 0, fmt.Errorf("fixture integer %q is outside %d..%d", raw, minimum, maximum)
	}
	return value, nil
}

func recordInvocation(mode string) (bool, error) {
	state := os.Getenv(stateEnvironment)
	evidence := os.Getenv(evidenceEnvironment)
	attemptID := os.Getenv(attemptEnvironment)
	if state == "" || evidence == "" || attemptID == "" {
		return false, errors.New("fixture requires fresh state, evidence, and attempt identity")
	}
	marker := filepath.Join(evidence, "attempt.marker.json")
	markerInfo, markerErr := os.Lstat(marker)
	markerPresent := markerErr == nil && markerInfo.Mode().IsRegular()
	identity := struct {
		AttemptID            string `json:"attempt_id"`
		Mode                 string `json:"mode"`
		PID                  int    `json:"pid"`
		ProcessGroupID       int    `json:"process_group_id"`
		MarkerPresentAtStart bool   `json:"marker_present_at_start"`
	}{AttemptID: attemptID, Mode: mode, PID: os.Getpid(), ProcessGroupID: processGroupID(), MarkerPresentAtStart: markerPresent}
	bytes, err := json.Marshal(identity)
	if err != nil {
		return false, err
	}
	path := filepath.Join(state, "invocation."+strconv.Itoa(os.Getpid())+".json")
	handle, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return false, err
	}
	if _, err := handle.Write(bytes); err != nil {
		_ = handle.Close()
		return false, err
	}
	return markerPresent, errors.Join(handle.Sync(), handle.Close())
}

func report(markerPresent bool) {
	keys := make([]string, 0)
	values := map[string]string{}
	for _, entry := range os.Environ() {
		name, value, present := strings.Cut(entry, "=")
		if !present {
			continue
		}
		keys = append(keys, name)
		values[name] = value
	}
	sort.Strings(keys)
	payload := struct {
		EnvironmentKeys      []string          `json:"environment_keys"`
		EnvironmentValues    map[string]string `json:"environment_values"`
		MarkerPresentAtStart bool              `json:"marker_present_at_start"`
		PID                  int               `json:"pid"`
		ProcessGroupID       int               `json:"process_group_id"`
	}{keys, values, markerPresent, os.Getpid(), processGroupID()}
	bytes, err := json.Marshal(payload)
	if err != nil {
		fatal(err)
	}
	_, _ = os.Stdout.Write(bytes)
}

func emit(file *os.File, count int, value byte) {
	chunk := []byte(strings.Repeat(string(value), 32<<10))
	for count > 0 {
		length := len(chunk)
		if count < length {
			length = count
		}
		if _, err := file.Write(chunk[:length]); err != nil {
			return
		}
		count -= length
	}
}

func spawnAndWait(mode string) {
	command := childCommand(mode)
	if err := command.Start(); err != nil {
		fatal(err)
	}
	if err := command.Wait(); err != nil {
		fatal(err)
	}
}

func spawnWithoutWait(mode string) {
	command := childCommand(mode)
	if err := command.Start(); err != nil {
		fatal(err)
	}
}

func spawnEscapeAndWait(mode string) {
	command := childCommand(mode)
	originalGroup := processGroupID()
	if originalGroup <= 0 {
		fatal(errors.New("escape fixture could not measure the original process group"))
	}
	command.Env = append(command.Env, originalGroupEnvironment+"="+strconv.Itoa(originalGroup))
	if err := command.Start(); err != nil {
		fatal(err)
	}
	if err := command.Wait(); err != nil {
		fatal(err)
	}
}

func childCommand(mode string) *exec.Cmd {
	executable, err := os.Executable()
	if err != nil {
		fatal(err)
	}
	return &exec.Cmd{
		Path:   executable,
		Args:   []string{filepath.Base(executable), "--mode", mode},
		Env:    append([]string(nil), os.Environ()...),
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

func spawnVersionPipeHolder() {
	executable, err := os.Executable()
	if err != nil {
		fatal(err)
	}
	command := &exec.Cmd{
		Path: executable, Args: []string{filepath.Base(executable), "--version-holder"},
		Env: append([]string(nil), os.Environ()...), Stdout: os.Stdout, Stderr: os.Stderr,
	}
	if err := command.Start(); err != nil {
		fatal(err)
	}
}

func writePID(name string) {
	state := os.Getenv(stateEnvironment)
	path := filepath.Join(state, name)
	if err := os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil {
		fatal(err)
	}
}

func writeEscapeIdentity(kind, name string) {
	originalGroup, err := strconv.Atoi(os.Getenv(originalGroupEnvironment))
	if err != nil || originalGroup <= 0 {
		fatal(errors.New("escape fixture received an invalid original process group"))
	}
	currentGroup := processGroupID()
	if currentGroup <= 0 || currentGroup == originalGroup || currentGroup != os.Getpid() {
		fatal(errors.New("escape fixture did not establish a distinct process group"))
	}
	payload := struct {
		Kind            string `json:"kind"`
		PID             int    `json:"pid"`
		ProcessGroupID  int    `json:"process_group_id"`
		OriginalGroupID int    `json:"original_process_group_id"`
	}{Kind: kind, PID: os.Getpid(), ProcessGroupID: currentGroup, OriginalGroupID: originalGroup}
	bytes, err := json.Marshal(payload)
	if err != nil {
		fatal(err)
	}
	state := os.Getenv(stateEnvironment)
	if err := os.WriteFile(filepath.Join(state, name), bytes, 0o600); err != nil {
		fatal(err)
	}
}

func waitForStateFile(name string) {
	path := filepath.Join(os.Getenv(stateEnvironment), name)
	deadline := time.Now().Add(2 * time.Second)
	for {
		if info, err := os.Lstat(path); err == nil && info.Mode().IsRegular() {
			return
		}
		if time.Now().After(deadline) {
			fatal(fmt.Errorf("timed out waiting for child state %s", name))
		}
		time.Sleep(time.Millisecond)
	}
}

func processGroupID() int {
	processGroupID, err := syscall.Getpgid(os.Getpid())
	if err != nil {
		return -1
	}
	return processGroupID
}

func waitForever() {
	for {
		time.Sleep(time.Hour)
	}
}

func fatal(err error) {
	_, _ = fmt.Fprintln(os.Stderr, err)
	os.Exit(70)
}
