package evidenceshare

import "time"

const Schema = "uiai.screenshot_evidence_share.v1"

type Scope struct {
	WorkpointRef  string `json:"workpoint_ref,omitempty"`
	ContinuityRef string `json:"continuity_ref,omitempty"`
}

type Input struct {
	Screenshot []byte
	Format     string
	Width      int
	Height     int
	SourceURL  string
	CapturedAt time.Time
	DurationMS int64
	Scope      Scope
}

type Manifest struct {
	Schema           string    `json:"schema"`
	ArtifactRef      string    `json:"artifact_ref"`
	ArtifactSHA256   string    `json:"artifact_sha256"`
	ScreenshotRef    string    `json:"screenshot_ref"`
	ScreenshotSHA256 string    `json:"screenshot_sha256"`
	Format           string    `json:"format"`
	MIME             string    `json:"mime"`
	Bytes            int       `json:"bytes"`
	Width            int       `json:"width"`
	Height           int       `json:"height"`
	SourceURL        string    `json:"source_url"`
	CapturedAt       time.Time `json:"captured_at"`
	DurationMS       int64     `json:"duration_ms"`
	Availability     string    `json:"availability"`
	Access           string    `json:"access"`
	Interaction      string    `json:"interaction"`
	Scope            Scope     `json:"scope"`
	TruthNotice      string    `json:"truth_notice"`
}

type Result struct {
	ArtifactRef    string
	ArtifactSHA256 string
	RelativePath   string
	Directory      string
}
