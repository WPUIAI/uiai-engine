package vision

import (
	"fmt"

	"github.com/go-rod/rod"
)

// inputElement restores the action context after the short lookup attempt and
// checks actual HTML interactivity. Rod v0.116.2 checks `readonly` rather than
// the DOM's `readOnly` property, so its Input can silently succeed without input.
func inputElement(page *rod.Page, selector string) (*rod.Element, error) {
	element, err := actionElement(page, selector)
	if err != nil {
		return nil, err
	}
	if err = element.Wait(rod.Eval(`() => !this.readOnly && !this.disabled`)); err != nil {
		return nil, fmt.Errorf("input not writable: %w", err)
	}
	return element, nil
}

// actionElement detaches only the short lookup deadline, preserving the caller's
// explicitly bounded action context rather than returning a poisoned element.
func actionElement(page *rod.Page, selector string) (*rod.Element, error) {
	element, err := retryElement(page, selector)
	if err != nil {
		return nil, err
	}
	return element.Context(page.GetContext()), nil
}
