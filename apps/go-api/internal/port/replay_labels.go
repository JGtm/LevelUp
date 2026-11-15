// Package port — replay_labels.go : la source des catalogues de libellés du rejeu, par titre.
package port

import "github.com/JGtm/halo-infinite-film-decoder/replay"

// ReplayLabelSource rend le catalogue de libellés du rejeu d'un titre : noms et clés des armes,
// teintes et effets de tir, familles de châssis qualifiées, objets d'objectif. Le catalogue
// appartient au titre ; un service le demande ici et ne sait ni d'où il vient ni quel titre le
// construit.
//
// Un service sans source (nil) sert sans catalogue : armes sans nom ni clé, familles de châssis
// non qualifiées. C'est un état servi, jamais une erreur.
type ReplayLabelSource interface {
	// LabelCatalogue rend le catalogue du titre. Il rend `games.ErrCapabilityNotSupported` pour un
	// titre qui n'a pas de rejeu : jamais le catalogue d'un autre titre. Le catalogue est partagé
	// entre les appelants, en lecture seule.
	LabelCatalogue(titleSlug string) (replay.LabelCatalog, error)
}
