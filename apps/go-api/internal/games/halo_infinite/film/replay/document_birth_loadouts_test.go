package replay

// document_birth_loadouts_test.go — LA PUBLICATION DES DOTATIONS DE NAISSANCE (schéma 69, lot
// M3.2 de la campagne « retours rejeu », 2026-09-23) : sur quelle vie elle se pose, ce qui n'est
// pas une arme, et l'ordre de `loadouts`.

import (
	"reflect"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// Deux armes que le catalogue nomme (MA40 AR, Mk51 Sidekick), et l'objet de départ du coup
// d'envoi que le catalogue ne nomme pas.
const (
	bdMA40     = 0x48C19D2D
	bdSidekick = 0xF408190F
	bdDepart   = 0x00007CA9
)

// bdPiste rend une piste [debut, fin] sur un slot.
func bdPiste(slot uint32, debut, fin int) Track {
	return Track{Slot: slot, StartFrame: debut, EndFrame: fin,
		Points: []Point{{T: debut}, {T: fin}}}
}

// bdNaissance rend une dotation (arme 1, arme 2, objet de départ) créée à la frame `f`.
func bdNaissance(slot uint32, f int) types.BirthLoadout {
	return types.BirthLoadout{TimestampUS: wcOrigin + uint64(f)*wcStep, Slot: slot, Generation: 1,
		Weapons: []types.BirthWeapon{{Emplacement: 0, Family: bdMA40}, {Emplacement: 1, Family: bdSidekick},
			{Emplacement: 2, Family: bdDepart}, {Emplacement: 3, Family: grammar.NoWeaponVariant}}}
}

// bdCreation rend la création de bipède qui porte la dotation.
func bdCreation(b types.BirthLoadout) grammar.BipedCreation {
	return grammar.BipedCreation{Slot: b.Slot, Generation: b.Generation, TimestampUS: b.TimestampUS}
}

// TestDotationSeposeSurLaVieQueSaCreationOuvre : deux vies du même slot. La première naît dans
// sa fenêtre ; la seconde ouvre sa piste APRÈS la création du corps (positions en retard) — le
// relevé est ramené sur la première frame de SA vie, jamais posé sur la fin de la précédente.
func TestDotationSeposeSurLaVieQueSaCreationOuvre(t *testing.T) {
	n1, n2 := bdNaissance(9, 2), bdNaissance(9, 148)
	in := birthInputs{births: []types.BirthLoadout{n1, n2}, stats: types.BirthLoadoutStats{Creations: 2, Read: 2},
		creations: []grammar.BipedCreation{bdCreation(n1), bdCreation(n2)}}
	// La vie 1 finit APRÈS la seconde création : c'est le piège que l'appariement ferme.
	pistes := []Track{bdPiste(9, 0, 160), bdPiste(9, 150, 300)}
	out, cov := buildBirthLoadouts(in, pistes, wcOrigin, wcStep)
	if len(out) != 2 || out[0].T != 2 || out[1].T != 150 {
		t.Fatalf("relevés %+v : attendu t=2 (vie 1) et t=150 (début de la vie 2)", out)
	}
	if cov.Published != 2 || cov.Snapped != 1 || cov.NoLife != 0 {
		t.Fatalf("couverture %+v : attendu 2 publiées dont 1 ramenée", *cov)
	}
}

// TestDotationPublieLesSeulesArmesDuCatalogue : l'objet de départ `00007CA9` et l'emplacement vide
// ne sont pas des armes ; les deux armes gardent leur emplacement.
func TestDotationPublieLesSeulesArmesDuCatalogue(t *testing.T) {
	n := bdNaissance(9, 10)
	in := birthInputs{births: []types.BirthLoadout{n}, stats: types.BirthLoadoutStats{Creations: 1, Read: 1},
		creations: []grammar.BipedCreation{bdCreation(n)}}
	out, cov := buildBirthLoadouts(in, []Track{bdPiste(9, 5, 100)}, wcOrigin, wcStep)
	want := Loadout{T: 10, Slot: 9, W: []string{"0x48C19D2D", "0xF408190F"}, Src: LoadoutSrcBirth, K: []int{0, 1}}
	if len(out) != 1 || !reflect.DeepEqual(out[0], want) {
		t.Fatalf("relevé %+v, attendu %+v", out, want)
	}
	if cov.NonWeapon != 1 {
		t.Fatalf("couverture %+v : l'objet de départ doit être compté comme non-arme", *cov)
	}
}

// TestDotationSansVieNEstPasPubliee : aucune piste du slot n'est appariée à la création.
func TestDotationSansVieNEstPasPubliee(t *testing.T) {
	n := bdNaissance(9, 400)
	in := birthInputs{births: []types.BirthLoadout{n}, stats: types.BirthLoadoutStats{Creations: 1, Read: 1},
		creations: []grammar.BipedCreation{bdCreation(n)}}
	out, cov := buildBirthLoadouts(in, []Track{bdPiste(9, 0, 100)}, wcOrigin, wcStep)
	if len(out) != 0 || cov.NoLife != 1 {
		t.Fatalf("relevés %+v, couverture %+v : attendu aucun relevé, 1 sans vie", out, *cov)
	}
}

// TestFilmSansCreationNePublieAucuneCouverture : pas de ligne de zéros.
func TestFilmSansCreationNePublieAucuneCouverture(t *testing.T) {
	if out, cov := buildBirthLoadouts(birthInputs{}, nil, wcOrigin, wcStep); out != nil || cov != nil {
		t.Fatalf("film sans création : relevés %+v, couverture %+v", out, cov)
	}
}

// TestMergeLoadoutsImageCleDAbordAInstantEgal : à frame et slot égaux, le relevé d'image-clé
// précède la dotation — c'est lui que l'inventaire du même instant suit.
func TestMergeLoadoutsImageCleDAbordAInstantEgal(t *testing.T) {
	images := []Loadout{{T: 10, Slot: 9, W: []string{"0xAAAAAAAA"}}, {T: 30, Slot: 9, W: []string{"0xBBBBBBBB"}}}
	naissances := []Loadout{{T: 10, Slot: 9, W: []string{"0xCCCCCCCC"}, Src: LoadoutSrcBirth}, {T: 5, Slot: 9, Src: LoadoutSrcBirth}}
	out := mergeLoadouts(images, naissances)
	var ordre []int
	for _, l := range out {
		ordre = append(ordre, l.T)
	}
	if !reflect.DeepEqual(ordre, []int{5, 10, 10, 30}) || out[1].Src != "" || out[2].Src != LoadoutSrcBirth {
		t.Fatalf("ordre %+v", out)
	}
}
