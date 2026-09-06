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
// Il ne MEMOISE rien. Chaque projection relit le fichier, soit trois a quatre lectures du
// meme artefact par cycle. Passer a UNE lecture par artefact — le document circulant entre
// les projections — est une amelioration REELLE mais qui change la forme de
// `persister*`/`projeter*` : elle est consignee au §7 du plan tactique, pas faite ici.
// Grouper les deux changements aurait melange une factorisation mecanique et une refonte
// de flux dans le meme diff.

import (
	"encoding/json"
	"fmt"
	"os"

	"levelup/go-api/internal/analysis/replay"
)

// lireDocumentRange lit UN artefact TEL QU IL EST RANGE SUR DISQUE et le deserialise.
//
// SUR DISQUE, PAS LE BLOB : `StoreArtifact` peut REFUSER les octets candidats (garde
// anti-regression) et conserver l'artefact precedent. Projeter le candidat ecrirait un
// derive que le disque ne porte pas — meme doctrine pour les quatre projections.
//
// Les erreurs sont enveloppees avec la MEME formulation pour toutes les projections : leurs
// journaux se lisent alors de la meme facon, et un `lecture artefact` dans un log designe
// toujours le meme evenement.
func lireDocumentRange(path string) (*replay.ReplayDocument, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("lecture artefact: %w", err)
	}
	var doc replay.ReplayDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse artefact: %w", err)
	}
	return &doc, nil
}
