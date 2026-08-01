//go:build darwin

package http

import (
	"bytes"
	"io"
	"math"
	"sync"
	"time"
)

type cappedCapture struct {
	limit        int64
	mu           sync.Mutex
	buffer       bytes.Buffer
	total        int64
	overflow     bool
	drainErr     error
	frozen       bool
	overflowOnce sync.Once
	overflowC    chan<- struct{}
}

func newCappedCapture(limit int64, overflowC chan<- struct{}) *cappedCapture {
	return &cappedCapture{limit: limit, overflowC: overflowC}
}

func (capture *cappedCapture) drain(reader io.Reader, done chan<- time.Time) {
	defer func() { done <- time.Now() }()
	buffer := make([]byte, 32<<10)
	for {
		count, err := reader.Read(buffer)
		if count > 0 {
			capture.retain(buffer[:count])
		}
		if err != nil {
			if err != io.EOF {
				capture.mu.Lock()
				if !capture.frozen {
					capture.drainErr = err
				}
				capture.mu.Unlock()
			}
			return
		}
	}
}

func (capture *cappedCapture) retain(data []byte) {
	capture.mu.Lock()
	if capture.frozen {
		capture.mu.Unlock()
		return
	}
	remaining := capture.limit - int64(capture.buffer.Len())
	if remaining > int64(len(data)) {
		remaining = int64(len(data))
	}
	if remaining > 0 {
		_, _ = capture.buffer.Write(data[:int(remaining)])
	}
	if int64(len(data)) > remaining {
		capture.overflow = true
	}
	if int64(len(data)) > math.MaxInt64-capture.total {
		capture.total = math.MaxInt64
	} else {
		capture.total += int64(len(data))
	}
	overflow := capture.overflow
	capture.mu.Unlock()
	if overflow {
		capture.overflowOnce.Do(func() {
			select {
			case capture.overflowC <- struct{}{}:
			default:
			}
		})
	}
}

func (capture *cappedCapture) overflowed() bool {
	capture.mu.Lock()
	defer capture.mu.Unlock()
	return capture.overflow
}

func (capture *cappedCapture) freeze() {
	capture.mu.Lock()
	capture.frozen = true
	capture.mu.Unlock()
}

func (capture *cappedCapture) snapshot() ([]byte, int64, bool, error) {
	capture.mu.Lock()
	defer capture.mu.Unlock()
	return append([]byte(nil), capture.buffer.Bytes()...), capture.total, capture.overflow, capture.drainErr
}
