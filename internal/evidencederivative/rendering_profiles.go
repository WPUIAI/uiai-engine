package evidencederivative

import "slices"

const PortableDataProfileRef = "rendering:portable-data-v1"
const PortableDataProfileSHA256 = "d1a3b248bbcc3a85a5d44c5b05f4cb9ec32fb979f166204af214140c2d916023"
const ArchiveStoreProfileRef = "rendering:archive-store-v1"
const ArchiveStoreProfileSHA256 = "4301bc5db3b3bacdfb649dcdf91ffe5add4694fd0a9d7201ae7e3c2c9998afdd"

func PortableDataRenderingProfile() RenderingProfile {
	return RenderingProfile{
		ProfileRef:      PortableDataProfileRef,
		ProfileSHA256:   PortableDataProfileSHA256,
		ColorProfileRef: "not_applicable",
	}
}

func ArchiveStoreRenderingProfile() RenderingProfile {
	return RenderingProfile{
		ProfileRef:      ArchiveStoreProfileRef,
		ProfileSHA256:   ArchiveStoreProfileSHA256,
		ColorProfileRef: "not_applicable",
	}
}

func renderingProfileMatches(actual, expected RenderingProfile) bool {
	return actual.ProfileRef == expected.ProfileRef &&
		actual.ProfileSHA256 == expected.ProfileSHA256 &&
		actual.ColorProfileRef == expected.ColorProfileRef &&
		slices.Equal(actual.FontRefs, expected.FontRefs) &&
		slices.Equal(actual.DependencyRefs, expected.DependencyRefs)
}
