package vision

// ProviderBrowserRejectedClass is raised when an identity provider rejects
// the automated browser context itself (#152): Google's "This browser or app
// may not be secure" page and equivalents. Retrying credentials inside the
// rejected context is the wrong move; the typed handoff routes are the fix.
const ProviderBrowserRejectedClass = "provider_browser_rejected"

// ProviderRejectionSurfaceHint is the honest completion guidance: the engine
// never fabricates provider compliance - it routes to supported browsers.
const ProviderRejectionSurfaceHint = "provider rejected this automated browser context; do not retry credentials here - complete consent through an official headed operator browser, attach to an explicitly authorized operator browser with browser_attach, or use the provider's native CLI/device flow, then resume the requesting workflow with the resulting token"

// FormatProviderRejectionHint renders the diagnostics hint for a pending
// provider rejection.
func FormatProviderRejectionHint() string {
	return "provider browser rejection observed: " + ProviderRejectionSurfaceHint
}
