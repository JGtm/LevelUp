//go:build research

package replaybuild

// navpoint_51_drapeau_research_test.go — LOT 5.1.2 : LES INTERVALLES DU DRAPEAU, CUITS PAR LA
// CHAINE DE PRODUCTION.
//
// # CE QUE CET INSTRUMENT A SERVI A ETABLIR (2026-09-18)
//
// Il est l ORACLE de la preuve du lot 5.1.2, et cette preuve a ECHOUE : le minuteur manuel du
// point de navigation est NUL pendant les lachers qu il publie ici. Le detail de la mesure et
// son oracle de position vivent du cote du balayage
// (`grammar/navpoint_51_minuteur_research_test.go`). Ce que CE fichier a fourni, sans quoi le
// negatif n aurait pas ete opposable :
//
//	film        drapeaux   intervalles   dont `dropped`   duree cumulee des lachers
//	bcb6d393       2            35            13                56,8 s
//	fb1a1a72       2            83            33               299,6 s
//
// Sans ce calque, « i12 est nul » ne dirait rien : il faut savoir QUE le film porte des lachers,
// et QUAND, pour que leur silence soit une mesure et non une absence de donnee.
//
// # POURQUOI CUIRE, ET POURQUOI ICI
//
// La preuve du lot 5.1.2 confronte le minuteur manuel du point de navigation (`ti=12 i11`/`i12`,
// porte au lot 5.1.1) aux intervalles `dropped` que `flagCarries` publie deja. Le calque du
// drapeau n est PAS produit par un balayage isole : il exige le catalogue versionne d objectifs
// de carte (les socles `flag_spawn`, joints par `map_id`), les evenements nommes du statborg, le
// fil des morts et le pont d identite. La decouverte D8 du lot 3.7 le dit sur pieces —
// `BuildFromFilm` avec le seul `Options.MapQuant` publie ZERO intervalle de drapeau, meme sur un
// CTF du corpus temoin. L instrument passe donc par `Builder.BuildBytes`, la chaine de
// PRODUCTION, avec les faits d equivalence deja versionnes (`replay/testdata/equivalence/`).
//
// AUCUNE BASE N EST OUVERTE : les faits viennent du fichier, les catalogues du disque, le film
// du cache. AUCUN ARTEFACT N EST ECRIT : `BuildBytes` rend les octets, il ne les range pas.
//
// # LE TEMPS EST RAMENE SUR L HORLOGE DU FILM
//
// Les intervalles du calque sont en FRAMES de l axe du rejeu ; le balayage de `ti=12`, lui, date
// ses lectures sur l horloge du MANIFESTE. La conversion est celle que le document publie
// lui-meme : `ms_film = t * frameIntervalMs + originMs`. Sans elle les deux tableaux ne se
// superposeraient pas, et une preuve qui compare deux axes differents ne prouve rien.
//
// UN SEUL FILM PAR INVOCATION (NAV51D_FILM).
//
//	NAV51D_FILM=bcb6d393 NAV51D_MAP=Cliffhanger NAV51D_OUT=<tmp>/bcb6d393.flagspans.tsv
//	go test -tags=research ./internal/replaybuild/ -run Navpoint51DrapeauSurFilm -v -timeout 60m

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/testutil"
)

func TestNavpoint51DrapeauSurFilm(t *testing.T) {
	court, carte, sortie := nav51dGarde(t)
	repoRoot, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine repo : %v", err)
	}
	factsPath := filepath.Join(repoRoot, "apps", "go-api", "internal", "games", "halo_infinite",
		"film", "replay", "testdata", "equivalence", court+".facts.json")
	faits, err := ReadFactsFile(factsPath)
	if err != nil {
		t.Fatalf("faits %s : %v", factsPath, err)
	}
	b, err := NewBuilder(repoRoot, title.DefaultSlug)
	if err != nil {
		t.Fatalf("preparation du builder : %v", err)
	}
	b.SansFaitsPersistes()
	cartes := faits.MapNames
	if carte != "" {
		cartes = []string{carte}
	}
	cacheRoot := title.NewPathResolver(repoRoot).CacheRootDir()
	built, err := b.BuildBytes(faits.MatchID, cartes,
		filmcache.ChunkDir(cacheRoot, court), faits.MatchFacts)
	if err != nil {
		t.Fatalf("cuisson de %s : %v", court, err)
	}
	var doc replay.ReplayDocument
	if err := json.Unmarshal(built.Blob, &doc); err != nil {
		t.Fatalf("relecture du document de %s : %v", court, err)
	}
	nav51dPublier(t, court, doc, sortie)
}

