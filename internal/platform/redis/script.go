package redis

import (
	"context"

	goredis "github.com/redis/go-redis/v9"
)

// Script wraps a Lua script for execution with RedisClient.
type Script struct {
	src string
}

// NewScript creates a new Script wrapper.
func NewScript(src string) *Script {
	return &Script{src: src}
}

// Run executes the script using EVAL.
func (s *Script) Run(ctx context.Context, client RedisClient, keys []string, args ...any) *goredis.Cmd {
	if s == nil {
		cmd := goredis.NewCmd(ctx)
		cmd.SetErr(goredis.Nil)
		return cmd
	}
	return client.Eval(ctx, s.src, keys, args...)
}
