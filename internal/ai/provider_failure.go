package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// Provider failures retain only safe identity/code/classification, never an
// upstream response message. Retry policy must not depend on its display text.
type providerFailure struct {
	provider  string
	code      int
	http      bool
	retryable bool
	phase     string
	canceled  bool
	deadline  bool
}

func (e *providerFailure) Error() string {
	if e.phase != "" {
		return fmt.Sprintf("%s %s failed", e.provider, e.phase)
	}
	if e.http {
		return fmt.Sprintf("%s returned HTTP %d", e.provider, e.code)
	}
	return fmt.Sprintf("%s %d: provider request failed", e.provider, e.code)
}

// Context identity is projected as safe flags, never a retained private cause.
func (e *providerFailure) Is(target error) bool {
	return e.canceled && target == context.Canceled || e.deadline && target == context.DeadlineExceeded
}

func providerPhaseFailure(provider, phase string) error {
	return &providerFailure{provider: provider, phase: phase}
}

func providerIOFailure(provider, phase string, err error) error {
	return &providerFailure{
		provider: provider, phase: phase, retryable: isRetryableMessage(err.Error()),
		canceled: errors.Is(err, context.Canceled), deadline: errors.Is(err, context.DeadlineExceeded),
	}
}

func providerHTTPFailure(provider string, code int) error {
	return &providerFailure{provider: provider, code: code, http: true, retryable: code == 429 || code >= 500 && code <= 599}
}

func providerApplicationFailure(provider string, code int, message string) error {
	// Preserve the existing application's retry signals without retaining or
	// emitting its untrusted message, including through wrapping or fallback logs.
	return &providerFailure{provider: provider, code: code, retryable: isRetryableMessage(fmt.Sprintf("%d:", code)) || isRetryableMessage(message)}
}

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	var failure *providerFailure
	if errors.As(err, &failure) {
		return failure.retryable
	}
	return isRetryableMessage(err.Error())
}

// Legacy transport/context errors and provider application signals share one
// classifier; structured HTTP errors use their numeric status instead.
func isRetryableMessage(msg string) bool {
	for _, code := range []string{"500:", "502:", "503:", "504:", "520:", "521:", "522:", "529:"} {
		if strings.Contains(msg, code) {
			return true
		}
	}
	if strings.Contains(msg, "429:") || strings.Contains(msg, "rate limit") || strings.Contains(msg, "Rate limit") {
		return true
	}
	if strings.Contains(msg, "context deadline exceeded") || strings.Contains(msg, "context canceled") {
		return true
	}
	if strings.Contains(msg, "connection refused") || strings.Contains(msg, "no such host") {
		return true
	}
	return strings.Contains(msg, "EOF") || strings.Contains(msg, "broken pipe")
}
