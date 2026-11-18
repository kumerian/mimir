package metadata

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/kumerian/mimir/internal/domain"
)

const (
	ContractVersion = "mimir.metadata/v1"
	maxDocumentSize = 1 << 20
)

var (
	envSecretReferencePattern = regexp.MustCompile(`^env://[A-Za-z_][A-Za-z0-9_]{0,127}$`)
	unknownFieldPattern       = regexp.MustCompile(`^json: unknown field "([A-Za-z_][A-Za-z0-9_.-]{0,63})"$`)
)

type Document struct {
	SchemaVersion string   `json:"schemaVersion"`
	Revision      string   `json:"revision"`
	Status        string   `json:"status"`
	Sources       []Source `json:"sources"`
}

type Source struct {
	ID             string            `json:"id"`
	Classification string            `json:"classification"`
	Objects        []Object          `json:"objects"`
	SecretRefs     []SecretReference `json:"secretReferences,omitempty"`
}

type SecretReference struct {
	ID        string `json:"id"`
	Reference string `json:"reference"`
}

type Object struct {
	ID                        string          `json:"id"`
	Path                      string          `json:"path"`
	Classification            string          `json:"classification,omitempty"`
	Columns                   []Column        `json:"columns,omitempty"`
	Transformations           []TransformRule `json:"transformations,omitempty"`
	QualityRules              []QualityRule   `json:"qualityRules,omitempty"`
	QualityFailureDisposition string          `json:"qualityFailureDisposition,omitempty"`
	Schedule                  *Schedule       `json:"schedule,omitempty"`
}

type Column struct {
	ID             string `json:"id"`
	Type           string `json:"type"`
	Classification string `json:"classification,omitempty"`
}

type TransformRule struct {
	ID          string          `json:"id"`
	Version     string          `json:"version"`
	Operation   string          `json:"operation"`
	Field       string          `json:"field"`
	TargetField string          `json:"targetField,omitempty"`
	Value       json.RawMessage `json:"value,omitempty"`
}

type QualityRule struct {
	ID        string   `json:"id"`
	Version   string   `json:"version"`
	Operation string   `json:"operation"`
	Field     string   `json:"field"`
	Expected  string   `json:"expected,omitempty"`
	Number    float64  `json:"number,omitempty"`
	Values    []string `json:"values,omitempty"`
}

type Schedule struct {
	Enabled          bool     `json:"enabled"`
	IntervalSeconds  int64    `json:"intervalSeconds"`
	StartsAt         string   `json:"startsAt,omitempty"`
	EndsAt           string   `json:"endsAt,omitempty"`
	Dependencies     []string `json:"dependencies,omitempty"`
	RetryLimit       int      `json:"retryLimit,omitempty"`
	ConcurrencyLimit int      `json:"concurrencyLimit,omitempty"`
}

type Parsed struct {
	Document  Document
	Canonical []byte
}

type EffectivePolicy struct {
	SourceID         string            `json:"sourceId"`
	ObjectID         string            `json:"objectId"`
	Classification   string            `json:"classification"`
	ClassificationAt string            `json:"classificationOrigin"`
	ColumnPolicies   map[string]string `json:"columnClassifications"`
	ColumnOrigins    map[string]string `json:"columnClassificationOrigins"`
}

func (d Document) ResolvePolicy(sourceID, objectID string) (EffectivePolicy, error) {
	for _, source := range d.Sources {
		if source.ID != sourceID {
			continue
		}
		for _, object := range source.Objects {
			if object.ID != objectID {
				continue
			}
			effective := EffectivePolicy{
				SourceID: source.ID, ObjectID: object.ID,
				Classification: source.Classification, ClassificationAt: "source:" + source.ID,
				ColumnPolicies: map[string]string{}, ColumnOrigins: map[string]string{},
			}
			if object.Classification != "" && classificationLevel(object.Classification) > classificationLevel(effective.Classification) {
				effective.Classification = object.Classification
				effective.ClassificationAt = "object:" + object.ID
			}
			for _, column := range object.Columns {
				classification := effective.Classification
				origin := effective.ClassificationAt
				if column.Classification != "" && classificationLevel(column.Classification) > classificationLevel(classification) {
					classification = column.Classification
					origin = "column:" + column.ID
				}
				effective.ColumnPolicies[column.ID] = classification
				effective.ColumnOrigins[column.ID] = origin
			}
			return effective, nil
		}
	}
	return EffectivePolicy{}, domain.NewFailure(domain.FailureInvalidInput, "metadata.resolve", "source or object scope was not found")
}

