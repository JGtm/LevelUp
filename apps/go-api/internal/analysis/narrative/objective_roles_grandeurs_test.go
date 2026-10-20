package narrative

// objective_roles_grandeurs_test.go — LA FRONTIÈRE entre « ce que la table porte » et « ce que
// le rôle veut dire ».
//
// Ces tests existent pour un risque précis et déjà coûteux ailleurs dans le dépôt : qu'une
// grandeur mesurée HORS de `match_objective_stats` se retrouve dans la liste d'où la couche
// repo génère ses SUM. Le symptôme serait un `SUM(flag_grabs_net)` sur une table qui n'a pas
// la colonne — une requête qui échoue en production, pas au compilateur.

import "testing"

// TestExtraGrandeurs_HorsDesColonnesDeLaTable : aucune grandeur hors colonne ne doit apparaître
// dans ObjectiveRoleColumns, ni dans les tables de poids.
func TestExtraGrandeurs_HorsDesColonnesDeLaTable(t *testing.T) {
	union := actionWeightKeysUnion()
	for _, role := range AllObjectiveRoles() {
		for _, g := range ObjectiveRoleExtraGrandeurs(role) {
			for _, col := range ObjectiveRoleColumns(role) {
				if col == g {
					t.Errorf("%q est à la fois une grandeur hors colonne et une colonne du rôle %s — "+
						"la couche repo en ferait un SUM sur match_objective_stats", g, role)
				}
			}
			if union[g] {
				t.Errorf("%q figure dans les tables de poids — l'index de participation en ferait "+
					"un SUM sur une colonne qui n'existe pas", g)
			}
		}
	}
}

// TestExtraGrandeurs_ChacuneAUneFamille : une grandeur sans famille s'afficherait sur la grille
// de tous les modes, y compris ceux qui n'ont pas de drapeau.
func TestExtraGrandeurs_ChacuneAUneFamille(t *testing.T) {
	for _, role := range AllObjectiveRoles() {
		for _, g := range ObjectiveRoleExtraGrandeurs(role) {
			if _, ok := ObjectiveExtraGrandeurFamily(g); !ok {
				t.Errorf("grandeur %q (rôle %s) sans famille déclarée", g, role)
			}
		}
	}
}

// TestPrisesNettes_EntrentDansPrendre : la décision produit du 2026-09-13, verrouillée. Et son
// revers : le compteur BRUT reste hors rôle.
func TestPrisesNettes_EntrentDansPrendre(t *testing.T) {
	trouve := false
	for _, g := range ObjectiveRoleGrandeurs(ObjectiveRoleTake) {
		if g == GrandeurFlagGrabsNet {
			trouve = true
		}
		if g == "flag_grabs" {
			t.Error("le compteur BRUT flag_grabs est entré dans « prendre » — il compte le jonglage")
		}
	}
	if !trouve {
		t.Errorf("%q absent du rôle « prendre »", GrandeurFlagGrabsNet)
	}
	if fam, _ := ObjectiveExtraGrandeurFamily(GrandeurFlagGrabsNet); fam != FamilyCTF {
		t.Errorf("famille de %q = %q, want %q", GrandeurFlagGrabsNet, fam, FamilyCTF)
	}
}

// TestObjectiveRoleGrandeurs_ContientLesColonnes : l'union ne perd rien.
func TestObjectiveRoleGrandeurs_ContientLesColonnes(t *testing.T) {
	for _, role := range AllObjectiveRoles() {
		cols := ObjectiveRoleColumns(role)
		grandeurs := ObjectiveRoleGrandeurs(role)
		if len(grandeurs) < len(cols) {
			t.Fatalf("rôle %s : %d grandeurs pour %d colonnes", role, len(grandeurs), len(cols))
		}
		for i, c := range cols {
			if grandeurs[i] != c {
				t.Errorf("rôle %s : grandeur #%d = %q, want la colonne %q (ordre non préservé)",
					role, i, grandeurs[i], c)
			}
		}
	}
}
