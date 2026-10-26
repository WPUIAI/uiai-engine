package vision

import (
	"sync"
	"testing"
	"time"
)

func idleExpiryFixture() (*SessionManager, *Session) {
	manager := NewSessionManager(nil)
	session := &Session{ID: "synthetic-idle", CreatedAt: time.Now(), LastUsed: time.Now().Add(-2 * SessionTTL)}
	manager.sessions[session.ID] = session
	return manager, session
}

func TestIdleCallbackRechecksPendingAuth(t *testing.T) {
	manager, session := idleExpiryFixture()
	challenge := session.raiseAuthChallenge("credential_form", "https://example.test/login")
	// A timer callback already dispatched before Stop may still run afterwards.
	manager.expireSession(session.ID)
	if _, ok := manager.Get(session.ID); !ok {
		t.Fatal("in-flight idle callback reaped pending auth session")
	}
	if session.CurrentAuthChallenge().ResumeToken != challenge.ResumeToken {
		t.Fatal("idle callback altered pending challenge")
	}
}

func TestIdleCallbackRechecksRecentActivity(t *testing.T) {
	manager, session := idleExpiryFixture()
	session.mu.Lock()
	session.touch()
	session.mu.Unlock()
	manager.expireSession(session.ID)
	if _, ok := manager.Get(session.ID); !ok {
		t.Fatal("stale idle callback ignored recent activity")
	}
}

func TestIdleCallbackRechecksResolvedAuthLease(t *testing.T) {
	manager, session := idleExpiryFixture()
	challenge := session.raiseAuthChallenge("credential_form", "https://example.test/login")
	if err := session.ResolveAuthChallenge(challenge.ResumeToken); err != nil {
		t.Fatal(err)
	}
	manager.expireSession(session.ID)
	if _, ok := manager.Get(session.ID); !ok {
		t.Fatal("stale idle callback ignored freshly resolved lease")
	}
}

func TestIdleCallbackStillExpiresTrulyIdleSession(t *testing.T) {
	manager, session := idleExpiryFixture()
	manager.expireSession(session.ID)
	if _, ok := manager.Get(session.ID); ok {
		t.Fatal("expired idle session retained")
	}
}

func TestConcurrentIdleCallbacksAndExplicitClose(t *testing.T) {
	manager, session := idleExpiryFixture()
	session.raiseAuthChallenge("credential_form", "https://example.test/login")
	var workers sync.WaitGroup
	results := make(chan error, 16)
	start := make(chan struct{})
	for i := 0; i < 32; i++ {
		workers.Add(1)
		go func(expire bool) {
			defer workers.Done()
			<-start
			if expire {
				manager.expireSession(session.ID)
			} else {
				results <- manager.Close(session.ID)
			}
		}(i%2 == 0)
	}
	close(start)
	workers.Wait()
	close(results)
	winners := 0
	for err := range results {
		if err == nil {
			winners++
		} else if err != ErrSessionNotFound {
			t.Fatal(err)
		}
	}
	if winners != 1 {
		t.Fatalf("explicit close winners=%d want1", winners)
	}
	if _, ok := manager.Get(session.ID); ok {
		t.Fatal("concurrent close retained session")
	}
}

func TestExplicitCloseStillClosesPendingAuth(t *testing.T) {
	manager, session := idleExpiryFixture()
	session.raiseAuthChallenge("credential_form", "https://example.test/login")
	if err := manager.Close(session.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := manager.Get(session.ID); ok {
		t.Fatal("idle guard blocked explicit close")
	}
	if err := manager.Close(session.ID); err != ErrSessionNotFound {
		t.Fatalf("missing-session contract changed: %v", err)
	}
}
