package hostepoch

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/nelsonwerd/countershape/internal/canon"
	"github.com/nelsonwerd/countershape/internal/domain"
)

const c3EpochLower = "12345678-90ab-cdef-8123-456789abcdef"

func TestC3HostEpochCanonicalMeasurement(t *testing.T) {
	calls := 0
	epoch, err := measureWithSource(context.Background(), func() ([]byte, error) {
		calls++
		return []byte("12345678-90AB-CDEF-8123-456789ABCDEF"), nil
	})
	if err != nil || !epoch.Valid() || calls != 2 {
		t.Fatalf("measurement failed or did not read exactly twice: calls=%d err=%v", calls, err)
	}
	want, err := canon.DigestBytes("DarwinBootSessionIdentity", []byte(c3EpochLower))
	if err != nil || epoch.Digest().String() != want.String() {
		t.Fatalf("typed digest differs: got %s want %s err=%v", epoch.Digest(), want.String(), err)
	}
}

func TestC3HostEpochMalformedSamplesRefuse(t *testing.T) {
	cases := map[string][]byte{
		"short":  []byte(c3EpochLower[:35]),
		"long":   []byte(c3EpochLower + "x"),
		"zero":   []byte("00000000-0000-0000-0000-000000000000"),
		"hyphen": func() []byte { value := []byte(c3EpochLower); value[8] = '_'; return value }(),
		"nonhex": func() []byte { value := []byte(c3EpochLower); value[0] = 'g'; return value }(),
		"nul":    func() []byte { value := []byte(c3EpochLower); value[35] = 0; return value }(),
	}
	for name, sample := range cases {
		t.Run(name, func(t *testing.T) {
			calls := 0
			_, err := measureWithSource(context.Background(), func() ([]byte, error) {
				calls++
				return append([]byte(nil), sample...), nil
			})
			if !IsCode(err, CodeInvalidMeasurement) || calls != 1 {
				t.Fatalf("malformed first sample did not fail closed: calls=%d err=%v", calls, err)
			}
		})
	}
}

func TestC3HostEpochFaultAndInstabilityRefuse(t *testing.T) {
	calls := 0
	_, err := measureWithSource(context.Background(), func() ([]byte, error) {
		calls++
		if calls == 2 {
			return nil, errors.New("read failed")
		}
		return []byte(c3EpochLower), nil
	})
	if !IsCode(err, CodeInvalidMeasurement) || calls != 2 {
		t.Fatalf("second read fault was not preserved: calls=%d err=%v", calls, err)
	}
	calls = 0
	_, err = measureWithSource(context.Background(), func() ([]byte, error) {
		calls++
		if calls == 2 {
			return []byte("22345678-90ab-cdef-8123-456789abcdef"), nil
		}
		return []byte(c3EpochLower), nil
	})
	if !IsCode(err, CodeUnstableMeasurement) || calls != 2 {
		t.Fatalf("sample disagreement did not refuse: calls=%d err=%v", calls, err)
	}
}

func TestC3HostEpochContextAndForgedAuthorityRefuse(t *testing.T) {
	closed, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	_, err := measureWithSource(closed, func() ([]byte, error) {
		calls++
		return []byte(c3EpochLower), nil
	})
	if !IsCode(err, CodeInvalidMeasurement) || calls != 0 {
		t.Fatalf("closed context reached the source: calls=%d err=%v", calls, err)
	}
	during, cancelDuring := context.WithCancel(context.Background())
	calls = 0
	_, err = measureWithSource(during, func() ([]byte, error) {
		calls++
		if calls == 2 {
			cancelDuring()
		}
		return []byte(c3EpochLower), nil
	})
	if !IsCode(err, CodeInvalidMeasurement) || calls != 2 {
		t.Fatalf("context closed by the second source issued authority: calls=%d err=%v", calls, err)
	}
	if (Epoch{}).Valid() || (Epoch{digest: domain.MustDigest("sha256:" + strings.Repeat("a", 64))}).Valid() {
		t.Fatal("zero or malformed epoch was valid")
	}
}

func TestC3HostEpochRevalidationRequiresSameFreshMeasurement(t *testing.T) {
	initial, err := measureWithSource(context.Background(), func() ([]byte, error) { return []byte(c3EpochLower), nil })
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := revalidateWithSource(context.Background(), initial, func() ([]byte, error) { return []byte(c3EpochLower), nil })
	if err != nil || !fresh.Valid() || fresh.Digest() != initial.Digest() {
		t.Fatalf("same epoch did not reopen: %v", err)
	}
	_, err = revalidateWithSource(context.Background(), initial, func() ([]byte, error) {
		return []byte("22345678-90ab-cdef-8123-456789abcdef"), nil
	})
	if !IsCode(err, CodeUnstableMeasurement) {
		t.Fatalf("different epoch reopened authority: %v", err)
	}
}

func TestC3HostEpochConcurrentMeasurementsNeverCache(t *testing.T) {
	const workers = 16
	var mu sync.Mutex
	calls := 0
	var wait sync.WaitGroup
	for range workers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, err := measureWithSource(context.Background(), func() ([]byte, error) {
				mu.Lock()
				calls++
				mu.Unlock()
				return []byte(c3EpochLower), nil
			})
			if err != nil {
				t.Errorf("measurement failed: %v", err)
			}
		}()
	}
	wait.Wait()
	if calls != workers*2 {
		t.Fatalf("measurements were cached: got %d source calls", calls)
	}
}
