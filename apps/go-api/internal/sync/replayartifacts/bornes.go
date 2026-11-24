package replayartifacts

// bornes.go — les bornes par cycle de l'étape des rejeux : garde-fou et valeur hébergée de la capacité
// FilmCatchUp du profil de déploiement, lues par le hook (Hook.PerCycle, Hook.Horizon) et les Deps.

import "levelup/go-api/internal/sync/killcollector"

// maxPerCycle : le GARDE-FOU du nombre d'artefacts construits par cycle sous la capacité FilmCatchUp, et non
// sa limite (hors capacité : [RejeuxParCycleHeberge]). La
// limite d'une passe est son TEMPS ([BudgetParCycle]) : un match synchronisé l'est en totalité, film
// traité compris, et le coût d'un film varie trop pour qu'un nombre le dise. La borne n'est que le
// plafond d'une liste de travail sans fin ; elle a la valeur de celle de la source des frags
// ([killcollector.GardeFouFilmsParCycle]), pour que les deux étapes film ne divergent pas. Le solde
// repart au cycle suivant, ou au backfill CLI (levelup backfill-replay), idempotent et repris par
// SchemaVersion.
const maxPerCycle = killcollector.GardeFouFilmsParCycle

// RejeuxParCycleHeberge : le plafond de matchs traités par cycle HORS de la capacité FilmCatchUp (serveur
// hébergé, 2 vCPU et 2 Go) : la borne d'avant le rattrapage de l'arriéré de films. Le solde repart au cycle
// suivant, ou au backfill CLI (levelup backfill-replay).
const RejeuxParCycleHeberge = 5

// PerCycle rend le plafond de matchs traités par cycle : [maxPerCycle] sous la capacité FilmCatchUp,
// [RejeuxParCycleHeberge] sinon (hook nil compris).
func (h *Hook) PerCycle() int {
	if h != nil && h.FilmCatchUp {
		return maxPerCycle
	}
	return RejeuxParCycleHeberge
}

// Horizon rend le nombre de matchs lus au registre pour le retard : [BacklogHorizon] sous la capacité
// FilmCatchUp, [BacklogHorizonHeberge] sinon (hook nil compris).
func (h *Hook) Horizon() int {
	if h != nil && h.FilmCatchUp {
		return BacklogHorizon
	}
	return BacklogHorizonHeberge
}
