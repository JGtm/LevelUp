// Package persist — player_persister.go : écriture INSERT-only atomique du
// sous-batch PlayerBatch dans stats.duckdb (DB d'un joueur).
//
// **Property clé — ajout facile d'enrichment** :
//
//   Pour ajouter un enrichment local à player_match_enrichment :
//
//     1. Migration DB → addColumnIfMissing(db, "player_match_enrichment", "X", "DOUBLE")
//     2. Champ pointer dans EnrichmentRow (rows.go) : `X *float64`
//     3. 1 if-block dans enrichmentFields() (ce fichier)
//
//   C'est tout. Pas de SQL à modifier (INSERT dynamique sur fields non-nil).
//
// Architecture identique à SharedPersister : 1 transaction par batch, INSERT-only,
// idempotence par EXISTS(player_match_enrichment WHERE match_id = ? AND stage='live').
//
// Append-only #23046 : player_match_enrichment est append-only (id PK + stage).
// Le collect live écrit UNE row stage='live' (baseline merge-on-read) ; l'ancre
// d'idempotence est donc ciblée stage='live' (pas match_id seul, sinon une row
// d'un autre stage — engagement/session post-sync — ferait skip tout le sous-batch).

package persist

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// PlayerPersister écrit le sous-batch PlayerBatch dans stats.duckdb du joueur.
type PlayerPersister struct {
	db txBeginner
}

// NewPlayerPersister construit un persister. `db` doit pointer vers la
// stats.duckdb d'UN joueur avec un write lease actif.
func NewPlayerPersister(db txBeginner) *PlayerPersister {
	return &PlayerPersister{db: db}
}

