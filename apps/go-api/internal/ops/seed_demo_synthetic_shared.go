// Package ops — seed_demo_synthetic_shared.go : shared_matches_v2.duckdb + shared_social.duckdb.
//
// DB fraîches migrées (RunForTitleDB TargetShared / TargetSharedSocial) puis INSERT
// synthétiques déterministes. Les noms carte/mode/playlist sont écrits dénormalisés
// (map_name/playlist_name/..._fr) → aucune lecture metadata au rendu.
package ops

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"

	_ "github.com/duckdb/duckdb-go/v2"
)

// synthDemoSeason : season_id fixe du corpus synthétique.
const synthDemoSeason = "demo-season"

// synthParticipant décrit un participant d'un match (principal, coéquipier, adversaire).
type synthParticipant struct {
	xuid, gamertag         string
	team                   int
	kills, deaths, assists int
	score, outcome         int
	accuracy               float64
}

// opponent gamertags/xuids stables (recurring adversaires — anonymes "Player N").
var synthOpponents = []synthParticipant{
	{demoXUIDForIndex(3), "Player 4", 1, 0, 0, 0, 0, 0, 0},
	{demoXUIDForIndex(4), "Player 5", 1, 0, 0, 0, 0, 0, 0},
	{demoXUIDForIndex(5), "Player 6", 1, 0, 0, 0, 0, 0, 0},
}

// writeSyntheticShared crée shared_matches_v2.duckdb + insère tout le corpus.
func writeSyntheticShared(ctx context.Context, path string, plan []synthMatch) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	if err := removeDuckDBForFreshWrite(path); err != nil {
		return err
	}
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer db.Close()
	if err := migration.RunForTitleDB(db, titlePkg.DefaultSlug, migration.TargetShared); err != nil {
		return fmt.Errorf("migrations shared: %w", err)
	}

	aliases := map[string]string{
		demoXUIDForIndex(0): DefaultDemoMainGamertag,
		demoXUIDForIndex(1): "DemoPlayer2",
		demoXUIDForIndex(2): "DemoPlayer3",
	}
	for _, m := range plan {
		if err := insertSharedMatch(ctx, db, m, aliases); err != nil {
			return fmt.Errorf("match %s: %w", m.matchID, err)
		}
	}
	for x, gt := range aliases {
		if _, err := db.ExecContext(ctx,
			`INSERT OR IGNORE INTO xuid_aliases (xuid, gamertag, last_seen, source) VALUES (?, ?, ?, 'demo')`,
			x, gt, synthAnchor); err != nil {
			return fmt.Errorf("alias %s: %w", x, err)
		}
	}
	return nil
}

// insertSharedMatch insère un match : registry + participants + médailles + armes +
// CSR + kill-feed + highlight events. Renseigne aliases pour tous les xuids vus.
func insertSharedMatch(ctx context.Context, db *sql.DB, m synthMatch, aliases map[string]string) error {
	if err := insertMatchRegistry(ctx, db, m); err != nil {
		return err
	}
	parts := buildMatchParticipants(m)
	for _, p := range parts {
		aliases[p.xuid] = p.gamertag
		if err := insertMatchParticipant(ctx, db, m, p); err != nil {
			return err
		}
	}
	// Médailles + armes + CSR + kill-feed pour le joueur principal.
	if err := insertMatchMedals(ctx, db, m); err != nil {
		return err
	}
	if err := insertMatchWeaponKills(ctx, db, m); err != nil {
		return err
	}
	if m.pl.ranked {
		if err := insertMatchCSR(ctx, db, m); err != nil {
			return err
		}
	}
	return insertMatchEvents(ctx, db, m, parts)
}

// insertMatchRegistry écrit une ligne match_registry avec les noms dénormalisés (+_fr).
func insertMatchRegistry(ctx context.Context, db *sql.DB, m synthMatch) error {
	pair := fmt.Sprintf("%s - %s", m.m.en, m.mode.en)
	pairFR := fmt.Sprintf("%s - %s", m.m.fr, m.mode.fr)
	_, err := db.ExecContext(ctx, `
		INSERT INTO match_registry
			(match_id, start_time, end_time, start_time_utc, end_time_utc,
			 playlist_id, playlist_name, playlist_name_fr,
			 map_id, map_name, map_name_fr, pair_id, pair_name, pair_name_fr,
			 game_variant_name, mode_category, is_ranked, duration_seconds,
			 team_0_score, team_1_score, player_count, events_loaded)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, TRUE)`,
		m.matchID, m.start, m.end, m.start, m.end,
		m.pl.id, m.pl.en, m.pl.fr,
		"map-"+m.m.en, m.m.en, m.m.fr, "pair-"+m.mode.en, pair, pairFR,
		m.mode.en, m.mode.en, m.pl.ranked, int(m.end.Sub(m.start).Seconds()),
		m.team0Score, m.team1Score, len(buildMatchParticipants(m)))
	return err
}

