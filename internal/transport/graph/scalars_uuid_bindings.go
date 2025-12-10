package graph

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/google/uuid"
	"github.com/vektah/gqlparser/v2/ast"
)

func (ec *executionContext) unmarshalInputUUID(ctx context.Context, v any) (uuid.UUID, error) {
	return UnmarshalUUID(v)
}

func (ec *executionContext) _UUID(ctx context.Context, sel ast.SelectionSet, v *uuid.UUID) graphql.Marshaler {
	if v == nil {
		if !graphql.HasFieldError(ctx, graphql.GetFieldContext(ctx)) {
			graphql.AddErrorf(ctx, "the requested element is null which the schema does not allow")
		}
		return graphql.Null
	}
	_ = sel
	return MarshalUUID(*v)
}
