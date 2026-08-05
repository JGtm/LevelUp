// Package ops — seed_demo_synthetic_player.go : player DBs synthétiques.
//
// DEMO (principal, 60 matchs) + DEMO2/DEMO3 (coéquipiers, matchs escouade). Chaque
// DB migrée (RunForTitleDB TargetPlayer) puis peuplée en INSERT :
//   - career_progression (rang, XP, bannière/emblème/backdrop pour l'identité home) ;
//   - sessions + player_match_enrichment (perf/session/engagement) ;
//   - match_skill_rank (CSR classé / LUSR non-classé, lecture pics via _latest) ;
//   - match_citations (append-only) ; player_csr_snapshots (CSR alltime, home peak).
package ops

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"

	_ "github.com/duckdb/duckdb-go/v2"
)

// synthPlayerSpec décrit un joueur démo (identité + rang affiché).
type synthPlayerSpec struct {
	dir      string
	xuid     string
	gamertag string
	rank     int
	rankName string
	rankTier string
}

// synthCitations : citation_name_norm présents dans citation_mappings (seed Go).
// Ce sont des IDENTIFIANTS de citations (données), un sous-ensemble curé des norms
// définis inline dans seed_citation_data.go — pas de la config dupliquée à centraliser.
//
//nolint:goconst // norms démo (data) partagés avec seed_citation_data.go
var synthCitations = []string{"multikill", "assistant", "close_combat", "assassin", "charge", "splatter"}

// writeSyntheticPlayers génère les 3 player DBs. Retourne le nombre de joueurs seedés.
func writeSyntheticPlayers(ctx context.Context, outDir string, plan []synthMatch) (int, error) {
	specs := []synthPlayerSpec{
		{demoDirForIndex(0), demoXUIDForIndex(0), DefaultDemoMainGamertag, 200, "Lieutenant", "Gold"},
		{demoDirForIndex(1), demoXUIDForIndex(1), "DemoPlayer2", 240, "Captain", tierNamePlatinum},
		{demoDirForIndex(2), demoXUIDForIndex(2), "DemoPlayer3", 120, "Sergeant", "Silver"},
	}
	for i, spec := range specs {
		// Sous-ensemble de matchs du joueur : principal = tous ; coéquipiers = escouade.
		matches := plan
		if i > 0 {
			matches = filterSquadMatches(plan)
		}
		path := filepath.Join(outDir, "players", spec.dir, "stats.duckdb")
		if err := writeOnePlayer(ctx, path, spec, matches); err != nil {
			return i, fmt.Errorf("player %s: %w", spec.gamertag, err)
		}
	}
	return len(specs), nil
}

// filterSquadMatches retourne les matchs en escouade (coéquipiers présents).
func filterSquadMatches(plan []synthMatch) []synthMatch {
	var out []synthMatch
	for _, m := range plan {
		if m.squad {
			out = append(out, m)
		}
	}
	return out
}

// writeOnePlayer crée + peuple une player DB.
func writeOnePlayer(ctx context.Context, path string, spec synthPlayerSpec, matches []synthMatch) error {
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
	// Aucune table pré-provisionnée à la main ici : depuis le lot « autorité de schéma
	// unique » (2026-08-05), la chaîne de migrations crée l'INTÉGRALITÉ du schéma player
	// — personal_score_awards et player_csr_snapshots (+ vues _latest) incluses, via
	// create_personal_score_awards_player_v1 / create_player_csr_snapshots_player_v1.
	// La copie de DDL qui vivait ici (3e exemplaire) a donc été supprimée.
	if err := migration.RunForTitleDB(db, titlePkg.DefaultSlug, migration.TargetPlayer); err != nil {
		return fmt.Errorf("migrations player: %w", err)
	}

	// INSERT pur : la player DB est fraîche (removeDuckDBForFreshWrite ci-dessus +
	// migrations qui ne seedent jamais la clé 'xuid') → aucune collision possible.
	// Surtout, INSERT OR REPLACE est un pattern ART interdit (garde-rail
	// TestNoARTPatternsOnProtectedTables, scan file-level de internal/ops/) :
	// ce fichier écrit aussi match_skill_rank /
	// player_csr_snapshots / player_match_enrichment (tables protégées).
	if _, err := db.ExecContext(ctx,
		`INSERT INTO sync_meta (key, value) VALUES ('xuid', ?)`, spec.xuid); err != nil {
		return fmt.Errorf("sync_meta: %w", err)
	}
	if err := insertCareerProgression(ctx, db, spec); err != nil {
		return err
	}
	if err := insertPlayerSessions(ctx, db, matches); err != nil {
		return err
	}
	for _, m := range matches {
		p, ok := participantFor(m, spec.xuid)
		if !ok {
			continue
		}
		if err := insertEnrichment(ctx, db, m, p, spec); err != nil {
			return err
		}
		if err := insertPlayerSkillRank(ctx, db, m); err != nil {
			return err
		}
	}
	if err := insertPlayerCitations(ctx, db, matches); err != nil {
		return err
	}
	return insertPlayerCSRSnapshot(ctx, db, spec)
}

