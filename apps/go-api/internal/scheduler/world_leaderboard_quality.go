// Package scheduler — world_leaderboard_quality.go : garde-fou de QUALITÉ du cron
// classement mondial (world_leaderboard_cron.go). Décide, playlist par playlist, si
// un lot fraîchement scrapé a le droit de remplacer le lot actuellement servi.
//
// Pourquoi ce garde-fou existe (incident du 2026-07-07) : la vue
// world_csr_leaderboard_latest sert le DERNIER batch par (titre, saison, playlist).
// Un cycle dégradé — page Waypoint à moitié rendue, pagination coupée, markup changé
// — persiste donc un lot pauvre qui MASQUE instantanément le lot sain précédent :
// 86 lignes sans aucun xuid ont ainsi recouvert 200 lignes dont 100 % de xuid. Rien
// n'est perdu au sens append-only (les anciennes lignes restent en table), mais plus
// rien ne les sert, et ces snapshots sont la SEULE archive du classement mondial
// (Halo Waypoint retire les saisons passées : csrseason13-2 a disparu du site).
//
// Règle appliquée : décision D1, définie UNE SEULE FOIS dans
// duckdb.DegradedBatchReason (volume effondré ou identification effondrée) — ce
// fichier n'en est qu'un appelant. L'autre appelant est le CLI -restore-best, qui
// s'en sert dans l'autre sens (le lot servi mérite-t-il d'être remplacé par un
// meilleur lot historique ?) : une seule règle, deux usages, aucune redéfinition.
//
// Un refus est un SKIP de la playlist, jamais une erreur de cycle : les autres
// playlists sont persistées normalement et le lot servi reste en place. Le plancher
// absolu minEntries (cron) s'applique en amont et reste indépendant de cette règle.
//
// Mais un garde-fou qui refuse SANS FIN est lui-même une panne : la playlist cesse
// d'être rafraîchie et le classement servi vieillit en silence. Les refus consécutifs
// sont donc comptés par playlist (refusalStreaks) et le log passe en ERROR au-delà de
// maxSilentBatchRefusals — même logique que l'escalade de la découverte de saison.
package scheduler

import (
	"context"
	"log/slog"
	"sync"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/observability/logging"
	"levelup/go-api/internal/platform/duckdb"
)

const (
	// worldBatchRefusedMetric : compteur expvar des lots refusés (toutes causes),
	// exposé sur /debug/vars → levelup.<nom>. Un compteur qui grimpe = Waypoint rend
	// des pages dégradées de façon répétée ; le classement servi, lui, reste sain.
	worldBatchRefusedMetric = "world_leaderboard_batch_refused_total"
	// maxSilentBatchRefusals : refus CONSÉCUTIFS tolérés en WARN pour une même
	// playlist. Au-delà (strictement plus), le log passe en ERROR : un refus isolé
	// protège le classement comme prévu, mais une SÉRIE veut dire que la playlist
	// n'est plus rafraîchie du tout — le garde-fou est alors devenu la panne, et
	// personne ne le verrait dans le flot des WARN quotidiens.
	maxSilentBatchRefusals = 3
)

// refusalStreaks compte les refus CONSÉCUTIFS par playlist. Process-local, protégé
// par mutex : les playlists sont traitées séquentiellement aujourd'hui, mais RunOnce
// est appelable hors du ticker (endpoint admin, tests) et rien ne doit dépendre de
// cette hypothèse.
type refusalStreaks struct {
	mu     sync.Mutex
	counts map[string]int
}

// record incrémente la série de la playlist et retourne sa nouvelle valeur.
func (r *refusalStreaks) record(playlist string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.counts == nil {
		r.counts = make(map[string]int)
	}
	r.counts[playlist]++
	return r.counts[playlist]
}

// reset efface la série de la playlist (un lot accepté clôt la série en cours).
func (r *refusalStreaks) reset(playlist string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.counts, playlist)
}

// streak retourne la série courante de la playlist (0 si aucune).
func (r *refusalStreaks) streak(playlist string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.counts[playlist]
}

// batchIsDegraded indique si le lot candidat doit être REFUSÉ pour cette playlist.
// Lit la qualité du lot servi hors lease writer (reader RO court, acquis puis relâché
// par playlist : le scrape complet dure plusieurs minutes, on ne tient pas la shared
// DB ouverte pendant tout ce temps).
//
// FAIL-OPEN : si la qualité du lot servi est illisible (table absente sur une DB
// neuve, hoquet du reader), on logue un WARN et on N'APPLIQUE PAS le refus. Un
// problème de lecture ne doit jamais empêcher une première capture ni geler le
// classement — le risque inverse (laisser passer un lot dégradé) est réparable, celui
// de ne plus jamais rien persister ne l'est pas.
func (c *WorldLeaderboardCron) batchIsDegraded(
	ctx context.Context, titleSlug, season, playlist string, entries []domain.LeaderboardEntry,
) bool {
	served, ok, err := c.servedBatchStats(ctx, titleSlug, season, playlist)
	if err != nil {
		slog.WarnContext(ctx, "world_leaderboard_cron: qualité du lot servi illisible — garde-fou non appliqué (lot accepté)",
			"module", logging.ModuleLeaderboard, "titleSlug", titleSlug,
			"season", season, "playlist", playlist, "err", err)
		c.batchRefusals.reset(playlist)
		return false
	}
	if !ok {
		c.batchRefusals.reset(playlist)
		return false // première capture de cette playlist : rien à protéger.
	}
	candidate := duckdb.WorldCSRStatsOfEntries(entries)
	// c.limit = profondeur que CE cycle avait le droit de ramener : sans elle, un lot
	// servi plus profond (archive CLI) refuserait éternellement le top nominal du cron.
	reason := duckdb.DegradedBatchReason(served, candidate, c.limit)
	if reason == "" {
		c.batchRefusals.reset(playlist)
		return false
	}
	observability.IncCounter(worldBatchRefusedMetric)
	streak := c.batchRefusals.record(playlist)
	attrs := []any{
		"module", logging.ModuleLeaderboard, "titleSlug", titleSlug,
		"season", season, "playlist", playlist, "raison", reason,
		"servi_lignes", served.Rows, "servi_xuid", served.WithXUID,
		"candidat_lignes", candidate.Rows, "candidat_xuid", candidate.WithXUID,
		"profondeur_demandee", c.limit, "consecutive_refusals", streak,
	}
	if streak > maxSilentBatchRefusals {
		slog.ErrorContext(ctx, "world_leaderboard_cron: lot REFUSÉ depuis plusieurs cycles consécutifs — cette playlist n'est PLUS "+
			"rafraîchie (le garde-fou protège un classement qui vieillit ; vérifier que Halo Waypoint rend encore la page "+
			"complète, ou que le lot servi n'est pas une archive plus profonde que ce cycle)", attrs...)
		return true
	}
	slog.WarnContext(ctx, "world_leaderboard_cron: lot dégradé REFUSÉ — le classement servi est conservé (aucune écriture pour cette playlist)",
		attrs...)
	return true
}

// servedBatchStats lit la qualité du lot servi via un reader RO court (même discipline
// que snapshotIsFresh : jamais de lease writer pendant la phase de scrape).
func (c *WorldLeaderboardCron) servedBatchStats(
	ctx context.Context, titleSlug, season, playlist string,
) (duckdb.WorldCSRBatchStats, bool, error) {
	db, release, err := c.provider.Get(ctx)
	if err != nil {
		return duckdb.WorldCSRBatchStats{}, false, err
	}
	defer release()
	return duckdb.WorldCSRServedBatchStats(ctx, db, titleSlug, season, playlist)
}
