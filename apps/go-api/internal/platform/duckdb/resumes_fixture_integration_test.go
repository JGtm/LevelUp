//go:build integration

package duckdb

// resumes_fixture_integration_test.go — la suite de l'aide des résumés pour les fixtures des tests
// d'intégration (deux bases d'un PlayerDB de test, bornes de fenêtre des vues filles).

import (
	"testing"
)

// poserJournalSurLesDeux pose le même journal (et ses résumés) sur les deux bases d'un PlayerDB de
// test dont les fixtures historiques écrivent à la fois sur le joueur et sur le partagé
// (execOnSharedDBs). Une seule base quand elles sont la même.
func poserJournalSurLesDeux(t *testing.T, pdb *PlayerDB, lignes ...ligneJournal) {
	t.Helper()
	poserJournal(t, pdb.Player.SQLDb(), lignes...)
	if pdb.Shared != pdb.Player {
		poserJournal(t, pdb.Shared.SQLDb(), lignes...)
	}
}

// poserJournalSurLesDeuxAJour : [poserJournalAJour] sur les deux bases (passe neuve d'un match déjà résumé).
func poserJournalSurLesDeuxAJour(t *testing.T, pdb *PlayerDB, lignes ...ligneJournal) {
	t.Helper()
	poserJournalAJour(t, pdb.Player.SQLDb(), lignes...)
	if pdb.Shared != pdb.Player {
		poserJournalAJour(t, pdb.Shared.SQLDb(), lignes...)
	}
}
