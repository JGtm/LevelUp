// Package ops — seed_demo_synthetic.go : génération d'une fixture démo SYNTHÉTIQUE
// déterministe (aucune donnée réelle, aucune DB de prod, aucun CGO d'extraction).
//
// POURQUOI : le chemin `seed-demo` historique extrait N matchs d'un JOUEUR RÉEL
// anonymisé (data de prod + player DBs prod) — non exécutable en CI. Sans fixture,
// ~60 specs Playwright data-dépendantes se SKIPENT (garde e2e/_helpers/demoData.ts).
// Ce générateur construit `data/demo/` DE ZÉRO : DuckDB vierges migrées (mêmes
// migrations que la prod → mêmes vues `_latest`, même schéma append-only) + INSERT
// synthétiques déterministes. Deux runs → données byte-identiques (seed fixe, date
// d'ancrage fixe, aucun time.Now() non ancré).
//
// Respect des règles d'écriture (anti-ART) : chaque DB démo est FRAÎCHE et NON
// partagée (aucune contrainte mono-writer) ; les tables append-only sont écrites en
// INSERT pur (written_at posé) et lues via les vues `_latest` créées par les
// migrations canoniques. Aucun UPSERT/ON CONFLICT concurrent.
//
// Utilisation : levelup seed-demo --synthetic [--out data/demo]
package ops

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
)

// synthAnchor : instant de référence FIXE du corpus synthétique (déterminisme). Le
// match le plus récent se termine à cette date ; les autres remontent ~3 semaines.
// Ne JAMAIS remplacer par time.Now() (casserait le byte-identique run-à-run).
var synthAnchor = time.Date(2026, 7, 10, 21, 0, 0, 0, time.UTC)

// synthSeed : graine PRNG fixe (déterminisme des stats par match).
const synthSeed int64 = 20260710

// SyntheticAnchor retourne l'instant de référence fixe du corpus synthétique
// (pour affichage CLI / diagnostic déterminisme).
func SyntheticAnchor() time.Time { return synthAnchor }

// SyntheticDemoOptions configure la génération synthétique.
type SyntheticDemoOptions struct {
	OutDir     string // racine de sortie (ex: data/demo) — layout PLAT du titre par défaut
	ServiceTag string // Spartan ID affiché sous le gamertag démo
}

// SyntheticDemoResult résume la génération.
type SyntheticDemoResult struct {
	OutDir   string
	Matches  int
	Sessions int
	Players  int
	// PrestigeRows : lignes Prestige/Progression générées par table (arcs, objectifs,
	// événements PP, séries, records, escouade) — même générateur que la démo réelle.
	PrestigeRows   map[string]int
	Duration       time.Duration
	ConfigsWritten bool
}

// ── Référentiels synthétiques (dénormalisés dans match_registry) ──────────────

type synthMap struct{ en, fr string }
type synthMode struct{ en, fr string }

// synthPlaylist décrit une playlist synthétique. group = groupe canonique de
// skill_rating (arena / big_team_battle / ...) ; ranked → matchs classés (CSR).
type synthPlaylist struct {
	id, en, fr, group string
	ranked            bool
}

var synthMaps = []synthMap{
	{"Aquarius", "Aquarius"}, {"Live Fire", "Tir réel"}, {"Streets", "Rues"},
	{"Recharge", "Recharge"}, {"Bazaar", "Bazar"}, {"Behemoth", "Béhémoth"},
	{"Catalyst", "Catalyseur"}, {"Solitude", "Solitude"}, {"Forbidden", "Interdit"},
	{"Empyrean", "Empyréen"},
}

var synthModes = []synthMode{
	{"Slayer", "Massacre"}, {"Oddball", "Balle folle"}, {"Strongholds", "Bastions"},
	{"Capture the Flag", "Capture de drapeau"}, {"King of the Hill", "Roi de la colline"},
	{"Total Control", "Contrôle total"},
}

