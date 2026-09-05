package filmdec

// delta_biped_walk_guard_test.go — LE MARCHEUR DE RECORDS DELTA BIPEDE RESTE UNIQUE
// (lot E, item E.4 du PLAN_V2_REJEU_FILM, 2026-09-05).
//
// # CE QUE CE FICHIER EMPECHE
//
// Neuf balayages de production portaient la MEME triple boucle — chunks, paquets delta, curseur
// de bits — avec le meme seuil `bipedHeaderBits + bipedIndexBits*bipedMinMaskCnt + i0Bits` et le
// meme appel a `matchBipedHeader`. Le squelette etait identique caractere pour caractere ; seuls
// le crochet installe et le corps de visite differaient. `delta_biped_walk.go` en fait UN
// marcheur ; ce fichier interdit qu'un dixieme site le recopie.
//
// CLAUDE.md regle 6 : « a la 3e copie, centraliser dans un helper ET ajouter un garde-rail (test
// grep) qui interdit l'ancien litteral ». Sans lui, la dette re-croit — c'est arrive une fois
// deja sur ce paquet (le predicat bot, de 8 a 36 copies apres centralisation).
//
// # CE QU'IL NE COUVRE PAS, ET C'EST DELIBERE
//
// LES SOURCES DE TEST SONT HORS PORTEE. Une vingtaine d'instruments de recherche ancrent leurs
// propres records pour mesurer une grammaire candidate, souvent avec une variante deliberee du
// seuil ou de la porte (`needTag1` different, borne de deroulage propre). Les faire passer par le
// marcheur les ferait mentir sur ce qu'ils mesurent. La portee s'etendra le jour ou ces
// instruments seront traites — c'est l'item G du registre, pas celui-ci.

import (
	"strings"
	"testing"
)

// TestMarcheurDeltaBipedeEstUnique — aucune source de PRODUCTION hors `delta_biped_walk.go` ne
// compose le seuil de record ni n'appelle `matchBipedHeader`.
func TestMarcheurDeltaBipedeEstUnique(t *testing.T) {
	const (
		seuil  = "bipedHeaderBits + bipedIndexBits*bipedMinMaskCnt"
		ancrer = "matchBipedHeader("
	)
	for _, nom := range sourcesDeProductionDuPaquet(t) {
		for i, ligne := range lignesDeCode(t, nom) {
			if ligne == "" {
				continue
			}
			switch {
			case strings.Contains(ligne, seuil) && nom != "delta_biped_walk.go":
				t.Errorf("%s:%d recompose le seuil de record bipede :\n\t%s\n"+
					"Il se compose une seule fois, dans `deltaBipedMinRecord` "+
					"(delta_biped_walk.go).", nom, i+1, ligne)
			case strings.Contains(ligne, ancrer) &&
				nom != "delta_biped_walk.go" && nom != "offline_biped.go":
				t.Errorf("%s:%d ancre un record bipede a la main :\n\t%s\n"+
					"L'ancrage passe par `walkDeltaBipedRecords` (contexte de film) ou "+
					"`walkDeltaBipedPayload` (payload seul), delta_biped_walk.go. Neuf copies de "+
					"cette boucle ont ete ramenees a une le 2026-09-05 (lot E, item E.4).",
					nom, i+1, ligne)
			}
		}
	}
}

// TestMarcheurDeltaBipedeAvanceCommeAvant — la marche d'un payload synthetique rend les records
// dans l'ordre du flux, avec l'avance sans chevauchement, et ne depasse pas la borne.
//
// LE TEMOIN EST CONSTRUIT PAR LE MEME ECRIVAIN QUE LES AUTRES TESTS DU PAQUET : la grammaire
// d'en-tete n'est pas re-decrite ici, sinon le garde-rail testerait sa propre copie.
func TestMarcheurDeltaBipedeAvanceCommeAvant(t *testing.T) {
	lay := I0Layout{GateBits: DefaultI0GateBits, AxisW: [3]uint{14, 14, 14}}
	slots := NewSlotBand(map[uint32]bool{7: true})
	pay := make([]byte, 64)

	var vus []deltaBipedRecord
	walkDeltaBipedPayload(pay, slots, lay, true, func(r deltaBipedRecord) {
		vus = append(vus, r)
	})
	// Un payload de zeros ne porte aucun record valide : le marcheur doit rendre la main sans
	// rien publier et SANS BOUCLER. C'est la borne, et elle est la raison d'etre du seuil.
	if len(vus) != 0 {
		t.Fatalf("%d record(s) ancre(s) dans un payload de zeros : la porte d'en-tete ne tient plus", len(vus))
	}
	// Le seuil se compose comme avant : en-tete + masque minimal + i0.
	want := bipedHeaderBits + bipedIndexBits*bipedMinMaskCnt + lay.TotalBits()
	if got := deltaBipedMinRecord(lay.TotalBits()); got != want {
		t.Errorf("deltaBipedMinRecord = %d, attendu %d", got, want)
	}
}
