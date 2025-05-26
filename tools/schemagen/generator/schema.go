package generator

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// DataType represents a database column type
type DataType string

// Common data types across different databases
const (
	TypeInteger   DataType = "integer"
	TypeSmallInt  DataType = "smallint"
	TypeBigInt    DataType = "bigint"
	TypeText      DataType = "text"
	TypeVarchar   DataType = "varchar"
	TypeChar      DataType = "char"
	TypeBool      DataType = "boolean"
	TypeFloat     DataType = "float"
	TypeNumeric   DataType = "numeric"
	TypeDate      DataType = "date"
	TypeTime      DataType = "time"
	TypeTimestamp DataType = "timestamp"
	TypeUUID      DataType = "uuid"
	TypeJSON      DataType = "json"
	TypeJSONB     DataType = "jsonb"
	TypeBLOB      DataType = "blob"
)

// Column represents a table column
type Column struct {
	Name          string            `yaml:"name"`
	Type          DataType          `yaml:"type"`
	Length        int               `yaml:"length,omitempty"`
	Precision     int               `yaml:"precision,omitempty"`
	Scale         int               `yaml:"scale,omitempty"`
	Nullable      bool              `yaml:"nullable,omitempty"`
	Default       string            `yaml:"default,omitempty"`
	PrimaryKey    bool              `yaml:"primary_key,omitempty"`
	Unique        bool              `yaml:"unique,omitempty"`
	AutoIncrement bool              `yaml:"auto_increment,omitempty"`
	References    string            `yaml:"references,omitempty"`
	OnDelete      string            `yaml:"on_delete,omitempty"`
	OnUpdate      string            `yaml:"on_update,omitempty"`
	Comment       string            `yaml:"comment,omitempty"`
	Properties    map[string]string `yaml:"properties,omitempty"`
}

// Index represents a table index
type Index struct {
	Name     string   `yaml:"name"`
	Columns  []string `yaml:"columns"`
	Unique   bool     `yaml:"unique,omitempty"`
	Method   string   `yaml:"method,omitempty"`   // e.g., btree, hash, etc.
	Includes []string `yaml:"includes,omitempty"` // For covering indexes
}

// Table represents a database table
type Table struct {
	Name       string             `yaml:"name"`
	Columns    map[string]*Column `yaml:"columns"`
	Indexes    []*Index           `yaml:"indexes,omitempty"`
	Comment    string             `yaml:"comment,omitempty"`
	Properties map[string]string  `yaml:"properties,omitempty"`
}

// Schema represents a complete database schema
type Schema struct {
	Tables map[string]*Table `yaml:"tables"`
}

// Migration represents up and down SQL migrations
type Migration struct {
	Up   string
	Down string
}

// NewSchema creates a new empty schema
func NewSchema() *Schema {
	return &Schema{
		Tables: make(map[string]*Table),
	}
}

// LoadSchemaFromFile loads a schema from a YAML file
func LoadSchemaFromFile(path string) (*Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	var schema Schema
	if err := yaml.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("failed to parse schema YAML: %w", err)
	}

	return &schema, nil
}

// ParseSchemaFromString parses a schema from a simplified string description
func ParseSchemaFromString(description string) (*Schema, error) {
	schema := NewSchema()

	// Basic pattern recognition for table(col:type,col:type) format
	tablePattern := regexp.MustCompile(`(\w+)\(([^)]+)\)`)
	tables := tablePattern.FindAllStringSubmatch(description, -1)

	for _, tableMatch := range tables {
		if len(tableMatch) < 3 {
			continue
		}

		tableName := tableMatch[1]
		columnDefs := tableMatch[2]

		table := &Table{
			Name:    tableName,
			Columns: make(map[string]*Column),
		}

		// Split column definitions
		colDefs := strings.Split(columnDefs, ",")
		for _, colDef := range colDefs {
			parts := strings.Split(strings.TrimSpace(colDef), ":")
			if len(parts) < 2 {
				continue
			}

			colName := parts[0]
			colType := DataType(parts[1])

			column := &Column{
				Name:     colName,
				Type:     colType,
				Nullable: true, // Default to nullable
			}

			// Process constraints if present
			if len(parts) > 2 {
				for i := 2; i < len(parts); i++ {
					constraint := parts[i]
					switch constraint {
					case "pk", "primary":
						column.PrimaryKey = true
						column.Nullable = false
					case "unique":
						column.Unique = true
					case "not_null", "notnull":
						column.Nullable = false
					case "auto", "autoincrement":
						column.AutoIncrement = true
						column.Nullable = false
					default:
						// Check if it's a default value
						if strings.HasPrefix(constraint, "default=") {
							column.Default = strings.TrimPrefix(constraint, "default=")
						}
						// Check if it's a foreign key reference
						if strings.HasPrefix(constraint, "references=") {
							column.References = strings.TrimPrefix(constraint, "references=")
						}
					}
				}
			}

			table.Columns[colName] = column
		}

		schema.Tables[tableName] = table
	}

	return schema, nil
}