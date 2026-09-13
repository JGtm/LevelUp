// Package service — synthesis_weapon_range_build_test.go : les FONCTIONS PURES de la section
// « Portée par arme » (`mergeWeaponSides` et `buildOpening`, synthesis_weapon_range_build.go).
//
// Pourquoi ce fichier est séparé de `synthesis_weapon_range_test.go` : ce dernier verrouille
// le SERVICE (port mocké, dégradations, régime de journalisation) et a franchi le seuil de
// 500 lignes du CLAUDE.md à la revue du lot 4. La coupure suit celle du code testé —
// `synthesis_weapon_range.go` (service) d'un côté, `synthesis_weapon_range_build.go` (pur) de
// l'autre. Aucun test n'a été modifié à la scission, seulement déplacé (lot 6, item 6.0b) :
// les deux tests d'entame gardent donc leur nom `TestLoadWeaponRange_*`, hérité de l'écriture
// du lot 4, alors que ce qu'ils épinglent est le contrat de `buildOpening`.
//
// Ce que ces tests verrouillent :
//
//  1. LE BLOC D'ENTAME EST NIL, JAMAIS UN ZÉRO (D5), et son sous-bloc `delta` l'est aussi
//     quand aucun frag ne porte les deux mesures. Un `SynthesisOpening{}` publié se lirait
//     « ce joueur engage au contact » alors que la vérité est « on ne sait pas » — et c'est
//     l'état NOMINAL tant que le backfill de kill_openings n'a pas tourné ;
//  2. l'ordre de lecture du graphe (D6) : médiane des FRAGS croissante, une arme mesurée d'un
//     seul côté se rangeant à la médiane de ce côté-là.
//
// Les fixtures (`wrKills`, `wrCanonRows`, `wrService`, `mockWeaponRangeRepo`) restent dans le
// fichier service : elles sont partagées, et le paquet de test est le même.
package service

import (
	"context"
	"math"
	"testing"

	"levelup/go-api/internal/analysis"
)

// TestLoadWeaponRange_EntamesSansCoupFatalMesure_DeltaOmis — le sous-bloc `delta` est NIL
// quand aucun frag ne porte les DEUX mesures (constat F9, revue adversariale du lot 4,
// 2026-09-06).
//
// LE CAS EST ATTEIGNABLE : `kill_positions` et `kill_openings` s'écrivent sous deux leases
// indépendants, et un scope peut porter des entames dont aucun coup fatal n'est placé. À plat,
// il publiait `closing_share_pct: 0` — un champ requis, donc toujours présent — qui se lit
// « ce joueur ne ferme jamais la distance » alors qu'aucune mesure ne le dit. La couverture de
// l'entame, elle, reste publiée : c'est un fait mesuré.
func TestLoadWeaponRange_EntamesSansCoupFatalMesure_DeltaOmis(t *testing.T) {
	kills := wrKills("hinf_br75", analysis.SideKiller, 9, 12, 0)
	// Les entames portent des instants qu'AUCUN coup fatal mesuré ne porte : rien n'apparie.
	openings := wrKills("hinf_br75", analysis.SideKiller, 9, 30, 0)
	for i := range openings {
		openings[i].TimeMS += 500000
	}
	repo := &mockWeaponRangeRepo{kills: kills, openings: openings}

	block := wrService(repo).loadWeaponRange(context.Background(), wrCanonRows(1, 9, 5))
	if block == nil || block.Opening == nil {
		t.Fatalf("bloc d'entame absent : %+v — 9 entames sont pourtant mesurées", block)
	}
	if block.Opening.MeasuredKills != 9 || math.Abs(block.Opening.MedianM-30) > epsRange {
		t.Errorf("entame = %d mesures à %v m, attendu 9 à 30 m (la couverture reste publiée)",
			block.Opening.MeasuredKills, block.Opening.MedianM)
	}
	if block.Opening.Delta != nil {
		t.Errorf("sous-bloc delta = %+v, attendu nil : aucun frag ne porte les deux mesures, "+
			"et un zéro publié se lirait « la distance ne bouge jamais »", *block.Opening.Delta)
	}
}