var synthPlaylists = []synthPlaylist{
	{"demo-ranked-arena", "Arène classée", "Arène classée", "arena", true},
	{"demo-quick-play", "Partie rapide", "Partie rapide", "arena", false},
	{"demo-btb", "Grand combat", "Grand combat", "big_team_battle", false},
	{"demo-fiesta", "Fiesta : Super", "Fiesta : Super", "arena", false},
}

// synthSession décrit une session de jeu (bloc de matchs rapprochés).
type synthSession struct {
	order       int // 1 = la plus ancienne
	dayOffset   int // décalage jours vs synthAnchor (négatif = passé)
	count       int // nombre de matchs
	squad       bool
	playlistIdx int
}

// synthSessions : 5 sessions sur ~3 semaines, 60 matchs, ≥2 sessions escouade
// (page Escouade + session-compare). La plus récente (order 5) est à l'ancre.
var synthSessions = []synthSession{
	{1, -20, 10, true, 0},  // Arène classée, escouade (la plus ancienne)
	{2, -14, 12, false, 1}, // Partie rapide, solo
	{3, -9, 12, true, 2},   // Grand combat, escouade
	{4, -4, 12, false, 0},  // Arène classée, solo
	{5, 0, 14, true, 3},    // Fiesta, escouade (la plus récente)
}

// synthMatch : plan complet d'un match démo (source unique lue par shared + player).
type synthMatch struct {
	idx          int
	matchID      string
	start        time.Time
	end          time.Time
	sessionID    string
	sessionLabel string
	sessionOrder int
	m            synthMap
	mode         synthMode
	pl           synthPlaylist
	squad        bool
	// stats du joueur principal (demo-player)
	kills, deaths, assists, score int
	outcome                       int // 2 win / 3 loss / 1 tie
	accuracy                      float64
	shotsFired, shotsHit          int
	damageDealt, damageTaken      float64
	headshots, meleeKills         int
	maxSpree                      int
	perfScore                     float64
	team0Score, team1Score        int
	// skill
	csrValue, lusrValue float64
	csrTier, csrTierFR  string
	csrSubTier          int
	csrDelta            float64
	// arme favorite (weapon_id présent dans weapon_labels seedé)
	// médailles (medal_name_id présents dans medal_definitions synthétiques)
	medals []int64
}

// Médailles synthétiques (medal_name_id) — insérées dans medal_definitions.
var synthMedalIDs = []int64{500000001, 500000002, 500000003, 500000004}

// Libellés de paliers EN réutilisés (>3 occurrences dans le package → constantes,
// garde-rail goconst).
const (
	tierNameBronze   = "Bronze"
	tierNamePlatinum = "Platinum"
)

// buildSynthPlan génère le corpus déterministe (60 matchs). Ordre chronologique
// croissant (idx 0 = le plus ancien). RNG à graine fixe → byte-identique.
func buildSynthPlan() []synthMatch {
	rng := rand.New(rand.NewSource(synthSeed)) //nolint:gosec // déterminisme voulu, pas de sécurité
	var plan []synthMatch
	idx := 0
	// CSR progresse doucement au fil du temps (feeling de progression).
	csr := 1150.0
	for _, s := range synthSessions {
		pl := synthPlaylists[s.playlistIdx]
		day := synthAnchor.AddDate(0, 0, s.dayOffset)
		// Début de session : 20h (heure locale fixe), matchs espacés ~12 min.
		sessionStart := time.Date(day.Year(), day.Month(), day.Day(), 20, 0, 0, 0, time.UTC)
		for k := 0; k < s.count; k++ {
			start := sessionStart.Add(time.Duration(k) * 12 * time.Minute)
			m := buildSynthMatch(rng, idx, s, pl, start, &csr)
			plan = append(plan, m)
			idx++
		}
	}
	return plan
}

