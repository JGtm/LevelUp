package scheduler

// data_health_kill_summaries.go — LA JAUGE DES RÉSUMÉS PAR MATCH DU JOURNAL (ADR 0039, décision 6).
//
// « Matchs dont la passe courante du journal des morts n'a pas de tête à jour » : aucune tête,
// une tête à une ancienne révision de la règle, ou une tête qui résume une autre passe que la
// courante. Un lecteur de résumés montre un tel match comme non mesuré, ou l'omet d'une somme :
// la jauge doit valoir 0 sur les bases servies, et `levelup backfill-match-summaries` la ramène
// à 0 (même texte SQL, `persist.CountPendingKillSummaries`).
//
// Elle se calcule ICI, dans le contrôle de santé périodique, et jamais au démarrage : la vue
// `_latest` du journal porte une fenêtre sur toute la table (plusieurs secondes sur le parc).
// Lecture seule : aucun écrivain n'est tenu.

import (
	"context"
	"expvar"
	"log/slog"

	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/platform/duckdb"
)

// killSummariesPendingGauge : jauge expvar (`/debug/vars`), une entrée par titre, valeur du
// dernier contrôle qui a pu MESURER ce titre. Une mesure en échec ne la touche pas : « non
// mesuré » n'est pas « sain ».
var killSummariesPendingGauge = expvar.NewMap("levelup.kill_summaries_pending")

// auditKillSummaries mesure la jauge pour un titre, la publie et l'ajoute à `res`. Un compte
// non nul est journalisé en WARN ; une sonde en échec compte dans `res.ProbeErrors`.
func (s *HealthScheduler) auditKillSummaries(ctx context.Context, db *duckdb.DB, slug string, res *DataHealthCheckResult) {
	n, err := persist.CountPendingKillSummaries(ctx, db.SQLDb())
	if err != nil {
		slog.WarnContext(ctx, "data_health: sonde échouée", "probe", "kill_summaries_pending",
			"titleSlug", slug, "err", err)
		res.ProbeErrors++
		return
	}
	gauge := new(expvar.Int)
	gauge.Set(int64(n))
	killSummariesPendingGauge.Set(slug, gauge)
	res.KillSummariesPending += n
	if n > 0 {
		slog.WarnContext(ctx, "data_health: matchs dont la passe courante n'a pas de résumés à jour — "+
			"lancer `levelup backfill-match-summaries` (serveur arrêté)",
			"titleSlug", slug, "kill_summaries_pending", n)
	}
}
