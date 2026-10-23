// Package desktopcontract defines the versioned, transport-neutral contracts
// shared by UIAI Engine, UIAI Cockpit, and Focusa Menubar for desktop browser
// presentation and app handoff. It deliberately contains no launcher, browser,
// window, daemon, or persistence authority.
package desktopcontract

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	SchemaBrowserRuntimeManifestV1     = "uiai.browser_runtime_manifest.v1"
	SchemaDesktopPresentationRequestV1 = "uiai.desktop_presentation_request.v1"
	SchemaDesktopPresentationReceiptV1 = "uiai.desktop_presentation_receipt.v1"
	SchemaDesktopPresentationStatusV1  = "uiai.desktop_presentation_status.v1"
	SchemaAppHandoffIntentV1           = "uiai.app_handoff_intent.v1"
	SchemaAppHandoffReceiptV1          = "uiai.app_handoff_receipt.v1"
	SchemaFocusaAppManifestV2          = "focusa.app.manifest.v2"
)

var SchemaIDs = []string{
	SchemaBrowserRuntimeManifestV1,
	SchemaDesktopPresentationRequestV1,
	SchemaDesktopPresentationReceiptV1,
	SchemaDesktopPresentationStatusV1,
	SchemaAppHandoffIntentV1,
	SchemaAppHandoffReceiptV1,
	SchemaFocusaAppManifestV2,
}

type ScopeRef struct {
	ProjectRootKey string `json:"project_root_key,omitempty"`
	WorkstreamKey  string `json:"workstream_key,omitempty"`
	ContinuityID   string `json:"continuity_id,omitempty"`
	ThreadID       string `json:"thread_id,omitempty"`
	SessionID      string `json:"session_id,omitempty"`
	AuthorityState string `json:"authority_state"`
}

type ClientRef struct {
	ClientType string `json:"client_type"`
	ClientID   string `json:"client_id"`
}

type BrowserRuntimeManifest struct {
	Schema            string `json:"schema"`
	RuntimeID         string `json:"runtime_id"`
	Engine            string `json:"engine"`
	Version           string `json:"version"`
	CDPProtocol       string `json:"cdp_protocol"`
	Platform          string `json:"platform"`
	Arch              string `json:"arch"`
	ExecutableRelPath string `json:"executable_relpath"`
	SHA256            string `json:"sha256"`
	Signed            bool   `json:"signed"`
	Source            string `json:"source"`
	BuiltAt           string `json:"built_at"`
}

type DesktopPresentationRequest struct {
	Schema         string    `json:"schema"`
	Mode           string    `json:"mode"`
	Reason         string    `json:"reason"`
	ScopeRef       *ScopeRef `json:"scope_ref,omitempty"`
	RequestedBy    ClientRef `json:"requested_by"`
	Focus          bool      `json:"focus"`
	ExpiresInMS    int64     `json:"expires_in_ms"`
	IdempotencyKey string    `json:"idempotency_key"`
}

type DesktopPresentationReceipt struct {
	Schema            string `json:"schema"`
	PresentationID    string `json:"presentation_id"`
	SessionID         string `json:"session_id"`
	Status            string `json:"status"`
	CockpitInstanceID string `json:"cockpit_instance_id,omitempty"`
	HandoffRef        string `json:"handoff_ref,omitempty"`
	ReasonCode        string `json:"reason_code,omitempty"`
	CreatedAt         string `json:"created_at"`
	ExpiresAt         string `json:"expires_at"`
}

type DesktopPresentationStatus struct {
	Schema         string `json:"schema"`
	PresentationID string `json:"presentation_id"`
	SessionID      string `json:"session_id"`
	Status         string `json:"status"`
	ReasonCode     string `json:"reason_code,omitempty"`
	ObservedAt     string `json:"observed_at"`
}

type AppHandoffIntent struct {
	Schema          string    `json:"schema"`
	IntentID        string    `json:"intent_id"`
	Scheme          string    `json:"scheme"`
	Route           string    `json:"route"`
	TargetRef       string    `json:"target_ref"`
	RequestedBy     ClientRef `json:"requested_by"`
	ProtocolVersion string    `json:"protocol_version"`
	CreatedAt       string    `json:"created_at"`
	ExpiresAt       string    `json:"expires_at"`
}

