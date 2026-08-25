package mcpserver

import "context"

type Actor struct {
	UserID        uint
	Authenticated bool
}

type actorContextKey struct{}

func ActorFromContext(ctx context.Context) Actor {
	if ctx == nil {
		return Actor{}
	}
	actor, _ := ctx.Value(actorContextKey{}).(Actor)
	return actor
}
