package captcha

import "github.com/go-rod/rod"

// RodBrowser exposes the connected browser runtime to the shared browser
// profile manager. Lifecycle remains owned by ProxiedBrowser.Close.
func (pb *ProxiedBrowser) RodBrowser() *rod.Browser {
	if pb == nil {
		return nil
	}
	return pb.browser
}

// PID returns the managed browser process ID when one is available.
func (pb *ProxiedBrowser) PID() int {
	if pb == nil || pb.launcher == nil {
		return 0
	}
	return pb.launcher.PID()
}

// RouteLabel returns the redacted network-route label.
func (pb *ProxiedBrowser) RouteLabel() string {
	if pb == nil {
		return ""
	}
	return pb.label
}
