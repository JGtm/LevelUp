package grammar

// keyframe_world_memoire_test.go — LA MARCHE MEMORISEE D UN FILM EST LA MARCHE FRAICHE
// (2026-09-28, [memoireDesMarches]).
//
// Trois proprietes, sur deux bobines par build et la bobine contigue de `killsource` :
//
//  1. la marche servie par la memoire (deuxieme appel) egale la marche faite sans memoire, a
//     l octet (records, ecartes, decisions) ;
//  2. ce qu un appelant fait de sa marche ne touche pas la memoire (copie a chaque lecture) ;
//  3. la memoire ne traverse pas deux contextes, et un tampon qui n est pas le payload du film
//     (meme octets, autre adresse ; payload tronque) n est pas servi par elle.
//
// MUTATIONS JOUEES LE 2026-09-28, ROUGES : rendre la marche memorisee sans copie (propriete 2) ;
// une cle sans la longueur (propriete 3, payload tronque).

import (
	"path/filepath"
	"reflect"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

func TestMarcheMemoriseeEgaleLaMarcheFraiche(t *testing.T) {
	dirs := []string{filepath.Join("..", "facts", "killsource", "testdata", "minibobine_000d5950")}
	for _, court := range []string{"a521164d", "fb1a1a72"} { // le plus ancien et le plus recent des builds
		dirs = append(dirs, filepath.Join("..", "..", "replay", "testdata", "minifilm_"+court))
	}
	payloads := 0
	for _, dir := range dirs {
		film, err := source.LoadDir(dir, nil)
		if err != nil {
			t.Fatalf("%s : %v", dir, err)
		}
		fc := NewFilmContext(film)
		m := fc.MarcheDImageCle()
		fraiche := MarcheDImageCle{preuve: m.preuve}
		if m.memoire == nil || fc.MarcheDImageCle().memoire != m.memoire {
			t.Fatalf("%s : le contexte ne garde pas UNE memoire de ses marches", dir)
		}
		if NewFilmContext(film).MarcheDImageCle().memoire == m.memoire {
			t.Fatalf("%s : deux contextes partagent la memoire des marches", dir)
		}
		for _, pay := range e191cPayloads(fc) {
			payloads++
			premiere := m.Marcher(pay)
			attendu := fraiche.Marcher(pay)
			if !reflect.DeepEqual(premiere, attendu) {
				t.Fatalf("%s : premiere marche %+v, marche fraiche %+v", dir, premiere.Stats, attendu.Stats)
			}
			if len(premiere.Records) > 0 {
				premiere.Records[0].Slot = -12345 // un appelant qui ecrit dans sa marche
			}
			if seconde := m.Marcher(pay); !reflect.DeepEqual(seconde, attendu) {
				t.Fatalf("%s : la marche servie par la memoire differe de la marche fraiche", dir)
			}
			copieDesOctets := append([]byte(nil), pay...)
			if !reflect.DeepEqual(m.Marcher(copieDesOctets), attendu) {
				t.Fatalf("%s : les memes octets a une autre adresse ne marchent pas pareil", dir)
			}
			if n := len(pay) / 2; n > 0 {
				if got, want := m.Marcher(pay[:n]), fraiche.Marcher(pay[:n]); !reflect.DeepEqual(got, want) {
					t.Fatalf("%s : le payload tronque a %d octets est servi par la memoire du payload entier", dir, n)
				}
			}
		}
	}
	if payloads < 20 {
		t.Fatalf("%d payloads seulement", payloads)
	}
}
