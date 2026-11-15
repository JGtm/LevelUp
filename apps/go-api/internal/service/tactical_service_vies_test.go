package service

// tactical_service_vies_test.go — LE PARAMETRE `vies` : « depart » (la premiere vie seule, defaut)
// contre « toutes » (premiere vie, reapparitions, debuts de manche), sur des sidecars poses a la
// main.
//
// Le corpus : trois matchs (le plancher d'une grappe), chacun avec QUATRE vies —
//
//	frame    0 : premiere vie, pres de (0, 0)         route par la cellule (10, 10)
//	frame 1000 : reapparition, pres de (50, 50)       route par la cellule (30, 30)
//	frame 2000 : debut de manche, pres de (100, 100)  route par la cellule (50, 50)
//	frame 3000 : piste sans drapeau, pres de (150, 150) route par la cellule (70, 70)
//
// plus un quatrieme match dont la seule vie est la premiere. La piste sans drapeau (une vie
// coupee par un trou de replication) n'est une apparition sous aucun vocabulaire.

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/i18n"
)

const (
	frameDepart      = 0
	frameReapparait  = 1000
	frameDebutManche = 2000
	frameSansDrapeau = 3000
)

// celluleDeLaVie est la cellule (col = lig) que la route de chaque vie traverse.
var celluleDeLaVie = map[int]int{frameDepart: 10, frameReapparait: 30, frameDebutManche: 50, frameSansDrapeau: 70}

// sidecarQuatreVies pose un joueur aux quatre vies ci-dessus, decalees de dx metres d'un match a
// l'autre (trois points voisins forment UN amas).
func sidecarQuatreVies(matchID string, dx float64) *domain.TacticalRasterSidecar {
	j := domain.TacticalRasterJoueur{
		XUID: tsMoi,
		Spawns: []domain.TacticalRasterSpawn{
			{Frame: frameDepart, X: 0.25 + dx, Y: 0.25, PremiereVie: true},
			{Frame: frameReapparait, X: 50.25 + dx, Y: 50.25, Reapparition: true, Source: "repli_ecart"},
			{Frame: frameDebutManche, X: 100.25 + dx, Y: 100.25, DebutDeManche: true},
			{Frame: frameSansDrapeau, X: 150.25 + dx, Y: 150.25},
		},
		Cellules: []domain.TacticalRasterCellule{{Col: 1, Lig: 1, Echantillons: 8}},
	}
	for _, frame := range []int{frameDepart, frameReapparait, frameDebutManche, frameSansDrapeau} {
		c := celluleDeLaVie[frame]
		j.Routes = append(j.Routes, domain.TacticalRasterRoute{
			DebutFrame: frame, Cases: []domain.TacticalRasterCase{{Col: c, Lig: c}},
		})
	}
	sc := sidecarPose(matchID)
	sc.Joueurs = []domain.TacticalRasterJoueur{j}
	return sc
}

// viesFixture : a1..a3 aux quatre vies, d1 avec sa seule premiere vie (pres du meme depart).
func viesFixture() (*mockRasterStore, domain.TacticalUnivers, []string) {
	sc := map[string]*domain.TacticalRasterSidecar{}
	variantes := map[string]string{}
	ids := []string{}
	for i, id := range []string{"a1", "a2", "a3"} {
		sc[id] = sidecarQuatreVies(id, float64(i)*0.4)
		variantes[id] = "Slayer:Arena"
		ids = append(ids, id)
	}
	seul := sidecarQuatreVies("d1", 1.2)
	seul.Joueurs[0].Spawns = seul.Joueurs[0].Spawns[:1]
	seul.Joueurs[0].Routes = seul.Joueurs[0].Routes[:1]
	sc["d1"] = seul
	variantes["d1"] = "Slayer:Arena"
	ids = append(ids, "d1")
	return &mockRasterStore{sidecars: sc}, universVariantes(variantes), ids
}

