package prestige

import (
	"strings"

	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/prestige"
)

// prestige_player_helpers.go — fonctions utilitaires pour les repos Prestige joueur.
//
// Séparées du fichier principal pour rester sous le seuil 500L de CLAUDE.md
// et pour permettre des tests unitaires de buildChallengeListQuery sans duckdb.DB.

// challengeSelectColumns liste les colonnes de la table challenge dans
// l'ordre attendu par scanChallenge. Centralise la requête pour éviter
// les divergences entre Get et List.
const challengeSelectColumns = `
	SELECT id, user_id, title_slug,
	       COALESCE(arc_id, ''), COALESCE(position, 0),
	       COALESCE(template_id, ''),
	       metric, target, COALESCE(target_per_member, 0),
	       window_type, COALESCE(window_value, ''),
	       cadence, eval_type, mode, COALESCE(tier, ''), data_tier,
	       COALESCE(label, ''), status,
	       created_at, committed_at, completed_at, expired_at, abandoned_at,
	       last_palier_recompute_at, is_private, COALESCE(source, '')
	FROM challenge`

// scanChallenge lit une ligne de la table challenge.
func scanChallenge(row duckdb.RowScanner) (prestige.Challenge, error) {
	var c prestige.Challenge
	var windowType, cadence, evalType, mode, tier, dataTier, status string

	err := row.Scan(
		&c.ID, &c.UserID, &c.TitleSlug,
		&c.ArcID, &c.Position,
		&c.TemplateID,
		&c.Metric, &c.Target, &c.TargetPerMember,
		&windowType, &c.WindowValue,
		&cadence, &evalType, &mode, &tier, &dataTier,
		&c.Label, &status,
		&c.CreatedAt, &c.CommittedAt, &c.CompletedAt, &c.ExpiredAt, &c.AbandonedAt,
		&c.LastPalierRecomputeAt, &c.IsPrivate, &c.Source,
	)
	if err != nil {
		return prestige.Challenge{}, err
	}
	c.WindowType = prestige.WindowType(windowType)
	c.Cadence = prestige.Cadence(cadence)
	c.EvalType = prestige.EvalType(evalType)
	c.Mode = prestige.ChallengeMode(mode)
	c.Tier = prestige.Tier(tier)
	c.DataTier = prestige.DataTier(dataTier)
	c.Status = prestige.ChallengeStatus(status)
	return c, nil
}

// scanArc lit une ligne de la table arc.
func scanArc(row duckdb.RowScanner) (prestige.Arc, error) {
	var a prestige.Arc
	err := row.Scan(&a.ID, &a.UserID, &a.TitleSlug, &a.Title, &a.Description,
		&a.IsPreset, &a.PresetID, &a.CreatedAt, &a.CompletedAt)
	return a, err
}

// buildChallengeListQuery construit la requête + arguments selon le filtre.
//
// Toujours scope par user_id + title_slug si fournis (cas typique d'usage).
// Les autres filtres sont optionnels.
func buildChallengeListQuery(f prestige.ChallengeFilter) (string, []any) {
	var conds []string
	var args []any

	if f.UserID != "" {
		conds = append(conds, "user_id = ?")
		args = append(args, f.UserID)
	}
	if f.TitleSlug != "" {
		conds = append(conds, "title_slug = ?")
		args = append(args, f.TitleSlug)
	}
	if f.Status != nil {
		conds = append(conds, "status = ?")
		args = append(args, string(*f.Status))
	}
	if f.ArcID != nil {
		conds = append(conds, "arc_id = ?")
		args = append(args, *f.ArcID)
	}
	if f.Mode != nil {
		conds = append(conds, "mode = ?")
		args = append(args, string(*f.Mode))
	}
	if f.Metric != nil {
		conds = append(conds, "metric = ?")
		args = append(args, *f.Metric)
	}

	q := challengeSelectColumns
	if len(conds) > 0 {
		q += " WHERE " + strings.Join(conds, " AND ")
	}
	q += " ORDER BY created_at DESC"
	if f.Limit > 0 {
		q += " LIMIT ?"
		args = append(args, f.Limit)
	}
	return q, args
}

// timestampColumnFor retourne le nom de la colonne timestamp dédiée à un statut.
func timestampColumnFor(status prestige.ChallengeStatus) string {
	switch status {
	case prestige.StatusCompleted:
		return "completed_at"
	case prestige.StatusExpired:
		return "expired_at"
	case prestige.StatusAbandoned:
		return "abandoned_at"
	case prestige.StatusActive:
		return "committed_at"
	}
	return ""
}