// buildMatchParticipants retourne le roster complet d'un match (déterministe).
func buildMatchParticipants(m synthMatch) []synthParticipant {
	main := synthParticipant{
		xuid: demoXUIDForIndex(0), gamertag: DefaultDemoMainGamertag, team: 0,
		kills: m.kills, deaths: m.deaths, assists: m.assists, score: m.score,
		outcome: m.outcome, accuracy: m.accuracy,
	}
	parts := []synthParticipant{main}
	oppOutcome := domain.OutcomeLoss
	if m.outcome == domain.OutcomeLoss {
		oppOutcome = domain.OutcomeWin
	}
	if m.squad {
		// Coéquipiers DemoPlayer2/3 (même équipe, même issue).
		parts = append(parts,
			derivedParticipant(m, demoXUIDForIndex(1), "DemoPlayer2", 0, m.outcome, 1),
			derivedParticipant(m, demoXUIDForIndex(2), "DemoPlayer3", 0, m.outcome, 2))
	}
	// Adversaires (2 en arène, 3 en grand combat).
	nOpp := 2
	if m.pl.group == "big_team_battle" {
		nOpp = 3
	}
	for i := 0; i < nOpp; i++ {
		o := synthOpponents[i]
		parts = append(parts, derivedParticipant(m, o.xuid, o.gamertag, 1, oppOutcome, i+3))
	}
	return parts
}

// derivedParticipant génère des stats déterministes pour un non-principal (dérivées
// de l'idx du match + un offset stable par joueur).
func derivedParticipant(m synthMatch, xuid, gt string, team, outcome, off int) synthParticipant {
	k := 6 + (m.idx*7+off*3)%14
	d := 5 + (m.idx*5+off*2)%11
	a := (m.idx + off) % 7
	return synthParticipant{
		xuid: xuid, gamertag: gt, team: team,
		kills: k, deaths: d, assists: a, score: k*100 + a*40,
		outcome: outcome, accuracy: 0.35 + float64((m.idx+off)%20)/100.0,
	}
}

// insertMatchParticipant écrit une ligne match_participants.
func insertMatchParticipant(ctx context.Context, db *sql.DB, m synthMatch, p synthParticipant) error {
	kda := float64(p.kills) + float64(p.assists)/3.0 - float64(p.deaths)
	shotsFired := 180 + p.kills*12
	_, err := db.ExecContext(ctx, `
		INSERT INTO match_participants
			(match_id, xuid, gamertag, team_id, outcome, rank, score, kills, deaths, assists,
			 kda, accuracy, shots_fired, shots_hit, damage_dealt, damage_taken, personal_score,
			 time_played_seconds, headshot_kills, max_killing_spree, melee_kills, power_weapon_kills,
			 present_at_beginning, present_at_completion, team_mmr, enemy_mmr)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, TRUE, TRUE, ?, ?)`,
		m.matchID, p.xuid, p.gamertag, p.team, p.outcome, 1, p.score, p.kills, p.deaths, p.assists,
		kda, p.accuracy, shotsFired, int(float64(shotsFired)*p.accuracy),
		float64(p.kills)*140, float64(p.deaths)*120, p.score,
		int(m.end.Sub(m.start).Seconds()), p.kills/3, 3+p.kills/4, p.assists/3, p.kills/5,
		1500.0, 1500.0)
	return err
}

// insertMatchMedals écrit medals_earned pour le joueur principal.
func insertMatchMedals(ctx context.Context, db *sql.DB, m synthMatch) error {
	for _, id := range m.medals {
		if _, err := db.ExecContext(ctx,
			`INSERT OR IGNORE INTO medals_earned (match_id, xuid, medal_name_id, count) VALUES (?, ?, ?, 1)`,
			m.matchID, demoXUIDForIndex(0), id); err != nil {
			return err
		}
	}
	return nil
}

