// Package service — elevation_cloud_section_test.go : le nuage « Dénivelé » vu du service
// (décision D25, proposition T5).
//
// Ce que ces tests verrouillent :
//
//  1. UNE SEULE LECTURE sert les deux blocs (portée par arme ET nuage) — deux lectures
//     auraient doublé l'emprunt du lecteur partagé et pu diverger ;
//  2. la couverture vient du SCOPE CANONIQUE, pas de la table de positions ;
//  3. AUCUN SEUIL sur le nuage : une arme à un seul frag, écartée de la portée par arme,
//     garde son point ;
//  4. les libellés d'arme sont un DICTIONNAIRE, best-effort — clé inconnue, pas d'entrée ;
//  5. un côté vide n'a pas de quartile (pointeurs nils), et `N` dit zéro.
package service

import (
	"context"
	"errors"
	"math"
	"testing"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/port"
)

// TestElevationUneSeuleLecturePourLesDeuxBlocs — le point d'architecture du lot.
func TestElevationUneSeuleLecturePourLesDeuxBlocs(t *testing.T) {
	repo := &mockWeaponRangeRepo{kills: wrKills("br", analysis.SideKiller, 10, 20, 2)}
	block, elevation := buildWeaponRangeSections(context.Background(), wrQuery(repo, wrCanonRows(1, 12, 4)))
	if block == nil || elevation == nil {
		t.Fatalf("les deux blocs attendus, got %v / %v", block, elevation)
	}
	if repo.killCalls != 1 {
		t.Errorf("une seule lecture des frags mesures attendue, got %d", repo.killCalls)
	}
	if len(elevation.Kills) != 10 {
		t.Errorf("10 points cote frags attendus, got %d", len(elevation.Kills))
	}
	if len(elevation.Deaths) != 0 {
		t.Errorf("aucun point cote morts attendu, got %d", len(elevation.Deaths))
	}
}

// TestElevationCouvertureDuScopeCanonique — le dénominateur est ce que le joueur a fait.
func TestElevationCouvertureDuScopeCanonique(t *testing.T) {
	kills := append(wrKills("br", analysis.SideKiller, 9, 20, 1),
		wrKills("ar", analysis.SideVictim, 4, 8, -1)...)
	_, elevation := buildWeaponRangeSections(context.Background(),
		wrQuery(&mockWeaponRangeRepo{kills: kills}, wrCanonRows(2, 15, 10)))
	if elevation == nil {
		t.Fatal("bloc attendu")
	}
	if elevation.MeasuredKills != 9 || elevation.TotalKills != 30 {
		t.Errorf("couverture des frags : 9/30 attendu, got %d/%d",
			elevation.MeasuredKills, elevation.TotalKills)
	}
	if elevation.MeasuredDeaths != 4 || elevation.TotalDeaths != 20 {
		t.Errorf("couverture des morts : 4/20 attendu, got %d/%d",
			elevation.MeasuredDeaths, elevation.TotalDeaths)
	}
}

// TestElevationAucunSeuilDePublication — la dimension arme ne compte pas (D23-b), donc le
// seuil de 8 mesures de la portée par arme ne s'applique PAS au nuage.
func TestElevationAucunSeuilDePublication(t *testing.T) {
	repo := &mockWeaponRangeRepo{kills: []analysis.MeasuredKill{
		wrKill("marteau", analysis.SideKiller, 1, 2, 0.5),
	}}
	block, elevation := buildWeaponRangeSections(context.Background(), wrQuery(repo, wrCanonRows(1, 1, 0)))
	if len(block.Weapons) != 0 {
		t.Fatalf("portee par arme : aucune arme publiable attendue, got %d", len(block.Weapons))
	}
	if elevation == nil || len(elevation.Kills) != 1 {
		t.Fatalf("nuage : le point unique doit survivre, got %v", elevation)
	}
	if elevation.KillsSummary.N != 1 {
		t.Errorf("N = 1 attendu, got %d", elevation.KillsSummary.N)
	}
}

