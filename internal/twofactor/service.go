package twofactor

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
)

type Service struct {
	cfg *config.Config
	Now func() time.Time
}

type Request struct {
	Profile string `json:"profile"`
	Issuer  string `json:"issuer,omitempty"`
	Name    string `json:"name,omitempty"`
}

type Response struct {
	Profile   string `json:"profile"`
	Provider  string `json:"provider"`
	Code      string `json:"code"`
	ExpiresIn int64  `json:"expires_in"`
	Period    int    `json:"period"`
	Digits    int    `json:"digits"`
	Issuer    string `json:"issuer,omitempty"`
	Name      string `json:"name,omitempty"`
}

func New(cfg *config.Config) *Service {
	return &Service{cfg: cfg, Now: time.Now}
}

func (s *Service) Code(ctx context.Context, req Request) (Response, error) {
	if s == nil || s.cfg == nil || !s.cfg.TwoFactor.Enabled {
		return Response{}, fmt.Errorf("two_factor is not enabled")
	}
	profileName := strings.TrimSpace(req.Profile)
	if profileName == "" {
		return Response{}, fmt.Errorf("profile required")
	}
	profile, ok := s.cfg.TwoFactor.Profiles[profileName]
	if !ok {
		return Response{}, fmt.Errorf("unknown two_factor profile %q", profileName)
	}
	provider := strings.ToLower(strings.TrimSpace(profile.Provider))
	if provider == "" {
		provider = "totp"
	}
	issuer := firstNonEmpty(req.Issuer, profile.Issuer)
	name := firstNonEmpty(req.Name, profile.Name)

	switch provider {
	case "totp":
		code, err := codeFromTOTPProfile(profile, s.Now())
		if err != nil {
			return Response{}, err
		}
		return Response{Profile: profileName, Provider: provider, Code: code.Code, ExpiresIn: code.ExpiresIn, Period: code.Period, Digits: code.Digits, Issuer: issuer, Name: name}, nil
	case "aegis", "aegis-rs", "command":
		return s.codeFromCommand(ctx, profileName, provider, profile, issuer, name)
	default:
		return Response{}, fmt.Errorf("unsupported two_factor provider %q", profile.Provider)
	}
}

func codeFromTOTPProfile(profile config.TwoFactorProfile, now time.Time) (Code, error) {
	if profile.OTPAuthURL != "" {
		return FromOTPAuth(profile.OTPAuthURL, now)
	}
	return Generate(profile.Secret, profile.Algorithm, profile.Digits, profile.Period, now)
}

func (s *Service) codeFromCommand(ctx context.Context, profileName, provider string, profile config.TwoFactorProfile, issuer, name string) (Response, error) {
	cmdPath := firstNonEmpty(profile.Command, "aegis")
	if profile.VaultFile == "" {
		return Response{}, fmt.Errorf("two_factor profile %q requires vault_file for %s provider", profileName, provider)
	}
	args := []string{"--json"}
	if issuer != "" {
		args = append(args, "--issuer", issuer)
	}
	if name != "" {
		args = append(args, "--name", name)
	}
	if profile.Password != "" {
		args = append(args, "--password", profile.Password)
	}
	if profile.PasswordFile != "" {
		args = append(args, "--password-file", profile.PasswordFile)
	}
	args = append(args, profile.VaultFile)

	cmdCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	out, err := exec.CommandContext(cmdCtx, cmdPath, args...).Output() // #nosec G204 -- command path/args are explicit operator config; no shell interpolation.
	if cmdCtx.Err() == context.DeadlineExceeded {
		return Response{}, fmt.Errorf("two_factor provider %q timed out", provider)
	}
	if err != nil {
		return Response{}, fmt.Errorf("two_factor provider %q failed: %w", provider, err)
	}
	parsed, err := parseCommandOTP(out)
	if err != nil {
		return Response{}, err
	}
	if parsed.Code == "" {
		return Response{}, fmt.Errorf("two_factor provider %q returned no code", provider)
	}
	if parsed.Period == 0 {
		parsed.Period = profile.Period
	}
	if parsed.Digits == 0 {
		parsed.Digits = len(parsed.Code)
	}
	return Response{Profile: profileName, Provider: provider, Code: parsed.Code, ExpiresIn: parsed.ExpiresIn, Period: parsed.Period, Digits: parsed.Digits, Issuer: firstNonEmpty(parsed.Issuer, issuer), Name: firstNonEmpty(parsed.Name, name)}, nil
}

type commandOTP struct {
	Code      string `json:"code"`
	OTP       string `json:"otp"`
	Token     string `json:"token"`
	ExpiresIn int64  `json:"expires_in"`
	Remaining int64  `json:"remaining_time"`
	Period    int    `json:"period"`
	Digits    int    `json:"digits"`
	Issuer    string `json:"issuer"`
	Name      string `json:"name"`
}

func parseCommandOTP(out []byte) (commandOTP, error) {
	var item commandOTP
	if err := json.Unmarshal(out, &item); err == nil {
		item.Code = firstNonEmpty(item.Code, item.OTP, item.Token)
		if item.ExpiresIn == 0 {
			item.ExpiresIn = item.Remaining
		}
		return item, nil
	}
	var items []commandOTP
	if err := json.Unmarshal(out, &items); err == nil && len(items) > 0 {
		items[0].Code = firstNonEmpty(items[0].Code, items[0].OTP, items[0].Token)
		if items[0].ExpiresIn == 0 {
			items[0].ExpiresIn = items[0].Remaining
		}
		return items[0], nil
	}
	text := strings.TrimSpace(string(out))
	if text == "" {
		return commandOTP{}, fmt.Errorf("two_factor provider returned empty output")
	}
	return commandOTP{Code: text, Digits: len(text)}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
