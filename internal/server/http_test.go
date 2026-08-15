package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

func TestStudioHTTPRequiresExactLoopbackCapabilities(t *testing.T) {
	server, launch := startTestServer(t, StateDecisionReady)
	token := launchToken(t, launch)

	response := requestStudio(t, http.MethodGet, launch.Origin+"/", "", "", nil)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("static bootstrap status = %d", response.StatusCode)
	}
	body := readBody(t, response)
	if strings.Contains(body, token) || !strings.Contains(body, `id="root"`) {
		t.Fatalf("static bootstrap leaked authority or omitted root: %q", body)
	}
	if response.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("static bootstrap emitted a permissive CORS header")
	}
	if !strings.Contains(response.Header.Get("Content-Security-Policy"), "default-src 'none'") {
		t.Fatal("static bootstrap omitted restrictive CSP")
	}

	response = requestStudio(t, http.MethodGet, launch.Origin+"/api/v1/session", "", "", nil)
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated API status = %d", response.StatusCode)
	}
	_ = readBody(t, response)

	request, err := http.NewRequest(http.MethodGet, launch.Origin+"/api/v1/session", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Host = "localhost:1"
	request.Header.Set("Authorization", "Bearer "+token)
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusMisdirectedRequest {
		t.Fatalf("foreign Host status = %d", response.StatusCode)
	}
	_ = readBody(t, response)

	session := readSession(t, launch.Origin, token)
	mutation := MutationEnvelope{ExpectedRevision: session.RevisionDigest, Surface: string(choiceSurfaceOriginal())}
	response = requestStudio(t, http.MethodPost, launch.Origin+"/api/v1/bench/"+StudyID+"/visit", token, "", mutation)
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("mutation without Origin/CSRF status = %d", response.StatusCode)
	}
	_ = readBody(t, response)

	response = requestMutation(t, launch, token, session.CSRF, "/api/v1/bench/"+StudyID+"/visit", mutation)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("authorized mutation status = %d: %s", response.StatusCode, readBody(t, response))
	}
	var bench BenchResponse
	decodeBody(t, response, &bench)
	if bench.RevisionDigest == session.RevisionDigest {
		t.Fatal("successful mutation did not advance CAS authority")
	}

	response = requestMutation(t, launch, token, session.CSRF, "/api/v1/bench/"+StudyID+"/visit", mutation)
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("stale CAS status = %d", response.StatusCode)
	}
	var refusal APIError
	decodeBody(t, response, &refusal)
	if refusal.Code != "STUDIO_STALE_REVISION" {
		t.Fatalf("stale CAS code = %q", refusal.Code)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := server.Close(ctx); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := server.WaitContext(ctx); err != nil {
		t.Fatalf("WaitContext() error = %v", err)
	}
}

func TestStudioBlindPayloadExcludesChoicepointRevealAuthorityByBytes(t *testing.T) {
	_, launch := startTestServer(t, StateDecisionReady)
	token := launchToken(t, launch)
	response := requestStudio(t, http.MethodGet, launch.Origin+"/api/v1/bench/"+StudyID, token, "", nil)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("bench status = %d", response.StatusCode)
	}
	var bench BenchResponse
	decodeBody(t, response, &bench)
	if len(bench.Blind) == 0 || len(bench.Reveal) != 0 {
		t.Fatal("decision-ready fixture must expose only the blind package DTO")
	}
	record, err := loadSeedChoicepoint()
	if err != nil {
		t.Fatal(err)
	}
	var source struct {
		CandidateExecutionKeys []string `json:"candidate_execution_keys"`
		CandidateReveals       []struct {
			DisplayRef       string `json:"display_ref"`
			ProducerMetadata string `json:"producer_metadata"`
		} `json:"candidate_reveals"`
	}
	if err := json.Unmarshal(record.CanonicalBytes(), &source); err != nil {
		t.Fatal(err)
	}
	for _, key := range source.CandidateExecutionKeys {
		if key != "" && bytes.Contains(bench.Blind, []byte(key)) {
			t.Fatalf("blind DTO contains candidate execution key %q", key)
		}
	}
	for _, reveal := range source.CandidateReveals {
		for _, forbidden := range []string{reveal.DisplayRef, reveal.ProducerMetadata} {
			if forbidden != "" && bytes.Contains(bench.Blind, []byte(forbidden)) {
				t.Fatalf("blind DTO contains provenance bytes %q", forbidden)
			}
		}
	}
	for _, forbiddenKey := range []string{"candidate_execution_key", "display_ref", "producer_metadata", "support_count"} {
		if bytes.Contains(bench.Blind, []byte(forbiddenKey)) {
			t.Fatalf("blind DTO contains forbidden key %q", forbiddenKey)
		}
	}
}

