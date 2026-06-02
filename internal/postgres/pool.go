package postgres

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PoolQuerier runs validated read-only selects against a pgx pool.
type PoolQuerier struct {
	Pool *pgxpool.Pool
}

func NewPoolFromEnv(dsnEnv string) (*pgxpool.Pool, error) {
	dsn := os.Getenv(dsnEnv)
	if dsn == "" {
		return nil, fmt.Errorf("environment variable %q is not set", dsnEnv)
	}
	return pgxpool.New(context.Background(), dsn)
}

func (p PoolQuerier) Query(ctx context.Context, q Query) ([]map[string]any, error) {
	rows, err := p.Pool.Query(ctx, q.SQL, q.Args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fieldDescriptions := rows.FieldDescriptions()
	cols := make([]string, len(fieldDescriptions))
	for i, fd := range fieldDescriptions {
		cols[i] = string(fd.Name)
	}

	var out []map[string]any
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, err
		}
		row := make(map[string]any, len(cols))
		for i, col := range cols {
			row[col] = jsonSafeValue(values[i])
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func jsonSafeValue(v any) any {
	switch x := v.(type) {
	case []byte:
		return string(x)
	default:
		return x
	}
}
