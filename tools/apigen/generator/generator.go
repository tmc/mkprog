package generator

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/dave/jennifer/jen"
	"github.com/tmc/mkprog/tools/apigen/openapi"
)

// Options represents generator options
type Options struct {
	PackageName    string        // Name of the generated package
	Prefix         string        // Prefix for generated type names
	Features       []string      // List of features to include
	ClientTimeout  time.Duration // Default client timeout
	DryRun         bool          // Whether to dry-run
	Verbose        bool          // Whether to show verbose output
}

// Generator is responsible for generating Go code from an OpenAPI spec
type Generator struct {
	options Options
}

// New creates a new generator
func New(options Options) *Generator {
	return &Generator{
		options: options,
	}
}

// GenerateClient generates a complete API client
func (g *Generator) GenerateClient(ctx context.Context, spec *openapi.Specification) (map[string]string, error) {
	// Initialize result map
	files := make(map[string]string)
	
	// Convert schemas to Go types
	converter := openapi.NewTypeConverter(g.options.Prefix)
	goTypes, err := converter.ConvertDefinitions(spec.GetDefinitions())
	if err != nil {
		return nil, fmt.Errorf("failed to convert definitions: %w", err)
	}
	
	// Check which features to include
	hasFeature := make(map[string]bool)
	for _, feature := range g.options.Features {
		hasFeature[feature] = true
	}
	
	// Generate model files if needed
	if hasFeature["models"] {
		modelsFile, err := g.generateModels(spec, goTypes)
		if err != nil {
			return nil, fmt.Errorf("failed to generate models: %w", err)
		}
		files["models.go"] = modelsFile
	}
	
	// Generate client if needed
	if hasFeature["client"] {
		clientFile, err := g.generateClient(spec, goTypes)
		if err != nil {
			return nil, fmt.Errorf("failed to generate client: %w", err)
		}
		files["client.go"] = clientFile
	}
	
	// Generate operations if needed
	if hasFeature["operations"] {
		operations := spec.GetOperations()
		for _, group := range groupOperationsByTag(operations) {
			opFile, err := g.generateOperations(spec, group.tag, group.operations, goTypes)
			if err != nil {
				return nil, fmt.Errorf("failed to generate operations for tag %s: %w", group.tag, err)
			}
			
			fileName := fmt.Sprintf("operations_%s.go", strings.ToLower(group.tag))
			files[fileName] = opFile
		}
	}
	
	// Generate auth if needed
	if hasFeature["auth"] {
		authFile, err := g.generateAuth(spec)
		if err != nil {
			return nil, fmt.Errorf("failed to generate auth: %w", err)
		}
		files["auth.go"] = authFile
	}
	
	// Generate middleware/interceptors
	if hasFeature["ratelimit"] {
		ratelimitFile, err := g.generateRateLimiter()
		if err != nil {
			return nil, fmt.Errorf("failed to generate rate limiter: %w", err)
		}
		files["ratelimit.go"] = ratelimitFile
	}
	
	if hasFeature["retry"] {
		retryFile, err := g.generateRetry()
		if err != nil {
			return nil, fmt.Errorf("failed to generate retry: %w", err)
		}
		files["retry.go"] = retryFile
	}
	
	if hasFeature["logging"] {
		loggingFile, err := g.generateLogging()
		if err != nil {
			return nil, fmt.Errorf("failed to generate logging: %w", err)
		}
		files["logging.go"] = loggingFile
	}
	
	// Generate tests/examples
	if hasFeature["tests"] {
		testsFile, err := g.generateTests(spec, goTypes)
		if err != nil {
			return nil, fmt.Errorf("failed to generate tests: %w", err)
		}
		files["client_test.go"] = testsFile
	}
	
	if hasFeature["examples"] {
		examplesFile, err := g.generateExamples(spec, goTypes)
		if err != nil {
			return nil, fmt.Errorf("failed to generate examples: %w", err)
		}
		files["examples_test.go"] = examplesFile
	}
	
	// Generate README.md
	readmeFile, err := g.generateReadme(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to generate readme: %w", err)
	}
	files["README.md"] = readmeFile
	
	return files, nil
}

// generateModels generates model files
func (g *Generator) generateModels(spec *openapi.Specification, goTypes map[string]openapi.GoType) (string, error) {
	file := jen.NewFile(g.options.PackageName)
	
	// Add standard imports
	file.ImportName("encoding/json", "json")
	file.ImportName("fmt", "fmt")
	
	// Track imports needed
	imports := make(map[string]bool)
	
	// Generate each model
	for name, goType := range goTypes {
		if goType.Import != "" {
			imports[goType.Import] = true
		}
		
		// Generate struct definition
		structDef := jen.Type().Id(name).Struct()
		
		// Add properties to struct
		for _, prop := range goType.Properties {
			fieldName := strings.Title(prop.Name)
			fieldType := jen.Id(prop.Name)
			
			// Add JSON tag
			tag := fmt.Sprintf(`json:"%s,omitempty"`, prop.Name)
			
			// Add the field to the struct
			structDef.Id(fieldName).Add(fieldType).Tag(map[string]string{"json": fmt.Sprintf("%s,omitempty", prop.Name)})
		}
		
		// Add the struct to the file
		file.Add(structDef)
		
		// Add a String() method for each struct
		file.Func().Params(
			jen.Id("m").Op("*").Id(name),
		).Id("String").Params().String().Block(
			jen.List(jen.Id("b"), jen.Err()).Op(":=").Qual("encoding/json", "Marshal").Call(jen.Id("m")),
			jen.If(jen.Err().Op("!=").Nil()).Block(
				jen.Return(jen.Lit("error marshaling to JSON: " + "\" + err.Error()")),
			),
			jen.Return(jen.Qual("string", "").Call(jen.Id("b"))),
		)
	}
	
	// Add imports
	for imp := range imports {
		file.ImportName(imp, filepath.Base(imp))
	}
	
	return file.GoString(), nil
}

