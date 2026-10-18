package main

// Témoins des passes locales de `levelup backfill` sur un titre choisi (--title) : sélection des
// joueurs DU titre, porte par capability, bases du titre (jamais celles du titre par défaut).
// Racines temporaires : aucune base du dépôt n'est ouverte.

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	go_sync "levelup/go-api/internal/sync"
	"levelup/go-api/internal/testutil"
)

const titreHalo5 = "halo_5"

// racineBackfillTitre : une racine temporaire avec les manifestes des titres et un
// db_profiles v3 — JGtm sur les deux titres, Solo sur Infinite seul, Auth en profil réservé à
// l'authentification sur Halo 5. Les bases joueur de JGtm (Halo 5) et Solo (Infinite) existent.
func racineBackfillTitre(t *testing.T) (*config.AppConfig, *titlePkg.PathResolver) {
	t.Helper()
	depot, err := testutil.RepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	racine := t.TempDir()
	if err := os.CopyFS(filepath.Join(racine, "config", "titles"), os.DirFS(filepath.Join(depot, "config", "titles"))); err != nil {
		t.Fatal(err)
	}
	profils := `{"version":"3.0","admin":"JGtm","profiles":{
		"halo_infinite":{"JGtm":{"xuid":"1001"},"Solo":{"xuid":"1002"}},
		"halo_5":{"JGtm":{"xuid":"1001"},"Auth":{"xuid":"1003","auth_only":true}}}}`
	chemin := filepath.Join(racine, "db_profiles.json")
	ecrireFichier(t, chemin, profils)
	cfg := &config.AppConfig{RepoRoot: racine, DBProfilesPath: chemin}
	pr := titlePkg.NewPathResolver(racine)
	for _, c := range []struct{ titre, joueur string }{{titreHalo5, "JGtm"}, {titlePkg.DefaultSlug, "Solo"}} {
		db, err := go_sync.OpenPlayerDB(pr.PlayerDBPath(c.titre, c.joueur))
		if err != nil {
			t.Fatalf("base joueur %s/%s: %v", c.titre, c.joueur, err)
		}
		_ = db.Close()
	}
	return cfg, pr
}

// passeEspion : une passe qui n'écrit rien et note les joueurs que le pilote lui livre.
func passeEspion(vus *[]string) titledPass {
	return titledPass{
		name: "espion", unit: "updated",
		run: func(_ context.Context, e *go_sync.SyncEngine, _ bool) (int, error) {
			*vus = append(*vus, e.GamertagForTest())
			return 1, nil
		},
	}
}

// TestRunTitledPass_SelectionDuTitre : --all prend les profils DU titre demandé (pas ceux d'un
// autre titre, pas les profils réservés à l'authentification) ; un seul joueur absent du titre
// est une erreur, pas une création de base.
func TestRunTitledPass_SelectionDuTitre(t *testing.T) {
	cfg, _ := racineBackfillTitre(t)
	for _, c := range []struct {
		titre string
		vus   []string
	}{
		{titreHalo5, []string{"JGtm"}},
		{titlePkg.DefaultSlug, []string{"JGtm", "Solo"}},
		{"", []string{"JGtm", "Solo"}},
	} {
		var vus []string
		sortie := sortieDe(t, func() {
			if err := runTitledPass(context.Background(), cfg, backfillTarget{titleSlug: c.titre, all: true}, false, passeEspion(&vus)); err != nil {
				t.Errorf("titre %q : %v", c.titre, err)
			}
		})
		// JGtm n'a de base joueur que sur Halo 5 ; Solo que sur Infinite : le reste est sauté.
		if c.titre == titreHalo5 && strings.Join(vus, ",") != "JGtm" {
			t.Errorf("halo_5 : joueurs livrés %v, attendu [JGtm]", vus)
		}
		if c.titre != titreHalo5 && strings.Join(vus, ",") != "Solo" {
			t.Errorf("titre par défaut : joueurs livrés %v, attendu [Solo] (JGtm sans base Infinite, sauté)\n%s", vus, sortie)
		}
	}

	err := runTitledPass(context.Background(), cfg, backfillTarget{titleSlug: titreHalo5, gamertag: "Solo"}, false, passeEspion(new([]string)))
	if err == nil || !strings.Contains(err.Error(), "pas de base joueur") {
		t.Errorf("Solo n'est pas suivi sur halo_5 : erreur attendue, got %v", err)
	}
}

// TestRunTitledPass_CitationsNonSupporteesSurHalo5 : citations, citations composites et recalcul
// total des citations répondent « non supporté » sur Halo 5 (citations.engine absente) et
// n'exécutent rien : aucune base du titre n'est migrée ni ouverte.
func TestRunTitledPass_CitationsNonSupporteesSurHalo5(t *testing.T) {
	cfg, pr := racineBackfillTitre(t)
	for _, pass := range []titledPass{passCitations, passCompositeOnly, passCitationsRecomputeAll} {
		t.Run(pass.name, func(t *testing.T) {
			sortie := sortieDe(t, func() {
				if err := runTitledPass(context.Background(), cfg, backfillTarget{titleSlug: titreHalo5, all: true}, true, pass); err != nil {
					t.Errorf("une passe non supportée n'est pas un échec : %v", err)
				}
			})
			if !strings.Contains(sortie, "non supporte pour le titre halo_5") || strings.Contains(sortie, " OK:") {
				t.Errorf("sortie inattendue :\n%s", sortie)
			}
			for _, base := range []string{pr.SharedDBPath(titreHalo5), pr.MetadataDBPath(titreHalo5)} {
				if _, err := os.Stat(base); !os.IsNotExist(err) {
					t.Errorf("la passe a ouvert %s alors qu'elle n'est pas supportée (stat: %v)", base, err)
				}
			}
		})
	}
}

