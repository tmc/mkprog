package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tmc/mkprog/tools/schemagen/generator"
)

var (
	dbFlag         string
	langFlag       string
	inputFlag      string
	outputDirFlag  string
	fromSchemaFlag string
	toSchemaFlag   string
	visualizeFlag  bool
	migrationsFlag bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "schemagen [flags] [schema_description]",
		Short: "Generate database schemas from descriptions",
		Long: `schemagen is a tool that generates database schemas and migrations from high-level descriptions.
It allows developers to define database schemas using natural language or simplified syntax.`,
		Args: cobra.MaximumNArgs(1),
		RunE: runSchemagen,
	}

	rootCmd.Flags().StringVar(&dbFlag, "db", "postgres", "Target database (postgres, mysql, sqlite)")
	rootCmd.Flags().StringVar(&langFlag, "lang", "sql", "Output language (sql, go, json, yaml)")
	rootCmd.Flags().StringVar(&inputFlag, "input", "", "Input schema file (YAML)")
	rootCmd.Flags().StringVar(&outputDirFlag, "output-dir", "", "Output directory for generated files")
	rootCmd.Flags().StringVar(&fromSchemaFlag, "from", "", "Original schema file for migrations")
	rootCmd.Flags().StringVar(&toSchemaFlag, "to", "", "Target schema file for migrations")
	rootCmd.Flags().BoolVar(&visualizeFlag, "visualize", false, "Generate schema visualization (DOT format)")
	rootCmd.Flags().BoolVar(&migrationsFlag, "migrations", false, "Generate migration files")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runSchemagen(cmd *cobra.Command, args []string) error {
	var schema *generator.Schema
	var err error

	// Load schema from file or description
	if inputFlag != "" {
		schema, err = generator.LoadSchemaFromFile(inputFlag)
	} else if len(args) > 0 {
		schema, err = generator.ParseSchemaFromString(args[0])
	} else {
		return fmt.Errorf("no input provided, use --input or provide a schema description")
	}

	if err != nil {
		return fmt.Errorf("failed to parse schema: %w", err)
	}

	// If migrations are requested, handle that separately
	if migrationsFlag {
		if fromSchemaFlag == "" || toSchemaFlag == "" {
			return fmt.Errorf("both --from and --to must be specified for migrations")
		}
		return generateMigrations()
	}

	// Generate output based on flags
	if visualizeFlag {
		return generateVisualization(schema)
	}

	// Generate schema in requested format
	return generateSchema(schema)
}

func generateSchema(schema *generator.Schema) error {
	var output string
	var err error

	switch langFlag {
	case "sql":
		output, err = generator.GenerateSQL(schema, dbFlag)
	case "go":
		output, err = generator.GenerateGo(schema, dbFlag)
	case "json":
		output, err = generator.GenerateJSON(schema)
	case "yaml":
		output, err = generator.GenerateYAML(schema)
	default:
		return fmt.Errorf("unsupported output language: %s", langFlag)
	}

	if err != nil {
		return fmt.Errorf("failed to generate schema: %w", err)
	}

	// Output to directory or stdout
	if outputDirFlag != "" {
		return writeToDirectory(output)
	}
	
	fmt.Println(output)
	return nil
}

func generateVisualization(schema *generator.Schema) error {
	graph, err := generator.GenerateGraph(schema)
	if err != nil {
		return fmt.Errorf("failed to generate visualization: %w", err)
	}

	if outputDirFlag != "" {
		dotFile := outputDirFlag + "/schema.dot"
		if err := os.WriteFile(dotFile, []byte(graph), 0644); err != nil {
			return fmt.Errorf("failed to write DOT file: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Wrote DOT file to %s\n", dotFile)
		fmt.Fprintf(os.Stderr, "Run 'dot -Tpng %s -o schema.png' to generate an image\n", dotFile)
		return nil
	}

	fmt.Println(graph)
	return nil
}

func generateMigrations() error {
	// Load the two schemas
	fromSchema, err := generator.LoadSchemaFromFile(fromSchemaFlag)
	if err != nil {
		return fmt.Errorf("failed to load source schema: %w", err)
	}

	toSchema, err := generator.LoadSchemaFromFile(toSchemaFlag)
	if err != nil {
		return fmt.Errorf("failed to load target schema: %w", err)
	}

	// Generate the migration
	migration, err := generator.GenerateMigration(fromSchema, toSchema, dbFlag)
	if err != nil {
		return fmt.Errorf("failed to generate migration: %w", err)
	}

	// Output to directory or stdout
	if outputDirFlag != "" {
		upFile := outputDirFlag + "/migration_up.sql"
		downFile := outputDirFlag + "/migration_down.sql"
		
		if err := os.WriteFile(upFile, []byte(migration.Up), 0644); err != nil {
			return fmt.Errorf("failed to write up migration: %w", err)
		}
		
		if err := os.WriteFile(downFile, []byte(migration.Down), 0644); err != nil {
			return fmt.Errorf("failed to write down migration: %w", err)
		}
		
		fmt.Fprintf(os.Stderr, "Wrote migration files to %s and %s\n", upFile, downFile)
		return nil
	}

	fmt.Println("-- Up Migration")
	fmt.Println(migration.Up)
	fmt.Println("\n-- Down Migration")
	fmt.Println(migration.Down)
	return nil
}

func writeToDirectory(output string) error {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(outputDirFlag, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Determine the output filename based on the language
	var filename string
	switch langFlag {
	case "sql":
		filename = "schema.sql"
	case "go":
		filename = "models.go"
	case "json":
		filename = "schema.json"
	case "yaml":
		filename = "schema.yaml"
	default:
		filename = "schema.txt"
	}

	// Write the output file
	outPath := outputDirFlag + "/" + filename
	if err := os.WriteFile(outPath, []byte(output), 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Generated %s in %s\n", filename, outputDirFlag)
	return nil
}