func classificationLevel(value string) int {
	switch value {
	case "public":
		return 0
	case "internal":
		return 1
	case "confidential":
		return 2
	case "restricted":
		return 3
	default:
		return -1
	}
}

func Parse(input io.Reader) (Parsed, error) {
	raw, err := io.ReadAll(io.LimitReader(input, maxDocumentSize+1))
	if err != nil {
		return Parsed{}, domain.NewFailure(domain.FailureInvalidInput, "metadata.parse", "unable to read metadata")
	}
	if len(raw) > maxDocumentSize {
		return Parsed{}, domain.NewFailure(domain.FailureInvalidInput, "metadata.parse", "metadata exceeds the 1 MiB limit")
	}
	if err := rejectDuplicateKeys(raw); err != nil {
		return Parsed{}, domain.NewFailure(domain.FailureInvalidInput, "metadata.parse", err.Error())
	}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var document Document
	if err := decoder.Decode(&document); err != nil {
		message := "invalid metadata JSON"
		if match := unknownFieldPattern.FindStringSubmatch(err.Error()); len(match) == 2 && safeDiagnosticFieldName(match[1]) {
			message = fmt.Sprintf("unknown metadata field %q", match[1])
		} else if strings.Contains(err.Error(), "unknown field") {
			message = "unknown metadata field (name withheld)"
		}
		return Parsed{}, domain.NewFailure(domain.FailureInvalidInput, "metadata.parse", message)
	}
	if err := ensureEOF(decoder); err != nil {
		return Parsed{}, domain.NewFailure(domain.FailureInvalidInput, "metadata.parse", "metadata must contain exactly one JSON document")
	}
	if document.SchemaVersion != ContractVersion {
		return Parsed{}, domain.NewFailure(domain.FailureInvalidInput, "metadata.version", "unsupported metadata version")
	}
	if diagnostics := ValidateSchema(raw); len(diagnostics) > 0 {
		return Parsed{}, domain.NewFailure(domain.FailureInvalidInput, "metadata.schema", diagnostics[0].Message+" at "+diagnostics[0].Path)
	}
	if err := validate(document); err != nil {
		return Parsed{}, domain.NewFailure(domain.FailureInvalidInput, "metadata.validate", err.Error())
	}
	canonical, err := json.Marshal(document)
	if err != nil {
		return Parsed{}, domain.NewFailure(domain.FailureInternal, "metadata.canonicalize", "unable to canonicalize metadata")
	}
	return Parsed{Document: document, Canonical: canonical}, nil
}

func ParseBytes(raw []byte) (Parsed, error) {
	return Parse(bytes.NewReader(raw))
}

func safeDiagnosticFieldName(name string) bool {
	lower := strings.ToLower(name)
	for _, marker := range []string{"secret", "token", "password", "credential", "canary", "bearer", "api_key", "private_key"} {
		if strings.Contains(lower, marker) {
			return false
		}
	}
	return true
}

func (d Document) RunEligible() bool {
	return d.Status == "active"
}

func (d Document) Validate() error {
	if d.SchemaVersion != ContractVersion {
		return domain.NewFailure(domain.FailureInvalidInput, "metadata.version", "unsupported metadata version")
	}
	raw, err := json.Marshal(d)
	if err != nil {
		return domain.NewFailure(domain.FailureInvalidInput, "metadata.schema", "metadata cannot be encoded for validation")
	}
	if diagnostics := ValidateSchema(raw); len(diagnostics) > 0 {
		return domain.NewFailure(domain.FailureInvalidInput, "metadata.schema", diagnostics[0].Message+" at "+diagnostics[0].Path)
	}
	if err := validate(d); err != nil {
		return domain.NewFailure(domain.FailureInvalidInput, "metadata.validate", err.Error())
	}
	return nil
}

