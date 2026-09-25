package replay

// fil_des_morts_faits_test.go — LE REJEU DEPUIS LES FAITS REND LE MEME VERDICT DU FIL DES MORTS
// QUE LE DECODAGE DU FILM (lot M8 des retours rejeu, 2026-09-24).
//
// LA BRECHE QUE CE FICHIER FERME. `coverage.bridge.deathsFeed` (lot M5.2) naissait au balayage et
// ne voyageait pas dans les faits persistes : un document rejoue depuis les faits d un film au fil
// VIDE ou ILLISIBLE publiait la cle ABSENTE la ou le decodage du meme film publiait `empty` ou
// `unreadable`. Les deux chemins de la cuisson ne rendaient donc plus le meme document.
//
// LE TEMOIN ILLISIBLE est le manifeste partiel d `ab526724` (34 entrees typees, aucun morceau des
// temps forts), copie a l octet dans `film/filmcache/testdata/manifeste_partiel_34.json` : c est
// sa FORME (des morceaux types sans temps forts) que ce test rejoue, sur les octets de la bobine
// v40. Le match n est qu un temoin.
//
// LES ENTREES VIENNENT DU FIXTURE VERSIONNE (`loadGoldenInputs`, deja passees par le codec, donc
// stables a l aller-retour) : seul le fil des morts et son verdict sont relus dans le film. La
// comparaison porte sur le DOCUMENT ENTIER serialise, pas sur la seule cle.

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/games/halo_infinite/film/internal/facts"
	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// temoinManifestePartiel rend l index du manifeste partiel d `ab526724`, lu par le lecteur de
// PRODUCTION (`filmcache.Open`) sur une copie du temoin rangee dans un cache temporaire.
func temoinManifestePartiel(t *testing.T) []types.ChunkMeta {
	t.Helper()
	blob, err := os.ReadFile(filepath.Join("..", "filmcache", "testdata", "manifeste_partiel_34.json"))
	if err != nil {
		t.Fatalf("temoin : %v", err)
	}
	racine := t.TempDir()
	if err := os.MkdirAll(filmcache.ManifestsRoot(racine), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filmcache.ManifestPath(racine, "ab526724"), blob, 0o600); err != nil {
		t.Fatal(err)
	}
	src, found, err := filmcache.Open(racine, "ab526724")
	if err != nil || !found || len(src.Meta()) != 34 {
		t.Fatalf("temoin illisible ou inattendu : found=%v err=%v", found, err)
	}
	return src.Meta()
}

// casDuFilDesMorts : les trois issues de la lecture, chacune sur de VRAIS octets.
type casDuFilDesMorts struct {
	nom    string
	chunks [][]byte
	meta   []types.ChunkMeta
	want   string
}

func casDesTroisIssues(t *testing.T) []casDuFilDesMorts {
	t.Helper()
	o := octetsBobineV40(t)
	typeTempsForts := []types.ChunkMeta{
		{Index: 0, ChunkType: 1}, {Index: 1, ChunkType: 2}, {Index: 2, ChunkType: filmcache.ChunkTypeTempsForts},
	}
	partiel := temoinManifestePartiel(t)
	morceaux := make([][]byte, len(partiel))
	for i := range morceaux {
		morceaux[i] = o[1] // de la replication, partout sauf au registre
	}
	morceaux[0] = o[0]
	return []casDuFilDesMorts{
		{"temps forts lus, des morts", o, typeTempsForts, DeathsFeedRead},
		{"temps forts lus, aucune mort", [][]byte{o[0], o[1], make([]byte, 64)}, typeTempsForts, DeathsFeedEmpty},
		{"temoin partiel d ab526724", morceaux, partiel, DeathsFeedUnreadable},
	}
}

