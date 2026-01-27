package languageserver

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type DebounceSuite struct {
	suite.Suite
}

func (s *DebounceSuite) TestNewDocumentDebouncer() {
	d := NewDocumentDebouncer(100 * time.Millisecond)
	s.NotNil(d)
	s.NotNil(d.timers)
	s.Equal(100*time.Millisecond, d.duration)
}

func (s *DebounceSuite) TestDebounce() {
	d := NewDocumentDebouncer(50 * time.Millisecond)
	var called atomic.Int32

	d.Debounce("test-uri", func() {
		called.Add(1)
	})

	s.True(d.HasPending("test-uri"))
	s.Equal(1, d.PendingCount())

	time.Sleep(100 * time.Millisecond)

	s.Equal(int32(1), called.Load())
	s.False(d.HasPending("test-uri"))
	s.Equal(0, d.PendingCount())
}

func (s *DebounceSuite) TestDebounce_CancelsPrevious() {
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

	s.Equal(int32(0), firstCalled.Load())
	s.Equal(int32(1), secondCalled.Load())
}

func (s *DebounceSuite) TestDebounce_MultipleURIs() {
	d := NewDocumentDebouncer(50 * time.Millisecond)
	var uri1Called, uri2Called atomic.Int32

	d.Debounce("uri-1", func() {
		uri1Called.Add(1)
	})

	d.Debounce("uri-2", func() {
		uri2Called.Add(1)
	})

	s.Equal(2, d.PendingCount())

	time.Sleep(100 * time.Millisecond)

	s.Equal(int32(1), uri1Called.Load())
	s.Equal(int32(1), uri2Called.Load())
}

func (s *DebounceSuite) TestCancel() {
	d := NewDocumentDebouncer(50 * time.Millisecond)
	var called atomic.Int32

	d.Debounce("test-uri", func() {
		called.Add(1)
	})

	d.Cancel("test-uri")

	time.Sleep(100 * time.Millisecond)

	s.Equal(int32(0), called.Load())
	s.False(d.HasPending("test-uri"))
}

func (s *DebounceSuite) TestCancel_NonExistent() {
	d := NewDocumentDebouncer(50 * time.Millisecond)
	d.Cancel("non-existent")
	s.Equal(0, d.PendingCount())
}

func (s *DebounceSuite) TestCancelAll() {
	d := NewDocumentDebouncer(50 * time.Millisecond)
	var uri1Called, uri2Called atomic.Int32

	d.Debounce("uri-1", func() {
		uri1Called.Add(1)
	})
	d.Debounce("uri-2", func() {
		uri2Called.Add(1)
	})

	s.Equal(2, d.PendingCount())

	d.CancelAll()

	s.Equal(0, d.PendingCount())

	time.Sleep(100 * time.Millisecond)

	s.Equal(int32(0), uri1Called.Load())
	s.Equal(int32(0), uri2Called.Load())
}

func (s *DebounceSuite) TestHasPending() {
	d := NewDocumentDebouncer(50 * time.Millisecond)

	s.False(d.HasPending("test-uri"))

	d.Debounce("test-uri", func() {})

	s.True(d.HasPending("test-uri"))
	s.False(d.HasPending("other-uri"))
}

func (s *DebounceSuite) TestFlush() {
	d := NewDocumentDebouncer(50 * time.Millisecond)

	d.Debounce("test-uri", func() {})

	d.Flush("test-uri")

	s.False(d.HasPending("test-uri"))
}

func TestDebounceSuite(t *testing.T) {
	suite.Run(t, new(DebounceSuite))
}
