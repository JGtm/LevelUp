package port

import "context"

// VehicleDestructionStats agrège, sur un scope de matchs, deux compteurs « fun »
// liés aux véhicules : véhicules détruits et vols à la tire (hijacks). Title-agnostic
// par construction — la SOURCE diffère par titre (Halo Infinite : personal_score_awards ;
// Halo 5 : véhicules détruits via commendations NATIVES match_commendations, hijacks
// via les médailles Hijack/Skyjack de medals_earned — il n'existe PAS de commendation
// « Grand Theft »/« Vol à la tire » dans le référentiel H5, cf. doc package
// internal/platform/duckdb/vehicle_commendation_stats_repo.go) mais la sémantique
// produit est commune (les deux alimentent SynthesisDetailedStats.TotalVehiclesDestroyed
// / TotalHijacks).
type VehicleDestructionStats struct {
	VehiclesDestroyed int
	Hijacks           int
}

// VehicleDestructionStatsRepository charge, pour un joueur (xuid) sur un scope fermé
// de matchs, les compteurs de véhicules détruits / vols à la tire.
//
// Contrat de dégradation : le référentiel de résolution (noms de commendations) peut
// être absent (titre sans données seedées) — dans ce cas l'implémentation retourne des
// compteurs à zéro SANS erreur (les cartes correspondantes disparaissent côté front,
// jamais de 500). Une erreur n'est retournée que pour une anomalie réelle (SQL invalide,
// connexion coupée) — le caller la traite en best-effort (log, pas d'échec de page).
//
// Implémenté par internal/platform/duckdb.VehicleCommendationStatsRepo (Halo 5 :
// commendations natives pour les véhicules détruits, médailles natives pour les
// hijacks). Câblé UNIQUEMENT pour les titres portant la capability
// commendations.native (jamais de gating par slug — cf. registry SynthesisCtx).
type VehicleDestructionStatsRepository interface {
	LoadVehicleDestructionStats(
		ctx context.Context,
		slug string,
		matchIDs []string,
		xuid string,
	) (VehicleDestructionStats, error)
}
