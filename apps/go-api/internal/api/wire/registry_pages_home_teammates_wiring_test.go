package wire

// registry_pages_home_teammates_wiring_test.go — LE CABLAGE DE L'ONGLET EMPRISE ET DE
// L'HISTORIQUE D'OBJECTIF DANS `TeammatesCtx` (revue L6.1 du plan
// PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26, constats R8 et R9).
//
// Le mode de panne est celui que decrit home_factories_parity_test.go : le service degrade EN
// SILENCE sur une dependance nil (WithEmprise(nil) -> « feuille non cablee », un warn et un bloc
// sans frags aux armes speciales ; un predicat du drapeau neutre absent -> l'historique n'ecarte
// aucun mode). Ni le typage ni les tests de service ne le voient : seul le cablage le dit. Les
// options sont lues dans l'arbre syntaxique de la factory — l'ARGUMENT de chaque option et la
// porte `if` qui l'entoure, pas seulement la presence du nom.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"
)

// appelOptionCable — un appel `.With<Nom>(...)` de la factory, son argument et les conditions des
// `if` qui l'entourent.
type appelOptionCable struct {
	args   []string
	portes []string
}

// appelsDansTeammatesCtx rend les appels `.<methode>(...)` du corps de TeammatesCtx.
func appelsDansTeammatesCtx(t *testing.T, methode string) []appelOptionCable {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "registry_pages_home.go", nil, 0)
	if err != nil {
		t.Fatalf("lecture de registry_pages_home.go : %v", err)
	}
	var corps *ast.BlockStmt
	for _, d := range f.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok && fn.Recv != nil && fn.Name.Name == "TeammatesCtx" {
			corps = fn.Body
		}
	}
	if corps == nil {
		t.Fatal("factory TeammatesCtx introuvable dans registry_pages_home.go")
	}
	var pile []ast.Node
	var out []appelOptionCable
	ast.Inspect(corps, func(n ast.Node) bool {
		if n == nil {
			pile = pile[:len(pile)-1]
			return true
		}
		pile = append(pile, n)
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != methode {
			return true
		}
		a := appelOptionCable{}
		for _, arg := range call.Args {
			a.args = append(a.args, types.ExprString(arg))
		}
		for _, p := range pile {
			if ifs, ok := p.(*ast.IfStmt); ok {
				a.portes = append(a.portes, types.ExprString(ifs.Cond))
			}
		}
		out = append(out, a)
		return true
	})
	return out
}

// TestTeammatesCtx_CableLeRepoDeLEmprise — R8 : la feuille de match de l'Emprise (frags aux armes
// spéciales, seule grandeur servie sans film, D10) est câblée SANS condition, sur le PlayerDB du
// joueur ; `WithEmprise(nil)` ou une porte ajoutée rougissent.
func TestTeammatesCtx_CableLeRepoDeLEmprise(t *testing.T) {
	appels := appelsDansTeammatesCtx(t, "WithEmprise")
	if len(appels) != 1 {
		t.Fatalf("%d appel(s) à WithEmprise dans TeammatesCtx, attendu 1", len(appels))
	}
	a := appels[0]
	if len(a.args) != 1 || a.args[0] != "duckdb.NewSquadEmpriseRepo(pdb)" {
		t.Errorf("WithEmprise(%s) : attendu WithEmprise(duckdb.NewSquadEmpriseRepo(pdb)) — sans lui, "+
			"l'onglet Emprise perd les frags aux armes spéciales en silence", strings.Join(a.args, ", "))
	}
	if len(a.portes) != 0 {
		t.Errorf("WithEmprise est sous condition (%v) : la feuille de match est écrite par tous les "+
			"titres, son câblage est inconditionnel (la capability ne gate que le film)", a.portes)
	}
}

