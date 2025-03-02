package openapi

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// Specification represents a parsed OpenAPI specification
type Specification struct {
	*openapi3.T
}

// ParseSpecification parses an OpenAPI specification file
func ParseSpecification(specPath string) (*Specification, error) {
	// Read the file extension to determine format
	ext := strings.ToLower(filepath.Ext(specPath))
	
	// Load the spec file
	loader := openapi3.NewLoader()
	var doc *openapi3.T
	var err error
	
	// Load the OpenAPI document
	switch ext {
	case ".json":
		doc, err = loader.LoadFromFile(specPath)
	case ".yaml", ".yml":
		doc, err = loader.LoadFromFile(specPath)
	default:
		// Try to guess the format
		content, readErr := os.ReadFile(specPath)
		if readErr != nil {
			return nil, fmt.Errorf("failed to read spec file: %w", readErr)
		}
		
		// Check if content looks like JSON
		if strings.TrimSpace(string(content))[0] == '{' {
			doc, err = loader.LoadFromData(content)
		} else {
			doc, err = loader.LoadFromData(content)
		}
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI spec: %w", err)
	}
	
	// Validate the document
	if err := doc.Validate(loader.Context); err != nil {
		return nil, fmt.Errorf("invalid OpenAPI spec: %w", err)
	}
	
	return &Specification{doc}, nil
}

// GetOperations returns all operations in the specification
func (s *Specification) GetOperations() []Operation {
	var operations []Operation
	
	// Iterate over all paths
	for path, pathItem := range s.Paths.Map() {
		// Iterate over all methods
		for method, operation := range pathItem.Operations() {
			// Create an Operation object
			op := Operation{
				ID:          operation.OperationID,
				Method:      method,
				Path:        path,
				Summary:     operation.Summary,
				Description: operation.Description,
				Tags:        operation.Tags,
			}
			
			// Process parameters
			for _, param := range operation.Parameters {
				if param.Value == nil {
					continue
				}
				
				parameter := Parameter{
					Name:        param.Value.Name,
					In:          param.Value.In,
					Required:    param.Value.Required,
					Description: param.Value.Description,
				}
				
				if param.Value.Schema != nil && param.Value.Schema.Value != nil {
					parameter.Type = param.Value.Schema.Value.Type
					parameter.Format = param.Value.Schema.Value.Format
				}
				
				op.Parameters = append(op.Parameters, parameter)
			}
			
			// Process request body
			if operation.RequestBody != nil && operation.RequestBody.Value != nil {
				for contentType, mediaType := range operation.RequestBody.Value.Content {
					if mediaType.Schema != nil && mediaType.Schema.Value != nil {
						op.RequestBody = &RequestBody{
							ContentType: contentType,
							Required:    operation.RequestBody.Value.Required,
							Schema:      mediaType.Schema.Value,
						}
						break // Use first content type for now
					}
				}
			}
			
			// Process responses
			for statusCode, response := range operation.Responses {
				if response.Value == nil {
					continue
				}
				
				resp := Response{
					StatusCode:  statusCode,
					Description: response.Value.Description,
				}
				
				// Get response schema from first content type
				for contentType, mediaType := range response.Value.Content {
					if mediaType.Schema != nil && mediaType.Schema.Value != nil {
						resp.ContentType = contentType
						resp.Schema = mediaType.Schema.Value
						break
					}
				}
				
				op.Responses = append(op.Responses, resp)
			}
			
			operations = append(operations, op)
		}
	}
	
	return operations
}

// GetDefinitions returns all schema definitions in the specification
func (s *Specification) GetDefinitions() map[string]*openapi3.Schema {
	definitions := make(map[string]*openapi3.Schema)
	
	// Process all Components.Schemas
	for name, schemaRef := range s.Components.Schemas {
		if schemaRef.Value != nil {
			definitions[name] = schemaRef.Value
		}
	}
	
	return definitions
}

// GetAuthSchemes returns all security schemes in the specification
func (s *Specification) GetAuthSchemes() map[string]*openapi3.SecurityScheme {
	schemes := make(map[string]*openapi3.SecurityScheme)
	
	// Process all Components.SecuritySchemes
	for name, schemeRef := range s.Components.SecuritySchemes {
		if schemeRef.Value != nil {
			schemes[name] = schemeRef.Value
		}
	}
	
	return schemes
}

// Operation represents an API operation
type Operation struct {
	ID          string
	Method      string
	Path        string
	Summary     string
	Description string
	Tags        []string
	Parameters  []Parameter
	RequestBody *RequestBody
	Responses   []Response
}

// Parameter represents an operation parameter
type Parameter struct {
	Name        string
	In          string
	Required    bool
	Description string
	Type        string
	Format      string
}

// RequestBody represents an operation request body
type RequestBody struct {
	ContentType string
	Required    bool
	Schema      *openapi3.Schema
}

// Response represents an operation response
type Response struct {
	StatusCode  string
	Description string
	ContentType string
	Schema      *openapi3.Schema
}