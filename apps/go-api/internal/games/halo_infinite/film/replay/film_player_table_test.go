package replay

// film_player_table_test.go — LA TABLE DU FILM, COTE ORCHESTRATION (lot 1.6.0).
//
// LA LECTURE EST DESCENDUE EN `grammar` AU LOT J4.2 (2026-09-26), avec ses tests T-LUE et T-REFUS
// (`grammar/film_player_table_test.go`). Reste ici ce que cette couche fait d un refus :
//
//	T-CABLE   le compteur expvar `filmdec_unknown_build_<build>` est INCREMENTE quand la table
//	          refuse un build — c'est le cablage que le lot 1.5 a nomme sans le poser.

import (
	"os"

	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/observability"
)

// chunk00DeLaBobine rend les octets DEJA DECOMPRESSES du `chunk_00` d'une bobine par build.
func chunk00DeLaBobine(t *testing.T, b buildMiniFilm) []byte {
	t.Helper()
	film, err := source.LoadDir(b.Dir(), nil)
	if err != nil {
		t.Fatalf("chargement de %s : %v", b.Dir(), err)
	}
	chunk0, ok := grammar.FilmRegistryChunk(film)
	if !ok {
		t.Fatalf("%s : la bobine ne porte pas son chunk_00", b.Short8)
	}
	return chunk0
}

// TestScanFilmPlayerTableCableLeCompteurDeBuildInconnu (T-CABLE) : un build hors table de profil
// rend la cause `build_inconnu` ET incremente le compteur expvar que `grammar` NOMME.
//
// CE TEST EST LA RAISON D'ETRE DE L'ITEM 1.6.0. Le lot 1.5 a nomme
// `grammar.UnknownBuildExpvarPairs` en ecrivant, dans son en-tete, que son cableur viendrait au
// lot 1.6 et que « si ce lot passe sans qu'elle soit cablee, c'est un defaut de 1.6 ». La
// mutation est la SEULE facon de le prouver : aucun des 1 351 films du cache n'a de build hors
// profil (mesure du 2026-09-14).
func TestScanFilmPlayerTableCableLeCompteurDeBuildInconnu(t *testing.T) {
	b := miniFilmBuilds()[len(miniFilmBuilds())-1]
	octets := append([]byte(nil), chunk00DeLaBobine(t, b)...)
	off := offsetDeLaChaineDeBuild(t, octets)
	const inconnu = "HI_9_99_0"
	copy(octets[off:off+len(inconnu)+1], append([]byte(inconnu), 0))
	const compteur = "filmdec_unknown_build_hi_9_99_0"
	avant := observability.LoadCounter(compteur)
	film, err := source.Load(source.MemoryChunks{octets}, nil)
	if err != nil {
		t.Fatalf("chargement du chunk_00 mute : %v", err)
	}
	table, err := grammar.ScanFilmPlayerTable(film)
	got := consignerLaTableDuFilm(table, err, b.Short8)
	if got.Refusal != grammar.FilmTableUnknownBuild {
		t.Fatalf("refus %q, attendu %q", got.Refusal, grammar.FilmTableUnknownBuild)
	}
	if got.Build != inconnu {
		t.Errorf("le refus doit NOMMER le build : %q", got.Build)
	}
	if apres := observability.LoadCounter(compteur); apres != avant+1 {
		t.Fatalf("compteur %q : %d -> %d, attendu +1 — le refus est INVISIBLE en production",
			compteur, avant, apres)
	}
}

// offsetDeLaChaineDeBuild retrouve l'offset de la chaine de build par la lecture de `grammar`,
// pour que la mutation porte sur l'octet que la grammaire designe et non sur une recherche a nous.
func offsetDeLaChaineDeBuild(t *testing.T, chunk0 []byte) int {
	t.Helper()
	return identiteDeLaBobine(t, chunk0).BuildOffset
}

// identiteDeLaBobine rend la section d'identification lue par `grammar`, pour que les mutations
// portent sur les octets que la GRAMMAIRE designe et non sur une recherche a nous.
func identiteDeLaBobine(t *testing.T, chunk0 []byte) profile.FilmIdentity {
	t.Helper()
	ident, err := grammar.ReadFilmIdentity(chunk0)
	if err != nil {
		t.Fatalf("identite du film illisible : %v", err)
	}
	if !strings.HasPrefix(ident.Build, "HI_") {
		t.Fatalf("build lu %q — l'ancre n'est pas celle attendue", ident.Build)
	}
	return ident
}

// TestMain n'existe pas ici : `observability` publie ses compteurs a l'init du paquet, et
// `LoadCounter` rend 0 pour un compteur absent. Cette sentinelle le verifie, pour qu'un
// changement de ce contrat ne rende pas T-CABLE vert par accident.
func TestCompteurAbsentVautZero(t *testing.T) {
	if got := observability.LoadCounter("filmdec_unknown_build_" + t.Name()); got != 0 {
		t.Fatalf("un compteur jamais ecrit vaut %d, attendu 0", got)
	}
	_ = os.Getenv(miniFilmCacheEnv)
}
