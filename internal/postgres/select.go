package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/fyne-coder/mcp-data-gateway/internal/config"
	"github.com/jackc/pgx/v5"
)

const ToolName = "postgres_select"

var (
	ErrToolPackDenied   = errors.New("tool pack not allowed")
	ErrTableDenied      = errors.New("table not allowed")
	ErrColumnDenied     = errors.New("column not allowed")
	ErrLimitExceeded    = errors.New("limit exceeds configured max rows")
	ErrEmptyColumns     = errors.New("columns must not be empty")
	ErrInvalidTableName = errors.New("invalid table name")
)

type SelectInput struct {
	Table   string   `json:"table"`
	Columns []string `json:"columns"`
	Limit   *int     `json:"limit,omitempty"`
}

type SelectOutput struct {
	Rows     []map[string]any `json:"rows"`
	RowCount int              `json:"row_count"`
	Table    string           `json:"table"`
	Columns  []string         `json:"columns"`
}

type Query struct {
	SQL     string
	Args    []any
	Table   string
	Columns []string
}

// Querier executes a validated read-only select.
type Querier interface {
	Query(ctx context.Context, q Query) ([]map[string]any, error)
}

type Selector struct {
	Allowlists config.PostgresConfig
	Querier    Querier
}

func (s Selector) Select(ctx context.Context, toolPack string, input SelectInput) (SelectOutput, error) {
	q, err := s.buildQuery(toolPack, input)
	if err != nil {
		return SelectOutput{}, err
	}
	rows, err := s.Querier.Query(ctx, q)
	if err != nil {
		return SelectOutput{}, err
	}
	return SelectOutput{
		Rows:     rows,
		RowCount: len(rows),
		Table:    input.Table,
		Columns:  append([]string(nil), input.Columns...),
	}, nil
}

func (s Selector) buildQuery(toolPack string, input SelectInput) (Query, error) {
	pack, ok := s.Allowlists.ToolPacks[toolPack]
	if !ok {
		return Query{}, ErrToolPackDenied
	}
	tableAllow, ok := pack.Tables[input.Table]
	if !ok {
		return Query{}, ErrTableDenied
	}
	if len(input.Columns) == 0 {
		return Query{}, ErrEmptyColumns
	}
	allowedCols := make(map[string]struct{}, len(tableAllow.Columns))
	for _, col := range tableAllow.Columns {
		allowedCols[col] = struct{}{}
	}
	for _, col := range input.Columns {
		if _, ok := allowedCols[col]; !ok {
			return Query{}, fmt.Errorf("%w: %q", ErrColumnDenied, col)
		}
	}

	limit := s.Allowlists.MaxRows
	if input.Limit != nil {
		if *input.Limit <= 0 {
			return Query{}, fmt.Errorf("%w: limit must be positive", ErrLimitExceeded)
		}
		if *input.Limit > s.Allowlists.MaxRows {
			return Query{}, ErrLimitExceeded
		}
		limit = *input.Limit
	}

	tableParts, err := splitTableName(input.Table)
	if err != nil {
		return Query{}, err
	}
	tableSQL := pgx.Identifier(tableParts).Sanitize()

	quotedCols := make([]string, len(input.Columns))
	for i, col := range input.Columns {
		quotedCols[i] = pgx.Identifier{col}.Sanitize()
	}

	sql := fmt.Sprintf("SELECT %s FROM %s LIMIT $1", strings.Join(quotedCols, ", "), tableSQL)
	return Query{
		SQL:     sql,
		Args:    []any{limit},
		Table:   input.Table,
		Columns: append([]string(nil), input.Columns...),
	}, nil
}

func splitTableName(table string) ([]string, error) {
	parts := strings.Split(table, ".")
	if len(parts) == 0 || parts[0] == "" {
		return nil, ErrInvalidTableName
	}
	for _, part := range parts {
		if part == "" {
			return nil, ErrInvalidTableName
		}
	}
	return parts, nil
}
