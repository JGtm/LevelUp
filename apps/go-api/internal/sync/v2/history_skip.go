// Package v2 — history_skip.go : rangement, après la persistance du cycle, des matchs que la découverte
// ne demande plus (historystate.SkipSetKey).
//
// Un match entre dans l'ensemble quand l'API le rend introuvable (404/410) à la récupération, ou quand le
// rattrapage l'abandonne (maxCatchUpAttempts passes en échec). Il y reste après la fin de l'historique :
// il ne devient jamais « connu » (rien n'est rangé pour lui), donc sans l'ensemble chaque delta le
// relirait et le redemanderait tant qu'il est proche du haut de la liste, et le compterait inséré.
// Joueur à l'historique complet comme au rattrapage : l'écriture se fait sous le bail d'écriture du
// joueur (WithHistoryWriter), seulement quand il y a du neuf.
package v2

import (
	"context"
	"database/sql"
	"log/slog"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/sync/historystate"
)

// counterHistorySkipped : matchs ajoutés à l'ensemble des matchs à ne plus demander.
const counterHistorySkipped = "sync_v2_history_skipped_total"

// freshSkips rend, dans l'ordre et sans doublon, les matchs de `notFound` et d'`abandoned` qui ne sont
// pas encore dans l'ensemble rangé `stored`.
func freshSkips(stored map[string]bool, notFound, abandoned []string) []historystate.Skip {
	var out []historystate.Skip
	seen := map[string]bool{}
	add := func(ids []string, why historystate.SkipReason) {
		for _, id := range ids {
			if stored[id] || seen[id] {
				continue
			}
			seen[id] = true
			out = append(out, historystate.Skip{MatchID: id, Reason: why})
		}
	}
	add(notFound, historystate.SkipNotFound)
	add(abandoned, historystate.SkipAbandoned)
	return out
}

// abandonedIDs rend les matchs abandonnés d'une liste d'échecs, dans l'ordre de la liste.
func abandonedIDs(failures []historystate.Failure) []string {
	var out []string
	for _, f := range failures {
		if isAbandoned(f.Attempts) {
			out = append(out, f.MatchID)
		}
	}
	return out
}

// openHistoryDB ouvre la base joueur en écriture sous le bail du joueur, pour au plus historyWriteWait.
// ok=false : rien n'est ouvert (écrivain non câblé ou base indisponible, journalisé). done libère le
// bail et le délai ; il est non nul seulement quand ok est vrai.
func (o *CycleOrchestratorImpl) openHistoryDB(ctx context.Context, p PlayerProfile) (wctx context.Context, db *sql.DB, done func(), ok bool) {
	if o.historyWriter == nil {
		slog.WarnContext(ctx, "sync.v2: rattrapage d'historique sans écrivain câblé — rien n'est rangé",
			"event", "sync.v2.history.no_writer", "player", p.PlayerSlug)
		return ctx, nil, nil, false
	}
	wctx, cancel := context.WithTimeout(ctx, historyWriteWait)
	db, release, err := o.historyWriter(wctx, p.Gamertag)
	if err != nil {
		cancel()
		slog.WarnContext(ctx, "sync.v2: base joueur indisponible en écriture — état de l'historique non rangé",
			"event", "sync.v2.history.write_unavailable", "player", p.PlayerSlug, "err", err)
		return ctx, nil, nil, false
	}
	return wctx, db, func() { release(); cancel() }, true
}

// addSkips range des matchs à ne plus demander, journalisés et comptés. Rend vrai quand ils sont rangés.
func addSkips(ctx context.Context, db *sql.DB, p PlayerProfile, adds []historystate.Skip) bool {
	if !historystate.AddSkips(ctx, db, adds) {
		return false
	}
	notFound := 0
	for _, s := range adds {
		if s.Reason == historystate.SkipNotFound {
			notFound++
		}
	}
	observability.AddIntT(ctxkeys.TitleSlug(ctx), counterHistorySkipped, int64(len(adds)))
	slog.InfoContext(ctx, "sync.v2: matchs à ne plus demander rangés",
		"event", "sync.v2.history.skipped", "player", p.PlayerSlug, "added", len(adds),
		"not_found", notFound, "abandoned", len(adds)-notFound)
	return true
}

// settleSkips range les matchs introuvables d'un joueur à l'historique COMPLET (le delta). Au
// rattrapage, settlePlayerHistory range les siens avec le reste de l'état. Rien de neuf : aucune
// ouverture de la base en écriture. Un rangement en échec est journalisé : le match sera redemandé au
// cycle suivant, puis rangé.
func (o *CycleOrchestratorImpl) settleSkips(ctx context.Context, p PlayerProfile, ph PlayerHistory, in settleInput) {
	adds := freshSkips(ph.Skip, notFoundListed(in.listed, in.fetchErrs), nil)
	if len(adds) == 0 {
		return
	}
	wctx, db, done, ok := o.openHistoryDB(ctx, p)
	if !ok {
		return
	}
	defer done()
	addSkips(wctx, db, p, adds)
}
