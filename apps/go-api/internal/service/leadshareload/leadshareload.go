// Package leadshareload héberge la LECTURE de la Mainmise commune aux services qui la servent
// (vue du match, page Sessions, Séries temporelles, Escouade). Extrait pour être appelé sans
// duplication par `package service` ET par `service/teammates`, qui ne peut pas importer son
// parent.
//
// Un seul point décide de la dégradation : la lecture en échec est journalisée avant de dégrader,
// sauf une capability absente (`games.ErrCapabilityNotSupported`), qui est le cas normal d'un titre
// sans la mesure ; dans les deux cas l'appelant ne pose rien, jamais un zéro.
package leadshareload

import (
	"context"
	"errors"
	"log/slog"

	"levelup/go-api/internal/analysis/leadshare"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// Load charge la Mainmise de l'équipe de `xuid` sur `ids` en UNE lecture. `scope` nomme l'appelant dans
// le journal quand la lecture échoue (hors capability absente). Rend faux quand l'appelant ne doit rien
// poser : lecteur nil, joueur ou liste vide, lecture en échec.
func Load(
	ctx context.Context, repo port.LeadShareRepository, ids []string, xuid, scope string,
) (map[string]leadshare.PlayerShare, bool) {
	if repo == nil || xuid == "" || len(ids) == 0 {
		return nil, false
	}
	got, err := repo.LoadLeadShare(ctx, ids, xuid)
	if err != nil {
		if !errors.Is(err, games.ErrCapabilityNotSupported) {
			slog.WarnContext(ctx, "mainmise illisible, réponse servie sans la mesure", "scope", scope, "matchs", len(ids), "err", err)
		}
		return nil, false
	}
	return got, true
}

// Pointers rend la part et les retournements d'un match mesuré sous la forme des deux pointeurs que
// portent les champs `lead_share` et `lead_changes` : un pointeur nil dit « non mesuré », jamais un
// zéro. Les pointeurs ne partagent rien avec `v`.
func Pointers(v leadshare.PlayerShare) (*float64, *int) {
	share, changes := v.Share, v.Changes
	return &share, &changes
}
