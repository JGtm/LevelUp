//go:build integration

// Package duckdb — squad_repo_kv_fallback_test.go : tests du fallback
// title-agnostic de synthèse d'events kill/death depuis killer_victim_pairs
// dans SquadRepo.LoadImpactEvents + lecture batch LoadKVPairs.
//
// Cause racine couverte : en Halo 5, highlight_events ne porte QUE des médailles
// (les kills horodatés vivent dans killer_victim_pairs). Sans fallback, les
// builders Escouade (first events, intensité, matrice d'impact) restent vides.
package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/analysis"
)

// recreateKVPairsWithTimeMS recrée shared.killer_victim_pairs + sa vue root-level
// avec la colonne time_ms (absente du seed par défaut, présente en prod). Le
// fallback LoadImpactEvents/LoadKVPairs lit time_ms (Q32c).
//
// Recrée sur Player ET Shared : la lecture passe par SharedReader =
// LegacySharedReader(player) (donc le schéma `shared` de la player DB), tandis
// que execOnSharedDBs insère dans les deux DBs — les deux schémas doivent donc
// porter la colonne time_ms.
func recreateKVPairsWithTimeMS(t *testing.T, pdb *PlayerDB, ctx context.Context) {
	t.Helper()
	ddl := []string{
		`DROP VIEW IF EXISTS killer_victim_pairs`,
		`DROP TABLE IF EXISTS shared.killer_victim_pairs`,
		`CREATE TABLE shared.killer_victim_pairs (
			match_id VARCHAR NOT NULL, killer_xuid VARCHAR NOT NULL,
			killer_gamertag VARCHAR, victim_xuid VARCHAR NOT NULL,
			victim_gamertag VARCHAR, kill_count INTEGER DEFAULT 1,
			time_ms BIGINT)`,
		`CREATE VIEW killer_victim_pairs AS SELECT * FROM shared.killer_victim_pairs`,
	}
	for _, db := range []*DB{pdb.Player, pdb.Shared} {
		for _, q := range ddl {
			if _, err := db.Exec(ctx, q); err != nil {
				t.Fatalf("recreateKVPairsWithTimeMS %q: %v", q, err)
			}
		}
	}
}

// TestSquadRepo_LoadImpactEvents_KVFallback_Synthesized : highlight_events ne
// contient qu'une médaille (medals-only, H5) + killer_victim_pairs peuplé →
// LoadImpactEvents doit synthétiser un kill (acteur=tueur) + un death
// (acteur=victime) et les fusionner, triés par TimeMS.
func TestSquadRepo_LoadImpactEvents_KVFallback_Synthesized(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	recreateKVPairsWithTimeMS(t, pdb, ctx)

	// highlight_events medals-only (aucun kill/death) → déclenche le fallback.
	execOnSharedDBs(t, pdb, ctx,
		`INSERT INTO shared.highlight_events (match_id, xuid, event_type, time_ms)
		 VALUES (?, ?, ?, ?)`,
		"m1", pTestXUID, "medal", 5000)
	// killer_victim_pairs : 2 kills horodatés sur m1.
	execOnSharedDBs(t, pdb, ctx,
		`INSERT INTO shared.killer_victim_pairs (match_id, killer_xuid, victim_xuid, kill_count, time_ms)
		 VALUES (?, ?, ?, ?, ?)`,
		"m1", pTestXUID, "victimA", 1, 3000)
	execOnSharedDBs(t, pdb, ctx,
		`INSERT INTO shared.killer_victim_pairs (match_id, killer_xuid, victim_xuid, kill_count, time_ms)
		 VALUES (?, ?, ?, ?, ?)`,
		"m1", "killerB", pTestXUID, 1, 8000)

	repo := NewSquadRepo(pdb)
	got, err := repo.LoadImpactEvents(ctx, []string{"m1"})
	if err != nil {
		t.Fatalf("LoadImpactEvents: %v", err)
	}

	// Attendu : 1 médaille + 2 paires × (kill + death) = 5 events.
	if len(got) != 5 {
		t.Fatalf("attendu 5 events (1 medal + 2 kills + 2 deaths), obtenu %d : %#v", len(got), got)
	}

	// Compter kills/deaths synthétisés + vérifier les acteurs.
	var kills, deaths, medals int
	killByTime := map[int64]string{}
	deathByTime := map[int64]string{}
	for _, e := range got {
		switch e.EventType {
		case analysis.EventTypeKill:
			kills++
			killByTime[e.TimeMS] = e.XUID
		case analysis.EventTypeDeath:
			deaths++
			deathByTime[e.TimeMS] = e.XUID
		case "medal":
			medals++
		}
	}
	if kills != 2 || deaths != 2 || medals != 1 {
		t.Fatalf("kills=%d deaths=%d medals=%d, want 2/2/1", kills, deaths, medals)
	}
	// kill@3000 : acteur = tueur pTestXUID ; death@8000 : acteur = victime pTestXUID.
	if killByTime[3000] != pTestXUID {
		t.Errorf("kill@3000 acteur = %q, want %q (tueur)", killByTime[3000], pTestXUID)
	}
	if deathByTime[8000] != pTestXUID {
		t.Errorf("death@8000 acteur = %q, want %q (victime)", deathByTime[8000], pTestXUID)
	}

	// Tri global par TimeMS croissant après merge.
	for i := 1; i < len(got); i++ {
		if got[i-1].TimeMS > got[i].TimeMS {
			t.Errorf("events non triés par TimeMS: [%d]=%d > [%d]=%d", i-1, got[i-1].TimeMS, i, got[i].TimeMS)
		}
	}
}

