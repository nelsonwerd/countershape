package server

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"mime"
	"net"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/nelsonwerd/countershape/internal/choice"
)

const (
	maxMutationBytes  = 1 << 20
	maxAssetBytes     = 2 << 20
	maxAssetTreeBytes = 8 << 20
	maxAssetFiles     = 128
	apiPrefix         = "/api/v1/"
)

// Start binds one authenticated studio to a literal IPv4 loopback listener.
// The returned fragment credential is intentionally absent from HTTP request
// targets, server logs, and static bootstrap bytes.
func Start(config Config) (*Server, Launch, error) {
	if config.Assets == nil {
		return nil, Launch{}, errors.New("studio assets are required")
	}
	if err := validateAssets(config.Assets); err != nil {
		return nil, Launch{}, err
	}
	if config.State == "" {
		config.State = StateDecisionReady
	}
	if config.Random == nil {
		config.Random = rand.Reader
	}
	state, err := newStudioState(config.State, config.Random)
	if err != nil {
		return nil, Launch{}, err
	}
	token, err := randomCapability(config.Random)
	if err != nil {
		return nil, Launch{}, errors.New("studio bearer authority could not be created")
	}
	csrf, err := randomCapability(config.Random)
	if err != nil {
		return nil, Launch{}, errors.New("studio mutation authority could not be created")
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, Launch{}, fmt.Errorf("studio loopback listener: %w", err)
	}
	address, ok := listener.Addr().(*net.TCPAddr)
	if !ok || address.IP.String() != "127.0.0.1" || address.Port <= 0 {
		_ = listener.Close()
		return nil, Launch{}, errors.New("studio listener escaped the literal IPv4 loopback boundary")
	}
	host := net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", address.Port))
	origin := "http://" + host
	server := &Server{
		origin: origin,
		host:   host,
		token:  token,
		csrf:   csrf,
		state:  state,
		done:   make(chan error, 1),
	}
	server.httpServer = &http.Server{
		Handler:           server.routes(config.Assets),
		ErrorLog:          log.New(io.Discard, "", 0),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    32 << 10,
	}
	go func() {
		err := server.httpServer.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		server.done <- err
		close(server.done)
	}()
	return server, Launch{Origin: origin, URL: origin + "/#access_token=" + token}, nil
}

func randomCapability(source io.Reader) (string, error) {
	body := make([]byte, 32)
	if _, err := io.ReadFull(source, body); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(body), nil
}

func (s *Server) routes(assets fs.FS) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		applySecurityHeaders(writer.Header())
		if request.Host != s.host {
			writeAPIError(writer, http.StatusMisdirectedRequest, "STUDIO_HOST_REFUSED", "the request Host is outside this studio listener")
			return
		}
		if request.URL.Path == "/api" || strings.HasPrefix(request.URL.Path, "/api/") {
			s.serveAPI(writer, request)
			return
		}
		s.serveStatic(writer, request, assets)
	})
}

func applySecurityHeaders(header http.Header) {
	header.Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; font-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'; object-src 'none'")
	header.Set("Cross-Origin-Opener-Policy", "same-origin")
	header.Set("Cross-Origin-Resource-Policy", "same-origin")
	header.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
	header.Set("Referrer-Policy", "no-referrer")
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("X-Frame-Options", "DENY")
	header.Set("Cache-Control", "no-store")
}

func (s *Server) serveStatic(writer http.ResponseWriter, request *http.Request, assets fs.FS) {
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		writeAPIError(writer, http.StatusMethodNotAllowed, "STUDIO_METHOD_REFUSED", "static bootstrap assets are read-only")
		return
	}
	clean := path.Clean("/" + request.URL.Path)
	if strings.Contains(clean, "..") {
		http.NotFound(writer, request)
		return
	}
	name := strings.TrimPrefix(clean, "/")
	if name == "" {
		name = "index.html"
	}
	body, err := fs.ReadFile(assets, name)
	if err != nil && path.Ext(name) == "" {
		name = "index.html"
		body, err = fs.ReadFile(assets, name)
	}
	if err != nil {
		http.NotFound(writer, request)
		return
	}
	contentType := mime.TypeByExtension(path.Ext(name))
	if contentType == "" {
		contentType = http.DetectContentType(body)
	}
	writer.Header().Set("Content-Type", contentType)
	writer.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
	writer.WriteHeader(http.StatusOK)
	if request.Method == http.MethodGet {
		_, _ = writer.Write(body)
	}
}

