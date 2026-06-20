package livesync

import (
	"context"
	"errors"
	"testing"
)

type fakeResolver struct {
	m   map[string]string
	err error
}

func (f fakeResolver) ResolveXUID(_ context.Context, gt string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	if x, ok := f.m[gt]; ok {
		return x, nil
	}
	return "", errors.New("gamertag inconnu")
}

func TestResolveXUIDClosure_Resolves(t *testing.T) {
	rx := ResolveXUIDClosure(context.Background(), fakeResolver{m: map[string]string{"JGtm": "xJG"}}, nil)
	if got := rx("JGtm"); got != "xJG" {
		t.Errorf("JGtm → %q, want xJG", got)
	}
	// gamertag inconnu → "" (jamais fabriqué).
	if got := rx("Inconnu"); got != "" {
		t.Errorf("inconnu → %q, want \"\"", got)
	}
}

func TestResolveXUIDClosure_NilResolver(t *testing.T) {
	if rx := ResolveXUIDClosure(context.Background(), nil, nil); rx("anything") != "" {
		t.Error("resolver nil → tout \"\"")
	}
}

func TestResolveXUIDClosure_ErrorSwallowed(t *testing.T) {
	rx := ResolveXUIDClosure(context.Background(), fakeResolver{err: errors.New("429")}, nil)
	if got := rx("JGtm"); got != "" {
		t.Errorf("erreur de résolution → \"\" (best-effort), got %q", got)
	}
}
