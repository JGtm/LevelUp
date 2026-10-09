// Package service — tactical_service_voisines_test.go : les lectures voisines servies avec la
// lecture demandee sont EXACTEMENT celles qu'une requete directe rendrait, et sortent de la
// MEME lecture de la base (une seule lecture des positions, un seul passage sur les sidecars).
package service

import (
	"context"
	"reflect"
	"testing"

	"levelup/go-api/internal/domain"
)

// corpusDesPositions : trois matchs (deux victoires, une defaite) ou je tue en (2,2) et ou je
// tombe en (10,10) — de quoi peupler morts, frags, solde et victoires − defaites.
func corpusDesPositions() *mockTacticalRepo {
	repo := &mockTacticalRepo{}
	repo.pos.Univers = domain.TacticalUnivers{Equipes: domain.EquipesParMatch{}}
	issues := map[string]int{"m1": domain.OutcomeWin, "m2": domain.OutcomeWin, "m3": domain.OutcomeLoss, "m4": domain.OutcomeLoss}
	for _, id := range []string{"m1", "m2", "m3", "m4"} {
		u := universUnMatch(id, issues[id])
		repo.pos.Univers.Matchs = append(repo.pos.Univers.Matchs, u.Matchs...)
		repo.pos.Univers.Equipes[id] = u.Equipes[id]
		repo.pos.Points = append(repo.pos.Points,
			domain.TacticalKillPosition{MatchID: id, KillerXUID: tsMoi, VictimXUID: tsAdv,
				KillerX: 2.0, KillerY: 2.0, VictimX: 6.0, VictimY: 6.0},
			domain.TacticalKillPosition{MatchID: id, KillerXUID: tsAdv, VictimXUID: tsMoi,
				KillerX: 14.0, KillerY: 14.0, VictimX: 10.0, VictimY: 10.0},
		)
	}
	return repo
}

// sansVoisinesNiZones : une lecture telle qu'une voisine la porte — ni zones ni voisines.
func sansVoisinesNiZones(r domain.TacticalRaster) domain.TacticalRaster {
	r.Voisines, r.Zones = nil, nil
	return r
}

// TestVoisines_Positions_UneLectureQuatreGrilles : demander « morts » sert aussi frags, solde et
// victoires − defaites, chacune identique a sa lecture directe, pour UNE lecture des positions.
func TestVoisines_Positions_UneLectureQuatreGrilles(t *testing.T) {
	repo := corpusDesPositions()
	svc := NewTacticalService(repo, capsPositionsSeules(), tsMoi)

	morts, err := svc.Raster(context.Background(), tsDemande(repo, tsCarte, domain.TacticalQuestionMorts, domain.TacticalQuiMoi))
	if err != nil {
		t.Fatalf("Raster(morts): %v", err)
	}
	if repo.lecturesPositions != 1 {
		t.Fatalf("lectures des positions = %d, attendu 1 (les voisines ne relisent rien)", repo.lecturesPositions)
	}
	attendues := []string{domain.TacticalQuestionKills, domain.TacticalQuestionSolde, domain.TacticalQuestionGagne}
	if len(morts.Voisines) != len(attendues) {
		t.Fatalf("voisines = %d, attendu %d", len(morts.Voisines), len(attendues))
	}
	for i, q := range attendues {
		voisine := morts.Voisines[i]
		if voisine.Question != q {
			t.Fatalf("voisine %d = %q, attendu %q", i, voisine.Question, q)
		}
		directe, err := svc.Raster(context.Background(), tsDemande(repo, tsCarte, q, domain.TacticalQuiMoi))
		if err != nil {
			t.Fatalf("Raster(%s): %v", q, err)
		}
		if !reflect.DeepEqual(voisine, sansVoisinesNiZones(directe)) {
			t.Errorf("voisine %s differente de sa lecture directe :\n voisine %+v\n directe %+v", q, voisine, sansVoisinesNiZones(directe))
		}
		if len(voisine.Voisines) != 0 {
			t.Errorf("une voisine ne porte pas de voisines (%s)", q)
		}
	}
}

// TestVoisines_Sidecars_TempsServiAvecTrajets : demander « temps » sert « trajets », identique a
// sa lecture directe, en un seul passage sur les sidecars de l'univers.
func TestVoisines_Sidecars_TempsServiAvecTrajets(t *testing.T) {
	joueur := func(id string) domain.TacticalRasterJoueur {
		j := joueurEn(tsMoi, 2, 3, 8)
		j.Routes = []domain.TacticalRasterRoute{{DebutFrame: 10, Cases: []domain.TacticalRasterCase{{Col: 2, Lig: 3}, {Col: 3, Lig: 3}}}}
		return j
	}
	store := &mockRasterStore{sidecars: map[string]*domain.TacticalRasterSidecar{
		"m1": sidecarPose("m1", joueur("m1")),
		"m2": sidecarPose("m2", joueur("m2")),
		"m3": sidecarPose("m3", joueur("m3")),
	}}
	svc, _ := occupationSvc(universTroisMatchs("m1", "m2", "m3"), store)
	demande := func(q string) domain.TacticalRasterRequest {
		return domain.TacticalRasterRequest{
			MapID: "streets", Question: q, Qui: domain.TacticalQuiMoi,
			Scope: domain.TacticalScope{MatchIDs: []string{"m1", "m2", "m3"}},
		}
	}
	temps, err := svc.Raster(context.Background(), demande(domain.TacticalQuestionTemps))
	if err != nil {
		t.Fatalf("Raster(temps): %v", err)
	}
	if len(store.vus) != 3 {
		t.Fatalf("sidecars lus = %d, attendu 3 (un passage pour les deux lectures)", len(store.vus))
	}
	if len(temps.Voisines) != 1 || temps.Voisines[0].Question != domain.TacticalQuestionRoutes {
		t.Fatalf("voisines = %+v, attendu « routes » seule", temps.Voisines)
	}
	routes, err := svc.Raster(context.Background(), demande(domain.TacticalQuestionRoutes))
	if err != nil {
		t.Fatalf("Raster(routes): %v", err)
	}
	if len(routes.Cellules) == 0 {
		t.Fatal("le corpus doit peupler la lecture des trajets")
	}
	if !reflect.DeepEqual(temps.Voisines[0], sansVoisinesNiZones(routes)) {
		t.Errorf("voisine routes differente de sa lecture directe :\n voisine %+v\n directe %+v", temps.Voisines[0], sansVoisinesNiZones(routes))
	}
}

// TestVoisines_Isole_Aucune : « isole » a sa propre lecture et ne sert aucune voisine.
func TestVoisines_Isole_Aucune(t *testing.T) {
	ids := []string{"m1", "m2", "m3"}
	table := variantes(map[string]string{}, "Slayer:Arena", ids...)
	svc, _ := svcIsole(universVariantes(table), mortsDansTroisMatchs(ids, tsMoi, 4, 4, m(30), 1)...)
	out := lireIsole(t, svc, ids...)
	if len(out.Voisines) != 0 {
		t.Fatalf("voisines de « isole » = %+v, attendu aucune", out.Voisines)
	}
}
