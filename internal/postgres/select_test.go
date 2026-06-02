package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/fyne-coder/mcp-data-gateway/internal/config"
)

func testPostgresConfig() config.PostgresConfig {
	return config.PostgresConfig{
		DSNEnv:  "MCP_DATA_GATEWAY_POSTGRES_DSN",
		MaxRows: 100,
		ToolPacks: map[string]config.ToolPackAllowlist{
			"analyst": {
				Tables: map[string]config.TableAllowlist{
					"public.example_table": {Columns: []string{"id", "name"}},
				},
			},
		},
	}
}

func TestBuildQueryUsesSanitizedIdentifiers(t *testing.T) {
	selector := Selector{Allowlists: testPostgresConfig(), Querier: &FakeQuerier{}}
	limit := 10
	q, err := selector.buildQuery("analyst", SelectInput{
		Table:   "public.example_table",
		Columns: []string{"id", "name"},
		Limit:   &limit,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `SELECT "id", "name" FROM "public"."example_table" LIMIT $1`
	if q.SQL != want {
		t.Fatalf("SQL = %q, want %q", q.SQL, want)
	}
	if len(q.Args) != 1 || q.Args[0] != 10 {
		t.Fatalf("Args = %v, want [10]", q.Args)
	}
}

func TestSelectDeniedTableDoesNotQuery(t *testing.T) {
	fake := &FakeQuerier{}
	selector := Selector{Allowlists: testPostgresConfig(), Querier: fake}
	_, err := selector.Select(context.Background(), "analyst", SelectInput{
		Table:   "public.secret_table",
		Columns: []string{"id"},
	})
	if !errors.Is(err, ErrTableDenied) {
		t.Fatalf("err = %v, want %v", err, ErrTableDenied)
	}
	if fake.QueryCount != 0 {
		t.Fatalf("QueryCount = %d, want 0", fake.QueryCount)
	}
}

func TestSelectDeniedColumnDoesNotQuery(t *testing.T) {
	fake := &FakeQuerier{}
	selector := Selector{Allowlists: testPostgresConfig(), Querier: fake}
	_, err := selector.Select(context.Background(), "analyst", SelectInput{
		Table:   "public.example_table",
		Columns: []string{"password"},
	})
	if !errors.Is(err, ErrColumnDenied) {
		t.Fatalf("err = %v, want %v", err, ErrColumnDenied)
	}
	if fake.QueryCount != 0 {
		t.Fatalf("QueryCount = %d, want 0", fake.QueryCount)
	}
}

func TestSelectDeniedToolPackDoesNotQuery(t *testing.T) {
	fake := &FakeQuerier{}
	selector := Selector{Allowlists: testPostgresConfig(), Querier: fake}
	_, err := selector.Select(context.Background(), "unknown", SelectInput{
		Table:   "public.example_table",
		Columns: []string{"id"},
	})
	if !errors.Is(err, ErrToolPackDenied) {
		t.Fatalf("err = %v, want %v", err, ErrToolPackDenied)
	}
	if fake.QueryCount != 0 {
		t.Fatalf("QueryCount = %d, want 0", fake.QueryCount)
	}
}

func TestSelectRejectsExcessiveLimit(t *testing.T) {
	fake := &FakeQuerier{}
	selector := Selector{Allowlists: testPostgresConfig(), Querier: fake}
	limit := 101
	_, err := selector.Select(context.Background(), "analyst", SelectInput{
		Table:   "public.example_table",
		Columns: []string{"id"},
		Limit:   &limit,
	})
	if !errors.Is(err, ErrLimitExceeded) {
		t.Fatalf("err = %v, want %v", err, ErrLimitExceeded)
	}
	if fake.QueryCount != 0 {
		t.Fatalf("QueryCount = %d, want 0", fake.QueryCount)
	}
}

func TestSelectAllowedPathQueries(t *testing.T) {
	fake := &FakeQuerier{
		Rows: []map[string]any{{"id": 1, "name": "example"}},
	}
	selector := Selector{Allowlists: testPostgresConfig(), Querier: fake}
	out, err := selector.Select(context.Background(), "analyst", SelectInput{
		Table:   "public.example_table",
		Columns: []string{"id", "name"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if fake.QueryCount != 1 {
		t.Fatalf("QueryCount = %d, want 1", fake.QueryCount)
	}
	if out.RowCount != 1 || out.Rows[0]["name"] != "example" {
		t.Fatalf("out = %#v", out)
	}
}

func TestUnconfiguredQuerierFailsSafely(t *testing.T) {
	_, err := UnconfiguredQuerier{DSNEnv: "MCP_DATA_GATEWAY_POSTGRES_DSN"}.Query(context.Background(), Query{})
	if err == nil {
		t.Fatal("Query succeeded, want unconfigured error")
	}
}
