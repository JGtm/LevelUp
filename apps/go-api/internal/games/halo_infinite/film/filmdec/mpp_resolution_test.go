package filmdec

// mpp_resolution_test.go — LA RESOLUTION DU DECOUPAGE MPP NE CONSULTE PAS LE BUILD (constat 2 de
// la revue de jalon M1, 2026-09-15, lentille D13).
//
// # LE DEFAUT QUE CES TESTS FERMENT
//
// Les deux sites de cuisson qui installent le decoupage du bloc `object-multiplayer-properties`
// resolvaient le PROFIL COMPLET (`BuildProfileFromFilm`), lequel refuse tout build absent de la
// table des SEPT (`personnalisationOctets`). Le registre des replis, lui, declare la condition
// `format_sans_profil_relu` et l ordre `apres_lecture` : la cle est la VERSION DE FORMAT
// (`chunk_00+4`, lot 1.9.1 ter).
//
// Consequence d un patch du jeu qui ne change PAS le format : un film au format 27, dont la
// largeur 9/5 est RELUE chez l ecrivain, tombait sur les largeurs CALIBREES — et
// `formatSansProfil` rendait faux, donc ni compteur ni avertissement. Un repli devant une
// lecture disponible, hors du ratchet des `devant_la_lecture` (D14 b).

import (
	"errors"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/filmsource"
)

// buildInconnuTemoin : le nom de build force dans le `chunk_00` du temoin. Il fait EXACTEMENT la
// longueur de `HI_1_12_0`, pour que la substitution ne deplace aucun octet de la section — le
// film reste lisible par ailleurs, seule la CLE BUILD devient inconnue.
const buildInconnuTemoin = "HI_9_99_9"

// filmAuFormat27SansBuildConnu forge le temoin : le `chunk_00` de `bcb6d393` (format 27, build
// `HI_1_12_0`) dont le nom de build est remplace par un build hors table.
//
// C EST LE SEUL MOYEN D AVOIR CE CAS : aucun film du cache ne le porte aujourd hui (mesure du
// 2026-09-15, `TestMPPResolutionCorpus` : 6 films au build hors table, tous a un format dont la
// largeur est indeterminee ou inconnue). Le cas arrivera au prochain patch du jeu qui garde le
// format 27, et c est precisement celui que la correction tient.
func filmAuFormat27SansBuildConnu(t *testing.T) *filmsource.Film {
	t.Helper()
	dir := filepath.Join("..", "replay", "testdata", "minifilm_bcb6d393")
	_, d0 := readChunk00(t, dir)
	patche := append([]byte(nil), d0...)
	if !remplacerBuild(patche, buildHI1120, buildInconnuTemoin) {
		t.Fatalf("le nom de build %q est introuvable dans le chunk_00 du temoin", buildHI1120)
	}
	f, err := filmsource.Load(filmsource.MemoryChunks{patche}, nil)
	if err != nil {
		t.Fatalf("chargement du temoin : %v", err)
	}
	return f
}

// remplacerBuild substitue un nom de build par un autre DE MEME LONGUEUR, en place. Rend faux si
// l ancien nom est absent.
func remplacerBuild(b []byte, ancien, neuf string) bool {
	if len(ancien) != len(neuf) {
		return false
	}
	for i := 0; i+len(ancien) <= len(b); i++ {
		if string(b[i:i+len(ancien)]) != ancien {
			continue
		}
		copy(b[i:], neuf)
		return true
	}
	return false
}

