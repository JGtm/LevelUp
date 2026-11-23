// Package v2 — retry.go : retenue, par identifiant, des matchs abandonnés depuis assez longtemps.
//
// Un match abandonné par le rattrapage (maxCatchUpAttempts passes en échec) n'est pas perdu : il est
// retenté une fois historystate.AbandonRetryAfter écoulé, tentatives remises à zéro. La liste de l'API
// ne suffit pas à le retrouver : à l'historique complet, un delta ne descend pas jusqu'à lui. La découverte
// l'ajoute donc à la liste du joueur, comme un orphelin de l'ensemble connu, pour que les phases suivantes
// le récupèrent par match_id. Son issue se règle avec le reste du cycle :
//
//   - rangé : il devient connu, plus rien à faire ;
//   - introuvable (404/410) : définitif (history_skip.go) ;
//   - en échec, au rattrapage : le compte de passes recommence (trackFailures) ;
//   - en échec, à l'historique complet : abandonné de nouveau pour historystate.AbandonRetryAfter, sauf
//     panne de l'API (history_skip.go).
package v2

import (
	"context"
	"log/slog"
)

// abandonRetryPerCycle borne le nombre de matchs abandonnés retentés par cycle et par joueur : après une
// longue panne, des centaines d'abandons échoient ensemble, et leur récupération se répartit sur
// plusieurs cycles (les premiers rangés ou abandonnés de nouveau sortent de la liste des échus).
const abandonRetryPerCycle = 50

// appendRetries ajoute à la liste d'un joueur, à la suite, les abandons échus (au plus
// abandonRetryPerCycle) que la pagination n'a pas déjà listés et qui ne sont pas déjà connus.
func appendRetries(ctx context.Context, p PlayerProfile, unknown, due []string, known map[string]bool) []string {
	if len(due) == 0 {
		return unknown
	}
	listed := toSet(unknown)
	added := 0
	for _, id := range due {
		if added >= abandonRetryPerCycle {
			break
		}
		if known[id] || listed[id] {
			continue
		}
		unknown = append(unknown, id)
		added++
	}
	if added > 0 {
		slog.InfoContext(ctx, "sync.v2: matchs abandonnés retentés par identifiant",
			"event", "sync.v2.discovery.retry", "player", p.PlayerSlug, "due", len(due), "added", added)
	}
	return unknown
}
