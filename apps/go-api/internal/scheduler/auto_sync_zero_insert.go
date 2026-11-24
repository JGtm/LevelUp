package scheduler

// auto_sync_zero_insert.go — le compteur « zero-insert » et son alerte.
//
// Un cycle qui n'insère aucun match n'est pas forcément un cycle sans nouveau match : le déclencheur de fin
// de partie du suivi en direct (ou un import) range le match avant que le cycle passe, et le cycle n'a plus
// rien à insérer. Compter ces cycles comme des échecs faisait sonner l'alerte en production chez un joueur
// qui jouait. Le compteur lit donc aussi le dernier match connu du joueur (port.LastMatchReader, lecture sous
// platform/duckdb) : un match est arrivé si le cycle en a inséré, OU si ce dernier match a avancé depuis le
// cycle précédent, par quelque voie que ce soit. L'alerte ne sonne que si aucun match n'est arrivé sur la
// fenêtre, une fois au franchissement du seuil puis une fois tous les ZeroInsertRewarnEvery cycles.

import (
	"context"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// ZeroInsertRewarnEvery : nombre de cycles sans match, après le franchissement du seuil, entre deux
// rappels de l'alerte « zero-insert prolongé ». Un joueur qui ne joue pas n'est pas une panne : l'alerte
// ne se répète pas à chaque cycle, elle se rappelle à intervalle long.
const ZeroInsertRewarnEvery = 48

// WithLastMatchReader branche la lecture du dernier match connu d'un joueur, dont le compteur
// zero-insert se sert pour voir les matchs rangés par une autre voie que le cycle. Sans lecteur, le
// compteur ne suit que les matchs insérés par le cycle.
func (s *AutoSyncScheduler) WithLastMatchReader(r port.LastMatchReader) *AutoSyncScheduler {
	s.lastMatch = r
	return s
}

// lastMatchOf rend le dernier match connu du joueur ; faux si le lecteur n'est pas branché, si le
// joueur n'a pas de xuid ni de match, ou si la lecture échoue (journalisée : le compteur retombe alors
// sur la règle du cycle seul).
func (s *AutoSyncScheduler) lastMatchOf(ctx context.Context, p domain.PlayerSummary) (time.Time, bool) {
	if s.lastMatch == nil || p.XUID == "" {
		return time.Time{}, false
	}
	slug := resolveTitleSlug(p)
	found, err := s.lastMatch.LastMatchByXUID(ctx, s.cfg.PathResolver().SharedDBPath(slug), slug, []string{p.XUID})
	if err != nil {
		slog.DebugContext(ctx, "auto_sync: zero-insert — dernier match connu illisible, le compteur ne suit que les matchs du cycle",
			"gamertag", p.Gamertag, "title_slug", slug, "err", err)
		return time.Time{}, false
	}
	last, ok := found[p.XUID]
	return last, ok
}

// countZeroInsert met à jour le compteur du joueur après un cycle réussi : remis à zéro si un match est
// arrivé (inséré par le cycle, ou dernier match connu avancé depuis le cycle précédent), incrémenté sinon,
// avec l'alerte au seuil. prevLast est le dernier match connu au cycle précédent (zéro : inconnu, et alors
// seuls les matchs du cycle comptent).
func (s *AutoSyncScheduler) countZeroInsert(
	ctx context.Context, p domain.PlayerSummary, detail *PlayerOutcomeDetail, prevZero int, prevLast time.Time, inserted int,
) {
	last, known := s.lastMatchOf(ctx, p)
	if known {
		detail.LastMatchAt = last
	}
	if inserted > 0 || (known && !prevLast.IsZero() && last.After(prevLast)) {
		detail.ConsecutiveZeroInserts = 0
		return
	}
	detail.ConsecutiveZeroInserts = prevZero + 1
	if !zeroInsertAlertDue(detail.ConsecutiveZeroInserts) {
		return
	}
	slog.WarnContext(ctx, "auto_sync: zero-insert prolongé — sync delta réussie mais 0 nouveau match, par aucune voie, sur N cycles consécutifs",
		"gamertag", p.Gamertag,
		"xuid", p.XUID,
		"consecutive_zero_inserts", detail.ConsecutiveZeroInserts,
		"threshold", ConsecutiveZeroInsertWarnThreshold,
		"rappel_tous_les_cycles", ZeroInsertRewarnEvery,
		"hint", "vérifier endpoint Halo + format URL + token resolved (probe /api/v1/_diag/auto-sync/probe)",
	)
}

// zeroInsertAlertDue dit si le compteur appelle l'alerte : au franchissement du seuil, puis tous les
// ZeroInsertRewarnEvery cycles.
func zeroInsertAlertDue(count int) bool {
	if count < ConsecutiveZeroInsertWarnThreshold {
		return false
	}
	return (count-ConsecutiveZeroInsertWarnThreshold)%ZeroInsertRewarnEvery == 0
}
