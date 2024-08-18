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
	"github.com/xitongsys/parquet-go/writer"
)

//go:embed system-prompt.txt
var systemPrompt string

var reverse bool

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	pretty := flag.Bool("pretty", false, "Enable pretty-printing of the JSON output")
	maxRows := flag.Int("max-rows", -1, "Specify the maximum number of rows to convert (default: all)")
	flag.BoolVar(&reverse, "reverse", false, "Reverse mode: convert JSON to Parquet")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("Usage: parquet2json [options] <input_file> [output_file]")
	}

	inputFile := args[0]
	outputFile := ""
	if len(args) == 2 {
		outputFile = args[1]
	} else {
		if reverse {
			outputFile = replaceExtension(inputFile, ".parquet")
		} else {
			outputFile = replaceExtension(inputFile, ".json")
		}
	}

	if reverse {
		return convertJSONToParquet(inputFile, outputFile, *maxRows)
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

func convertJSONToParquet(inputFile, outputFile string, maxRows int) error {
	jsonFile, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer jsonFile.Close()

	decoder := json.NewDecoder(jsonFile)

	// Read the first object to determine the schema
	var firstObject map[string]interface{}
	if err := decoder.Decode(&firstObject); err != nil {
		return fmt.Errorf("failed to read first JSON object: %w", err)
	}

	// Create a schema based on the first object
	schema, err := createParquetSchema(firstObject)
	if err != nil {
		return fmt.Errorf("failed to create Parquet schema: %w", err)
	}

	// Create Parquet file writer
	fw, err := local.NewLocalFileWriter(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer fw.Close()

	pw, err := writer.NewJSONWriter(schema, fw, 4)
	if err != nil {
		return fmt.Errorf("failed to create Parquet writer: %w", err)
	}
	defer pw.WriteStop()

	// Write the first object
	if err := pw.Write(firstObject); err != nil {
		return fmt.Errorf("failed to write first row: %w", err)
	}

	rowCount := 1
	// Read and write the rest of the objects
	for decoder.More() && (maxRows == -1 || rowCount < maxRows) {
		var obj map[string]interface{}
		if err := decoder.Decode(&obj); err != nil {
			return fmt.Errorf("failed to read JSON object: %w", err)
		}
		if err := pw.Write(obj); err != nil {
			return fmt.Errorf("failed to write row %d: %w", rowCount+1, err)
		}
		rowCount++
	}

	fmt.Printf("Successfully converted %d rows from %s to %s\n", rowCount, inputFile, outputFile)
	return nil
}

func createParquetSchema(obj map[string]interface{}) (string, error) {
	var schema string
	schema += "message schema {\n"
	for key, value := range obj {
		schema += fmt.Sprintf("  required %s %s;\n", getParquetType(value), key)
	}
	schema += "}\n"
	return schema, nil
}

func getParquetType(value interface{}) string {
	switch value.(type) {
	case float64:
		return "double"
	case string:
		return "binary"
	case bool:
		return "boolean"
	default:
		return "binary" // Default to binary for complex types
	}
}

func replaceExtension(filename, newExt string) string {
	ext := filepath.Ext(filename)
	return filename[:len(filename)-len(ext)] + newExt
}
