package local

import (
	"github.com/kumerian/mimir/internal/domain"
	"github.com/kumerian/mimir/internal/metadata"
	"github.com/kumerian/mimir/internal/ports"
)

// CheckProfile validates the built-in adapter registrations without opening
// files. It does not check filesystem readiness or authorize a run.
func CheckProfile() error {
	return validateLocalProfile((fileSourceAdapter{}).Capabilities(), (fileDestinationAdapter{}).Capabilities())
}

func validateLocalProfile(source ports.SourceCapabilities, destination ports.DestinationCapabilities) error {
	if source.Version != "local.source/v1" || destination.Version != "local.destination/v1" ||
		source.MaximumBatchRecords != maxSourceRecords || destination.MaximumBatchRecords != maxSourceRecords ||
		source.Acknowledgement || !source.SchemaObservation {
		return domain.NewFailure(domain.FailureUnsupportedCapability, "profile.startup", "local reference adapter capability does not match the approved profile")
	}
	return ports.ValidateBoundedRun(source, destination)
}

func validateLocalMetadataCapabilities(document metadata.Document) error {
	for _, source := range document.Sources {
		if len(source.SecretRefs) != 0 {
			return domain.NewFailure(domain.FailureUnsupportedCapability, "profile.credentials", "local file adapters do not consume secret references")
		}
	}
	return nil
}
