package testutils

import (
	"bytes"
	"sync"
)

type safeBuffer struct {
	buffer bytes.Buffer
	m      sync.Mutex
}

// NewSaveBuffer creates a new buffer that can be used to capture output
// from TUI apps in headless mode in a thread-safe manner.
func NewSaveBuffer() *safeBuffer {
	return &safeBuffer{
		buffer: bytes.Buffer{},
		m:      sync.Mutex{},
	}
}

func (b *safeBuffer) Read(p []byte) (n int, err error) {
	b.m.Lock()
	defer b.m.Unlock()
	return b.buffer.Read(p)
}

func (b *safeBuffer) Write(p []byte) (n int, err error) {
	b.m.Lock()
	defer b.m.Unlock()
	return b.buffer.Write(p)
}

func (b *safeBuffer) String() string {
	b.m.Lock()
	defer b.m.Unlock()
	return b.buffer.String()
}
