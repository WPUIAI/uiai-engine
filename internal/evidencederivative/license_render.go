package evidencederivative

import "sort"

func normalizedLicenses(licenses []LicenseAttestation, assetRefs []string) ([]LicenseAttestation, error) {
	if err := validateLicenses(licenses, assetRefs); err != nil {
		return nil, err
	}
	result := append([]LicenseAttestation(nil), licenses...)
	sort.Slice(result, func(i, j int) bool {
		if result[i].AssetRef == result[j].AssetRef {
			return result[i].LicenseRef < result[j].LicenseRef
		}
		return result[i].AssetRef < result[j].AssetRef
	})
	return result, nil
}
