package db

import (
	"context"
	"testing"
)

func TestQueryRow(t *testing.T) {
	stub := NewStub()
	stub.Row = []any{"OK"}

	got, err := stub.Placeholder(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if got != "OK" {
		t.Fatalf("got %q, want %q", got, "OK")
	}
}

func TestQuery(t *testing.T) {
	stub := NewStub()
	stub.Rows = [][]any{{"one"}, {"two"}}

	rows, err := stub.Query(context.Background(), "select")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var got []string

	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			t.Fatal(err)
		}

		got = append(got, value)
	}

	if len(got) != 2 || got[0] != "one" || got[1] != "two" {
		t.Fatalf("got rows %v, want [one two]", got)
	}
}
