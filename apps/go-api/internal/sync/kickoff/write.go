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

// Compteurs expvar (ADR 0009), par titre. Ils comptent des coups d'envoi RÉSOLUS ET RANGÉS : un plan
// qui change quelque chose et que le persisteur a écrit (publication d'un rejeu, rattrapage), ou une
// résolution à l'insertion d'un match de la synchronisation ou de l'import ([ObserveResolved]), comptée
// AVANT l'insertion — la file d'écriture l'acquitte plus tard, donc une insertion qui échoue est comptée
// et le match revient au cycle suivant. Une simulation, ou un rattrapage qui retrouve ce que
// la base porte déjà, ne les touche pas.
const (
	// prefixeSource + source + "_total" : coups d'envoi résolus et rangés (plan écrit, ou résolution à
	// l'insertion d'un match), par source retenue.
	prefixeSource = "kickoff_source_"
	// prefixeEstimation + issue + "_total" : issues de l'estimation des coups d'envoi résolus et rangés.
	prefixeEstimation = "kickoff_estimation_"
	// CompteurRejeuHorsBornes : coups d'envoi résolus et rangés dont le rejeu publiait une valeur hors de la
	// fenêtre du match (refusée, source suivante).
	CompteurRejeuHorsBornes = "kickoff_film_hors_bornes_total"
	// CompteurEcrits : matchs dont le coup d'envoi ou un temps de jeu a été réécrit.
	CompteurEcrits = "kickoff_matchs_ecrits_total"
	// CompteurEchecs : matchs dont l'écriture a échoué (retentés au passage suivant).
	CompteurEchecs = "kickoff_matchs_echecs_total"
	// CompteurArtefactsIllisibles : rejeux présents mais illisibles (rien de neuf n'en est tiré).
	CompteurArtefactsIllisibles = "kickoff_artefacts_illisibles_total"
	// CompteurAbsentsDuRegistre : rejeux ou matchs du lot sans ligne au registre (rien n'est créé).
	CompteurAbsentsDuRegistre = "kickoff_absents_du_registre_total"
)

// sourceLabel nomme une source dans un compteur (la source inconnue est vide).
func sourceLabel(s timeline.KickoffSource) string {
	if s == timeline.KickoffSourceUnknown {
		return "inconnue"
	}
	return string(s)
}

// ObserveResolved journalise et compte un coup d'envoi RÉSOLU à l'insertion d'un match (synchronisation,
// import) : la source retenue, l'issue de l'estimation quand elle a été jouée (une seule arrivée,
// arrivées dispersées, rejet) et le refus d'un coup d'envoi du rejeu hors de la fenêtre du match. Le
// compte est fait à la résolution, AVANT l'insertion : elle part par une file qui l'acquitte plus tard,
// et l'appelant ne connaît pas son issue. Une insertion qui échoue est donc comptée, puis le match se
// résout de nouveau (et se recompte) au cycle suivant.
func ObserveResolved(ctx context.Context, titleSlug, matchID string, k timeline.Kickoff) {
	countKickoff(titleSlug, k)
	logResolution(ctx, titleSlug, matchID, k)
}

// countKickoff compte un coup d'envoi résolu et rangé : après l'écriture d'un plan ([Persist]), ou à la
// résolution d'un match à l'insertion ([ObserveResolved], avant elle ; cf. les compteurs ci-dessus).
func countKickoff(titleSlug string, k timeline.Kickoff) {
	observability.IncCounterT(titleSlug, prefixeSource+sourceLabel(k.Source)+"_total")
	if k.Estimate != "" {
		observability.IncCounterT(titleSlug, prefixeEstimation+string(k.Estimate)+"_total")
	}
	if k.FilmOutOfBounds {
		observability.IncCounterT(titleSlug, CompteurRejeuHorsBornes)
	}
}

// logResolution journalise une résolution, écrite ou non ; un coup d'envoi du rejeu hors de la
// fenêtre du match se dit en avertissement.
func logResolution(ctx context.Context, titleSlug, matchID string, k timeline.Kickoff) {
	if k.FilmOutOfBounds {
		slog.WarnContext(ctx, "coup d'envoi : le rejeu publie une valeur hors de la fenêtre du match, refusée",
			"titleSlug", titleSlug, "match_id", matchID, "source_retenue", sourceLabel(k.Source))
	}
	slog.DebugContext(ctx, "coup d'envoi résolu", "titleSlug", titleSlug, "match_id", matchID,
		"source", sourceLabel(k.Source), "estimation", string(k.Estimate), "decalage_ms", k.OffsetMs)
}

// Prepare lit l'état rangé des matchs de films et résout leur coup d'envoi. films associe à chaque
// match ce que son rejeu dit. Un match absent du registre est rendu dans missing, sans plan.
func Prepare(ctx context.Context, db *sql.DB, titleSlug string, films map[string]Film) (plans []Plan, missing []string, err error) {
	ids := make([]string, 0, len(films))
	for id := range films {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return prepareIDs(ctx, db, titleSlug, ids, films)
}

// prepareIDs résout les matchs ids, dans cet ordre.
func prepareIDs(ctx context.Context, q queryer, titleSlug string, ids []string, films map[string]Film) ([]Plan, []string, error) {
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
		logResolution(ctx, titleSlug, id, p.Kickoff)
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
		countKickoff(titleSlug, plan.Kickoff)
		written++
	}
	observability.AddIntT(titleSlug, CompteurEcrits, int64(written))
	observability.AddIntT(titleSlug, CompteurEchecs, int64(len(failed)))
	return written, failed
}
