package utils

import (
	"github.com/google/uuid"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

type uuidv7Generator struct{}

// NewUUIDv7Generator creates a new blueprint framework
// ID generator that uses UUIDv7.
// This is useful for generating IDs that can be sorted
// based on time of creation.
func NewUUIDv7Generator() core.IDGenerator {
	return &uuidv7Generator{}
}

func (g *uuidv7Generator) GenerateID() (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}

	return id.String(), nil
}
