package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/replaybuild"
)

const (
	shaTest      = "0123456789abcdef0123456789abcdef01234567"
	artefactTest = `{"schemaVersion":1,"matchId":"abcd1234"}`
	faitsTest    = "FILMFACTS-octets-binaires"
)

func cleDeTest() cleBase {
	return cleBase{
		Version: versionCacheBase, BaseSHA: shaTest, Temoin: "abcd1234", TitleSlug: "halo_infinite",
		MatchID: "abcd1234-0000", FaitsSHA256: "f", FilmSHA256: "m", GoVersion: "go1.26.0",
		GOOS: "windows", GOARCH: "amd64", MemGiB: 4,
	}
}

// capturerLogs redirige slog vers un tampon pour la duree du test.
func capturerLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	ancien := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(ancien) })
	return &buf
}

func artefactCuit(t *testing.T, contenu string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "cuit.json")
	if err := os.WriteFile(p, []byte(contenu), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestCleBaseStable(t *testing.T) {
	a, b := cleDeTest(), cleDeTest()
	if a.empreinte() != b.empreinte() {
		t.Fatalf("deux cles identiques, deux empreintes : %s / %s", a.empreinte(), b.empreinte())
	}
	if err := a.valider(); err != nil {
		t.Fatalf("cle complete refusee : %v", err)
	}
}

// TestCleBaseInvalideQuandUnParametreChange — chaque champ de la cle, pris UN A UN par
// reflexion (un champ ajoute plus tard est couvert sans toucher ce test), doit changer
// l'empreinte ET rendre l'ancienne entree introuvable. Retirer le SHA de la cle le fait tomber.
func TestCleBaseInvalideQuandUnParametreChange(t *testing.T) {
	ref := cleDeTest()
	v := reflect.TypeFor[cleBase]()
	for i := 0; i < v.NumField(); i++ {
		modif := ref
		f := reflect.ValueOf(&modif).Elem().Field(i)
		switch f.Kind() {
		case reflect.String:
			f.SetString(f.String() + "x")
		case reflect.Int:
			f.SetInt(f.Int() + 1)
		default:
			t.Fatalf("champ %s de type inattendu %s", v.Field(i).Name, f.Kind())
		}
		if modif.empreinte() == ref.empreinte() {
			t.Errorf("changer %s ne change pas l'empreinte : le cache servirait un artefact perime", v.Field(i).Name)
		}
		bc := baseCache{Racine: t.TempDir()}
		if err := bc.ranger(ref, artefactCuit(t, artefactTest), faitsCuits(t)); err != nil {
			t.Fatal(err)
		}
		if _, ok := bc.chercher(context.Background(), modif); ok {
			t.Errorf("changer %s : l'ancienne entree est servie", v.Field(i).Name)
		}
	}
}

func TestCleBaseIncompleteRefusee(t *testing.T) {
	for nom, mut := range map[string]func(*cleBase){
		"SHA court":  func(c *cleBase) { c.BaseSHA = "abc123" },
		"SHA vide":   func(c *cleBase) { c.BaseSHA = "" },
		"branche":    func(c *cleBase) { c.BaseSHA = "origin/feat/v75" },
		"go inconnu": func(c *cleBase) { c.GoVersion = "" },
		"film vide":  func(c *cleBase) { c.FilmSHA256 = "" },
		"faits vide": func(c *cleBase) { c.FaitsSHA256 = "" },
	} {
		c := cleDeTest()
		mut(&c)
		if c.valider() == nil {
			t.Errorf("%s : cle incomplete acceptee", nom)
		}
		if err := (baseCache{Racine: t.TempDir()}).ranger(c, artefactCuit(t, artefactTest), faitsCuits(t)); err == nil {
			t.Errorf("%s : rangement sur cle incomplete", nom)
		}
	}
}

func TestRangerPuisChercherRelitLArtefactEtLesFaits(t *testing.T) {
	bc := baseCache{Racine: t.TempDir()}
	c := cleDeTest()
	if _, ok := bc.chercher(context.Background(), c); ok {
		t.Fatal("hit sur un cache vide")
	}
	if err := bc.ranger(c, artefactCuit(t, artefactTest), faitsCuits(t)); err != nil {
		t.Fatal(err)
	}
	e, ok := bc.chercher(context.Background(), c)
	if !ok {
		t.Fatal("miss apres rangement")
	}
	got, _ := os.ReadFile(e.Artefact)
	if string(got) != artefactTest {
		t.Fatalf("artefact relu %q", got)
	}
	faits, _ := os.ReadFile(e.Faits)
	if string(faits) != faitsTest {
		t.Fatalf("faits du film relus %q", faits)
	}
	if !strings.HasSuffix(e.Faits, ".filmfacts.bin") {
		t.Fatalf("les faits doivent porter l'extension canonique, got %s", e.Faits)
	}
}

// TestCacheCorromptuIgnoreEtJournalise — chaque forme de corruption est un miss ET un Warn, y
// compris une entree qui a l'artefact SANS les faits (incomplete).
func TestCacheCorromptuIgnoreEtJournalise(t *testing.T) {
	c := cleDeTest()
	cas := map[string]func(e entreeBase){
		"artefact tronque": func(e entreeBase) { _ = os.WriteFile(e.Artefact, []byte(`{"schemaV`), 0o600) },
		"artefact altere": func(e entreeBase) {
			_ = os.WriteFile(e.Artefact, []byte(strings.Replace(artefactTest, "1", "2", 1)), 0o600)
		},
		"artefact supprime": func(e entreeBase) { _ = os.Remove(e.Artefact) },
		"faits supprimes":   func(e entreeBase) { _ = os.Remove(e.Faits) },
		"faits altere":      func(e entreeBase) { _ = os.WriteFile(e.Faits, []byte("autre"), 0o600) },
		"faits tronques":    func(e entreeBase) { _ = os.WriteFile(e.Faits, []byte(faitsTest[:2]), 0o600) },
		"meta illisible":    func(e entreeBase) { _ = os.WriteFile(e.Meta, []byte("pas du json"), 0o600) },
		"meta autre cle": func(e entreeBase) {
			modifierMeta(t, e.Meta, func(mc *metaCacheBase) { mc.Cle.FilmSHA256 = "autre" })
		},
		"artefact non JSON": func(e entreeBase) { remplacerArtefactEtMeta(t, e, "pas du json") },
		"meta vide":         func(e entreeBase) { _ = os.WriteFile(e.Meta, nil, 0o600) },
		"taille incoherente": func(e entreeBase) {
			modifierMeta(t, e.Meta, func(mc *metaCacheBase) { mc.ArtefactOctets++ })
		},
		"taille des faits incoherente": func(e entreeBase) {
			modifierMeta(t, e.Meta, func(mc *metaCacheBase) { mc.FaitsOctets++ })
		},
	}
	for nom, corrompre := range cas {
		t.Run(nom, func(t *testing.T) {
			logs := capturerLogs(t)
			bc := baseCache{Racine: t.TempDir()}
			if err := bc.ranger(c, artefactCuit(t, artefactTest), faitsCuits(t)); err != nil {
				t.Fatal(err)
			}
			corrompre(bc.chemins(c))
			if _, ok := bc.chercher(context.Background(), c); ok {
				t.Fatal("entree corrompue ou incomplete servie comme un hit")
			}
			if !strings.Contains(logs.String(), "level=WARN") || !strings.Contains(logs.String(), "recuisson") {
				t.Fatalf("corruption non journalisee en Warn :\n%s", logs)
			}
		})
	}
}

func TestFichiersSansMetaSontUnMiss(t *testing.T) {
	bc := baseCache{Racine: t.TempDir()}
	c := cleDeTest()
	e := bc.chemins(c)
	_ = os.MkdirAll(filepath.Dir(e.Artefact), 0o750)
	_ = os.WriteFile(e.Artefact, []byte(artefactTest), 0o600)
	_ = os.WriteFile(e.Faits, []byte(faitsTest), 0o600)
	if _, ok := bc.chercher(context.Background(), c); ok {
		t.Fatal("des fichiers sans meta (crash avant le marqueur) sont servis")
	}
}

// TestEcritureAtomique — aucun temporaire ne survit, et un rangement qui echoue (source
// absente) ne laisse ni artefact, ni faits, ni meta.
func TestEcritureAtomique(t *testing.T) {
	bc := baseCache{Racine: t.TempDir()}
	c := cleDeTest()
	absent := filepath.Join(t.TempDir(), "absent")
	for nom, args := range map[string][2]string{
		"artefact absent": {absent, faitsCuits(t)},
		"faits absents":   {artefactCuit(t, artefactTest), absent},
	} {
		if err := bc.ranger(c, args[0], args[1]); err == nil {
			t.Fatalf("%s : rangement sans erreur", nom)
		}
		if _, ok := bc.chercher(context.Background(), c); ok {
			t.Fatalf("%s : une entree incomplete est servie", nom)
		}
		if _, err := os.Stat(bc.chemins(c).Meta); err == nil {
			t.Fatalf("%s : un meta existe apres un rangement echoue", nom)
		}
	}
	if err := bc.ranger(c, artefactCuit(t, artefactTest), faitsCuits(t)); err != nil {
		t.Fatal(err)
	}
	entrees, _ := os.ReadDir(filepath.Dir(bc.chemins(c).Meta))
	var noms []string
	for _, e := range entrees {
		noms = append(noms, e.Name())
	}
	if len(noms) != 3 {
		t.Fatalf("attendu exactement artefact + faits + meta, trouve %v", noms)
	}
	// Le rangement remplace une entree existante sans laisser de fichier partiel.
	if err := bc.ranger(c, artefactCuit(t, artefactTest), faitsCuits(t)); err != nil {
		t.Fatal(err)
	}
	if _, ok := bc.chercher(context.Background(), c); !ok {
		t.Fatal("entree remplacee illisible")
	}
}

func TestResoudreAvecCache(t *testing.T) {
	bc := baseCache{Racine: t.TempDir()}
	c := cleDeTest()
	appels := 0
	cuire := func() (string, string, error) {
		appels++
		return artefactCuit(t, artefactTest), faitsCuits(t), nil
	}
	resoudre := func(forcer bool) baseResolue {
		t.Helper()
		r, err := resoudreAvecCache(context.Background(), bc, c, forcer, cuire)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}

	if r := resoudre(false); r.DuCache || appels != 1 || !r.ArtefactEnCache || !r.FaitsEnCache {
		t.Fatalf("1er passage : %+v appels=%d, attendu cuisson puis entree rangee (A+F)", r, appels)
	}
	if r := resoudre(false); !r.DuCache || appels != 1 || !r.ArtefactEnCache || !r.FaitsEnCache {
		t.Fatalf("2e passage : %+v appels=%d, attendu cache sans cuisson", r, appels)
	}
	if r := resoudre(true); r.DuCache || appels != 2 {
		t.Fatalf("--sans-cache-base : %+v appels=%d, attendu recuisson", r, appels)
	}
	capturerLogs(t)
	// Un cache qui a l'artefact mais PAS les faits est incomplet : recuisson.
	_ = os.Remove(bc.chemins(c).Faits)
	if r := resoudre(false); r.DuCache || appels != 3 {
		t.Fatalf("artefact sans faits : %+v appels=%d, attendu recuisson", r, appels)
	}
	_ = os.WriteFile(bc.chemins(c).Artefact, []byte("casse"), 0o600)
	if r := resoudre(false); r.DuCache || appels != 4 {
		t.Fatalf("cache corrompu : %+v appels=%d, attendu recuisson", r, appels)
	}
	if r := resoudre(false); !r.DuCache || appels != 4 {
		t.Fatalf("apres recuisson le cache doit etre repare : %+v appels=%d", r, appels)
	}
}

// TestCuissonSansFaitsNeRangeRien — la cuisson n'a pas laisse de faits : on compare quand meme
// avec la cuisson fraiche, mais rien n'est range (une entree sans faits serait incomplete) et
// le rapport le dit (ni A ni F).
func TestCuissonSansFaitsNeRangeRien(t *testing.T) {
	logs := capturerLogs(t)
	bc := baseCache{Racine: t.TempDir()}
	c := cleDeTest()
	r, err := resoudreAvecCache(context.Background(), bc, c, false, func() (string, string, error) {
		return artefactCuit(t, artefactTest), filepath.Join(t.TempDir(), "absent.filmfacts.bin"), nil
	})
	if err != nil || r.DuCache || r.ArtefactEnCache || r.FaitsEnCache {
		t.Fatalf("resolution %+v err=%v, attendu cuisson fraiche sans rien en cache", r, err)
	}
	if _, ok := bc.chercher(context.Background(), c); ok {
		t.Fatal("une entree sans faits a ete rangee")
	}
	if !strings.Contains(logs.String(), "level=WARN") {
		t.Fatalf("l'absence de faits n'est pas journalisee :\n%s", logs)
	}
}

func TestResoudreAvecCacheErreurDeCuissonPasRangee(t *testing.T) {
	bc := baseCache{Racine: t.TempDir()}
	c := cleDeTest()
	boom := errors.New("cuisson en echec")
	if _, err := resoudreAvecCache(context.Background(), bc, c, false,
		func() (string, string, error) { return "", "", boom }); !errors.Is(err, boom) {
		t.Fatalf("erreur de cuisson avalee : %v", err)
	}
	if _, ok := bc.chercher(context.Background(), c); ok {
		t.Fatal("une cuisson en echec a ete rangee")
	}
}

func TestEmpreinteFaitsEtFilm(t *testing.T) {
	f1 := replaybuild.FactsFile{MatchID: "m", MapNames: []string{"a"}}
	f2 := replaybuild.FactsFile{MatchID: "m", MapNames: []string{"a", "b"}}
	e1, _ := empreinteFaits(f1)
	e1bis, _ := empreinteFaits(f1)
	e2, _ := empreinteFaits(f2)
	if e1 != e1bis || e1 == e2 {
		t.Fatalf("empreinte des faits : stable=%v distincte=%v", e1 == e1bis, e1 != e2)
	}

	work := t.TempDir()
	cacheRoot := title.NewPathResolver(work).CacheRootDir()
	man := filmcache.ManifestPath(cacheRoot, "abcd1234")
	chunk := filepath.Join(filmcache.ChunkDir(cacheRoot, "abcd1234"), "c0.bin")
	for p, s := range map[string]string{man: "{}", chunk: "octets"} {
		_ = os.MkdirAll(filepath.Dir(p), 0o750)
		_ = os.WriteFile(p, []byte(s), 0o600)
	}
	h1, err := empreinteFilm(work, "abcd1234")
	h1bis, _ := empreinteFilm(work, "abcd1234")
	if err != nil || h1 != h1bis {
		t.Fatalf("empreinte du film instable : %v", err)
	}
	_ = os.WriteFile(chunk, []byte("octetS"), 0o600)
	if h2, _ := empreinteFilm(work, "abcd1234"); h2 == h1 {
		t.Fatal("un chunk modifie ne change pas l'empreinte du film")
	}
	if _, err := empreinteFilm(work, "inconnu"); err == nil {
		t.Fatal("film absent : erreur attendue (cle non capturable)")
	}
}

func TestCleCacheBaseNonCapturableSansSHA(t *testing.T) {
	work := t.TempDir()
	cacheRoot := title.NewPathResolver(work).CacheRootDir()
	man := filmcache.ManifestPath(cacheRoot, "abcd1234")
	_ = os.MkdirAll(filepath.Dir(man), 0o750)
	_ = os.WriteFile(man, []byte("{}"), 0o600)
	tc := temoinContexte{WorkRootBase: work, TitleSlug: "halo_infinite", BaseGoVersion: "go1"}
	if _, err := tc.cleCacheBase(Temoin{ID: "abcd1234"}, replaybuild.FactsFile{MatchID: "m"}); err == nil {
		t.Fatal("cle sans SHA de base acceptee")
	}
}

func modifierMeta(t *testing.T, m string, f func(*metaCacheBase)) {
	t.Helper()
	blob, err := os.ReadFile(m)
	if err != nil {
		t.Fatal(err)
	}
	var mc metaCacheBase
	if err := json.Unmarshal(blob, &mc); err != nil {
		t.Fatal(err)
	}
	f(&mc)
	out, _ := json.Marshal(mc)
	if err := os.WriteFile(m, out, 0o600); err != nil {
		t.Fatal(err)
	}
}

// remplacerArtefactEtMeta ecrit un contenu non JSON avec un meta qui en decrit EXACTEMENT les
// octets : seule la decodabilite de l'artefact est alors en cause.
func remplacerArtefactEtMeta(t *testing.T, e entreeBase, contenu string) {
	t.Helper()
	_ = os.WriteFile(e.Artefact, []byte(contenu), 0o600)
	somme, n, _ := sha256Fichier(e.Artefact)
	modifierMeta(t, e.Meta, func(mc *metaCacheBase) { mc.ArtefactSHA256, mc.ArtefactOctets = somme, n })
}

func faitsCuits(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "abcd1234"+title.ExtensionFilmFacts)
	if err := os.WriteFile(p, []byte(faitsTest), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestRapportIndiqueLOrigineEtLaPresenceDansLeCache(t *testing.T) {
	lignes := []ligneRapport{
		{Temoin: Temoin{ID: "aaaa1111"}, BaseDuCache: true, BaseArtefactEnCache: true, BaseFaitsEnCache: true},
		{Temoin: Temoin{ID: "bbbb2222"}, BaseArtefactEnCache: true},
		{Temoin: Temoin{ID: "cccc3333"}},
	}
	var b strings.Builder
	imprimerTableau(&b, lignes, "base")
	out := b.String()
	for _, attendu := range []string{"cache", "cuite", "A+F", "  A ", " -  "} {
		if !strings.Contains(out, attendu) {
			t.Errorf("le tableau ne porte pas %q :\n%s", attendu, out)
		}
	}
	j0, j1, j2 := ligneVersJSON(lignes[0]), ligneVersJSON(lignes[1]), ligneVersJSON(lignes[2])
	if !j0.BaseDuCache || !j0.BaseArtefactCache || !j0.BaseFaitsCache {
		t.Errorf("JSON du hit : %+v", j0)
	}
	if j1.BaseDuCache || !j1.BaseArtefactCache || j1.BaseFaitsCache {
		t.Errorf("JSON de l'entree sans faits : %+v", j1)
	}
	if j2.BaseDuCache || j2.BaseArtefactCache || j2.BaseFaitsCache {
		t.Errorf("JSON sans cache : %+v", j2)
	}
}
