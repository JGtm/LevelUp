package kickoff

// write.go — préparer (lire et résoudre) puis écrire les plans d'un lot de matchs, et observer
// chaque résolution (journal et compteurs : l'issue de l'estimation n'est rangée nulle part).

import (
	"context"
	"database/sql"
	"log/slog"
	"sort"

	"levelup/go-api/internal/analysis/timeline"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/persist"
)

// Compteurs expvar (ADR 0009), par titre.
const (
	// prefixeSource + source + "_total" : coups d'envoi résolus, par source retenue.
	prefixeSource = "kickoff_source_"
	// prefixeEstimation + issue + "_total" : issues de l'estimation, quand elle a été jouée.
	prefixeEstimation = "kickoff_estimation_"
	// CompteurEcrits : matchs dont le coup d'envoi ou un temps de jeu a été réécrit.
	CompteurEcrits = "kickoff_matchs_ecrits_total"
	// CompteurEchecs : matchs dont l'écriture a échoué (retentés au passage suivant).
	CompteurEchecs = "kickoff_matchs_echecs_total"
	// CompteurArtefactsIllisibles : rejeux présents mais illisibles, traités comme absents.
	CompteurArtefactsIllisibles = "kickoff_artefacts_illisibles_total"
)

// sourceLabel nomme une source dans un compteur (la source inconnue est vide).
func sourceLabel(s timeline.KickoffSource) string {
	if s == timeline.KickoffSourceUnknown {
		return "inconnue"
	}
	return string(s)
}

// Observe journalise et compte la résolution du coup d'envoi d'un match : la source retenue et,
// quand l'estimation a été jouée, son issue (une seule arrivée, arrivées dispersées, rejet).
func Observe(ctx context.Context, titleSlug, matchID string, k timeline.Kickoff) {
	observability.IncCounterT(titleSlug, prefixeSource+sourceLabel(k.Source)+"_total")
	if k.Estimate != "" {
		observability.IncCounterT(titleSlug, prefixeEstimation+string(k.Estimate)+"_total")
	}
	slog.DebugContext(ctx, "coup d'envoi résolu", "titleSlug", titleSlug, "match_id", matchID,
		"source", sourceLabel(k.Source), "estimation", string(k.Estimate), "decalage_ms", k.OffsetMs)
}

// Prepare lit l'état rangé des matchs de films et résout leur coup d'envoi. films associe à chaque
// match le coup d'envoi publié par son rejeu (nil : aucun). Un match absent du registre est rendu
// dans missing, sans plan.
func Prepare(ctx context.Context, db *sql.DB, titleSlug string, films map[string]*int64) (plans []Plan, missing []string, err error) {
	ids := make([]string, 0, len(films))
	for id := range films {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return prepareIDs(ctx, db, titleSlug, ids, films)
}

// prepareIDs résout les matchs ids, dans cet ordre.
func prepareIDs(ctx context.Context, q queryer, titleSlug string, ids []string, films map[string]*int64) ([]Plan, []string, error) {
	state, err := loadMatches(ctx, q, ids)
	if err != nil {
		return nil, nil, err
	}
	plans := make([]Plan, 0, len(ids))
	var missing []string
	for _, id := range ids {
		m, ok := state[id]
		if !ok {
			missing = append(missing, id)
			continue
		}
		p := Resolve(*m, films[id])
		Observe(ctx, titleSlug, id, p.Kickoff)
		plans = append(plans, p)
	}
	return plans, missing, nil
}

// Persist écrit les plans qui changent quelque chose, un match par transaction, sur db (qui porte
// le writer partagé). Rend le nombre de matchs écrits et les matchs en échec ; chaque échec est
// journalisé et compté, les autres matchs ne sont pas retenus.
func Persist(ctx context.Context, db *sql.DB, titleSlug string, plans []Plan) (written int, failed []string) {
	p := persist.NewKickoffPersister(db)
	for _, plan := range plans {
		if plan.Unchanged() {
			continue
		}
		if err := p.WriteMatch(ctx, plan.Write()); err != nil {
			slog.WarnContext(ctx, "coup d'envoi : écriture du match échouée, retentée au passage suivant",
				"titleSlug", titleSlug, "match_id", plan.MatchID, "err", err)
			failed = append(failed, plan.MatchID)
			continue
		}
		written++
	}
	observability.AddIntT(titleSlug, CompteurEcrits, int64(written))
	observability.AddIntT(titleSlug, CompteurEchecs, int64(len(failed)))
	return written, failed
}
