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

// Deux armes que le catalogue nomme (MA40 AR, Mk51 Sidekick), et l'objet « mains nues » que le jeu
// remet au troisième emplacement de chaque bipède (sonde CA9, règle nommée `filmshell.IsUnarmedFamily`).
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

// TestDotationPublieLesSeulesArmesDuCatalogue : l'objet « mains nues » `00007CA9` et l'emplacement
// vide ne sont pas des armes ; les deux armes gardent leur emplacement. LES MAINS NUES SE COMPTENT
// A PART (`unarmedGrants`, règle nommée de M6.3, rebranchée à la fusion de la campagne à jour, lot
// D-fix) : ROUGE avant la fusion, où la dotation de naissance les comptait en `nonWeapon` faute
// de la règle.
func TestDotationPublieLesSeulesArmesDuCatalogue(t *testing.T) {
	n := bdNaissance(9, 10)
	in := birthInputs{births: []types.BirthLoadout{n}, stats: types.BirthLoadoutStats{Creations: 1, Read: 1},
		creations: []grammar.BipedCreation{bdCreation(n)}}
	out, cov := buildBirthLoadouts(in, []Track{bdPiste(9, 5, 100)}, wcOrigin, wcStep)
	want := Loadout{T: 10, Slot: 9, W: []string{"0x48C19D2D", "0xF408190F"}, Src: LoadoutSrcBirth, K: []int{0, 1}}
	if len(out) != 1 || !reflect.DeepEqual(out[0], want) {
		t.Fatalf("relevé %+v, attendu %+v", out, want)
	}
	if cov.UnarmedGrants != 1 || cov.NonWeapon != 0 {
		t.Fatalf("couverture %+v : les mains nues se comptent en unarmedGrants, pas en nonWeapon", *cov)
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

// TestFermetureSansArmeNEstPasUneLecture : un record qui se FERME mais dont aucun emplacement ne
// porte une arme du catalogue n'est pas une dotation lue (constat F3 de la revue adverse du lot
// M3.2, 2026-09-24 : sur les builds antérieurs à HI_1_12_0, `read` annonçait 11 dotations
// « lues » dont aucune n'avait d'arme). ROUGE sur 73dda0fb5 : `read` recopiait les fermetures.
// La partition est `closed` = `read` + `noDisplayable`, et `read` = `published` + `noLife` +
// `beforeOrigin`.
func TestFermetureSansArmeNEstPasUneLecture(t *testing.T) {
	arme := bdNaissance(9, 10)
	bruit := types.BirthLoadout{TimestampUS: wcOrigin + 50*wcStep, Slot: 11, Generation: 1,
		Weapons: []types.BirthWeapon{{Emplacement: 0, Family: 0x12345678}, {Emplacement: 1, Family: bdDepart}}}
	sansVie := bdNaissance(13, 400)
	in := birthInputs{births: []types.BirthLoadout{arme, bruit, sansVie},
		stats:     types.BirthLoadoutStats{Creations: 3, Read: 3},
		creations: []grammar.BipedCreation{bdCreation(arme), bdCreation(bruit), bdCreation(sansVie)}}
	pistes := []Track{bdPiste(9, 5, 100), bdPiste(11, 40, 120), bdPiste(13, 0, 100)}
	out, cov := buildBirthLoadouts(in, pistes, wcOrigin, wcStep)
	if len(out) != 1 {
		t.Fatalf("relevés %+v : attendu la seule dotation armée du slot 9", out)
	}
	// Hors catalogue : la seule famille `0x12345678` ; les trois mains nues sont les remises.
	want := BirthLoadoutCoverage{Creations: 3, Closed: 3, Read: 2, Published: 1, NoLife: 1,
		NonWeapon: 1, UnarmedGrants: 3, NoDisplayable: 1}
	if *cov != want {
		t.Fatalf("couverture %+v, attendu %+v", *cov, want)
	}
}
