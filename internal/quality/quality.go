package quality

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/kumerian/mimir/internal/domain"
	"github.com/kumerian/mimir/internal/metadata"
)

const RuleVersion = "v1"

type Rule struct {
	ID        string   `json:"id"`
	Version   string   `json:"version"`
	Operation string   `json:"operation"`
	Field     string   `json:"field"`
	Expected  string   `json:"expected,omitempty"`
	Number    float64  `json:"number,omitempty"`
	Values    []string `json:"values,omitempty"`
}

type Failure struct {
	RuleID      string `json:"ruleId"`
	RuleVersion string `json:"ruleVersion"`
	Field       string `json:"field"`
	Reason      string `json:"reason"`
}

type Result struct {
	InputDigest string    `json:"inputDigest"`
	Passed      bool      `json:"passed"`
	Disposition string    `json:"disposition"`
	Failures    []Failure `json:"failures"`
}

func Evaluate(record map[string]any, rules []Rule, failureDisposition string) (Result, error) {
	dispositions := map[string]bool{"warning": true, "reject": true, "quarantine": true, "hold": true, "cleaning_required": true}
	if !dispositions[failureDisposition] {
		return Result{}, domain.NewFailure(domain.FailureInvalidInput, "quality.policy", "an explicit supported failure disposition is required")
	}
	canonical, err := metadata.CanonicalJSON(record)
	if err != nil {
		return Result{}, domain.NewFailure(domain.FailureInvalidInput, "quality.input", "input record cannot be canonicalized")
	}
	result := Result{InputDigest: digest(canonical), Passed: true, Disposition: "pass", Failures: []Failure{}}
	seen := map[string]bool{}
	for _, rule := range rules {
		if rule.ID == "" || rule.Version == "" || rule.Field == "" || seen[rule.ID] {
			return Result{}, domain.NewFailure(domain.FailureInvalidInput, "quality.rule", "rule identity, version, field, and unique identity are required")
		}
		if rule.Version != RuleVersion {
			return Result{}, domain.NewFailure(domain.FailureUnsupportedCapability, "quality.rule", "unsupported quality rule version")
		}
		seen[rule.ID] = true
		value, exists := record[rule.Field]
		reason := ""
		switch rule.Operation {
		case "required":
			if !exists || value == nil || (text(value) == "" && isString(value)) {
				reason = "required value is missing"
			}
		case "type":
			if !exists || !matchesType(value, rule.Expected) {
				reason = "value does not match required type"
			}
		case "min", "max":
			number, ok := numeric(value)
			if !exists || !ok {
				reason = "value is not numeric"
			} else if (rule.Operation == "min" && number < rule.Number) || (rule.Operation == "max" && number > rule.Number) {
				reason = "numeric value is outside configured bound"
			}
		case "enum":
			if !exists || !contains(rule.Values, fmt.Sprint(value)) {
				reason = "value is not in the allowed set"
			}
		default:
			return Result{}, domain.NewFailure(domain.FailureUnsupportedCapability, "quality.rule", "unsupported quality operation")
		}
		if reason != "" {
			result.Passed = false
			result.Failures = append(result.Failures, Failure{RuleID: rule.ID, RuleVersion: rule.Version, Field: rule.Field, Reason: reason})
		}
	}
	if !result.Passed {
		result.Disposition = failureDisposition
	}
	return result, nil
}

func matchesType(value any, expected string) bool {
	switch expected {
	case "string":
		_, ok := value.(string)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "number":
		_, ok := numeric(value)
		return ok
	case "integer":
		number, ok := numeric(value)
		return ok && math.Trunc(number) == number
	default:
		return false
	}
}

func numeric(value any) (float64, bool) {
	switch number := value.(type) {
	case int:
		return float64(number), true
	case int64:
		return float64(number), true
	case float64:
		return number, true
	case fmt.Stringer:
		parsed, err := strconv.ParseFloat(number.String(), 64)
		return parsed, err == nil
	case string:
		parsed, err := strconv.ParseFloat(number, 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func text(value any) string {
	if stringValue, ok := value.(string); ok {
		return strings.TrimSpace(stringValue)
	}
	return fmt.Sprint(value)
}

func isString(value any) bool {
	_, ok := value.(string)
	return ok
}

func contains(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

func digest(data []byte) string {
	return metadata.Digest(data)
}
