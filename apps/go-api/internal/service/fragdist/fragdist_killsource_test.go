// fragdist_killsource_test.go — la 3e provenance du sunburst : la SOURCE DE DEGAT du
// film (lot « kills hors arme a feu », 2026-08-29).
//
// Ce que ces tests verrouillent :
//
//  1. NON-REGRESSION ABSOLUE : sources vide => sortie BYTE-IDENTIQUE a l'ancienne.
//     C'est le test qui protege Halo 5 et tous les matchs sans film ;
//  2. les kills sortent du RESIDU et de nulle part ailleurs (invariant a tenu, aucune
//     classe d'arme touchee) — c'est le critere de succes n2 du plan ;
//  3. le niveau 2 est par OBJET, avec le libelle du registre ;
//  4. le residu ne peut pas devenir negatif (invariant c).
package fragdist

import (
	"reflect"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// killSourceRows : 3 bobines + 1 repulseur + de la chute, tels que le repo les rend.
func killSourceRows() []port.KillSourceClassRow {
	return []port.KillSourceClassRow{
		{XUID: "x1", WeaponKey: "hinf_coil_kinetic", Class: domain.FragClassEnvironmental,
			Label: "Bobine à fusion UNSC", Kills: 3},
		{XUID: "x1", WeaponKey: "hinf_coil_plasma", Class: domain.FragClassEnvironmental,
			Label: "Bobine à plasma", Kills: 1},
		{XUID: "x1", WeaponKey: "hinf_environment", Class: domain.FragClassEnvironmental,
			Label: "Chute et environnement", Kills: 2},
		{XUID: "x1", WeaponKey: "hinf_repulsor", Class: domain.FragClassEquipment,
			Label: "Répulseur", Kills: 1},
	}
}

func classByKey(fd domain.FragDistribution) map[string]domain.FragClassEntry {
	out := map[string]domain.FragClassEntry{}
	for _, c := range fd.Classes {
		out[c.Class] = c
	}
	return out
}

// TestKillSource_SourcesVideEstByteIdentique : LE test de non-regression. Avec la meme
// entree et `sources` vide (nil ou tranche vide), la sortie doit etre exactement celle
// d'avant l'ajout de la provenance.
func TestKillSource_SourcesVideEstByteIdentique(t *testing.T) {
	counts := domain.FragKillTypeCounts{Melee: 4, Grenade: 3, Total: 40}
	ref := Build(heterogeneousInfiniteRows(), nil, counts, false)

	if got := Build(heterogeneousInfiniteRows(), []port.KillSourceClassRow{}, counts, false); !reflect.DeepEqual(got, ref) {
		t.Errorf("tranche vide != nil :\n got %+v\nwant %+v", got, ref)
	}
	// Une ligne a 0 kill, ou sans classe, ou sans cle : ignoree comme si elle n'existait
	// pas. Une classe vide dans le sunburst serait un arc invisible qui decale les
	// couleurs.
	bruit := []port.KillSourceClassRow{
		{XUID: "x1", WeaponKey: "hinf_repulsor", Class: domain.FragClassEquipment, Kills: 0},
		{XUID: "x1", WeaponKey: "hinf_repulsor", Class: "", Kills: 5},
		{XUID: "x1", WeaponKey: "", Class: domain.FragClassEquipment, Kills: 5},
	}
	if got := Build(heterogeneousInfiniteRows(), bruit, counts, false); !reflect.DeepEqual(got, ref) {
		t.Errorf("lignes non exploitables prises en compte :\n got %+v\nwant %+v", got, ref)
	}
}

// TestKillSource_DecoupeLeResiduEtRienDAutre : le critere de succes n2 du plan. Les
// nouvelles classes prennent EXACTEMENT leurs kills sur « Non attribue » ; aucune classe
// d'arme ne bouge d'une unite, et la somme reste egale au total.
func TestKillSource_DecoupeLeResiduEtRienDAutre(t *testing.T) {
	counts := domain.FragKillTypeCounts{Melee: 4, Grenade: 3, Total: 60}
	avant := classByKey(Build(heterogeneousInfiniteRows(), nil, counts, false))
	apres := classByKey(Build(heterogeneousInfiniteRows(), killSourceRows(), counts, false))

	for class, e := range avant {
		if class == domain.FragClassUnattributed {
			continue
		}
		if apres[class].Kills != e.Kills {
			t.Errorf("classe %q : %d -> %d kills — le lot ne doit RIEN retirer aux autres",
				class, e.Kills, apres[class].Kills)
		}
	}
	// 7 kills reclasses (3+1+2 environnement, 1 equipement) : le residu perd exactement ca.
	if got, want := apres[domain.FragClassUnattributed].Kills,
		avant[domain.FragClassUnattributed].Kills-7; got != want {
		t.Errorf("residu = %d, want %d (il perd exactement les kills reclasses)", got, want)
	}
	if apres[domain.FragClassEnvironmental].Kills != 6 {
		t.Errorf("environnement = %d kills, want 6", apres[domain.FragClassEnvironmental].Kills)
	}
	if apres[domain.FragClassEquipment].Kills != 1 {
		t.Errorf("equipement = %d kills, want 1", apres[domain.FragClassEquipment].Kills)
	}
	// Invariant (a).
	sum := 0
	for _, e := range apres {
		sum += e.Kills
	}
	if sum != counts.Total {
		t.Errorf("somme des classes = %d, want %d (invariant a)", sum, counts.Total)
	}
}

// TestKillSource_NiveauDeuxParObjet : « Bobine a plasma » est une information,
// « environnement » n'en est pas une. Tri par kills decroissants, libelle du registre,
// et invariant (b) : la somme du niveau 2 fait le total de la classe.
func TestKillSource_NiveauDeuxParObjet(t *testing.T) {
	fd := Build(nil, killSourceRows(), domain.FragKillTypeCounts{Total: 20}, false)
	env := classByKey(fd)[domain.FragClassEnvironmental]

	if len(env.Roles) != 3 {
		t.Fatalf("%d objets au niveau 2, want 3 : %+v", len(env.Roles), env.Roles)
	}
	if env.Roles[0].Role != "hinf_coil_kinetic" || env.Roles[0].Kills != 3 {
		t.Errorf("premier objet = %+v, want hinf_coil_kinetic a 3 kills (tri kills desc)",
			env.Roles[0])
	}
	if env.Roles[0].Label != "Bobine à fusion UNSC" {
		t.Errorf("libelle = %q, want celui du registre", env.Roles[0].Label)
	}
	sum := 0
	for _, r := range env.Roles {
		sum += r.Kills
	}
	if sum != env.Kills {
		t.Errorf("somme du niveau 2 = %d, want %d (invariant b)", sum, env.Kills)
	}
}

// TestKillSource_ResiduJamaisNegatif : invariant (c). Si les sources depassent le total
// (donnee incoherente : film decode d'un match dont les compteurs API disent moins), le
// residu n'est PAS ajoute — il ne devient jamais negatif.
func TestKillSource_ResiduJamaisNegatif(t *testing.T) {
	fd := Build(nil, killSourceRows(), domain.FragKillTypeCounts{Total: 2}, false)
	for _, c := range fd.Classes {
		if c.Kills < 0 {
			t.Errorf("classe %q a %d kills", c.Class, c.Kills)
		}
		if c.Class == domain.FragClassUnattributed {
			t.Errorf("residu present (%d) alors que les sources depassent le total", c.Kills)
		}
	}
}

// TestKillSource_OrdreCanonique : equipement et environnement se placent apres les
// engins et AVANT le residu. L'ordre est un contrat d'affichage, pas un detail.
func TestKillSource_OrdreCanonique(t *testing.T) {
	fd := Build(nil, killSourceRows(), domain.FragKillTypeCounts{Total: 20}, false)
	var order []string
	for _, c := range fd.Classes {
		order = append(order, c.Class)
	}
	want := []string{
		domain.FragClassEquipment, domain.FragClassEnvironmental, domain.FragClassUnattributed,
	}
	if !reflect.DeepEqual(order, want) {
		t.Errorf("ordre = %v, want %v", order, want)
	}
}

// TestKillSource_ClassesHorsBreakdownParArme : equipement et environnement ne doivent
// PAS entrer dans le breakdown par-arme ni dans le graphe de precision — un repulseur
// n'a pas de « tir au but », une bobine n'est pas un style de jeu.
func TestKillSource_ClassesHorsBreakdownParArme(t *testing.T) {
	for _, class := range []string{domain.FragClassEquipment, domain.FragClassEnvironmental} {
		if !domain.IsNonCombatFragClass(class) {
			t.Errorf("%s : IsNonCombatFragClass = false, want true", class)
		}
		if domain.WeaponClassHasAccuracy(class) {
			t.Errorf("%s : WeaponClassHasAccuracy = true, want false", class)
		}
	}
}
