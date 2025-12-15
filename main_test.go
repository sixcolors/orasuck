package main

import (
	"bytes"
	"database/sql/driver"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"testing"
)

type mockDataSet struct {
	columns []string
	rows    [][]driver.Value
	curr    int
}

func (m *mockDataSet) Columns() []string {
	return m.columns
}

func (m *mockDataSet) Next(dest []driver.Value) error {
	if m.curr >= len(m.rows) {
		return io.EOF
	}
	copy(dest, m.rows[m.curr])
	m.curr++
	return nil
}

func (m *mockDataSet) Close() error {
	return nil
}

func TestProcessResults_Console(t *testing.T) {
	mock := &mockDataSet{
		columns: []string{"ID", "NAME"},
		rows: [][]driver.Value{
			{1, "Alice"},
			{2, nil},
		},
	}

	var buf bytes.Buffer
	cw := &ConsoleWriter{
		out: &buf,
	}

	err := processResults(mock, cw, nil)
	if err != nil {
		t.Fatalf("processResults failed: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Fatal("Buffer is empty")
	}

	if !strings.Contains(output, "Alice") {
		t.Errorf("Console output missing data 'Alice'. Got:\n%s", output)
	}
	if !strings.Contains(output, "<nil>") {
		t.Errorf("Console output missing data '<nil>'. Got:\n%s", output)
	}
}

func TestProcessResults_CSV(t *testing.T) {
	mock := &mockDataSet{
		columns: []string{"ID", "NAME"},
		rows: [][]driver.Value{
			{1, "Alice"},
			{2, nil},
		},
	}

	var buf bytes.Buffer
	rw := &CSVWriter{w: csv.NewWriter(&buf)}
	err := processResults(mock, rw, nil)
	if err != nil {
		t.Fatalf("processResults failed: %v", err)
	}

	expected := "ID,NAME\n1,Alice\n2,\n"
	if got := buf.String(); got != expected {
		t.Errorf("CSV output mismatch.\nGot:\n%q\nWant:\n%q", got, expected)
	}
}

func TestProcessResults_JSON(t *testing.T) {
	mock := &mockDataSet{
		columns: []string{"ID", "NAME"},
		rows: [][]driver.Value{
			{1, "Alice"},
			{2, "Bob"},
		},
	}

	var buf bytes.Buffer
	rw := &JSONWriter{w: &buf}
	err := processResults(mock, rw, nil)
	if err != nil {
		t.Fatalf("processResults failed: %v", err)
	}

	expected := `[{"ID":1,"NAME":"Alice"},{"ID":2,"NAME":"Bob"}]`
	if got := buf.String(); got != expected {
		t.Errorf("JSON output mismatch.\nGot:\n%q\nWant:\n%q", got, expected)
	}
}

func TestProcessResults_Console_WideContent(t *testing.T) {
	longText := strings.Repeat("A very long string ", 10) // ~190 chars
	mock := &mockDataSet{
		columns: []string{"ID", "DESCRIPTION"},
		rows: [][]driver.Value{
			{1, longText},
		},
	}

	var buf bytes.Buffer
	cw := &ConsoleWriter{
		out: &buf,
	}

	err := processResults(mock, cw, nil)
	if err != nil {
		t.Fatalf("processResults failed: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Fatal("Buffer is empty")
	}

	// We just want to ensure it runs without panic and produces output
	// The exact formatting depends on terminal width detection which might vary
	if !strings.Contains(output, "DESCRIPTION") {
		t.Errorf("Console output missing header. Got:\n%s", output)
	}
}

func TestProcessResults_Console_Empty(t *testing.T) {
	mock := &mockDataSet{
		columns: []string{},
		rows:    [][]driver.Value{},
	}

	var buf bytes.Buffer
	cw := &ConsoleWriter{
		out: &buf,
	}

	err := processResults(mock, cw, nil)
	if err != nil {
		t.Fatalf("processResults failed: %v", err)
	}

	output := buf.String()
	if output != "" {
		t.Errorf("Expected empty output, got %q", output)
	}
}

type errorMockDataSet struct {
	mockDataSet
	failNext bool
}

func (m *errorMockDataSet) Next(dest []driver.Value) error {
	if m.failNext {
		return fmt.Errorf("mock next error")
	}
	return m.mockDataSet.Next(dest)
}

func TestProcessResults_Error(t *testing.T) {
	mock := &errorMockDataSet{
		mockDataSet: mockDataSet{
			columns: []string{"ID"},
			rows:    [][]driver.Value{{1}},
		},
		failNext: true,
	}

	rw := &ConsoleWriter{out: &bytes.Buffer{}}
	err := processResults(mock, rw, nil)
	if err == nil {
		t.Fatal("Expected error from processResults, got nil")
	}
	if err.Error() != "mock next error" {
		t.Errorf("Expected 'mock next error', got '%v'", err)
	}
}

type errorWriter struct {
	failInit  bool
	failWrite bool
}

func (ew *errorWriter) Init(columns []string) error {
	if ew.failInit {
		return fmt.Errorf("mock init error")
	}
	return nil
}

func (ew *errorWriter) Write(values []driver.Value) error {
	if ew.failWrite {
		return fmt.Errorf("mock write error")
	}
	return nil
}

func (ew *errorWriter) Finish() error {
	return nil
}

func TestProcessResults_WriterErrors(t *testing.T) {
	mock := &mockDataSet{
		columns: []string{"ID"},
		rows:    [][]driver.Value{{1}},
	}

	// Test Init Error
	ewInit := &errorWriter{failInit: true}
	err := processResults(mock, ewInit, nil)
	if err == nil || err.Error() != "mock init error" {
		t.Errorf("Expected 'mock init error', got %v", err)
	}

	// Test Write Error
	// Reset mock cursor
	mock.curr = 0
	ewWrite := &errorWriter{failWrite: true}
	err = processResults(mock, ewWrite, nil)
	if err == nil || err.Error() != "mock write error" {
		t.Errorf("Expected 'mock write error', got %v", err)
	}
}