func validate(d Document) error {
	if !validDomainIdentifier(domain.MetadataRevisionID, d.Revision) {
		return fmt.Errorf("revision must be 1-128 safe identifier characters")
	}
	if d.Status != "draft" && d.Status != "active" {
		return fmt.Errorf("status must be draft or active")
	}
	if len(d.Sources) == 0 {
		return fmt.Errorf("at least one source is required")
	}
	seenSources := map[string]bool{}
	for _, source := range d.Sources {
		if !validDomainIdentifier(domain.SourceID, source.ID) || seenSources[source.ID] {
			return fmt.Errorf("source identifiers must be non-empty and unique")
		}
		seenSources[source.ID] = true
		if !validClassification(source.Classification) {
			return fmt.Errorf("source %q has an unsupported classification", source.ID)
		}
		if len(source.Objects) == 0 {
			return fmt.Errorf("source %q must declare at least one object", source.ID)
		}
		seenSecretRefs := map[string]bool{}
		for _, reference := range source.SecretRefs {
			if !validDomainIdentifier(domain.SecretReferenceID, reference.ID) ||
				seenSecretRefs[reference.ID] || !envSecretReferencePattern.MatchString(reference.Reference) {
				return fmt.Errorf("source %q secret references require unique IDs and supported opaque env:// references", source.ID)
			}
			seenSecretRefs[reference.ID] = true
		}
		seenObjects := map[string]bool{}
		scheduleDependencies := make(map[string][]string)
		for _, object := range source.Objects {
			if !validDomainIdentifier(domain.ObjectID, object.ID) || seenObjects[object.ID] {
				return fmt.Errorf("object identifiers in source %q must be non-empty and unique", source.ID)
			}
			seenObjects[object.ID] = true
			if object.Path == "" || strings.HasPrefix(object.Path, "/") ||
				strings.Contains(object.Path, "..") || strings.ContainsAny(object.Path, `\:`) {
				return fmt.Errorf("object %q path must be a non-empty relative path without traversal", object.ID)
			}
			if object.Classification != "" && !validClassification(object.Classification) {
				return fmt.Errorf("object %q has an unsupported classification", object.ID)
			}
			seenColumns := map[string]bool{}
			for _, column := range object.Columns {
				if !validDomainIdentifier(domain.ColumnID, column.ID) || seenColumns[column.ID] {
					return fmt.Errorf("column identifiers in object %q must be non-empty and unique", object.ID)
				}
				seenColumns[column.ID] = true
				switch column.Type {
				case "string", "number", "integer", "boolean":
				default:
					return fmt.Errorf("column %q has an unsupported type", column.ID)
				}
				if column.Classification != "" && !validClassification(column.Classification) {
					return fmt.Errorf("column %q has an unsupported classification", column.ID)
				}
			}
			seenTransformations := map[string]bool{}
			for _, rule := range object.Transformations {
				if !validDomainIdentifier(domain.RuleID, rule.ID) || rule.Version == "" ||
					rule.Field == "" || seenTransformations[rule.ID] {
					return fmt.Errorf("transformation rules in object %q require unique IDs, versions, and fields", object.ID)
				}
				if rule.Version != "v1" {
					return fmt.Errorf("transformation rule %q has an unsupported version", rule.ID)
				}
				seenTransformations[rule.ID] = true
				switch rule.Operation {
				case "trim", "lowercase", "uppercase", "to_integer":
				case "rename":
					if rule.TargetField == "" {
						return fmt.Errorf("transformation %q requires a target field", rule.ID)
					}
				case "replace_null":
					if len(rule.Value) == 0 || bytes.Equal(bytes.TrimSpace(rule.Value), []byte("null")) {
						return fmt.Errorf("transformation %q requires an explicit null replacement value", rule.ID)
					}
				default:
					return fmt.Errorf("transformation %q has an unsupported operation", rule.ID)
				}
				if len(rule.Value) > 0 && !json.Valid(rule.Value) {
					return fmt.Errorf("transformation %q has an invalid JSON value", rule.ID)
				}
			}
			seenQualityRules := map[string]bool{}
			for _, rule := range object.QualityRules {
				if !validDomainIdentifier(domain.RuleID, rule.ID) || rule.Version == "" ||
					rule.Field == "" || seenQualityRules[rule.ID] {
					return fmt.Errorf("quality rules in object %q require unique IDs, versions, and fields", object.ID)
				}
				if rule.Version != "v1" {
					return fmt.Errorf("quality rule %q has an unsupported version", rule.ID)
				}
				seenQualityRules[rule.ID] = true
				switch rule.Operation {
				case "required":
				case "type":
					switch rule.Expected {
					case "string", "number", "integer", "boolean":
					default:
						return fmt.Errorf("quality rule %q has an unsupported expected type", rule.ID)
					}
				case "min", "max":
				case "enum":
					if len(rule.Values) == 0 {
						return fmt.Errorf("quality rule %q requires allowed values", rule.ID)
					}
				default:
					return fmt.Errorf("quality rule %q has an unsupported operation", rule.ID)
				}
			}
			switch object.QualityFailureDisposition {
			case "", "warning", "reject", "quarantine", "hold", "cleaning_required":
			default:
				return fmt.Errorf("object %q has an unsupported quality failure disposition", object.ID)
			}
			if len(object.QualityRules) > 0 && object.QualityFailureDisposition == "" {
				return fmt.Errorf("object %q quality rules require an explicit failure disposition", object.ID)
			}
			if object.Schedule != nil {
				if object.Schedule.IntervalSeconds < 1 || object.Schedule.IntervalSeconds > 31536000 ||
					object.Schedule.RetryLimit < 0 || object.Schedule.RetryLimit > 10 ||
					object.Schedule.ConcurrencyLimit < 0 || object.Schedule.ConcurrencyLimit > 16 {
					return fmt.Errorf("object %q schedule has an invalid interval, retry limit, or concurrency limit", object.ID)
				}
				start, startErr := parseOptionalTime(object.Schedule.StartsAt)
				end, endErr := parseOptionalTime(object.Schedule.EndsAt)
				if startErr != nil || endErr != nil || (!start.IsZero() && !end.IsZero() && !end.After(start)) {
					return fmt.Errorf("object %q schedule window is invalid", object.ID)
				}
				seenDependencies := map[string]bool{}
				for _, dependency := range object.Schedule.Dependencies {
					if dependency == "" || dependency == object.ID || seenDependencies[dependency] {
						return fmt.Errorf("object %q schedule dependencies must be unique other object IDs", object.ID)
					}
					seenDependencies[dependency] = true
					scheduleDependencies[object.ID] = append(scheduleDependencies[object.ID], dependency)
				}
			}
		}
		for objectID, dependencies := range scheduleDependencies {
			for _, dependency := range dependencies {
				if !seenObjects[dependency] {
					return fmt.Errorf("object %q depends on unknown object %q", objectID, dependency)
				}
			}
		}
		if hasScheduleCycle(scheduleDependencies) {
			return fmt.Errorf("schedule dependency graph contains a cycle")
		}
	}
	return nil
}