// documentsDuFilmEtDesFaits rend le document ASSEMBLE A LA SORTIE DU BALAYAGE et celui REJOUE
// DEPUIS LES FAITS que ce balayage aurait persistes — les deux branches de la cuisson.
func documentsDuFilmEtDesFaits(t *testing.T, cas casDuFilDesMorts) (ReplayDocument, ReplayDocument) {
	t.Helper()
	entry := goldenEntryPourTest(t)
	g := loadGoldenInputs(t)
	film, err := source.Load(source.MemoryChunks(cas.chunks), cas.meta)
	if err != nil {
		t.Fatalf("%s : chargement : %v", cas.nom, err)
	}
	s := &filmScan{matchID: goldenFilm, film: film, in: g.FilmInputs}
	s.lireLeFilDesMorts()
	depuisLeFilm := s.assembler("halo_infinite",
		Options{MapQuant: &entry, Fallbacks: fallback.NouveauCompteur()})

	blob, err := EncodeFilmFactsFile(&FilmFactsFile{
		Coverage: *couvertureDuDecodeur(nil),
		Facts: FilmFacts{Film: g.Film, MapModule: g.MapModule, AxisW: g.AxisW,
			LayoutDetected: g.LayoutDetected, FilmInputs: s.in},
	})
	if err != nil {
		t.Fatalf("%s : encodage des faits : %v", cas.nom, err)
	}
	f, err := DecodeFilmFactsFile(blob, entry)
	if err != nil {
		t.Fatalf("%s : relecture des faits : %v", cas.nom, err)
	}
	return depuisLeFilm, BuildFromFacts(goldenFilm, "halo_infinite", f, Options{MapQuant: &entry})
}

// TestLeRejeuDepuisLesFaitsRendLeVerdictDuFilDesMorts : film a fil LU, VIDE et ILLISIBLE — le
// document rejoue depuis les faits est, A L OCTET, celui du decodage, et il publie le verdict.
func TestLeRejeuDepuisLesFaitsRendLeVerdictDuFilDesMorts(t *testing.T) {
	for _, cas := range casDesTroisIssues(t) {
		depuisLeFilm, depuisLesFaits := documentsDuFilmEtDesFaits(t, cas)
		if depuisLeFilm.Coverage == nil || depuisLesFaits.Coverage == nil {
			t.Fatalf("%s : document sans couverture", cas.nom)
		}
		if got := depuisLeFilm.Coverage.Bridge.DeathsFeed; got != cas.want {
			t.Fatalf("%s : le decodage publie %q, attendu %q — le temoin ne mesure plus rien",
				cas.nom, got, cas.want)
		}
		if got := depuisLesFaits.Coverage.Bridge.DeathsFeed; got != cas.want {
			t.Errorf("%s : le rejeu depuis les faits publie deathsFeed = %q, le decodage %q : le "+
				"verdict du fil des morts ne voyage pas dans les faits", cas.nom, got, cas.want)
		}
		a, errA := json.Marshal(depuisLeFilm)
		b, errB := json.Marshal(depuisLesFaits)
		if errA != nil || errB != nil {
			t.Fatalf("%s : serialisation : %v / %v", cas.nom, errA, errB)
		}
		if string(a) != string(b) {
			t.Errorf("%s : document rejoue depuis les faits (%d o) != document du decodage (%d o)",
				cas.nom, len(b), len(a))
		}
	}
}

// TestLErreurRelueGardeLeTexteDeLaLecture : l erreur reconstruite depuis les faits existe quand la
// lecture du film en a rendu une, et porte son TEXTE — ce que ses consommateurs lisent (presence,
// journal). Le verdict, lui, voyage tel quel : c est lui qui distingue le fil vide de l illisible.
func TestLErreurRelueGardeLeTexteDeLaLecture(t *testing.T) {
	for _, cas := range casDesTroisIssues(t) {
		film, err := source.Load(source.MemoryChunks(cas.chunks), cas.meta)
		if err != nil {
			t.Fatalf("%s : chargement : %v", cas.nom, err)
		}
		deaths, errLue := ScanDeaths(film)
		v := verdictDeLaLecture(deaths, errLue)
		w := &gwriter{}
		encodeVerdictDuFilDesMorts(w, v)
		relu := decodeVerdictDuFilDesMorts(&greader{b: w.b})
		if relu != v || relu.Verdict != cas.want {
			t.Fatalf("%s : verdict relu %+v, ecrit %+v, attendu %q", cas.nom, relu, v, cas.want)
		}
		errRelue := relu.Erreur()
		if (errRelue == nil) != (errLue == nil) {
			t.Fatalf("%s : erreur relue %v, erreur lue %v", cas.nom, errRelue, errLue)
		}
		if errLue != nil && errRelue.Error() != errLue.Error() {
			t.Errorf("%s : texte relu %q, lu %q", cas.nom, errRelue, errLue)
		}
	}
}

