package languageserver

import (
	"sync"
	"time"
)

// DocumentDebouncer provides debouncing for document change events.
// This reduces redundant parsing during rapid typing.
type DocumentDebouncer struct {
	timers    map[string]*time.Timer
	callbacks map[string]func()
	duration  time.Duration
	mu        sync.Mutex
}

// NewDocumentDebouncer creates a new debouncer with the specified duration.
func NewDocumentDebouncer(duration time.Duration) *DocumentDebouncer {
	return &DocumentDebouncer{
		timers:    make(map[string]*time.Timer),
		callbacks: make(map[string]func()),
		duration:  duration,
	}
}

// Debounce schedules a function to be called after the debounce duration.
// If called again for the same URI before the duration elapses,
// the previous call is cancelled and the timer resets.
func (d *DocumentDebouncer) Debounce(uri string, fn func()) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Cancel existing timer for this URI
	if timer, exists := d.timers[uri]; exists {
		timer.Stop()
	}

	// Store the callback for potential Flush() calls
	d.callbacks[uri] = fn

	// Schedule new timer
	d.timers[uri] = time.AfterFunc(d.duration, func() {
		d.mu.Lock()
		delete(d.timers, uri)
		delete(d.callbacks, uri)
		d.mu.Unlock()
		fn()
	})
}

// Cancel cancels any pending debounced call for the given URI.
func (d *DocumentDebouncer) Cancel(uri string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if timer, exists := d.timers[uri]; exists {
		timer.Stop()
		delete(d.timers, uri)
		delete(d.callbacks, uri)
	}
}

// CancelAll cancels all pending debounced calls.
func (d *DocumentDebouncer) CancelAll() {
	d.mu.Lock()
	defer d.mu.Unlock()

	for uri, timer := range d.timers {
		timer.Stop()
		delete(d.timers, uri)
		delete(d.callbacks, uri)
	}
}

// Flush immediately executes any pending debounced call for the given URI.
func (d *DocumentDebouncer) Flush(uri string) {
	d.mu.Lock()
	timer, exists := d.timers[uri]
	callback := d.callbacks[uri]
	if exists {
		timer.Stop()
		delete(d.timers, uri)
		delete(d.callbacks, uri)
	}
	d.mu.Unlock()

	if callback != nil {
		callback()
	}
}

// HasPending returns true if there's a pending debounced call for the given URI.
func (d *DocumentDebouncer) HasPending(uri string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, exists := d.timers[uri]
	return exists
}

// PendingCount returns the number of pending debounced calls.
func (d *DocumentDebouncer) PendingCount() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.timers)
}
