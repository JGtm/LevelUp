package main

// sync_report.go — compte rendu d'une passe de sync, source UNIQUE du verdict CLI.
//
// Avant le 2026-09-16, les quatre runners (`sync-delta`, `sync-full`, et leurs variantes
// `--all`) recopiaient chacun leur `fmt.Printf("sync ... OK: ...")` et rendaient `nil`
// quoi qu'il arrive : une passe arrêtée par un 429 sortait en code 0 sous le mot « OK »
// alors qu'elle n'avait rien inséré. Le verdict se lit désormais dans `SyncResult.Status()`
// — un seul endroit, testable sans base ni réseau (D1, plan robustesse 2026-09-16).

import (
	"fmt"
	"io"
	"strings"

	"levelup/go-api/internal/domain"
)

// reportSyncResult écrit le compte rendu d'une passe et rend une erreur quand le statut
// n'est pas `success`. Fonction PURE : ni base, ni réseau, ni horloge.
//
// `mode` nomme la passe (« delta », « full ») ; `gamertag` reste dans la ligne parce que
// les variantes `--all` en émettent une par joueur (le plan écrivait trois paramètres, le
// quatrième évite de perdre l'identité du joueur dans le lot).
func reportSyncResult(w io.Writer, mode, gamertag string, r *domain.SyncResult) error {
	if r == nil {
		return fmt.Errorf("sync %s: aucun résultat pour %s", mode, gamertag)
	}

	status := r.Status()
	postSync := r.PostSync != nil
	careerSynced := postSync && r.PostSync.CareerSynced

	var b strings.Builder
	fmt.Fprintf(&b,
		"sync %s %s: gamertag=%s inserted=%d skipped=%d status=%s post_sync=%t career_synced=%t duration=%.2fs\n",
		mode, strings.ToUpper(status), gamertag,
		r.MatchesInserted, r.MatchesSkipped, status, postSync, careerSynced, r.DurationSeconds,
	)
	if len(r.Warnings) > 0 {
		fmt.Fprintf(&b, "  warnings=%d first=%s\n", len(r.Warnings), r.Warnings[0])
	}
	if len(r.Errors) > 0 {
		fmt.Fprintf(&b, "  errors=%d first_error=%s\n", len(r.Errors), r.Errors[0])
	}
	if _, err := io.WriteString(w, b.String()); err != nil {
		return fmt.Errorf("sync %s: écriture du compte rendu: %w", mode, err)
	}

	// Status() == "success" si et seulement si Errors est vide : la première erreur
	// existe toujours dans les deux autres cas (partial_success, failure).
	if status != "success" {
		return fmt.Errorf("sync %s %s: gamertag=%s première erreur: %s", mode, status, gamertag, r.Errors[0])
	}
	return nil
}