// buildSynthMatch génère un match déterministe. csr est muté (progression continue).
func buildSynthMatch(rng *rand.Rand, idx int, s synthSession, pl synthPlaylist, start time.Time, csr *float64) synthMatch {
	mp := synthMaps[idx%len(synthMaps)]
	mode := synthModes[idx%len(synthModes)]
	kills := 8 + rng.Intn(18)
	deaths := 5 + rng.Intn(12)
	assists := rng.Intn(8)
	// Issue : ~55% victoires (win_rate crédible, wins ET losses).
	outcome := domain.OutcomeLoss
	t0, t1 := 40+rng.Intn(10), 50
	if rng.Float64() < 0.55 {
		outcome = domain.OutcomeWin
		t0, t1 = 50, 40+rng.Intn(10)
	}
	dur := 480 + rng.Intn(360) // 8-14 min
	acc := 0.38 + rng.Float64()*0.22
	shotsFired := 200 + rng.Intn(300)
	// CSR progression : +~8 en victoire, -~7 en défaite (borné 1050..1500).
	delta := -7.0 - rng.Float64()*4
	if outcome == domain.OutcomeWin {
		delta = 7.0 + rng.Float64()*4
	}
	*csr += delta
	if *csr < 1050 {
		*csr = 1050
	}
	if *csr > 1499 {
		*csr = 1499
	}
	tier, tierFR, subTier := csrTierFor(*csr)
	// Médailles : 0 à 3 par match (déterministe).
	var medals []int64
	nMed := rng.Intn(4)
	for i := 0; i < nMed; i++ {
		medals = append(medals, synthMedalIDs[(idx+i)%len(synthMedalIDs)])
	}
	return synthMatch{
		idx:          idx,
		matchID:      fmt.Sprintf("demo-match-%04d", idx),
		start:        start,
		end:          start.Add(time.Duration(dur) * time.Second),
		sessionID:    fmt.Sprintf("demo-session-%d", s.order),
		sessionLabel: fmt.Sprintf("Session %d", s.order),
		sessionOrder: s.order,
		m:            mp,
		mode:         mode,
		pl:           pl,
		squad:        s.squad,
		kills:        kills,
		deaths:       deaths,
		assists:      assists,
		score:        kills*100 + assists*40,
		outcome:      outcome,
		accuracy:     acc,
		shotsFired:   shotsFired,
		shotsHit:     int(float64(shotsFired) * acc),
		damageDealt:  float64(kills)*140 + rng.Float64()*300,
		damageTaken:  float64(deaths)*120 + rng.Float64()*250,
		headshots:    rng.Intn(kills/2 + 1),
		meleeKills:   rng.Intn(3),
		maxSpree:     3 + rng.Intn(6),
		perfScore:    50 + rng.Float64()*45,
		team0Score:   t0,
		team1Score:   t1,
		csrValue:     *csr,
		csrTier:      tier,
		csrTierFR:    tierFR,
		csrSubTier:   subTier,
		csrDelta:     delta,
		lusrValue:    1000 + (*csr-1150)*0.9 + rng.Float64()*20,
		medals:       medals,
	}
}

// csrTierFor mappe une valeur CSR → (tier EN, tier FR, sous-palier 1..6).
func csrTierFor(v float64) (string, string, int) {
	switch {
	case v < 1100:
		return "Silver", "Argent", 3
	case v < 1300:
		return "Gold", "Or", 2
	case v < 1450:
		return tierNamePlatinum, "Platine", 1
	default:
		return "Diamond", "Diamant", 1
	}
}

