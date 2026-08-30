// Package killsourceload — LE chemin de chargement des kills par SOURCE DE
// DEGAT, partage par toutes les surfaces qui dessinent la « Repartition des frags ».
//
// POURQUOI UN FOYER UNIQUE. Cinq surfaces construisent le meme sunburst (Vue match,
// Synthese, Series temporelles, Sessions, Explorer — et l Escouade dans son propre
// paquet). Recopier cinq fois « valider les filtres, appeler le repo, avaler
// ErrCapabilityNotSupported, logguer le compteur » aurait garanti que les cinq copies
// divergent (regle CLAUDE.md n6 : a la 3e copie, on centralise). Il n y a donc qu une
// fonction, et les appelants ne portent qu un `if repo != nil`.
//
// CE QUE LA FONCTION AVALE, ET CE QU ELLE NE DOIT PAS AVALER.
// `games.ErrCapabilityNotSupported` n est PAS une erreur : c est un match (ou un scope)
// dont le film n a jamais ete decode — l etat nominal de 30 % du parc. Toute AUTRE erreur
// est loggee en ERROR avant la degradation : le sunburst restera juste (ces kills
// retombent dans « Non attribue »), mais silencieusement moins precis, et un silence ici
// se paierait en mesure fausse a l oeil du lecteur.
package killsourceload

import (
	"context"
	"errors"
	"log/slog"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// Load charge les kills agreges par classe de source de degat pour un scope
// (matchs x joueurs).
//
// Rend toujours une tranche exploitable (nil compris) et JAMAIS d erreur : toutes les
// surfaces appelantes sont en best-effort — un sunburst sans ces classes reste juste.
//
// repo nil, scope vide : rien a faire, aucun log. Le titre n a pas la capability (le
// cablage n injecte alors pas de repo), ou le scope est vide.
func Load(
	ctx context.Context,
	repo port.KillSourceClassRepository,
	surface, slug string,
	matchIDs, xuids []string,
) []port.KillSourceClassRow {
	if repo == nil || len(matchIDs) == 0 || len(xuids) == 0 {
		return nil
	}
	filters := port.KillSourceClassFilters{MatchIDs: matchIDs, XUIDs: xuids}
	if err := filters.Validate(); err != nil {
		slog.WarnContext(ctx, "kill sources: filtres invalides", "surface", surface, "err", err)
		return nil
	}
	rows, err := repo.LoadKillSourceClassesAggregated(ctx, slug, filters)
	if err != nil {
		if errors.Is(err, games.ErrCapabilityNotSupported) {
			slog.DebugContext(ctx, "kill sources: aucune passe de film sur ce scope",
				"surface", surface, "title", slug, "match_count", len(matchIDs))
			return nil
		}
		slog.ErrorContext(ctx, "kill sources: lecture echouee",
			"surface", surface, "title", slug, "match_count", len(matchIDs), "err", err)
		return nil
	}
	if len(rows) > 0 {
		kills, nonPub := 0, 0
		for _, r := range rows {
			kills += r.Kills
			nonPub += r.NonPublishableKills
		}
		slog.InfoContext(ctx, "kill sources: kills hors arme a feu",
			"surface", surface, "title", slug, "classes", len(rows),
			"kills", kills, "non_publishable", nonPub, "match_count", len(matchIDs))
	}
	return rows
}
