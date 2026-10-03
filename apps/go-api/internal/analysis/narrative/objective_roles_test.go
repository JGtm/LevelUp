package narrative

// objective_roles_test.go — le garde-rail de la classification par rôle : la
// partition prendre / défendre / écartées couvre EXACTEMENT le vocabulaire des
// tables de poids (aucune colonne orpheline, aucun recouvrement, aucune colonne
// fantôme), et « tenir » est EXACTEMENT l'union des colonnes de durée par famille.
// Une divergence silencieuse ferait deux vocabulaires — le défaut que la source
// unique de objective_participation.go interdit.

import "testing"

// actionWeightKeysUnion : l'union des clés des tables de poids, la référence.
func actionWeightKeysUnion() map[string]bool {
	union := map[string]bool{}
	for _, weights := range ObjectiveFamilyActionWeights {
		for col := range weights {
			union[col] = true
		}
	}
	return union
}

func TestObjectiveRoles_PartitionDesColonnesDAction(t *testing.T) {
	union := actionWeightKeysUnion()

	classees := map[string]string{}
	classer := func(cols []string, role string) {
		for _, c := range cols {
			if avant, deja := classees[c]; deja {
				t.Errorf("colonne %q classée deux fois (%s puis %s)", c, avant, role)
			}
			classees[c] = role
			if !union[c] {
				t.Errorf("colonne %q (%s) absente des tables de poids — vocabulaire parallèle interdit", c, role)
			}
		}
	}
	classer(objectiveRoleTakeColumns, "prendre")
	classer(objectiveRoleDefendColumns, "défendre")
	classer(objectiveRoleExcludedActionColumns, "écartée")

	for col := range union {
		if _, ok := classees[col]; !ok {
			t.Errorf("colonne d'action %q sans rôle ni exclusion justifiée", col)
		}
	}
}

func TestObjectiveRoles_TenirEstLUnionDesDurees(t *testing.T) {
	attendu := map[string]bool{}
	for _, cols := range ObjectiveFamilyHoldColumns {
		for _, c := range cols {
			attendu[c] = true
		}
	}
	hold := ObjectiveRoleColumns(ObjectiveRoleHold)
	vues := map[string]bool{}
	for _, c := range hold {
		if vues[c] {
			t.Errorf("colonne de durée %q en double dans le rôle tenir", c)
		}
		vues[c] = true
		if !attendu[c] {
			t.Errorf("colonne %q du rôle tenir absente d'ObjectiveFamilyHoldColumns", c)
		}
	}
	if len(vues) != len(attendu) {
		t.Errorf("rôle tenir : %d colonnes, ObjectiveFamilyHoldColumns en porte %d", len(vues), len(attendu))
	}
}

func TestObjectiveRoles_ColumnsRendUneCopie(t *testing.T) {
	a := ObjectiveRoleColumns(ObjectiveRoleTake)
	if len(a) == 0 {
		t.Fatal("rôle prendre vide")
	}
	a[0] = "mutation"
	b := ObjectiveRoleColumns(ObjectiveRoleTake)
	if b[0] == "mutation" {
		t.Error("ObjectiveRoleColumns rend la slice interne — une mutation d'appelant corromprait la table")
	}
}
