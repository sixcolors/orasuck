package main

import (
	"database/sql/driver"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/schollz/progressbar/v3"
	go_ora "github.com/sijms/go-ora/v2"
	"golang.org/x/term"
)

// Version holds the application version, typically set via linker flags during build
var Version string

// DataSet represents a database result set with methods to iterate over rows.
// It abstracts the underlying database driver's result set interface.
type DataSet interface {
	Columns() []string
	Next(values []driver.Value) error
	Close() error
}

// ResultWriter is the interface for writing query results in different formats.
// Implementations include ConsoleWriter, CSVWriter, and JSONWriter.
type ResultWriter interface {
	Init(columns []string) error
	Write(values []driver.Value) error
	Finish() error
}

// ConsoleWriter writes query results as a formatted table to the console.
// It collects all rows in memory before rendering to calculate optimal column widths.
type ConsoleWriter struct {
	columns []string
	rows    [][]string
	out     io.Writer
}

func (cw *ConsoleWriter) Init(columns []string) error {
	cw.columns = columns
	cw.rows = make([][]string, 0)
	return nil
}

func (cw *ConsoleWriter) Write(values []driver.Value) error {
	if len(values) != len(cw.columns) {
		return fmt.Errorf("column count mismatch: expected %d, got %d", len(cw.columns), len(values))
	}
	row := make([]string, len(values))
	for i, v := range values {
		if v == nil {
			row[i] = "<nil>"
		} else {
			row[i] = fmt.Sprintf("%v", v)
		}
	}
	cw.rows = append(cw.rows, row)
	return nil
}

func (cw *ConsoleWriter) Finish() error {
	out := cw.out
	if out == nil {
		out = os.Stdout
	}
	table := tablewriter.NewWriter(out)

	// Set Header
	header := make([]interface{}, len(cw.columns))
	for i, v := range cw.columns {
		header[i] = v
	}
	table.Header(header...)

	// Calculate widths
	numCols := len(cw.columns)
	if numCols == 0 {
		return nil
	}

	maxContentWidths := make([]int, numCols)

	// Check headers first
	for i, h := range cw.columns {
		if len(h) > maxContentWidths[i] {
			maxContentWidths[i] = len(h)
		}
	}

	// Check rows
	for _, row := range cw.rows {
		for i, cell := range row {
			// Bounds check to prevent panic if row has more cells than columns
			if i < numCols && len(cell) > maxContentWidths[i] {
				maxContentWidths[i] = len(cell)
			}
		}
	}

	// Add padding to maxContentWidths (1 left + 1 right)
	for i := range maxContentWidths {
		maxContentWidths[i] += 2
	}

	// Get terminal width based on output writer
	termWidth := 80
	if f, ok := out.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		if w, _, err := term.GetSize(int(f.Fd())); err == nil {
			termWidth = w
		}
	}

	// Calculate available space
	// Overhead is just the borders: | col | col |
	// 1 char per column separator + 1 char for starting border
	overhead := numCols + 1
	availableSpace := termWidth - overhead

	if availableSpace < numCols {
		availableSpace = numCols
	}

	// Distribute space
	allocatedWidths := make([]int, numCols)

	totalRequested := 0
	for _, w := range maxContentWidths {
		totalRequested += w
	}

	if totalRequested <= availableSpace {
		copy(allocatedWidths, maxContentWidths)
	} else {
		remainingSpace := availableSpace
		remainingCols := numCols

		requests := make([]int, numCols)
		copy(requests, maxContentWidths)
		satisfied := make([]bool, numCols)

		for remainingCols > 0 {
			fairShare := remainingSpace / remainingCols

			progress := false
			for i := 0; i < numCols; i++ {
				if !satisfied[i] && requests[i] <= fairShare {
					allocatedWidths[i] = requests[i]
					remainingSpace -= requests[i]
					satisfied[i] = true
					remainingCols--
					progress = true
				}
			}

			if !progress {
				for i := 0; i < numCols; i++ {
					if !satisfied[i] {
						allocatedWidths[i] = fairShare
						satisfied[i] = true
					}
				}
				break
			}
		}
	}

	// Configure table
	table.Configure(func(cfg *tablewriter.Config) {
		cfg.MaxWidth = termWidth
		cfg.Row.Formatting.AutoWrap = tw.WrapNormal
		cfg.Row.Padding.Global.Left = " "
		cfg.Row.Padding.Global.Right = " "

		cfg.Widths.PerColumn = make(map[int]int)
		for i, w := range allocatedWidths {
			cfg.Widths.PerColumn[i] = w
		}
	})

	// Add rows
	for _, row := range cw.rows {
		rowIf := make([]interface{}, len(row))
		for i, v := range row {
			rowIf[i] = v
		}
		if err := table.Append(rowIf...); err != nil {
			return err
		}
	}

	if err := table.Render(); err != nil {
		return err
	}
	return nil
}

