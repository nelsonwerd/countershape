package model

import (
	"bytes"
	"path"
	"strings"
	"unicode/utf8"

	"github.com/nelsonwerd/countershape/internal/domain"
)

const (
	maxPathBytes          = 4096
	maxQueryEntries       = 128
	maxQueryPartBytes     = 4096
	maxQueryAggregate     = 64 << 10
	maxHeaderEntries      = 128
	maxHeaderNameBytes    = 128
	maxHeaderValueBytes   = 8192
	maxHeaderAggregate    = 64 << 10
	maxRequestBodyBytes   = 1 << 20
	maxSeedFiles          = 256
	maxSeedPathBytes      = 1024
	maxSeedFileBytes      = 1 << 20
	maxSeedTotalBytes     = 16 << 20
	maxSeedTotalPathBytes = 256 << 10
)

type Presence string

const (
	PresenceAbsent  Presence = "ABSENT"
	PresencePresent Presence = "PRESENT"
)

func (p Presence) valid() bool { return p == PresenceAbsent || p == PresencePresent }

type HTTPMethod string

const (
	MethodGET    HTTPMethod = "GET"
	MethodPOST   HTTPMethod = "POST"
	MethodPUT    HTTPMethod = "PUT"
	MethodPATCH  HTTPMethod = "PATCH"
	MethodDELETE HTTPMethod = "DELETE"
)

func (m HTTPMethod) Valid() bool {
	switch m {
	case MethodGET, MethodPOST, MethodPUT, MethodPATCH, MethodDELETE:
		return true
	default:
		return false
	}
}

// HTTPQueryEntry preserves both order and the distinction between `?flag`
// and `?flag=`. Duplicate names are deliberately not normalized or joined.
type HTTPQueryEntry struct {
	name     string
	presence Presence
	value    string
}

func QueryFlag(name string) (HTTPQueryEntry, error) {
	if !validQueryPart(name) {
		return HTTPQueryEntry{}, refuse(CodeInvalidQuery, "query name is invalid or exceeds the v1 bound")
	}
	return HTTPQueryEntry{name: name, presence: PresenceAbsent}, nil
}

func QueryValue(name, value string) (HTTPQueryEntry, error) {
	if !validQueryPart(name) || !validQueryPart(value) {
		return HTTPQueryEntry{}, refuse(CodeInvalidQuery, "query name or value is invalid or exceeds the v1 bound")
	}
	return HTTPQueryEntry{name: name, presence: PresencePresent, value: value}, nil
}

func (e HTTPQueryEntry) Name() string       { return e.name }
func (e HTTPQueryEntry) Presence() Presence { return e.presence }
func (e HTTPQueryEntry) Present() bool      { return e.presence == PresencePresent }
func (e HTTPQueryEntry) Value() (string, bool) {
	return e.value, e.Present()
}
func (e HTTPQueryEntry) valid() bool {
	return validQueryPart(e.name) && e.presence.valid() &&
		(e.presence == PresencePresent || e.value == "") && validQueryPart(e.value)
}

// HTTPRequestHeader is one ordered multimap member. Names are normalized to
// lower-case ASCII at construction; values, including present-empty, remain
// exact bytes. Fields owned by the wire encoder are not constructible.
type HTTPRequestHeader struct {
	name  string
	value string
}

func NewRequestHeader(name, value string) (HTTPRequestHeader, error) {
	normalized := strings.ToLower(name)
	if !validHeaderName(normalized) || !validHeaderValue(value) {
		return HTTPRequestHeader{}, refuse(CodeInvalidHeader, "header is outside the closed ASCII HTTP/1.1 profile")
	}
	if forbiddenRequestHeader(normalized) {
		return HTTPRequestHeader{}, refuse(CodeForbiddenHeader, normalized)
	}
	return HTTPRequestHeader{name: normalized, value: value}, nil
}

