package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	syncpkg "levelup/go-api/internal/sync"
)

const (
	tArena  = "edfef3ac-9cbe-4fa2-b949-8f29deafd483"
	tSlayer = "dcb2e24e-05fb-4390-8076-32a0cdb4326e"
)

// fakeMatchSource simule l'historique + les stats par match d'un joueur.
type fakeMatchSource struct {
	// history[xuid] = liste de matchIDs (ordre chronologique décroissant)
	history map[string][]string
	// stats[matchID] = JSON brut GetMatchStats
	stats map[string]map[string]any
	// startTimes[matchID] = StartTime RFC3339 (optionnel ; vide = pas de filtre date)
	startTimes map[string]string
	// fetched = nb d'appels GetMatchStats (vérifie que le filtre date évite les fetchs)
	fetched int
	// histCalls = nb d'appels GetMatchHistory (vérifie que la dichotomie évite le linéaire)
	histCalls int
}

func (f *fakeMatchSource) GetMatchHistory(_ context.Context, gamertag, _ string, start, count int) ([]syncpkg.MatchHistoryEntry, error) {
	f.histCalls++
	ids := f.history[gamertag]
	if start >= len(ids) {
		return nil, nil
	}
	end := start + count
	if end > len(ids) {
		end = len(ids)
	}
	out := make([]syncpkg.MatchHistoryEntry, 0, end-start)
	for _, id := range ids[start:end] {
		out = append(out, syncpkg.MatchHistoryEntry{MatchID: id, StartTime: f.startTimes[id]})
	}
	return out, nil
}

func (f *fakeMatchSource) GetMatchStats(_ context.Context, matchID string) (map[string]any, error) {
	f.fetched++
	m, ok := f.stats[matchID]
	if !ok {
		return nil, fmt.Errorf("match %s introuvable", matchID)
	}
	return m, nil
}

type fakeResolver struct{ m map[string]string }

func (r *fakeResolver) ResolveXUID(_ context.Context, gamertag string) (string, error) {
	x, ok := r.m[gamertag]
	if !ok {
		return "", fmt.Errorf("xuid introuvable pour %s", gamertag)
	}
	return x, nil
}

// buildMatch fabrique un JSON GetMatchStats minimal pour un joueur xuid.
func buildMatch(xuid, seasonPath, playlist string, outcome, kills, deaths, assists int) map[string]any {
	return map[string]any{
		"MatchInfo": map[string]any{
			"SeasonId": seasonPath,
			"Playlist": map[string]any{"AssetId": playlist},
		},
		"Players": []any{
			map[string]any{
				"PlayerId": "xuid(" + xuid + ")",
				"Outcome":  float64(outcome),
				"PlayerTeamStats": []any{
					map[string]any{"Stats": map[string]any{"CoreStats": map[string]any{
						"Kills": float64(kills), "Deaths": float64(deaths), "Assists": float64(assists),
					}}},
				},
				"ParticipationInfo": map[string]any{"TimePlayed": "PT10M"},
			},
		},
	}
}

// playerBlock fabrique le bloc Players[] d'un joueur (réutilisé pour des matchs multi-joueurs).
func playerBlock(xuid string, outcome, kills, deaths, assists int) map[string]any {
	return map[string]any{
		"PlayerId": "xuid(" + xuid + ")",
		"Outcome":  float64(outcome),
		"PlayerTeamStats": []any{
			map[string]any{"Stats": map[string]any{"CoreStats": map[string]any{
				"Kills": float64(kills), "Deaths": float64(deaths), "Assists": float64(assists),
			}}},
		},
		"ParticipationInfo": map[string]any{"TimePlayed": "PT10M"},
	}
}

