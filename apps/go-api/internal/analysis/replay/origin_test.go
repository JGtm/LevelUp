package replay

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

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

// TestCoverageSaysWhetherOriginIsResolved — L'ARTEFACT DOIT DIRE QUE SON AXE DE TEMPS EST
// DOUTEUX (correctif de revue R1).
//
// Quand l'origine n'est pas etablie, les calques dates depuis l'horloge du film (actions
// d'objectif, courbe de score) sont poses avec une soustraction de zero, donc decales de 3,6 s a
// 50,8 s selon le match. Rien, dans l'artefact, ne le signalait : le rendu ne pouvait pas les
// masquer, il les dessinait au mauvais instant.
func TestCoverageSaysWhetherOriginIsResolved(t *testing.T) {
	for _, cas := range []struct {
		nom      string
		clockUS  uint64
		veutVrai bool
	}{
		{"origine etablie", 1_000_000, true},
		{"origine absente", 0, false},
	} {
		t.Run(cas.nom, func(t *testing.T) {
			doc := BuildFromPositions("m", "halo_infinite", positionsPourOrigine(), nil,
				Options{FilmClockOriginUS: cas.clockUS})
			if doc.Coverage == nil {
				t.Fatal("aucune couverture publiee")
			}
			if doc.Coverage.OriginResolved != cas.veutVrai {
				t.Errorf("coverage.originResolved = %v, attendu %v (originMs = %v)",
					doc.Coverage.OriginResolved, cas.veutVrai, doc.OriginMs)
			}
			if (doc.OriginMs != nil) != doc.Coverage.OriginResolved {
				t.Errorf("le drapeau ne suit pas la publication de l'origine : originMs=%v drapeau=%v",
					doc.OriginMs, doc.Coverage.OriginResolved)
			}
		})
	}
}

// positionsPourOrigine rend deux positions d'un meme slot, assez pour qu'une trace soit publiee.
func positionsPourOrigine() []filmdec.BipedPosition {
	return []filmdec.BipedPosition{
		{Slot: 1, TimestampUS: 2_000_000, X: 1, Y: 1, Z: 1},
		{Slot: 1, TimestampUS: 2_100_000, X: 2, Y: 2, Z: 1},
		{Slot: 1, TimestampUS: 2_200_000, X: 3, Y: 3, Z: 1},
	}
}