func (h HTTPRequestHeader) Name() string  { return h.name }
func (h HTTPRequestHeader) Value() string { return h.value }
func (h HTTPRequestHeader) valid() bool {
	return validHeaderName(h.name) && h.name == strings.ToLower(h.name) &&
		validHeaderValue(h.value) && !forbiddenRequestHeader(h.name)
}

func forbiddenRequestHeader(name string) bool {
	switch name {
	case "host", "connection", "content-length", "transfer-encoding", "content-encoding",
		"expect", "cookie", "authorization", "proxy-authorization", "accept-encoding", "upgrade", "te":
		return true
	default:
		// MUTATION_ANCHOR: http-wire-refuses-proxy-and-ambient-policy-headers
		return strings.HasPrefix(name, "proxy-")
	}
}

type HTTPBody struct {
	presence Presence
	bytes    []byte
}

func AbsentBody() HTTPBody { return HTTPBody{presence: PresenceAbsent, bytes: []byte{}} }

func PresentBody(value []byte) (HTTPBody, error) {
	if len(value) > maxRequestBodyBytes {
		return HTTPBody{}, refuse(CodeInputLimit, "request body exceeds the v1 byte ceiling")
	}
	return HTTPBody{presence: PresencePresent, bytes: append([]byte(nil), value...)}, nil
}

func (b HTTPBody) Presence() Presence { return b.presence }
func (b HTTPBody) Present() bool      { return b.presence == PresencePresent }
func (b HTTPBody) Bytes() []byte      { return append([]byte(nil), b.bytes...) }
func (b HTTPBody) valid() bool {
	return b.presence.valid() && len(b.bytes) <= maxRequestBodyBytes &&
		(b.presence == PresencePresent || len(b.bytes) == 0)
}

type SeedMode string

const SeedMode0644 SeedMode = "100644"

type HTTPSeedFile struct {
	path     string
	contents []byte
	mode     SeedMode
}

func NewSeedFile(relativePath string, contents []byte, mode SeedMode) (HTTPSeedFile, error) {
	if err := validateSeedPath(relativePath); err != nil {
		return HTTPSeedFile{}, err
	}
	if mode != SeedMode0644 {
		return HTTPSeedFile{}, refuse(CodeInvalidSeed, "seed must use the sole regular-file mode")
	}
	if len(contents) > maxSeedFileBytes {
		return HTTPSeedFile{}, refuse(CodeInputLimit, "seed exceeds the per-file byte ceiling")
	}
	return HTTPSeedFile{path: relativePath, contents: append([]byte(nil), contents...), mode: mode}, nil
}

func (f HTTPSeedFile) Path() string     { return f.path }
func (f HTTPSeedFile) Contents() []byte { return append([]byte(nil), f.contents...) }
func (f HTTPSeedFile) Mode() SeedMode   { return f.mode }
func (f HTTPSeedFile) valid() bool {
	return validateSeedPath(f.path) == nil && f.mode == SeedMode0644 && len(f.contents) <= maxSeedFileBytes
}

type HTTPStimulusConfig struct {
	Method  HTTPMethod
	Path    string
	Query   []HTTPQueryEntry
	Headers []HTTPRequestHeader
	Body    HTTPBody
	Seeds   []HTTPSeedFile
}

type HTTPStimulus struct {
	digest                         domain.Digest
	canonicalBytes                 []byte
	executionPayloadDigest         domain.Digest
	executionPayloadCanonicalBytes []byte
	method                         HTTPMethod
	path                           string
	query                          []HTTPQueryEntry
	headers                        []HTTPRequestHeader
	body                           HTTPBody
	seeds                          []HTTPSeedFile
	measure                        HTTPStimulusMeasure
}

type queryIdentity struct {
	Name     string `json:"name"`
	Presence string `json:"value_presence"`
	Value    string `json:"value"`
}

type headerIdentity struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type bodyIdentity struct {
	Presence   string `json:"presence"`
	ByteLength int    `json:"byte_length"`
	ByteDigest string `json:"byte_digest"`
}

type seedIdentity struct {
	Path           string `json:"path"`
	Mode           string `json:"mode"`
	ContentsBytes  int    `json:"contents_bytes"`
	ContentsDigest string `json:"contents_digest"`
}

