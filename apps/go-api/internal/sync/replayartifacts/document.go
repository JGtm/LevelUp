package replayartifacts

// document.go — LA LECTURE D'UN ARTEFACT RANGE, ECRITE UNE SEULE FOIS.
//
// # POURQUOI CE FICHIER EXISTE (constat C7 de la revue de la phase 6)
//
// Les quatre projections post-cuisson de ce paquet lisent LE MEME FICHIER de la MEME
// facon : `os.ReadFile` puis `json.Unmarshal` vers `replay.ReplayDocument`. A la troisieme
// copie, la regle du depot impose un helper ET un garde-rail (CLAUDE.md n 6) — sans quoi
// la quatrieme arrive, puis la cinquieme, et une correction (un cas d'erreur, une garde de
// taille, un compteur) n'en touche qu'une.
//
// # CE QU'IL NE FAIT PAS, ET C'EST DELIBERE
//
// Il ne MEMOISE rien lui-meme : LA memoisation vit une couche au-dessus, dans le lot
// `[]artefactLu` que [lireArtefacts] (derivations.go) construit UNE fois par cycle et fait
// circuler vers les quatre projections (`a.doc`). Ce fichier reste la SEULE fonction qui
// ouvre et deserialise un artefact ; le rattrapage CLI (`ProjeterRasterTactique(path)`,
// appele sans lot en main) est le seul appelant legitime qui y repasse a chaque match.
//
// # LOT M6 (2026-09-07) — LA DERNIERE PROJECTION A REJOINDRE LA REGLE
//
// Jusqu'ici, `projeterRastersTactiques` (raster.go) etait la SEULE des quatre projections a
// rappeler `lireDocumentRange` pour son propre compte au lieu de reutiliser `a.doc` deja en
// memoire — une double lecture/parse de chaque artefact par cycle (constat du registre
// `DECOUVERTES_TACTIQUE`, fusion `origin/feat/v75` -> `feat/tactique`). Elle est alignee sur
// t0film.go/usage.go/bombstats.go : ce sont desormais les QUATRE qui partagent le meme
// document, jamais un de plus.

import (
	"encoding/json"
	"fmt"
	"os"

	"levelup/go-api/internal/analysis/replay"
)

// ouvrirArtefact est le point d'OUVERTURE du fichier, isole de la deserialisation qui suit.
//
// Variable, pas fonction : c'est le seul seam qui permet a un test de COMPTER les lectures
// (raster_lecture_unique_test.go, lot M6) sans introduire de dependance sur un `fs.FS` dans
// tout le paquet pour un seul besoin de diagnostic. Jamais substitue hors des tests.
var ouvrirArtefact = os.ReadFile

// lireDocumentRange lit UN artefact TEL QU IL EST RANGE SUR DISQUE et le deserialise.
//
// SUR DISQUE, PAS LE BLOB : `StoreArtifact` peut REFUSER les octets candidats (garde
// anti-regression) et conserver l'artefact precedent. Projeter le candidat ecrirait un
// derive que le disque ne porte pas — meme doctrine pour les quatre projections.
//
// Les erreurs sont enveloppees avec la MEME formulation pour toutes les projections : leurs
// journaux se lisent alors de la meme facon, et un `lecture artefact` dans un log designe
// toujours le meme evenement.
//
// RENVOIE AUSSI LA TAILLE LUE (octets) : [lireArtefacts] (derivations.go) en a besoin pour la
// marque de derivation (`replaybuild.WriteDerivationsMark`), et c'est le meme octet-la — le
// deduire d'un second `os.Stat` risquerait une taille qui a change entre les deux appels.
func lireDocumentRange(path string) (*replay.ReplayDocument, int, error) {
	raw, err := ouvrirArtefact(path)
	if err != nil {
		return nil, 0, fmt.Errorf("lecture artefact: %w", err)
	}
	var doc replay.ReplayDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, 0, fmt.Errorf("parse artefact: %w", err)
	}
	return &doc, len(raw), nil
}