func hasScheduleCycle(graph map[string][]string) bool {
	visited := make(map[string]bool, len(graph))
	active := make(map[string]bool, len(graph))
	var visit func(string) bool
	visit = func(node string) bool {
		if active[node] {
			return true
		}
		if visited[node] {
			return false
		}
		visited[node] = true
		active[node] = true
		for _, dependency := range graph[node] {
			if visit(dependency) {
				return true
			}
		}
		active[node] = false
		return false
	}
	for node := range graph {
		if visit(node) {
			return true
		}
	}
	return false
}

func parseOptionalTime(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, value)
}

func validClassification(value string) bool {
	switch value {
	case "public", "internal", "confidential", "restricted":
		return true
	default:
		return false
	}
}

func validDomainIdentifier(kind domain.IdentifierKind, value string) bool {
	_, err := domain.NewIdentifier(kind, value)
	return err == nil
}

func ensureEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("extra JSON document")
		}
		return err
	}
	return nil
}

func rejectDuplicateKeys(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if _, err := consumeValue(decoder); err != nil {
		return err
	}
	if err := ensureEOF(decoder); err != nil {
		return fmt.Errorf("metadata must contain exactly one JSON document")
	}
	return nil
}

// ValidateJSONDocument checks one JSON value for syntax, trailing content, and
// duplicate keys without imposing a metadata or record shape.
func ValidateJSONDocument(raw []byte) error {
	return rejectDuplicateKeys(raw)
}