// nav51dGarde lit les variables d environnement. SKIP propre si le film manque.
func nav51dGarde(t *testing.T) (court, carte, sortie string) {
	t.Helper()
	court, carte, sortie = os.Getenv("NAV51D_FILM"), os.Getenv("NAV51D_MAP"), os.Getenv("NAV51D_OUT")
	if court == "" || sortie == "" {
		t.Skip("NAV51D_FILM et NAV51D_OUT requis")
	}
	if strings.ContainsAny(court, ",;") {
		t.Fatal("NAV51D_FILM ne prend QU UN film")
	}
	return court, carte, sortie
}

// nav51dPublier ecrit les intervalles sur l horloge du FILM et resume la couverture.
func nav51dPublier(t *testing.T, court string, doc replay.ReplayDocument, sortie string) {
	t.Helper()
	pas, origine := doc.FrameIntervalMS, int64(0)
	if pas <= 0 {
		pas = replay.DefaultFrameIntervalMS
	}
	if doc.OriginMs != nil {
		origine = *doc.OriginMs
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# intervalles du drapeau — film %s ; pas=%d ms ; origine=%d ms\n", court, pas, origine)
	b.WriteString("drapeau\tequipe\tetat\tt0_frame\tt1_frame\tt0_film_ms\tt1_film_ms\txuid\n")
	type ligne struct {
		drapeau, equipe, t0, t1 int
		etat, xuid              string
	}
	var lignes []ligne
	for i, fc := range doc.FlagCarries {
		for _, s := range fc.Spans {
			x := ""
			if s.XUID != nil {
				x = *s.XUID
			}
			lignes = append(lignes, ligne{drapeau: i, equipe: fc.Team, t0: s.T0, t1: s.T1, etat: s.State, xuid: x})
		}
	}
	sort.SliceStable(lignes, func(i, j int) bool { return lignes[i].t0 < lignes[j].t0 })
	for _, l := range lignes {
		fmt.Fprintf(&b, "%d\t%d\t%s\t%d\t%d\t%d\t%d\t%s\n", l.drapeau, l.equipe, l.etat, l.t0, l.t1,
			int64(l.t0*pas)+origine, int64(l.t1*pas)+origine, l.xuid)
	}
	if err := os.WriteFile(sortie, []byte(b.String()), 0o600); err != nil {
		t.Fatalf("ecriture de %s : %v", sortie, err)
	}
	t.Logf("FILM %s — %d drapeau(x), %d intervalle(s) ecrits dans %s", court, len(doc.FlagCarries), len(lignes), sortie)
	for _, l := range lignes {
		if l.etat == replay.FlagStateDropped {
			t.Logf("FILM %s — LACHER drapeau %d (equipe %d) : frames %d..%d = film %d..%d ms",
				court, l.drapeau, l.equipe, l.t0, l.t1, int64(l.t0*pas)+origine, int64(l.t1*pas)+origine)
		}
	}
	if c := doc.Coverage.FlagCarries; c != nil {
		t.Logf("FILM %s — couverture : filmDrapeau=%t prises=%d socles=%d sansPont=%d fermes=%d ouverts=%d",
			court, c.FlagFilm, c.Openings, c.Spawns, c.NoBridge, c.Closed, c.Open)
	} else {
		t.Logf("FILM %s — AUCUNE couverture flagCarries : l appelant n a rien fourni a lire", court)
	}
}
