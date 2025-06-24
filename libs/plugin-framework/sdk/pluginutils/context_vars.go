package pluginutils

import "github.com/newstack-cloud/bluelink/libs/blueprint/core"

// SessionIDKey is the plain text key used to store the session ID in a Go context
// or as a part of the blueprint framework's context variables.
const SessionIDKey = "bluelink.sessionId"

// ContextKey provides a unique key type for Bluelink context variables.
type ContextKey string

func (c ContextKey) String() string {
	return "bluelink context key " + string(c)
}

var (
	// ContextSessionIDKey is the context key used to store the session ID
	// in a Go context.
	ContextSessionIDKey = core.ContextKey(SessionIDKey)
)
