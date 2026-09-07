package evidenceregistry

import (
	"context"
	"fmt"

	"github.com/WPUIAI/uiai-engine/internal/evidenceartifact"
)

// RebuildResult is the deterministic projection outcome from immutable artifact storage.
type RebuildResult struct {
	Schema      string `json:"schema"`
	Artifacts   uint64 `json:"artifacts"`
	Projects    uint64 `json:"projects"`
	IndexErrors uint64 `json:"index_errors"`
}

// RebuildFromArtifactStore replays immutable manifests into rebuildable per-project indexes.
// The artifact store remains authoritative for bytes and manifests; the registry is only a read projection.
func (m *Manager) RebuildFromArtifactStore(ctx context.Context, artifacts *evidenceartifact.Store) (RebuildResult, error) {
	result := RebuildResult{Schema: "uiai.evidence_registry_rebuild.v1"}
	if artifacts == nil {
		return result, fmt.Errorf("%w: artifact store unavailable", ErrIndexUnavailable)
	}
	projects := make(map[string]struct{})
	for _, entry := range artifacts.List() {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		manifest, _, err := artifacts.GetManifest(entry.ArtifactID, entry.Revision)
		if err != nil {
			result.IndexErrors++
			continue
		}
		store, err := m.EnsureProject(ctx, manifest.Scope.Project.ProjectRef)
		if err != nil {
			result.IndexErrors++
			continue
		}
		if _, err := store.Index(ctx, IndexInput{Manifest: manifest, ManifestSHA256: entry.ManifestSHA256}); err != nil {
			result.IndexErrors++
			continue
		}
		result.Artifacts++
		projects[manifest.Scope.Project.ProjectRef] = struct{}{}
	}
	result.Projects = uint64(len(projects))
	if result.IndexErrors != 0 {
		return result, fmt.Errorf("%w: %d immutable manifests could not be projected", ErrIndexUnavailable, result.IndexErrors)
	}
	return result, nil
}