// participantFor retourne l'entrée participant d'un joueur dans un match.
func participantFor(m synthMatch, xuid string) (synthParticipant, bool) {
	for _, p := range buildMatchParticipants(m) {
		if p.xuid == xuid {
			return p, true
		}
	}
	return synthParticipant{}, false
}

// insertCareerProgression écrit l'identité Spartan (rang, XP, images non vides pour la
// bannière home). recorded_at = ancre (identité la plus récente).
func insertCareerProgression(ctx context.Context, db *sql.DB, spec synthPlayerSpec) error {
	base := "https://blobs-infiniteugc.svc.halowaypoint.com/demo/" + spec.dir + "/"
	_, err := db.ExecContext(ctx, `
		INSERT INTO career_progression
			(xuid, rank, rank_name, rank_tier, current_xp, xp_for_next_rank, xp_total,
			 is_max_rank, spartan_id, banner_image_url, emblem_image_url, backdrop_image_url, recorded_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, FALSE, 'DEMO', ?, ?, ?, ?)`,
		spec.xuid, spec.rank, spec.rankName, spec.rankTier,
		1_500_000, 2_400_000, 5_200_000+spec.rank*10_000,
		base+"nameplate.png", base+"emblem.png", base+"backdrop.png", synthAnchor)
	return err
}

// insertPlayerSessions écrit une ligne sessions par session distincte du joueur.
func insertPlayerSessions(ctx context.Context, db *sql.DB, matches []synthMatch) error {
	seen := map[int]bool{}
	counts := map[int]int{}
	starts := map[int]synthMatch{}
	for _, m := range matches {
		counts[m.sessionOrder]++
		if _, ok := starts[m.sessionOrder]; !ok {
			starts[m.sessionOrder] = m
		}
	}
	for _, m := range matches {
		if seen[m.sessionOrder] {
			continue
		}
		seen[m.sessionOrder] = true
		first := starts[m.sessionOrder]
		if _, err := db.ExecContext(ctx, `
			INSERT OR IGNORE INTO sessions (session_id, label, start_time, end_time, match_count)
			VALUES (?, ?, ?, ?, ?)`,
			m.sessionOrder, m.sessionLabel, first.start,
			first.start.Add(time.Duration(counts[m.sessionOrder])*12*time.Minute), counts[m.sessionOrder]); err != nil {
			return fmt.Errorf("session %d: %w", m.sessionOrder, err)
		}
	}
	return nil
}