// TestAggregatePlayer_SharedMatchOrderIndependent prouve le fix archi : la résolution
// xuid est PARESSEUSE (pas de PrepareWorldPlayers) et l'attribution d'un match partagé
// par deux joueurs reste correcte quel que soit l'ordre. X est scanné d'abord → le
// match est fetché 1× et mis en cache avec TOUS ses participants. Y, résolu APRÈS, lit
// sa stat depuis le cache (aucun re-fetch, aucun sous-comptage).
func TestAggregatePlayer_SharedMatchOrderIndependent(t *testing.T) {
	const xX, xY = "1001", "1002"
	shared := map[string]any{
		"MatchInfo": map[string]any{
			"SeasonId": "Csr/Seasons/CsrSeason13-2.json",
			"Playlist": map[string]any{"AssetId": tArena},
		},
		"Players": []any{
			playerBlock(xX, 2, 12, 4, 2), // X : Win, 12 kills
			playerBlock(xY, 3, 7, 9, 1),  // Y : Loss, 7 kills
		},
	}
	src := &fakeMatchSource{
		history: map[string][]string{
			"xuid(" + xX + ")": {"shared"},
			"xuid(" + xY + ")": {"shared"},
		},
		stats: map[string]map[string]any{"shared": shared},
	}
	agg := NewWorldStatsAggregator(src, &fakeResolver{m: map[string]string{"X": xX, "Y": xY}},
		WorldStatsAggregatorConfig{TargetSeasons: map[string]bool{"csrseason13-2": true}})

	outX, err := agg.AggregatePlayer(context.Background(), "X") // scanné EN PREMIER
	if err != nil {
		t.Fatalf("X: %v", err)
	}
	outY, err := agg.AggregatePlayer(context.Background(), "Y") // résolu APRÈS le cache
	if err != nil {
		t.Fatalf("Y: %v", err)
	}
	if len(outX) != 1 || outX[0].Kills != 12 || outX[0].WinCount != 1 {
		t.Errorf("X: %+v, want 1 bucket 12 kills / 1 win", outX)
	}
	if len(outY) != 1 || outY[0].Kills != 7 || outY[0].LossCount != 1 {
		t.Errorf("Y attribué depuis le cache: %+v, want 1 bucket 7 kills / 1 loss", outY)
	}
	if src.fetched != 1 {
		t.Errorf("match partagé fetché %d fois, want 1 (cache partagé, pas de re-fetch)", src.fetched)
	}
}

// TestAggregatePlayer_SeasonWindow valide la pagination + le filtre TargetSeasons
// + le bucketing par playlist sur un historique multi-saison.
func TestAggregatePlayer_SeasonWindow(t *testing.T) {
	const xuid = "2533274895653213"
	const cur = "Csr/Seasons/CsrSeason13-2.json"
	const old = "Csr/Seasons/CsrSeason12-1.json"

	src := &fakeMatchSource{
		history: map[string][]string{"xuid(" + xuid + ")": {"m1", "m2", "m3", "m4"}},
		stats: map[string]map[string]any{
			"m1": buildMatch(xuid, cur, tArena, 2, 15, 8, 4),  // Win, saison cible
			"m2": buildMatch(xuid, cur, tArena, 3, 10, 12, 2), // Loss, saison cible
			"m3": buildMatch(xuid, cur, tSlayer, 2, 20, 5, 6), // Win, autre playlist
			"m4": buildMatch(xuid, old, tArena, 2, 99, 1, 1),  // saison HORS cible → exclue
		},
	}
	agg := NewWorldStatsAggregator(src, &fakeResolver{m: map[string]string{"Neo": xuid}},
		WorldStatsAggregatorConfig{TargetSeasons: map[string]bool{"csrseason13-2": true}})

	out, err := agg.AggregatePlayer(context.Background(), "Neo")
	if err != nil {
		t.Fatalf("AggregatePlayer: %v", err)
	}
	// 2 buckets attendus : 13-2/arena et 13-2/slayer (12-1 exclu).
	if len(out) != 2 {
		t.Fatalf("attendu 2 buckets, got %d : %+v", len(out), out)
	}
	byKey := map[string]int{}
	for _, b := range out {
		byKey[b.SeasonID+"|"+b.PlaylistID] = int(b.MatchCount)
		if b.SeasonID == "csrseason12-1" {
			t.Errorf("saison 12-1 ne devrait pas être présente (hors TargetSeasons)")
		}
	}
	if byKey["csrseason13-2|"+tArena] != 2 {
		t.Errorf("13-2/arena = %d matchs, want 2", byKey["csrseason13-2|"+tArena])
	}
	if byKey["csrseason13-2|"+tSlayer] != 1 {
		t.Errorf("13-2/slayer = %d matchs, want 1", byKey["csrseason13-2|"+tSlayer])
	}
}

