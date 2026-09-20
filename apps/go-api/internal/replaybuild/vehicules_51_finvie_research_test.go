//go:build research

package replaybuild

// vehicules_51_finvie_research_test.go — LOT 5.1.4 : L ATTRIBUTION DE LA FIN DE VIE DES
// VEHICULES, MESUREE AVANT ET APRES.
//
// # LA QUESTION, POSEE PAR LE LOT 3.7 ET NON RESOLUE PAR LUI
//
// `VehicleTrack.TEnd` — la fin datee par le dead-state `ti=40 i11`, lu depuis le 2026-09-05 —
// n est renseigne qu UNE FOIS SUR CENT (1/109 sur `a349fea8`, 1/42 sur `a521164d`, cuisson de
// PRODUCTION, note 3.7 § 5). Le calque publie pourtant ses 109 vies : le balayage marche, c est
// le RATTACHEMENT du dead-state a la vie qui ne se fait pas. Sans lui, aucun cycle de
// reapparition de vehicule n est etablissable — le lot 5.1.5 en depend entierement.
//
// # CE QUE CET INSTRUMENT REND, ET POURQUOI IL NE DECIDE RIEN
//
// Les compteurs que le document publie DEJA (`coverage.vehicles`), qui disent exactement ou la
// chaine perd la mort :
//
//	DeathsRead        les morts `ti=40` que la marche rend. A zero : le film n en ecrit pas, ou
//	                  la marche n a rien lu.
//	DeathsMatched     celles qu une vie recensee reprend ;
//	DeathsUnmatched   celles qu AUCUNE vie ne reprend — la mort est lue, la vie ne l est pas.
//	EndDestroyed      les vies dont la fin est DATEE. C est la cible du lot.
//	EndFilmEnd        les vies qui courent jusqu au bout du film (une LECTURE, pas une ignorance).
//	EndUnknown        les vies qui cessent d etre recensees sans que le film ecrive leur mort.
//
// LE CRITERE DU LOT EST ECRIT AVANT LA MESURE : `EndDestroyed` doit MONTER, `DeathsMatched` doit
// monter ou rester egal, et AUCUNE mort appariee ne doit etre perdue (`DeathsMatched` ne descend
// jamais). Une fin ne s invente pas la ou rien n est lu : `DeathsRead` est le plafond.
//
// UN SEUL FILM PAR INVOCATION. Aucune base ouverte, aucun artefact ecrit.
//
//	VEH51_FILM=a349fea8 VEH51_MAP="Fragmentation Heavies" VEH51_OUT=<tmp>/a349fea8.veh.tsv
//	go test -tags=research ./internal/replaybuild/ -run Vehicules51FinDeVie -v -timeout 60m
//
// `4f77afc1` n a PAS de faits d equivalence versionnes : il se cuit sans faits (VEH51_NOFACTS=1),
// exactement comme le corpus gate le fait — sa carte est Flood Gulch (`config/replay_corpus.toml`).

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/testutil"
)

func TestVehicules51FinDeVie(t *testing.T) {
	court, carte, sortie, sansFaits := veh51Garde(t)
	repoRoot, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine repo : %v", err)
	}
	matchID, cartes, faits := veh51Entrees(t, repoRoot, court, carte, sansFaits)
	b, err := NewBuilder(repoRoot, title.DefaultSlug)
	if err != nil {
		t.Fatalf("preparation du builder : %v", err)
	}
	b.SansFaitsPersistes()
	cacheRoot := title.NewPathResolver(repoRoot).CacheRootDir()
	built, err := b.BuildBytes(matchID, cartes, filmcache.ChunkDir(cacheRoot, court), faits)
	if err != nil {
		t.Fatalf("cuisson de %s : %v", court, err)
	}
	var doc replay.ReplayDocument
	if err := json.Unmarshal(built.Blob, &doc); err != nil {
		t.Fatalf("relecture du document de %s : %v", court, err)
	}
	veh51Publier(t, court, doc, sortie)
}

