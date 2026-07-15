package world

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

// configuredLimit reports the immutable limit owned by this exact physical
// capture instance. Receipts must use this value, never a second copy of the
// request, so a miswired stdout/stderr constructor remains observable.
func (c *cappedCapture) configuredLimit() int64 { return c.limit }

func (c *cappedCapture) drain(reader io.Reader, done chan<- time.Time) {
	defer func() { done <- time.Now() }()
	buffer := make([]byte, 32<<10)
	for {
		count, err := reader.Read(buffer)
		if count > 0 {
			c.retain(buffer[:count])
		}
		if err != nil {
			if err != io.EOF {
				c.mu.Lock()
				if !c.frozen {
					c.drainErr = err
				}
				c.mu.Unlock()
			}
			return
		}
		if count == 0 {
			continue
		}
	}
}

func (c *cappedCapture) retain(data []byte) {
	c.mu.Lock()
	if c.frozen {
		c.mu.Unlock()
		return
	}
	remaining := c.limit - int64(c.buffer.Len())
	if remaining > int64(len(data)) {
		remaining = int64(len(data))
	}
	if remaining > 0 {
		_, _ = c.buffer.Write(data[:int(remaining)])
	}
	if int64(len(data)) > remaining {
		c.overflow = true
	}
	if int64(len(data)) > math.MaxInt64-c.total {
		c.total = math.MaxInt64
	} else {
		c.total += int64(len(data))
	}
	overflow := c.overflow
	c.mu.Unlock()
	if overflow {
		c.overflowOnce.Do(func() {
			select {
			case c.overflowC <- struct{}{}:
			default:
			}
		})
	}
}

func (c *cappedCapture) overflowed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.overflow
}

func (c *cappedCapture) freeze() {
	c.mu.Lock()
	c.frozen = true
	c.mu.Unlock()
}

func (c *cappedCapture) snapshot() (data []byte, total int64, overflow bool, drainErr error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]byte(nil), c.buffer.Bytes()...), c.total, c.overflow, c.drainErr
}