type AppHandoffReceipt struct {
	Schema      string `json:"schema"`
	IntentID    string `json:"intent_id"`
	Status      string `json:"status"`
	TargetApp   string `json:"target_app"`
	ResolvedRef string `json:"resolved_ref,omitempty"`
	ReasonCode  string `json:"reason_code,omitempty"`
	ObservedAt  string `json:"observed_at"`
}

type AppManifest struct {
	Schema       string            `json:"schema"`
	App          string            `json:"app"`
	Version      string            `json:"version"`
	Channel      string            `json:"channel"`
	Protocols    map[string]string `json:"protocols"`
	Capabilities []string          `json:"capabilities"`
}

type FixtureBundle struct {
	RuntimeManifest     BrowserRuntimeManifest     `json:"runtime_manifest"`
	PresentationRequest DesktopPresentationRequest `json:"presentation_request"`
	PresentationReceipt DesktopPresentationReceipt `json:"presentation_receipt"`
	PresentationStatus  DesktopPresentationStatus  `json:"presentation_status"`
	HandoffIntent       AppHandoffIntent           `json:"handoff_intent"`
	HandoffReceipt      AppHandoffReceipt          `json:"handoff_receipt"`
	AppManifest         AppManifest                `json:"app_manifest"`
}

var opaqueRefPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,159}$`)
var sha256Pattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

var presentationModes = map[string]struct{}{
	"full": {}, "pip": {}, "focus_existing": {},
}

var presentationReasons = map[string]struct{}{
	"operator_request": {}, "takeover_required": {}, "policy_confirmation": {},
	"failure_recovery": {}, "workflow": {},
}

var presentationStatuses = map[string]struct{}{
	"requested": {}, "resolving_session": {}, "resolving_cockpit": {}, "launching": {},
	"attaching": {}, "visible": {}, "focused": {}, "already_visible": {}, "blocked": {},
	"unavailable": {}, "failed": {}, "blocked_scope": {}, "session_missing": {},
	"cockpit_missing": {}, "incompatible": {}, "attach_failed": {}, "desktop_unavailable": {},
	"expired": {}, "cancelled": {},
}

var authorityStates = map[string]struct{}{
	"verified": {}, "missing": {}, "stale": {}, "conflict": {}, "read_only": {},
}

var allowedHandoffRoutes = map[string]map[string]struct{}{
	"focusa": {
		"mission": {}, "card": {}, "workpoint": {}, "connect": {},
	},
	"cockpit": {
		"live/session": {}, "focusa": {}, "evidence": {}, "settings/pairing": {},
	},
}

func ValidateOpaqueRef(value string) error {
	if !opaqueRefPattern.MatchString(value) {
		return errors.New("must be an opaque identifier of 1-160 safe characters")
	}
	if strings.Contains(value, "://") || strings.ContainsAny(value, `/\\?#@`) {
		return errors.New("must not contain a URL, path, query, fragment, or authority marker")
	}
	return nil
}

func ValidateRuntimeManifest(v BrowserRuntimeManifest) error {
	if v.Schema != SchemaBrowserRuntimeManifestV1 {
		return fmt.Errorf("unexpected runtime schema %q", v.Schema)
	}
	if err := ValidateOpaqueRef(v.RuntimeID); err != nil {
		return fmt.Errorf("runtime_id: %w", err)
	}
	if v.Engine != "chromium" || v.Version == "" || v.CDPProtocol == "" || v.Platform == "" || v.Arch == "" {
		return errors.New("runtime engine, version, CDP protocol, platform, and architecture are required")
	}
	if v.ExecutableRelPath == "" || strings.HasPrefix(v.ExecutableRelPath, "/") || strings.Contains(v.ExecutableRelPath, "..") {
		return errors.New("executable_relpath must be a bounded relative package path")
	}
	if !sha256Pattern.MatchString(v.SHA256) {
		return errors.New("sha256 must be 64 lowercase hexadecimal characters")
	}
	if v.Source != "uiai-release" {
		return errors.New("runtime source must be uiai-release")
	}
	return validateRFC3339("built_at", v.BuiltAt)
}

