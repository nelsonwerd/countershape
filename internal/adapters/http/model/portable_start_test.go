package model

import (
	"bytes"
	"testing"
)

func TestPortableHTTPStartIsDistinctWithoutRewritingHistoricalBytes(t *testing.T) {
	legacy, err := NewHTTPStartSpec("fixture/server.mjs")
	if err != nil {
		t.Fatal(err)
	}
	portable, err := NewPortableHTTPStartSpec("fixture/server.mjs")
	if err != nil {
		t.Fatal(err)
	}
	if !legacy.Valid() || !portable.Valid() {
		t.Fatal("HTTP start authority did not retain sealed validity")
	}
	if legacy.Authority() != HTTPStartAuthorityV1 || legacy.Version() != HTTPStartVersionV1 ||
		portable.Authority() != HTTPPortableStartAuthorityV1 || portable.Version() != HTTPPortableStartVersionV1 {
		t.Fatal("HTTP start authority roster drifted")
	}
	const legacyStartDigest = "sha256:29342cbe04d7500a592af52a8a3fe651faf98ca7b3b4554f5df1a4968e6b7508"
	const legacyStartBytes = `{"authority":"NODE_CORE_INHERITED_LOOPBACK_LISTENER_V1","kind":"HTTPStartSpec","logical_argv":["node","fixture/server.mjs"],"schema_version":"countershape/v1","setup_argv":[],"version":"http-start/v1"}`
	if legacy.Digest().String() != legacyStartDigest || string(legacy.CanonicalBytes()) != legacyStartBytes {
		t.Fatal("portable constructor rewrote the pinned inherited-listener identity")
	}
	if legacy.Digest() == portable.Digest() || bytes.Equal(legacy.CanonicalBytes(), portable.CanonicalBytes()) {
		t.Fatal("portable child-bind start collapsed into inherited-listener identity")
	}

	legacyReady, err := NewHTTPReadinessContract()
	if err != nil {
		t.Fatal(err)
	}
	portableReady, err := NewPortableHTTPReadinessContract()
	if err != nil {
		t.Fatal(err)
	}
	const legacyReadyDigest = "sha256:7c3a0738acada6db0f76ce35e12784ace05d102b6edb1e63ee99c442b9fccfb7"
	const legacyReadyBytes = `{"eof_required":true,"http_probe":false,"kind":"HTTPReadinessContract","protocol":"ONE_BYTE_0X01_THEN_EOF_V1","schema_version":"countershape/v1","signal_name":"ready-byte","success_byte":1,"success_length":1,"version":"http-readiness/v1"}`
	if legacyReady.Digest().String() != legacyReadyDigest || string(legacyReady.CanonicalBytes()) != legacyReadyBytes {
		t.Fatal("portable readiness constructor rewrote the pinned one-byte readiness identity")
	}
	if portableReady.Protocol() != PortableReadinessProtocolV1 ||
		portableReady.SignalName() != PortableReadinessSignalNameV1 ||
		portableReady.Version() != HTTPPortableReadinessVersionV1 || portableReady.Digest() == legacyReady.Digest() {
		t.Fatal("portable readiness authority is not the distinct bounded port frame")
	}
	prefix, limit, eof, ok := portableReady.PortableFrameProfile()
	if !ok || prefix != PortableReadinessFramePrefix || limit != PortableReadinessFrameMax || !eof {
		t.Fatal("portable readiness contract did not expose its exact frame profile")
	}
	const portableStartDigest = "sha256:c5843dc36c2f5208b57ff5f00e1ed651b1d630b4c8e824eab9ac02cbeacd0b54"
	const portableStartBytes = `{"authority":"NODE_LOOPBACK_CHILD_BIND_PIPE_READY_V1","kind":"HTTPStartSpec","logical_argv":["node","fixture/server.mjs"],"schema_version":"countershape/v1","setup_argv":[],"version":"http-start/child-bind-pipe-ready/v1"}`
	if portable.Digest().String() != portableStartDigest || string(portable.CanonicalBytes()) != portableStartBytes {
		t.Fatal("portable child-bind start v1 identity drifted")
	}
	const portableReadyDigest = "sha256:aa1055fbfcd1d56b54617f82c97f796a71190dfb874e3fc480954190420bba70"
	const portableReadyBytes = `{"eof_required":true,"frame_max_bytes":32,"frame_prefix":"COUNTERSHAPE_READY_V1 ","http_probe":false,"kind":"HTTPReadinessContract","protocol":"ASCII_COUNTERSHAPE_READY_V1_SPACE_PORT_LF_THEN_EOF_V1","schema_version":"countershape/v1","signal_name":"ready-port-frame","version":"http-readiness/child-port-frame/v1"}`
	if portableReady.Digest().String() != portableReadyDigest || string(portableReady.CanonicalBytes()) != portableReadyBytes {
		t.Fatal("portable readiness frame v1 identity drifted")
	}
}