// TestLoadWeaponRange_SansEntame_BlocNilJamaisZero — D5. C'est l'état NOMINAL tant que le
// backfill de kill_openings n'a pas tourné : la portée se publie, l'entame ne s'invente pas.
func TestLoadWeaponRange_SansEntame_BlocNilJamaisZero(t *testing.T) {
	repo := &mockWeaponRangeRepo{kills: wrKills("hinf_br75", analysis.SideKiller, 9, 12, 0)}

	block := wrService(repo).loadWeaponRange(context.Background(), wrCanonRows(1, 9, 5))
	if block == nil {
		t.Fatal("section nil : l'absence d'entame ne doit PAS emporter la portée")
	}
	if block.Opening != nil {
		t.Errorf("bloc d'entame = %+v, attendu nil (un zéro se lirait « engage au contact »)",
			*block.Opening)
	}
}

// TestMergeWeaponSides_TriEtCoteUnique — l'ordre de lecture du graphe (D6) : médiane des
// FRAGS croissante, et une arme sans frag publié se range à la médiane de ses MORTS.
func TestMergeWeaponSides_TriEtCoteUnique(t *testing.T) {
	kills := wrKills("hinf_sniper", analysis.SideKiller, 9, 40, 0)                  // frags à 40 m
	kills = append(kills, wrKills("hinf_br75", analysis.SideKiller, 9, 12, 0)...)   // frags à 12 m
	kills = append(kills, wrKills("hinf_shotgun", analysis.SideVictim, 9, 3, 0)...) // morts seules, 3 m
	repo := &mockWeaponRangeRepo{kills: kills}

	block := wrService(repo).loadWeaponRange(context.Background(), wrCanonRows(1, 30, 10))
	if block == nil || len(block.Weapons) != 3 {
		t.Fatalf("armes = %+v, attendu 3 lignes", block)
	}
	ordre := []string{block.Weapons[0].WeaponKey, block.Weapons[1].WeaponKey, block.Weapons[2].WeaponKey}
	attendu := []string{"hinf_shotgun", "hinf_br75", "hinf_sniper"}
	for i := range attendu {
		if ordre[i] != attendu[i] {
			t.Fatalf("ordre = %v, attendu %v (médiane croissante, morts seules incluses)", ordre, attendu)
		}
	}
	if block.Weapons[0].Kills != nil {
		t.Errorf("le fusil à pompe ne porte aucun frag : Kills = %+v, attendu nil", block.Weapons[0].Kills)
	}
	if block.Weapons[0].Deaths == nil {
		t.Error("le fusil à pompe doit porter son côté morts")
	}
}

// TestMergeWeaponSides_LaMedianeDesFragsPrimeSurCelleDesMorts — la règle de tri de
// `mergeWeaponSides` (constat F4, revue adversariale du lot 4, 2026-09-06).
//
// Une arme mesurée DES DEUX CÔTÉS se range à la médiane de ses FRAGS, jamais à celle de ses
// morts : le graphe se lit « où je frague », les morts en sont le contrepoint. La règle
// n'était épinglée par rien — le témoin `if out[i].Kills == nil` remplacé par `if true`
// restait vert, et le tri basculait silencieusement sur le dernier côté rencontré.
//
// FIXTURE CONSTRUITE POUR QUE LA MUTATION SE VOIE : le BR75 frague à 12 m et tue son porteur à
// 20 m, l'Hydra ne frague qu'à 15 m. Par la médiane des frags -> BR75 (12) puis Hydra (15) ;
// par celle des morts -> Hydra (15) puis BR75 (20). L'ordre s'inverse.
func TestMergeWeaponSides_LaMedianeDesFragsPrimeSurCelleDesMorts(t *testing.T) {
	kills := wrKills("hinf_br75", analysis.SideKiller, 9, 12, 0)
	kills = append(kills, wrKills("hinf_br75", analysis.SideVictim, 9, 20, 0)...)
	kills = append(kills, wrKills("hinf_hydra", analysis.SideKiller, 9, 15, 0)...)
	repo := &mockWeaponRangeRepo{kills: kills}

	block := wrService(repo).loadWeaponRange(context.Background(), wrCanonRows(1, 30, 12))
	if block == nil || len(block.Weapons) != 2 {
		t.Fatalf("armes = %+v, attendu 2 lignes", block)
	}
	if block.Weapons[0].WeaponKey != "hinf_br75" || block.Weapons[1].WeaponKey != "hinf_hydra" {
		t.Fatalf("ordre = [%s %s], attendu [hinf_br75 hinf_hydra] : le BR75 se range à la "+
			"médiane de ses FRAGS (12 m), pas à celle de ses morts (20 m)",
			block.Weapons[0].WeaponKey, block.Weapons[1].WeaponKey)
	}
	if block.Weapons[0].Kills == nil || block.Weapons[0].Deaths == nil {
		t.Errorf("le BR75 doit porter ses deux côtés : %+v", block.Weapons[0])
	}
}
