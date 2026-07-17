package service

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

// rivalNotifNow : horloge fixe des tests de détection rival croisé.
func rivalNotifNow() time.Time {
	return time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
}

// oneRival : une ligne relation qualifiant comme top rival (EnemyCount >= seuil
// momentsRivalMinEnemyMatches). XUID/Gamertag stables pour les assertions.
func oneRival() []domain.RelationRawRow {
	return []domain.RelationRawRow{
		{XUID: "rival-1", Gamertag: "Nemesis", TotalMatches: 12, EnemyCount: 10, EnemyWins: 3},
	}
}

// duelAt construit un duel brut (match en ennemi) à l'instant t.
func duelAt(matchID string, t time.Time, result, kills, deaths int) domain.RelationDuelRawRow {
	return domain.RelationDuelRawRow{
		MatchID: matchID, StartTime: t, Result: result, KillsOnRival: kills, DeathsByRival: deaths,
	}
}

func TestDetectRivalEncounters_NewDuelDetected(t *testing.T) {
	now := rivalNotifNow()
	watermark := now.Add(-2 * time.Hour) // dernier match connu avant la sync
	repo := &mockRelationsRepo{
		rows: oneRival(),
		timelineByXUID: map[string][]domain.RelationDuelRawRow{
			"rival-1": {
				duelAt("m-old", watermark.Add(-24*time.Hour), 2, 4, 9), // avant watermark
				duelAt("m-new", now.Add(-30*time.Minute), 1, 12, 5),    // après watermark, gagné
			},
		},
	}
	svc := NewRelationsService(repo).withNow(func() time.Time { return now })

	got, err := svc.DetectRivalEncounters(context.Background(), watermark)
	if err != nil {
		t.Fatalf("DetectRivalEncounters: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("attendu 1 duel détecté, obtenu %d", len(got))
	}
	e := got[0]
	if e.MatchID != "m-new" || e.Gamertag != "Nemesis" || e.Outcome != "win" || e.KillsOnRival != 12 || e.DeathsByRival != 5 {
		t.Fatalf("duel détecté inattendu: %+v", e)
	}
}

func TestDetectRivalEncounters_DuelBeforeWatermarkIgnored(t *testing.T) {
	now := rivalNotifNow()
	watermark := now.Add(-1 * time.Hour)
	repo := &mockRelationsRepo{
		rows: oneRival(),
		timelineByXUID: map[string][]domain.RelationDuelRawRow{
			// Uniquement des duels AU watermark ou avant → aucun « nouveau ».
			"rival-1": {
				duelAt("m-a", watermark, 1, 5, 5),                  // == watermark (non strict) → ignoré
				duelAt("m-b", watermark.Add(-3*time.Hour), 2, 2, 8), // avant → ignoré
			},
		},
	}
	svc := NewRelationsService(repo).withNow(func() time.Time { return now })

	got, err := svc.DetectRivalEncounters(context.Background(), watermark)
	if err != nil {
		t.Fatalf("DetectRivalEncounters: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("attendu 0 duel (tous <= watermark), obtenu %d", len(got))
	}
}

func TestDetectRivalEncounters_MaxAgeDaysFiltersOldBackfill(t *testing.T) {
	now := rivalNotifNow()
	// Watermark très ancien (backfill d'historique) : un duel postérieur au
	// watermark mais plus vieux que rivalNotifMaxAgeDays ne doit PAS être notifié.
	watermark := now.Add(-60 * 24 * time.Hour)
	repo := &mockRelationsRepo{
		rows: oneRival(),
		timelineByXUID: map[string][]domain.RelationDuelRawRow{
			"rival-1": {
				duelAt("m-stale", now.Add(-30*24*time.Hour), 1, 7, 3), // > watermark mais > 7 j → ignoré
				duelAt("m-fresh", now.Add(-2*time.Hour), 2, 3, 6),     // récent → détecté
			},
		},
	}
	svc := NewRelationsService(repo).withNow(func() time.Time { return now })

	got, err := svc.DetectRivalEncounters(context.Background(), watermark)
	if err != nil {
		t.Fatalf("DetectRivalEncounters: %v", err)
	}
	if len(got) != 1 || got[0].MatchID != "m-fresh" {
		t.Fatalf("attendu seulement m-fresh (m-stale filtré par maxAgeDays), obtenu %+v", got)
	}
}

func TestDetectRivalEncounters_CapMaxPerSyncKeepsMostRecent(t *testing.T) {
	now := rivalNotifNow()
	watermark := now.Add(-6 * time.Hour)
	// 5 duels frais postérieurs au watermark → plafonnés à rivalNotifMaxPerSync (3),
	// les 3 PLUS RÉCENTS conservés.
	duels := []domain.RelationDuelRawRow{
		duelAt("m1", now.Add(-5*time.Hour), 1, 1, 1),
		duelAt("m2", now.Add(-4*time.Hour), 1, 1, 1),
		duelAt("m3", now.Add(-3*time.Hour), 1, 1, 1),
		duelAt("m4", now.Add(-2*time.Hour), 1, 1, 1),
		duelAt("m5", now.Add(-1*time.Hour), 1, 1, 1),
	}
	repo := &mockRelationsRepo{
		rows:           oneRival(),
		timelineByXUID: map[string][]domain.RelationDuelRawRow{"rival-1": duels},
	}
	svc := NewRelationsService(repo).withNow(func() time.Time { return now })

	got, err := svc.DetectRivalEncounters(context.Background(), watermark)
	if err != nil {
		t.Fatalf("DetectRivalEncounters: %v", err)
	}
	if len(got) != rivalNotifMaxPerSync {
		t.Fatalf("attendu %d (plafond), obtenu %d", rivalNotifMaxPerSync, len(got))
	}
	if got[0].MatchID != "m5" || got[1].MatchID != "m4" || got[2].MatchID != "m3" {
		t.Fatalf("attendu les 3 plus récents (m5,m4,m3), obtenu %s,%s,%s", got[0].MatchID, got[1].MatchID, got[2].MatchID)
	}
}

func TestDetectRivalEncounters_ZeroWatermarkSkips(t *testing.T) {
	repo := &mockRelationsRepo{rows: oneRival()}
	svc := NewRelationsService(repo).withNow(rivalNotifNow)

	got, err := svc.DetectRivalEncounters(context.Background(), time.Time{})
	if err != nil {
		t.Fatalf("DetectRivalEncounters: %v", err)
	}
	if got != nil {
		t.Fatalf("watermark zéro → attendu nil, obtenu %+v", got)
	}
	if repo.scopeSeen {
		t.Fatal("watermark zéro ne doit PAS interroger le repo (court-circuit)")
	}
}
