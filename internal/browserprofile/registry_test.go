package browserprofile

import "testing"

func TestReadyDefaultsResolve(t *testing.T) {
	registry, err := NewRegistry(ReadyDefaultConfig())
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	for _, name := range []string{"detect", "no_detect", "operator", "research"} {
		profile, err := registry.Resolve(name)
		if err != nil {
			t.Fatalf("Resolve(%s): %v", name, err)
		}
		if profile.Digest == "" {
			t.Fatalf("Resolve(%s): empty digest", name)
		}
	}
}

func TestNoDetectInheritsAndOverrides(t *testing.T) {
	registry, err := NewRegistry(ReadyDefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	profile, err := registry.Resolve("no_detect")
	if err != nil {
		t.Fatal(err)
	}
	if profile.Mode != ModeNoDetect {
		t.Fatalf("mode = %q", profile.Mode)
	}
	if !profile.Identity.PatchWebDriver || !profile.Identity.DisableAutomationControlled {
		t.Fatal("no_detect profile did not inherit stealth identity posture")
	}
	if profile.Identity.Viewport.Width != 1280 || profile.Identity.Locale != "en-US" {
		t.Fatal("no_detect profile lost detect base identity defaults")
	}
	if profile.Challenge.Policy != ChallengeSolveAndRetry {
		t.Fatalf("challenge policy = %q", profile.Challenge.Policy)
	}
}

func TestExplicitModeSelection(t *testing.T) {
	registry, err := NewRegistry(ReadyDefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	profile, selection, err := registry.Select("https://example.com", "", ModeResearch)
	if err != nil {
		t.Fatal(err)
	}
	if profile.ID != "research" || selection.EffectiveMode != ModeResearch {
		t.Fatalf("unexpected selection: %#v", selection)
	}
}

func TestDomainRuleSelection(t *testing.T) {
	cfg := ReadyDefaultConfig()
	cfg.DomainRules = []DomainRule{{Pattern: "*.example.com", Profile: "no_detect", Priority: 10}}
	registry, err := NewRegistry(cfg)
	if err != nil {
		t.Fatal(err)
	}
	profile, selection, err := registry.Select("https://app.example.com/path", "", ModeAuto)
	if err != nil {
		t.Fatal(err)
	}
	if profile.ID != "no_detect" {
		t.Fatalf("profile = %q", profile.ID)
	}
	if len(selection.Reasons) == 0 || selection.Reasons[0] != "domain_rule" {
		t.Fatalf("reasons = %#v", selection.Reasons)
	}
}

func TestInheritanceCycleRejected(t *testing.T) {
	cfg := Config{DefaultProfile: "a", Profiles: map[string]Profile{
		"a": {Extends: "b", Mode: ModeDetect, Engine: EngineChromium},
		"b": {Extends: "a", Mode: ModeDetect, Engine: EngineChromium},
	}}
	if _, err := NewRegistry(cfg); err == nil {
		t.Fatal("expected inheritance cycle error")
	}
}

func TestOperatorRequiresExclusiveLock(t *testing.T) {
	cfg := ReadyDefaultConfig()
	operator := cfg.Profiles["operator"]
	operator.Storage.ExclusiveLock = false
	cfg.Profiles["operator"] = operator
	if _, err := NewRegistry(cfg); err == nil {
		t.Fatal("expected operator exclusive-lock validation error")
	}
}
