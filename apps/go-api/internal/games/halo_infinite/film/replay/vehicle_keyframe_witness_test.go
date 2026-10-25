package replay

import (
	"context"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// TestImageCleVideeDUnCoupNEstPasUnTemoinDAbsence : deux images-cles consecutives sans aucun
// vehicule apres une image-cle qui en recensait deux, puis une image-cle qui les recense a nouveau.
// La vie 778 ne finit pas a la premiere image-cle videe : sa fenetre va jusqu a l image-cle temoin
// suivante, et la mort ecrite apres le trou lui est appariee.
func TestImageCleVideeDUnCoupNEstPasUnTemoinDAbsence(t *testing.T) {
	const s = 20_000_000
	kf := grammar.WorldObjectKeyframes{
		TimesUS: []uint64{0, s, 2 * s, 3 * s, 4 * s},
		SeenUS: map[types.LifeKey][]uint64{
			{Slot: 778, Gen: 1}: {0, s, 4 * s},
			{Slot: 780, Gen: 1}: {0, s, 4 * s},
		},
	}
	deaths := []types.ObjectDeath{{TimestampUS: 2*s + 5_000_000, Slot: 778, Gen: 1}}
	lives, tally := vehicleLives(context.Background(), kf, deaths)
	if tally.emptied != 2 {
		t.Fatalf("images-cles videes d un coup = %d, attendu 2", tally.emptied)
	}
	if tally.matched != 1 {
		t.Fatalf("la mort ecrite apres le trou n est pas appariee : %+v", tally)
	}
	for _, l := range lives {
		if l.key.Slot == 778 && l.deathUS == 0 {
			t.Fatalf("la vie 778 n a pas recu sa mort : %+v", l)
		}
	}
}

// TestUnVehiculeQuiFinitResteUneFin : le passage d UN vehicule a zero n est pas une marche perdue.
// La premiere image-cle suivante reste la preuve d absence, et rien n est reporte ni compte.
func TestUnVehiculeQuiFinitResteUneFin(t *testing.T) {
	const s = 20_000_000
	kf := grammar.WorldObjectKeyframes{
		TimesUS: []uint64{0, s, 2 * s, 3 * s},
		SeenUS:  map[types.LifeKey][]uint64{{Slot: 778, Gen: 1}: {0, s}},
	}
	lives, tally := vehicleLives(context.Background(), kf, nil)
	if len(lives) != 1 || lives[0].goneByUS != 2*s {
		t.Fatalf("goneByUS = %+v, attendu %d", lives, 2*s)
	}
	if tally.emptied != 0 || tally.deferred != 0 {
		t.Fatalf("compteurs %+v : un vehicule seul qui finit ne se compte pas", tally)
	}
}

// TestSansImageCleTemoinOnGardeLaPremiereSuivante : quand toutes les images-cles posterieures sont
// videes d un coup, la preuve d absence reste la premiere : la regle ne fabrique pas une fin de film.
func TestSansImageCleTemoinOnGardeLaPremiereSuivante(t *testing.T) {
	const s = 20_000_000
	kf := grammar.WorldObjectKeyframes{
		TimesUS: []uint64{0, s, 2 * s, 3 * s},
		SeenUS: map[types.LifeKey][]uint64{
			{Slot: 778, Gen: 1}: {0, s},
			{Slot: 780, Gen: 1}: {0, s},
		},
	}
	lives, tally := vehicleLives(context.Background(), kf, nil)
	for _, l := range lives {
		if l.goneByUS != 2*s {
			t.Fatalf("vie %v : goneByUS=%d, attendu %d", l.key, l.goneByUS, 2*s)
		}
	}
	if tally.deferred != 0 {
		t.Fatalf("vies reportees = %d, attendu 0", tally.deferred)
	}
}