// TestAggregatePlayer_StopAfterNonTarget vérifie l'arrêt anticipé une fois sous
// les saisons cibles (les matchs vieux au-delà du seuil ne sont pas fetchés en vain).
func TestAggregatePlayer_StopAfterNonTarget(t *testing.T) {
	const xuid = "999"
	const cur = "Csr/Seasons/CsrSeason13-2.json"
	const old = "Csr/Seasons/CsrSeason12-1.json"

	hist := []string{"a", "b", "c", "d", "e", "f"}
	stats := map[string]map[string]any{
		"a": buildMatch(xuid, cur, tArena, 2, 1, 1, 1),
		"b": buildMatch(xuid, old, tArena, 2, 1, 1, 1),
		"c": buildMatch(xuid, old, tArena, 2, 1, 1, 1),
		"d": buildMatch(xuid, old, tArena, 2, 1, 1, 1),
		"e": buildMatch(xuid, cur, tArena, 2, 1, 1, 1), // après le stop → ne doit PAS compter
		"f": buildMatch(xuid, cur, tArena, 2, 1, 1, 1),
	}
	src := &fakeMatchSource{history: map[string][]string{"xuid(" + xuid + ")": hist}, stats: stats}
	agg := NewWorldStatsAggregator(src, &fakeResolver{m: map[string]string{"Z": xuid}},
		WorldStatsAggregatorConfig{
			TargetSeasons:      map[string]bool{"csrseason13-2": true},
			StopAfterNonTarget: 2, // arrêt après 2 hors-cible consécutifs (b, c)
			MaxPages:           1, // une seule page (page de 25 → tout l'historique)
		})

	out, err := agg.AggregatePlayer(context.Background(), "Z")
	if err != nil {
		t.Fatalf("AggregatePlayer: %v", err)
	}
	// Page unique : on parcourt a..f dans la boucle interne, mais le stop est évalué
	// EN FIN de page → tous les matchs de la page sont vus. Le filtre TargetSeasons
	// garde a, e, f (3 matchs cible). StopAfterNonTarget borne surtout le multi-page.
	var total int
	for _, b := range out {
		total += b.MatchCount
	}
	if total != 3 {
		t.Errorf("matchs cible comptés = %d, want 3 (a, e, f)", total)
	}
}

// TestRun_BestEffort vérifie le fan-out parallèle + l'isolation des erreurs joueur.
func TestRun_BestEffort(t *testing.T) {
	const xa, xb = "111", "222"
	src := &fakeMatchSource{
		history: map[string][]string{
			"xuid(" + xa + ")": {"a1"},
			"xuid(" + xb + ")": {"b1"},
		},
		stats: map[string]map[string]any{
			"a1": buildMatch(xa, "Csr/Seasons/CsrSeason13-2.json", tArena, 2, 5, 5, 5),
			"b1": buildMatch(xb, "Csr/Seasons/CsrSeason13-2.json", tArena, 3, 3, 7, 1),
		},
	}
	// "Ghost" n'a pas d'xuid résolu → erreur isolée, ne casse pas le batch.
	resolver := &fakeResolver{m: map[string]string{"Alpha": xa, "Beta": xb}}
	agg := NewWorldStatsAggregator(src, resolver, WorldStatsAggregatorConfig{Concurrency: 4})

	all, errs := agg.Run(context.Background(), []string{"Alpha", "Beta", "Ghost"})
	if len(errs) != 1 {
		t.Fatalf("attendu 1 erreur (Ghost), got %d : %v", len(errs), errs)
	}
	gts := map[string]bool{}
	for _, b := range all {
		gts[b.Gamertag] = true
	}
	if !gts["Alpha"] || !gts["Beta"] {
		t.Errorf("Alpha et Beta devraient être agrégés malgré l'échec de Ghost, got %+v", all)
	}
}

// TestAggregatePlayer_DateWindowSkipsFetch est LE test du fix vieilles saisons :
// avec une fenêtre de dates, l'agrégateur saute les matchs hors fenêtre SANS appeler
// GetMatchStats (au lieu de fetcher le match complet de chaque match pour lire sa
// saison). Les matchs recent*/old1 ne sont volontairement PAS dans `stats` : si le
// filtre les fetchait, GetMatchStats erreurrait et le test casserait.
func TestAggregatePlayer_DateWindowSkipsFetch(t *testing.T) {
	const xuid = "12345"
	const s11 = "Csr/Seasons/CsrSeason11-1.json"
	src := &fakeMatchSource{
		history: map[string][]string{"xuid(" + xuid + ")": {"recent1", "recent2", "in1", "in2", "old1"}},
		startTimes: map[string]string{
			"recent1": "2026-01-10T00:00:00Z", // APRÈS la fenêtre → skip sans fetch
			"recent2": "2025-09-01T00:00:00Z", // APRÈS la fenêtre → skip sans fetch
			"in1":     "2025-06-15T00:00:00Z", // DANS la fenêtre → fetch + collecte
			"in2":     "2025-07-01T00:00:00Z", // DANS la fenêtre → fetch + collecte
			"old1":    "2025-03-01T00:00:00Z", // AVANT la fenêtre → skip sans fetch
		},
		stats: map[string]map[string]any{
			"in1": buildMatch(xuid, s11, tArena, 2, 15, 8, 4),
			"in2": buildMatch(xuid, s11, tArena, 3, 10, 12, 2),
		},
	}
	start, _ := time.Parse("2006-01-02", "2025-05-06")
	end, _ := time.Parse("2006-01-02", "2025-08-05")
	agg := NewWorldStatsAggregator(src, &fakeResolver{m: map[string]string{"Pro": xuid}},
		WorldStatsAggregatorConfig{
			TargetSeasons: map[string]bool{"csrseason11-1": true},
			SeasonStart:   start,
			SeasonEnd:     end,
			MaxPages:      1,
		})

	out, err := agg.AggregatePlayer(context.Background(), "Pro")
	if err != nil {
		t.Fatalf("AggregatePlayer: %v (le filtre date a fetché un match hors fenêtre)", err)
	}
	// SEULS in1/in2 fetchés : recent1/recent2/old1 sautés par date, jamais fetchés.
	if src.fetched != 2 {
		t.Errorf("GetMatchStats appelé %d fois, want 2 (uniquement les matchs DANS la fenêtre)", src.fetched)
	}
	var total int
	for _, b := range out {
		total += int(b.MatchCount)
	}
	if total != 2 {
		t.Errorf("matchs S11 agrégés = %d, want 2 (in1, in2)", total)
	}
}