// Persist écrit le PlayerBatch en 1 transaction INSERT-only.
//
// Cas particuliers :
//
//   - batch == nil                            → error.
//   - batch.PlayerData.Enrichment == nil      → no-op.
//   - row stage='live' existe déjà pour ce match_id → skip (idempotent ACK).
//
// L'enrichment 'live' est l'ancre d'idempotence : si la row baseline existe déjà
// pour ce match_id, tout le sous-batch (skill_rank/lusr/citations/psa/career) est
// considéré déjà persisté. Cible stage='live' car la table est append-only et
// porte N rows par match (1 par stage post-sync).
func (p *PlayerPersister) Persist(ctx context.Context, batch *MatchBatch) error {
	if batch == nil {
		return errors.New("persist: PlayerPersister.Persist: batch nil")
	}
	pb := &batch.PlayerData
	if pb.Enrichment == nil {
		return nil
	}

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("persist: BeginTx player: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var exists bool
	if err := tx.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM player_match_enrichment WHERE match_id = ? AND stage = 'live')`,
		pb.Enrichment.MatchID,
	).Scan(&exists); err != nil {
		return fmt.Errorf("persist: check enrichment exists %s: %w", pb.Enrichment.MatchID, err)
	}
	if exists {
		return nil
	}

	if err := persistEnrichment(ctx, tx, pb.Enrichment); err != nil {
		return err
	}
	if err := persistSkillRank(ctx, tx, pb.SkillRank); err != nil {
		return err
	}
	if err := persistLUSRComponents(ctx, tx, pb.LUSRComponents); err != nil {
		return err
	}
	if err := persistCitations(ctx, tx, pb.Citations); err != nil {
		return err
	}
	if err := persistPersonalScoreAwards(ctx, tx, pb.PersonalScoreAwards); err != nil {
		return err
	}
	if err := persistCareerProgression(ctx, tx, pb.CareerProgression); err != nil {
		return err
	}
	// `Session` est hors scope du batch flow — cf. doc.go (sessions table
	// non peuplée par MatchBatch ; session_id reference dans enrichment).

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("persist: Commit player %s: %w", pb.Enrichment.MatchID, err)
	}
	return nil
}

// ─── INSERT dynamique sur player_match_enrichment ──────────────────────────

// fieldEntry est une paire (column_name, value) pour construire un INSERT
// dynamique. Le seul endroit qui doit lister les champs d'EnrichmentRow.
type fieldEntry struct {
	name  string
	value any
}

// enrichmentFields produit la liste des colonnes à INSERTer pour cette row
// (uniquement les champs pointer non-nil, plus le PK match_id).
//
// RECETTE 3 ÉTAPES pour ajouter un enrichment local (ADR 0019 + persist/doc.go) :
//  1. migration ALTER TABLE (nouvelle colonne) ;
//  2. champ pointer dans EnrichmentRow ;
//  3. if-block ci-dessous ajoutant le fieldEntry.
func enrichmentFields(row *EnrichmentRow) []fieldEntry {
	fields := []fieldEntry{
		{"match_id", row.MatchID},
	}
	if row.PerformanceScore != nil {
		fields = append(fields, fieldEntry{"performance_score", *row.PerformanceScore})
	}
	if row.PerformanceChain != nil {
		fields = append(fields, fieldEntry{"performance_chain", *row.PerformanceChain})
	}
	if row.DominanceFlag != nil {
		fields = append(fields, fieldEntry{"dominance_flag", *row.DominanceFlag})
	}
	if row.EngagementScore != nil {
		fields = append(fields, fieldEntry{"engagement_score", *row.EngagementScore})
	}
	if row.EngagementScoreBrut != nil {
		fields = append(fields, fieldEntry{"engagement_score_brut", *row.EngagementScoreBrut})
	}
	if row.EngagementScoreConfidence != nil {
		fields = append(fields, fieldEntry{"engagement_score_confidence", *row.EngagementScoreConfidence})
	}
	if row.EngagementPacePlayer != nil {
		fields = append(fields, fieldEntry{"engagement_pace_player", *row.EngagementPacePlayer})
	}
	if row.EngagementPaceTeam != nil {
		fields = append(fields, fieldEntry{"engagement_pace_team", *row.EngagementPaceTeam})
	}
	if row.EngagementPaceLobby != nil {
		fields = append(fields, fieldEntry{"engagement_pace_lobby", *row.EngagementPaceLobby})
	}
	if row.EngagementPlayerActivity != nil {
		fields = append(fields, fieldEntry{"engagement_player_activity", *row.EngagementPlayerActivity})
	}
	if row.ModeCategory != nil {
		fields = append(fields, fieldEntry{"mode_category", *row.ModeCategory})
	}
	if row.SessionID != nil {
		fields = append(fields, fieldEntry{"session_id", *row.SessionID})
	}
	if row.SessionLabel != nil {
		fields = append(fields, fieldEntry{"session_label", *row.SessionLabel})
	}
	if row.IsWithFriends != nil {
		fields = append(fields, fieldEntry{"is_with_friends", *row.IsWithFriends})
	}
	if row.TeammatesSignature != nil {
		fields = append(fields, fieldEntry{"teammates_signature", *row.TeammatesSignature})
	}
	if row.HadBotTeammate != nil {
		fields = append(fields, fieldEntry{"had_bot_teammate", *row.HadBotTeammate})
	}
	if row.UpdatedAt != nil {
		fields = append(fields, fieldEntry{"updated_at", *row.UpdatedAt})
	}
	return fields
}

func persistEnrichment(ctx context.Context, tx *sql.Tx, row *EnrichmentRow) error {
	if row == nil {
		return nil
	}
	fields := enrichmentFields(row)
	// Append-only #23046 : le collect live écrit UNE row baseline stage='live'
	// (multi-colonnes). Les writers post-sync owner-stage l'overrident par colonne
	// via la vue merge-on-read. Pas de split par stage ici (cf. décision design #1).
	fields = append(fields, fieldEntry{"stage", "live"})
	cols := make([]string, len(fields))
	placeholders := make([]string, len(fields))
	args := make([]any, len(fields))
	for i, f := range fields {
		cols[i] = f.name
		placeholders[i] = "?"
		args[i] = f.value
	}
	q := "INSERT INTO player_match_enrichment (" +
		strings.Join(cols, ", ") +
		") VALUES (" +
		strings.Join(placeholders, ", ") +
		")"
	if _, err := tx.ExecContext(ctx, q, args...); err != nil {
		return fmt.Errorf("persist: INSERT player_match_enrichment %s: %w", row.MatchID, err)
	}
	return nil
}

// ─── INSERT helpers — autres tables player ─────────────────────────────────

func persistSkillRank(ctx context.Context, tx *sql.Tx, row *SkillRankInsert) error {
	if row == nil {
		return nil
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO match_skill_rank (
			match_id, rating_type, rating_value, rating_deviation,
			tier, tier_fr, sub_tier, tier_label, rating_delta, playlist_group
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		row.MatchID, row.RatingType, row.RatingValue, row.RatingDeviation,
		row.Tier, row.TierFR, row.SubTier, row.TierLabel, row.RatingDelta, row.PlaylistGroup,
	)
	if err != nil {
		return fmt.Errorf("persist: INSERT match_skill_rank %s: %w", row.MatchID, err)
	}
	return nil
}

func persistLUSRComponents(ctx context.Context, tx *sql.Tx, rows []LUSRComponentInsert) error {
	if len(rows) == 0 {
		return nil
	}
	for _, c := range rows {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO lusr_component_history (match_id, component_name, value, weight)
			VALUES (?, ?, ?, ?)`,
			c.MatchID, c.ComponentName, c.Value, c.Weight,
		)
		if err != nil {
			return fmt.Errorf("persist: INSERT lusr_component_history %s/%s: %w",
				c.MatchID, c.ComponentName, err)
		}
	}
	return nil
}

