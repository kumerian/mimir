package local

import (
	"github.com/kumerian/mimir/internal/domain"
	"github.com/kumerian/mimir/internal/metadata"
)

// localMetadataPolicy enforces the local profile's classification ceiling at
// every effective object and column scope, including inherited labels.
func localMetadataPolicy(document metadata.Document) error {
	for _, source := range document.Sources {
		if source.Classification != "public" && source.Classification != "internal" {
			return domain.NewFailure(domain.FailureAccessDenied, "metadata.policy", "local profile permits only public or internal classifications")
		}
		for _, object := range source.Objects {
			policy, err := document.ResolvePolicy(source.ID, object.ID)
			if err != nil {
				return err
			}
			if policy.Classification != "public" && policy.Classification != "internal" {
				return domain.NewFailure(domain.FailureAccessDenied, "metadata.policy", "local profile permits only public or internal classifications")
			}
			for _, classification := range policy.ColumnPolicies {
				if classification != "public" && classification != "internal" {
					return domain.NewFailure(domain.FailureAccessDenied, "metadata.policy", "local profile permits only public or internal classifications")
				}
			}
		}
	}
	return nil
}
