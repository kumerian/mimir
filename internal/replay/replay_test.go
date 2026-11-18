package replay

import "testing"

func TestPrepareReplayPinsOriginalInputsAndRejectsChangedSource(t *testing.T) {
	original := Plan{
		RunID: "run-1", MetadataRevision: "rev-1", MetadataDigest: "meta-hash",
		SourceDigest: "source-hash", AdapterVersions: []string{"source/1", "destination/1"},
		RuleVersions: []string{"normalize/1"},
	}
	result, err := Prepare(original, original, "operator_request")
	if err != nil {
		t.Fatal(err)
	}
	if result.OriginalRunID != original.RunID || result.RunID == original.RunID || result.Reason == "" {
		t.Fatalf("replay identity is not linked: %+v", result)
	}
	changed := original
	changed.SourceDigest = "other-source"
	if _, err := Prepare(original, changed, "retry"); err == nil {
		t.Fatal("Prepare() accepted changed source snapshot")
	}
}