func ValidatePresentationRequest(v DesktopPresentationRequest) error {
	if v.Schema != SchemaDesktopPresentationRequestV1 {
		return fmt.Errorf("unexpected presentation request schema %q", v.Schema)
	}
	if _, ok := presentationModes[v.Mode]; !ok {
		return fmt.Errorf("unsupported presentation mode %q", v.Mode)
	}
	if _, ok := presentationReasons[v.Reason]; !ok {
		return fmt.Errorf("unsupported presentation reason %q", v.Reason)
	}
	if err := validateClientRef(v.RequestedBy); err != nil {
		return err
	}
	if v.ExpiresInMS < 1000 || v.ExpiresInMS > 300000 {
		return errors.New("expires_in_ms must be between 1000 and 300000")
	}
	if err := ValidateOpaqueRef(v.IdempotencyKey); err != nil {
		return fmt.Errorf("idempotency_key: %w", err)
	}
	if v.ScopeRef != nil {
		if _, ok := authorityStates[v.ScopeRef.AuthorityState]; !ok {
			return fmt.Errorf("unsupported authority_state %q", v.ScopeRef.AuthorityState)
		}
		for name, value := range map[string]string{
			"project_root_key": v.ScopeRef.ProjectRootKey,
			"workstream_key":   v.ScopeRef.WorkstreamKey,
			"continuity_id":    v.ScopeRef.ContinuityID,
			"thread_id":        v.ScopeRef.ThreadID,
			"session_id":       v.ScopeRef.SessionID,
		} {
			if value != "" {
				if err := ValidateOpaqueRef(value); err != nil {
					return fmt.Errorf("scope_ref.%s: %w", name, err)
				}
			}
		}
	}
	return nil
}