func TestStudioRefusesAmbiguousHeadersQueriesAndUnsafeAssets(t *testing.T) {
	_, launch := startTestServer(t, StateDecisionReady)
	token := launchToken(t, launch)

	request, err := http.NewRequest(http.MethodGet, launch.Origin+"/api/v1/session?shadow=true", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("query status = %d", response.StatusCode)
	}
	_ = readBody(t, response)

	request, err = http.NewRequest(http.MethodGet, launch.Origin+"/api/v1/session", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Add("Authorization", "Bearer "+token)
	request.Header.Add("Authorization", "Bearer "+token)
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("duplicated bearer status = %d", response.StatusCode)
	}
	_ = readBody(t, response)

	unsafeAssets := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("ok"), Mode: 0o644},
		"linked.js":  &fstest.MapFile{Data: []byte("target.js"), Mode: os.ModeSymlink | 0o777},
	}
	if _, _, err := Start(Config{Assets: unsafeAssets, State: StateEmpty}); err == nil {
		t.Fatal("asset inventory admitted a symbolic link")
	}
}

func TestStudioHTTPRefusalMatrixIsClosedAndLeavesStateUnchanged(t *testing.T) {
	server, launch := startTestServer(t, StateDecisionReady)
	token := launchToken(t, launch)
	session := readSession(t, launch.Origin, token)
	route := launch.Origin + "/api/v1/bench/" + StudyID + "/visit"
	validBody, err := json.Marshal(MutationEnvelope{ExpectedRevision: session.RevisionDigest, Surface: string(choiceSurfaceOriginal())})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		method      string
		target      string
		bearer      string
		origin      string
		csrf        string
		contentType string
		body        []byte
		wantStatus  int
		wantCode    string
	}{
		{"wrong bearer", http.MethodGet, launch.Origin + "/api/v1/session", "wrong", "", "", "", nil, http.StatusUnauthorized, "STUDIO_AUTH_REFUSED"},
		{"null origin", http.MethodPost, route, token, "null", session.CSRF, "application/json", validBody, http.StatusForbidden, "STUDIO_ORIGIN_REFUSED"},
		{"foreign origin", http.MethodPost, route, token, "http://127.0.0.1:1", session.CSRF, "application/json", validBody, http.StatusForbidden, "STUDIO_ORIGIN_REFUSED"},
		{"wrong csrf", http.MethodPost, route, token, launch.Origin, "wrong", "application/json", validBody, http.StatusForbidden, "STUDIO_CSRF_REFUSED"},
		{"form post", http.MethodPost, route, token, launch.Origin, session.CSRF, "application/x-www-form-urlencoded", validBody, http.StatusUnsupportedMediaType, "STUDIO_CONTENT_TYPE_REFUSED"},
		{"parameterized json", http.MethodPost, route, token, launch.Origin, session.CSRF, "application/json; charset=utf-8", validBody, http.StatusUnsupportedMediaType, "STUDIO_CONTENT_TYPE_REFUSED"},
		{"unknown json field", http.MethodPost, route, token, launch.Origin, session.CSRF, "application/json", append(validBody[:len(validBody)-1], []byte(`,"shadow":true}`)...), http.StatusBadRequest, "STUDIO_JSON_REFUSED"},
		{"trailing json", http.MethodPost, route, token, launch.Origin, session.CSRF, "application/json", append(append([]byte(nil), validBody...), []byte(` {}`)...), http.StatusBadRequest, "STUDIO_JSON_REFUSED"},
		{"oversized json", http.MethodPost, route, token, launch.Origin, session.CSRF, "application/json", bytes.Repeat([]byte(" "), maxMutationBytes+1), http.StatusBadRequest, "STUDIO_JSON_REFUSED"},
		{"guessed study id", http.MethodGet, launch.Origin + "/api/v1/bench/guessed", token, "", "", "", nil, http.StatusNotFound, "STUDIO_ROUTE_REFUSED"},
		{"api options", http.MethodOptions, launch.Origin + "/api/v1/session", token, "", "", "", nil, http.StatusNotFound, "STUDIO_ROUTE_REFUSED"},
		{"static form", http.MethodPost, launch.Origin + "/", "", "", "", "application/x-www-form-urlencoded", []byte("x=y"), http.StatusMethodNotAllowed, "STUDIO_METHOD_REFUSED"},
	}
	for _, item := range tests {
		t.Run(item.name, func(t *testing.T) {
			request, requestErr := http.NewRequest(item.method, item.target, bytes.NewReader(item.body))
			if requestErr != nil {
				t.Fatal(requestErr)
			}
			if item.bearer != "" {
				request.Header.Set("Authorization", "Bearer "+item.bearer)
			}
			if item.origin != "" {
				request.Header.Set("Origin", item.origin)
			}
			if item.csrf != "" {
				request.Header.Set("X-CSRF-Token", item.csrf)
			}
			if item.contentType != "" {
				request.Header.Set("Content-Type", item.contentType)
			}
			response, requestErr := http.DefaultClient.Do(request)
			if requestErr != nil {
				t.Fatal(requestErr)
			}
			if response.StatusCode != item.wantStatus {
				t.Fatalf("status = %d", response.StatusCode)
			}
			var refusal APIError
			decodeBody(t, response, &refusal)
			if refusal.Code != item.wantCode || strings.Contains(refusal.Detail, token) || strings.Contains(refusal.Detail, session.CSRF) {
				t.Fatalf("refusal = %#v", refusal)
			}
			if response.Header.Get("Access-Control-Allow-Origin") != "" {
				t.Fatal("refusal emitted a CORS allowance")
			}
		})
	}

	for _, header := range []string{"Origin", "X-CSRF-Token", "Content-Type"} {
		t.Run("duplicate "+header, func(t *testing.T) {
			request, requestErr := http.NewRequest(http.MethodPost, route, bytes.NewReader(validBody))
			if requestErr != nil {
				t.Fatal(requestErr)
			}
			request.Header.Set("Authorization", "Bearer "+token)
			request.Header.Set("Origin", launch.Origin)
			request.Header.Set("X-CSRF-Token", session.CSRF)
			request.Header.Set("Content-Type", "application/json")
			request.Header.Add(header, request.Header.Get(header))
			response, requestErr := http.DefaultClient.Do(request)
			if requestErr != nil {
				t.Fatal(requestErr)
			}
			if response.StatusCode == http.StatusOK {
				t.Fatalf("duplicated %s was admitted", header)
			}
			_ = readBody(t, response)
		})
	}

	response := requestMutation(t, launch, token, session.CSRF, "/api/v1/bench/"+StudyID+"/visit", MutationEnvelope{ExpectedRevision: session.RevisionDigest, Surface: string(choiceSurfaceOriginal())})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("refusals changed state: status = %d, body = %s", response.StatusCode, readBody(t, response))
	}
	_ = readBody(t, response)
	if server.httpServer.ErrorLog == nil || server.httpServer.ErrorLog.Writer() != io.Discard {
		t.Fatal("HTTP protocol diagnostics are not bound to the nonlogging sink")
	}
}

