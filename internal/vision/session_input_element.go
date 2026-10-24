package vision

import (
	"fmt"

	"github.com/go-rod/rod"
)

// inputElement restores the action context after the short lookup attempt and
// checks actual HTML interactivity. Rod v0.116.2 checks `readonly` rather than
// the DOM's `readOnly` property, so its Input can silently succeed without input.
func inputElement(page *rod.Page, selector string) (*rod.Element, error) {
	element, err := retryElement(page, selector)
	if err != nil {
		return nil, err
	}
	element = element.Context(page.GetContext())
	if err = element.Wait(rod.Eval(`() => !this.readOnly && !this.disabled`)); err != nil {
		return nil, fmt.Errorf("input not writable: %w", err)
	}
	return element, nil
}
