package metadata

import (
	"bytes"
	_ "embed"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

const schemaLocation = "https://mimir.example/contracts/metadata/v1.schema.json"

//go:embed v1.schema.json
var embeddedV1Schema []byte

type SchemaDiagnostic struct {
	Code    string `json:"code"`
	Path    string `json:"path"`
	Message string `json:"message"`
}

var (
	compileSchemaOnce sync.Once
	compiledV1Schema  *jsonschema.Schema
	compileSchemaErr  error
)

func ValidateSchema(raw []byte) []SchemaDiagnostic {
	value, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		return []SchemaDiagnostic{{Code: "invalid_json", Path: "/", Message: "metadata is not valid JSON"}}
	}
	schema, err := loadV1Schema()
	if err != nil {
		return []SchemaDiagnostic{{Code: "schema_unavailable", Path: "/", Message: "metadata schema could not be loaded"}}
	}
	if err := schema.Validate(value); err != nil {
		var validation *jsonschema.ValidationError
		if !asSchemaValidationError(err, &validation) {
			return []SchemaDiagnostic{{Code: "schema_unavailable", Path: "/", Message: "metadata schema validation failed"}}
		}
		diagnostics := make([]SchemaDiagnostic, 0)
		collectSchemaDiagnostics(validation, &diagnostics)
		sort.Slice(diagnostics, func(i, j int) bool {
			if diagnostics[i].Path == diagnostics[j].Path {
				return diagnostics[i].Code < diagnostics[j].Code
			}
			return diagnostics[i].Path < diagnostics[j].Path
		})
		return deduplicateSchemaDiagnostics(diagnostics)
	}
	return []SchemaDiagnostic{}
}

func loadV1Schema() (*jsonschema.Schema, error) {
	compileSchemaOnce.Do(func() {
		document, err := jsonschema.UnmarshalJSON(bytes.NewReader(embeddedV1Schema))
		if err != nil {
			compileSchemaErr = fmt.Errorf("decode embedded schema: %w", err)
			return
		}
		compiler := jsonschema.NewCompiler()
		compiler.AssertFormat()
		if err := compiler.AddResource(schemaLocation, document); err != nil {
			compileSchemaErr = fmt.Errorf("register embedded schema: %w", err)
			return
		}
		compiledV1Schema, compileSchemaErr = compiler.Compile(schemaLocation)
	})
	return compiledV1Schema, compileSchemaErr
}

func asSchemaValidationError(err error, target **jsonschema.ValidationError) bool {
	validation, ok := err.(*jsonschema.ValidationError)
	if !ok {
		return false
	}
	*target = validation
	return true
}

func collectSchemaDiagnostics(validation *jsonschema.ValidationError, into *[]SchemaDiagnostic) {
	if len(validation.Causes) > 0 {
		for _, cause := range validation.Causes {
			collectSchemaDiagnostics(cause, into)
		}
		return
	}
	*into = append(*into, SchemaDiagnostic{
		Code: "schema_violation", Path: safeInstancePath(validation.InstanceLocation),
		Message: "metadata does not conform to the v1 JSON Schema",
	})
}

func safeInstancePath(segments []string) string {
	var path strings.Builder
	path.WriteByte('/')
	wroteSegment := false
	for _, segment := range segments {
		if !wroteSegment {
			wroteSegment = true
		} else {
			path.WriteByte('/')
		}
		if _, err := strconv.Atoi(segment); err == nil {
			path.WriteString(segment)
		} else if safeDiagnosticFieldName(segment) {
			path.WriteString(strings.ReplaceAll(strings.ReplaceAll(segment, "~", "~0"), "/", "~1"))
		} else {
			path.WriteByte('*')
		}
	}
	if !wroteSegment {
		return "/"
	}
	return path.String()
}

func deduplicateSchemaDiagnostics(diagnostics []SchemaDiagnostic) []SchemaDiagnostic {
	if len(diagnostics) < 2 {
		return diagnostics
	}
	out := diagnostics[:1]
	for _, diagnostic := range diagnostics[1:] {
		last := out[len(out)-1]
		if diagnostic.Code != last.Code || diagnostic.Path != last.Path {
			out = append(out, diagnostic)
		}
	}
	return out
}
