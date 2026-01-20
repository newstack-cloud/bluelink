package languageserver

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDocumentDebouncer(t *testing.T) {
	d := NewDocumentDebouncer(100 * time.Millisecond)
	require.NotNil(t, d)
	assert.NotNil(t, d.timers)
	assert.Equal(t, 100*time.Millisecond, d.duration)
}

func TestDocumentDebouncer_Debounce(t *testing.T) {
	d := NewDocumentDebouncer(50 * time.Millisecond)
	var called atomic.Int32

	d.Debounce("test-uri", func() {
		called.Add(1)
	})

	assert.True(t, d.HasPending("test-uri"))
	assert.Equal(t, 1, d.PendingCount())

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, int32(1), called.Load())
	assert.False(t, d.HasPending("test-uri"))
	assert.Equal(t, 0, d.PendingCount())
}

func TestDocumentDebouncer_Debounce_CancelsPrevious(t *testing.T) {
	d := NewDocumentDebouncer(50 * time.Millisecond)
	var firstCalled, secondCalled atomic.Int32

	d.Debounce("test-uri", func() {
		firstCalled.Add(1)
	})

	time.Sleep(25 * time.Millisecond)

	d.Debounce("test-uri", func() {
		secondCalled.Add(1)
	})

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, int32(0), firstCalled.Load())
	assert.Equal(t, int32(1), secondCalled.Load())
}

func TestDocumentDebouncer_Debounce_MultipleURIs(t *testing.T) {
	d := NewDocumentDebouncer(50 * time.Millisecond)
	var uri1Called, uri2Called atomic.Int32

	d.Debounce("uri-1", func() {
		uri1Called.Add(1)
	})

	d.Debounce("uri-2", func() {
		uri2Called.Add(1)
	})

	assert.Equal(t, 2, d.PendingCount())

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, int32(1), uri1Called.Load())
	assert.Equal(t, int32(1), uri2Called.Load())
}

func TestDocumentDebouncer_Cancel(t *testing.T) {
	d := NewDocumentDebouncer(50 * time.Millisecond)
	var called atomic.Int32

	d.Debounce("test-uri", func() {
		called.Add(1)
	})

	d.Cancel("test-uri")

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, int32(0), called.Load())
	assert.False(t, d.HasPending("test-uri"))
}

func TestDocumentDebouncer_Cancel_NonExistent(t *testing.T) {
	d := NewDocumentDebouncer(50 * time.Millisecond)
	d.Cancel("non-existent")
	assert.Equal(t, 0, d.PendingCount())
}

func TestDocumentDebouncer_CancelAll(t *testing.T) {
	d := NewDocumentDebouncer(50 * time.Millisecond)
	var uri1Called, uri2Called atomic.Int32

	d.Debounce("uri-1", func() {
		uri1Called.Add(1)
	})
	d.Debounce("uri-2", func() {
		uri2Called.Add(1)
	})

	assert.Equal(t, 2, d.PendingCount())

	d.CancelAll()

	assert.Equal(t, 0, d.PendingCount())

	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, int32(0), uri1Called.Load())
	assert.Equal(t, int32(0), uri2Called.Load())
}

func TestDocumentDebouncer_HasPending(t *testing.T) {
	d := NewDocumentDebouncer(50 * time.Millisecond)

	assert.False(t, d.HasPending("test-uri"))

	d.Debounce("test-uri", func() {})

	assert.True(t, d.HasPending("test-uri"))
	assert.False(t, d.HasPending("other-uri"))
}

func TestDocumentDebouncer_Flush(t *testing.T) {
	d := NewDocumentDebouncer(50 * time.Millisecond)

	d.Debounce("test-uri", func() {})

	d.Flush("test-uri")

	assert.False(t, d.HasPending("test-uri"))
}