// TestFaitsSansVerdictDuFilDesMortsSontRefuses : UN FICHIER AU SCHEMA 4 ECRIT AVANT LE LOT M8 N EST
// JAMAIS SERVI SANS VERDICT.
//
// Le lot M8 a ajoute le verdict au complement de la section 1 SANS monter `SchemaDesFaits` : la
// montee de la vague D (3 -> 4) n etait pas publiee (arbitrage du plan, §9). Un tel fichier a donc
// un EN-TETE FRAIS, et son refus ne peut tomber qu au decodage de la section 1 — le chemin
// « illisible malgre un en-tete frais », sur lequel la cuisson redecode le film. Ce test tient
// cette promesse du commentaire de [SchemaDesFaits] : un decodeur qui tolererait la fin de section
// servirait un verdict vide, donc un document qui tait `deathsFeed` la ou le film le publie.
func TestFaitsSansVerdictDuFilDesMortsSontRefuses(t *testing.T) {
	entry := goldenEntryPourTest(t)
	f := fichierTemoin(t)
	f.Coverage.SourceRev, f.Coverage.ProfileRev = source.Rev, profile.Rev
	f.Coverage.GrammarRev, f.Coverage.FactsRev = grammar.Rev, facts.Rev
	f.Facts.DeathsFeed = VerdictDuFilDesMorts{Verdict: DeathsFeedUnreadable, Cause: "pas de temps forts"}
	blob, err := EncodeFilmFactsFile(f)
	if err != nil {
		t.Fatalf("encodage : %v", err)
	}
	if relu, err := DecodeFilmFactsFile(blob, entry); err != nil || relu.Facts.DeathsFeed != f.Facts.DeathsFeed {
		t.Fatalf("controle : le fichier complet ne rend pas son verdict (%v)", err)
	}
	avantM8 := sansLeVerdictDuFilDesMorts(t, blob, f.Facts.DeathsFeed)
	e, err := DecodeFilmFactsEntete(avantM8)
	if err != nil || e.Utilisable(entry) != nil {
		t.Fatalf("le fichier d avant M8 doit avoir un en-tete FRAIS (schema %d) : %v", e.Schema, err)
	}
	if relu, err := DecodeFilmFactsFile(avantM8, entry); err == nil {
		t.Fatalf("un fichier au schema %d sans verdict du fil des morts est SERVI (verdict relu %+v) : "+
			"il doit etre refuse au decodage de la section 1, puis redecode", SchemaDesFaits,
			relu.Facts.DeathsFeed)
	}
}

// sansLeVerdictDuFilDesMorts rend `blob` tel que l ecrivait le codec d avant M8 : la section 1
// privee des deux chaines du verdict, recadree, le reste du fichier a l octet.
func sansLeVerdictDuFilDesMorts(t *testing.T, blob []byte, v VerdictDuFilDesMorts) []byte {
	t.Helper()
	r := &greader{b: blob, off: len(magieFaitsDeFilm)}
	r.u()
	r.u()
	r.tranche(int(r.u()))
	debut := r.off
	id := int(r.u())
	charge := r.tranche(int(r.u()))
	suffixe := &gwriter{}
	encodeVerdictDuFilDesMorts(suffixe, v)
	if r.err != nil || id != sectionEntrees || !bytes.HasSuffix(charge, suffixe.b) {
		t.Fatalf("section 1 introuvable ou sans verdict en dernier (id %d, err %v)", id, r.err)
	}
	w := &gwriter{b: append([]byte(nil), blob[:debut]...)}
	ecrireSection(w, id, charge[:len(charge)-len(suffixe.b)])
	return append(w.b, blob[r.off:]...)
}
