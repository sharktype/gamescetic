package db

import (
	"context"
	"fmt"
	"reflect"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/sharktype/satchel/generated"
)

// Stub implements the database client.
//
// Since transactions are ignored, you should only use this in the simplest of cases or where
// actual behaviour of the database doesn't matter. Otherwise, use the testing client.
type Stub struct {
	*generated.Queries

	QueryRowErr error

	// Row is returned on QueryRow scans.
	Row []any

	// Rows is returned on Query scans.
	Rows [][]any
}

func NewStub() *Stub {
	stub := &Stub{}
	stub.Queries = generated.New(stub)

	return stub
}

func (s *Stub) IsTx() bool {
	return false
}

func (s *Stub) ExecTx(_ context.Context, fn func(Client) error) error {
	return fn(s)
}

// Additional stubbing types and methods:

func (s *Stub) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return &rowsStub{rows: s.Rows}, nil
}

func (s *Stub) QueryRow(context.Context, string, ...any) pgx.Row {
	return rowStub{values: s.Row, err: s.QueryRowErr}
}

func (s *Stub) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

type rowStub struct {
	values []any
	err    error
}

func (r rowStub) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}

	return scanRow(r.values, dest...)
}

type rowsStub struct {
	pgx.Rows

	rows  [][]any
	index int
}

func (r *rowsStub) Close() {}

func (r *rowsStub) Err() error {
	return nil
}

func (r *rowsStub) Next() bool {
	if r.index >= len(r.rows) {
		return false
	}

	r.index++

	return true
}

func (r *rowsStub) Scan(dest ...any) error {
	if r.index == 0 {
		return fmt.Errorf("db.Stub: Scan called before Next")
	}

	return scanRow(r.rows[r.index-1], dest...)
}

// Helpers:

func scanRow(values []any, dest ...any) error {
	if len(values) == 0 {
		return nil
	}

	if len(values) != len(dest) {
		return fmt.Errorf("db.Stub: %d row values for %d destinations", len(values), len(dest))
	}

	for i, value := range values {
		target := reflect.ValueOf(dest[i])
		if target.Kind() != reflect.Pointer || target.IsNil() {
			return fmt.Errorf("db.Stub: destination %d is not a non-nil pointer", i)
		}

		if value == nil {
			target.Elem().SetZero()

			continue
		}

		source := reflect.ValueOf(value)
		if !source.Type().AssignableTo(target.Elem().Type()) {
			return fmt.Errorf("db.Stub: cannot assign %T to %s", value, target.Elem().Type())
		}

		target.Elem().Set(source)
	}

	return nil
}
