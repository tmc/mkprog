package main

import (
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/xitongsys/parquet-go-source/local"
	"github.com/xitongsys/parquet-go/reader"
)

//go:embed system-prompt.txt
var systemPrompt string

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	pretty := flag.Bool("pretty", false, "Enable pretty-printing of the JSON output")
	maxRows := flag.Int("max-rows", -1, "Specify the maximum number of rows to convert (default: all)")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("Usage: parquet2json [options] <input_parquet_file> [output_json_file]")
	}

	inputFile := args[0]
	outputFile := ""
	if len(args) == 2 {
		outputFile = args[1]
	} else {
		outputFile = replaceExtension(inputFile, ".json")
	}

	return convertParquetToJSON(inputFile, outputFile, *pretty, *maxRows)
}

func convertParquetToJSON(inputFile, outputFile string, pretty bool, maxRows int) error {
	fr, err := local.NewLocalFileReader(inputFile)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer fr.Close()

	pr, err := reader.NewParquetReader(fr, nil, 4)
	if err != nil {
		return fmt.Errorf("failed to create Parquet reader: %w", err)
	}
	defer pr.ReadStop()

	fw, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer fw.Close()

	encoder := json.NewEncoder(fw)
	if pretty {
		encoder.SetIndent("", "  ")
	}

	numRows := int(pr.GetNumRows())
	if maxRows > 0 && maxRows < numRows {
		numRows = maxRows
	}

	for i := 0; i < numRows; i++ {
		row, err := pr.ReadByNumber(1)
		if err != nil {
			return fmt.Errorf("failed to read row %d: %w", i+1, err)
		}

		if err := encoder.Encode(row[0]); err != nil {
			return fmt.Errorf("failed to encode row %d: %w", i+1, err)
		}
	}

	fmt.Printf("Successfully converted %d rows from %s to %s\n", numRows, inputFile, outputFile)
	return nil
}

func replaceExtension(filename, newExt string) string {
	ext := filepath.Ext(filename)
	return filename[:len(filename)-len(ext)] + newExt
}
