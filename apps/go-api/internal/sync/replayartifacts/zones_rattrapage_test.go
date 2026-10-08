package replayartifacts

// zones_rattrapage_test.go — LES PROMESSES DU RATTRAPAGE DES ZONES NOMMÉES.
//
//	1. carte DÉJÀ COUVERTE      -> zéro appel réseau, rien d'écrit ;
//	2. variante AU CACHE        -> lue hors ligne, carte ajoutée au catalogue GÉNÉRÉ ;
//	3. variante ABSENTE         -> UN appel par carte (jamais par match), dépôt au cache ;
//	4. hors ligne / à blanc     -> aucun appel, rien d'écrit ;
//	5. tout ÉCHEC               -> compté, jamais fatal ; capability absente -> rien.
//
// L'espion de fetcher (mvar_rattrapage_test.go) est la pièce maîtresse : « aucun appel » ne se
// vérifie que par lui.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/mapcatalog"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/port"
)

// Contenus de variante reconnus par l'extraction factice (cf. zrExtractionFactice).
const (
	zrVarianteAvecZones = "AVEC-ZONES"
	zrVarianteSansZone  = "SANS-ZONE"
	zrVarianteMuette    = "ZONES-MUETTES"
	zrSidConnu          = 0x4E9E5E09
	zrSidInconnu        = 0x0BADF00D
	zrMapIDVersionnee   = "aaaaaaaa-0000-0000-0000-00000000000a"
)

// zrExtractionFactice remplace la chaîne `.mvar` -> entrée : le dépôt ne versionne aucune
// variante porteuse de zones (cf. entreeZonesFn).
func zrExtractionFactice(t *testing.T) {
	t.Helper()
	avant := entreeZonesFn
	entreeZonesFn = func(blob []byte, lex mapcatalog.Lexique) (replay.MapCalloutsEntry, mapcatalog.CouvertureLibelles, error) {
		var sid uint32
		switch string(blob) {
		case zrVarianteAvecZones:
			sid = zrSidConnu
		case zrVarianteMuette:
			sid = zrSidInconnu
		case zrVarianteSansZone:
			return replay.MapCalloutsEntry{Provenance: replay.CalloutsProvenanceMvar}, mapcatalog.CouvertureLibelles{}, nil
		default:
			return replay.MapCalloutsEntry{}, mapcatalog.CouvertureLibelles{}, errors.New("variante illisible")
		}
		l, connu := lex[sid]
		couv := mapcatalog.CouvertureLibelles{Zones: 1, StringIDs: map[uint32]bool{sid: connu}}
		if connu {
			couv.Nommees = 1
		}
		return replay.MapCalloutsEntry{Provenance: replay.CalloutsProvenanceMvar, Zones: []replay.CalloutZone{{
			VolumeIndex: 3, EN: l.EN, FR: l.FR, Polygon: [][2]float64{{0, 0}, {5, 0}, {5, 5}},
		}}}, couv, nil
	}
	t.Cleanup(func() { entreeZonesFn = avant })
}

// zrRacine monte une racine de test : catalogue versionné (une carte Forge), lexique (un nom),
// et la configuration LIVRÉE du titre pour la porte de capability.
func zrRacine(t *testing.T) string {
	t.Helper()
	racine := t.TempDir()
	src := filepath.Join(racineDepot(t), "config", "titles", title.DefaultSlug)
	dst := filepath.Join(racine, "config", "titles", title.DefaultSlug)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.CopyFS(dst, os.DirFS(src)); err != nil {
		t.Fatalf("copie de la configuration du titre : %v", err)
	}
	res := title.NewPathResolver(racine)
	versionne := res.MapCalloutsPath(title.DefaultSlug)
	if err := os.MkdirAll(filepath.Dir(versionne), 0o755); err != nil {
		t.Fatal(err)
	}
	mrEcrire(t, versionne, &replay.MapCalloutsCatalog{
		SchemaVersion: replay.MapCalloutsSchemaVersion, TitleSlug: title.DefaultSlug,
		Maps: map[string]replay.MapCalloutsEntry{},
		MapsByID: map[string]replay.MapCalloutsEntry{zrMapIDVersionnee: {
			Provenance: replay.CalloutsProvenanceMvar, Zones: []replay.CalloutZone{{EN: "Cave", FR: "Grotte"}},
		}},
	})
	lexique := fmt.Sprintf("string_id;en;fr\n0x%08X;River;Rivière\n", zrSidConnu)
	if err := os.WriteFile(res.MapCalloutsLexiquePath(title.DefaultSlug), []byte(lexique), 0o600); err != nil {
		t.Fatal(err)
	}
	return racine
}

