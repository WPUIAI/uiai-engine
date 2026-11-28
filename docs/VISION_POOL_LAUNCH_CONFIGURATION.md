# Vision pool launch configuration and browser compatibility

## Scope and defaults

The Go vision pool (`/api/session`, one-shot screenshots and MultiPool) previously hard-coded headless Chromium. Configure actual launch preferences independently of named managed browser profiles:

```yaml
vision:
  headless: true
  # For a display-backed layout or operator test:
  # headless: false
  # window_size: "1920,1080"
  # locale: "en-US"
```

Omitting headless keeps existing server behavior. A headed process requires an available display; failures retain the existing typed launch diagnostics. The optional window dimensions configure the native browser window, not a session's explicit viewport. Locale is a Chromium localization preference. These fields do not override a site's security checks or change automation disclosure.

Named managed browser profiles already have their own headless, viewport and locale settings. Attached browsers are owned by their operator and their launch configuration is not rewritten. Challenge-related runtimes use separate configuration; this PR does not change challenge handling.

## Acceptance and compatibility

- Single and multiple vision pools receive the same explicit configuration, with headless still the default.
- Invalid dimensions or malformed locale are refused at initialization, not silently ignored.
- Diagnostics report the configured launch preferences for each pool.
- Existing screenshot dimensions, ephemeral context isolation, SSRF checks, origin policy, evidence capture, and explicit session viewport stay unchanged.
- Verify the existing Chromium screenshot and evidence regression suites on Browser Reliability CI. A headed smoke test needs an operator-approved display-backed host and an owned test page. No production third-party login or anti-bot bypass test is part of this change.