func lireEnVies(t *testing.T, svc *TacticalService, question, vies, spawn string, ids []string) domain.TacticalRaster {
	t.Helper()
	out, err := svc.Raster(i18n.WithContext(context.Background(), i18n.Default), domain.TacticalRasterRequest{
		MapID: "streets", Question: question, Qui: domain.TacticalQuiMoi,
		Scope: domain.TacticalScope{MatchIDs: ids, Spawn: spawn, Vies: vies},
	})
	if err != nil {
		t.Fatalf("lecture %s (vies=%q, spawn=%q): %v", question, vies, spawn, err)
	}
	return out
}

// grappeAutour rend l'identifiant de la grappe dont le barycentre est pres de (x, y).
func grappeAutour(t *testing.T, grappes []domain.TacticalGrappe, x, y float64) string {
	t.Helper()
	for _, g := range grappes {
		if g.X > x-5 && g.X < x+5 && g.Y > y-5 && g.Y < y+5 {
			return g.ID
		}
	}
	t.Fatalf("aucune grappe pres de (%v, %v) : %+v", x, y, grappes)
	return ""
}

// aLaCellule dit si la lecture peint la cellule (c, c).
func aLaCellule(out domain.TacticalRaster, c int) bool {
	return celluleEn(out.Cellules, (float64(c)+0.5)*0.5, (float64(c)+0.5)*0.5) != nil
}

// TestVies_DepartEstLeDefaut — sans parametre, ou avec `depart`, la lecture est la meme et les
// grappes ne portent que la premiere vie.
func TestVies_DepartEstLeDefaut(t *testing.T) {
	store, univ, ids := viesFixture()
	svc := svcArtefact(univ, store)
	vide := lireEnVies(t, svc, domain.TacticalQuestionTemps, "", "", ids)
	depart := lireEnVies(t, svc, domain.TacticalQuestionTemps, domain.TacticalViesDepart, "", ids)
	if len(vide.Grappes) != 1 {
		t.Fatalf("grappes par defaut = %+v, attendu 1 (la premiere vie seule)", vide.Grappes)
	}
	if len(depart.Grappes) != 1 || depart.Grappes[0].ID != vide.Grappes[0].ID {
		t.Fatalf("`depart` = %+v, attendu la meme grappe que le defaut %+v", depart.Grappes, vide.Grappes)
	}
	if vide.Grappes[0].Matchs != 4 {
		t.Fatalf("la grappe de depart compte %d matchs, attendu 4", vide.Grappes[0].Matchs)
	}
}

// TestVies_ToutesRegroupeToutesLesApparitions — premiere vie, reapparition et debut de manche
// forment trois grappes ; la piste sans drapeau n'en forme aucune.
func TestVies_ToutesRegroupeToutesLesApparitions(t *testing.T) {
	store, univ, ids := viesFixture()
	svc := svcArtefact(univ, store)
	out := lireEnVies(t, svc, domain.TacticalQuestionTemps, domain.TacticalViesToutes, "", ids)
	if len(out.Grappes) != 3 {
		t.Fatalf("grappes = %+v, attendu 3 (depart, reapparition, debut de manche)", out.Grappes)
	}
	grappeAutour(t, out.Grappes, 0, 0)
	grappeAutour(t, out.Grappes, 50, 50)
	grappeAutour(t, out.Grappes, 100, 100)
	for _, g := range out.Grappes {
		if g.X > 140 {
			t.Fatalf("la piste sans drapeau a forme une grappe : %+v", g)
		}
	}
}