// TestElevationQuartilesEtCoteVide — pointeurs posés d'un côté, nils de l'autre.
func TestElevationQuartilesEtCoteVide(t *testing.T) {
	kills := []analysis.MeasuredKill{}
	for i, d := range []float64{10, 20, 30, 40, 50} {
		kills = append(kills, wrKill("br", analysis.SideKiller, int64(i), d, 2))
	}
	_, elevation := buildWeaponRangeSections(context.Background(),
		wrQuery(&mockWeaponRangeRepo{kills: kills}, wrCanonRows(1, 5, 0)))
	k := elevation.KillsSummary
	if k.DistanceP25 == nil || k.DistanceP50 == nil || k.DistanceP75 == nil {
		t.Fatalf("quartiles de distance attendus, got %+v", k)
	}
	if math.Abs(*k.DistanceP50-30) > epsRange {
		t.Errorf("mediane de distance : 30 attendu, got %v", *k.DistanceP50)
	}
	if k.DeltaZP50 == nil || math.Abs(*k.DeltaZP50-2) > epsRange {
		t.Errorf("mediane de denivele : 2 attendu, got %v", k.DeltaZP50)
	}
	d := elevation.DeathsSummary
	if d.N != 0 || d.DistanceP50 != nil || d.DeltaZP50 != nil {
		t.Errorf("cote vide : N=0 et quartiles nils attendus, got %+v", d)
	}
}

// TestElevationLibellesDArme — dictionnaire hydraté, clé inconnue sans entrée.
func TestElevationLibellesDArme(t *testing.T) {
	repo := &mockWeaponRangeRepo{
		kills: []analysis.MeasuredKill{
			wrKill("br", analysis.SideKiller, 1, 20, 1),
			wrKill("inconnue", analysis.SideVictim, 2, 5, -1),
		},
		labels: map[string]port.WeaponLabel{"br": {Label: "Fusil de precision", LabelEN: "Battle Rifle"}},
	}
	_, elevation := buildWeaponRangeSections(context.Background(), wrQuery(repo, wrCanonRows(1, 1, 1)))
	if got := elevation.WeaponLabels["br"].Label; got != "Fusil de precision" {
		t.Errorf("libelle FR attendu, got %q", got)
	}
	if got := elevation.WeaponLabels["br"].LabelEN; got != "Battle Rifle" {
		t.Errorf("libelle EN attendu, got %q", got)
	}
	if _, ok := elevation.WeaponLabels["inconnue"]; ok {
		t.Error("une cle sans libelle ne doit PAS avoir d'entree (le front retombe sur la cle)")
	}
}

// TestElevationLibellesEnEchec — la panne de metadata n'emporte pas le nuage.
func TestElevationLibellesEnEchec(t *testing.T) {
	repo := &mockWeaponRangeRepo{
		kills:     wrKills("br", analysis.SideKiller, 3, 20, 1),
		labelsErr: errors.New("metadata indisponible"),
	}
	_, elevation := buildWeaponRangeSections(context.Background(), wrQuery(repo, wrCanonRows(1, 3, 0)))
	if elevation == nil || len(elevation.Kills) != 3 {
		t.Fatalf("nuage servi malgre l'echec des libelles attendu, got %v", elevation)
	}
	if elevation.WeaponLabels != nil {
		t.Errorf("aucun dictionnaire attendu apres echec, got %v", elevation.WeaponLabels)
	}
}

// TestElevationAbsenteQuandRienNEstMesure — pas de bloc vide.
func TestElevationAbsenteQuandRienNEstMesure(t *testing.T) {
	_, elevation := buildWeaponRangeSections(context.Background(),
		wrQuery(&mockWeaponRangeRepo{}, wrCanonRows(1, 9, 5)))
	if elevation != nil {
		t.Errorf("aucun bloc attendu sans frag mesure, got %v", elevation)
	}
}
