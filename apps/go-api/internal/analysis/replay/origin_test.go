package replay

import "testing"

// origin_test.go — L'ORIGINE PUBLIEE : ce qu'elle vaut, et quand elle se TAIT.
//
// Le tableau des mesures reelles (quatre films, deux methodes independantes) vit dans
// origin.go ; il se rejoue par `origin_research_test.go`. Ici on verrouille la REGLE.

func TestResolveOriginMs_LectureNominale(t *testing.T) {
	// Chiffres reels de 000d5950 : premier paquet du film 4 517 903 087 us, premier paquet
	// de position 4 521 507 487 us, temoin du fil des morts 4 517 847 ms.
	got := resolveOriginMs(4_521_507_487, 4_517_903_087, 4_517_847, 90)
	if got == nil {
		t.Fatalf("origine nil alors que la lecture et son temoin concordent")
	}
	if *got != 3604 {
		t.Fatalf("origine = %d ms, attendu 3604", *got)
	}
}

func TestResolveOriginMs_ZeroEstUneValeur(t *testing.T) {
	// Un film dont le premier paquet porte deja une position : origine ZERO, et c'est une
	// mesure. Le pointeur existe pour qu'elle ne se lise pas « pas d'origine ».
	got := resolveOriginMs(1_000_000, 1_000_000, 1_000, 90)
	if got == nil || *got != 0 {
		t.Fatalf("origine = %v, attendu un pointeur sur 0", got)
	}
}

func TestResolveOriginMs_SansHorlogeFilm(t *testing.T) {
	if got := resolveOriginMs(4_521_507_487, 0, 4_517_847, 90); got != nil {
		t.Fatalf("origine = %d publiee sans horloge de film : le client doit retomber sur l'appariement", *got)
	}
}

func TestResolveOriginMs_PositionAvantLeFilm(t *testing.T) {
	// Incoherence de lecture (chunk 1 posterieur au premier paquet de position) : rien n'est
	// publie plutot qu'une origine negative.
	if got := resolveOriginMs(1_000, 2_000, 0, 90); got != nil {
		t.Fatalf("origine = %d publiee alors que la position precede le film", *got)
	}
}

func TestResolveOriginMs_TemoinContradictoire(t *testing.T) {
	// Temoin decale de 5 s : au-dela de la tolerance, on ne sert rien.
	if got := resolveOriginMs(4_521_507_487, 4_517_903_087, 4_512_847, 90); got != nil {
		t.Fatalf("origine = %d publiee malgre un temoin a 5 s : jamais en silence", *got)
	}
}

func TestResolveOriginMs_TemoinPauvreNeContreditPas(t *testing.T) {
	// Meme desaccord, mais 3 morts appariees seulement : le temoin ne dit rien de la
	// lecture, et faire taire l'origine perdrait les films les plus fragiles.
	got := resolveOriginMs(4_521_507_487, 4_517_903_087, 4_512_847, 3)
	if got == nil || *got != 3604 {
		t.Fatalf("origine = %v, attendu 3604 : un temoin pauvre n'est pas une contradiction", got)
	}
}

func TestScanFilmClockOrigin_LitUnHorodatage(t *testing.T) {
	// LA VALEUR N'EST PAS VERROUILLEE ICI, et c'est delibere : la mini-bobine REORDONNE les
	// paquets de son chunk 1 (identites d'abord, cf. minifilm_test.go), son premier
	// horodatage n'est donc pas le zero d'un vrai film. Ce qui se verrouille est la LECTURE.
	got, err := ScanFilmClockOrigin(MiniFilmDir)
	if err != nil {
		t.Fatalf("lecture de l'origine d'horloge : %v", err)
	}
	if got == 0 {
		t.Fatalf("horodatage nul : l'en-tete de paquet n'a pas ete lu")
	}
}

func TestScanFilmClockOrigin_FilmAbsent(t *testing.T) {
	if _, err := ScanFilmClockOrigin("testdata/film_inexistant"); err == nil {
		t.Fatalf("aucune erreur sur un film absent")
	}
}
