package main

import "testing"

func TestExactFailClosedSelfTest(t *testing.T) {
	if err := selfTest(); err != nil {
		t.Fatal(err)
	}
}

func TestImplicitConstraintNames(t *testing.T) {
	tests := map[string]bool{
		"plain.go":                       false,
		"plain_test.go":                  false,
		"knowledge.go":                   false,
		"nacl.go":                        false,
		"worker_linux.go":                true,
		"worker_nacl.go":                 true,
		"worker_amd64.go":                true,
		"worker_linux_amd64.go":          true,
		"worker_linux_amd64_test.go":     true,
		"worker_linux.generated.go":      true,
		"worker_linux_amd64.coverage.go": true,
	}
	for name, want := range tests {
		if got := hasImplicitBuildSuffix(name); got != want {
			t.Errorf("hasImplicitBuildSuffix(%q) = %t, want %t", name, got, want)
		}
	}
	for suffix := range implicitGOOS {
		for _, name := range []string{
			"worker_" + suffix + ".go",
			"worker_" + suffix + "_test.go",
			"worker_" + suffix + ".generated.go",
		} {
			if !hasImplicitBuildSuffix(name) {
				t.Errorf("known GOOS suffix escaped filename policy: %q", name)
			}
		}
	}
	for suffix := range implicitGOARCH {
		for _, name := range []string{
			"worker_" + suffix + ".go",
			"worker_" + suffix + "_test.go",
			"worker_" + suffix + ".generated.go",
		} {
			if !hasImplicitBuildSuffix(name) {
				t.Errorf("known GOARCH suffix escaped filename policy: %q", name)
			}
		}
	}
}