// TestMPPWidthsForFilmNeConsultePasLeBuild — LE TEMOIN DU CONSTAT 2.
//
// Le film porte le format 27 et un build hors table. La resolution rend les largeurs RELUES ;
// le profil complet, lui, REFUSE. La divergence des deux est la mesure du defaut.
func TestMPPWidthsForFilmNeConsultePasLeBuild(t *testing.T) {
	f := filmAuFormat27SansBuildConnu(t)

	id, err := ReadFilmIdentity(mustRegistryChunk(t, f))
	if err != nil {
		t.Fatalf("identite du temoin illisible : %v", err)
	}
	if id.Build != buildInconnuTemoin {
		t.Fatalf("le temoin porte le build %q, %q attendu", id.Build, buildInconnuTemoin)
	}
	if _, errProfil := BuildProfileFromFilm(f); !errors.Is(errProfil, ErrUnknownBuild) {
		t.Fatalf("le profil complet devrait refuser ce build (%v) — sans ce refus le temoin ne "+
			"mesure rien", errProfil)
	}

	res := MPPWidthsForFilm(f)
	if res.FormatVersion != 27 {
		t.Errorf("version de format = %d, 27 attendue", res.FormatVersion)
	}
	if res.FormatInconnu {
		t.Error("FormatInconnu = true : le format 27 est dans la table, aucun compteur " +
			"`filmdec_unknown_format_*` ne doit monter")
	}
	if !res.Relue() {
		t.Fatal("Relue() = false : la largeur du format 27 est RELUE chez l ecrivain, la " +
			"calibration ne doit pas decider")
	}
	if res.Widths.Lead != 9 || res.Widths.Index != 5 {
		t.Errorf("largeurs = %d/%d, 9/5 attendues", res.Widths.Lead, res.Widths.Index)
	}
}

// TestMPPWidthsForFilmSurEntreeTronquee — la borne de lecture, sur un `chunk_00` trop court.
//
// UN FILM SANS VERSION LISIBLE N EST PAS UN FILM AU FORMAT 0 : il est `FormatInconnu`, donc
// compte sous `filmdec_unknown_format_0`, et sa largeur n est PAS posee.
func TestMPPWidthsForFilmSurEntreeTronquee(t *testing.T) {
	for n := 0; n < 8; n++ {
		f, err := filmsource.Load(filmsource.MemoryChunks{make([]byte, n)}, nil)
		if err != nil {
			t.Fatalf("chunk de %d octets : chargement %v", n, err)
		}
		res := MPPWidthsForFilm(f)
		if !res.FormatInconnu || res.FormatVersion != FilmFormatVersionUnknown || res.Relue() {
			t.Errorf("chunk de %d octets : %+v, (0, inconnu, non relue) attendu", n, res)
		}
	}
}

// TestMPPWidthsForFilmSurLesBobines — les sept bobines, par la porte unique.
//
// Il FIGE la frontiere que la correction deplace : seules les bobines au format 27 sont relues,
// les autres restent au repli calibre. Sans lui, un elargissement de `mppWidthsPourFormat`
// basculerait le parc ancien sans que rien ne le dise.
func TestMPPWidthsForFilmSurLesBobines(t *testing.T) {
	relues := map[string]bool{"bcb6d393": true, "fb1a1a72": true}
	for _, court := range closureMiniFilms() {
		dir := filepath.Join("..", "replay", "testdata", "minifilm_"+court)
		_, d0 := readChunk00(t, dir)
		f, err := filmsource.Load(filmsource.MemoryChunks{d0}, nil)
		if err != nil {
			t.Fatalf("%s : chargement %v", court, err)
		}
		res := MPPWidthsForFilm(f)
		if res.Relue() != relues[court] {
			t.Errorf("%s (format %d) : Relue() = %v, %v attendu", court, res.FormatVersion,
				res.Relue(), relues[court])
		}
		if res.FormatInconnu {
			t.Errorf("%s : format %d absent de la table — les sept bobines y sont toutes",
				court, res.FormatVersion)
		}
	}
}

// mustRegistryChunk rend le chunk de registre d un film, ou echoue.
func mustRegistryChunk(t *testing.T, f *filmsource.Film) []byte {
	t.Helper()
	reg, ok := FilmRegistryChunk(f)
	if !ok {
		t.Fatal("le temoin ne porte pas de chunk de registre")
	}
	return reg
}