func (s *Server) serveAPI(writer http.ResponseWriter, request *http.Request) {
	if !s.authorized(request) {
		writer.Header().Set("WWW-Authenticate", `Bearer realm="countershape-studio"`)
		writeAPIError(writer, http.StatusUnauthorized, "STUDIO_AUTH_REFUSED", "one exact bearer credential is required")
		return
	}
	if request.URL.RawQuery != "" {
		writeAPIError(writer, http.StatusBadRequest, "STUDIO_QUERY_REFUSED", "studio API routes do not accept query parameters")
		return
	}
	switch {
	case request.Method == http.MethodGet && request.URL.Path == "/api/v1/session":
		writeJSON(writer, http.StatusOK, s.state.sessionResponse(s.csrf))
	case request.Method == http.MethodGet && request.URL.Path == "/api/v1/bench/"+StudyID:
		writeJSON(writer, http.StatusOK, s.state.benchResponse())
	case request.Method == http.MethodPost && request.URL.Path == "/api/v1/bench/"+StudyID+"/visit":
		s.mutate(writer, request, func(input MutationEnvelope) (BenchResponse, error) {
			return s.state.visit(input.ExpectedRevision, choice.ReviewSurface(input.Surface))
		})
	case request.Method == http.MethodPost && request.URL.Path == "/api/v1/bench/"+StudyID+"/propose":
		s.mutate(writer, request, func(input MutationEnvelope) (BenchResponse, error) {
			if input.Draft == nil {
				return BenchResponse{}, &APIError{Code: "STUDIO_DRAFT_SHAPE_REFUSED", Detail: "one explicit draft is required"}
			}
			return s.state.propose(input.ExpectedRevision, *input.Draft)
		})
	case request.Method == http.MethodPost && request.URL.Path == "/api/v1/bench/"+StudyID+"/reveal":
		s.mutate(writer, request, func(input MutationEnvelope) (BenchResponse, error) {
			return s.state.revealProvenance(input.ExpectedRevision)
		})
	case request.Method == http.MethodPost && request.URL.Path == "/api/v1/bench/"+StudyID+"/revise":
		s.mutate(writer, request, func(input MutationEnvelope) (BenchResponse, error) {
			if input.Draft == nil {
				return BenchResponse{}, &APIError{Code: "STUDIO_DRAFT_SHAPE_REFUSED", Detail: "one explicit draft is required"}
			}
			return s.state.revise(input.ExpectedRevision, *input.Draft, input.Rationale)
		})
	case request.Method == http.MethodPost && request.URL.Path == "/api/v1/bench/"+StudyID+"/finalize":
		s.mutate(writer, request, func(input MutationEnvelope) (BenchResponse, error) {
			return s.state.finalize(input.ExpectedRevision, input.Actor, input.Annotation)
		})
	default:
		writeAPIError(writer, http.StatusNotFound, "STUDIO_ROUTE_REFUSED", "the requested API route is not in the closed studio surface")
	}
}

func (s *Server) authorized(request *http.Request) bool {
	const prefix = "Bearer "
	values := request.Header.Values("Authorization")
	if len(values) != 1 {
		return false
	}
	value := values[0]
	return strings.HasPrefix(value, prefix) && len(value) == len(prefix)+len(s.token) &&
		hmac.Equal([]byte(value[len(prefix):]), []byte(s.token))
}

func (s *Server) mutate(writer http.ResponseWriter, request *http.Request, operation func(MutationEnvelope) (BenchResponse, error)) {
	origins := request.Header.Values("Origin")
	if len(origins) != 1 || origins[0] != s.origin {
		writeAPIError(writer, http.StatusForbidden, "STUDIO_ORIGIN_REFUSED", "mutation Origin must exactly match this studio")
		return
	}
	csrfValues := request.Header.Values("X-CSRF-Token")
	if len(csrfValues) != 1 || !hmac.Equal([]byte(csrfValues[0]), []byte(s.csrf)) {
		writeAPIError(writer, http.StatusForbidden, "STUDIO_CSRF_REFUSED", "mutation CSRF authority is absent or stale")
		return
	}
	contentTypes := request.Header.Values("Content-Type")
	if len(contentTypes) != 1 || contentTypes[0] != "application/json" {
		writeAPIError(writer, http.StatusUnsupportedMediaType, "STUDIO_CONTENT_TYPE_REFUSED", "mutations require exact application/json")
		return
	}
	request.Body = http.MaxBytesReader(writer, request.Body, maxMutationBytes)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	var input MutationEnvelope
	if err := decoder.Decode(&input); err != nil {
		writeAPIError(writer, http.StatusBadRequest, "STUDIO_JSON_REFUSED", "mutation JSON is malformed, oversized, or contains an unknown field")
		return
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		writeAPIError(writer, http.StatusBadRequest, "STUDIO_JSON_REFUSED", "mutation body must contain exactly one JSON value")
		return
	}
	response, err := operation(input)
	if err != nil {
		status := http.StatusBadRequest
		var apiError *APIError
		if errors.As(err, &apiError) && (apiError.Code == "STUDIO_STALE_REVISION" || apiError.Code == "STUDIO_STATE_IMMUTABLE") {
			status = http.StatusConflict
		}
		writeError(writer, status, err)
		return
	}
	writeJSON(writer, http.StatusOK, response)
}

func validateAssets(assets fs.FS) error {
	files := 0
	var total int64
	foundIndex := false
	err := fs.WalkDir(assets, ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if name == "." || entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > maxAssetBytes {
			return errors.New("studio asset inventory contains an invalid file")
		}
		files++
		total += info.Size()
		if files > maxAssetFiles || total > maxAssetTreeBytes {
			return errors.New("studio asset inventory exceeds the closed resource profile")
		}
		if name == "index.html" {
			foundIndex = true
		}
		return nil
	})
	if err != nil || !foundIndex || files == 0 {
		return errors.New("studio asset inventory is invalid")
	}
	return nil
}

func writeError(writer http.ResponseWriter, status int, err error) {
	var apiError *APIError
	if errors.As(err, &apiError) {
		writeJSON(writer, status, apiError)
		return
	}
	writeAPIError(writer, status, "STUDIO_REQUEST_REFUSED", "the studio refused this request")
}

func writeAPIError(writer http.ResponseWriter, status int, code, detail string) {
	writeJSON(writer, status, &APIError{Code: code, Detail: detail})
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(true)
	_ = encoder.Encode(value)
}

// ShutdownBudget is the maximum graceful close interval used by the command
// owner after it receives one terminal signal.
func ShutdownBudget() time.Duration { return serverShutdownBudget }

// WaitContext waits for terminal server state without leaking a goroutine.
func (s *Server) WaitContext(ctx context.Context) error {
	if s == nil || s.done == nil {
		return errInvalidServer
	}
	select {
	case err := <-s.done:
		return err
	case <-ctx.Done():
		return context.Cause(ctx)
	}
}