// CSVWriter writes query results in CSV format.
// Null values are written as empty strings.
type CSVWriter struct {
	w       *csv.Writer
	columns []string
}

func (cw *CSVWriter) Init(columns []string) error {
	cw.columns = columns
	return cw.w.Write(columns)
}

func (cw *CSVWriter) Write(values []driver.Value) error {
	if len(values) != len(cw.columns) {
		return fmt.Errorf("column count mismatch: expected %d, got %d", len(cw.columns), len(values))
	}
	aRow := make([]string, len(values))
	for i, c := range values {
		if c == nil {
			aRow[i] = ""
		} else {
			aRow[i] = fmt.Sprintf("%v", c)
		}
	}
	return cw.w.Write(aRow)
}

func (cw *CSVWriter) Finish() error {
	cw.w.Flush()
	return cw.w.Error()
}

// JSONWriter writes query results as a JSON array of objects.
// Each row is written as a JSON object with column names as keys.
type JSONWriter struct {
	w       io.Writer
	columns []string
	first   bool
}

func (jw *JSONWriter) Init(columns []string) error {
	jw.columns = columns
	jw.first = true
	_, err := jw.w.Write([]byte("["))
	return err
}

func (jw *JSONWriter) Write(values []driver.Value) error {
	if len(values) != len(jw.columns) {
		return fmt.Errorf("column count mismatch: expected %d, got %d", len(jw.columns), len(values))
	}

	if !jw.first {
		if _, err := jw.w.Write([]byte(",")); err != nil {
			return err
		}
	}
	jw.first = false

	rowMap := make(map[string]interface{})
	for i, col := range jw.columns {
		rowMap[col] = values[i]
	}

	b, err := json.Marshal(rowMap)
	if err != nil {
		return err
	}
	_, err = jw.w.Write(b)
	return err
}

func (jw *JSONWriter) Finish() error {
	_, err := jw.w.Write([]byte("]"))
	return err
}