// TestRunTitledPass_PassesRealesSurLesBasesDeHalo5 : perf, modèle des assistances, LUSR v2 et
// engagement travaillent sur les bases de Halo 5 — la base partagée de Halo 5 apparaît, aucune
// base partagée du titre par défaut n'est créée (c'était la destination de toute passe avant
// --title).
func TestRunTitledPass_PassesRealesSurLesBasesDeHalo5(t *testing.T) {
	cfg, pr := racineBackfillTitre(t)
	migrerLeSharedDeHalo5(t, pr)
	for _, pass := range []titledPass{passPerf, passAssistsModel, passLUSR, passEngagement} {
		t.Run(pass.name, func(t *testing.T) {
			sortie := sortieDe(t, func() {
				if err := runTitledPass(context.Background(), cfg, backfillTarget{titleSlug: titreHalo5, all: true}, true, pass); err != nil {
					t.Errorf("%s sur halo_5 : %v", pass.name, err)
				}
			})
			if !strings.Contains(sortie, "backfill "+pass.name+" OK: title=halo_5 gamertag=JGtm") {
				t.Errorf("la passe devait réussir pour JGtm sur halo_5 :\n%s", sortie)
			}
			if _, err := os.Stat(pr.SharedDBPath(titreHalo5)); err != nil {
				t.Errorf("la base partagée de halo_5 devait être ouverte: %v", err)
			}
			if _, err := os.Stat(pr.SharedDBPath(titlePkg.DefaultSlug)); !os.IsNotExist(err) {
				t.Errorf("la base partagée du titre par défaut ne devait pas être créée (stat: %v)", err)
			}
		})
	}
}

// TestRunBackfill_CSRRefuseUnTitreSansLaCapability : --csr / --shared-csr se règlent sur la
// capability « match.skill.snapshot » (CSR par match) du titre, pas sur son slug : Halo 5 ne la
// déclare pas, le refus la nomme ; --title ne redirige rien en silence vers le titre par défaut.
func TestRunBackfill_CSRRefuseUnTitreSansLaCapability(t *testing.T) {
	cfg, _ := racineBackfillTitre(t)
	for _, flag := range []string{"--csr", "--shared-csr"} {
		err := runBackfill(cfg, []string{"--all", flag, "--title", titreHalo5})
		if err == nil || !strings.Contains(err.Error(), string(games.CapMatchSkillSnapshot)) {
			t.Errorf("%s --title halo_5 : refus nommant la capability attendu, got %v", flag, err)
		}
	}
}

// TestRequireCSRCapability : le titre qui déclare la capability passe (Infinite : degraded), celui
// qui ne la déclare pas ou n'a pas de mappings est refusé.
func TestRequireCSRCapability(t *testing.T) {
	cfg, _ := racineBackfillTitre(t)
	if err := requireCSRCapability(cfg, titlePkg.DefaultSlug); err != nil {
		t.Errorf("%s déclare la capability : accepté attendu, got %v", titlePkg.DefaultSlug, err)
	}
	if err := requireCSRCapability(cfg, titreHalo5); err == nil {
		t.Errorf("%s ne la déclare pas : refus attendu", titreHalo5)
	}
	if err := requireCSRCapability(cfg, "titre_inconnu"); err == nil {
		t.Error("titre sans mappings : refus attendu")
	}
}

// TestBackfillPasses_AucunTitreParDefautEnDur : garde-rail. Les passes ne codent plus le titre
// par défaut (l'ancien littéral rendait --title sans effet sur Halo 5) : seuls le repli du titre
// vide du pilote et la valeur par défaut du drapeau le nomment.
func TestBackfillPasses_AucunTitreParDefautEnDur(t *testing.T) {
	for fichier, max := range map[string]int{"cmd_backfill_titled.go": 1, "cmd_backfill.go": 1} {
		src, err := os.ReadFile(fichier)
		if err != nil {
			t.Fatal(err)
		}
		if n := strings.Count(string(src), "titlePkg.DefaultSlug"); n > max {
			t.Errorf("%s nomme titlePkg.DefaultSlug %d fois (max %d) : une passe doit viser le titre demandé", fichier, n, max)
		}
	}
}

// migrerLeSharedDeHalo5 crée la base partagée de Halo 5 migrée comme au boot du serveur (jeu
// complet des étapes du titre) : les passes la trouvent en l'état, comme en production.
func migrerLeSharedDeHalo5(t *testing.T, pr *titlePkg.PathResolver) {
	t.Helper()
	chemin := pr.SharedDBPath(titreHalo5)
	if err := os.MkdirAll(filepath.Dir(chemin), 0o755); err != nil {
		t.Fatal(err)
	}
	_ = migration.All()
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	db, err := duckdbpkg.OpenReadWrite(chemin)
	if err != nil {
		t.Fatalf("ouverture du shared halo_5: %v", err)
	}
	defer db.Close()
	if err := migration.RunForTitleDB(db.SQLDb(), titreHalo5, migration.TargetShared); err != nil {
		t.Fatalf("migrations shared halo_5: %v", err)
	}
}
