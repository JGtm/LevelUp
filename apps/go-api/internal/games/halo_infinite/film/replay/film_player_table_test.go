package replay

// film_player_table_test.go — LE LECTEUR DE PRODUCTION DE LA TABLE DU FILM (lot 1.6.0).
//
// Ce que ces tests prouvent, et rien d'autre :
//
//	T-LUE     les sept bobines par build rendent une table lue, aux comptes de `grammar` ;
//	T-REFUS   chaque cause d'echec rend une cause NOMMEE, jamais une table partielle ni un panic ;
//	T-CABLE   le compteur expvar `filmdec_unknown_build_<build>` est INCREMENTE quand la table
//	          refuse un build — c'est le cablage que le lot 1.5 a nomme sans le poser.

import (
	"os"
	"path/filepath"
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

// TestScanFilmPlayerTableSurLesBobines (T-LUE) : les sept builds rendent une table EMPLOYABLE.
func TestScanFilmPlayerTableSurLesBobines(t *testing.T) {
	for _, b := range miniFilmBuilds() {
		t.Run(b.Build, func(t *testing.T) {
			got := lireTableDeChunk0(chunk00DeLaBobine(t, b), b.Short8)
			if !got.Lue() {
				t.Fatalf("table NON employable : refus=%q sieges=%d intercale=%v",
					got.Refusal, got.Occupied, got.InterleavedVacant)
			}
			if got.Build != b.Build {
				t.Errorf("build lu %q, attendu %q", got.Build, b.Build)
			}
			if got.Occupied+got.Vacant != 32 {
				t.Errorf("%d occupe(s) + %d vacant(s) != 32 — la borne de l'ecrivain est 32",
					got.Occupied, got.Vacant)
			}
			if len(got.Seats) != got.Occupied {
				t.Errorf("%d siege(s) rendu(s) pour %d occupe(s)", len(got.Seats), got.Occupied)
			}
			for _, s := range got.Seats {
				if s.XUID == 0 {
					t.Errorf("siege de rang %d sans XUID — un siege occupe en porte un", s.FilmIndex)
				}
				if s.Gamertag == "" {
					t.Errorf("siege de rang %d sans gamertag — le film l'ecrit", s.FilmIndex)
				}
			}
		})
	}
}

// TestScanFilmPlayerTableSansRegistre (T-REFUS) : la bobine historique n'a PAS de `chunk_00`, et
// c'est le cas le plus simple d'un film qui ne porte pas sa table. Cause NOMMEE, aucun panic.
func TestScanFilmPlayerTableSansRegistre(t *testing.T) {
	film, err := source.LoadDir(filepath.Join(goldenDir, "minifilm_"+goldenFilm), nil)
	if err != nil {
		t.Fatalf("chargement de la bobine historique : %v", err)
	}
	got := ScanFilmPlayerTable(film, goldenFilm)
	if got.Refusal != FilmTableNoRegistry {
		t.Fatalf("refus %q, attendu %q", got.Refusal, FilmTableNoRegistry)
	}
	if got.Lue() || len(got.Seats) != 0 {
		t.Fatalf("une table refusee ne rend AUCUN siege : %d", len(got.Seats))
	}
}

// TestScanFilmPlayerTableCausesNommees (T-REFUS) : chaque coupe d'un `chunk_00` sain rend une
// cause nommee, jamais une lecture partielle en silence.
func TestScanFilmPlayerTableCausesNommees(t *testing.T) {
	b := miniFilmBuilds()[len(miniFilmBuilds())-1]
	sain := chunk00DeLaBobine(t, b)
	corps := identiteDeLaBobine(t, sain).BodyBit / 8
	cas := []struct {
		nom      string
		octets   []byte
		attendue FilmTableRefusal
	}{
		// LES TROIS CAUSES SONT MESUREES, PAS SUPPOSEES (2026-09-14). Un tampon vide n'a pas de
		// registre : troncature. Une coupe a la MOITIE laisse le registre et la section lisibles
		// (ils tiennent dans le premier tiers) et ampute le corps d'assez pour qu'aucun depart ne
		// ferme 32 slots : table INTROUVABLE. Une coupe juste APRES le debut du corps tombe avant
		// la section elle-meme sur ce build : troncature. Les deux dernieres ne se devinent pas —
		// elles dependent de la place du corps dans le tampon, et c'est pourquoi ce tableau porte
		// la valeur MESUREE et non celle qu'on aurait ecrite d'avance.
		// COUPER LA QUEUE NE REFUSE RIEN : les slots vacants y sont des zeros, et la lecture va
		// jusqu'au bout du tampon. Aucune de ces coupes ne porte donc sur la queue.
		{"tampon vide", nil, FilmTableTruncated},
		{"corps ampute de moitie", append([]byte(nil), sain[:len(sain)/2]...), FilmTableNotFound},
		{"coupe au debut du corps", append([]byte(nil), sain[:corps+1000]...), FilmTableTruncated},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			got := lireTableDeChunk0(c.octets, "coupe")
			if got.Refusal == FilmTableRead {
				t.Fatalf("une entree coupee a rendu une table LUE (%d sieges)", len(got.Seats))
			}
			if got.Refusal != c.attendue {
				t.Errorf("refus %q, attendu %q", got.Refusal, c.attendue)
			}
			if len(got.Seats) != 0 {
				t.Errorf("%d siege(s) rendus par une lecture refusee", len(got.Seats))
			}
		})
	}
}

// TestScanFilmPlayerTableSansSection (T-REFUS) : effacer les trois champs de chaine d'un film
// sain reproduit les 5 films du cache sans section d'identification.
func TestScanFilmPlayerTableSansSection(t *testing.T) {
	b := miniFilmBuilds()[len(miniFilmBuilds())-1]
	octets := append([]byte(nil), chunk00DeLaBobine(t, b)...)
	off := offsetDeLaChaineDeBuild(t, octets)
	for i := off - 0x20; i < off+0x40; i++ {
		octets[i] = 0
	}
	got := lireTableDeChunk0(octets, b.Short8)
	if got.Refusal != FilmTableNoSection {
		t.Fatalf("refus %q, attendu %q", got.Refusal, FilmTableNoSection)
	}
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
	got := lireTableDeChunk0(octets, b.Short8)
	if got.Refusal != FilmTableUnknownBuild {
		t.Fatalf("refus %q, attendu %q", got.Refusal, FilmTableUnknownBuild)
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