func dieOnError(msg string, err error) {
	if err != nil {
		fmt.Println(msg, err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Println()
	fmt.Println("orasuck", Version)
	fmt.Println("  query data from oracle, optionally export to csv or json.")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println(`  orasuck -server server_url [-file filename] [-json] sql_query`)
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println(`  orasuck -server "oracle://user:pass@server/service_name" "select * from my_table"`)
	fmt.Println(`  orasuck -server "oracle://user:pass@server/service_name" -file "out.csv" "select * from my_table"`)
	fmt.Println(`  orasuck -server "oracle://user:pass@server/service_name" -file "out.json" "select * from my_table"`)
	fmt.Println(`  orasuck -server "oracle://user:pass@server/service_name" -json "select * from my_table"`)
	fmt.Println(`  orasuck -server "oracle://user:pass@server/service_name" -csv "select * from my_table"`)
	fmt.Println()
}

func main() {
	var (
		server  string
		file    string
		jsonFmt bool
		csvFmt  bool
		version bool
		query   string
	)
	flag.StringVar(&server, "server", "", "Server's URL, oracle://user:pass@server/service_name")
	flag.StringVar(&file, "file", "", "Target file (defaults to JSON if extension is .json, CSV otherwise)")
	flag.BoolVar(&jsonFmt, "json", false, "Output in JSON format (default if file ends in .json)")
	flag.BoolVar(&csvFmt, "csv", false, "Output in CSV format")
	flag.BoolVar(&version, "version", false, "Display version information")
	flag.Parse()

	if version {
		fmt.Printf("orasuck %s\n", Version)
		os.Exit(0)
	}

	// Validate conflicting flags
	if jsonFmt && csvFmt {
		fmt.Println("Error: cannot specify both -json and -csv flags")
		usage()
		os.Exit(1)
	}

	if len(flag.Args()) < 1 {
		fmt.Println("Missing query")
		usage()
		os.Exit(1)
	}

	query = flag.Arg(0)
	connStr := os.ExpandEnv(server)
	if connStr == "" {
		fmt.Println("Missing -server option")
		usage()
		os.Exit(1)
	}
	filename := os.ExpandEnv(file)

	// Auto-detect JSON format from file extension
	if filename != "" {
		ext := strings.ToLower(filepath.Ext(filename))
		if !jsonFmt && !csvFmt && ext == ".json" {
			jsonFmt = true
		}
		// Warn if explicit format flag does not match file extension
		if jsonFmt && ext == ".csv" {
			fmt.Fprintf(os.Stderr, "Warning: -json flag specified but output file has .csv extension\n")
		} else if csvFmt && ext == ".json" {
			fmt.Fprintf(os.Stderr, "Warning: -csv flag specified but output file has .json extension\n")
		}
	}

	DB, err := go_ora.NewConnection(connStr, nil)
	dieOnError("Can't open the driver:", err)
	err = DB.Open()
	dieOnError("Can't open the connection:", err)

	defer func() {
		if err := DB.Close(); err != nil {
			log.Println("error closing DB:", err)
		}
	}()

	stmt := go_ora.NewStmt(query, DB)

	defer func() {
		if err := stmt.Close(); err != nil {
			log.Println("error closing stmt:", err)
		}
	}()

	rows, err := stmt.Query(nil)
	dieOnError("Can't query", err)
	defer func() {
		if err := rows.Close(); err != nil {
			log.Println("error closing rows:", err)
		}
	}()

	var rw ResultWriter
	var f *os.File
	var bar *progressbar.ProgressBar

	if filename != "" {
		var err error
		f, err = os.Create(filename) //#nosec G304 (CWE-22) this is intentional
		if err != nil {
			log.Fatalf("failed to open file %s %v\n", filename, err)
		}

		if jsonFmt {
			rw = &JSONWriter{w: f}
		} else {
			rw = &CSVWriter{w: csv.NewWriter(f)}
		}
		bar = progressbar.Default(-1, fmt.Sprintf("Exporting to %s...", filename))
	} else {
		if jsonFmt {
			rw = &JSONWriter{w: os.Stdout}
		} else if csvFmt {
			rw = &CSVWriter{w: csv.NewWriter(os.Stdout)}
		} else {
			rw = &ConsoleWriter{}
		}
	}

	err = processResults(rows, rw, bar)
	dieOnError("Can't process results", err)

	if f != nil {
		if err := f.Close(); err != nil {
			log.Fatalln(err)
		}
	}
}

// processResults iterates over the provided DataSet, writes each row using the given ResultWriter,
// and updates the progress bar if provided. It initializes the writer, processes all rows,
// and finalizes the writer. Returns an error if any operation fails.
//
// Parameters:
//
//	rows - the DataSet to iterate over (implements Columns and Next)
//	rw   - the ResultWriter to output each row (implements Init, Write, and Finish)
//	bar  - an optional progress bar to update for each row processed (can be nil)
//
// Returns:
//
//	error - non-nil if an error occurs during processing, writing, or finalization
func processResults(rows DataSet, rw ResultWriter, bar *progressbar.ProgressBar) error {
	columns := rows.Columns()
	values := make([]driver.Value, len(columns))

	if err := rw.Init(columns); err != nil {
		return err
	}

	for {
		err := rows.Next(values)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		if err := rw.Write(values); err != nil {
			return err
		}

		if bar != nil {
			if err := bar.Add(1); err != nil {
				log.Println("error updating progress bar:", err.Error())
			}
		}
	}
	return rw.Finish()
}
