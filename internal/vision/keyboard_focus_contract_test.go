package vision

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/cdp"
)

var errKeyboardFixture = errors.New("synthetic keyboard transport stop")

type keyboardActivationClient struct {
	stalledScreenshotClient
	denyActivation bool
	activated      bool
	dispatched     bool
}

func (c *keyboardActivationClient) Call(ctx context.Context, session, method string, params interface{}) ([]byte, error) {
	switch method {
	case "Target.activateTarget":
		c.activated = true
		if c.denyActivation {
			return nil, errKeyboardFixture
		}
		return []byte(`{}`), nil
	case "Input.dispatchKeyEvent":
		c.dispatched = true
		return nil, errKeyboardFixture
	default:
		return c.stalledScreenshotClient.Call(ctx, session, method, params)
	}
}

func TestPressRequiresActivationBeforeInput(t *testing.T) {
	for _, deny := range []bool{false, true} {
		name := "activation_before_dispatch"
		if deny {
			name = "activation_denial_stops_dispatch"
		}
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			client := &keyboardActivationClient{stalledScreenshotClient: stalledScreenshotClient{events: make(chan *cdp.Event)}, denyActivation: deny}
			browser := rod.New().NoDefaultDevice().Context(ctx).Client(client)
			if err := browser.Connect(); err != nil {
				t.Fatal(err)
			}
			page, err := browser.PageFromTarget("synthetic-keyboard-target")
			if err != nil {
				t.Fatal(err)
			}
			_, err = (&Session{page: page}).Press("a")
			if !errors.Is(err, errKeyboardFixture) {
				t.Fatalf("transport failure not preserved: %v", err)
			}
			if !client.activated {
				t.Fatal("keyboard input attempted without target activation")
			}
			if client.dispatched == deny {
				t.Fatal("activation denial/input dispatch boundary violated")
			}
			if deny && !strings.Contains(err.Error(), "page focus failed") {
				t.Fatal("activation failure was reported as successful keyboard dispatch")
			}
		})
	}
}