func TestStudioHostileTextStaysEscapedJSONData(t *testing.T) {
	hostile := `<script>alert("secret")</script><img src=x onerror=alert(1)>` +
		"\x1b]8;;https://attacker.invalid\x07PASS\x1b]8;;\x07" +
		"\u202e../private/token\ufffd\n# PASS verified software"
	recorder := httptest.NewRecorder()
	writeJSON(recorder, http.StatusBadRequest, APIError{Code: "STUDIO_HOSTILE_FIXTURE", Detail: hostile})
	body := recorder.Body.String()
	if strings.Contains(body, "<script>") || strings.Contains(body, "</script>") || strings.Contains(body, "<img") || !strings.Contains(body, `\u003cscript\u003e`) {
		t.Fatalf("hostile JSON was not HTML escaped: %q", body)
	}
	var roundTrip APIError
	if err := json.Unmarshal(recorder.Body.Bytes(), &roundTrip); err != nil {
		t.Fatal(err)
	}
	if roundTrip.Detail != hostile {
		t.Fatalf("hostile data did not round trip exactly: %q", roundTrip.Detail)
	}
}

func startTestServer(t *testing.T, state PresentationState) (*Server, Launch) {
	t.Helper()
	assets := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte(`<!doctype html><div id="root"></div>`), Mode: 0o644},
	}
	server, launch, err := Start(Config{Assets: fs.FS(assets), State: state})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = server.Close(ctx)
	})
	return server, launch
}

func launchToken(t *testing.T, launch Launch) string {
	t.Helper()
	parsed, err := url.Parse(launch.URL)
	if err != nil {
		t.Fatal(err)
	}
	values, err := url.ParseQuery(parsed.Fragment)
	if err != nil {
		t.Fatal(err)
	}
	token := values.Get("access_token")
	if token == "" || strings.Contains(launch.Origin, token) {
		t.Fatal("launch fragment token is absent or present in the origin")
	}
	return token
}

func readSession(t *testing.T, origin, token string) SessionResponse {
	t.Helper()
	response := requestStudio(t, http.MethodGet, origin+"/api/v1/session", token, "", nil)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("session status = %d: %s", response.StatusCode, readBody(t, response))
	}
	var session SessionResponse
	decodeBody(t, response, &session)
	return session
}

func requestMutation(t *testing.T, launch Launch, token, csrf, route string, value any) *http.Response {
	t.Helper()
	return requestStudio(t, http.MethodPost, launch.Origin+route, token, csrf, value)
}

func requestStudio(t *testing.T, method, target, token, csrf string, value any) *http.Response {
	t.Helper()
	var body io.Reader
	if value != nil {
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequest(method, target, body)
	if err != nil {
		t.Fatal(err)
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	if value != nil {
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Origin", request.URL.Scheme+"://"+request.URL.Host)
	}
	if csrf != "" {
		request.Header.Set("X-CSRF-Token", csrf)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func decodeBody(t *testing.T, response *http.Response, target any) {
	t.Helper()
	defer response.Body.Close()
	decoder := json.NewDecoder(response.Body)
	if err := decoder.Decode(target); err != nil {
		t.Fatal(err)
	}
}

func readBody(t *testing.T, response *http.Response) string {
	t.Helper()
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

// Kept local so this test does not silently widen the production API.
func choiceSurfaceOriginal() string { return "ORIGINAL_WITNESS" }