// generateClient generates the base client
func (g *Generator) generateClient(spec *openapi.Specification, goTypes map[string]openapi.GoType) (string, error) {
	file := jen.NewFile(g.options.PackageName)
	
	// Add standard imports
	file.ImportName("bytes", "bytes")
	file.ImportName("context", "context")
	file.ImportName("encoding/json", "json")
	file.ImportName("fmt", "fmt")
	file.ImportName("io", "io")
	file.ImportName("io/ioutil", "ioutil")
	file.ImportName("net/http", "http")
	file.ImportName("net/url", "url")
	file.ImportName("time", "time")
	
	// Define ClientOptions struct
	file.Type().Id("ClientOptions").Struct(
		jen.Id("BaseURL").String().Tag(map[string]string{"json": "baseURL"}),
		jen.Id("HTTPClient").Op("*").Qual("net/http", "Client").Tag(map[string]string{"json": "httpClient"}),
		jen.Id("Timeout").Qual("time", "Duration").Tag(map[string]string{"json": "timeout"}),
		jen.Id("UserAgent").String().Tag(map[string]string{"json": "userAgent"}),
		jen.Id("Debug").Bool().Tag(map[string]string{"json": "debug"}),
	)
	
	// Define Client struct
	file.Type().Id("Client").Struct(
		jen.Id("options").Id("ClientOptions"),
		jen.Id("httpClient").Op("*").Qual("net/http", "Client"),
	)
	
	// Define NewClient function
	file.Func().Id("NewClient").Params(
		jen.Id("options").Id("ClientOptions"),
	).Op("*").Id("Client").Block(
		jen.If(jen.Id("options").Dot("HTTPClient").Op("==").Nil()).Block(
			jen.Id("options").Dot("HTTPClient").Op("=").Op("&").Qual("net/http", "Client").Values(
				jen.Id("Timeout").Op(":").Id("options").Dot("Timeout"),
			),
		),
		jen.If(jen.Id("options").Dot("Timeout").Op("==").Lit(0)).Block(
			jen.Id("options").Dot("Timeout").Op("=").Lit(10).Op("*").Qual("time", "Second"),
		),
		jen.If(jen.Id("options").Dot("UserAgent").Op("==").Lit("")).Block(
			jen.Id("options").Dot("UserAgent").Op("=").Lit(fmt.Sprintf("%s/%s", g.options.PackageName, spec.Info.Version)),
		),
		jen.Return(jen.Op("&").Id("Client").Values(
			jen.Id("options").Op(":").Id("options"),
			jen.Id("httpClient").Op(":").Id("options").Dot("HTTPClient"),
		)),
	)
	
	// Add helper methods
	
	// NewRequest method
	file.Func().Params(
		jen.Id("c").Op("*").Id("Client"),
	).Id("NewRequest").Params(
		jen.Id("ctx").Qual("context", "Context"),
		jen.Id("method").String(),
		jen.Id("path").String(),
		jen.Id("body").Interface(),
	).Params(
		jen.Op("*").Qual("net/http", "Request"),
		jen.Error(),
	).Block(
		jen.Id("u").Op(":=").Id("c").Dot("options").Dot("BaseURL").Op("+").Id("path"),
		jen.List(jen.Id("reqURL"), jen.Err()).Op(":=").Qual("net/url", "Parse").Call(jen.Id("u")),
		jen.If(jen.Err().Op("!=").Nil()).Block(
			jen.Return(jen.Nil(), jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to parse URL: %v"), jen.Err())),
		),
		jen.Var().Id("reqBody").Qual("io", "Reader").Op("=").Nil(),
		jen.If(jen.Id("body").Op("!=").Nil()).Block(
			jen.List(jen.Id("bodyBytes"), jen.Err()).Op(":=").Qual("encoding/json", "Marshal").Call(jen.Id("body")),
			jen.If(jen.Err().Op("!=").Nil()).Block(
				jen.Return(jen.Nil(), jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to marshal request body: %v"), jen.Err())),
			),
			jen.Id("reqBody").Op("=").Qual("bytes", "NewBuffer").Call(jen.Id("bodyBytes")),
		),
		jen.List(jen.Id("req"), jen.Err()).Op(":=").Qual("net/http", "NewRequestWithContext").Call(
			jen.Id("ctx"),
			jen.Id("method"),
			jen.Id("reqURL").Dot("String").Call(),
			jen.Id("reqBody"),
		),
		jen.If(jen.Err().Op("!=").Nil()).Block(
			jen.Return(jen.Nil(), jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to create request: %v"), jen.Err())),
		),
		jen.Id("req").Dot("Header").Dot("Set").Call(jen.Lit("Content-Type"), jen.Lit("application/json")),
		jen.Id("req").Dot("Header").Dot("Set").Call(jen.Lit("Accept"), jen.Lit("application/json")),
		jen.Id("req").Dot("Header").Dot("Set").Call(jen.Lit("User-Agent"), jen.Id("c").Dot("options").Dot("UserAgent")),
		jen.Return(jen.Id("req"), jen.Nil()),
	)
	
	// Do method
	file.Func().Params(
		jen.Id("c").Op("*").Id("Client"),
	).Id("Do").Params(
		jen.Id("req").Op("*").Qual("net/http", "Request"),
		jen.Id("v").Interface(),
	).Params(
		jen.Op("*").Qual("net/http", "Response"),
		jen.Error(),
	).Block(
		jen.List(jen.Id("resp"), jen.Err()).Op(":=").Id("c").Dot("httpClient").Dot("Do").Call(jen.Id("req")),
		jen.If(jen.Err().Op("!=").Nil()).Block(
			jen.Return(jen.Nil(), jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to send request: %v"), jen.Err())),
		),
		jen.Defer().Id("resp").Dot("Body").Dot("Close").Call(),
		jen.Id("body").Op(":=").Id("resp").Dot("Body"),
		jen.List(jen.Id("respBody"), jen.Err()).Op(":=").Qual("io/ioutil", "ReadAll").Call(jen.Id("body")),
		jen.If(jen.Err().Op("!=").Nil()).Block(
			jen.Return(jen.Id("resp"), jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to read response body: %v"), jen.Err())),
		),
		jen.If(jen.Id("resp").Dot("StatusCode").Op(">=").Lit(400)).Block(
			jen.Var().Id("apiErr").Id("APIError"),
			jen.Err().Op("=").Qual("encoding/json", "Unmarshal").Call(jen.Id("respBody"), jen.Op("&").Id("apiErr")),
			jen.If(jen.Err().Op("==").Nil()).Block(
				jen.Return(jen.Id("resp"), jen.Op("&").Id("apiErr")),
			),
			jen.Return(jen.Id("resp"), jen.Qual("fmt", "Errorf").Call(jen.Lit("HTTP error: %s (%d)"), jen.Id("resp").Dot("Status"), jen.Id("resp").Dot("StatusCode"))),
		),
		jen.If(jen.Id("v").Op("!=").Nil()).Block(
			jen.If(jen.Id("w").Op(",").Id("ok").Op(":=").Id("v").Assert(jen.Qual("io", "Writer")), jen.Id("ok")).Block(
				jen.List(jen.Id("_"), jen.Err()).Op("=").Id("w").Dot("Write").Call(jen.Id("respBody")),
				jen.Return(jen.Id("resp"), jen.Err()),
			),
			jen.Err().Op("=").Qual("encoding/json", "Unmarshal").Call(jen.Id("respBody"), jen.Id("v")),
			jen.If(jen.Err().Op("!=").Nil()).Block(
				jen.Return(jen.Id("resp"), jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to unmarshal response: %v"), jen.Err())),
			),
		),
		jen.Return(jen.Id("resp"), jen.Nil()),
	)
	
	// Define APIError
	file.Type().Id("APIError").Struct(
		jen.Id("Code").Int().Tag(map[string]string{"json": "code"}),
		jen.Id("Message").String().Tag(map[string]string{"json": "message"}),
	)
	
	// Error method for APIError
	file.Func().Params(
		jen.Id("e").Op("*").Id("APIError"),
	).Id("Error").Params().String().Block(
		jen.Return(jen.Qual("fmt", "Sprintf").Call(
			jen.Lit("API error: %s (code: %d)"),
			jen.Id("e").Dot("Message"),
			jen.Id("e").Dot("Code"),
		)),
	)
	
	return file.GoString(), nil
}

// Dummy implementations of other generator functions to be implemented later
func (g *Generator) generateOperations(spec *openapi.Specification, tag string, operations []openapi.Operation, goTypes map[string]openapi.GoType) (string, error) {
	file := jen.NewFile(g.options.PackageName)
	
	// Add standard imports
	file.ImportName("context", "context")
	file.ImportName("fmt", "fmt")
	file.ImportName("net/http", "http")
	file.ImportName("net/url", "url")
	file.ImportName("strings", "strings")
	
	// Add operations for this tag
	for _, op := range operations {
		g.generateOperation(file, op)
	}
	
	return file.GoString(), nil
}

func (g *Generator) generateOperation(file *jen.File, op openapi.Operation) {
	// Create function name
	funcName := toCamelCase(op.ID)
	if funcName == "" {
		// Generate a name from method and path if no operation ID
		parts := strings.Split(op.Path, "/")
		var nameParts []string
		for _, part := range parts {
			if part != "" && !strings.HasPrefix(part, "{") {
				nameParts = append(nameParts, part)
			}
		}
		
		if len(nameParts) > 0 {
			funcName = toCamelCase(op.Method + strings.Join(nameParts, ""))
		} else {
			funcName = toCamelCase(op.Method + "Root")
		}
	}
	
	// Function parameters
	params := []jen.Code{
		jen.Id("c").Op("*").Id("Client"),
		jen.Id("ctx").Qual("context", "Context"),
	}
	
	// Add request parameters
	for _, param := range op.Parameters {
		if param.In == "path" || param.In == "query" {
			typeExpr := jen.String()
			switch param.Type {
			case "integer":
				typeExpr = jen.Int()
			case "boolean":
				typeExpr = jen.Bool()
			}
			
			params = append(params, jen.Id(toCamelCase(param.Name)).Add(typeExpr))
		}
	}
	
	// Add request body if needed
	if op.RequestBody != nil {
		params = append(params, jen.Id("request").Interface())
	}
	
	// Function return values
	returns := []jen.Code{
		jen.Op("*").Qual("net/http", "Response"),
		jen.Error(),
	}
	
	// If there's a successful response, add it to returns
	var successResponse *openapi.Response
	for _, resp := range op.Responses {
		if strings.HasPrefix(resp.StatusCode, "2") {
			successResponse = &resp
			break
		}
	}
	
	// Add return type for successful response
	if successResponse != nil && successResponse.Schema != nil {
		// For 204 No Content, don't return a response type
		if successResponse.StatusCode != "204" {
			returns = []jen.Code{
				jen.Interface(), // Response type placeholder
				jen.Op("*").Qual("net/http", "Response"),
				jen.Error(),
			}
		}
	}
	
	// Function definition
	funcDef := jen.Func().Params(
		jen.Id("c").Op("*").Id("Client"),
	).Id(funcName).Params(params...).Params(returns...).Block(
		// Placeholder implementation
		jen.Var().Id("urlPath").Op("=").Lit(op.Path),
		
		// Replace path parameters
		func() jen.Code {
			var statements []jen.Code
			for _, param := range op.Parameters {
				if param.In == "path" {
					statements = append(statements, 
						jen.Id("urlPath").Op("=").Qual("strings", "Replace").Call(
							jen.Id("urlPath"),
							jen.Lit(fmt.Sprintf("{%s}", param.Name)),
							jen.Qual("fmt", "Sprintf").Call(jen.Lit("%v"), jen.Id(toCamelCase(param.Name))),
							jen.Lit(1),
						),
					)
				}
			}
			if len(statements) > 0 {
				return jen.Add(statements...)
			}
			return jen.Empty()
		}(),
		
		// Build query parameters
		func() jen.Code {
			var hasQueryParams bool
			for _, param := range op.Parameters {
				if param.In == "query" {
					hasQueryParams = true
					break
				}
			}
			
			if hasQueryParams {
				var statements []jen.Code
				statements = append(statements, 
					jen.List(jen.Id("baseURL"), jen.Err()).Op(":=").Qual("url", "Parse").Call(jen.Id("urlPath")),
					jen.If(jen.Err().Op("!=").Nil()).Block(
						jen.Return(jen.List(jen.Nil(), jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to parse URL: %v"), jen.Err()))),
					),
					jen.Id("query").Op(":=").Id("baseURL").Dot("Query").Call(),
				)
				
				for _, param := range op.Parameters {
					if param.In == "query" {
						statements = append(statements, 
							jen.Id("query").Dot("Set").Call(
								jen.Lit(param.Name),
								jen.Qual("fmt", "Sprintf").Call(jen.Lit("%v"), jen.Id(toCamelCase(param.Name))),
							),
						)
					}
				}
				
				statements = append(statements,
					jen.Id("baseURL").Dot("RawQuery").Op("=").Id("query").Dot("Encode").Call(),
					jen.Id("urlPath").Op("=").Id("baseURL").Dot("String").Call(),
				)
				
				return jen.Add(statements...)
			}
			return jen.Empty()
		}(),
		
		// Create the request
		func() jen.Code {
			if op.RequestBody != nil {
				return jen.List(jen.Id("req"), jen.Err()).Op(":=").Id("c").Dot("NewRequest").Call(
					jen.Id("ctx"),
					jen.Lit(strings.ToUpper(op.Method)),
					jen.Id("urlPath"),
					jen.Id("request"),
				)
			}
			return jen.List(jen.Id("req"), jen.Err()).Op(":=").Id("c").Dot("NewRequest").Call(
				jen.Id("ctx"),
				jen.Lit(strings.ToUpper(op.Method)),
				jen.Id("urlPath"),
				jen.Nil(),
			)
		}(),
		
		jen.If(jen.Err().Op("!=").Nil()).Block(
			jen.Return(jen.List(jen.Nil(), jen.Err())),
		),
		
		// Execute the request
		func() jen.Code {
			if successResponse != nil && successResponse.Schema != nil && successResponse.StatusCode != "204" {
				return jen.Var().Id("response").Interface()
			}
			return jen.Empty()
		}(),
		
		// Execute the request and handle response
		func() jen.Code {
			if successResponse != nil && successResponse.Schema != nil && successResponse.StatusCode != "204" {
				return jen.List(jen.Id("resp"), jen.Err()).Op(":=").Id("c").Dot("Do").Call(
					jen.Id("req"),
					jen.Op("&").Id("response"),
				)
			} else {
				return jen.List(jen.Id("resp"), jen.Err()).Op(":=").Id("c").Dot("Do").Call(
					jen.Id("req"),
					jen.Nil(),
				)
			}
		}(),
		
		jen.If(jen.Err().Op("!=").Nil()).Block(
			jen.Return(jen.List(jen.Nil(), jen.Err())),
		),
		
		// Return the response
		func() jen.Code {
			if successResponse != nil && successResponse.Schema != nil && successResponse.StatusCode != "204" {
				return jen.Return(jen.List(jen.Id("response"), jen.Id("resp"), jen.Nil()))
			}
			return jen.Return(jen.List(jen.Id("resp"), jen.Nil()))
		}(),
	)
	
	// Add function to file
	file.Add(funcDef)
}

func (g *Generator) generateAuth(spec *openapi.Specification) (string, error) {
	file := jen.NewFile(g.options.PackageName)
	
	// Add standard imports
	file.ImportName("context", "context")
	file.ImportName("net/http", "http")
	
	// AuthType enum
	file.Type().Id("AuthType").String()
	file.Const().Defs(
		jen.Id("AuthTypeNone").Id("AuthType").Op("=").Lit("none"),
		jen.Id("AuthTypeBasic").Id("AuthType").Op("=").Lit("basic"),
		jen.Id("AuthTypeAPIKey").Id("AuthType").Op("=").Lit("apiKey"),
		jen.Id("AuthTypeBearer").Id("AuthType").Op("=").Lit("bearer"),
	)
	
	// AuthConfig struct
	file.Type().Id("AuthConfig").Struct(
		jen.Id("Type").Id("AuthType").Tag(map[string]string{"json": "type"}),
		jen.Id("Username").String().Tag(map[string]string{"json": "username"}),
		jen.Id("Password").String().Tag(map[string]string{"json": "password"}),
		jen.Id("APIKey").String().Tag(map[string]string{"json": "apiKey"}),
		jen.Id("APIKeyHeader").String().Tag(map[string]string{"json": "apiKeyHeader"}),
		jen.Id("BearerToken").String().Tag(map[string]string{"json": "bearerToken"}),
	)
	
	// Authentication interface
	file.Type().Id("Authentication").Interface(
		jen.Id("Apply").Params(
			jen.Id("req").Op("*").Qual("net/http", "Request"),
		),
	)
	
	// BasicAuth
	file.Type().Id("BasicAuth").Struct(
		jen.Id("Username").String(),
		jen.Id("Password").String(),
	)
	
	file.Func().Params(
		jen.Id("a").Id("BasicAuth"),
	).Id("Apply").Params(
		jen.Id("req").Op("*").Qual("net/http", "Request"),
	).Block(
		jen.Id("req").Dot("SetBasicAuth").Call(
			jen.Id("a").Dot("Username"),
			jen.Id("a").Dot("Password"),
		),
	)
	
	// APIKeyAuth
	file.Type().Id("APIKeyAuth").Struct(
		jen.Id("Key").String(),
		jen.Id("Header").String(),
	)
	
	file.Func().Params(
		jen.Id("a").Id("APIKeyAuth"),
	).Id("Apply").Params(
		jen.Id("req").Op("*").Qual("net/http", "Request"),
	).Block(
		jen.Id("req").Dot("Header").Dot("Set").Call(
			jen.Id("a").Dot("Header"),
			jen.Id("a").Dot("Key"),
		),
	)
	
	// BearerAuth
	file.Type().Id("BearerAuth").Struct(
		jen.Id("Token").String(),
	)
	
	file.Func().Params(
		jen.Id("a").Id("BearerAuth"),
	).Id("Apply").Params(
		jen.Id("req").Op("*").Qual("net/http", "Request"),
	).Block(
		jen.Id("req").Dot("Header").Dot("Set").Call(
			jen.Lit("Authorization"),
			jen.Lit("Bearer ").Op("+").Id("a").Dot("Token"),
		),
	)
	
	// AuthFromConfig
	file.Func().Id("AuthFromConfig").Params(
		jen.Id("config").Id("AuthConfig"),
	).Id("Authentication").Block(
		jen.Switch(jen.Id("config").Dot("Type")).Block(
			jen.Case(jen.Id("AuthTypeBasic")).Block(
				jen.Return(jen.Id("BasicAuth").Values(
					jen.Dict{
						jen.Id("Username"): jen.Id("config").Dot("Username"),
						jen.Id("Password"): jen.Id("config").Dot("Password"),
					},
				)),
			),
			jen.Case(jen.Id("AuthTypeAPIKey")).Block(
				jen.Return(jen.Id("APIKeyAuth").Values(
					jen.Dict{
						jen.Id("Key"):    jen.Id("config").Dot("APIKey"),
						jen.Id("Header"): jen.Id("config").Dot("APIKeyHeader"),
					},
				)),
			),
			jen.Case(jen.Id("AuthTypeBearer")).Block(
				jen.Return(jen.Id("BearerAuth").Values(
					jen.Dict{
						jen.Id("Token"): jen.Id("config").Dot("BearerToken"),
					},
				)),
			),
			jen.Default().Block(
				jen.Return(jen.Nil()),
			),
		),
	)
	
	return file.GoString(), nil
}

func (g *Generator) generateRateLimiter() (string, error) {
	file := jen.NewFile(g.options.PackageName)
	
	// Add standard imports
	file.ImportName("context", "context")
	file.ImportName("net/http", "http")
	file.ImportName("time", "time")
	file.ImportName("golang.org/x/time/rate", "rate")
	
	// RateLimiter struct
	file.Type().Id("RateLimiter").Struct(
		jen.Id("limiter").Op("*").Qual("golang.org/x/time/rate", "Limiter"),
	)
	
	// NewRateLimiter function
	file.Func().Id("NewRateLimiter").Params(
		jen.Id("rps").Float64(),
		jen.Id("burst").Int(),
	).Op("*").Id("RateLimiter").Block(
		jen.Return(jen.Op("&").Id("RateLimiter").Values(
			jen.Dict{
				jen.Id("limiter"): jen.Qual("golang.org/x/time/rate", "NewLimiter").Call(
					jen.Qual("golang.org/x/time/rate", "Limit").Call(jen.Id("rps")),
					jen.Id("burst"),
				),
			},
		)),
	)
	
	// RoundTripper implementation
	file.Type().Id("rateLimitedTransport").Struct(
		jen.Id("limiter").Op("*").Id("RateLimiter"),
		jen.Id("transport").Qual("net/http", "RoundTripper"),
	)
	
	file.Func().Params(
		jen.Id("t").Op("*").Id("rateLimitedTransport"),
	).Id("RoundTrip").Params(
		jen.Id("req").Op("*").Qual("net/http", "Request"),
	).Params(
		jen.Op("*").Qual("net/http", "Response"),
		jen.Error(),
	).Block(
		jen.Err().Op(":=").Id("t").Dot("limiter").Dot("limiter").Dot("Wait").Call(jen.Id("req").Dot("Context").Call()),
		jen.If(jen.Err().Op("!=").Nil()).Block(
			jen.Return(jen.Nil(), jen.Err()),
		),
		jen.Return(jen.Id("t").Dot("transport").Dot("RoundTrip").Call(jen.Id("req"))),
	)
	
	// LimitClient method
	file.Func().Params(
		jen.Id("r").Op("*").Id("RateLimiter"),
	).Id("LimitClient").Params(
		jen.Id("client").Op("*").Qual("net/http", "Client"),
	).Block(
		jen.Id("transport").Op(":=").Id("client").Dot("Transport"),
		jen.If(jen.Id("transport").Op("==").Nil()).Block(
			jen.Id("transport").Op("=").Qual("net/http", "DefaultTransport"),
		),
		jen.Id("client").Dot("Transport").Op("=").Op("&").Id("rateLimitedTransport").Values(
			jen.Dict{
				jen.Id("limiter"):   jen.Id("r"),
				jen.Id("transport"): jen.Id("transport"),
			},
		),
	)
	
	return file.GoString(), nil
}

func (g *Generator) generateRetry() (string, error) {
	file := jen.NewFile(g.options.PackageName)
	
	// Add standard imports
	file.ImportName("context", "context")
	file.ImportName("net/http", "http")
	file.ImportName("time", "time")
	file.ImportName("math/rand", "rand")
	
	// RetryConfig struct
	file.Type().Id("RetryConfig").Struct(
		jen.Id("MaxRetries").Int().Tag(map[string]string{"json": "maxRetries"}),
		jen.Id("RetryWaitMin").Qual("time", "Duration").Tag(map[string]string{"json": "retryWaitMin"}),
		jen.Id("RetryWaitMax").Qual("time", "Duration").Tag(map[string]string{"json": "retryWaitMax"}),
		jen.Id("RetryStatusCodes").Index().Int().Tag(map[string]string{"json": "retryStatusCodes"}),
	)
	
	// DefaultRetryConfig function
	file.Func().Id("DefaultRetryConfig").Params().Id("RetryConfig").Block(
		jen.Return(jen.Id("RetryConfig").Values(
			jen.Dict{
				jen.Id("MaxRetries"):      jen.Lit(3),
				jen.Id("RetryWaitMin"):    jen.Lit(100).Op("*").Qual("time", "Millisecond"),
				jen.Id("RetryWaitMax"):    jen.Lit(1000).Op("*").Qual("time", "Millisecond"),
				jen.Id("RetryStatusCodes"): jen.Index().Int().Values(
					jen.Lit(408), // Request Timeout
					jen.Lit(429), // Too Many Requests
					jen.Lit(500), // Internal Server Error
					jen.Lit(502), // Bad Gateway
					jen.Lit(503), // Service Unavailable
					jen.Lit(504), // Gateway Timeout
				),
			},
		)),
	)
	
	// RetryTransport struct
	file.Type().Id("RetryTransport").Struct(
		jen.Id("Transport").Qual("net/http", "RoundTripper"),
		jen.Id("Config").Id("RetryConfig"),
	)
	
	// NewRetryTransport function
	file.Func().Id("NewRetryTransport").Params(
		jen.Id("transport").Qual("net/http", "RoundTripper"),
		jen.Id("config").Id("RetryConfig"),
	).Op("*").Id("RetryTransport").Block(
		jen.If(jen.Id("transport").Op("==").Nil()).Block(
			jen.Id("transport").Op("=").Qual("net/http", "DefaultTransport"),
		),
		jen.Return(jen.Op("&").Id("RetryTransport").Values(
			jen.Dict{
				jen.Id("Transport"): jen.Id("transport"),
				jen.Id("Config"):    jen.Id("config"),
			},
		)),
	)
	
	// shouldRetry helper
	file.Func().Params(
		jen.Id("t").Op("*").Id("RetryTransport"),
	).Id("shouldRetry").Params(
		jen.Id("resp").Op("*").Qual("net/http", "Response"),
		jen.Id("err").Error(),
	).Bool().Block(
		jen.If(jen.Err().Op("!=").Nil()).Block(
			jen.Return(jen.True()),
		),
		jen.If(jen.Id("resp").Op("==").Nil()).Block(
			jen.Return(jen.False()),
		),
		jen.For(jen.List(jen.Id("_"), jen.Id("statusCode")).Op(":=").Range().Id("t").Dot("Config").Dot("RetryStatusCodes")).Block(
			jen.If(jen.Id("resp").Dot("StatusCode").Op("==").Id("statusCode")).Block(
				jen.Return(jen.True()),
			),
		),
		jen.Return(jen.False()),
	)
	
	// waitDuration helper
	file.Func().Params(
		jen.Id("t").Op("*").Id("RetryTransport"),
		jen.Id("attempt").Int(),
	).Id("waitDuration").Params().Qual("time", "Duration").Block(
		jen.Comment("Exponential backoff with jitter"),
		jen.Id("min").Op(":=").Id("t").Dot("Config").Dot("RetryWaitMin"),
		jen.Id("max").Op(":=").Id("t").Dot("Config").Dot("RetryWaitMax"),
		jen.Id("wait").Op(":=").Id("min").Op("*").Qual("time", "Duration").Call(jen.Lit(1)<<jen.Id("attempt")),
		jen.If(jen.Id("wait").Op(">").Id("max")).Block(
			jen.Id("wait").Op("=").Id("max"),
		),
		jen.Comment("Add jitter to avoid request collisions"),
		jen.Id("jitter").Op(":=").Qual("time", "Duration").Call(
			jen.Qual("math/rand", "Int63n").Call(jen.Int64().Call(jen.Id("wait"))),
		),
		jen.Return(jen.Id("wait").Op("-").Id("jitter").Op("/").Lit(2)),
	)
	
	// RoundTrip implementation
	file.Func().Params(
		jen.Id("t").Op("*").Id("RetryTransport"),
	).Id("RoundTrip").Params(
		jen.Id("req").Op("*").Qual("net/http", "Request"),
	).Params(
		jen.Op("*").Qual("net/http", "Response"),
		jen.Error(),
	).Block(
		jen.Var().Id("resp").Op("*").Qual("net/http", "Response"),
		jen.Var().Id("err").Error(),
		
		jen.For(jen.Id("attempt").Op(":=").Lit(0), jen.Id("attempt").Op("<").Id("t").Dot("Config").Dot("MaxRetries"), jen.Id("attempt").Op("++")).Block(
			jen.If(jen.Id("attempt").Op(">").Lit(0)).Block(
				jen.Comment("Sleep before retrying"),
				jen.Id("select").Block(
					jen.Case(jen.Id("req").Dot("Context").Call().Dot("Done").Call().Op("<-")).Block(
						jen.Return(jen.Nil(), jen.Id("req").Dot("Context").Call().Dot("Err").Call()),
					),
					jen.Case(jen.Qual("time", "After").Call(jen.Id("t").Dot("waitDuration").Call(jen.Id("attempt"))).Op("<-")),
				),
				jen.Comment("Clone the request body for retries"),
				jen.If(jen.Id("req").Dot("Body").Op("!=").Nil()).Block(
					jen.Id("req").Dot("Body").Op("=").Qual("net/http", "DupCloseableBody").Call(jen.Id("req").Dot("Body")),
				),
			),
			jen.Comment("Make the request"),
			jen.Id("resp").Op(",").Id("err").Op("=").Id("t").Dot("Transport").Dot("RoundTrip").Call(jen.Id("req")),
			
			jen.If(jen.Op("!").Id("t").Dot("shouldRetry").Call(jen.Id("resp"), jen.Id("err"))).Block(
				jen.Return(jen.Id("resp"), jen.Id("err")),
			),
			
			jen.Comment("Close the response body if we're retrying"),
			jen.If(jen.Id("resp").Op("!=").Nil().Op("&&").Id("resp").Dot("Body").Op("!=").Nil()).Block(
				jen.Id("resp").Dot("Body").Dot("Close").Call(),
			),
		),
		
		jen.Return(jen.Id("resp"), jen.Id("err")),
	)
	
	// EnableRetries client method
	file.Func().Id("EnableRetries").Params(
		jen.Id("client").Op("*").Id("Client"),
		jen.Id("config").Id("RetryConfig"),
	).Block(
		jen.Id("transport").Op(":=").Id("client").Dot("httpClient").Dot("Transport"),
		jen.If(jen.Id("transport").Op("==").Nil()).Block(
			jen.Id("transport").Op("=").Qual("net/http", "DefaultTransport"),
		),
		jen.Id("client").Dot("httpClient").Dot("Transport").Op("=").Id("NewRetryTransport").Call(
			jen.Id("transport"),
			jen.Id("config"),
		),
	)
	
	return file.GoString(), nil
}

func (g *Generator) generateLogging() (string, error) {
	file := jen.NewFile(g.options.PackageName)
	
	// Add standard imports
	file.ImportName("log", "log")
	file.ImportName("net/http", "http")
	file.ImportName("net/http/httputil", "httputil")
	file.ImportName("time", "time")
	
	// LoggingTransport struct
	file.Type().Id("LoggingTransport").Struct(
		jen.Id("Transport").Qual("net/http", "RoundTripper"),
		jen.Id("Logger").Op("*").Qual("log", "Logger"),
		jen.Id("LogHeaders").Bool(),
		jen.Id("LogBody").Bool(),
	)
	
	// NewLoggingTransport function
	file.Func().Id("NewLoggingTransport").Params(
		jen.Id("transport").Qual("net/http", "RoundTripper"),
		jen.Id("logger").Op("*").Qual("log", "Logger"),
		jen.Id("logHeaders").Bool(),
		jen.Id("logBody").Bool(),
	).Op("*").Id("LoggingTransport").Block(
		jen.If(jen.Id("transport").Op("==").Nil()).Block(
			jen.Id("transport").Op("=").Qual("net/http", "DefaultTransport"),
		),
		jen.If(jen.Id("logger").Op("==").Nil()).Block(
			jen.Id("logger").Op("=").Qual("log", "Default").Call(),
		),
		jen.Return(jen.Op("&").Id("LoggingTransport").Values(
			jen.Dict{
				jen.Id("Transport"):  jen.Id("transport"),
				jen.Id("Logger"):     jen.Id("logger"),
				jen.Id("LogHeaders"): jen.Id("logHeaders"),
				jen.Id("LogBody"):    jen.Id("logBody"),
			},
		)),
	)
	
	// RoundTrip implementation
	file.Func().Params(
		jen.Id("t").Op("*").Id("LoggingTransport"),
	).Id("RoundTrip").Params(
		jen.Id("req").Op("*").Qual("net/http", "Request"),
	).Params(
		jen.Op("*").Qual("net/http", "Response"),
		jen.Error(),
	).Block(
		jen.Comment("Log the request"),
		jen.Id("t").Dot("Logger").Dot("Printf").Call(
			jen.Lit("Request: %s %s"),
			jen.Id("req").Dot("Method"),
			jen.Id("req").Dot("URL"),
		),
		
		jen.If(jen.Id("t").Dot("LogHeaders").Op("||").Id("t").Dot("LogBody")).Block(
			jen.Id("reqDump").Op(",").Id("err").Op(":=").Qual("net/http/httputil", "DumpRequestOut").Call(
				jen.Id("req"),
				jen.Id("t").Dot("LogBody"),
			),
			jen.If(jen.Err().Op("==").Nil()).Block(
				jen.Id("t").Dot("Logger").Dot("Printf").Call(
					jen.Lit("Request dump:\n%s"),
					jen.Qual("string", "").Call(jen.Id("reqDump")),
				),
			),
		),
		
		jen.Comment("Measure request time"),
		jen.Id("start").Op(":=").Qual("time", "Now").Call(),
		
		jen.Comment("Make the request"),
		jen.Id("resp").Op(",").Id("err").Op(":=").Id("t").Dot("Transport").Dot("RoundTrip").Call(jen.Id("req")),
		
		jen.Comment("Calculate duration"),
		jen.Id("duration").Op(":=").Qual("time", "Since").Call(jen.Id("start")),
		
		jen.If(jen.Err().Op("!=").Nil()).Block(
			jen.Id("t").Dot("Logger").Dot("Printf").Call(
				jen.Lit("Request error: %v"),
				jen.Err(),
			),
			jen.Return(jen.Id("resp"), jen.Err()),
		),
		
		jen.Comment("Log the response"),
		jen.Id("t").Dot("Logger").Dot("Printf").Call(
			jen.Lit("Response: %s %s -> %d (%s)"),
			jen.Id("req").Dot("Method"),
			jen.Id("req").Dot("URL"),
			jen.Id("resp").Dot("StatusCode"),
			jen.Id("duration"),
		),
		
		jen.If(jen.Id("t").Dot("LogHeaders").Op("||").Id("t").Dot("LogBody")).Block(
			jen.Id("respDump").Op(",").Id("err").Op(":=").Qual("net/http/httputil", "DumpResponse").Call(
				jen.Id("resp"),
				jen.Id("t").Dot("LogBody"),
			),
			jen.If(jen.Err().Op("==").Nil()).Block(
				jen.Id("t").Dot("Logger").Dot("Printf").Call(
					jen.Lit("Response dump:\n%s"),
					jen.Qual("string", "").Call(jen.Id("respDump")),
				),
			),
		),
		
		jen.Return(jen.Id("resp"), jen.Id("err")),
	)
	
	// EnableLogging client method
	file.Func().Id("EnableLogging").Params(
		jen.Id("client").Op("*").Id("Client"),
		jen.Id("logger").Op("*").Qual("log", "Logger"),
		jen.Id("logHeaders").Bool(),
		jen.Id("logBody").Bool(),
	).Block(
		jen.Id("transport").Op(":=").Id("client").Dot("httpClient").Dot("Transport"),
		jen.If(jen.Id("transport").Op("==").Nil()).Block(
			jen.Id("transport").Op("=").Qual("net/http", "DefaultTransport"),
		),
		jen.Id("client").Dot("httpClient").Dot("Transport").Op("=").Id("NewLoggingTransport").Call(
			jen.Id("transport"),
			jen.Id("logger"),
			jen.Id("logHeaders"),
			jen.Id("logBody"),
		),
	)
	
	return file.GoString(), nil
}

func (g *Generator) generateTests(spec *openapi.Specification, goTypes map[string]openapi.GoType) (string, error) {
	file := jen.NewFile(g.options.PackageName)
	
	// Add standard imports
	file.ImportName("context", "context")
	file.ImportName("net/http", "http")
	file.ImportName("net/http/httptest", "httptest")
	file.ImportName("testing", "testing")
	file.ImportName("time", "time")
	
	// Setup test server helper
	file.Func().Id("setupTestServer").Params(
		jen.Id("handler").Qual("net/http", "HandlerFunc"),
	).Params(
		jen.Op("*").Id("Client"),
		jen.Op("*").Qual("httptest", "Server"),
		jen.Func(),
	).Block(
		jen.Id("server").Op(":=").Qual("net/http", "httptest").Dot("NewServer").Call(jen.Id("handler")),
		
		jen.Id("client").Op(":=").Id("NewClient").Call(
			jen.Id("ClientOptions").Values(
				jen.Dict{
					jen.Id("BaseURL"):   jen.Id("server").Dot("URL"),
					jen.Id("Timeout"):   jen.Lit(5).Op("*").Qual("time", "Second"),
					jen.Id("UserAgent"): jen.Lit("test-client"),
				},
			),
		),
		
		jen.Id("cleanup").Op(":=").Func().Params().Block(
			jen.Id("server").Dot("Close").Call(),
		),
		
		jen.Return(jen.Id("client"), jen.Id("server"), jen.Id("cleanup")),
	)
	
	// TestNewClient function
	file.Func().Id("TestNewClient").Params(
		jen.Id("t").Op("*").Qual("testing", "T"),
	).Block(
		jen.Id("client").Op(":=").Id("NewClient").Call(
			jen.Id("ClientOptions").Values(
				jen.Dict{
					jen.Id("BaseURL"):   jen.Lit("https://api.example.com"),
					jen.Id("Timeout"):   jen.Lit(5).Op("*").Qual("time", "Second"),
					jen.Id("UserAgent"): jen.Lit("test-client"),
				},
			),
		),
		
		jen.If(jen.Id("client").Op("==").Nil()).Block(
			jen.Id("t").Dot("Fatal").Call(jen.Lit("expected client to not be nil")),
		),
		
		jen.If(jen.Id("client").Dot("options").Dot("BaseURL").Op("!=").Lit("https://api.example.com")).Block(
			jen.Id("t").Dot("Errorf").Call(
				jen.Lit("expected BaseURL to be %q, got %q"),
				jen.Lit("https://api.example.com"),
				jen.Id("client").Dot("options").Dot("BaseURL"),
			),
		),
		
		jen.If(jen.Id("client").Dot("options").Dot("Timeout").Op("!=").Lit(5).Op("*").Qual("time", "Second")).Block(
			jen.Id("t").Dot("Errorf").Call(
				jen.Lit("expected Timeout to be %s, got %s"),
				jen.Lit(5).Op("*").Qual("time", "Second"),
				jen.Id("client").Dot("options").Dot("Timeout"),
			),
		),
		
		jen.If(jen.Id("client").Dot("options").Dot("UserAgent").Op("!=").Lit("test-client")).Block(
			jen.Id("t").Dot("Errorf").Call(
				jen.Lit("expected UserAgent to be %q, got %q"),
				jen.Lit("test-client"),
				jen.Id("client").Dot("options").Dot("UserAgent"),
			),
		),
	)
	
	// TestClient_Do function
	file.Func().Id("TestClient_Do").Params(
		jen.Id("t").Op("*").Qual("testing", "T"),
	).Block(
		jen.Comment("Setup a test server"),
		jen.Id("client").Op(",").Id("_").Op(",").Id("cleanup").Op(":=").Id("setupTestServer").Call(
			jen.Func().Params(
				jen.Id("w").Qual("net/http", "ResponseWriter"),
				jen.Id("r").Op("*").Qual("net/http", "Request"),
			).Block(
				jen.Id("w").Dot("Header").Call().Dot("Set").Call(
					jen.Lit("Content-Type"),
					jen.Lit("application/json"),
				),
				jen.Id("w").Dot("WriteHeader").Call(jen.Qual("net/http", "StatusOK")),
				jen.Id("w").Dot("Write").Call(
					jen.Index().Byte().Call(jen.Lit(`{"message":"ok"}`)),
				),
			),
		),
		jen.Defer().Id("cleanup").Call(),
		
		jen.Comment("Create a test request"),
		jen.Id("req").Op(",").Id("err").Op(":=").Id("client").Dot("NewRequest").Call(
			jen.Qual("context", "Background").Call(),
			jen.Lit("GET"),
			jen.Lit("/test"),
			jen.Nil(),
		),
		jen.If(jen.Err().Op("!=").Nil()).Block(
			jen.Id("t").Dot("Fatalf").Call(
				jen.Lit("failed to create request: %v"),
				jen.Err(),
			),
		),
		
		jen.Comment("Execute the request"),
		jen.Id("resp").Op(",").Id("err").Op(":=").Id("client").Dot("Do").Call(
			jen.Id("req"),
			jen.Nil(),
		),
		jen.If(jen.Err().Op("!=").Nil()).Block(
			jen.Id("t").Dot("Fatalf").Call(
				jen.Lit("failed to execute request: %v"),
				jen.Err(),
			),
		),
		
		jen.Comment("Verify response"),
		jen.If(jen.Id("resp").Dot("StatusCode").Op("!=").Qual("net/http", "StatusOK")).Block(
			jen.Id("t").Dot("Errorf").Call(
				jen.Lit("expected status %d, got %d"),
				jen.Qual("net/http", "StatusOK"),
				jen.Id("resp").Dot("StatusCode"),
			),
		),
	)
	
	return file.GoString(), nil
}

func (g *Generator) generateExamples(spec *openapi.Specification, goTypes map[string]openapi.GoType) (string, error) {
	file := jen.NewFile(g.options.PackageName)
	
	// Add standard imports
	file.ImportName("context", "context")
	file.ImportName("fmt", "fmt")
	file.ImportName("log", "log")
	file.ImportName("time", "time")
	
	// Create an example function
	file.Func().Id("Example").Params().Block(
		jen.Comment("Create a new client"),
		jen.Id("client").Op(":=").Id("NewClient").Call(
			jen.Id("ClientOptions").Values(
				jen.Dict{
					jen.Id("BaseURL"):   jen.Lit("https://api.example.com"),
					jen.Id("Timeout"):   jen.Lit(10).Op("*").Qual("time", "Second"),
					jen.Id("UserAgent"): jen.Lit("example-client"),
					jen.Id("Debug"):     jen.True(),
				},
			),
		),
		
		jen.Comment("Create context with timeout"),
		jen.Id("ctx").Op(",").Id("cancel").Op(":=").Qual("context", "WithTimeout").Call(
			jen.Qual("context", "Background").Call(),
			jen.Lit(30).Op("*").Qual("time", "Second"),
		),
		jen.Defer().Id("cancel").Call(),
		
		jen.Comment("Enable logging"),
		jen.Id("EnableLogging").Call(
			jen.Id("client"),
			jen.Nil(),
			jen.True(),
			jen.False(),
		),
		
		jen.Comment("Enable retries"),
		jen.Id("EnableRetries").Call(
			jen.Id("client"),
			jen.Id("DefaultRetryConfig").Call(),
		),
		
		jen.Comment("Make API requests here"),
		jen.Comment("Example: client.List(ctx, ...)"),
		
		jen.Comment("Output:"),
		jen.Comment(""),
	)
	
	// Add examples for each operation
	operations := spec.GetOperations()
	for _, op := range operations {
		// Use the first operation with a tag as an example
		if len(op.Tags) > 0 {
			tag := op.Tags[0]
			funcName := toCamelCase(op.ID)
			
			// Create example function
			file.Func().Id("Example_" + funcName).Params().Block(
				jen.Comment("Create a new client"),
				jen.Id("client").Op(":=").Id("NewClient").Call(
					jen.Id("ClientOptions").Values(
						jen.Dict{
							jen.Id("BaseURL"):   jen.Lit("https://api.example.com"),
							jen.Id("Timeout"):   jen.Lit(10).Op("*").Qual("time", "Second"),
							jen.Id("UserAgent"): jen.Lit("example-client"),
						},
					),
				),
				
				jen.Comment("Create context"),
				jen.Id("ctx").Op(":=").Qual("context", "Background").Call(),
				
				jen.Comment(fmt.Sprintf("Call %s operation", op.ID)),
				jen.List(jen.Id("resp"), jen.Id("err")).Op(":=").Id("client").Dot(funcName).Call(
					jen.Id("ctx"),
					// Add example parameters here
				),
				
				jen.Comment("Handle response"),
				jen.If(jen.Err().Op("!=").Nil()).Block(
					jen.Qual("log", "Fatalf").Call(
						jen.Lit(fmt.Sprintf("failed to call %s: %%v", op.ID)),
						jen.Err(),
					),
				),
				
				jen.Qual("fmt", "Printf").Call(
					jen.Lit(fmt.Sprintf("Response from %s: %%+v\n", op.ID)),
					jen.Id("resp"),
				),
				
				jen.Comment("Output:"),
				jen.Comment(""),
			)
			
			// We only need one example
			break
		}
	}
	
	return file.GoString(), nil
}

func (g *Generator) generateReadme(spec *openapi.Specification) (string, error) {
	apiTitle := spec.Info.Title
	if apiTitle == "" {
		apiTitle = "API"
	}
	
	apiVersion := spec.Info.Version
	if apiVersion == "" {
		apiVersion = "1.0.0"
	}
	
	apiDescription := spec.Info.Description
	if apiDescription == "" {
		apiDescription = fmt.Sprintf("Go client for the %s", apiTitle)
	}
	
	// Group operations by tag for the README
	operations := spec.GetOperations()
	tagGroups := groupOperationsByTag(operations)
	
	// Build the README
	var sb strings.Builder
	
	// Title and description
	sb.WriteString(fmt.Sprintf("# %s Go Client\n\n", apiTitle))
	sb.WriteString(fmt.Sprintf("A Go client library for the %s (v%s).\n\n", apiTitle, apiVersion))
	sb.WriteString(apiDescription + "\n\n")
	
	// Installation
	sb.WriteString("## Installation\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("go get github.com/your-org/%s\n", g.options.PackageName))
	sb.WriteString("```\n\n")
	
	// Usage
	sb.WriteString("## Usage\n\n")
	sb.WriteString("```go\n")
	sb.WriteString("package main\n\n")
	sb.WriteString("import (\n")
	sb.WriteString("    \"context\"\n")
	sb.WriteString("    \"fmt\"\n")
	sb.WriteString("    \"time\"\n\n")
	sb.WriteString(fmt.Sprintf("    \"%s\"\n", g.options.PackageName))
	sb.WriteString(")\n\n")
	sb.WriteString("func main() {\n")
	sb.WriteString("    // Create a new client\n")
	sb.WriteString(fmt.Sprintf("    client := %s.NewClient(%s.ClientOptions{\n", g.options.PackageName, g.options.PackageName))
	sb.WriteString("        BaseURL:   \"https://api.example.com\",\n")
	sb.WriteString("        Timeout:   10 * time.Second,\n")
	sb.WriteString(fmt.Sprintf("        UserAgent: \"my-app/%s-client\",\n", g.options.PackageName))
	sb.WriteString("    })\n\n")
	sb.WriteString("    // Create context with timeout\n")
	sb.WriteString("    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)\n")
	sb.WriteString("    defer cancel()\n\n")
	
	// Example for a specific operation if available
	if len(operations) > 0 {
		op := operations[0]
		funcName := toCamelCase(op.ID)
		sb.WriteString(fmt.Sprintf("    // Call %s\n", op.ID))
		sb.WriteString(fmt.Sprintf("    resp, err := client.%s(ctx)\n", funcName))
		sb.WriteString("    if err != nil {\n")
		sb.WriteString("        fmt.Printf(\"Error: %v\\n\", err)\n")
		sb.WriteString("        return\n")
		sb.WriteString("    }\n\n")
		sb.WriteString("    fmt.Printf(\"Response: %+v\\n\", resp)\n")
	}
	
	sb.WriteString("}\n")
	sb.WriteString("```\n\n")
	
	// Features
	sb.WriteString("## Features\n\n")
	sb.WriteString("- Complete API coverage for " + apiTitle + "\n")
	sb.WriteString("- Connection pooling and request timeouts\n")
	sb.WriteString("- Automatic request retries with backoff\n")
	sb.WriteString("- Rate limiting support\n")
	sb.WriteString("- Detailed error handling\n")
	sb.WriteString("- API logging and debugging\n")
	sb.WriteString("- Authentication support\n\n")
	
	// API Operations
	sb.WriteString("## API Operations\n\n")
	
	// List operations by tag
	for _, group := range tagGroups {
		sb.WriteString(fmt.Sprintf("### %s\n\n", strings.Title(group.tag)))
		
		for _, op := range group.operations {
			funcName := toCamelCase(op.ID)
			sb.WriteString(fmt.Sprintf("- `%s`: %s\n", funcName, op.Summary))
		}
		
		sb.WriteString("\n")
	}
	
	// Authentication
	sb.WriteString("## Authentication\n\n")
	schemes := spec.GetAuthSchemes()
	if len(schemes) > 0 {
		for name, scheme := range schemes {
			sb.WriteString(fmt.Sprintf("### %s Authentication\n\n", strings.Title(name)))
			sb.WriteString(scheme.Description + "\n\n")
			
			// Example based on auth type
			sb.WriteString("```go\n")
			switch scheme.Type {
			case "apiKey":
				sb.WriteString(fmt.Sprintf("auth := %s.APIKeyAuth{\n", g.options.PackageName))
				sb.WriteString("    Key:    \"your-api-key\",\n")
				sb.WriteString(fmt.Sprintf("    Header: \"%s\",\n", scheme.Name))
				sb.WriteString("}\n")
			case "http":
				if scheme.Scheme == "basic" {
					sb.WriteString(fmt.Sprintf("auth := %s.BasicAuth{\n", g.options.PackageName))
					sb.WriteString("    Username: \"your-username\",\n")
					sb.WriteString("    Password: \"your-password\",\n")
					sb.WriteString("}\n")
				} else if scheme.Scheme == "bearer" {
					sb.WriteString(fmt.Sprintf("auth := %s.BearerAuth{\n", g.options.PackageName))
					sb.WriteString("    Token: \"your-token\",\n")
					sb.WriteString("}\n")
				}
			}
			sb.WriteString("```\n\n")
		}
	} else {
		sb.WriteString("This API does not require authentication.\n\n")
	}
	
	// Error Handling
	sb.WriteString("## Error Handling\n\n")
	sb.WriteString("```go\n")
	sb.WriteString("resp, err := client.SomeOperation(ctx)\n")
	sb.WriteString("if err != nil {\n")
	sb.WriteString(fmt.Sprintf("    if apiErr, ok := err.(*%s.APIError); ok {\n", g.options.PackageName))
	sb.WriteString("        // Handle API-specific error\n")
	sb.WriteString("        fmt.Printf(\"API Error: %s (Code: %d)\\n\", apiErr.Message, apiErr.Code)\n")
	sb.WriteString("    } else {\n")
	sb.WriteString("        // Handle other errors (network, timeout, etc.)\n")
	sb.WriteString("        fmt.Printf(\"Error: %v\\n\", err)\n")
	sb.WriteString("    }\n")
	sb.WriteString("}\n")
	sb.WriteString("```\n\n")
	
	// License
	sb.WriteString("## License\n\n")
	sb.WriteString("MIT\n")
	
	return sb.String(), nil
}

// Helper functions and types

// OperationGroup groups operations by tag
type operationGroup struct {
	tag        string
	operations []openapi.Operation
}

// groupOperationsByTag groups operations by their tags
func groupOperationsByTag(operations []openapi.Operation) []operationGroup {
	groups := make(map[string][]openapi.Operation)
	
	// Group operations by tag
	for _, op := range operations {
		if len(op.Tags) == 0 {
			// Default group for operations without tags
			groups["default"] = append(groups["default"], op)
			continue
		}
		
		// Add operation to each of its tags
		for _, tag := range op.Tags {
			groups[tag] = append(groups[tag], op)
		}
	}
	
	// Convert map to slice
	var result []operationGroup
	for tag, ops := range groups {
		result = append(result, operationGroup{
			tag:        tag,
			operations: ops,
		})
	}
	
	// Sort by tag name
	sort.Slice(result, func(i, j int) bool {
		return result[i].tag < result[j].tag
	})
	
	return result
}

// toCamelCase converts a string to camelCase
func toCamelCase(s string) string {
	if s == "" {
		return ""
	}
	
	// Split by non-alphanumeric characters
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'))
	})
	
	// Capitalize each part except the first one
	for i, part := range parts {
		if i == 0 {
			parts[i] = strings.ToLower(part)
		} else if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}
	
	return strings.Join(parts, "")
}