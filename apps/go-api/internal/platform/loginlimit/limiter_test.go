package loginlimit

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// fakeClock est une horloge avancée à la main.
type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time          { return c.t }
func (c *fakeClock) advance(d time.Duration) { c.t = c.t.Add(d) }

func newTestLimiter() (*Limiter, *fakeClock) {
	c := &fakeClock{t: time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)}
	return newWithClock(c.now), c
}

func fail(l *Limiter, n int, identifier, ip string) {
	for range n {
		l.RecordFailure(identifier, ip)
	}
}

func TestLoginLimiter_BlocksAtMaxFailures(t *testing.T) {
	l, _ := newTestLimiter()
	fail(l, MaxFailures-1, "alice", "")
	if _, blocked := l.Blocked("alice", ""); blocked {
		t.Fatalf("%d échecs : bloqué trop tôt", MaxFailures-1)
	}
	l.RecordFailure("alice", "")
	wait, blocked := l.Blocked("alice", "")
	if !blocked || wait != Lockout {
		t.Fatalf("%d échecs : blocked=%v wait=%v, want true %v", MaxFailures, blocked, wait, Lockout)
	}
}

func TestLoginLimiter_IdentifierIsNormalized(t *testing.T) {
	l, _ := newTestLimiter()
	for _, id := range []string{"Bob", " bob", "BOB ", "bob", "bOb"} {
		l.RecordFailure(id, "")
	}
	if _, blocked := l.Blocked("  BoB  ", ""); !blocked {
		t.Fatal("variantes de casse et d'espaces d'un même identifiant : un seul compteur attendu")
	}
}

func TestLoginLimiter_IPCountsAcrossIdentifiers(t *testing.T) {
	l, _ := newTestLimiter()
	for i := range MaxFailures {
		l.RecordFailure(fmt.Sprintf("compte-%d", i), "203.0.113.1")
	}
	if _, blocked := l.Blocked("autre-compte", "203.0.113.1"); !blocked {
		t.Fatal("IP aux 5 échecs sur 5 comptes : bloquée pour tout identifiant")
	}
	if _, blocked := l.Blocked("autre-compte", "203.0.113.2"); blocked {
		t.Fatal("une autre IP ne doit pas être bloquée")
	}
}

func TestLoginLimiter_BlockExpiresAfterLockout(t *testing.T) {
	l, c := newTestLimiter()
	fail(l, MaxFailures, "alice", "203.0.113.1")
	c.advance(Lockout - time.Second)
	if wait, blocked := l.Blocked("alice", "203.0.113.1"); !blocked || wait != time.Second {
		t.Fatalf("1 s avant la fin : blocked=%v wait=%v, want true 1s", blocked, wait)
	}
	c.advance(time.Second)
	if _, blocked := l.Blocked("alice", "203.0.113.1"); blocked {
		t.Fatal("blocage échu : la clé doit être libérée")
	}
	fail(l, MaxFailures-1, "alice", "203.0.113.1")
	if _, blocked := l.Blocked("alice", "203.0.113.1"); blocked {
		t.Fatal("après un blocage échu, le compte d'échecs repart de zéro")
	}
}

func TestLoginLimiter_FailuresOutsideWindowDoNotCount(t *testing.T) {
	l, c := newTestLimiter()
	fail(l, MaxFailures-1, "alice", "")
	c.advance(Window + time.Second)
	l.RecordFailure("alice", "")
	if _, blocked := l.Blocked("alice", ""); blocked {
		t.Fatal("des échecs sortis de la fenêtre ne doivent plus compter")
	}
}

func TestLoginLimiter_SuccessResetsBothKeys(t *testing.T) {
	l, _ := newTestLimiter()
	fail(l, MaxFailures-1, "alice", "203.0.113.1")
	l.RecordSuccess("ALICE", "203.0.113.1")
	fail(l, MaxFailures-1, "alice", "203.0.113.1")
	if _, blocked := l.Blocked("alice", "203.0.113.1"); blocked {
		t.Fatal("un succès doit remettre l'identifiant et l'IP à zéro")
	}
}

func TestLoginLimiter_EmptyKeysAreIgnored(t *testing.T) {
	l, _ := newTestLimiter()
	fail(l, 2*MaxFailures, "", "")
	if _, blocked := l.Blocked("", ""); blocked {
		t.Fatal("des clés vides ne doivent jamais être bloquées")
	}
	if len(l.byIdentifier) != 0 || len(l.byIP) != 0 {
		t.Fatalf("des clés vides ne doivent rien mémoriser (%d, %d)", len(l.byIdentifier), len(l.byIP))
	}
}

func TestLoginLimiter_LongIdentifierIsTruncated(t *testing.T) {
	l, _ := newTestLimiter()
	long := strings.Repeat("é", 10*maxIdentifierRunes)
	l.RecordFailure(long, "")
	for key := range l.byIdentifier {
		if n := len([]rune(key)); n > maxIdentifierRunes {
			t.Fatalf("clé de %d runes, want <= %d", n, maxIdentifierRunes)
		}
	}
}

// TestLoginLimiter_TableIsBoundedAndKeepsBlockedKeys : un afflux d'identifiants
// distincts ne fait pas croître la table au-delà de maxKeys et n'évince pas une clé
// bloquée tant qu'il reste des clés non bloquées.
func TestLoginLimiter_TableIsBoundedAndKeepsBlockedKeys(t *testing.T) {
	l, c := newTestLimiter()
	fail(l, MaxFailures, "victime", "")
	for i := range maxKeys + 500 {
		c.advance(time.Millisecond)
		l.RecordFailure(fmt.Sprintf("flood-%d", i), "")
	}
	if n := len(l.byIdentifier); n > maxKeys {
		t.Fatalf("table de %d clés, want <= %d", n, maxKeys)
	}
	if _, blocked := l.Blocked("victime", ""); !blocked {
		t.Fatal("la clé bloquée a été évincée par l'afflux")
	}
}