// veh51Garde lit les variables d environnement. SKIP propre si elles manquent.
func veh51Garde(t *testing.T) (court, carte, sortie string, sansFaits bool) {
	t.Helper()
	court, carte, sortie = os.Getenv("VEH51_FILM"), os.Getenv("VEH51_MAP"), os.Getenv("VEH51_OUT")
	sansFaits = os.Getenv("VEH51_NOFACTS") != ""
	if court == "" || sortie == "" {
		t.Skip("VEH51_FILM et VEH51_OUT requis")
	}
	if strings.ContainsAny(court, ",;") {
		t.Fatal("VEH51_FILM ne prend QU UN film")
	}
	return court, carte, sortie, sansFaits
}

// veh51Entrees rend l identifiant de match, les cartes et les faits — depuis le fichier
// d equivalence quand il existe, sinon depuis le seul film (la carte devient obligatoire).
func veh51Entrees(t *testing.T, repoRoot, court, carte string, sansFaits bool) (
	string, []string, port.MatchFacts,
) {
	t.Helper()
	if sansFaits {
		if carte == "" {
			t.Fatal("VEH51_MAP est obligatoire sans faits d equivalence")
		}
		return court, []string{carte}, port.MatchFacts{}
	}
	chemin := filepath.Join(repoRoot, "apps", "go-api", "internal", "games", "halo_infinite",
		"film", "replay", "testdata", "equivalence", court+".facts.json")
	faits, err := ReadFactsFile(chemin)
	if err != nil {
		t.Fatalf("faits %s : %v — poser VEH51_NOFACTS=1 et VEH51_MAP pour cuire sans faits", chemin, err)
	}
	cartes := faits.MapNames
	if carte != "" {
		cartes = []string{carte}
	}
	return faits.MatchID, cartes, faits.MatchFacts
}

// veh51Publier ecrit la couverture et le detail par vie.
func veh51Publier(t *testing.T, court string, doc replay.ReplayDocument, sortie string) {
	t.Helper()
	c := doc.Coverage.Vehicles
	if c == nil {
		t.Fatalf("FILM %s : AUCUNE couverture vehicules — le calque n a pas tourne", court)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# fin de vie des vehicules — film %s\n", court)
	fmt.Fprintf(&b, "# balaye=%t recensees=%d publiees=%d\n", c.Scanned, c.Lives, c.Published)
	fmt.Fprintf(&b, "# mortsLues=%d appariees=%d nonAppariees=%d queueRompue=%d\n",
		c.DeathsRead, c.DeathsMatched, c.DeathsUnmatched, c.DeathsTailDesync)
	fmt.Fprintf(&b, "# finDatee=%d finFilm=%d finInconnue=%d echantillonsApresFin=%d\n",
		c.EndDestroyed, c.EndFilmEnd, c.EndUnknown, c.SamplesAfterEnd)
	b.WriteString("slot\tgen\tchassis\tt0\tt1\ttEnd\tfin\techantillons\n")
	for _, tr := range doc.Vehicles {
		fin := "-"
		if tr.TEnd != nil {
			fin = fmt.Sprintf("%d", *tr.TEnd)
		}
		fmt.Fprintf(&b, "%d\t%d\t%s\t%d\t%d\t%s\t%s\t%d\n", tr.Slot, tr.Gen, tr.Chassis,
			tr.T0, tr.T1, fin, tr.End, len(tr.Samples))
	}
	if err := os.WriteFile(sortie, []byte(b.String()), 0o600); err != nil {
		t.Fatalf("ecriture de %s : %v", sortie, err)
	}
	t.Logf("FILM %s — recensees=%d publiees=%d", court, c.Lives, c.Published)
	t.Logf("FILM %s — MORTS : lues=%d appariees=%d nonAppariees=%d queueRompue=%d",
		court, c.DeathsRead, c.DeathsMatched, c.DeathsUnmatched, c.DeathsTailDesync)
	t.Logf("FILM %s — FINS : datee=%d film=%d inconnue=%d (echantillons apres fin=%d)",
		court, c.EndDestroyed, c.EndFilmEnd, c.EndUnknown, c.SamplesAfterEnd)
}
