// Package replayview projette le document de rejeu STOCKE
// (`internal/analysis/replay.ReplayDocument`, le format du fichier d'artefact) sur le
// document SERVI (`internal/domain/replaydoc.ReplayDocument`, la forme de fil publique).
//
// C'EST LA SEULE ARETE ENTRE LES DEUX MONDES, et c'est tout l'objet du paquet : tant que
// la projection existe, ajouter un calque a la cuisson ne touche plus le contrat public, et
// renommer un champ pour le client n'invalide plus le parc d'artefacts deja cuits.
//
// FONCTIONS PURES, ZERO E/S. Rien ici ne lit un fichier, n'ouvre une base ni ne journalise :
// la lecture de l'artefact reste au service appelant. Les conversions sont exhaustives —
// `parity_test.go` exige une decision ECRITE pour chaque champ exporte du document stocke
// (copie, transforme, ou inscrit dans la liste datee `champsNonServis`) et pour chaque champ
// du document servi (une source).
//
// PARTAGE DE MEMOIRE ASSUME : les tranches de scalaires (positions de projectile, polygones,
// listes de grenades) sont reprises TELLES QUELLES, sans recopie — l'appelant lit l'artefact,
// projette, et jette la valeur stockee. Recopier des megaoctets de coordonnees pour une
// valeur qui meurt a la ligne suivante serait payer le rejeu deux fois.
package replayview

import (
	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/replaydoc"
)

// FromArtifact projette le document d'artefact sur le document servi.
func FromArtifact(doc replay.ReplayDocument) replaydoc.ReplayDocument {
	return toReplayDocument(doc)
}

// MapBackgroundOf projette le calage du fond de carte. nil reste nil : l'absence de fond
// est un cas nominal (toutes les cartes n'ont pas d'image figee), pas une erreur.
func MapBackgroundOf(bg *replay.MapBackground) *replaydoc.MapBackground {
	return ptrOf(bg, toMapBackground)
}

// MapCalloutsOf projette l'entree de zones nommees d'une carte. nil reste nil : une carte
// Forge n'en porte aucune, par construction.
func MapCalloutsOf(e *replay.MapCalloutsEntry) *replaydoc.MapCalloutsEntry {
	return ptrOf(e, toMapCalloutsEntry)
}

// sliceOf projette chaque element. LA NULLITE EST PRESERVEE, et ce n'est pas un detail :
// sur les tranches sans `omitempty` (`tracks`, `points`, `spawns`, `zones`...), une tranche
// nulle se serialise `null` et une tranche vide `[]` — deux corps differents pour le client.
func sliceOf[S, D any](in []S, conv func(S) D) []D {
	if in == nil {
		return nil
	}
	out := make([]D, len(in))
	for i := range in {
		out[i] = conv(in[i])
	}
	return out
}

// ptrOf projette la valeur pointee ; nil reste nil (champ absent du corps).
func ptrOf[S, D any](in *S, conv func(S) D) *D {
	if in == nil {
		return nil
	}
	out := conv(*in)
	return &out
}

// mapOf projette chaque valeur ; nil reste nil, meme raison que sliceOf.
func mapOf[S, D any](in map[string]S, conv func(S) D) map[string]D {
	if in == nil {
		return nil
	}
	out := make(map[string]D, len(in))
	for k, v := range in {
		out[k] = conv(v)
	}
	return out
}
