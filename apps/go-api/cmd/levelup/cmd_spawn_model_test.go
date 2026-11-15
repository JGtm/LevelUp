package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/tactical"
	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/testutil"
)

func TestEvenementsDuMatch_DeuxInstantanes(t *testing.T) {
	pistes := []tactical.Piste{
		{XUID: "a", Equipe: 0, Points: []tactical.PointPiste{{T: 0, X: 1}, {T: 100, X: 2}}},
		{XUID: "a", Equipe: 0, Points: []tactical.PointPiste{{T: 150, X: 30}, {T: 300, X: 31}}},
		{XUID: "e", Equipe: 1, Points: []tactical.PointPiste{{T: 0, X: 50}, {T: 400, X: 60}}},
		{XUID: "m", Equipe: 0, Points: []tactical.PointPiste{{T: 0, X: 10}, {T: 400, X: 10}}},
		{XUID: "sans", Equipe: tactical.EquipeInconnue, Points: []tactical.PointPiste{{T: 0, X: 5}, {T: 100, X: 5}}},
		{XUID: "sans", Equipe: tactical.EquipeInconnue, Points: []tactical.PointPiste{{T: 150, X: 6}, {T: 300, X: 6}}},
	}
	e := tactical.EntreeOccupation{MatchID: "m1", IntervalleFrameMs: 100, Pistes: pistes}
	jeu := &jeuReapparitions{}
	jeu.bilan.parFamille = map[string]int{}
	jeu.evenementsDuMatch(e, infoMatch{matchID: "m1", carteID: "c", famille: "assassin"})

	if len(jeu.evts) != 1 {
		t.Fatalf("une reapparition mesurable attendue (celle de \"sans\" n'a pas d'equipe), obtenu %d", len(jeu.evts))
	}
	if jeu.bilan.reapRepli != 2 || jeu.bilan.sansEquipe != 1 {
		t.Errorf("bilan = %+v", jeu.bilan)
	}
	ev := jeu.evts[0]
	// A la reapparition (frame 150) : l'adversaire "e" tient sa derniere position connue (x = 50),
	// le coequipier "m" est a 10 ; "sans" (equipe inconnue) n'est dans aucun des deux camps.
	if len(ev.reap.Ennemis) != 1 || ev.reap.Ennemis[0].X != 50 || len(ev.reap.Equipiers) != 1 {
		t.Errorf("instant de la reapparition = %+v", ev.reap)
	}
	// A la mort (frame 100) : mort de "a" ; les autres sont les memes.
	if len(ev.mort.Ennemis) != 1 || ev.mort.Mort.X != 2 || ev.mort.Reel.X != 30 {
		t.Errorf("instant de la mort = %+v", ev.mort)
	}
}

func TestPoidsArrondis(t *testing.T) {
	w := poidsArrondis(map[string]float64{"a": 2.001339496453945, "b": -0.65318830})
	if w["a"] != 2.0013 || w["b"] != -0.6532 {
		t.Errorf("poids arrondis au dix-millieme : %v", w)
	}
	if arrondi1(62.76) != 62.8 {
		t.Errorf("arrondi1(62.76) = %v", arrondi1(62.76))
	}
}

func TestVerifierAjustementComplet(t *testing.T) {
	cas := []struct {
		nom     string
		o       spawnModelOptions
		refuse  bool
		attendu string
	}{
		{"parc entier", spawnModelOptions{}, false, ""},
		{"une famille sans sortie", spawnModelOptions{famille: "assassin"}, true, "--sortie"},
		{"un extrait sans sortie", spawnModelOptions{limite: 50}, true, "--sortie"},
		{"une famille avec sortie", spawnModelOptions{famille: "assassin", sortie: "x.json"}, false, ""},
		{"un extrait avec sortie", spawnModelOptions{limite: 50, sortie: "x.json"}, false, ""},
	}
	for _, c := range cas {
		err := verifierAjustementComplet(c.o)
		if (err != nil) != c.refuse {
			t.Errorf("%s : erreur %v, refus attendu %v", c.nom, err, c.refuse)
		}
		if err != nil && !strings.Contains(err.Error(), c.attendu) {
			t.Errorf("%s : le message doit dire comment ecrire ailleurs : %v", c.nom, err)
		}
	}
}

