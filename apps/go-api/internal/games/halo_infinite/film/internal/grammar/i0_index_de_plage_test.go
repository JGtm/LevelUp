package grammar

// i0_index_de_plage_test.go — LE CHEMIN ABSOLU D i0 SUIT L INDEX DE PLAGE (correctif D1 (3.4),
// lot 3.4.1-b).
//
// # LE DEFAUT QUE CES TEMOINS FERMENT
//
// `consumeAbsolutePayload` portait le commentaire « index 0 = the map replication bounds ;
// index 1 / no-index = +/-20000 (off-map) » et le code en tirait `if idx != 0 { return }`. Le
// desassemblage de `FUN_14076e524` ne donne les bornes `+/-20000` que pour `index == -1` (bit de
// porte pose) ; tout `index >= 0` adresse `DAT_14462cbe0 + index*0x18`, une plage REELLE. Sur
// une carte dont la plage jouee n est pas la 0 — Live Fire, `region = 1` sur 2 bits, 4 plages
// declarees par `ds/globals/common` — le filtre gardait donc exactement ce qu il fallait jeter
// et jetait ce qu il fallait garder : 59 376 des 59 377 records i0 de ses deux films portent
// l index 01.
//
// # POURQUOI DES FLUX SYNTHETIQUES ET PAS UN FILM
//
// La regle se prouve au BIT, et un flux fabrique la prouve dans les DEUX sens — il porte l index
// qu on veut, ce qu un film ne fait pas. Les deux films Live Fire du corpus restent la mesure
// de PORTEE (combien de positions sont gagnees), pas la mesure de REGLE ; ils exigent un
// decodage, ces temoins non.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// carteALaPlageUn : l entree de catalogue de Live Fire, reduite a ce que la regle lit — quatre
// plages declarees (index sur 2 bits), l arene est la 1, largeurs d axe 12/12/11.
func carteALaPlageUn() profile.MapQuantEntry {
	return profile.MapQuantEntry{
		Module: "sgh_interlock", AxisWidths: [3]uint{12, 12, 11},
		Region: 1, RegionIndexBits: 2,
		Min: [3]float32{-30, -30, -10}, Max: [3]float32{30, 30, 10},
	}
}

// fluxAbsoluAvecIndex fabrique le flux d un chemin absolu direct d i0 portant `idx` sur `idxW`
// bits, puis trois axes a zero. En-tete : bUsePred=0, bDelta=0, precHigh=0, puis le SELECTEUR
// d index — a 0 l index suit, a 1 il n y en a pas et le lecteur prend la table DEFAUT.
func fluxAbsoluAvecIndex(idx int, idxW uint) []byte {
	bits := "000" // bUsePred, bDelta, precHigh
	if idx < 0 {
		bits += "1" // selecteur pose : aucun index ne suit
	} else {
		bits += "0"
		for b := int(idxW) - 1; b >= 0; b-- {
			bits += string(rune('0' + byte((idx>>uint(b))&1)))
		}
	}
	return bitsDe(bits, 32)
}

// positionEmise rejoue le deserialiseur d i0 sur `buf` sous le profil de `e` et dit si une
// position a ete emise, avec le nombre de bits consommes.
func positionEmise(t *testing.T, e profile.MapQuantEntry, buf []byte) (bool, int) {
	t.Helper()
	defer poserProfilDInstrument(profilDeCarte(e.Layout()))()
	var vu bool
	prec := observateur.PosCaptureHook
	observateur.PosCaptureHook = func(PositionSample) { vu = true }
	defer func() { observateur.PosCaptureHook = prec }()
	br := lecteurDInstrument(buf)
	consumeObjectPositionDynamicPrecisionD(br, br.traversal())
	return vu, br.BitPos()
}

// TestCheminAbsoluSuitLIndexDePlage — LA REGLE, DANS LES DEUX SENS.
//
// Mutation qui doit le faire rougir : remettre `if idx != 0 { return }` dans
// `consumeAbsolutePayload` (le cas `plage jouee` cesse d emettre, le cas `plage 0` se met a
// emettre) ; ou rendre la largeur d index au descripteur de l appelant (les comptes de bits
// tombent de deux).
func TestCheminAbsoluSuitLIndexDePlage(t *testing.T) {
	defer AbsIndexHistogram() // le compteur d observation ne fuit pas sur les autres tests
	e := carteALaPlageUn()
	// Derivation des comptes, avec un index de plage sur 2 bits et les axes 12/12/11 :
	//   avec index : 3 (bUsePred+bDelta+precHigh) + 1 (selecteur) + 2 (index) + 12+12+11 + 2 = 43
	//   sans index : 3 + 1 (selecteur pose) + 3x22 (table DEFAUT du build) + 2 = 72
	cas := []struct {
		nom     string
		idx     int
		emise   bool
		attendu int
	}{
		{"plage JOUEE (celle du catalogue)", 1, true, 43},
		{"plage 0, dont le catalogue ne porte pas les bornes", 0, false, 43},
		{"plage 2, idem", 2, false, 43},
		{"aucun index : la boite monde du build", -1, false, 72},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			emise, bits := positionEmise(t, e, fluxAbsoluAvecIndex(c.idx, 2))
			if emise != c.emise {
				t.Errorf("position emise = %t, attendu %t — la regle d emission ne suit pas "+
					"l index de plage catalogue (`Region` = %d)", emise, c.emise, e.Region)
			}
			if bits != c.attendu {
				t.Errorf("i0 consomme %d bits, derivation %d — la largeur d index ou les "+
					"largeurs d axe ne viennent pas de la carte", bits, c.attendu)
			}
		})
	}
}

// TestCheminAbsoluSurUneCarteSansRegion : le cas des 78 cartes sur 79, ou la plage jouee est la
// 0 et l index tient sur 1 bit. Il garde le comportement HISTORIQUE — sans lui, le correctif
// pourrait n avoir ete qu un deplacement du defaut d une carte vers les autres.
func TestCheminAbsoluSurUneCarteSansRegion(t *testing.T) {
	defer AbsIndexHistogram()
	e := profile.MapQuantEntry{Module: "ridgeline", AxisWidths: [3]uint{13, 13, 14}}
	// 3 + 1 (selecteur) + 1 (index sur 1 bit) + 13+13+14 + 2 = 47 : le compte de la capture CE.
	if emise, bits := positionEmise(t, e, fluxAbsoluAvecIndex(0, 1)); !emise || bits != 47 {
		t.Errorf("plage 0 d une carte sans region : emise=%t bits=%d, attendu emise=true bits=47",
			emise, bits)
	}
	if emise, bits := positionEmise(t, e, fluxAbsoluAvecIndex(1, 1)); emise || bits != 47 {
		t.Errorf("plage 1 d une carte qui n en declare qu une : emise=%t bits=%d, attendu "+
			"emise=false bits=47", emise, bits)
	}
}
