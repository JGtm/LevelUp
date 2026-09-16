package replay

// mpp_format_inconnu_test.go — LA BRANCHE DE PRODUCTION SOUS GARDE-RAIL (lot 1.9.1 ter).
//
// CE QUE CE TEST EXISTE POUR EMPECHER, ET IL N Y AVAIT RIEN AVANT LUI. La question posee par le
// pilote : « un patch du jeu ne doit pas eteindre le decodeur sur tout le parc neuf ». La
// reponse du code est un REPLI (les largeurs sont CALIBREES sur le film), pas un refus sec — et
// jusqu ici cette reponse n etait garantie par AUCUN test. `TestBuildProfileRefuseUnFormatInconnu`
// garde l unite (`ErrUnknownFormat`), pas la branche de production.
//
// LA MUTATION QUI ROUGIT est celle qui transformerait le repli en refus : faire rendre
// `MPPWidths{}` a `gwWidthsForFilm` au lieu de `calibrees`. Le test le voit, parce qu il compare
// aux largeurs calibrees PASSEES EN ENTREE et non a « quelque chose de valide ». Retirer le
// compteur le fait rougir aussi, par la seconde assertion.
//
// AUCUN DES 1 351 FILMS DU CACHE N A DE FORMAT HORS TABLE (mesure du 2026-09-15 : 20, 21, 24,
// 25, 27, tous connus). La seule facon de jouer cette branche est donc de PRESENTER une bobine
// sous un format inconnu, et c est ce que fait `bobineAuFormat` — en reecrivant les quatre
// octets que la grammaire designe (`chunk_00+4`), pas une position devinee.

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/filmsource"
	"levelup/go-api/internal/games/halo_infinite/film/grammar"
	"levelup/go-api/internal/observability"
)

// formatHorsTable : une version de format que la table de profil ne connait pas. 28 est le
// PROCHAIN format plausible — le cas exact du pilote, « 28 au prochain patch du jeu ».
const formatHorsTable = 28

// TestFormatInconnuTombeSurLesLargeursCalibrees — LA BRANCHE DE PRODUCTION.
//
// Un film presente sous un format inconnu doit (a) garder les largeurs CALIBREES, donc rester
// decodable, et (b) incrementer `filmdec_unknown_format_28`, donc rester VISIBLE.
func TestFormatInconnuTombeSurLesLargeursCalibrees(t *testing.T) {
	film := bobineAuFormat(t, "fb1a1a72", formatHorsTable)
	fc := grammar.NewFilmContext(film)

	// Un decoupage volontairement DISTINCT du profil relu (9/5) : si la branche prenait le
	// profil au lieu du repli, la comparaison le verrait.
	calibrees := grammar.MPPWidths{Lead: 7, Index: 4}
	const compteur = "filmdec_unknown_format_28"
	avant := observability.LoadCounter(compteur)

	got := gwWidthsForFilm(fc, calibrees)

	if got != calibrees {
		t.Fatalf("largeurs %s, %s attendues — un format inconnu doit tomber sur le REPLI calibre, "+
			"jamais sur un refus sec ni sur le profil d'un format voisin (D-4 d'ADR 0034). "+
			"Un patch du jeu eteindrait le decodeur sur tout le parc neuf.", got, calibrees)
	}
	if apres := observability.LoadCounter(compteur); apres != avant+1 {
		t.Fatalf("compteur %q : %d -> %d, attendu +1 — le repli serait SILENCIEUX, et un "+
			"changement de format du jeu ne se verrait qu'en derive de qualite sans cause",
			compteur, avant, apres)
	}
}

// TestFormatConnuNeCompteRien — L AUTRE MOITIE DU CONTRAT, sans laquelle la premiere ne prouve
// rien : un format CONNU dont la largeur est indeterminee (le parc ancien, formats 20/21/24/25)
// tombe sur le meme repli mais N EST PAS un evenement, et ne doit RIEN compter. Un compteur qui
// s incremente sur tout le parc ancien ne signalerait plus aucun patch.
func TestFormatConnuNeCompteRien(t *testing.T) {
	// La bobine INTACTE : `a521164d` (HI_1_4_1) porte le format 21 — connu de la table, sans
	// largeur relue. C'est le parc ancien tel qu'il est, aucune reecriture.
	film, err := filmsource.LoadDir(filepath.Join("testdata", "minifilm_a521164d"), nil)
	if err != nil {
		t.Fatalf("chargement de la bobine a521164d : %v", err)
	}
	if format, _ := grammar.FilmFormatVersion(film); format != 21 {
		t.Fatalf("a521164d porte le format %d, 21 attendu — la bobine a change", format)
	}
	if format, sansProfil := formatSansProfil(film); sansProfil {
		t.Fatalf("format %d declare hors table alors qu'il y est — le parc ancien serait compte "+
			"comme un patch du jeu", format)
	}
}

// bobineAuFormat recopie une bobine versionnee dans un repertoire temporaire et y REECRIT la
// version de format, aux quatre octets que `grammar` designe (`chunk_00+4`).
//
// Le `chunk_00` est ecrit DECOMPRESSE : `filmsource.Inflate` rend le tampon inchange quand il
// n'est pas zlib, et les 1 351 `chunk_00` du cache sont dans ce cas — la bobine reecrite est
// donc de la meme forme que la production, pas un cas de figure invente pour le test.
func bobineAuFormat(t *testing.T, court string, format int) *filmsource.Film {
	t.Helper()
	src := filepath.Join("testdata", "minifilm_"+court)
	origine, err := filmsource.LoadDir(src, nil)
	if err != nil {
		t.Fatalf("chargement de %s : %v", src, err)
	}
	chunk0, ok := grammar.FilmRegistryChunk(origine)
	if !ok {
		t.Fatalf("%s : la bobine ne porte pas son chunk_00", court)
	}
	if avant, _ := grammar.FilmFormatVersionFromHeader(chunk0); avant == format {
		t.Fatalf("%s porte deja le format %d : la mutation ne prouverait rien", court, format)
	}
	dst := t.TempDir()
	entrees, err := os.ReadDir(src)
	if err != nil {
		t.Fatalf("lecture de %s : %v", src, err)
	}
	for _, e := range entrees {
		if e.IsDir() || filepath.Ext(e.Name()) != ".bin" {
			continue
		}
		octets, err := os.ReadFile(filepath.Join(src, e.Name()))
		if err != nil {
			t.Fatalf("lecture de %s : %v", e.Name(), err)
		}
		if e.Name() == "chunk_00.bin" {
			octets = append([]byte(nil), chunk0...)
			ecrireU32(octets, 4, format)
		}
		if err := os.WriteFile(filepath.Join(dst, e.Name()), octets, 0o600); err != nil {
			t.Fatalf("ecriture de %s : %v", e.Name(), err)
		}
	}
	mute, err := filmsource.LoadDir(dst, nil)
	if err != nil {
		t.Fatalf("rechargement de la bobine reecrite : %v", err)
	}
	relu, ok := grammar.FilmFormatVersion(mute)
	if !ok || relu != format {
		t.Fatalf("format relu %d (ok=%v), %d attendu — la reecriture n'a pas porte", relu, ok, format)
	}
	return mute
}

// ecrireU32 pose un u32 petit-boutiste, la meme convention que le lecteur.
func ecrireU32(d []byte, off, v int) {
	d[off] = byte(v)
	d[off+1] = byte(v >> 8)
	d[off+2] = byte(v >> 16)
	d[off+3] = byte(v >> 24)
}
