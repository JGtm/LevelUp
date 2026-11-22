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

func TestReaderFor_SuitLaCapaciteFilmCatchUp(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	Publier("halo_infinite", Frags, 4)

	if r := ReaderFor(false); r != nil {
		t.Errorf("sans la capacité, aucun lecteur n'est branché : %T", r)
	}
	r := ReaderFor(true)
	if r == nil {
		t.Fatal("avec la capacité, le lecteur de l'arriéré publié est branché")
	}
	if got := r.FilmBacklog("halo_infinite"); got.Frags != 4 || !got.Pending() {
		t.Errorf("arriéré lu = %+v, attendu 4 frags", got)
	}
}

func TestAjouterEcrits_CumulParTitreSansToucherLArriere(t *testing.T) {
	Reset()
	Publier("halo_infinite", Frags, 10)
	AjouterEcrits("halo_infinite", 3)
	AjouterEcrits("halo_infinite", 0)
	AjouterEcrits("halo_infinite", -4)
	AjouterEcrits("halo_infinite", 2)
	// Une publication d'arriéré ne remet pas le cumul à zéro.
	Publier("halo_infinite", Rejeux, 1)

	got := Lire("halo_infinite")
	if got.Ecrits != 5 {
		t.Errorf("Ecrits = %d, attendu 5 (3 + 2, le zéro et le négatif sont sans effet)", got.Ecrits)
	}
	if got.Frags != 10 || got.Rejeux != 1 || got.Total() != 11 {
		t.Errorf("arriéré = %+v : le cumul ne doit pas modifier l'arriéré", got)
	}
	if Lire("halo_5").Ecrits != 0 {
		t.Error("le cumul est par titre")
	}
}
