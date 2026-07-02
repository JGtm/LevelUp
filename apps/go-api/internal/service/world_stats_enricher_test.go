package service

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"levelup/go-api/internal/domain"
)

// countingResolver compte les appels à ResolveXUID pour prouver le court-circuit
// PeopleHub quand un xuid est pré-seedé (B1).
type countingResolver struct {
	m     map[string]string
	calls int64
}

func (r *countingResolver) ResolveXUID(_ context.Context, gamertag string) (string, error) {
	atomic.AddInt64(&r.calls, 1)
	x, ok := r.m[gamertag]
	if !ok {
		return "", fmt.Errorf("xuid introuvable pour %s", gamertag)
	}
	return x, nil
}

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

	// Demande la saison 13-2 (chemin brut) → 12-1 doit être exclu. Neo est fourni SANS
	// xuid → l'enricher retombe sur le résolveur (chemin fallback).
	stats, errs := enr.EnrichSeason(context.Background(), "Csr/Seasons/CsrSeason13-2.json", []domain.WorldPlayerRef{{Gamertag: "Neo"}})
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

	// Aucun joueur → no-op silencieux.
	if out, e := enr.EnrichSeason(context.Background(), "csrseason13-2", nil); out != nil || e != nil {
		t.Errorf("EnrichSeason(nil) = (%v, %v), want (nil, nil)", out, e)
	}
}

// TestEnrichSeason_SeededXUIDSkipsResolver prouve le cœur de B1 : quand le xuid est
// fourni (scrapé du snapshot Waypoint), l'enricher NE re-résout PAS via PeopleHub.
func TestEnrichSeason_SeededXUIDSkipsResolver(t *testing.T) {
	const xuid = "42"
	src := &fakeMatchSource{
		history: map[string][]string{"xuid(" + xuid + ")": {"m1"}},
		stats: map[string]map[string]any{
			"m1": buildMatch(xuid, "Csr/Seasons/CsrSeason13-2.json", tArena, 2, 10, 5, 3),
		},
	}
	res := &countingResolver{m: map[string]string{"Neo": xuid}}
	enr := NewWorldStatsEnricher(src, res, WorldStatsAggregatorConfig{Concurrency: 2})

	// Neo fourni AVEC son xuid → le résolveur ne doit jamais être appelé.
	stats, errs := enr.EnrichSeason(context.Background(), "csrseason13-2",
		[]domain.WorldPlayerRef{{Gamertag: "Neo", XUID: xuid}})
	if len(errs) != 0 {
		t.Fatalf("erreurs inattendues: %v", errs)
	}
	if len(stats) != 1 || stats[0].Kills != 10 {
		t.Fatalf("bucket attendu (13-2/arena, 10 kills), got %+v", stats)
	}
	if n := atomic.LoadInt64(&res.calls); n != 0 {
		t.Errorf("ResolveXUID appelé %d fois, attendu 0 (xuid pré-seedé, B1)", n)
	}
}