func ValidatePresentationReceipt(v DesktopPresentationReceipt) error {
	if v.Schema != SchemaDesktopPresentationReceiptV1 {
		return fmt.Errorf("unexpected presentation receipt schema %q", v.Schema)
	}
	for name, value := range map[string]string{
		"presentation_id": v.PresentationID, "session_id": v.SessionID,
	} {
		if err := ValidateOpaqueRef(value); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	if v.CockpitInstanceID != "" {
		if err := ValidateOpaqueRef(v.CockpitInstanceID); err != nil {
			return fmt.Errorf("cockpit_instance_id: %w", err)
		}
	}
	if v.HandoffRef != "" {
		if err := ValidateOpaqueRef(v.HandoffRef); err != nil {
			return fmt.Errorf("handoff_ref: %w", err)
		}
	}
	if _, ok := presentationStatuses[v.Status]; !ok {
		return fmt.Errorf("unsupported presentation status %q", v.Status)
	}
	if err := validateRFC3339("created_at", v.CreatedAt); err != nil {
		return err
	}
	return validateRFC3339("expires_at", v.ExpiresAt)
}

func ValidatePresentationStatus(v DesktopPresentationStatus) error {
	if v.Schema != SchemaDesktopPresentationStatusV1 {
		return fmt.Errorf("unexpected presentation status schema %q", v.Schema)
	}
	if err := ValidateOpaqueRef(v.PresentationID); err != nil {
		return fmt.Errorf("presentation_id: %w", err)
	}
	if err := ValidateOpaqueRef(v.SessionID); err != nil {
		return fmt.Errorf("session_id: %w", err)
	}
	if _, ok := presentationStatuses[v.Status]; !ok {
		return fmt.Errorf("unsupported presentation status %q", v.Status)
	}
	return validateRFC3339("observed_at", v.ObservedAt)
}

func ValidateHandoffIntent(v AppHandoffIntent) error {
	if v.Schema != SchemaAppHandoffIntentV1 {
		return fmt.Errorf("unexpected handoff intent schema %q", v.Schema)
	}
	if err := ValidateOpaqueRef(v.IntentID); err != nil {
		return fmt.Errorf("intent_id: %w", err)
	}
	routes, ok := allowedHandoffRoutes[v.Scheme]
	if !ok {
		return fmt.Errorf("unsupported handoff scheme %q", v.Scheme)
	}
	if _, ok := routes[v.Route]; !ok {
		return fmt.Errorf("unsupported %s handoff route %q", v.Scheme, v.Route)
	}
	if err := ValidateOpaqueRef(v.TargetRef); err != nil {
		return fmt.Errorf("target_ref: %w", err)
	}
	if err := validateClientRef(v.RequestedBy); err != nil {
		return err
	}
	if v.ProtocolVersion != "1" {
		return fmt.Errorf("unsupported protocol_version %q", v.ProtocolVersion)
	}
	if err := validateRFC3339("created_at", v.CreatedAt); err != nil {
		return err
	}
	if err := validateRFC3339("expires_at", v.ExpiresAt); err != nil {
		return err
	}
	created, _ := time.Parse(time.RFC3339, v.CreatedAt)
	expires, _ := time.Parse(time.RFC3339, v.ExpiresAt)
	if !expires.After(created) || expires.Sub(created) > 5*time.Minute {
		return errors.New("handoff expiry must be after creation and no more than five minutes later")
	}
	return nil
}

func ValidateHandoffReceipt(v AppHandoffReceipt) error {
	if v.Schema != SchemaAppHandoffReceiptV1 {
		return fmt.Errorf("unexpected handoff receipt schema %q", v.Schema)
	}
	if err := ValidateOpaqueRef(v.IntentID); err != nil {
		return fmt.Errorf("intent_id: %w", err)
	}
	if v.ResolvedRef != "" {
		if err := ValidateOpaqueRef(v.ResolvedRef); err != nil {
			return fmt.Errorf("resolved_ref: %w", err)
		}
	}
	if v.TargetApp != "focusa-menubar" && v.TargetApp != "uaiengine-cockpit" {
		return fmt.Errorf("unsupported target_app %q", v.TargetApp)
	}
	if v.Status != "opened" && v.Status != "focused" && v.Status != "blocked" && v.Status != "unavailable" && v.Status != "failed" {
		return fmt.Errorf("unsupported handoff receipt status %q", v.Status)
	}
	return validateRFC3339("observed_at", v.ObservedAt)
}

func ValidateAppManifest(v AppManifest) error {
	if v.Schema != SchemaFocusaAppManifestV2 {
		return fmt.Errorf("unexpected app manifest schema %q", v.Schema)
	}
	if v.App != "focusa-menubar" && v.App != "uaiengine-cockpit" {
		return fmt.Errorf("unsupported app %q", v.App)
	}
	if v.Version == "" || (v.Channel != "stable" && v.Channel != "preview" && v.Channel != "dev") {
		return errors.New("version and supported channel are required")
	}
	for _, protocol := range []string{"focusa_deep_link", "cockpit_deep_link", "desktop_presentation", "fpv"} {
		if v.Protocols[protocol] == "" {
			return fmt.Errorf("protocol %q is required", protocol)
		}
	}
	if len(v.Capabilities) == 0 {
		return errors.New("at least one capability is required")
	}
	return nil
}

func ValidateFixtureBundle(v FixtureBundle) error {
	validators := []func() error{
		func() error { return ValidateRuntimeManifest(v.RuntimeManifest) },
		func() error { return ValidatePresentationRequest(v.PresentationRequest) },
		func() error { return ValidatePresentationReceipt(v.PresentationReceipt) },
		func() error { return ValidatePresentationStatus(v.PresentationStatus) },
		func() error { return ValidateHandoffIntent(v.HandoffIntent) },
		func() error { return ValidateHandoffReceipt(v.HandoffReceipt) },
		func() error { return ValidateAppManifest(v.AppManifest) },
	}
	for _, validate := range validators {
		if err := validate(); err != nil {
			return err
		}
	}
	return nil
}

func validateClientRef(v ClientRef) error {
	allowed := map[string]struct{}{
		"pi": {}, "cockpit": {}, "menubar": {}, "api": {}, "mcp": {}, "cli": {},
	}
	if _, ok := allowed[v.ClientType]; !ok {
		return fmt.Errorf("unsupported client_type %q", v.ClientType)
	}
	if err := ValidateOpaqueRef(v.ClientID); err != nil {
		return fmt.Errorf("client_id: %w", err)
	}
	return nil
}

func validateRFC3339(name, value string) error {
	if _, err := time.Parse(time.RFC3339, value); err != nil {
		return fmt.Errorf("%s must be RFC3339: %w", name, err)
	}
	return nil
}

// RejectURLLikeValue is exported for adapters that validate a decoded link
// before mapping it into AppHandoffIntent.
func RejectURLLikeValue(value string) error {
	if parsed, err := url.Parse(value); err == nil && parsed.IsAbs() {
		return errors.New("absolute URLs are not allowed in handoff refs")
	}
	return ValidateOpaqueRef(value)
}
