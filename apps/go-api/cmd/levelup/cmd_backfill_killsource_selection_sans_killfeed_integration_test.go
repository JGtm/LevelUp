//go:build integration

package main

// cmd_backfill_killsource_selection_sans_killfeed_integration_test.go — LE FILM LU SANS KILL DANS
// LA SELECTION DU RATTRAPAGE (2026-10-02).
//
// Un film complet decode sans aucun kill n ecrit aucune ligne de `match_kill_events` : seule
// `match_registry.killsource_sans_killfeed_rev` dit qu il est a jour, et pour SA revision
// seulement. Sans elle, `backfill-killsource` (hors ligne et `--online`) le redecodait a chaque
// passe, comme l etape 1.57 a chaque cycle.

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
)

func TestMatchsAJour_FilmLuSansKillALaRevisionCourante(t *testing.T) {
	db := baseDeSelection(t)
	quand := time.Date(2026, 10, 2, 7, 34, 0, 0, time.UTC)
	for id, rev := range map[string]any{
		"muet-courant": decfilm.Rev,
		"muet-ancien":  "killsource-1999-01-01",
		"jamais-lu":    nil,
	} {
		executer(t, db, `INSERT INTO match_registry (match_id, start_time, start_time_utc,
			killsource_sans_killfeed_rev) VALUES (?, ?, ?, ?)`, id, quand, quand, rev)
	}

	aJour, err := matchsAJour(context.Background(), db)
	if err != nil {
		t.Fatalf("matchsAJour: %v", err)
	}
	if !aJour["muet-courant"] {
		t.Error("muet-courant : film lu sans kill a la revision courante, encore candidat — " +
			"il serait redecode a chaque passe")
	}
	for _, id := range []string{"muet-ancien", "jamais-lu"} {
		if aJour[id] {
			t.Errorf("%s : tenu pour a jour — une autre revision (ou aucune) doit le relire", id)
		}
	}
}
