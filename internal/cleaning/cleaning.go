package cleaning

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/kumerian/mimir/internal/domain"
	"github.com/kumerian/mimir/internal/metadata"
)

const CorrectionVersion = "v1"

type Correction struct {
	ID        string `json:"id"`
	Version   string `json:"version"`
	Operation string `json:"operation"`
	Field     string `json:"field"`
	Value     any    `json:"value,omitempty"`
}

type Audit struct {
	CorrectionID      string `json:"correctionId"`
	CorrectionVersion string `json:"correctionVersion"`
	Actor             string `json:"actor"`
	InputReference    string `json:"inputReference"`
	OutputDigest      string `json:"outputDigest"`
	IdempotencyKey    string `json:"idempotencyKey"`
	OccurredAt        string `json:"occurredAt"`
}

type Result struct {
	Record map[string]any `json:"record"`
	Audit  Audit          `json:"audit"`
}

func Apply(input map[string]any, correction Correction, actor string, authorized bool) (Result, error) {
	if !authorized || actor == "" {
		return Result{}, domain.NewFailure(domain.FailureAccessDenied, "cleaning.apply", "explicit cleaning authorization and actor are required")
	}
	if correction.ID == "" || correction.Version == "" || correction.Field == "" {
		return Result{}, domain.NewFailure(domain.FailureInvalidInput, "cleaning.rule", "correction identity, version, and field are required")
	}
	if correction.Version != CorrectionVersion {
		return Result{}, domain.NewFailure(domain.FailureUnsupportedCapability, "cleaning.rule", "unsupported cleaning correction version")
	}
	canonical, err := metadata.CanonicalJSON(input)
	if err != nil {
		return Result{}, domain.NewFailure(domain.FailureInvalidInput, "cleaning.input", "input record cannot be canonicalized")
	}
	if _, exists := input[correction.Field]; !exists {
		return Result{}, domain.NewFailure(domain.FailureInvalidInput, "cleaning.rule", "correction field does not exist")
	}
	raw, err := json.Marshal(input)
	if err != nil {
		return Result{}, domain.NewFailure(domain.FailureInvalidInput, "cleaning.input", "input record cannot be copied")
	}
	var output map[string]any
	if err := json.Unmarshal(raw, &output); err != nil {
		return Result{}, domain.NewFailure(domain.FailureInvalidInput, "cleaning.input", "input record cannot be copied")
	}
	value := output[correction.Field]
	switch correction.Operation {
	case "trim", "uppercase", "lowercase":
		text, ok := value.(string)
		if !ok {
			return Result{}, domain.NewFailure(domain.FailureInvalidInput, "cleaning.rule", "string correction received a non-string value")
		}
		switch correction.Operation {
		case "trim":
			output[correction.Field] = strings.TrimFunc(text, unicode.IsSpace)
		case "uppercase":
			output[correction.Field] = strings.ToUpper(text)
		case "lowercase":
			output[correction.Field] = strings.ToLower(text)
		}
	case "replace_null":
		if value != nil {
			return Result{}, domain.NewFailure(domain.FailureInvalidInput, "cleaning.rule", "replace_null only applies to a null value")
		}
		if correction.Value == nil {
			return Result{}, domain.NewFailure(domain.FailureInvalidInput, "cleaning.rule", "replace_null requires an explicit replacement value")
		}
		output[correction.Field] = correction.Value
	default:
		return Result{}, domain.NewFailure(domain.FailureUnsupportedCapability, "cleaning.rule", "unsupported correction operation")
	}
	outputBytes, err := metadata.CanonicalJSON(output)
	if err != nil {
		return Result{}, domain.NewFailure(domain.FailureInvalidInput, "cleaning.output", "cleaned record cannot be canonicalized")
	}
	inputDigest := metadata.Digest(canonical)
	outputDigest := metadata.Digest(outputBytes)
	idempotency := metadata.Digest([]byte(strings.Join([]string{inputDigest, correction.ID, correction.Version, actor}, "\x00")))
	return Result{
		Record: output,
		Audit: Audit{
			CorrectionID: correction.ID, CorrectionVersion: correction.Version, Actor: actor,
			InputReference: inputDigest, OutputDigest: outputDigest, IdempotencyKey: idempotency,
			OccurredAt: time.Now().UTC().Format(time.RFC3339Nano),
		},
	}, nil
}

func (a Audit) String() string {
	return fmt.Sprintf("cleaning:%s/%s:%s", a.CorrectionID, a.CorrectionVersion, a.IdempotencyKey)
}
