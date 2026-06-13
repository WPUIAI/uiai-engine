package config

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const rbwLookupTimeout = 10 * time.Second

// expandConfigRefs expands normal environment references and UIAI secret-provider
// references. Supported secret refs:
//
//	${rbw:item}        -> rbw get item
//	${rbw:item:field}  -> rbw get --field field item
//
// rbw execution is opt-in: it only runs when a config value explicitly uses an
// rbw reference. Set UIAI_RBW_BIN to choose a specific rbw binary; otherwise
// PATH lookup is used. Secret values are returned to the config loader only and
// must not be logged by callers.
func expandConfigRefs(input string) (string, error) {
	var firstErr error
	expanded := os.Expand(input, func(key string) string {
		if strings.HasPrefix(key, "rbw:") || strings.HasPrefix(key, "RBW:") {
			value, err := resolveRBWRef(key[4:])
			if err != nil && firstErr == nil {
				firstErr = err
			}
			return value
		}
		return os.Getenv(key)
	})
	if firstErr != nil {
		return "", firstErr
	}
	return expanded, nil
}

func resolveRBWRef(ref string) (string, error) {
	parts := strings.SplitN(ref, ":", 2)
	item := strings.TrimSpace(parts[0])
	field := ""
	if len(parts) == 2 {
		field = strings.TrimSpace(parts[1])
	}
	if item == "" {
		return "", fmt.Errorf("rbw config reference missing item name")
	}

	bin := strings.TrimSpace(os.Getenv("UIAI_RBW_BIN"))
	if bin == "" {
		var err error
		bin, err = exec.LookPath("rbw")
		if err != nil {
			return "", fmt.Errorf("resolve rbw config reference %q: rbw not found in PATH; set UIAI_RBW_BIN", item)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), rbwLookupTimeout)
	defer cancel()

	args := []string{"get"}
	if field != "" {
		args = append(args, "--field", field)
	}
	args = append(args, item)

	out, err := exec.CommandContext(ctx, bin, args...).Output() // #nosec G204 -- rbw path is explicit operator config/PATH; args are not shell-interpreted.
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("resolve rbw config reference %q: rbw timed out", item)
	}
	if err != nil {
		return "", fmt.Errorf("resolve rbw config reference %q: %w", item, err)
	}
	return strings.TrimRight(string(out), "\r\n"), nil
}