type executionPayloadIdentity struct {
	SchemaVersion string           `json:"schema_version"`
	Kind          string           `json:"kind"`
	Method        string           `json:"method"`
	Path          string           `json:"path"`
	Query         []queryIdentity  `json:"ordered_query_multimap"`
	Headers       []headerIdentity `json:"ordered_request_header_multimap"`
	Body          bodyIdentity     `json:"body"`
	Seeds         []seedIdentity   `json:"declarative_seed_files"`
}

type stimulusIdentity struct {
	SchemaVersion          string           `json:"schema_version"`
	Kind                   string           `json:"kind"`
	ExecutionPayloadDigest string           `json:"execution_payload_digest"`
	Method                 string           `json:"method"`
	Path                   string           `json:"path"`
	Query                  []queryIdentity  `json:"ordered_query_multimap"`
	Headers                []headerIdentity `json:"ordered_request_header_multimap"`
	Body                   bodyIdentity     `json:"body"`
	Seeds                  []seedIdentity   `json:"declarative_seed_files"`
	Measure                measureIdentity  `json:"measure"`
}

func NewHTTPStimulus(config HTTPStimulusConfig) (HTTPStimulus, error) {
	if !config.Method.Valid() {
		return HTTPStimulus{}, refuse(CodeInvalidMethod, string(config.Method))
	}
	if !validOriginPath(config.Path) {
		return HTTPStimulus{}, refuse(CodeInvalidPath, "path must be one bounded origin-form path without query or fragment")
	}
	if !config.Body.valid() {
		return HTTPStimulus{}, refuse(CodeInvalidBody, "body must be tagged absent or present within the byte ceiling")
	}
	query, err := validateQuery(config.Query)
	if err != nil {
		return HTTPStimulus{}, err
	}
	headers, err := validateHeaders(config.Headers)
	if err != nil {
		return HTTPStimulus{}, err
	}
	seeds, err := validateSeeds(config.Seeds)
	if err != nil {
		return HTTPStimulus{}, err
	}
	bodyID, err := bodyIdentityOf(config.Body)
	if err != nil {
		return HTTPStimulus{}, err
	}
	seedIDs, err := seedIdentities(seeds)
	if err != nil {
		return HTTPStimulus{}, err
	}
	queryIDs := queryIdentities(query)
	headerIDs := headerIdentities(headers)
	payload := executionPayloadIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "HTTPExecutionPayload", Method: string(config.Method), Path: config.Path,
		Query: queryIDs, Headers: headerIDs, Body: bodyID, Seeds: seedIDs,
	}
	payloadDigest, payloadBytes, err := digestTyped("HTTPExecutionPayload", payload)
	if err != nil {
		return HTTPStimulus{}, err
	}
	body := cloneBody(config.Body)
	measure := measureOf(query, headers, body, seeds)
	identity := stimulusIdentity{
		SchemaVersion: domain.SchemaVersion, Kind: "HTTPStimulus", ExecutionPayloadDigest: payloadDigest.String(),
		Method: string(config.Method), Path: config.Path, Query: queryIDs, Headers: headerIDs, Body: bodyID,
		Seeds: seedIDs, Measure: measure.identity(),
	}
	digest, canonicalBytes, err := digestTyped("HTTPStimulus", identity)
	if err != nil {
		return HTTPStimulus{}, err
	}
	return HTTPStimulus{
		digest: digest, canonicalBytes: canonicalBytes, executionPayloadDigest: payloadDigest,
		executionPayloadCanonicalBytes: payloadBytes, method: config.Method, path: config.Path,
		query: query, headers: headers, body: body, seeds: seeds, measure: measure,
	}, nil
}