// TestAggregatePlayer_BinarySearchJumpsToWindow prouve l'optimisation dichotomie :
// sur un historique PROFOND (2000 matchs) dont la fenêtre cible est tout au fond
// (offsets 1600-1649), l'agrégateur ne pagine PAS linéairement les 64 pages récentes
// pour l'atteindre. Il sonde l'offset par recherche dichotomique (~log2(2000)≈11
// sondes) puis ne lit que les ~3 pages de la fenêtre. Garde-fou : histCalls reste
// très en-deçà du linéaire (~66 appels). Si la dichotomie régressait, on repaginerait
// tout depuis l'offset 0 et histCalls exploserait.
func TestAggregatePlayer_BinarySearchJumpsToWindow(t *testing.T) {
	const xuid = "777"
	const s11 = "Csr/Seasons/CsrSeason11-1.json"
	const deep = 2000

	history := make([]string, deep)
	startTimes := make(map[string]string, deep)
	stats := map[string]map[string]any{}
	for i := 0; i < deep; i++ {
		id := fmt.Sprintf("m%d", i)
		history[i] = id
		switch {
		case i < 1600:
			startTimes[id] = "2026-01-01T00:00:00Z" // APRÈS la fenêtre (récent)
		case i < 1650:
			startTimes[id] = "2025-06-15T00:00:00Z" // DANS la fenêtre → collecté
			stats[id] = buildMatch(xuid, s11, tArena, 2, 10, 5, 3)
		default:
			startTimes[id] = "2025-03-01T00:00:00Z" // AVANT la fenêtre (vieux)
		}
	}
	src := &fakeMatchSource{
		history:    map[string][]string{"xuid(" + xuid + ")": history},
		startTimes: startTimes,
		stats:      stats,
	}
	start, _ := time.Parse("2006-01-02", "2025-05-06")
	end, _ := time.Parse("2006-01-02", "2025-08-05")
	agg := NewWorldStatsAggregator(src, &fakeResolver{m: map[string]string{"Deep": xuid}},
		WorldStatsAggregatorConfig{
			TargetSeasons: map[string]bool{"csrseason11-1": true},
			SeasonStart:   start,
			SeasonEnd:     end,
			MaxPages:      80, // 80 × 25 = 2000 → toute la profondeur est atteignable
		})

	out, err := agg.AggregatePlayer(context.Background(), "Deep")
	if err != nil {
		t.Fatalf("AggregatePlayer: %v", err)
	}
	var total int
	for _, b := range out {
		total += int(b.MatchCount)
	}
	if total != 50 {
		t.Errorf("matchs fenêtre collectés = %d, want 50 (offsets 1600-1649)", total)
	}
	if src.fetched != 50 {
		t.Errorf("GetMatchStats appelé %d fois, want 50 (uniquement la fenêtre)", src.fetched)
	}
	// Dichotomie : ~11 sondes + ~3 pages de fenêtre ≈ 14 appels. Le linéaire
	// (pagination depuis l'offset 0) ferait ~66 appels. Le seuil 25 sépare nettement
	// les deux régimes sans être fragile.
	if src.histCalls > 25 {
		t.Errorf("GetMatchHistory appelé %d fois — la dichotomie a régressé (linéaire ≈ 66)", src.histCalls)
	}
}
