package wire

// registry_pages_timeseries.go — LE CABLAGE DE L'ONGLET « USAGES » DES SERIES TEMPORELLES (Emprise
// solo, plan `.ai/PLAN_TIMESERIES_USAGES_EMPRISE_2026-10-05.md`, lot L2.8), appele par la factory
// `Timeseries` (registry_pages.go, qui depasse deja le seuil de taille du depot). Le resume d'usage,
// lui, reste cable par la factory sous film.usage_summary : il sert aussi d'autres blocs.

import (
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/service"
)

// cablerUsagesTimeseries pose les dependances propres a l'Emprise solo.
func (r *ServiceRegistry) cablerUsagesTimeseries(svc *service.TimeseriesService, pdb *duckdb.PlayerDB) *service.TimeseriesService {
	// La feuille de match (frags aux armes speciales) est ecrite par tous les titres : cablage
	// INCONDITIONNEL, comme l'Escouade. Jamais slug==.
	svc = svc.WithEmprise(duckdb.NewSquadEmpriseRepo(pdb)).
		// Emblème de la fiche « Ma part à l'objectif » : le MEME chargeur que les fiches de
		// medailles de l'Escouade ; degradation silencieuse (initiale) par contrat du chargeur.
		WithEmblemLoader(duckdb.NewSquadV2LoaderAdapter(r.resolveByGT))
	// Ressource vehicules : la capability fine film.vehicle_usage, comme l'Escouade (absente pour
	// Halo 5 -> ressource absente). Jamais slug==.
	if r.capabilitiesForPDB(pdb).Has(games.CapFilmVehicleUsage) {
		svc = svc.WithVehicleUsage(duckdb.NewSquadVehicleRepo(pdb, r.killSourceClassifierFor(pdb)))
	}
	return svc
}
