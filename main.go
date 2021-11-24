package main

import (
	"database/sql/driver"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/schollz/progressbar/v3"
	go_ora "github.com/sijms/go-ora"
)

func dieOnError(msg string, err error) {
	if err != nil {
		fmt.Println(msg, err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Println()
	fmt.Println("orasuck")
	fmt.Println("  query data from oracle, optionally export to csv.")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println(`  orasuck -server server_url [-file filename.csv] sql_query`)
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println(`  orasuck -server "oracle://user:pass@server/service_name" "select * from my_table"`)
	fmt.Println(`  orasuck -server "oracle://user:pass@server/service_name" -file "out.csv" "select * from my_table"`)
	fmt.Println()
}

func main() {
	var (
		server string
		file   string
		query  string
	)
	flag.StringVar(&server, "server", "", "Server's URL, oracle://user:pass@server/service_name")
	flag.StringVar(&file, "file", "", "Target file, out.csv")
	flag.Parse()

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
	toCsv := false
	filename := os.ExpandEnv(file)
	if filename == "" {
		fmt.Println("Missing -file option")
		usage()
		os.Exit(1)
	}

	DB, err := go_ora.NewConnection(connStr)
	dieOnError("Can't open the driver:", err)
	err = DB.Open()
	dieOnError("Can't open the connection:", err)

	defer DB.Close()

	stmt := go_ora.NewStmt(query, DB)

	defer stmt.Close()

	rows, err := stmt.Query(nil)
	dieOnError("Can't query", err)
	defer rows.Close()

	columns := rows.Columns()

	values := make([]driver.Value, len(columns))

	var f *os.File
	var w *csv.Writer
	if toCsv {
		var err error
		f, err = os.Create(filename)
		if err != nil {
			log.Fatalf("failed to open file %s %v\n", filename, err)
		}
		defer f.Close()
		w = csv.NewWriter(f)
		defer w.Flush()
	}

	var bar *progressbar.ProgressBar

	if toCsv {
		bar = progressbar.Default(-1, fmt.Sprintf("Exporting to %s...", filename))
		w.Write(columns)
		w.Flush()
	} else {
		Header(columns)
	}

	for {
		err = rows.Next(values)
		if err != nil {
			break
		}
		if !toCsv {
			Record(columns, values)
		} else {
			aRow := []string{}
			for _, c := range values {
				colValue := fmt.Sprintf("%v", c)
				if colValue == "<nil>" {
					colValue = ""
					aRow = append(aRow, colValue)
				}
			}
			if err := w.Write(aRow); err != nil {
				log.Fatalln(err)
			}
			w.Flush()
			bar.Add(1)
		}
	}
	if err != io.EOF {
		dieOnError("Can't Next", err)
	}
}

func Header(columns []string) {

}

func Record(columns []string, values []driver.Value) {
	for i, c := range values {
		fmt.Printf("%-25s: %v\n", columns[i], c)
	}
	fmt.Println()
}
