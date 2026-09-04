package evidenceartifact

const maxCaptureAnnotations = 2048

func validateCaptureAnnotations(manifest Manifest, annotations []CaptureAnnotation, observations []CaptureObservation) error {
	if len(annotations) > maxCaptureAnnotations {
		return ErrAnnotationInvalid
	}
	observationAssets := make(map[string]string, len(observations))
	for _, observation := range observations {
		observationAssets[observation.ObservationRef] = observation.AssetID
	}
	assets := make(map[string]Asset, len(manifest.Assets))
	for _, asset := range manifest.Assets {
		assets[asset.AssetID] = asset
	}
	seen := make(map[string]struct{}, len(annotations))
	for _, annotation := range annotations {
		if !validRef(annotation.AnnotationRef, true) || !validRef(annotation.ObservationRef, true) ||
			!validRef(annotation.SourceAssetID, true) || !validRef(annotation.AuthorRef, true) ||
			!validSHA256(annotation.SourceAssetSHA256) || !validSHA256(annotation.OverlaySHA256) ||
			annotation.SourceRevision != manifest.Revision || !validText(annotation.Label, 320, true) {
			return ErrAnnotationInvalid
		}
		if _, duplicate := seen[annotation.AnnotationRef]; duplicate {
			return ErrAnnotationInvalid
		}
		seen[annotation.AnnotationRef] = struct{}{}
		if _, ok := canonicalTime(annotation.CreatedAt, true); !ok {
			return ErrAnnotationInvalid
		}
		observedAsset, ok := observationAssets[annotation.ObservationRef]
		if !ok || observedAsset != annotation.SourceAssetID {
			return ErrAnnotationInvalid
		}
		asset, ok := assets[annotation.SourceAssetID]
		if !ok || asset.SHA256 != annotation.SourceAssetSHA256 || asset.Width <= 0 || asset.Height <= 0 {
			return ErrAnnotationInvalid
		}
		if err := validateAnnotationGeometry(annotation.Geometry, asset.Width, asset.Height); err != nil {
			return err
		}
	}
	return nil
}

func validateAnnotationGeometry(geometry AnnotationGeometry, assetWidth, assetHeight int) error {
	if geometry.CoordinateSpace != "source_pixels" || geometry.SourceWidth != assetWidth || geometry.SourceHeight != assetHeight ||
		!rectangleWithin(geometry.X, geometry.Y, geometry.Width, geometry.Height, 0, 0, assetWidth, assetHeight) {
		return ErrAnnotationInvalid
	}
	switch geometry.RotationDegrees {
	case 0, 90, 180, 270:
	default:
		return ErrAnnotationInvalid
	}
	cropAbsent := geometry.CropX == 0 && geometry.CropY == 0 && geometry.CropWidth == 0 && geometry.CropHeight == 0
	if cropAbsent {
		return nil
	}
	if !rectangleWithin(geometry.CropX, geometry.CropY, geometry.CropWidth, geometry.CropHeight, 0, 0, assetWidth, assetHeight) ||
		!rectangleWithin(geometry.X, geometry.Y, geometry.Width, geometry.Height, geometry.CropX, geometry.CropY, geometry.CropWidth, geometry.CropHeight) {
		return ErrAnnotationInvalid
	}
	return nil
}

func rectangleWithin(x, y, width, height, boundX, boundY, boundWidth, boundHeight int) bool {
	if x < boundX || y < boundY || width <= 0 || height <= 0 || boundWidth <= 0 || boundHeight <= 0 ||
		width > boundWidth || height > boundHeight {
		return false
	}
	return x-boundX <= boundWidth-width && y-boundY <= boundHeight-height
}