// zrDeposer pose une variante au cache, là où le rattrapage des socles la dépose.
func zrDeposer(t *testing.T, cacheRoot, mapID, contenu string) {
	t.Helper()
	if err := deposerMvar(cacheRoot, mapID, mapcatalog.NomDeLaVariante, []byte(contenu)); err != nil {
		t.Fatal(err)
	}
}

// zrPreparer prépare un rattrapage sur une racine de test.
func zrPreparer(t *testing.T, racine, cacheRoot string, f MvarFetcher, aBlanc bool) *RattrapageZones {
	t.Helper()
	r, err := PreparerRattrapageZones(context.Background(), OptionsRattrapageZones{
		RepoRoot: racine, TitleSlug: title.DefaultSlug, CacheRoot: cacheRoot, Fetcher: f, ABlanc: aBlanc,
	})
	if err != nil {
		t.Fatalf("préparation : %v", err)
	}
	return r
}

// zrGenere relit le catalogue généré.
func zrGenere(t *testing.T, racine string) *replay.MapCalloutsCatalog {
	t.Helper()
	cat, err := mapcatalog.ChargerCatalogueGenere(
		title.NewPathResolver(racine).MapCalloutsOverlayPath(title.DefaultSlug), title.DefaultSlug)
	if err != nil {
		t.Fatal(err)
	}
	return cat
}

// TestZonesCarteDejaCouverteNeFaitAucunAppel — promesse 1.
func TestZonesCarteDejaCouverteNeFaitAucunAppel(t *testing.T) {
	zrExtractionFactice(t)
	racine := zrRacine(t)
	f := &mrFetcherEspion{blob: []byte(zrVarianteAvecZones), base: "map.mvar"}
	r := zrPreparer(t, racine, t.TempDir(), f, false)
	res := r.TraiterCarte(context.Background(), mapcatalog.IdentitesDeCarte{MapID: zrMapIDVersionnee})
	if res.Statut != ZonesDejaCouvertes || res.Origine != mapcatalog.OrigineVersionne || f.appels != 0 {
		t.Fatalf("carte versionnée : statut %q origine %q, %d appel(s) ; attendu déjà couverte, 0 appel",
			res.Statut, res.Origine, f.appels)
	}
	if n := len(zrGenere(t, racine).MapsByID); n != 0 {
		t.Errorf("catalogue généré : %d cartes, attendu 0", n)
	}
}