// insertEnrichment écrit player_match_enrichment (perf/session/engagement) — append-only,
// stage 'legacy', lecture via player_match_enrichment_latest.
func insertEnrichment(ctx context.Context, db *sql.DB, m synthMatch, p synthParticipant, spec synthPlayerSpec) error {
	perf := m.perfScore
	if spec.xuid != demoXUIDForIndex(0) {
		perf = 45 + float64(p.kills-p.deaths)*2.0
	}
	sig := ""
	if m.squad {
		sig = "demo-squad"
	}
	eng := 0.45 + float64(p.kills%10)/20.0
	_, err := db.ExecContext(ctx, `
		INSERT INTO player_match_enrichment
			(match_id, performance_score, session_id, session_label, is_with_friends,
			 teammates_signature, engagement_score, engagement_score_brut, engagement_score_confidence,
			 mode_category, engagement_pace_player, engagement_pace_team, engagement_pace_lobby,
			 engagement_player_activity, stage, written_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'high', ?, ?, ?, ?, ?, 'legacy', ?)`,
		m.matchID, perf, fmt.Sprintf("%d", m.sessionOrder), m.sessionLabel, m.squad,
		sig, eng, eng*100, m.mode.en,
		1.1+eng, 1.0+eng*0.9, 1.2, p.kills+p.assists, m.end)
	return err
}

// insertPlayerSkillRank écrit match_skill_rank : CSR pour un match classé, LUSR sinon.
// Lecture pics home via match_skill_rank_latest (classification is_ranked côté repo).
func insertPlayerSkillRank(ctx context.Context, db *sql.DB, m synthMatch) error {
	if m.pl.ranked {
		label := fmt.Sprintf("%s %d", m.csrTierFR, m.csrSubTier)
		_, err := db.ExecContext(ctx, `
			INSERT INTO match_skill_rank
				(match_id, rating_type, rating_value, tier, tier_fr, sub_tier, tier_label,
				 rating_delta, playlist_group, start_time, written_at)
			VALUES (?, 'CSR', ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			m.matchID, m.csrValue, m.csrTier, m.csrTierFR, m.csrSubTier, label,
			m.csrDelta, m.pl.group, m.start, m.end)
		return err
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO match_skill_rank
			(match_id, rating_type, rating_value, playlist_group, start_time, written_at)
		VALUES (?, 'LUSR', ?, ?, ?, ?)`,
		m.matchID, m.lusrValue, m.pl.group, m.start, m.end)
	return err
}

// insertPlayerCitations écrit match_citations (append-only) — norms présents dans
// citation_mappings (seed Go) pour un affichage résolu.
func insertPlayerCitations(ctx context.Context, db *sql.DB, matches []synthMatch) error {
	for _, m := range matches {
		n := 1 + m.idx%3
		for i := 0; i < n; i++ {
			norm := synthCitations[(m.idx+i)%len(synthCitations)]
			// written_at ancré sur synthAnchor : la table est append-only (DEFAULT
			// now() posé par la migration), sans valeur explicite chaque run diverge.
			if _, err := db.ExecContext(ctx,
				`INSERT OR IGNORE INTO match_citations (match_id, citation_name_norm, value, written_at) VALUES (?, ?, ?, ?)`,
				m.matchID, norm, 1+i, synthAnchor); err != nil {
				return fmt.Errorf("citation %s/%s: %w", m.matchID, norm, err)
			}
		}
	}
	return nil
}

// insertPlayerCSRSnapshot écrit un snapshot CSR alltime (playlist Arène) — source du
// pic CSR home (loadCSRAlltimePeak lit player_csr_snapshots.alltime_*).
func insertPlayerCSRSnapshot(ctx context.Context, db *sql.DB, spec synthPlayerSpec) error {
	// fetched_at ancré sur synthAnchor : sans valeur explicite il retombe sur son
	// DEFAULT CURRENT_TIMESTAMP → player DBs différentes à chaque run (déterminisme
	// cassé, cf. TestSeedDemoSynthetic_Deterministic).
	_, err := db.ExecContext(ctx, `
		INSERT INTO player_csr_snapshots
			(playlist_id, playlist_name, queue, input, season_id,
			 current_value, current_tier, current_sub_tier, current_measurement_remaining,
			 season_value, season_tier, season_sub_tier,
			 alltime_value, alltime_tier, alltime_sub_tier, fetched_at, written_at)
		VALUES ('demo-ranked-arena', 'Arène classée', 'open', 'crossplay', ?,
			1420, ?, 1, 0, 1420, ?, 1, 1460, ?, 1, ?, ?)`,
		synthDemoSeason, spec.rankTier, spec.rankTier, spec.rankTier, synthAnchor, synthAnchor)
	return err
}
