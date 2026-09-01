package replayartifacts

// mvar_rattrapage_test.go — LES TROIS PROMESSES DU RATTRAPAGE.
//
//	1. carte CONNUE   -> zero appel reseau, et l'entree existante ne bouge pas d'un octet ;
//	2. carte INCONNUE -> entree ajoutee, les autres byte-identiques ;
//	3. tout ECHEC     -> journalise et compte, sans jamais interrompre quoi que ce soit.
//
// LE COMPTEUR D'APPELS EST LA PIECE MAITRESSE du premier test : « la carte n'a pas ete
// modifiee » se verifie par comparaison, mais « aucun appel n'a ete fait » ne se verifie que
// par un espion. Sans lui, un rattrapage qui telechargerait a chaque match — la lourdeur meme
// que cette conception refuse — passerait inapercu.

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/analysis/replay/mapvar"
	"levelup/go-api/internal/port"
)

// mrFetcherEspion compte ses appels et rend ce qu'on lui dit de rendre.
type mrFetcherEspion struct {
	appels int
	blob   []byte
	base   string
	err    error
}

func (f *mrFetcherEspion) FetchMvarForMap(_ context.Context, _, _ string) ([]byte, string, error) {
	f.appels++
	return f.blob, f.base, f.err
}

// mrDeps monte une arborescence minimale : catalogue des socles + catalogue d'objectifs.
func mrDeps(t *testing.T, cartesAuCatalogue []string, cartesAuxObjectifs []string) (Deps, string) {
	t.Helper()
	racine := t.TempDir()
	dir := filepath.Join(racine, "data", "titles", "halo_infinite", "reference")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	pads := &replay.MapWeaponPadsCatalog{
		SchemaVersion: replay.MapWeaponPadsSchemaVersion, TitleSlug: "halo_infinite",
		Maps: map[string]replay.MapWeaponPadsEntry{},
	}
	for _, id := range cartesAuCatalogue {
		sp := []replay.MapSpawnPointSpot{
			{Pos: mapvar.Vec3{X: 1}, TypeID: "0xADEEE6D8", Kind: "grenade", Objects: 1},
		}
		pads.Maps[id] = replay.MapWeaponPadsEntry{
			MapID: id, MvarFile: id + ".mvar", ObjectsN: 400, LevelID: 7,
			Pads: []replay.MapWeaponPadSpot{{
				Pos: mapvar.Vec3{X: 2}, TypeID: "0x5F379533", Family: "power", Objects: 1}},
			SpawnPoints: &sp,
		}
	}
	obj := &replay.MapObjectivesCatalog{
		SchemaVersion: replay.MapObjectivesSchemaVersion,
		Maps:          map[string]replay.MapObjectivesEntry{},
	}
	for _, id := range cartesAuxObjectifs {
		obj.Maps[id] = replay.MapObjectivesEntry{MapID: id, MvarFile: id + ".mvar"}
	}
	mrEcrire(t, filepath.Join(dir, "map_weapon_pads.json"), pads)
	mrEcrire(t, filepath.Join(dir, "map_objectives.json"), obj)
	return Deps{RepoRoot: racine, TitleSlug: "halo_infinite", CacheRoot: t.TempDir()},
		filepath.Join(dir, "map_weapon_pads.json")
}

func mrEcrire(t *testing.T, path string, v any) {
	t.Helper()
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
}

func mrTravail(mapIDs ...string) []buildWork {
	out := make([]buildWork, 0, len(mapIDs))
	for i, id := range mapIDs {
		out = append(out, buildWork{
			matchID: "match-" + id + "-" + string(rune('a'+i)),
			facts:   port.MatchFacts{MapID: id},
		})
	}
	return out
}

