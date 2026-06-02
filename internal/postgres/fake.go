package postgres

import (
	"context"
	"fmt"
)

// FakeQuerier records whether Query was invoked for deny-path tests.
type FakeQuerier struct {
	Rows       []map[string]any
	Err        error
	QueryCount int
	LastQuery  Query
}

func (f *FakeQuerier) Query(ctx context.Context, q Query) ([]map[string]any, error) {
	f.QueryCount++
	f.LastQuery = q
	if f.Err != nil {
		return nil, f.Err
	}
	if f.Rows != nil {
		return f.Rows, nil
	}
	return []map[string]any{}, nil
}

// UnconfiguredQuerier fails tool calls when no live Postgres DSN is configured.
type UnconfiguredQuerier struct {
	DSNEnv string
}

func (q UnconfiguredQuerier) Query(context.Context, Query) ([]map[string]any, error) {
	return nil, fmt.Errorf("postgres is not configured: set %s", q.DSNEnv)
}
