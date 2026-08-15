// Package server owns Countershape's authenticated loopback presentation edge.
// It projects package-owned choice facts for one local browser and never mints
// canonical product identity in JavaScript.
package server

import (
	"context"
	"io"
	"io/fs"
	"net/http"
	"time"
)

const (
	SchemaVersion = "countershape/studio/v1"
	StudyID       = "seed-cli-precedence"

	TrustedCodeWarning = "Countershape will run trusted repository code with your user permissions and host network access. The temporary directory is for repeatability, not security isolation."
)

// PresentationState selects one deterministic U8 state fixture. Only
// decision-ready and the package-produced ruling states are mutable. The other
// values exercise honest presentation without claiming an implemented durable
// workflow behind that state.
type PresentationState string

const (
	StateEmpty             PresentationState = "empty"
	StatePreparing         PresentationState = "preparing"
	StateActive            PresentationState = "active"
	StatePartial           PresentationState = "partial"
	StateError             PresentationState = "error"
	StateCompleted         PresentationState = "completed"
	StateUnstable          PresentationState = "unstable"
	StateUncomparable      PresentationState = "uncomparable"
	StateIncomplete        PresentationState = "incomplete"
	StateDiscovered        PresentationState = "discovered"
	StateDecisionReady     PresentationState = "decision-ready"
	StatePredicateEditing  PresentationState = "predicate-editing"
	StateIdentityReveal    PresentationState = "identity-reveal"
	StateResolved          PresentationState = "resolved"
	StateRejectAllResolved PresentationState = "reject-all-resolved"
	StateDeferred          PresentationState = "deferred"
	StateStale             PresentationState = "stale"
	StateInvalidated       PresentationState = "invalidated"
)

// Config supplies the embedded browser assets and an optional deterministic
// state fixture. Random defaults to crypto/rand.Reader and must not be replaced
// outside tests.
type Config struct {
	Assets fs.FS
	State  PresentationState
	Random io.Reader
}

// Launch is a process-local capability returned only to the caller that starts
// the server. URL contains the fragment credential and must never be logged.
type Launch struct {
	Origin string
	URL    string
}

// Server owns one listener, one browser credential, and one decision session.
type Server struct {
	origin     string
	host       string
	token      string
	csrf       string
	httpServer *http.Server
	state      *studioState
	done       chan error
}

func (s *Server) Origin() string {
	if s == nil {
		return ""
	}
	return s.origin
}

func (s *Server) Wait() error {
	if s == nil || s.done == nil {
		return errInvalidServer
	}
	return <-s.done
}

func (s *Server) Close(ctx context.Context) error {
	if s == nil || s.httpServer == nil {
		return errInvalidServer
	}
	return s.httpServer.Shutdown(ctx)
}

const serverShutdownBudget = 5 * time.Second
