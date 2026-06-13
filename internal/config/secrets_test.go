package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestExpandConfigRefsEnv(t *testing.T) {
	t.Setenv("UIAI_TEST_SECRET", "env-secret")
	got, err := expandConfigRefs("webhook: ${UIAI_TEST_SECRET}")
	if err != nil {
		t.Fatalf("expandConfigRefs returned error: %v", err)
	}
	if got != "webhook: env-secret" {
		t.Fatalf("unexpected expansion: %q", got)
	}
}

func TestExpandConfigRefsRBWPassword(t *testing.T) {
	bin := fakeRBW(t)
	t.Setenv("UIAI_RBW_BIN", bin)

	got, err := expandConfigRefs("secret: ${rbw:WPUIAI Login}")
	if err != nil {
		t.Fatalf("expandConfigRefs returned error: %v", err)
	}
	if got != "secret: rbw-password-for-WPUIAI Login" {
		t.Fatalf("unexpected expansion: %q", got)
	}
}

func TestExpandConfigRefsRBWField(t *testing.T) {
	bin := fakeRBW(t)
	t.Setenv("UIAI_RBW_BIN", bin)

	got, err := expandConfigRefs("api_key: ${rbw:WPUIAI Login:api key}")
	if err != nil {
		t.Fatalf("expandConfigRefs returned error: %v", err)
	}
	if got != "api_key: rbw-field-api key-for-WPUIAI Login" {
		t.Fatalf("unexpected expansion: %q", got)
	}
}

func TestExpandConfigRefsRBWMissingBinary(t *testing.T) {
	t.Setenv("UIAI_RBW_BIN", filepath.Join(t.TempDir(), "missing-rbw"))
	_, err := expandConfigRefs("secret: ${rbw:Missing}")
	if err == nil || !strings.Contains(err.Error(), "resolve rbw config reference") {
		t.Fatalf("expected rbw resolution error, got %v", err)
	}
}

func fakeRBW(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	name := "rbw"
	if runtime.GOOS == "windows" {
		name = "rbw.bat"
	}
	path := filepath.Join(dir, name)
	var script string
	if runtime.GOOS == "windows" {
		script = "@echo off\nsetlocal enabledelayedexpansion\nset field=\nset item=\n:loop\nif \"%1\"==\"\" goto done\nif \"%1\"==\"--field\" (set field=%2& shift& shift& goto loop)\nif \"%1\"==\"get\" (shift& goto loop)\nset item=%1\nshift\ngoto loop\n:done\nif not \"%field%\"==\"\" (echo rbw-field-%field%-for-%item%) else (echo rbw-password-for-%item%)\n"
	} else {
		script = `#!/usr/bin/env sh
set -eu
field=""
item=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    get) shift ;;
    --field) field="$2"; shift 2 ;;
    *) item="$1"; shift ;;
  esac
done
if [ -n "$field" ]; then
  printf 'rbw-field-%s-for-%s\n' "$field" "$item"
else
  printf 'rbw-password-for-%s\n' "$item"
fi
`
	}
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatalf("write fake rbw: %v", err)
	}
	return path
}
