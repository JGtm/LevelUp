package main

// compare.go — CONFRONTER L'ARTEFACT DU PARC A CELUI FRAICHEMENT CUIT, PAR TOUS LES AXES DE
// `cmd/replay-diff` (y compris la somme des durees, ajoutee dans ce meme chantier).
//
// AUCUNE LOGIQUE DE COMPARAISON ICI : elle vit dans `internal/replaydiff`, partagee avec
// `cmd/replay-diff` (le meme code qui a servi au balayage du parc du 2026-09-06) — zero copie.

import (
	"fmt"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/replaydiff"
)

// referenceArtifactPath rend le chemin de l'artefact de REFERENCE — celui deja cuit dans le
// parc reel, jamais celui de la racine de travail.
func referenceArtifactPath(parcRoot, titleSlug, matchID string) string {
	return title.NewPathResolver(parcRoot).ReplayArtifactPath(titleSlug, matchID)
}

// freshArtifactPath rend le chemin de l'artefact FRAIS, ecrit par bakeTemoin dans la racine
// de travail — jamais dans le parc.
func freshArtifactPath(workRoot, titleSlug, matchID string) string {
	return title.NewPathResolver(workRoot).ReplayArtifactPath(titleSlug, matchID)
}

// compareTemoin lit les deux artefacts et rend leur rapport de comparaison.
func compareTemoin(refPath, freshPath string) (replaydiff.Rapport, error) {
	docRef, err := replaydiff.LireDocument(refPath)
	if err != nil {
		return replaydiff.Rapport{}, fmt.Errorf("artefact de reference (%s) : %w", refPath, err)
	}
	docFresh, err := replaydiff.LireDocument(freshPath)
	if err != nil {
		return replaydiff.Rapport{}, fmt.Errorf("artefact frais (%s) : %w", freshPath, err)
	}
	rap := replaydiff.Comparer(replaydiff.Empreindre(docRef), replaydiff.Empreindre(docFresh))
	rap.FichierAncien, rap.FichierNouveau = refPath, freshPath
	return rap, nil
}
