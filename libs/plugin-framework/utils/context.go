package utils

// ContextKey is a type used for keys in context.Context
// that hold plugin framework specific values.
type ContextKey string

func (c ContextKey) String() string {
	return "plugin framework context key " + string(c)
}

const (
	// ContextKeyLinkID is the key used to store the link ID in the context.
	ContextKeyLinkID ContextKey = "linkID"
)