func consumeValue(decoder *json.Decoder) (any, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("invalid metadata JSON")
	}
	delimiter, isDelimiter := token.(json.Delim)
	if !isDelimiter {
		return token, nil
	}
	switch delimiter {
	case '{':
		values := map[string]any{}
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return nil, fmt.Errorf("invalid metadata JSON object")
			}
			key, ok := keyToken.(string)
			if !ok {
				return nil, fmt.Errorf("invalid metadata JSON object key")
			}
			if _, exists := values[key]; exists {
				return nil, fmt.Errorf("duplicate metadata key %q", key)
			}
			value, err := consumeValue(decoder)
			if err != nil {
				return nil, err
			}
			values[key] = value
		}
		if _, err := decoder.Token(); err != nil {
			return nil, fmt.Errorf("invalid metadata JSON object")
		}
		return values, nil
	case '[':
		values := []any{}
		for decoder.More() {
			value, err := consumeValue(decoder)
			if err != nil {
				return nil, err
			}
			values = append(values, value)
		}
		if _, err := decoder.Token(); err != nil {
			return nil, fmt.Errorf("invalid metadata JSON array")
		}
		return values, nil
	default:
		return nil, fmt.Errorf("unexpected JSON delimiter")
	}
}

// CanonicalJSON is useful for comparisons and stable identities. encoding/json
// sorts string map keys; the helper also rejects values unsupported by JSON.
func CanonicalJSON(value any) ([]byte, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var decoded any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&decoded); err != nil {
		return nil, err
	}
	return marshalCanonical(decoded)
}

func Digest(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

// CanonicalizeRecord validates one object record and returns deterministic JSON
// bytes. Duplicate keys and non-object roots are rejected before use.
func CanonicalizeRecord(raw []byte) ([]byte, error) {
	if len(raw) == 0 || len(raw) > 256<<10 {
		return nil, fmt.Errorf("record must be between 1 byte and 256 KiB")
	}
	if err := rejectDuplicateKeys(raw); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("record is not valid JSON")
	}
	if err := ensureEOF(decoder); err != nil {
		return nil, fmt.Errorf("record contains trailing JSON")
	}
	if _, ok := value.(map[string]any); !ok {
		return nil, fmt.Errorf("record must be a JSON object")
	}
	return marshalCanonical(value)
}

func marshalCanonical(value any) ([]byte, error) {
	switch typed := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		var output bytes.Buffer
		output.WriteByte('{')
		for i, key := range keys {
			if i > 0 {
				output.WriteByte(',')
			}
			keyBytes, _ := json.Marshal(key)
			output.Write(keyBytes)
			output.WriteByte(':')
			valueBytes, err := marshalCanonical(typed[key])
			if err != nil {
				return nil, err
			}
			output.Write(valueBytes)
		}
		output.WriteByte('}')
		return output.Bytes(), nil
	case []any:
		var output bytes.Buffer
		output.WriteByte('[')
		for i, item := range typed {
			if i > 0 {
				output.WriteByte(',')
			}
			itemBytes, err := marshalCanonical(item)
			if err != nil {
				return nil, err
			}
			output.Write(itemBytes)
		}
		output.WriteByte(']')
		return output.Bytes(), nil
	default:
		return json.Marshal(value)
	}
}
