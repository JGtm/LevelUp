package grammar

// movement_states_sprint_test.go — LES DEUX RATCHETS DU SPRINT LU (lot 5.9.5).
//
// Le sprint ne se publie pas sur un seuil : il se LIT dans `ti=35 i57`, qui porte l INDEX DE LA
// FENTE DE CAPACITE ACTIVE. Deux valeurs decident de tout, et une erreur sur l une d elles
// publierait silencieusement une AUTRE capacite sous le nom du sprint. Ces tests les figent.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// TestSprintLitLaFenteUn fige le DECALAGE, et il est la raison d etre du test.
//
// Le flux lit `R(2)` et l ecrivain pose `bloc+3 = valeur - 1` (`FUN_142f268c4`) : la fente `1`
// se lit donc `2` dans le flux. Publier la fente `1` en comparant a `1` publierait la fente 0,
// c est-a-dire l ESQUIVE, sous le nom du sprint — et rien ne le dirait, parce que les deux sont
// des capacites plausibles.
func TestSprintLitLaFenteUn(t *testing.T) {
	if sprintAbilitySlotRaw != 2 {
		t.Fatalf("sprintAbilitySlotRaw = %d, attendu 2 : le flux ecrit la fente DECALEE DE +1 "+
			"(`FUN_142f268c4` pose `bloc+3 = R(2) - 1`), et la fente du sprint est la 1 — "+
			"'sasp' est desenregistre par `FUN_14319d1ec`, qui teste l index actif contre "+
			"`comp+0x20`", sprintAbilitySlotRaw)
	}
}

// TestSprintEstUnGenreLU est le pendant de `TestSautDeriveNommeSonGenre` : le sprint porte un nom
// NU, sans le mot « derive », et c est ce qui le distingue du saut.
//
// LES TROIS FENTES SONT NOMMEES PAR L IMAGE, pas par un score (`FUN_1407e9ce4` aiguille sur le
// groupe de tag de la definition et appelle, pour chacun, un desenregistreur qui teste l index
// actif contre SA fente) :
//
//	'saev' (0x73616576, esquive) -> FUN_14319d0ac : fente `comp+0x1c`, index 0
//	'sasp' (0x73617370, SPRINT)  -> FUN_14319d1ec : fente `comp+0x20`, index 1
//	'sagh' (0x73616768, grappin) -> FUN_14319d14c : fente `comp+0x24`, index 2
func TestSprintEstUnGenreLU(t *testing.T) {
	if types.MovementSprint != "sprint" {
		t.Fatalf("genre du sprint = %q, attendu \"sprint\" : il est LU dans `i57`, donc son nom "+
			"ne porte AUCUN mot de derivation — contrairement a %q",
			types.MovementSprint, types.MovementJumpDerived)
	}
	if types.MovementSprint == types.MovementJumpDerived {
		t.Fatal("le sprint et le saut ne peuvent pas partager un genre : l un est lu, l autre calcule")
	}
}
