// Package scheduler — auto_sync_notify_test.go : fusion pure des joueurs à
// notifier en fin de cycle auto-sync (notification Discord « nouveaux matchs »).
package scheduler

import (
	"testing"
	"time"

	"levelup/go-api/internal/notify"
)

func TestMergeCycleNewMatchPlayers(t *testing.T) {
	cycleStart := time.Date(2026, 7, 25, 10, 0, 0, 0, time.UTC)
	before := cycleStart.Add(-time.Hour)
	during := cycleStart.Add(time.Minute)

	t.Run("fusionne moteur (V2) + live/filet (snapshot) et somme les matchs", func(t *testing.T) {
		engine := []notify.PlayerSyncResult{{Gamertag: "MoteurInfinite", MatchesSynced: 3}}
		snapshot := []PlayerOutcomeDetail{
			{Gamertag: "LiveHalo5", Outcome: "ok", MatchesInserted: 2, AttemptedAt: during},
		}
		got := mergeCycleNewMatchPlayers(cycleStart, engine, snapshot)
		if len(got) != 2 {
			t.Fatalf("attendu 2 joueurs, obtenu %d (%+v)", len(got), got)
		}
		total := 0
		for _, p := range got {
			total += p.MatchesSynced
		}
		if total != 5 {
			t.Fatalf("total matchs = %d, attendu 5", total)
		}
	})

	t.Run("exclut les joueurs sans nouveau match, non-ok, ou d'un cycle antérieur", func(t *testing.T) {
		engine := []notify.PlayerSyncResult{
			{Gamertag: "AvecMatchs", MatchesSynced: 4},
			{Gamertag: "SansMatch", MatchesSynced: 0}, // 0 insert → exclu
		}
		snapshot := []PlayerOutcomeDetail{
			{Gamertag: "Echec", Outcome: "failed", MatchesInserted: 5, AttemptedAt: during},  // non-ok → exclu
			{Gamertag: "Ancien", Outcome: "ok", MatchesInserted: 9, AttemptedAt: before},     // avant le cycle → exclu
			{Gamertag: "ZeroInsert", Outcome: "ok", MatchesInserted: 0, AttemptedAt: during}, // 0 insert → exclu
			{Gamertag: "LiveOK", Outcome: "ok", MatchesInserted: 1, AttemptedAt: cycleStart}, // pile au début → inclus
		}
		got := mergeCycleNewMatchPlayers(cycleStart, engine, snapshot)
		if len(got) != 2 {
			t.Fatalf("attendu 2 joueurs (AvecMatchs + LiveOK), obtenu %d (%+v)", len(got), got)
		}
		names := map[string]int{}
		for _, p := range got {
			names[p.Gamertag] = p.MatchesSynced
		}
		if names["AvecMatchs"] != 4 || names["LiveOK"] != 1 {
			t.Fatalf("sélection inattendue : %+v", names)
		}
	})

	// Régression contre-revue V7.2 : l'ancienne implémentation DÉDUPLIQUAIT par
	// gamertag et jetait les occurrences suivantes — le 2e titre d'un même joueur
	// disparaissait du compte de la notification Discord.
	t.Run("même gamertag sur 2 titres moteur : comptes SOMMÉS, 1 seule entrée", func(t *testing.T) {
		engine := []notify.PlayerSyncResult{
			{Gamertag: "MultiTitre", MatchesSynced: 3}, // titre A
			{Gamertag: "MultiTitre", MatchesSynced: 4}, // titre B — ne doit PAS être jeté
		}
		got := mergeCycleNewMatchPlayers(cycleStart, engine, nil)
		if len(got) != 1 {
			t.Fatalf("1 entrée par joueur attendue, obtenu %d (%+v)", len(got), got)
		}
		if got[0].MatchesSynced != 7 {
			t.Fatalf("comptes des 2 titres attendus sommés (7), obtenu %d", got[0].MatchesSynced)
		}
	})

	t.Run("même gamertag moteur + live-only : comptes SOMMÉS, 1 seule entrée", func(t *testing.T) {
		engine := []notify.PlayerSyncResult{{Gamertag: "MoteurEtLive", MatchesSynced: 3}}
		snapshot := []PlayerOutcomeDetail{
			// Titre live-only (Halo 5) du MÊME joueur : ensemble disjoint du moteur
			// (auto_sync_run.go orchPlayers vs livePlayers) → aucun double comptage.
			{Gamertag: "MoteurEtLive", Outcome: "ok", MatchesInserted: 2, AttemptedAt: during},
		}
		got := mergeCycleNewMatchPlayers(cycleStart, engine, snapshot)
		if len(got) != 1 {
			t.Fatalf("1 entrée par joueur attendue, obtenu %d (%+v)", len(got), got)
		}
		if got[0].MatchesSynced != 5 {
			t.Fatalf("moteur (3) + live (2) = 5 attendu, obtenu %d", got[0].MatchesSynced)
		}
	})

	t.Run("aucun nouveau match : liste vide", func(t *testing.T) {
		got := mergeCycleNewMatchPlayers(cycleStart, nil, []PlayerOutcomeDetail{
			{Gamertag: "Idle", Outcome: "ok", MatchesInserted: 0, AttemptedAt: during},
		})
		if len(got) != 0 {
			t.Fatalf("liste vide attendue, obtenu %+v", got)
		}
	})
}