func (s HTTPStimulus) Valid() bool {
	rebuilt, err := NewHTTPStimulus(HTTPStimulusConfig{
		Method: s.method, Path: s.path, Query: s.query, Headers: s.headers, Body: s.body, Seeds: s.seeds,
	})
	return err == nil && rebuilt.digest == s.digest && rebuilt.executionPayloadDigest == s.executionPayloadDigest &&
		bytes.Equal(rebuilt.canonicalBytes, s.canonicalBytes) &&
		bytes.Equal(rebuilt.executionPayloadCanonicalBytes, s.executionPayloadCanonicalBytes)
}

func (s HTTPStimulus) Digest() domain.Digest                 { return s.digest }
func (s HTTPStimulus) CanonicalBytes() []byte                { return append([]byte(nil), s.canonicalBytes...) }
func (s HTTPStimulus) ExecutionPayloadDigest() domain.Digest { return s.executionPayloadDigest }
func (s HTTPStimulus) ExecutionPayloadCanonicalBytes() []byte {
	return append([]byte(nil), s.executionPayloadCanonicalBytes...)
}
func (s HTTPStimulus) Method() HTTPMethod      { return s.method }
func (s HTTPStimulus) Path() string            { return s.path }
func (s HTTPStimulus) Query() []HTTPQueryEntry { return append([]HTTPQueryEntry(nil), s.query...) }
func (s HTTPStimulus) Headers() []HTTPRequestHeader {
	return append([]HTTPRequestHeader(nil), s.headers...)
}
func (s HTTPStimulus) Body() HTTPBody               { return cloneBody(s.body) }
func (s HTTPStimulus) Seeds() []HTTPSeedFile        { return cloneSeeds(s.seeds) }
func (s HTTPStimulus) Measure() HTTPStimulusMeasure { return s.measure }

func validOriginPath(value string) bool {
	if !utf8.ValidString(value) || len(value) < 1 || len(value) > maxPathBytes || value[0] != '/' ||
		strings.ContainsAny(value, "?#\\\x00") || strings.Contains(value, "//") {
		return false
	}
	for _, r := range value {
		if r <= 0x20 || r >= 0x7f {
			return false
		}
	}
	return true
}

func validQueryPart(value string) bool {
	if !utf8.ValidString(value) || len(value) > maxQueryPartBytes || strings.ContainsRune(value, '\x00') {
		return false
	}
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}

func validateQuery(input []HTTPQueryEntry) ([]HTTPQueryEntry, error) {
	if len(input) > maxQueryEntries {
		return nil, refuse(CodeInputLimit, "query exceeds the entry ceiling")
	}
	result := append([]HTTPQueryEntry(nil), input...)
	total := 0
	for _, entry := range result {
		if !entry.valid() {
			return nil, refuse(CodeInvalidQuery, "query contains an invalid ordered member")
		}
		total += len(entry.name) + len(entry.value)
	}
	if total > maxQueryAggregate {
		return nil, refuse(CodeInputLimit, "query exceeds the aggregate byte ceiling")
	}
	return result, nil
}

func validateHeaders(input []HTTPRequestHeader) ([]HTTPRequestHeader, error) {
	if len(input) > maxHeaderEntries {
		return nil, refuse(CodeInputLimit, "headers exceed the entry ceiling")
	}
	result := append([]HTTPRequestHeader(nil), input...)
	total := 0
	for _, header := range result {
		if !header.valid() {
			return nil, refuse(CodeInvalidHeader, "headers contain an invalid ordered member")
		}
		total += len(header.name) + len(header.value)
	}
	if total > maxHeaderAggregate {
		return nil, refuse(CodeInputLimit, "headers exceed the aggregate byte ceiling")
	}
	return result, nil
}

