package metadata

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
)

const ValidationReportVersion = "mimir.metadata-validation/v1"

type ValidationReport struct {
	ContractVersion       string             `json:"contractVersion"`
	InputDigest           string             `json:"inputDigest"`
	Valid                 bool               `json:"valid"`
	RunEligible           bool               `json:"runEligible"`
	SecurityImpact        string             `json:"securityImpact"`
	CompatibilityDecision string             `json:"compatibilityDecision"`
	Diagnostics           []SchemaDiagnostic `json:"diagnostics"`
}

func SafeValidationReport(report ValidationReport) bool {
	if report.ContractVersion != ValidationReportVersion || report.RunEligible ||
		report.SecurityImpact != "input_not_retained" ||
		(report.CompatibilityDecision != "rejected" && report.CompatibilityDecision != "unsupported_version" &&
			report.CompatibilityDecision != "accepted_v1") ||
		(report.Valid && len(report.Diagnostics) != 0) ||
		(!report.Valid && len(report.Diagnostics) == 0) ||
		(report.Valid && report.CompatibilityDecision != "accepted_v1") ||
		(!report.Valid && report.CompatibilityDecision == "accepted_v1") {
		return false
	}
	messages := map[string]string{
		"invalid_json":       "metadata is not valid JSON",
		"schema_unavailable": "metadata schema could not be loaded",
		"schema_violation":   "metadata does not conform to the v1 JSON Schema",
		"document_too_large": "metadata exceeds the 1 MiB limit",
		"metadata_invalid":   "metadata was rejected by the v1 validator",
	}
	for _, diagnostic := range report.Diagnostics {
		message, known := messages[diagnostic.Code]
		if !known || diagnostic.Message != message || !safeReportPath(diagnostic.Path) {
			return false
		}
	}
	return true
}

func safeReportPath(path string) bool {
	if path == "/" {
		return true
	}
	if !strings.HasPrefix(path, "/") {
		return false
	}
	for _, segment := range strings.Split(strings.TrimPrefix(path, "/"), "/") {
		if segment == "*" {
			continue
		}
		if _, err := strconv.Atoi(segment); err == nil {
			continue
		}
		if !safeDiagnosticFieldName(strings.ReplaceAll(strings.ReplaceAll(segment, "~1", "/"), "~0", "~")) {
			return false
		}
	}
	return true
}

// Validate produces deterministic, content-free validation evidence. The
// digest identifies the exact input bytes but does not preserve the input.
func Validate(raw []byte) ValidationReport {
	sum := sha256.Sum256(raw)
	report := ValidationReport{
		ContractVersion:       ValidationReportVersion,
		InputDigest:           hex.EncodeToString(sum[:]),
		RunEligible:           false,
		SecurityImpact:        "input_not_retained",
		CompatibilityDecision: "rejected",
		Diagnostics:           []SchemaDiagnostic{},
	}
	if len(raw) > maxDocumentSize {
		report.Diagnostics = []SchemaDiagnostic{{
			Code: "document_too_large", Path: "/", Message: "metadata exceeds the 1 MiB limit",
		}}
		return report
	}
	if diagnostics := ValidateSchema(raw); len(diagnostics) > 0 {
		report.Diagnostics = diagnostics
	}
	var envelope struct {
		SchemaVersion string `json:"schemaVersion"`
	}
	if json.Unmarshal(raw, &envelope) == nil && envelope.SchemaVersion != "" && envelope.SchemaVersion != ContractVersion {
		report.CompatibilityDecision = "unsupported_version"
	}
	if _, err := ParseBytes(raw); err != nil {
		if len(report.Diagnostics) == 0 {
			report.Diagnostics = []SchemaDiagnostic{{
				Code: "metadata_invalid", Path: "/", Message: "metadata was rejected by the v1 validator",
			}}
		}
		return report
	}
	report.Valid = true
	report.CompatibilityDecision = "accepted_v1"
	return report
}
