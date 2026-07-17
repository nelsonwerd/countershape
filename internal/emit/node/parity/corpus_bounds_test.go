package parity

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestContractParityCorpusExactSizeBoundaries(t *testing.T) {
	line := sizedCorpusBoundaryLine(t, "bounds.line", MaxFrameBodyBytes)
	if len(line) != MaxFrameBodyBytes {
		t.Fatalf("boundary line = %d bytes; want %d", len(line), MaxFrameBodyBytes)
	}
	if _, err := parseCorpusLine(line); err != nil {
		t.Fatalf("exact-cap corpus line was rejected: %v", err)
	}
	if _, err := parseCorpusLine(sizedCorpusBoundaryLine(t, "bounds.line", MaxFrameBodyBytes+1)); err == nil {
		t.Fatal("cap-plus-one corpus line was accepted")
	}

	exact := sizedCorpusEnvelope(t, maxCorpusBytes)
	vectors, err := parseCorpus(exact)
	if err != nil || len(vectors) != maxCorpusVectors {
		t.Fatalf("exact 1 MiB corpus = %d vectors, %v; want %d", len(vectors), err, maxCorpusVectors)
	}
	if _, err := parseCorpus(sizedCorpusEnvelope(t, maxCorpusBytes+1)); err == nil {
		t.Fatal("1 MiB plus one corpus was accepted")
	}
}

func TestContractParityCorpusRejectsClosedSchemaDrift(t *testing.T) {
	tests := []struct {
		name string
		line []byte
	}{
		{"invalid ID", corpusSchemaLine("Bad-ID", `{"bytes_base64":"e30="}`, "CANONICALIZE_JSON", `["boundary"]`, `{"status":"OK","value":{"canonical_base64":"e30="}}`)},
		{"duplicate tags", corpusSchemaLine("bounds.tags.duplicate", `{"bytes_base64":"e30="}`, "CANONICALIZE_JSON", `["a","a"]`, `{"status":"OK","value":{"canonical_base64":"e30="}}`)},
		{"unsorted tags", corpusSchemaLine("bounds.tags.unsorted", `{"bytes_base64":"e30="}`, "CANONICALIZE_JSON", `["b","a"]`, `{"status":"OK","value":{"canonical_base64":"e30="}}`)},
		{"unknown operation", corpusSchemaLine("bounds.operation", `{"bytes_base64":"e30="}`, "UNKNOWN", `["boundary"]`, `{"status":"OK","value":{"canonical_base64":"e30="}}`)},
		{"wrong input roster", corpusSchemaLine("bounds.input", `{}`, "CANONICALIZE_JSON", `["boundary"]`, `{"status":"OK","value":{"canonical_base64":"e30="}}`)},
		{"wrong refusal roster", corpusSchemaLine("bounds.refusal", `{"bytes_base64":"e30="}`, "CANONICALIZE_JSON", `["boundary"]`, `{"code":"OUTPUT_LIMIT","status":"REFUSED"}`)},
		{"wrong OK roster", corpusSchemaLine("bounds.expected", `{"bytes_base64":"e30="}`, "CANONICALIZE_JSON", `["boundary"]`, `{"status":"OK","value":{"match":true}}`)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := parseCorpusLine(test.line); err == nil {
				t.Fatal("schema mutation was accepted")
			}
		})
	}

	lines := baseCorpusEnvelopeLines(t)
	lines[1] = sizedCorpusBoundaryLine(t, "bounds.000", len(lines[1]))
	if _, err := parseCorpus(joinCorpusLines(lines)); err == nil {
		t.Fatal("duplicate corpus ID was accepted")
	}
}

func sizedCorpusBoundaryLine(t testing.TB, id string, size int) []byte {
	t.Helper()
	base := corpusBoundaryLine(id, 1)
	descriptionBytes := size - len(base) + 1
	if descriptionBytes < 1 {
		t.Fatalf("line size %d is smaller than the closed boundary vector", size)
	}
	line := corpusBoundaryLine(id, descriptionBytes)
	if len(line) != size {
		t.Fatalf("sized line = %d; want %d", len(line), size)
	}
	return line
}

func sizedCorpusEnvelope(t testing.TB, size int) []byte {
	t.Helper()
	lines := baseCorpusEnvelopeLines(t)
	current := maxCorpusVectors
	for _, line := range lines {
		current += len(line)
	}
	remaining := size - current
	if remaining < 0 {
		t.Fatalf("corpus size %d is smaller than the closed base corpus", size)
	}
	for index := range lines {
		if remaining == 0 {
			break
		}
		capacity := MaxFrameBodyBytes - len(lines[index])
		add := min(remaining, capacity)
		lines[index] = corpusBoundaryLine(fmt.Sprintf("bounds.%03d", index), 1+add)
		remaining -= add
	}
	if remaining != 0 {
		t.Fatalf("corpus size %d cannot fit within %d bounded lines", size, maxCorpusVectors)
	}
	corpus := joinCorpusLines(lines)
	if len(corpus) != size {
		t.Fatalf("sized corpus = %d; want %d", len(corpus), size)
	}
	return corpus
}

func baseCorpusEnvelopeLines(t testing.TB) [][]byte {
	t.Helper()
	lines := make([][]byte, maxCorpusVectors)
	for index := range lines {
		lines[index] = corpusBoundaryLine(fmt.Sprintf("bounds.%03d", index), 1)
		if _, err := parseCorpusLine(lines[index]); err != nil {
			t.Fatalf("base boundary line %d is invalid: %v", index, err)
		}
	}
	return lines
}

func joinCorpusLines(lines [][]byte) []byte {
	var result bytes.Buffer
	for _, line := range lines {
		result.Write(line)
		result.WriteByte('\n')
	}
	return result.Bytes()
}

func corpusBoundaryLine(id string, descriptionBytes int) []byte {
	return corpusSchemaLine(
		id,
		`{"bytes_base64":"e30="}`,
		"CANONICALIZE_JSON",
		`["boundary"]`,
		`{"status":"OK","value":{"canonical_base64":"e30="}}`,
		strings.Repeat("a", descriptionBytes),
	)
}

func corpusSchemaLine(id, input, operation, tags, expected string, description ...string) []byte {
	text := "schema boundary vector"
	if len(description) == 1 {
		text = description[0]
	}
	return []byte(fmt.Sprintf(
		`{"description":%q,"expected":%s,"id":%q,"input":%s,"operation":%q,"tags":%s}`,
		text, expected, id, input, operation, tags,
	))
}
