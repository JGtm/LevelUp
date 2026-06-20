// Package classification — stratégies REUTILISABLES, config-driven, pour déterminer
// le caractère classé/non-classé (et PvE) d'un match, INDEPENDAMMENT du titre.
//
// Le contrat de SORTIE est stable et canonique (*bool, sémantique
// canonical.MatchSummary.IsRanked / IsPvE) : nil = INDETERMINE (pas de donnée
// autoritative) → les consommateurs (ingestion, CSR, LUSR, UI) ne mintent JAMAIS
// un faux « classé » sur nil. La DETERMINATION est une STRATEGIE choisie par
// l'adapter du titre — pas du code bespoke réécrit à chaque titre.
//
// Ce package est un LEAF : il ne dépend ni de canonical, ni d'aucun titre. Il
// renvoie des *bool (la sémantique du contrat canonique) sans importer canonical,
// pour rester réutilisable par n'importe quel titre sans créer de cycle.
//
// Stratégie #1 livrée ici : appartenance à un SET autoritatif d'ids de playlist
// classées (HopperId pour Halo 5). La plus universelle — tout titre Halo a des
// playlists + une notion autoritative de « classé ». Réutilisation = DATA seule
// (un nouveau titre fournit son TOML d'ids) → zéro code. Cf.
// .ai/HANDOFF_H5_RANKED_CLASSIFICATION.md §7 (extensibilité Halo 7).
package classification

// RankedClassifier détermine, pour une playlist donnée (par son id), si le match
// est classé et/ou PvE. Verdict nil = INDETERMINE (jamais coercé en true qui
// minterait un CSR). Contrat de sortie STABLE, partagé par tous les titres : un
// titre futur réutilisant une stratégie connue ne fournit que sa data ; une
// stratégie inédite implémente cette interface, tout le reste inchangé.
type RankedClassifier interface {
	// IsRanked retourne le verdict « classé » de la playlist (nil = indéterminé).
	IsRanked(playlistID string) *bool
	// IsPvE retourne le verdict « PvE/coopératif » de la playlist (nil = indéterminé).
	IsPvE(playlistID string) *bool
}
