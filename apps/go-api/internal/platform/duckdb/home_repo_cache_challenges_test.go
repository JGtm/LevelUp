// Package duckdb — home_repo_cache_challenges_test.go : garde-fou C4.
//
// Régression directe du bug « Défis indisponibles » : le cache DOIT reconstruire de
// vraies cartes (Items) depuis les snapshots porteurs d'un titre rendu, sinon le
// frontend retombe sur l'état indisponible alors qu'un cache frais existe.
package duckdb

import (
	"database/sql"
	"testing"
	"time"
)

func chNullStr(s string) sql.NullString { return sql.NullString{String: s, Valid: s != ""} }
func chNullI64(v int64) sql.NullInt64   { return sql.NullInt64{Int64: v, Valid: true} }

func TestBuildChallengesResponseFromSnapshots_ReconstructsItems(t *testing.T) {
	now := time.Now().UTC()
	snapshots := []challengeSnapshotRow{
		{ // actif rendu (titre + progression) → carte
			challengePath: "Challenges/Tracking/a", status: "Active", xpReward: 500,
			progressCurrent: chNullI64(3), progressTarget: chNullI64(10),
			snapshotAt: now, title: chNullStr("Tuer 10 Spartans"), imageURL: chNullStr("/img/a.png"),
		},
		{ // actif rendu, sans image
			challengePath: "Challenges/Tracking/b", status: "Active", xpReward: 250,
			progressCurrent: chNullI64(1), progressTarget: chNullI64(4),
			snapshotAt: now, title: chNullStr("Gagner 4 parties"),
		},
		{ // actif SANS titre (legacy / pas encore re-snapshotté) → omis des cartes, mais compté
			challengePath: "Challenges/Tracking/c", status: "Active", xpReward: 100,
			snapshotAt: now,
		},
		{ // complété → compté, pas de carte
			challengePath: "Challenges/Tracking/d", status: "Completed",
			snapshotAt: now, title: chNullStr("Défi terminé"),
		},
	}

	resp := buildChallengesResponseFromSnapshots(snapshots)
	if resp == nil || !resp.Available || !resp.FromCache {
		t.Fatalf("réponse cache attendue Available+FromCache, got %+v", resp)
	}
	if resp.Total == nil || *resp.Total != 4 {
		t.Errorf("Total attendu 4 (tous snapshots), got %v", resp.Total)
	}
	if resp.Completed == nil || *resp.Completed != 1 {
		t.Errorf("Completed attendu 1, got %v", resp.Completed)
	}
	// Seuls les actifs AVEC titre deviennent des cartes (a, b) — pas c (sans titre), pas d (complété).
	if len(resp.Items) != 2 {
		t.Fatalf("Items attendus 2 (actifs rendus), got %d", len(resp.Items))
	}
	first := resp.Items[0]
	if first.Title != "Tuer 10 Spartans" {
		t.Errorf("titre carte attendu, got %q", first.Title)
	}
	if first.ProgressPercent == nil || *first.ProgressPercent != 30 {
		t.Errorf("ProgressPercent attendu 30 (3/10), got %v", first.ProgressPercent)
	}
	if first.ImageURL == nil || *first.ImageURL != "/img/a.png" {
		t.Errorf("ImageURL attendu, got %v", first.ImageURL)
	}
}

func TestBuildChallengesResponseFromSnapshots_AllUntitled_NoItemsButCounts(t *testing.T) {
	// Cache legacy (aucun titre) : pas de cartes, MAIS available+counts → le service
	// rend quand même (parité Battle Pass), jamais « indisponible ».
	snapshots := []challengeSnapshotRow{
		{challengePath: "Challenges/Tracking/x", status: "Active", xpReward: 100, snapshotAt: time.Now().UTC()},
		{challengePath: "Challenges/Tracking/y", status: "Active", xpReward: 100, snapshotAt: time.Now().UTC()},
	}
	resp := buildChallengesResponseFromSnapshots(snapshots)
	if !resp.Available || !resp.FromCache {
		t.Error("doit rester Available+FromCache même sans titres")
	}
	if len(resp.Items) != 0 {
		t.Errorf("aucune carte attendue sans titre, got %d", len(resp.Items))
	}
	if resp.Total == nil || *resp.Total != 2 {
		t.Errorf("Total attendu 2, got %v", resp.Total)
	}
}