// depotDeTest bâtit un dépôt temporaire qui porte les mappings versionnés d'un titre (tous, sauf
// ceux de `sauf`), et rend sa configuration.
func depotDeTest(t *testing.T, slug string, sauf ...string) *config.AppConfig {
	t.Helper()
	racine, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot: %v", err)
	}
	dest := t.TempDir()
	src := filepath.Join(racine, "config", "titles", slug, "mappings")
	entrees, err := os.ReadDir(src)
	if err != nil {
		t.Fatalf("lecture de %s: %v", src, err)
	}
	cible := filepath.Join(dest, "config", "titles", slug, "mappings")
	if err := os.MkdirAll(cible, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, e := range entrees {
		if e.IsDir() || slices.Contains(sauf, e.Name()) {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(src, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(cible, e.Name()), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return &config.AppConfig{RepoRoot: dest}
}

// `--variantes` liste les variantes puis s'arrête : une sortie normale pour `fit` comme pour
// `evaluate`, jamais une erreur.
func TestSpawnModel_VariantesSansErreurPourFitEtEvaluate(t *testing.T) {
	cfg := depotDeTest(t, "halo_infinite")
	creerBaseDeTest(t, titlePkg.NewPathResolver(cfg.RepoRoot).SharedDBPath("halo_infinite"),
		`CREATE TABLE match_registry (match_id VARCHAR, map_id VARCHAR, map_name VARCHAR,
			game_variant_name VARCHAR, mode_category VARCHAR, is_firefight BOOLEAN)`)
	for _, sous := range []string{"fit", "evaluate"} {
		if err := runSpawnModel(cfg, []string{sous, "--variantes"}); err != nil {
			t.Errorf("spawn-model %s --variantes : %v", sous, err)
		}
	}
	// Un ajustement partiel est refusé AVANT toute lecture, avec ou sans variantes.
	if err := runSpawnModel(cfg, []string{"fit", "--famille", "assassin"}); err == nil {
		t.Error("fit --famille sans --sortie doit etre refuse")
	}
}

// Un titre qui ne déclare pas la prévision (Halo 5, titre synthétique) s'arrête proprement : aucune
// lecture, aucune écriture, pas d'erreur.
func TestSpawnModel_TitreSansCapacite(t *testing.T) {
	for _, slug := range []string{"halo_5", "synthetic_title_b"} {
		cfg := depotDeTest(t, slug)
		for _, sous := range []string{"fit", "evaluate"} {
			if err := runSpawnModel(cfg, []string{sous, "--title", slug}); err != nil {
				t.Errorf("%s %s : %v", sous, slug, err)
			}
		}
		if _, err := os.Stat(titlePkg.NewPathResolver(cfg.RepoRoot).SpawnModelPath(slug)); err == nil {
			t.Errorf("%s : aucun fichier de poids ne doit etre ecrit", slug)
		}
	}
}

// Un titre qui déclare la prévision sans déclarer de famille de modes est une erreur de
// configuration dite en clair, pas une mesure sur un jeu de modes sans nom.
func TestSpawnModel_TitreSansFamilleDeclaree(t *testing.T) {
	cfg := depotDeTest(t, "halo_infinite", fichierFamilles)
	err := runSpawnModel(cfg, []string{"evaluate"})
	if err == nil || !strings.Contains(err.Error(), fichierFamilles) {
		t.Fatalf("erreur attendue sur %s absent, obtenu %v", fichierFamilles, err)
	}
}

func TestDefinitionDesPoids_DitLeSeuilEtLePerimetre(t *testing.T) {
	jeu := &jeuReapparitions{bilan: bilanLecture{reapFilm: 3, reapRepli: 7}}
	d := definitionDesPoids(jeu, spawnModelOptions{matchsMinCarte: 3})
	for _, attendu := range []string{"toutes les familles", "au moins 3 matchs", "avant filtres", "AUTRES matchs", "retenues"} {
		if !strings.Contains(d, attendu) {
			t.Errorf("la definition ne dit pas %q : %s", attendu, d)
		}
	}
	if p := definitionDesPoids(jeu, spawnModelOptions{famille: "assassin", sortie: "x"}); !strings.Contains(p, "PARTIEL") {
		t.Errorf("un ajustement partiel le dit : %s", p)
	}
}

// Les effectifs du fichier comptent les reapparitions RETENUES : une carte sous le seuil de matchs
// n'est comptee ni en cartes, ni en matchs, ni en reapparitions.
func TestPoidsDeGroupe_EffectifsSurLeJeuAjuste(t *testing.T) {
	jeu := &jeuReapparitions{}
	pts := []tactical.Point3{{X: 0}, {X: 20}, {X: 40}, {X: 60}, {X: 80}, {X: 100}}
	ajoute := func(carte string, matchs int) {
		for m := 0; m < matchs; m++ {
			id := fmt.Sprintf("%s-m%d", carte, m)
			for rep := 0; rep < 3; rep++ {
				for i, p := range pts {
					e := tactical.EvenementReapparition{MatchID: id, CarteID: carte, Reel: p, Mort: p,
						Ennemis: []tactical.Point3{{X: float64(-10 - i)}}}
					jeu.evts = append(jeu.evts, evenementDouble{famille: "assassin", reap: e, mort: e})
				}
			}
		}
	}
	ajoute("grande", 30) // 540 reapparitions retenues
	ajoute("petite", 2)  // sous le seuil de 3 matchs : ecartee
	p, ok := poidsDeGroupe(context.Background(), jeu, "assassin", map[string]bool{"assassin": true},
		tactical.ToutesLesVariables, spawnModelOptions{matchsMinCarte: 3})
	if !ok {
		t.Fatal("effectif suffisant : des poids propres sont attendus")
	}
	if p.Reapparitions != 540 || p.Cartes != 1 || p.Matchs != 30 {
		t.Errorf("effectifs = %d reapparitions, %d cartes, %d matchs ; attendu 540, 1, 30", p.Reapparitions, p.Cartes, p.Matchs)
	}
	if p.Scores.Rayon3Hasard <= p.Scores.Rayon1Hasard {
		t.Errorf("le hasard sur trois tirages (%v) depasse celui d'un tirage (%v)", p.Scores.Rayon3Hasard, p.Scores.Rayon1Hasard)
	}
}

// Le rapport M-2 / M-3 compte les matchs sur les echantillons RETENUS, comme poidsDeGroupe : une
// carte sous le seuil n'apporte ni carte, ni reapparition, ni match.
func TestEvaluerEchantillons_MatchsSurLeJeuRetenu(t *testing.T) {
	var evts []tactical.EvenementReapparition
	pts := []tactical.Point3{{X: 0}, {X: 20}, {X: 40}, {X: 60}}
	ajoute := func(carte string, matchs int) {
		for m := 0; m < matchs; m++ {
			for i, p := range pts {
				evts = append(evts, tactical.EvenementReapparition{MatchID: fmt.Sprintf("%s-m%d", carte, m),
					CarteID: carte, Reel: p, Mort: p, Ennemis: []tactical.Point3{{X: float64(-10 - i)}}})
			}
		}
	}
	ajoute("grande", 6)
	ajoute("petite", 2) // sous le seuil de 3 matchs : ecartee
	reglages := tactical.ReglagesEchantillons{MatchsMinCarte: 3}
	r := evaluerEchantillons(tactical.ConstruireEchantillons(evts, tactical.CandidatsParCarte(evts), reglages), false)
	if r.matchs != 6 || r.cartes != 1 || r.n != 24 {
		t.Errorf("effectifs = %d matchs, %d cartes, %d reapparitions ; attendu 6, 1, 24", r.matchs, r.cartes, r.n)
	}
}
