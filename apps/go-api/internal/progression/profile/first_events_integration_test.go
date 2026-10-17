//go:build integration

// first_events_integration_test.go — computeFKFD sur une base partagée migrée : Infinite (kill/death
// de highlight_events), Halo 5 (repli sur les frags horodatés), Campagne exclue.

package profile

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/platform/duckdb"
)

// seedFKFDMatch pose un match du registre dans la fenêtre, avec le joueur du test (et un
// adversaire) comme participants. variantID vide = pas de variante.
func seedFKFDMatch(t *testing.T, pdb *duckdb.PlayerDB, matchID, variantID string, start time.Time) {
	t.Helper()
	ctx := context.Background()
	if _, err := pdb.Shared.Exec(ctx,
		`INSERT INTO match_registry (match_id, start_time, game_variant_id) VALUES (?, ?, NULLIF(?, ''))`,
		matchID, start, variantID); err != nil {
		t.Fatalf("match_registry %s: %v", matchID, err)
	}
	for _, xuid := range []string{testXUID, "xuid-adversaire"} {
		if _, err := pdb.Shared.Exec(ctx, `
			INSERT INTO match_participants (match_id, xuid, gamertag, team_id, outcome, kills, deaths, assists,
				personal_score, max_killing_spree, time_played_seconds)
			VALUES (?, ?, ?, 1, 2, 10, 8, 3, 1500, 5, 600)`, matchID, xuid, xuid); err != nil {
			t.Fatalf("match_participants %s/%s: %v", matchID, xuid, err)
		}
	}
}

func seedHighlight(t *testing.T, pdb *duckdb.PlayerDB, matchID, eventType, xuid string, ms int) {
	t.Helper()
	if _, err := pdb.Shared.Exec(context.Background(),
		`INSERT INTO highlight_events (id, match_id, event_type, time_ms, xuid)
		 VALUES ((SELECT COALESCE(MAX(id), 0) + 1 FROM highlight_events), ?, ?, ?, ?)`,
		matchID, eventType, ms, xuid); err != nil {
		t.Fatalf("highlight_events %s: %v", matchID, err)
	}
}

// TestComputeFKFD_Infinite : le joueur ouvre un match (m1), meurt en premier dans un autre (m2) ;
// m3 est ouvert par l'adversaire ; un match hors fenêtre et un match sans le joueur ne comptent pas.
func TestComputeFKFD_Infinite(t *testing.T) {
	pdb := setupProfileEnv(t)
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	for i, id := range []string{"m1", "m2", "m3", "m_hors_fenetre"} {
		start := now.AddDate(0, 0, -i-1)
		if id == "m_hors_fenetre" {
			start = now.AddDate(0, -6, 0)
		}
		seedFKFDMatch(t, pdb, id, "", start)
	}
	adv := "xuid-adversaire"
	// m1 : premier frag et première mort de l'adversaire opposés : je frague en premier.
	seedHighlight(t, pdb, "m1", "kill", testXUID, 1000)
	seedHighlight(t, pdb, "m1", "death", adv, 1000)
	seedHighlight(t, pdb, "m1", "kill", adv, 4000)
	seedHighlight(t, pdb, "m1", "death", testXUID, 4000)
	// m2 : je meurs en premier ; l'adversaire frague en premier.
	seedHighlight(t, pdb, "m2", "kill", adv, 500)
	seedHighlight(t, pdb, "m2", "death", testXUID, 500)
	// m3 : l'adversaire frague, je ne meurs qu'après un autre joueur.
	seedHighlight(t, pdb, "m3", "kill", adv, 700)
	seedHighlight(t, pdb, "m3", "death", "xuid-tiers", 700)
	seedHighlight(t, pdb, "m3", "death", testXUID, 9000)
	// hors fenêtre : ne doit rien ajouter.
	seedHighlight(t, pdb, "m_hors_fenetre", "kill", testXUID, 100)
	seedHighlight(t, pdb, "m_hors_fenetre", "death", testXUID, 100)

	svc := NewServiceFromPlayerDB(pdb)
	fk, fd, err := svc.computeFKFD(context.Background(), testXUID, now.AddDate(0, 0, -30), now)
	if err != nil {
		t.Fatalf("computeFKFD: %v", err)
	}
	if fk != 1 || fd != 1 {
		t.Fatalf("premiers frags / premieres morts = %d / %d, attendu 1 / 1", fk, fd)
	}
}

// TestComputeFKFD_Halo5RepliSurLesFrags : Halo 5 n'écrit ni kill ni death dans highlight_events ;
// les frags horodatés viennent du journal canonique des morts (repli du chargeur). La Campagne,
// masquée par la fenêtre de matchs, ne compte pas.
func TestComputeFKFD_Halo5RepliSurLesFrags(t *testing.T) {
	pdb := setupProfileEnv(t)
	pdb.TitleSlug = "halo_5"
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	campaign := analysis.CampaignExcludedVariantIDs("halo_5")[0]
	seedFKFDMatch(t, pdb, "h1", "", now.AddDate(0, 0, -1))
	seedFKFDMatch(t, pdb, "h2", "", now.AddDate(0, 0, -2))
	seedFKFDMatch(t, pdb, "h_campagne", campaign, now.AddDate(0, 0, -3))
	adv := "xuid-adversaire"
	ctx := context.Background()
	for _, k := range []struct {
		match, killer, victim string
		ms                    int
	}{
		{"h1", testXUID, adv, 2000}, {"h1", adv, testXUID, 3000}, // h1 : j'ouvre le match, je ne suis pas le premier mort
		{"h2", adv, testXUID, 1500}, // h2 : l'adversaire ouvre, je meurs en premier
		{"h_campagne", testXUID, adv, 100},
	} {
		if _, err := pdb.Shared.Exec(ctx, `
			INSERT INTO match_kill_events (match_id, decode_pass, decoder_rev, publishable, time_ms,
				victim_gamertag, victim_xuid, feed_killer_gamertag, feed_killer_xuid, feed_present,
				assist_known, assist_extra_count, read_path, read_origin)
			VALUES (?, ?, 'test', TRUE, ?, ?, ?, ?, ?, TRUE, FALSE, 0, 'kill-feed', 'credit-seul')`,
			k.match, "pairs-"+k.match, k.ms, k.victim, k.victim, k.killer, k.killer); err != nil {
			t.Fatalf("match_kill_events %s: %v", k.match, err)
		}
	}

	svc := NewServiceFromPlayerDB(pdb)
	fk, fd, err := svc.computeFKFD(ctx, testXUID, now.AddDate(0, 0, -30), now)
	if err != nil {
		t.Fatalf("computeFKFD: %v", err)
	}
	if fk != 1 || fd != 1 {
		t.Fatalf("premiers frags / premieres morts = %d / %d, attendu 1 / 1 (h1 ouvert par le joueur, h2 premiere mort ; campagne exclue)", fk, fd)
	}
}
