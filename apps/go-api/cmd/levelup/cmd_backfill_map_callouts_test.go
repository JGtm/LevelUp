package main

// Témoins de `levelup backfill-map-callouts` sur des racines de test : AUCUNE base ouverte
// (--carte), AUCUN réseau (--hors-ligne), la chaîne réelle de l'extraction sur une variante
// versionnée.

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/mapcatalog"
	"levelup/go-api/internal/sync/replayartifacts"
	"levelup/go-api/internal/testutil"
)

// sortieDe rend ce que `f` écrit sur la sortie standard.
func sortieDe(t *testing.T, f func()) string {
	t.Helper()
	ancienne := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe : %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = ancienne }()
	lu := make(chan []byte, 1)
	go func() {
		out, _ := io.ReadAll(r)
		lu <- out
	}()
	f()
	_ = w.Close()
	return string(<-lu)
}

// racineZonesCLI monte une racine : configuration LIVRÉE du titre, catalogue versionné d'une
// carte Forge, lexique LIVRÉ.
func racineZonesCLI(t *testing.T, versionnee string) string {
	t.Helper()
	depot, err := testutil.RepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	racine := t.TempDir()
	dst := filepath.Join(racine, "config", "titles", titlePkg.DefaultSlug)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.CopyFS(dst, os.DirFS(filepath.Join(depot, "config", "titles", titlePkg.DefaultSlug))); err != nil {
		t.Fatal(err)
	}
	res, ref := titlePkg.NewPathResolver(racine), titlePkg.NewPathResolver(depot)
	ecrireFichier(t, res.MapCalloutsPath(titlePkg.DefaultSlug), `{"schema_version":1,"maps":{},"maps_by_id":{"`+
		versionnee+`":{"module":"","provenance":"mvar","zones":[{"volume_index":1,"en":"Cave","fr":"Grotte"}]}}}`)
	lexique, err := os.ReadFile(ref.MapCalloutsLexiquePath(titlePkg.DefaultSlug))
	if err != nil {
		t.Fatal(err)
	}
	ecrireFichier(t, res.MapCalloutsLexiquePath(titlePkg.DefaultSlug), string(lexique))
	return racine
}

func ecrireFichier(t *testing.T, chemin, contenu string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(chemin), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(chemin, []byte(contenu), 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestBackfillMapCalloutsHorsLigne — trois cartes, trois issues, sans base ni réseau : la carte
// versionnée est sautée, la variante au cache (fichier-lien d'un canevas, réel) est lue et ne
// pose aucune zone, la carte sans variante reste à télécharger. Rien n'est écrit.
func TestBackfillMapCalloutsHorsLigne(t *testing.T) {
	const versionnee, auCache, absente = "aaaa", "bbbb", "cccc"
	racine := racineZonesCLI(t, versionnee)
	depot, _ := testutil.RepoRoot()
	blob, err := os.ReadFile(filepath.Join(depot, ".ai", "V7.5", "dumps", "mapvar", "vagabond_fo08_wetland.mvar"))
	if err != nil {
		t.Fatal(err)
	}
	cache := t.TempDir()
	ecrireFichier(t, filepath.Join(cache, "mvar", auCache, mapcatalog.NomDeLaVariante), string(blob))

	var errPasse error
	sortie := sortieDe(t, func() {
		errPasse = runBackfillMapCallouts(&config.AppConfig{RepoRoot: racine}, []string{
			"--hors-ligne", "--cache-dir", cache, "--carte", versionnee + "," + auCache + "," + absente,
		})
	})
	if errPasse != nil {
		t.Fatalf("passe hors ligne : %v\n%s", errPasse, sortie)
	}
	for _, attendu := range []string{
		string(replayartifacts.ZonesDejaCouvertes) + " ", versionnee + ` "-" matchs=0 par=versionne`,
		string(replayartifacts.ZonesAbsentes) + " ", string(replayartifacts.ZonesATelecharger) + " ",
		"bilan (hors ligne) : 3 carte(s)",
	} {
		if !strings.Contains(sortie, attendu) {
			t.Errorf("sortie sans %q :\n%s", attendu, sortie)
		}
	}
	overlay := titlePkg.NewPathResolver(racine).MapCalloutsOverlayPath(titlePkg.DefaultSlug)
	if _, err := os.Stat(overlay); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("catalogue généré écrit alors qu'aucune carte n'était publiable : %v", err)
	}
}

// TestBackfillMapCalloutsTitreSansCapability — un titre qui ne déclare pas `map.forge_callouts`
// fait une passe VIDE, sans ouvrir de base ni demander de jeton.
func TestBackfillMapCalloutsTitreSansCapability(t *testing.T) {
	depot, err := testutil.RepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	var errPasse error
	sortie := sortieDe(t, func() {
		errPasse = runBackfillMapCallouts(&config.AppConfig{RepoRoot: depot}, []string{"--title", "halo_5"})
	})
	if errPasse != nil || !strings.Contains(sortie, "capability map.forge_callouts absente") {
		t.Fatalf("halo_5 : err %v, sortie %q", errPasse, sortie)
	}
}

// TestLigneDeCarteDitTout — statut, identité, mesure des libellés et cause d'échec sur une ligne.
func TestLigneDeCarteDitTout(t *testing.T) {
	ligne := ligneDeCarte(replayartifacts.CarteJouee{MapID: "m", Noms: []string{"Lattice - Ranked"}, Matchs: 148},
		replayartifacts.ResultatZones{
			Statut: replayartifacts.ZonesEchecEcriture, Telechargee: true, Err: errors.New("disque plein"),
			Couverture: mapcatalog.CouvertureLibelles{Zones: 12, Nommees: 11, StringIDs: map[uint32]bool{1: true, 2: false}},
		})
	for _, attendu := range []string{"echec_ecriture", `"Lattice - Ranked"`, "matchs=148", "zones=12 nommees=11",
		"string_id_sans_libelle=1", "(telechargee)", "erreur=disque plein"} {
		if !strings.Contains(ligne, attendu) {
			t.Errorf("ligne sans %q : %s", attendu, ligne)
		}
	}
}

// TestAideCiteBackfillMapCallouts — la commande est dans l'aide, avec son catalogue.
func TestAideCiteBackfillMapCallouts(t *testing.T) {
	bloc, ok := entreesBackfill(aideCapturee(t))["backfill-map-callouts"]
	if !ok || !strings.Contains(bloc, "generated/map_callouts.json") || !strings.HasSuffix(strings.TrimSpace(bloc), ")") {
		t.Fatalf("aide de backfill-map-callouts :\n%s", bloc)
	}
}