func validHeaderName(value string) bool {
	if len(value) < 1 || len(value) > maxHeaderNameBytes {
		return false
	}
	for index := 0; index < len(value); index++ {
		b := value[index]
		if (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') || strings.ContainsRune("!#$%&'*+-.^_`|~", rune(b)) {
			continue
		}
		return false
	}
	return true
}

func validHeaderValue(value string) bool {
	if len(value) > maxHeaderValueBytes || !utf8.ValidString(value) {
		return false
	}
	for index := 0; index < len(value); index++ {
		if value[index] < 0x20 || value[index] > 0x7e {
			return false
		}
	}
	return true
}

func validateSeedPath(value string) error {
	if !utf8.ValidString(value) || len(value) < 1 || len(value) > maxSeedPathBytes ||
		strings.HasPrefix(value, "/") || strings.Contains(value, "\\") || path.Clean(value) != value || value == "." {
		return refuse(CodeInvalidSeed, "seed path must be one clean relative slash path")
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == "" || segment == "." || segment == ".." || strings.EqualFold(segment, ".git") {
			return refuse(CodeInvalidSeed, "seed path contains a reserved or traversal segment")
		}
	}
	return nil
}

func validateSeeds(input []HTTPSeedFile) ([]HTTPSeedFile, error) {
	if len(input) > maxSeedFiles {
		return nil, refuse(CodeInputLimit, "seed set exceeds the file ceiling")
	}
	result := cloneSeeds(input)
	totalBytes := 0
	totalPathBytes := 0
	for index, seed := range result {
		if !seed.valid() {
			return nil, refuse(CodeInvalidSeed, "seed set contains an invalid regular file")
		}
		if index > 0 && result[index-1].path >= seed.path {
			return nil, refuse(CodeSeedOrder, "seed files must be supplied in strict path order")
		}
		totalBytes += len(seed.contents)
		totalPathBytes += len(seed.path)
	}
	if totalBytes > maxSeedTotalBytes || totalPathBytes > maxSeedTotalPathBytes {
		return nil, refuse(CodeInputLimit, "seed set exceeds an aggregate byte ceiling")
	}
	for left := 0; left < len(result); left++ {
		leftFolded := strings.ToLower(result[left].path)
		for right := left + 1; right < len(result); right++ {
			rightFolded := strings.ToLower(result[right].path)
			if leftFolded == rightFolded || strings.HasPrefix(rightFolded, leftFolded+"/") ||
				strings.HasPrefix(leftFolded, rightFolded+"/") {
				return nil, refuse(CodeSeedCollision, "seed files collide by path or file/directory prefix")
			}
		}
	}
	return result, nil
}

func queryIdentities(input []HTTPQueryEntry) []queryIdentity {
	result := make([]queryIdentity, len(input))
	for index, entry := range input {
		result[index] = queryIdentity{Name: entry.name, Presence: string(entry.presence), Value: entry.value}
	}
	return result
}

func headerIdentities(input []HTTPRequestHeader) []headerIdentity {
	result := make([]headerIdentity, len(input))
	for index, header := range input {
		result[index] = headerIdentity{Name: header.name, Value: header.value}
	}
	return result
}

func bodyIdentityOf(body HTTPBody) (bodyIdentity, error) {
	result := bodyIdentity{Presence: string(body.presence), ByteLength: len(body.bytes)}
	if body.presence == PresencePresent {
		digest, err := digestExactBytes("HTTPRequestBodyBytes", body.bytes)
		if err != nil {
			return bodyIdentity{}, err
		}
		result.ByteDigest = digest.String()
	}
	return result, nil
}

func seedIdentities(input []HTTPSeedFile) ([]seedIdentity, error) {
	result := make([]seedIdentity, len(input))
	for index, seed := range input {
		digest, err := digestExactBytes("HTTPSeedContents", seed.contents)
		if err != nil {
			return nil, err
		}
		result[index] = seedIdentity{Path: seed.path, Mode: string(seed.mode), ContentsBytes: len(seed.contents), ContentsDigest: digest.String()}
	}
	return result, nil
}

func cloneBody(input HTTPBody) HTTPBody {
	return HTTPBody{presence: input.presence, bytes: append([]byte(nil), input.bytes...)}
}

func cloneSeeds(input []HTTPSeedFile) []HTTPSeedFile {
	result := make([]HTTPSeedFile, len(input))
	for index, seed := range input {
		result[index] = HTTPSeedFile{path: seed.path, mode: seed.mode, contents: append([]byte(nil), seed.contents...)}
	}
	return result
}
