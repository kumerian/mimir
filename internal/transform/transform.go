package transform

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/kumerian/mimir/internal/domain"
	"github.com/kumerian/mimir/internal/metadata"
)

const RuleVersion = "v1"

type Rule struct {
	ID          string `json:"id"`
	Version     string `json:"version"`
	Operation   string `json:"operation"`
	Field       string `json:"field"`
	TargetField string `json:"targetField,omitempty"`
	Value       any    `json:"value,omitempty"`
}

type OperationEvidence struct {
	RuleID      string `json:"ruleId"`
	RuleVersion string `json:"ruleVersion"`
	Operation   string `json:"operation"`
	Field       string `json:"field"`
}

type Result struct {
	Record       map[string]any      `json:"record"`
	InputDigest  string              `json:"inputDigest"`
	OutputDigest string              `json:"outputDigest"`
	Provenance   []OperationEvidence `json:"provenance"`
}

func Apply(input map[string]any, rules []Rule) (Result, error) {
	copyRecord, err := cloneRecord(input)
	if err != nil {
		return Result{}, err
	}
	inputBytes, err := metadata.CanonicalJSON(input)
	if err != nil {
		return Result{}, domain.NewFailure(domain.FailureInvalidInput, "transform.input", "input record cannot be canonicalized")
	}
	seen := map[string]bool{}
	provenance := make([]OperationEvidence, 0, len(rules))
	for _, rule := range rules {
		if rule.ID == "" || rule.Version == "" || rule.Field == "" || seen[rule.ID] {
			return Result{}, domain.NewFailure(domain.FailureInvalidInput, "transform.rule", "rule identity, version, field, and unique ordering are required")
		}
		if rule.Version != RuleVersion {
			return Result{}, domain.NewFailure(domain.FailureUnsupportedCapability, "transform.rule", "unsupported transformation rule version")
		}
		seen[rule.ID] = true
		value, exists := copyRecord[rule.Field]
		switch rule.Operation {
		case "trim", "lowercase", "uppercase":
			if !exists {
				return Result{}, domain.NewFailure(domain.FailureInvalidInput, "transform.rule", "string operation references a missing field")
			}
			text, ok := value.(string)
			if !ok {
				return Result{}, domain.NewFailure(domain.FailureInvalidInput, "transform.rule", "string operation received a non-string value")
			}
			switch rule.Operation {
			case "trim":
				copyRecord[rule.Field] = strings.TrimFunc(text, unicode.IsSpace)
			case "lowercase":
				copyRecord[rule.Field] = strings.ToLower(text)
			case "uppercase":
				copyRecord[rule.Field] = strings.ToUpper(text)
			}
		case "rename":
			if !exists || rule.TargetField == "" {
				return Result{}, domain.NewFailure(domain.FailureInvalidInput, "transform.rule", "rename requires an existing source field and target field")
			}
			if _, targetExists := copyRecord[rule.TargetField]; targetExists {
				return Result{}, domain.NewFailure(domain.FailureConflict, "transform.rule", "rename target already exists")
			}
			copyRecord[rule.TargetField] = value
			delete(copyRecord, rule.Field)
		case "to_integer":
			if !exists {
				return Result{}, domain.NewFailure(domain.FailureInvalidInput, "transform.rule", "integer conversion references a missing field")
			}
			text := fmt.Sprint(value)
			number, parseErr := strconv.ParseInt(text, 10, 64)
			if parseErr != nil {
				return Result{}, domain.NewFailure(domain.FailureInvalidInput, "transform.rule", "value cannot be converted to a signed 64-bit integer")
			}
			copyRecord[rule.Field] = number
		case "replace_null":
			if !exists {
				return Result{}, domain.NewFailure(domain.FailureInvalidInput, "transform.rule", "null replacement references a missing field")
			}
			if value == nil {
				if rule.Value == nil {
					return Result{}, domain.NewFailure(domain.FailureInvalidInput, "transform.rule", "null replacement requires an explicit value")
				}
				copyRecord[rule.Field] = rule.Value
			}
		default:
			return Result{}, domain.NewFailure(domain.FailureUnsupportedCapability, "transform.rule", "unsupported transformation operation")
		}
		provenance = append(provenance, OperationEvidence{RuleID: rule.ID, RuleVersion: rule.Version, Operation: rule.Operation, Field: rule.Field})
	}
	outputBytes, err := metadata.CanonicalJSON(copyRecord)
	if err != nil {
		return Result{}, domain.NewFailure(domain.FailureInvalidInput, "transform.output", "transformed record cannot be canonicalized")
	}
	return Result{
		Record: copyRecord, InputDigest: digest(inputBytes), OutputDigest: digest(outputBytes), Provenance: provenance,
	}, nil
}

func cloneRecord(input map[string]any) (map[string]any, error) {
	data, err := json.Marshal(input)
	if err != nil {
		return nil, domain.NewFailure(domain.FailureInvalidInput, "transform.input", "input record cannot be copied")
	}
	var result map[string]any
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.UseNumber()
	if err := decoder.Decode(&result); err != nil || result == nil {
		return nil, domain.NewFailure(domain.FailureInvalidInput, "transform.input", "input must be a JSON object")
	}
	return result, nil
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
