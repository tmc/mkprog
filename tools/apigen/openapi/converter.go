package openapi

import (
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// GoType represents a Go type
type GoType struct {
	Name       string
	Import     string
	IsBuiltin  bool
	IsArray    bool
	IsMap      bool
	IsPointer  bool
	Item       *GoType  // For arrays
	Key        *GoType  // For maps
	Properties []GoType // For structs
}

// TypeConverter converts OpenAPI types to Go types
type TypeConverter struct {
	prefix        string
	typeMap       map[string]GoType
	currentSchema map[string]*openapi3.Schema
}

// NewTypeConverter creates a new TypeConverter
func NewTypeConverter(prefix string) *TypeConverter {
	return &TypeConverter{
		prefix:        prefix,
		typeMap:       make(map[string]GoType),
		currentSchema: make(map[string]*openapi3.Schema),
	}
}

// ConvertSchema converts an OpenAPI schema to a Go type
func (c *TypeConverter) ConvertSchema(name string, schema *openapi3.Schema) (GoType, error) {
	if schema == nil {
		return GoType{Name: "interface{}", IsBuiltin: true}, nil
	}

	// Check if we're already processing this schema to avoid infinite recursion
	if _, exists := c.currentSchema[name]; exists {
		// We're in a cycle, use a pointer to the type
		return GoType{
			Name:      c.prefix + name,
			IsPointer: true,
		}, nil
	}

	// Mark this schema as being processed
	c.currentSchema[name] = schema

	// Check if we already have this type
	if goType, exists := c.typeMap[name]; exists {
		delete(c.currentSchema, name)
		return goType, nil
	}

	var goType GoType

	switch schema.Type {
	case "integer":
		goType = convertIntegerType(schema)
	case "number":
		goType = convertNumberType(schema)
	case "string":
		goType = convertStringType(schema)
	case "boolean":
		goType = GoType{Name: "bool", IsBuiltin: true}
	case "array":
		item, err := c.ConvertSchema(name+"Item", schema.Items.Value)
		if err != nil {
			return GoType{}, err
		}
		goType = GoType{
			Name:    "[]" + item.Name,
			IsArray: true,
			Item:    &item,
		}
	case "object":
		// For a named object type
		if name != "" {
			goType = GoType{
				Name: c.prefix + toCamelCase(name),
			}

			// Store the type to break cyclic references
			c.typeMap[name] = goType

			// Process properties
			if len(schema.Properties) > 0 {
				// For each property in the schema
				for propName, propSchema := range schema.Properties {
					if propSchema.Value != nil {
						propType, err := c.ConvertSchema(name+toCamelCase(propName), propSchema.Value)
						if err != nil {
							return GoType{}, err
						}

						// Add the property to our properties list
						propType.Name = toCamelCase(propName)
						goType.Properties = append(goType.Properties, propType)
					}
				}
			} else {
				// Generic object (map)
				goType = GoType{
					Name:   "map[string]interface{}",
					IsMap:  true,
					Key:    &GoType{Name: "string", IsBuiltin: true},
					Item:   &GoType{Name: "interface{}", IsBuiltin: true},
				}
			}
		} else {
			// Anonymous object
			goType = GoType{
				Name:   "map[string]interface{}",
				IsMap:  true,
				Key:    &GoType{Name: "string", IsBuiltin: true},
				Item:   &GoType{Name: "interface{}", IsBuiltin: true},
			}
		}
	default:
		// Default to interface{}
		goType = GoType{Name: "interface{}", IsBuiltin: true}
	}

	// Mark schema as processed
	delete(c.currentSchema, name)

	// Add to type map
	if name != "" {
		c.typeMap[name] = goType
	}

	return goType, nil
}

// ConvertDefinitions converts all schema definitions to Go types
func (c *TypeConverter) ConvertDefinitions(definitions map[string]*openapi3.Schema) (map[string]GoType, error) {
	result := make(map[string]GoType)

	for name, schema := range definitions {
		goType, err := c.ConvertSchema(name, schema)
		if err != nil {
			return nil, err
		}
		result[name] = goType
	}

	return result, nil
}

// Convert specific types
func convertIntegerType(schema *openapi3.Schema) GoType {
	switch schema.Format {
	case "int64":
		return GoType{Name: "int64", IsBuiltin: true}
	case "int32":
		return GoType{Name: "int32", IsBuiltin: true}
	default:
		return GoType{Name: "int", IsBuiltin: true}
	}
}

func convertNumberType(schema *openapi3.Schema) GoType {
	switch schema.Format {
	case "float":
		return GoType{Name: "float32", IsBuiltin: true}
	case "double":
		return GoType{Name: "float64", IsBuiltin: true}
	default:
		return GoType{Name: "float64", IsBuiltin: true}
	}
}

func convertStringType(schema *openapi3.Schema) GoType {
	switch schema.Format {
	case "byte":
		return GoType{Name: "[]byte", IsBuiltin: true, IsArray: true, Item: &GoType{Name: "byte", IsBuiltin: true}}
	case "binary":
		return GoType{Name: "[]byte", IsBuiltin: true, IsArray: true, Item: &GoType{Name: "byte", IsBuiltin: true}}
	case "date", "date-time":
		return GoType{Name: "time.Time", Import: "time", IsBuiltin: false}
	case "uuid":
		return GoType{Name: "uuid.UUID", Import: "github.com/google/uuid", IsBuiltin: false}
	default:
		return GoType{Name: "string", IsBuiltin: true}
	}
}

// Helper functions
func toCamelCase(s string) string {
	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
	
	for i, word := range words {
		if i == 0 {
			words[i] = strings.Title(strings.ToLower(word))
		} else {
			words[i] = strings.Title(word)
		}
	}
	
	return strings.Join(words, "")
}