// SeedDemoSynthetic construit une fixture démo synthétique complète dans OutDir.
func SeedDemoSynthetic(ctx context.Context, opts SyntheticDemoOptions) (SyntheticDemoResult, error) {
	start := time.Now()
	res := SyntheticDemoResult{OutDir: opts.OutDir}
	if opts.OutDir == "" {
		return res, fmt.Errorf("seed-demo synthetic: OutDir requis")
	}
	if opts.ServiceTag == "" {
		opts.ServiceTag = "DEMO"
	}

	plan := buildSynthPlan()
	res.Matches = len(plan)
	res.Sessions = len(synthSessions)

	slog.InfoContext(ctx, "seed-demo synthetic: démarrage", "out", opts.OutDir, "matches", len(plan), "sessions", len(synthSessions))

	warehouse := filepath.Join(opts.OutDir, "warehouse")
	if err := os.MkdirAll(warehouse, 0o755); err != nil {
		return res, fmt.Errorf("seed-demo synthetic: mkdir warehouse: %w", err)
	}

	// 1. Metadata (référentiels migrés + seeds Go + medals/ranks synthétiques).
	metaPath := filepath.Join(warehouse, "metadata.duckdb")
	if err := writeSyntheticMetadata(ctx, metaPath); err != nil {
		return res, fmt.Errorf("seed-demo synthetic: metadata: %w", err)
	}

	// 2. Shared (matchs partagés + participants + médailles + armes + CSR + kill-feed).
	sharedPath := filepath.Join(warehouse, "shared_matches_v2.duckdb")
	if err := writeSyntheticShared(ctx, sharedPath, plan); err != nil {
		return res, fmt.Errorf("seed-demo synthetic: shared: %w", err)
	}

	// 3. shared_social (vide mais migré : la page Média répond {items: []}).
	socialPath := filepath.Join(warehouse, "shared_social.duckdb")
	if err := writeSyntheticSharedSocial(socialPath); err != nil {
		return res, fmt.Errorf("seed-demo synthetic: shared_social: %w", err)
	}

	// 4. Player DBs (DemoPlayer principal + 2 coéquipiers).
	nPlayers, err := writeSyntheticPlayers(ctx, opts.OutDir, plan)
	if err != nil {
		return res, fmt.Errorf("seed-demo synthetic: players: %w", err)
	}
	res.Players = nPlayers

	// 4b. Échantillons Prestige (arcs, objectifs, points de progression, séries,
	// records, jalons, escouade) — même générateur que la démo réelle, dérivé du
	// corpus synthétique. Sans eux, app_settings pose prestige_enabled=true et les
	// pages Ascension/Prestige de la fixture CI seraient vides.
	prestigeRows, err := seedDemoPrestige(ctx, opts.OutDir, titlePkg.DefaultSlug, nil)
	if err != nil {
		return res, fmt.Errorf("seed-demo synthetic: prestige: %w", err)
	}
	res.PrestigeRows = prestigeRows

	// 5. Configs (db_profiles v3 + app_settings) — même forme que la démo réelle.
	if err := writeSyntheticConfigs(opts.OutDir, opts.ServiceTag); err != nil {
		return res, fmt.Errorf("seed-demo synthetic: configs: %w", err)
	}
	res.ConfigsWritten = true

	res.Duration = time.Since(start)
	slog.InfoContext(ctx, "seed-demo synthetic: terminé", "duration", res.Duration, "matches", res.Matches, "players", res.Players)
	return res, nil
}

// writeSyntheticConfigs écrit db_profiles.json (v3, roster de 3) + app_settings.json.
// Réutilise demoAppSettings + writeDemoConfigsV3 (parité avec la démo réelle).
func writeSyntheticConfigs(outDir, serviceTag string) error {
	byTitle := map[string][]seededDemoPlayer{
		titlePkg.DefaultSlug: {
			{Dir: demoDirForIndex(0), XUID: demoXUIDForIndex(0), Gamertag: DefaultDemoMainGamertag},
			{Dir: demoDirForIndex(1), XUID: demoXUIDForIndex(1), Gamertag: "DemoPlayer2"},
			{Dir: demoDirForIndex(2), XUID: demoXUIDForIndex(2), Gamertag: "DemoPlayer3"},
		},
	}
	return writeDemoConfigsV3(outDir, byTitle, serviceTag, false)
}
