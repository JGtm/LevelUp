package filmbacklog

import "testing"

func TestPublierEtLire_ParTitreEtParEtape(t *testing.T) {
	Reset()
	Publier("halo_infinite", Frags, 395)
	Publier("halo_infinite", Rejeux, 49)
	Publier("halo_5", Frags, 2)

	hi := Lire("halo_infinite")
	if hi.Frags != 395 || hi.Rejeux != 49 || hi.Total() != 444 || !hi.Pending() {
		t.Errorf("halo_infinite = %+v, attendu 395 frags et 49 rejeux", hi)
	}
	if h5 := Lire("halo_5"); h5.Frags != 2 || h5.Rejeux != 0 {
		t.Errorf("halo_5 = %+v : chaque étape et chaque titre portent leur propre valeur", h5)
	}

	// Une étape publie sa valeur d'après la passe, pas un cumul ; une valeur négative vaut zéro.
	Publier("halo_infinite", Frags, 0)
	Publier("halo_infinite", Rejeux, -3)
	if got := Lire("halo_infinite"); got.Pending() {
		t.Errorf("halo_infinite = %+v, attendu un arriéré vide", got)
	}
}

func TestLire_TitreInconnuVide(t *testing.T) {
	Reset()
	if Lire("titre-jamais-vu").Pending() {
		t.Error("un titre dont aucune passe n'a tourné n'a aucun arriéré")
	}
	if (Reader{}).FilmBacklog("titre-jamais-vu").Pending() {
		t.Error("le lecteur suit Lire")
	}
}
