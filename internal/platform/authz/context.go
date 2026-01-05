package authz

import "context"

type contextKey struct{}

// Actor represents the authenticated principal for authorization checks.
type Actor struct {
	UserID string
	Role   string
}

// ContextWithActor stores an Actor in the context.
func ContextWithActor(ctx context.Context, actor *Actor) context.Context {
	return context.WithValue(ctx, contextKey{}, actor)
}

// FromContext retrieves the Actor from the context, if present.
func FromContext(ctx context.Context) *Actor {
	actor, _ := ctx.Value(contextKey{}).(*Actor)
	return actor
}