// TestVies_ToutesRetientLaRouteDeLaVieDeLaGrappe — sous un filtre de grappe, seule la route de
// la vie partie de cette grappe compte ; les routes des trois autres vies sortent de la lecture.
func TestVies_ToutesRetientLaRouteDeLaVieDeLaGrappe(t *testing.T) {
	store, univ, ids := viesFixture()
	svc := svcArtefact(univ, store)
	grappes := lireEnVies(t, svc, domain.TacticalQuestionRoutes, domain.TacticalViesToutes, "", ids).Grappes
	reapparition := grappeAutour(t, grappes, 50, 50)

	out := lireEnVies(t, svc, domain.TacticalQuestionRoutes, domain.TacticalViesToutes, reapparition, ids)
	if out.MatchsRetenus != 3 || out.MatchsFiltres != 3 {
		t.Fatalf("matchs retenus/filtres = %d/%d, attendu 3/3 (d1 n'a pas cette vie)",
			out.MatchsRetenus, out.MatchsFiltres)
	}
	if !aLaCellule(out, celluleDeLaVie[frameReapparait]) {
		t.Fatalf("la route de la vie partie de la grappe n'est pas peinte : %+v", out.Cellules)
	}
	for _, frame := range []int{frameDepart, frameDebutManche, frameSansDrapeau} {
		if aLaCellule(out, celluleDeLaVie[frame]) {
			t.Fatalf("la route de la vie %d, partie d'ailleurs, est peinte sous le filtre : %+v", frame, out.Cellules)
		}
	}
	// LES GRAPPES RESTENT TOUTES : la liste proposee ne se reduit pas a la selection.
	if len(out.Grappes) != 3 {
		t.Fatalf("grappes sous filtre = %+v, attendu les 3", out.Grappes)
	}
}

// TestVies_ToutesSansFiltre_LesRoutesNeChangentPas — sans filtre de grappe, `toutes` ne change
// que la liste des grappes : les quatre routes comptent.
func TestVies_ToutesSansFiltre_LesRoutesNeChangentPas(t *testing.T) {
	store, univ, ids := viesFixture()
	svc := svcArtefact(univ, store)
	out := lireEnVies(t, svc, domain.TacticalQuestionRoutes, domain.TacticalViesToutes, "", ids)
	for frame, c := range celluleDeLaVie {
		if !aLaCellule(out, c) {
			t.Fatalf("la route de la vie %d (cellule %d) manque sans filtre de grappe", frame, c)
		}
	}
}

// TestVies_DepartSousFiltre_RoutesInchangees — sous `depart`, le filtre de grappe restreint
// l'univers aux matchs et rien d'autre : TOUTES les routes de ces matchs comptent, comme avant.
func TestVies_DepartSousFiltre_RoutesInchangees(t *testing.T) {
	store, univ, ids := viesFixture()
	svc := svcArtefact(univ, store)
	grappes := lireEnVies(t, svc, domain.TacticalQuestionRoutes, "", "", ids).Grappes
	out := lireEnVies(t, svc, domain.TacticalQuestionRoutes, "", grappes[0].ID, ids)
	if out.MatchsRetenus != 4 {
		t.Fatalf("matchs retenus = %d, attendu les 4 matchs partis de la grappe", out.MatchsRetenus)
	}
	// d1 n'a qu'une route : les trois autres cellules sont vues dans 3 matchs, au plancher.
	for frame, c := range celluleDeLaVie {
		if !aLaCellule(out, c) {
			t.Fatalf("`depart` sous filtre : la route de la vie %d manque (cellule %d)", frame, c)
		}
	}
}

// TestVies_ToutesNeModifiePasLesSidecarsLus — le sidecar que le lecteur rend peut etre partage :
// reduire les routes se fait sur une copie.
func TestVies_ToutesNeModifiePasLesSidecarsLus(t *testing.T) {
	store, univ, ids := viesFixture()
	svc := svcArtefact(univ, store)
	grappes := lireEnVies(t, svc, domain.TacticalQuestionRoutes, domain.TacticalViesToutes, "", ids).Grappes
	lireEnVies(t, svc, domain.TacticalQuestionRoutes, domain.TacticalViesToutes, grappeAutour(t, grappes, 50, 50), ids)
	for id, sc := range store.sidecars {
		if want := len(sc.Joueurs[0].Spawns); len(sc.Joueurs[0].Routes) != want {
			t.Fatalf("le sidecar %s a perdu des routes : %d pour %d spawns", id, len(sc.Joueurs[0].Routes), want)
		}
	}
}

