package vision

import (
	"testing"
	"time"
)

func TestInputPropagatesReadOnlyFailureWithoutMutation(t *testing.T) {
	_, manager, url := storageIsolationFixture(t, 1)
	t.Cleanup(manager.CloseAll)
	for _, method := range []struct {
		name  string
		apply func(*Session, string, string) (*SnapResult, error)
	}{{"fill", (*Session).Fill}, {"type", (*Session).Type}} {
		t.Run(method.name, func(t *testing.T) {
			session, _, err := manager.Open(url, 640, 480)
			if err != nil {
				t.Fatal(err)
			}
			defer manager.Close(session.ID)
			_, err = session.page.Timeout(2 * time.Second).Eval(`() => document.body.innerHTML='<input id="readonly" value="preserved" readonly>'`)
			if err != nil {
				t.Fatal(err)
			}
			start := time.Now()
			snapshot, err := method.apply(session, "#readonly", "synthetic-type-value")
			if err == nil || snapshot != nil {
				t.Fatal("input reported success despite a read-only input failure")
			}
			if time.Since(start) > 6*time.Second {
				t.Fatal("input failure exceeded its bounded action window")
			}
			if session.page.GetContext().Err() != nil {
				t.Fatal("input cancelled the shared page context")
			}
			value, err := session.page.Timeout(2 * time.Second).Eval(`() => document.querySelector('input').value`)
			if err != nil || value.Value.Str() != "preserved" {
				t.Fatal("rejected input changed the read-only field")
			}
		})
	}
}