// TestRattrapageCarteConnueNeFaitAucunAppel — promesse 1.
func TestRattrapageCarteConnueNeFaitAucunAppel(t *testing.T) {
	d, catPath := mrDeps(t, []string{"connue"}, []string{"connue"})
	avant, err := os.ReadFile(catPath)
	if err != nil {
		t.Fatal(err)
	}
	esp := &mrFetcherEspion{}
	// DEUX matchs de la MEME carte connue : ni l'un ni l'autre ne doit declencher d'appel.
	b := rattraperCartesAbsentes(context.Background(), d, mrTravail("connue", "connue"), esp)
	if esp.appels != 0 {
		t.Errorf("carte CONNUE : %d appel(s) reseau, attendu 0 — verifier la derive d'une carte "+
			"connue couterait un appel PAR MATCH, ce que cette conception refuse", esp.appels)
	}
	if b.dejaLa != 1 || b.ajoutees != 0 {
		t.Errorf("bilan = %+v, attendu dejaLa 1 et ajoutees 0 (les deux matchs sont la meme "+
			"carte : le lot ne la traite qu'une fois)", b)
	}
	apres, err := os.ReadFile(catPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(avant) != string(apres) {
		t.Error("le catalogue a ete REECRIT alors qu'aucune carte n'etait a ajouter")
	}
}

// TestRattrapageCarteInconnueAjouteSansToucherLesAutres — promesse 2.
func TestRattrapageCarteInconnueAjouteSansToucherLesAutres(t *testing.T) {
	d, catPath := mrDeps(t, []string{"connue"}, []string{"connue", "inconnue"})
	avant := mrLire(t, catPath)
	mrAvecEntreeFactice(t, nil)
	esp := &mrFetcherEspion{blob: []byte("peu importe"), base: "inconnue.mvar"}
	b := rattraperCartesAbsentes(context.Background(), d, mrTravail("inconnue"), esp)
	if esp.appels != 1 {
		t.Fatalf("carte INCONNUE : %d appel(s), attendu exactement 1", esp.appels)
	}
	if b.ajoutees != 1 {
		t.Fatalf("bilan = %+v, attendu ajoutees 1", b)
	}
	apres := mrLire(t, catPath)
	if _, ok := apres.Maps["inconnue"]; !ok {
		t.Error("la carte inconnue n'a pas ete ajoutee au catalogue")
	}
	// L'ENTREE EXISTANTE EST BYTE-IDENTIQUE : c'est la promesse de l'ajout-seul.
	a, _ := json.Marshal(avant.Maps["connue"])
	z, _ := json.Marshal(apres.Maps["connue"])
	if string(a) != string(z) {
		t.Errorf("l'entree existante a CHANGE :\navant %s\napres %s", a, z)
	}
	// LE `.mvar` EST DEPOSE au cache, pour que la passe soit rejouable hors ligne.
	if _, err := os.Stat(filepath.Join(d.CacheRoot, "mvar", "inconnue", "inconnue.mvar")); err != nil {
		t.Errorf("le .mvar n'a pas ete depose au cache : %v", err)
	}
}

// TestRattrapageEchecNInterrompRien — promesse 3.
//
// Trois formes d'echec, et aucune ne doit faire autre chose que compter : reseau, variante
// illisible, et catalogue d'objectifs qui ignore la carte.
func TestRattrapageEchecNInterrompRien(t *testing.T) {
	cas := []struct {
		nom       string
		esp       *mrFetcherEspion
		objectifs []string
		echecs    int
		hors      int
	}{
		{"echec reseau", &mrFetcherEspion{err: errors.New("503")}, []string{"inconnue"}, 1, 0},
		{"variante illisible", &mrFetcherEspion{blob: []byte("pas un mvar"), base: "x.mvar"},
			[]string{"inconnue"}, 1, 0},
		{"carte hors catalogue d'objectifs", &mrFetcherEspion{}, nil, 0, 1},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			d, catPath := mrDeps(t, []string{"connue"}, c.objectifs)
			avant, err := os.ReadFile(catPath)
			if err != nil {
				t.Fatal(err)
			}
			// L'APPEL NE PANIQUE PAS ET REND LA MAIN : c'est ce que « le fetch de film
			// continue » veut dire au niveau de cette fonction.
			b := rattraperCartesAbsentes(context.Background(), d, mrTravail("inconnue"), c.esp)
			if b.echecs != c.echecs || b.horsObjectifs != c.hors {
				t.Errorf("bilan = %+v, attendu echecs %d et horsObjectifs %d",
					b, c.echecs, c.hors)
			}
			if b.ajoutees != 0 {
				t.Errorf("un echec ne doit RIEN ajouter, bilan = %+v", b)
			}
			apres, err := os.ReadFile(catPath)
			if err != nil {
				t.Fatal(err)
			}
			if string(avant) != string(apres) {
				t.Error("le catalogue a change alors que le rattrapage a echoue")
			}
		})
	}
}

// TestRattrapageSansFetcherEstInactif — le client sans la capacite desarme, il ne casse pas.
func TestRattrapageSansFetcherEstInactif(t *testing.T) {
	d, catPath := mrDeps(t, []string{"connue"}, []string{"connue", "inconnue"})
	avant, err := os.ReadFile(catPath)
	if err != nil {
		t.Fatal(err)
	}
	b := rattraperCartesAbsentes(context.Background(), d, mrTravail("inconnue"), nil)
	if b != (bilanRattrapage{}) {
		t.Errorf("sans fetcher, le bilan doit etre vide, obtenu %+v", b)
	}
	apres, _ := os.ReadFile(catPath)
	if string(avant) != string(apres) {
		t.Error("sans fetcher, le catalogue ne doit pas bouger")
	}
}

// mrAvecEntreeFactice remplace la construction d'entree par une fonction controlee.
//
// Le chemin « variante illisible » NE l'utilise PAS : il laisse la vraie fonction echouer sur
// des octets qui n'en sont pas — c'est le seul moyen de verifier que l'echec de parse est bien
// traite comme un echec.
func mrAvecEntreeFactice(t *testing.T, err error) {
	t.Helper()
	ancien := entryFromMvarFn
	entryFromMvarFn = func(mapID string, e replay.MapObjectivesEntry, _ []byte, base string,
	) (replay.MapWeaponPadsEntry, int, int, error) {
		if err != nil {
			return replay.MapWeaponPadsEntry{}, 0, 0, err
		}
		sp := []replay.MapSpawnPointSpot{
			{Pos: mapvar.Vec3{X: 9}, TypeID: "0xE42158DF", Kind: "equipment", Objects: 1},
		}
		return replay.MapWeaponPadsEntry{
			MapID: mapID, MvarFile: base, ObjectsN: 321, LevelID: 11,
			Pads: []replay.MapWeaponPadSpot{{
				Pos: mapvar.Vec3{X: 8}, TypeID: "0x6253CFC0", Family: "rack", Objects: 1}},
			SpawnPoints: &sp,
		}, 0, 0, nil
	}
	t.Cleanup(func() { entryFromMvarFn = ancien })
}

func mrLire(t *testing.T, path string) *replay.MapWeaponPadsCatalog {
	t.Helper()
	cat, err := replay.LoadMapWeaponPads(path)
	if err != nil {
		t.Fatalf("relecture du catalogue : %v", err)
	}
	return cat
}
