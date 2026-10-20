//go:build windows

package main

// cmd_compact_passes_remplacement_win.go — le remplacement de la base par le fichier réécrit,
// sous Windows (cf. cmd_compact_passes_rewrite.go, « LA MÉTHODE »).
// Contrainte de plateforme EXPLICITE (ligne `//go:build`), jamais par le nom : un suffixe `_windows`
// / `_linux` serait une contrainte implicite (archlint `TestAucunSuffixeDePlateformeAccidentel`).
//
// Windows refuse de remplacer un fichier que DuckDB tient (mesuré le 2026-09-27 : « Accès
// refusé »). La connexion est donc FERMÉE, puis le rename suit IMMÉDIATEMENT — rien entre les deux
// que le point d'observation des tests (nil en production). Un tiers qui a ouvert la base dans
// l'intervalle la tient au moment du rename : le rename échoue et c'est un refus, la base
// d'origine intacte, le tiers continuant sur elle.

import "fmt"

// remplacerSousVerrou rend (true, …) dès que la base a été remplacée.
func remplacerSousVerrou(r *reecriture) (bool, error) {
	if err := verifierWALAvantRemplacement(r); err != nil {
		return false, err
	}
	if err := r.fermerSource(); err != nil {
		return false, err
	}
	if err := passerEtape(etapeApresFermeture); err != nil {
		return false, err
	}
	if err := renommer(r.neuf, r.path); err != nil {
		return false, fmt.Errorf("réécriture refusée : %s n'a pas pu être remplacée (un autre "+
			"processus l'a ouverte entre la fermeture et le remplacement ?) : %w", r.path, err)
	}
	return true, nil
}