// TestTeammatesCtx_CableLePredicatDuDrapeauNeutre — R9 : l'historique d'objectif reçoit le VRAI
// prédicat de la source unique des sous-modes (skillchain.IsNeutralFlagSubMode), sous la
// capability des stats d'objectif qu'il accompagne ; un autre prédicat, nil, ou une porte
// retirée rougissent.
func TestTeammatesCtx_CableLePredicatDuDrapeauNeutre(t *testing.T) {
	appels := appelsDansTeammatesCtx(t, "WithObjectiveHistory")
	if len(appels) != 1 {
		t.Fatalf("%d appel(s) à WithObjectiveHistory dans TeammatesCtx, attendu 1", len(appels))
	}
	a := appels[0]
	if len(a.args) != 1 || a.args[0] != "skillchain.IsNeutralFlagSubMode" {
		t.Errorf("WithObjectiveHistory(%s) : attendu skillchain.IsNeutralFlagSubMode (source unique, "+
			"ratchet no_objective_submode_list_test.go)", strings.Join(a.args, ", "))
	}
	gate := false
	for _, p := range a.portes {
		if strings.Contains(p, "games.CapMatchObjectiveStats") {
			gate = true
		}
	}
	if !gate {
		t.Errorf("WithObjectiveHistory hors de la porte games.CapMatchObjectiveStats (portes : %v)", a.portes)
	}
}

// TestTeammatesCtx_CableLePlacementDesVies — plan Emprise vies, lot V3.3 : le lecteur du placement
// des vies est câblé sur le PlayerDB du joueur SOUS la porte `film.kill_positions` (celle des vies
// au sync, décision V10), et la portée courante du radar l'accompagne SANS condition (table de
// l'onglet Tactique). Un autre argument, une porte retirée ou déplacée, ou la portée perdue
// rougissent : sans eux, le bloc « Groupés ou isolés » disparaît en silence.
func TestTeammatesCtx_CableLePlacementDesVies(t *testing.T) {
	appels := appelsDansTeammatesCtx(t, "WithLifePlacement")
	if len(appels) != 1 {
		t.Fatalf("%d appel(s) à WithLifePlacement dans TeammatesCtx, attendu 1", len(appels))
	}
	a := appels[0]
	if len(a.args) != 1 || a.args[0] != "duckdb.NewSquadLifePlacementRepo(pdb)" {
		t.Errorf("WithLifePlacement(%s) : attendu WithLifePlacement(duckdb.NewSquadLifePlacementRepo(pdb))",
			strings.Join(a.args, ", "))
	}
	gate := false
	for _, p := range a.portes {
		if strings.Contains(p, "games.CapFilmKillPositions") {
			gate = true
		}
	}
	if !gate || len(a.portes) != 1 {
		t.Errorf("WithLifePlacement hors de la seule porte games.CapFilmKillPositions (portes : %v)", a.portes)
	}

	radar := appelsDansTeammatesCtx(t, "WithRadarRange")
	if len(radar) != 1 || len(radar[0].args) != 1 || radar[0].args[0] != "r.radarRangeFor(pdb)" || len(radar[0].portes) != 0 {
		t.Errorf("WithRadarRange = %+v : attendu un appel inconditionnel WithRadarRange(r.radarRangeFor(pdb))", radar)
	}
}

// TestTeammatesCtx_CableLesVehiculesDeLEmprise — plan Emprise véhicules, lot L7.3 : le lecteur de la
// ressource véhicules est câblé sur le PlayerDB du joueur avec le classificateur de source de dégât
// du titre, SOUS la seule porte `film.vehicle_usage` (celle de sa dérivation au sync). Un autre
// argument (sans classificateur, plus de frags par camp), une porte retirée, élargie ou déplacée
// rougissent : sans eux, la ressource disparaît du bloc — ou sort pour un titre sans film — en
// silence.
func TestTeammatesCtx_CableLesVehiculesDeLEmprise(t *testing.T) {
	appels := appelsDansTeammatesCtx(t, "WithVehicleUsage")
	if len(appels) != 1 {
		t.Fatalf("%d appel(s) à WithVehicleUsage dans TeammatesCtx, attendu 1", len(appels))
	}
	a := appels[0]
	const attendu = "duckdb.NewSquadVehicleRepo(pdb, r.killSourceClassifierFor(pdb))"
	if len(a.args) != 1 || a.args[0] != attendu {
		t.Errorf("WithVehicleUsage(%s) : attendu WithVehicleUsage(%s)", strings.Join(a.args, ", "), attendu)
	}
	if len(a.portes) != 1 || !strings.Contains(a.portes[0], "games.CapFilmVehicleUsage") {
		t.Errorf("WithVehicleUsage hors de la seule porte games.CapFilmVehicleUsage (portes : %v)", a.portes)
	}
}