func TestPortableHTTPReadyPortFrameUsesOneStrictGrammar(t *testing.T) {
	for _, port := range []int{1, 9, 10, 8080, 65535} {
		frame, err := NewHTTPReadyPortFrame(port)
		if err != nil || !frame.Valid() || int(frame.Port()) != port {
			t.Fatalf("valid port %d refused: %v", port, err)
		}
		reopened, err := ParseHTTPReadyPortFrame(frame.CanonicalBytes())
		if err != nil || reopened.Port() != frame.Port() || !bytes.Equal(reopened.CanonicalBytes(), frame.CanonicalBytes()) {
			t.Fatalf("valid port %d did not round-trip: %v", port, err)
		}
		copyBytes := frame.CanonicalBytes()
		copyBytes[0] ^= 0xff
		if !frame.Valid() {
			t.Fatal("readiness frame exposed mutable canonical bytes")
		}
	}
	for _, invalid := range [][]byte{
		nil, {}, []byte(PortableReadinessFramePrefix + "\n"),
		[]byte(PortableReadinessFramePrefix + "0\n"),
		[]byte(PortableReadinessFramePrefix + "00\n"),
		[]byte(PortableReadinessFramePrefix + "01\n"),
		[]byte(PortableReadinessFramePrefix + "+1\n"),
		[]byte(PortableReadinessFramePrefix + "-1\n"),
		[]byte(PortableReadinessFramePrefix + " 1\n"),
		[]byte(PortableReadinessFramePrefix + "1 \n"),
		[]byte(PortableReadinessFramePrefix + "1\r\n"),
		[]byte(PortableReadinessFramePrefix + "1"),
		[]byte(PortableReadinessFramePrefix + "1\nextra"),
		[]byte(PortableReadinessFramePrefix + "1\n" + PortableReadinessFramePrefix + "2\n"),
		[]byte(PortableReadinessFramePrefix + "65536\n"),
		[]byte("COUNTERSHAPE_READY_V2 1\n"),
	} {
		if _, err := ParseHTTPReadyPortFrame(invalid); err == nil {
			t.Fatalf("invalid readiness frame accepted: %q", invalid)
		}
	}
	for _, port := range []int{-1, 0, 65536} {
		if _, err := NewHTTPReadyPortFrame(port); err == nil {
			t.Fatalf("invalid readiness port %d accepted", port)
		}
	}
}

func FuzzHTTPReadyPortFrameOnlyAcceptsExactCanonicalGrammar(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte(PortableReadinessFramePrefix + "1\n"),
		[]byte(PortableReadinessFramePrefix + "65535\n"),
		[]byte(PortableReadinessFramePrefix + "01\n"),
		[]byte("COUNTERSHAPE_READY_V2 1\n"),
		{0xff, 0x00, '\n'},
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, exact []byte) {
		original := append([]byte(nil), exact...)
		frame, err := ParseHTTPReadyPortFrame(exact)
		if err != nil {
			return
		}
		if !frame.Valid() || frame.Port() < 1 || !bytes.Equal(frame.CanonicalBytes(), original) {
			t.Fatalf("accepted readiness frame did not preserve exact canonical authority: port=%d bytes=%q", frame.Port(), frame.CanonicalBytes())
		}
		if len(exact) > 0 {
			exact[0] ^= 0xff
		}
		if !bytes.Equal(frame.CanonicalBytes(), original) {
			t.Fatal("accepted readiness frame retained caller-owned bytes")
		}
		reopened, reopenErr := ParseHTTPReadyPortFrame(frame.CanonicalBytes())
		if reopenErr != nil || reopened.Port() != frame.Port() || !bytes.Equal(reopened.CanonicalBytes(), original) {
			t.Fatalf("accepted readiness frame failed exact reopen: %v", reopenErr)
		}
	})
}

func TestPortableHTTPStartReadinessPairsSelectDistinctExecutionAuthority(t *testing.T) {
	legacyStart, _ := NewHTTPStartSpec("fixture/server.mjs")
	portableStart, _ := NewPortableHTTPStartSpec("fixture/server.mjs")
	legacyReady, _ := NewHTTPReadinessContract()
	portableReady, _ := NewPortableHTTPReadinessContract()
	if authority, ok := httpExecutionAuthorityFor(legacyStart, legacyReady); !ok || authority != HTTPExecutionAuthorityV1 {
		t.Fatal("legacy HTTP pair lost its pinned execution authority")
	}
	if authority, ok := httpExecutionAuthorityFor(portableStart, portableReady); !ok || authority != HTTPPortableExecutionAuthorityV1 {
		t.Fatal("portable HTTP pair lost its distinct execution authority")
	}
	if _, ok := httpExecutionAuthorityFor(legacyStart, portableReady); ok {
		t.Fatal("legacy start paired with portable readiness")
	}
	if _, ok := httpExecutionAuthorityFor(portableStart, legacyReady); ok {
		t.Fatal("portable start paired with legacy readiness")
	}
}

func TestPortableHTTPStartUsesExactBundleEntrypointGrammar(t *testing.T) {
	for _, invalid := range []string{
		".git/server.mjs", "fixture/.server.mjs", "fixture/sérver.mjs", "fixture/server name.mjs",
		"fixture/server\n.mjs", "../server.mjs", "/fixture/server.mjs", "-server.mjs",
		`fixture\server.mjs`, "fixture/server.py",
	} {
		if _, err := NewPortableHTTPStartSpec(invalid); err == nil {
			t.Fatalf("portable start admitted invalid bundle entrypoint %q", invalid)
		}
	}
	legacy, err := NewHTTPStartSpec("fixture/.server.mjs")
	if err != nil || !legacy.Valid() || legacy.Authority() != HTTPStartAuthorityV1 {
		t.Fatal("portable grammar silently narrowed the historical constructor")
	}
}
