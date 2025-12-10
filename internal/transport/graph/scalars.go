package graph

import (
	"fmt"
	"io"
	"strconv"

	"github.com/99designs/gqlgen/graphql"
	"github.com/google/uuid"
)

// MarshalUUID is a custom marshaller for the UUID scalar type.
// It converts a uuid.UUID object into a string for GraphQL responses.
func MarshalUUID(id uuid.UUID) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		_, _ = io.WriteString(w, strconv.Quote(id.String()))
	})
}

// UnmarshalUUID is a custom unmarshaller for the UUID scalar type.
// It parses a string from a GraphQL query into a uuid.UUID object.
func UnmarshalUUID(v any) (uuid.UUID, error) {
	str, ok := v.(string)
	if !ok {
		return uuid.UUID{}, fmt.Errorf("UUID must be a string")
	}

	id, err := uuid.Parse(str)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("failed to parse UUID: %w", err)
	}
	return id, nil
}
