package killcollector

// carte_avant_telechargement.go — LA CARTE SE RESOUT AVANT LE TELECHARGEMENT (2026-09-27).
//
// Un match dont la carte n est pas resolue (Forge, carte hors catalogue de bornes, pas de nom en
// base) ne se decode pas : « le flux du film est la seule source fiable. Pas de repli. » Retire
// APRES le telechargement, il consommait une place de la liste de travail et un telechargement a
// chaque passe — et comme il ne quitte jamais le backlog (aucun marqueur terminal : il redevient
// decodable le jour ou le catalogue connait sa carte), des matchs Forge recents, tries en tete,
// affamaient le vrai backlog. Les trois chemins de collecte (post-sync, `backfill-killsource` hors
// ligne et `--online`) filtrent donc leur liste ICI, avant la borne et avant le telechargement.

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/observability"
)

// echantillonDesEcartes : combien d identifiants le journal cite. La liste entiere se relit en base.
const echantillonDesEcartes = 5

// RetenirLesMatchsAvecCarte parcourt `ids` DANS L ORDRE et rend, dans cet ordre, au plus `plafond`
// matchs (`plafond <= 0` : tous) dont la carte se resout ; les autres sont RETIRES, comptes
// (`killsource_ecartes_carte_avant_telechargement`) et journalises en une ligne. La resolution
// s arrete des que le plafond est atteint : un cycle ne lit pas la carte de tout le backlog.
//
// UNE ERREUR DE LECTURE DU NOM N ECARTE PAS : ce n est pas une carte absente mais une panne, et le
// match reste dans la liste — le collecteur la rendra en erreur, retentee au cycle suivant.
func (c *KillSourceCollector) RetenirLesMatchsAvecCarte(
	ctx context.Context, ids []string, plafond int,
) (retenus []string, ecartes int) {
	var echantillon []string
	for _, id := range ids {
		if plafond > 0 && len(retenus) >= plafond {
			break
		}
		if _, err := c.carteDuMatch(ctx, id); err != nil && carteNonResolue(err) {
			ecartes++
			if len(echantillon) < echantillonDesEcartes {
				echantillon = append(echantillon, id)
			}
			continue
		}
		retenus = append(retenus, id)
	}
	if ecartes > 0 {
		observability.AddInt(metricCarteAvantTelechargement, int64(ecartes))
		slog.WarnContext(ctx, "killsource: matchs RETIRES de la liste de travail — carte non resolue "+
			"(aucun telechargement ; ils restent candidats et reviendront quand leur carte se resoudra)",
			"ecartes", ecartes, "exemples", echantillon, "cable", c.CaptureCablee())
	}
	return retenus, ecartes
}