// TestZonesVarianteAuCacheLueHorsLigne — promesse 2, et la carte ajoutée devient couverte.
func TestZonesVarianteAuCacheLueHorsLigne(t *testing.T) {
	zrExtractionFactice(t)
	racine, cache := zrRacine(t), t.TempDir()
	zrDeposer(t, cache, "bbbb", zrVarianteAvecZones)
	f := &mrFetcherEspion{err: errors.New("le réseau ne doit pas être appelé")}
	res := zrPreparer(t, racine, cache, f, false).TraiterCarte(context.Background(),
		mapcatalog.IdentitesDeCarte{MapID: "bbbb", Noms: []string{"Lattice - Ranked"}})
	if res.Statut != ZonesAjoutees || f.appels != 0 || res.Telechargee {
		t.Fatalf("variante au cache : statut %q, %d appel(s), err %v", res.Statut, f.appels, res.Err)
	}
	e, err := zrGenere(t, racine).LookupByID("bbbb")
	if err != nil || e.Zones[0].FR != "Rivière" {
		t.Fatalf("catalogue généré : %+v / %v", e, err)
	}
	// La passe suivante voit la carte couverte PAR LE CATALOGUE GÉNÉRÉ.
	res = zrPreparer(t, racine, cache, f, false).TraiterCarte(context.Background(), mapcatalog.IdentitesDeCarte{MapID: "bbbb"})
	if res.Statut != ZonesDejaCouvertes || res.Origine != mapcatalog.OrigineGenere {
		t.Errorf("seconde passe : statut %q origine %q", res.Statut, res.Origine)
	}
}

// TestZonesUnAppelParCarteEtDepot — promesse 3 : trois films de la même carte ouvrent UN appel,
// la variante est déposée au cache, et le cycle suivant ne rappelle pas.
func TestZonesUnAppelParCarteEtDepot(t *testing.T) {
	zrExtractionFactice(t)
	racine, cache := zrRacine(t), t.TempDir()
	d := Deps{RepoRoot: racine, TitleSlug: title.DefaultSlug, CacheRoot: cache}
	f := &mrFetcherEspion{blob: []byte(zrVarianteSansZone), base: "map.mvar"}
	ctx := ctxkeys.WithTitleSlug(context.Background(), title.DefaultSlug)
	work := []buildWork{zrTravail("m1", "cccc"), zrTravail("m2", "cccc"), zrTravail("m3", "cccc"), zrTravail("m4", "")}

	rattraperZonesNommees(ctx, d, work, f)
	if f.appels != 1 {
		t.Fatalf("appels = %d, attendu 1 (une fois par carte, jamais par match)", f.appels)
	}
	if _, err := lireMvarEnCache(cache, "cccc"); err != nil {
		t.Fatalf("variante non déposée au cache : %v", err)
	}
	if got := observability.LoadCounterT(title.DefaultSlug, JaugeZonesSansZone); got != 1 {
		t.Errorf("jauge sans_zone = %d, attendu 1", got)
	}
	rattraperZonesNommees(ctx, d, work, f)
	if f.appels != 1 {
		t.Errorf("second cycle : appels = %d, attendu toujours 1 (relecture hors ligne)", f.appels)
	}
}

// TestZonesHorsLigneEtABlancNeTelechargentRien — promesse 4.
func TestZonesHorsLigneEtABlancNeTelechargentRien(t *testing.T) {
	zrExtractionFactice(t)
	racine, cache := zrRacine(t), t.TempDir()
	f := &mrFetcherEspion{blob: []byte(zrVarianteAvecZones), base: "map.mvar"}
	id := mapcatalog.IdentitesDeCarte{MapID: "dddd"}
	if res := zrPreparer(t, racine, cache, nil, false).TraiterCarte(context.Background(), id); res.Statut != ZonesATelecharger {
		t.Errorf("hors ligne : statut %q, attendu à télécharger", res.Statut)
	}
	if res := zrPreparer(t, racine, cache, f, true).TraiterCarte(context.Background(), id); res.Statut != ZonesATelecharger || f.appels != 0 {
		t.Errorf("à blanc : statut %q, %d appel(s) ; attendu à télécharger, 0 appel", res.Statut, f.appels)
	}
	zrDeposer(t, cache, "dddd", zrVarianteAvecZones)
	if res := zrPreparer(t, racine, cache, f, true).TraiterCarte(context.Background(), id); res.Statut != ZonesAjoutables {
		t.Errorf("à blanc, variante au cache : statut %q, attendu ajoutable", res.Statut)
	}
	if n := len(zrGenere(t, racine).MapsByID); n != 0 {
		t.Errorf("à blanc : %d carte(s) écrite(s) au catalogue généré", n)
	}
}