func persistCitations(ctx context.Context, tx *sql.Tx, rows []CitationInsert) error {
	if len(rows) == 0 {
		return nil
	}
	// Append-only #23046 (Phase 2) : alloue UNE génération partagée par le batch
	// (match_citations_generation_seq) ; la vue match_citations_latest ne lit que
	// la génération MAX par match_id. Plus de PK composite ni ON CONFLICT.
	var gen int64
	if err := tx.QueryRowContext(ctx, `SELECT nextval('match_citations_generation_seq')`).Scan(&gen); err != nil {
		return fmt.Errorf("persist: match_citations generation: %w", err)
	}
	for _, c := range rows {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO match_citations (match_id, citation_name_norm, value, generation_id)
			VALUES (?, ?, ?, ?)`,
			c.MatchID, c.CitationNameNorm, c.Value, gen,
		)
		if err != nil {
			return fmt.Errorf("persist: INSERT match_citations %s/%s: %w",
				c.MatchID, c.CitationNameNorm, err)
		}
	}
	return nil
}

func persistPersonalScoreAwards(ctx context.Context, tx *sql.Tx, rows []PersonalScoreAwardInsert) error {
	if len(rows) == 0 {
		return nil
	}
	// Append-only #23046 (Phase 2) : alloue UNE génération partagée par le batch
	// (psa_generation_seq) ; la vue personal_score_awards_latest ne lit que la
	// génération MAX par (match_id,xuid). Sans ce marqueur, re-soumettre le même
	// match dupliquerait les awards (le persister fait déjà un INSERT pur).
	var gen int64
	if err := tx.QueryRowContext(ctx, `SELECT nextval('psa_generation_seq')`).Scan(&gen); err != nil {
		return fmt.Errorf("persist: psa generation: %w", err)
	}
	for _, a := range rows {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO personal_score_awards
				(match_id, xuid, award_name, award_category, award_count, award_score, generation_id)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			a.MatchID, a.XUID, a.AwardName, a.AwardCategory, a.AwardCount, a.AwardScore, gen,
		)
		if err != nil {
			return fmt.Errorf("persist: INSERT personal_score_awards %s/%s/%s: %w",
				a.MatchID, a.XUID, a.AwardName, err)
		}
	}
	return nil
}

func persistCareerProgression(ctx context.Context, tx *sql.Tx, row *CareerProgressionInsert) error {
	if row == nil {
		return nil
	}
	recorded := row.RecordedAt
	if recorded.IsZero() {
		recorded = time.Now().UTC()
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO career_progression (
			xuid, rank, rank_name, rank_tier,
			current_xp, xp_for_next_rank, xp_total, is_max_rank,
			adornment_path, spartan_id,
			banner_image_url, emblem_image_url, backdrop_image_url,
			recorded_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		row.XUID, row.Rank, row.RankName, row.RankTier,
		row.CurrentXP, row.XPForNextRank, row.XPTotal, row.IsMaxRank,
		row.AdornmentPath, row.SpartanID,
		row.BannerImageURL, row.EmblemImageURL, row.BackdropImageURL,
		recorded,
	)
	if err != nil {
		return fmt.Errorf("persist: INSERT career_progression %s: %w", row.XUID, err)
	}
	return nil
}
