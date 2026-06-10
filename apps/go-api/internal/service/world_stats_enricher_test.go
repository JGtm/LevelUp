package service

import (
	"context"
	"testing"
)

// TestEnrichSeason_TargetsRequestedSeason vérifie que l'enricher cible bien la
// saison demandée (normalisée depuis un chemin) à chaque appel, sans fuite entre
// saisons : un match d'une autre saison est exclu même si le joueur l'a joué.
func TestEnrichSeason_TargetsRequestedSeason(t *testing.T) {
	const xuid = "42"
	src := &fakeMatchSource{
		history: map[string][]string{"xuid(" + xuid + ")": {"m1", "m2"}},
		stats: map[string]map[string]any{
			"m1": buildMatch(xuid, "Csr/Seasons/CsrSeason13-2.json", tArena, 2, 10, 5, 3),
			"m2": buildMatch(xuid, "Csr/Seasons/CsrSeason12-1.json", tArena, 2, 99, 1, 1),
		},
	}
	enr := NewWorldStatsEnricher(src, &fakeResolver{m: map[string]string{"Neo": xuid}},
		WorldStatsAggregatorConfig{Concurrency: 2})

	// Demande la saison 13-2 (chemin brut) → 12-1 doit être exclu.
	stats, errs := enr.EnrichSeason(context.Background(), "Csr/Seasons/CsrSeason13-2.json", []string{"Neo"})
	if len(errs) != 0 {
		t.Fatalf("erreurs inattendues: %v", errs)
	}
	if len(stats) != 1 {
		t.Fatalf("attendu 1 bucket (13-2/arena), got %d : %+v", len(stats), stats)
	}
	s := stats[0]
	if s.SeasonID != "csrseason13-2" || s.MatchCount != 1 || s.Kills != 10 {
		t.Errorf("bucket = %+v, want season csrseason13-2 / 1 match / 10 kills", s)
	}

	// Aucun gamertag → no-op silencieux.
	if out, e := enr.EnrichSeason(context.Background(), "csrseason13-2", nil); out != nil || e != nil {
		t.Errorf("EnrichSeason(nil) = (%v, %v), want (nil, nil)", out, e)
	}
}
