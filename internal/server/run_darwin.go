//go:build darwin

package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

const testLaunchAuthority = "COUNTERSHAPE_STUDIO_TEST_LAUNCH"

type cliConfig struct {
	state      PresentationState
	noOpen     bool
	launchFile string
}

// RunCLI owns the local studio process lifetime. It returns only after the
// listener is terminal; browser credentials are never written to stdout.
func RunCLI(args []string, assets fs.FS, stdout, stderr io.Writer) int {
	config, err := parseCLI(args)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "countershape studio: %v\n", err)
		return 2
	}
	service, launch, err := Start(Config{Assets: assets, State: config.state})
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "countershape studio: local server refused to start")
		return 1
	}
	if config.launchFile != "" {
		if err := writePrivateLaunchFile(config.launchFile, launch); err != nil {
			closeStartedServer(service)
			_, _ = fmt.Fprintln(stderr, "countershape studio: test launch authority could not be published")
			return 1
		}
	}
	if !config.noOpen {
		if err := openBrowser(launch.URL); err != nil {
			closeStartedServer(service)
			_, _ = fmt.Fprintln(stderr, "countershape studio: browser launch failed")
			return 1
		}
	}
	_, _ = fmt.Fprintf(stdout, "Countershape Studio ready at %s (credential delivered outside HTTP request targets).\n", launch.Origin)

	signalContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	select {
	case err := <-service.done:
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "countershape studio: server ended unexpectedly")
			return 1
		}
		return 0
	case <-signalContext.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), ShutdownBudget())
		defer cancel()
		if err := service.Close(shutdownContext); err != nil {
			_, _ = fmt.Fprintln(stderr, "countershape studio: graceful shutdown failed")
			return 1
		}
		if err := <-service.done; err != nil {
			_, _ = fmt.Fprintln(stderr, "countershape studio: server did not end cleanly")
			return 1
		}
		return 0
	}
}

func parseCLI(args []string) (cliConfig, error) {
	result := cliConfig{state: StateDecisionReady}
	for index := 0; index < len(args); index++ {
		switch args[index] {
		case "--no-open":
			result.noOpen = true
		case "--seed-state":
			index++
			if index >= len(args) {
				return cliConfig{}, errors.New("--seed-state requires one value")
			}
			result.state = PresentationState(args[index])
			if _, present := stateDefinitions[result.state]; !present {
				return cliConfig{}, errors.New("--seed-state is outside the closed fixture roster")
			}
		case "--launch-file":
			index++
			if index >= len(args) {
				return cliConfig{}, errors.New("--launch-file requires one path")
			}
			if os.Getenv(testLaunchAuthority) != "1" || result.launchFile != "" {
				return cliConfig{}, errors.New("--launch-file is test-only authority")
			}
			result.launchFile = args[index]
		default:
			return cliConfig{}, fmt.Errorf("unknown argument %q", args[index])
		}
	}
	return result, nil
}

func writePrivateLaunchFile(name string, launch Launch) error {
	if name == "" || !filepath.IsAbs(name) {
		return errors.New("launch path must be absolute")
	}
	parent := filepath.Dir(name)
	info, err := os.Lstat(parent)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0o700 || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("launch parent must be one private real directory")
	}
	body, err := json.Marshal(struct {
		Origin string `json:"origin"`
		URL    string `json:"url"`
	}{launch.Origin, launch.URL})
	if err != nil {
		return err
	}
	body = append(body, '\n')
	file, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	written, writeErr := file.Write(body)
	closeErr := file.Close()
	if writeErr != nil || written != len(body) || closeErr != nil {
		_ = os.Remove(name)
		return errors.Join(writeErr, closeErr)
	}
	return nil
}

func openBrowser(target string) error {
	context, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(context, "/usr/bin/open", target)
	command.Stdin = nil
	command.Stdout = io.Discard
	command.Stderr = io.Discard
	return command.Run()
}

func closeStartedServer(service *Server) {
	context, cancel := context.WithTimeout(context.Background(), ShutdownBudget())
	defer cancel()
	_ = service.Close(context)
	select {
	case <-service.done:
	case <-context.Done():
	}
}