// TestVies_ToutesGrappeInconnue — une grappe qui n'existe pas sous `toutes` est un refus type,
// jamais une lecture non filtree.
func TestVies_ToutesGrappeInconnue(t *testing.T) {
	store, univ, ids := viesFixture()
	svc := svcArtefact(univ, store)
	_, err := svc.Raster(context.Background(), domain.TacticalRasterRequest{
		MapID: "streets", Question: domain.TacticalQuestionRoutes, Qui: domain.TacticalQuiMoi,
		Scope: domain.TacticalScope{MatchIDs: ids, Spawn: "s+999999+999999", Vies: domain.TacticalViesToutes},
	})
	if !errors.Is(err, domain.ErrTacticalSpawnInconnu) {
		t.Fatalf("err = %v, attendu ErrTacticalSpawnInconnu", err)
	}
}

// TestVies_ValeurInconnue — une valeur hors vocabulaire est refusee (400 cote HTTP), pour la
// lecture agregee comme pour le detail d'une cellule ; jamais remplacee par `depart`.
func TestVies_ValeurInconnue(t *testing.T) {
	store, univ, ids := viesFixture()
	svc := svcArtefact(univ, store)
	_, err := svc.Raster(context.Background(), domain.TacticalRasterRequest{
		MapID: "streets", Question: domain.TacticalQuestionRoutes, Qui: domain.TacticalQuiMoi,
		Scope: domain.TacticalScope{MatchIDs: ids, Vies: "toutes-les-vies"},
	})
	if !errors.Is(err, domain.ErrTacticalViesInconnue) {
		t.Fatalf("Raster : err = %v, attendu ErrTacticalViesInconnue", err)
	}
	_, err = svc.Cellule(context.Background(), domain.TacticalCelluleRequest{
		MapID: "streets", Question: domain.TacticalQuestionRoutes, Qui: domain.TacticalQuiMoi,
		Scope: domain.TacticalScope{MatchIDs: ids, Vies: "autre"}, Col: 10, Lig: 10,
	})
	if !errors.Is(err, domain.ErrTacticalViesInconnue) {
		t.Fatalf("Cellule : err = %v, attendu ErrTacticalViesInconnue", err)
	}
}

func celluleEnVies(t *testing.T, svc *TacticalService, vies, spawn string, ids []string, c int) domain.TacticalCelluleReponse {
	t.Helper()
	got, err := svc.Cellule(context.Background(), domain.TacticalCelluleRequest{
		MapID: "streets", Question: domain.TacticalQuestionRoutes, Qui: domain.TacticalQuiMoi,
		Scope: domain.TacticalScope{MatchIDs: ids, Spawn: spawn, Vies: vies}, Col: c, Lig: c,
	})
	if err != nil {
		t.Fatalf("Cellule (vies=%q, spawn=%q, cellule %d): %v", vies, spawn, c, err)
	}
	return got
}

// TestVies_CelluleToutes — le detail d'une cellule suit la lecture agregee : sous `toutes` et un
// filtre de grappe, il ne liste que les vies parties de la grappe ; sans filtre, toutes.
func TestVies_CelluleToutes(t *testing.T) {
	store, univ, ids := viesFixture()
	svc := svcArtefact(univ, store)
	grappes := lireEnVies(t, svc, domain.TacticalQuestionRoutes, domain.TacticalViesToutes, "", ids).Grappes
	reapparition := grappeAutour(t, grappes, 50, 50)

	if n := len(celluleEnVies(t, svc, domain.TacticalViesToutes, reapparition, ids, celluleDeLaVie[frameReapparait]).Contributions); n != 3 {
		t.Fatalf("route de la vie de la grappe : %d contributions, attendu 3 (un match chacune)", n)
	}
	if n := len(celluleEnVies(t, svc, domain.TacticalViesToutes, reapparition, ids, celluleDeLaVie[frameDepart]).Contributions); n != 0 {
		t.Fatalf("route d'une autre vie : %d contributions, attendu 0 sous le filtre", n)
	}
	if n := len(celluleEnVies(t, svc, domain.TacticalViesToutes, "", ids, celluleDeLaVie[frameDepart]).Contributions); n != 4 {
		t.Fatalf("sans filtre de grappe : %d contributions, attendu 4 (la route de la premiere vie, dans les 4 matchs)", n)
	}
}