// TestZonesVerdictsEtEchecs — sans libellé, variante illisible, téléchargement refusé : chacun
// a son statut, aucun n'écrit.
func TestZonesVerdictsEtEchecs(t *testing.T) {
	zrExtractionFactice(t)
	racine, cache := zrRacine(t), t.TempDir()
	zrDeposer(t, cache, "eeee", zrVarianteMuette)
	zrDeposer(t, cache, "ffff", "octets quelconques")
	r := zrPreparer(t, racine, cache, &mrFetcherEspion{err: errors.New("403")}, false)
	cas := map[string]StatutZones{"eeee": ZonesSansLibelle, "ffff": ZonesVarianteIllisible, "gggg": ZonesEchecTelechargement}
	for mapID, attendu := range cas {
		res := r.TraiterCarte(context.Background(), mapcatalog.IdentitesDeCarte{MapID: mapID})
		if res.Statut != attendu {
			t.Errorf("%s : statut %q, attendu %q (err %v)", mapID, res.Statut, attendu, res.Err)
		}
	}
	if res := r.TraiterCarte(context.Background(), mapcatalog.IdentitesDeCarte{MapID: "eeee"}); res.Couverture.SansLibelle() != 1 {
		t.Errorf("sans libellé : %d string_id non résolu(s), attendu 1", res.Couverture.SansLibelle())
	}
	if n := len(zrGenere(t, racine).MapsByID); n != 0 {
		t.Errorf("%d carte(s) écrite(s) malgré des verdicts négatifs", n)
	}
}

// TestZonesPorteEtLexiqueManquants — promesse 5 : sans configuration lisible, ou sans lexique,
// le rattrapage compte un échec et ne fait rien d'autre.
func TestZonesPorteEtLexiqueManquants(t *testing.T) {
	ctx := ctxkeys.WithTitleSlug(context.Background(), title.DefaultSlug)
	f := &mrFetcherEspion{blob: []byte(zrVarianteAvecZones), base: "map.mvar"}
	rattraperZonesNommees(ctx, Deps{RepoRoot: t.TempDir(), TitleSlug: title.DefaultSlug, CacheRoot: t.TempDir()},
		[]buildWork{zrTravail("m1", "hhhh")}, f)
	if f.appels != 0 || observability.LoadCounterT(title.DefaultSlug, JaugeZonesEchecs) != 1 {
		t.Errorf("capabilities illisibles : %d appel(s), jauge d'échecs %d", f.appels,
			observability.LoadCounterT(title.DefaultSlug, JaugeZonesEchecs))
	}
	racine := zrRacine(t)
	if err := os.Remove(title.NewPathResolver(racine).MapCalloutsLexiquePath(title.DefaultSlug)); err != nil {
		t.Fatal(err)
	}
	if _, err := PreparerRattrapageZones(ctx, OptionsRattrapageZones{RepoRoot: racine, TitleSlug: title.DefaultSlug}); err == nil {
		t.Error("lexique absent : la préparation doit échouer — sans lui aucune zone n'est nommée")
	}
}

// TestZonesPorteSurLesTOMLLivres — halo_infinite déclare `map.forge_callouts`, halo_5 non.
func TestZonesPorteSurLesTOMLLivres(t *testing.T) {
	root := racineDepot(t)
	for slug, attendu := range map[string]bool{"halo_infinite": true, "halo_5": false} {
		armee, incident := porteCapability(context.Background(), Deps{RepoRoot: root, TitleSlug: slug},
			"map.forge_callouts", "zones nommees Forge", nil)
		if armee != attendu || incident {
			t.Errorf("%s : armée %v (incident %v), attendu %v", slug, armee, incident, attendu)
		}
	}
}

// zrTravail fabrique un match du lot portant une carte.
func zrTravail(matchID, mapID string) buildWork {
	return buildWork{matchID: matchID, facts: port.MatchFacts{MapID: mapID}}
}