// insertMatchWeaponKills écrit weapon_kills (arme favorite) pour le joueur principal.
// Lecture home via v_weapon_kills → weapon_labels (seed migration).
func insertMatchWeaponKills(ctx context.Context, db *sql.DB, m synthMatch) error {
	for i := 0; i < m.favWeaponKills; i++ {
		// written_at ancré sur synthAnchor : weapon_kills est append-only (DEFAULT
		// now() posé par la migration) ; sans valeur explicite chaque run diverge.
		if _, err := db.ExecContext(ctx, `
			INSERT INTO weapon_kills (match_id, xuid, time_ms, weapon_id, confidence, attribution_path, written_at)
			VALUES (?, ?, ?, ?, 'high', 'demo', ?)`,
			m.matchID, demoXUIDForIndex(0), (i+1)*20000, int64(m.favWeaponID), synthAnchor); err != nil {
			return err
		}
	}
	return nil
}

// insertMatchCSR écrit match_csrs (append-only : written_at posé, lecture _latest).
func insertMatchCSR(ctx context.Context, db *sql.DB, m synthMatch) error {
	label := fmt.Sprintf("%s %d", m.csrTierFR, m.csrSubTier)
	_, err := db.ExecContext(ctx, `
		INSERT INTO match_csrs
			(match_id, xuid, rating_type, rating_value, tier, sub_tier, tier_label,
			 rating_delta, measurement_matches_remaining, season_id, written_at)
		VALUES (?, ?, 'CSR', ?, ?, ?, ?, ?, 0, ?, ?)`,
		m.matchID, demoXUIDForIndex(0), m.csrValue, m.csrTier, m.csrSubTier, label,
		m.csrDelta, synthDemoSeason, m.end)
	return err
}

// insertMatchEvents écrit highlight_events (kills/deaths des 2 équipes → timeline
// combat / tug-of-war) + killer_victim_pairs (kill-feed).
func insertMatchEvents(ctx context.Context, db *sql.DB, m synthMatch, parts []synthParticipant) error {
	dur := int(m.end.Sub(m.start).Seconds()) * 1000
	// Kills du joueur principal (répartis sur la durée).
	for i := 0; i < m.kills; i++ {
		t := (i + 1) * dur / (m.kills + 1)
		if _, err := db.ExecContext(ctx,
			`INSERT INTO highlight_events (match_id, event_type, time_ms, xuid) VALUES (?, 'kill', ?, ?)`,
			m.matchID, t, demoXUIDForIndex(0)); err != nil {
			return err
		}
	}
	for i := 0; i < m.deaths; i++ {
		t := (i + 1) * dur / (m.deaths + 1)
		if _, err := db.ExecContext(ctx,
			`INSERT INTO highlight_events (match_id, event_type, time_ms, xuid) VALUES (?, 'death', ?, ?)`,
			m.matchID, t, demoXUIDForIndex(0)); err != nil {
			return err
		}
	}
	// Kills d'un adversaire (team 1) → tug-of-war a bien les 2 côtés.
	var opp *synthParticipant
	for i := range parts {
		if parts[i].team == 1 {
			opp = &parts[i]
			break
		}
	}
	if opp != nil {
		for i := 0; i < opp.kills; i++ {
			t := (i + 1) * dur / (opp.kills + 1)
			if _, err := db.ExecContext(ctx,
				`INSERT INTO highlight_events (match_id, event_type, time_ms, xuid) VALUES (?, 'kill', ?, ?)`,
				m.matchID, t, opp.xuid); err != nil {
				return err
			}
			// kill-feed principal → adversaire.
			if _, err := db.ExecContext(ctx, `
				INSERT INTO killer_victim_pairs
					(match_id, killer_xuid, killer_gamertag, victim_xuid, victim_gamertag, kill_count, time_ms)
				VALUES (?, ?, ?, ?, ?, 1, ?)`,
				m.matchID, demoXUIDForIndex(0), DefaultDemoMainGamertag, opp.xuid, opp.gamertag, t); err != nil {
				return err
			}
		}
	}
	return nil
}

// writeSyntheticSharedSocial crée shared_social.duckdb migré (vide). Suffit pour que
// la page Média réponde {items: []} (schéma media_files/associations présent).
func writeSyntheticSharedSocial(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	if err := removeDuckDBForFreshWrite(path); err != nil {
		return err
	}
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer db.Close()
	if err := migration.RunForTitleDB(db, titlePkg.DefaultSlug, migration.TargetSharedSocial); err != nil {
		return fmt.Errorf("migrations shared_social: %w", err)
	}
	return nil
}