// TestSquadRepo_LoadImpactEvents_NativeKills_NoFallback : highlight_events porte
// déjà un kill natif (cas Infinite) → le fallback kvPairs ne doit JAMAIS
// s'appliquer (pas de double-comptage), même si killer_victim_pairs est peuplé.
func TestSquadRepo_LoadImpactEvents_NativeKills_NoFallback(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	recreateKVPairsWithTimeMS(t, pdb, ctx)

	// highlight_events porte un kill natif → HasCanonicalKillOrDeath == true.
	execOnSharedDBs(t, pdb, ctx,
		`INSERT INTO shared.highlight_events (match_id, xuid, event_type, time_ms)
		 VALUES (?, ?, ?, ?)`,
		"m1", pTestXUID, "kill", 4000)
	// killer_victim_pairs peuplé : si le fallback se déclenchait à tort, on
	// verrait des events synthétiques supplémentaires.
	execOnSharedDBs(t, pdb, ctx,
		`INSERT INTO shared.killer_victim_pairs (match_id, killer_xuid, victim_xuid, kill_count, time_ms)
		 VALUES (?, ?, ?, ?, ?)`,
		"m1", pTestXUID, "victimA", 1, 4000)

	repo := NewSquadRepo(pdb)
	got, err := repo.LoadImpactEvents(ctx, []string{"m1"})
	if err != nil {
		t.Fatalf("LoadImpactEvents: %v", err)
	}
	// Exactement la row native, aucune synthèse.
	if len(got) != 1 {
		t.Fatalf("NO-OP Infinite attendu (1 event natif), obtenu %d : %#v", len(got), got)
	}
	if got[0].EventType != analysis.EventTypeKill {
		t.Errorf("event = %q, want kill", got[0].EventType)
	}
}

// TestSquadRepo_LoadKVPairs_Batch : LoadKVPairs lit en batch et peuple MatchID.
func TestSquadRepo_LoadKVPairs_Batch(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	recreateKVPairsWithTimeMS(t, pdb, ctx)

	execOnSharedDBs(t, pdb, ctx,
		`INSERT INTO shared.killer_victim_pairs (match_id, killer_xuid, victim_xuid, kill_count, time_ms)
		 VALUES (?, ?, ?, ?, ?)`,
		"m1", "k1", "v1", 2, 1000)
	execOnSharedDBs(t, pdb, ctx,
		`INSERT INTO shared.killer_victim_pairs (match_id, killer_xuid, victim_xuid, kill_count, time_ms)
		 VALUES (?, ?, ?, ?, ?)`,
		"m2", "k2", "v2", 1, 2000)

	repo := NewSquadRepo(pdb)

	// Empty input → nil sans erreur.
	if got, err := repo.LoadKVPairs(ctx, nil); err != nil || got != nil {
		t.Fatalf("LoadKVPairs(nil) = (%v, %v), want (nil, nil)", got, err)
	}

	got, err := repo.LoadKVPairs(ctx, []string{"m1", "m2"})
	if err != nil {
		t.Fatalf("LoadKVPairs: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("attendu 2 paires, obtenu %d", len(got))
	}
	byMatch := map[string]bool{}
	for _, kv := range got {
		byMatch[kv.MatchID] = true
		if kv.KillerXUID == "" || kv.VictimXUID == "" {
			t.Errorf("paire incomplète: %#v", kv)
		}
	}
	if !byMatch["m1"] || !byMatch["m2"] {
		t.Errorf("MatchID non peuplé correctement: %#v", got)
	}
}
