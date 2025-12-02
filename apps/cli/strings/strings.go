package strings

// FromPointer returns the string value of a pointer to a string.
// If the pointer is nil, it returns an empty string.
func FromPointer(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
