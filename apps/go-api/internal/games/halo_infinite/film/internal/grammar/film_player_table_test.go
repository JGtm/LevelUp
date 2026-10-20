package grammar

// film_player_table_test.go — LA LECTURE DE LA TABLE DU FILM (lot 1.6.0).
//
// DESCENDU DE `film/replay` AU LOT J4.2 (2026-09-26) avec `ScanFilmPlayerTable` : T-LUE et
// T-REFUS portent sur la LECTURE. T-CABLE (le compteur expvar du build inconnu, pose par
// l orchestrateur) reste en `replay`, avec le journal et le compteur qu il verifie.
//
// Ce que ces tests prouvent, et rien d'autre :
//
//	T-LUE     les sept bobines par build rendent une table lue, aux comptes de `grammar` ;
//	T-REFUS   chaque cause d'echec rend une cause NOMMEE, jamais une table partielle ni un panic.

import (
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// TestScanFilmPlayerTableSurLesBobines (T-LUE) : les sept builds rendent une table EMPLOYABLE.
func TestScanFilmPlayerTableSurLesBobines(t *testing.T) {
	for _, b := range bobinesIdentite() {
		t.Run(b.build, func(t *testing.T) {
			got, _ := lireTableDeChunk0(bobineChunk00(t, b.film))
			if !got.Lue() {
				t.Fatalf("table NON employable : refus=%q sieges=%d intercale=%v",
					got.Refusal, got.Occupied, got.InterleavedVacant)
			}
			if got.Build != b.build {
				t.Errorf("build lu %q, attendu %q", got.Build, b.build)
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
	film, err := source.LoadDir(miniBobineChunks, nil)
	if err != nil {
		t.Fatalf("chargement de la bobine historique : %v", err)
	}
	got, err := ScanFilmPlayerTable(film)
	if err != nil {
		t.Fatalf("un film sans registre est un refus NOMME, pas une erreur de lecture : %v", err)
	}
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
	b := bobinesIdentite()[len(bobinesIdentite())-1]
	sain := bobineChunk00(t, b.film)
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
			got, _ := lireTableDeChunk0(c.octets)
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
	b := bobinesIdentite()[len(bobinesIdentite())-1]
	octets := append([]byte(nil), bobineChunk00(t, b.film)...)
	off := offsetDeLaChaineDeBuild(t, octets)
	for i := off - 0x20; i < off+0x40; i++ {
		octets[i] = 0
	}
	got, _ := lireTableDeChunk0(octets)
	if got.Refusal != FilmTableNoSection {
		t.Fatalf("refus %q, attendu %q", got.Refusal, FilmTableNoSection)
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
	ident, err := ReadFilmIdentity(chunk0)
	if err != nil {
		t.Fatalf("identite du film illisible : %v", err)
	}
	if !strings.HasPrefix(ident.Build, "HI_") {
		t.Fatalf("build lu %q — l'ancre n'est pas celle attendue", ident.Build)
	}
	return ident
}
