// Package watcher — tests des garde-fous défensifs de PlayerWatcher.
//
// Ces tests vérifient que le watcher ne panique pas dans les cas de
// configuration dégradée (fetcher nil, pool absent, etc.). Régressions
// connues : incident mai 2026 "MatchPoller nil pointer dereference" qui
// a provoqué une boucle de redémarrage du serveur Go.
package watcher

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

// Garde-fou anti-régression : si le MatchFetcher n'est pas configuré (mode
// dégradé, pool absent), startPoller doit logger un Warn-once et ne PAS
// paniquer. Référence : commit b144c246 a introduit le MatchPoller sans
// brancher d'implémentation prod ; le crash a tourné en boucle pendant
// plusieurs heures avant fix.
func TestPlayerWatcher_StartPollerNilFetcher_DoesNotPanic(t *testing.T) {
	trigger := newMockSyncTrigger()
	// fetcher=nil simule la régression
	pw := NewPlayerWatcher("player1", "xuid1", nil, trigger)

	// Capture les logs slog dans un buffer pour vérifier le Warn
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})
	prev := slog.Default()
	slog.SetDefault(slog.New(handler))
	defer slog.SetDefault(prev)

	ctx := context.Background()

	// Ne doit pas paniquer
	pw.OnPresenceActive(ctx)

	if pw.fsm.State() != StateWatching {
		t.Errorf("state = %v, want Watching (FSM doit transiter même sans fetcher)", pw.fsm.State())
	}
	if !strings.Contains(buf.String(), "pas de MatchFetcher configuré") {
		t.Errorf("log Warn manquant : %q", buf.String())
	}

	// 2e appel direct à startPoller : pas de nouveau Warn (sync.Once)
	buf.Reset()
	pw.startPoller(ctx)
	if strings.Contains(buf.String(), "pas de MatchFetcher configuré") {
		t.Errorf("Warn re-loggué au 2e startPoller (sync.Once cassé) : %q", buf.String())
	}
}
