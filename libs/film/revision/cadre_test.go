package revision

// cadre_test.go — LE CADRE HISTORIQUE DU HACHAGE (decision D2-F du plan
// `.ai/V8/PLAN_DECOUPLAGE_BRIQUES_D2_2026-10-28.md`) : la brique hachee depuis le module de demain
// rend les memes empreintes que depuis celui d aujourd hui.
//
// Ces tests sont INTERNES au paquet : ils posent le chemin courant de la brique a la valeur du
// module de demain (`Module.brique`, parametre `brique` de [jetonsDe]), ce que la production ne
// fait jamais — elle lit [cheminModuleCourant].

import (
	"os"
	"path"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// briqueFuture : le chemin d import de la brique une fois devenue le module `libs/film`.
const briqueFuture = "github.com/JGtm/LevelUp/libs/film"

// sourceImportant rend une source qui importe deux paquets de la brique sous `prefixe`, plus un
// paquet hors de la brique, et qui cite la racine de la brique dans une chaine brute.
func sourceImportant(prefixe string) string {
	return "package g\n\nimport (\n" +
		"\t_ \"" + prefixe + "/internal/grammar\"\n" +
		"\tsrc \"" + prefixe + "/internal/source\"\n" +
		"\t_ \"github.com/JGtm/LevelUp/libs/gamefiles/mapvar\"\n" +
		")\n\nconst Racine = `" + prefixe + "`\n\nvar _ = src.S\n"
}

func TestCadreHistoriqueDesImports(t *testing.T) {
	t.Run("une source ecrite sous le module de demain hache comme sous celui d aujourd hui", casSourceDeDemain)
	t.Run("un litteral hors du chemin de la brique est hache tel quel", casLitteralHorsDeLaBrique)
	t.Run("un dossier de la brique se nomme par son dossier historique", casDossierHistorique)
}

// casSourceDeDemain : les chaines d import ecrites sous le module de demain rendent le flux de
// jetons du cadre historique.
func casSourceDeDemain(t *testing.T) {
	hier, _, err := jetonsDe("g.go", sourceImportant(cadreHistorique), cadreHistorique)
	if err != nil {
		t.Fatalf("jetons sous le cadre historique : %v", err)
	}
	demain, _, err := jetonsDe("g.go", sourceImportant(briqueFuture), briqueFuture)
	if err != nil {
		t.Fatalf("jetons sous le module de demain : %v", err)
	}
	if demain != hier {
		t.Fatalf("LE FLUX DE JETONS DEPEND DU CHEMIN DU MODULE : la brique deplacee dans "+
			"`libs/film` changerait les cinq empreintes sans qu un octet de grammaire ait bouge.\n"+
			"  hier   :\n%s\n  demain :\n%s", hier, demain)
	}
	if !strings.Contains(hier, `"`+cadreHistorique+`/internal/grammar"`) {
		t.Errorf("le flux ne porte pas la chaine d import historique :\n%s", hier)
	}
}

// casLitteralHorsDeLaBrique : seul un litteral qui commence par le chemin de la brique, a la
// frontiere d un chemin, change de prefixe.
func casLitteralHorsDeLaBrique(t *testing.T) {
	for _, lit := range []string{
		`"github.com/JGtm/LevelUp/libs/gamefiles/mapvar"`,
		`"github.com/JGtm/LevelUp/libs/filmx"`,
		`"github.com/JGtm/LevelUp/libs/film_ancien/x"`,
		`"levelup/go-api/internal/domain/highlightevent"`,
		`"x github.com/JGtm/LevelUp/libs/film"`,
		`"`,
		`'a'`,
	} {
		if got := litteralDansLeCadre(lit, briqueFuture); got != lit {
			t.Errorf("%s rendu %s : seul un litteral qui COMMENCE par le chemin de la brique, a "+
				"la frontiere d un chemin, change de prefixe", lit, got)
		}
	}
	for lit, attendu := range map[string]string{
		`"` + briqueFuture + `"`:          `"` + cadreHistorique + `"`,
		`"` + briqueFuture + `/types"`:    `"` + cadreHistorique + `/types"`,
		"`" + briqueFuture + "/replay`":   "`" + cadreHistorique + "/replay`",
		`"` + cadreHistorique + `/types"`: `"` + cadreHistorique + `/types"`,
	} {
		brique := briqueFuture
		if strings.Contains(lit, cadreHistorique) {
			brique = cadreHistorique
		}
		if got := litteralDansLeCadre(lit, brique); got != attendu {
			t.Errorf("%s (brique %s) rendu %s, attendu %s", lit, brique, got, attendu)
		}
	}
}

// casDossierHistorique : la traduction des dossiers, dans les deux sens, sous les trois sortes de
// module (la brique, son ancien hote, une fixture qui ne la porte pas).
func casDossierHistorique(t *testing.T) {
	demain := Module{Racine: "R", Chemin: briqueFuture, brique: briqueFuture}
	hier := Module{Racine: "R", Chemin: moduleHistorique, brique: cadreHistorique}
	fixture := Module{Racine: "R", Chemin: "exemple.test/m", brique: briqueFuture}
	for _, cas := range []struct {
		m                Module
		courant, nomHist string
	}{
		{demain, "internal/source", "internal/games/halo_infinite/film/internal/source"},
		{demain, "types", "internal/games/halo_infinite/film/types"},
		{hier, "internal/games/halo_infinite/film/internal/source", "internal/games/halo_infinite/film/internal/source"},
		{hier, "internal/domain/highlightevent", "internal/domain/highlightevent"},
		{hier, "internal/games/halo_infinite/filmprofile", "internal/games/halo_infinite/filmprofile"},
		{fixture, "couches/a", "couches/a"},
	} {
		if got := cas.m.historique(cas.courant); got != cas.nomHist {
			t.Errorf("module %s : historique(%s) = %s, attendu %s", cas.m.Chemin, cas.courant, got, cas.nomHist)
		}
		if got := cas.m.courant(cas.nomHist); got != cas.courant {
			t.Errorf("module %s : courant(%s) = %s, attendu %s", cas.m.Chemin, cas.nomHist, got, cas.courant)
		}
	}
	if got, attendu := demain.SurDisque(racineDesCouches+"source"), filepath.Join("R", "internal", "source"); got != attendu {
		t.Errorf("SurDisque sous le module de demain = %s, attendu %s", got, attendu)
	}
}

// TestCadreHistorique_LaBriqueDeplaceeRendLaMemeEmpreinte : LA PREUVE DE BOUT EN BOUT. La meme
// brique, posee une fois dans un module `levelup/go-api` et une fois comme module
// `github.com/JGtm/LevelUp/libs/film`, rend la meme empreinte de couche et le meme perimetre —
// amont, paquet importe hors couche et fichier embarque compris.
func TestCadreHistorique_LaBriqueDeplaceeRendLaMemeEmpreinte(t *testing.T) {
	couches := []Couche{
		{Nom: "source", Racine: racineDesCouches + "source"},
		{Nom: "grammar", Racine: racineDesCouches + "grammar"},
	}
	calculer := func(chemin, dossier, brique string, retouche bool) (Resultat, Perimetre) {
		t.Helper()
		m := Module{Racine: briqueDeFixture(t, chemin, dossier, retouche), Chemin: chemin, brique: brique}
		res, per, err := m.CalculerCouche("grammar", couches, nil, map[string]string{"source": "source-1"})
		if err != nil {
			t.Fatalf("module %s : %v", chemin, err)
		}
		return res, per
	}
	resHier, perHier := calculer(moduleHistorique, dossierHistoriqueDeLaBrique, cadreHistorique, false)
	resDemain, perDemain := calculer(briqueFuture, "", briqueFuture, false)
	if resDemain != resHier {
		t.Errorf("empreinte de la couche grammar : hier %+v, demain %+v — le deplacement de la "+
			"brique change l empreinte", resHier, resDemain)
	}
	if !perDemain.Egal(perHier) {
		t.Errorf("perimetre : hier %+v, demain %+v — le golden de perimetre serait a reecrire",
			perHier, perDemain)
	}
	attendus := []string{racineDesCouches + "grammar", racineDesCouches + "grammar/sous", dossierHistoriqueDeLaBrique + "/types"}
	if !slices.Equal(perHier.Paquets, attendus) {
		t.Errorf("perimetre de la fixture = %v, attendu %v", perHier.Paquets, attendus)
	}
	if resMute, _ := calculer(briqueFuture, "", briqueFuture, true); resMute == resDemain {
		t.Error("la donnee embarquee d un paquet importe change et l empreinte de demain ne bouge " +
			"pas : la fixture ne prouve rien")
	}
}

// briqueDeFixture pose une brique minimale (couches `source` et `grammar`, paquet `types` qui
// embarque une table) dans un module `chemin`, sous le dossier `dossier` du module.
func briqueDeFixture(t *testing.T, chemin, dossier string, retouche bool) string {
	t.Helper()
	imp := path.Join(chemin, dossier)
	table := "1\ta\n"
	if retouche {
		table = "1\tb\n"
	}
	fichiers := map[string]string{
		"internal/source/s.go": "package source\n\nconst S = 1\n",
		"internal/grammar/g.go": "package grammar\n\nimport (\n\t_ \"" + imp + "/internal/grammar/sous\"\n" +
			"\t_ \"" + imp + "/internal/source\"\n\t_ \"" + imp + "/types\"\n)\n\nconst Brique = \"" + imp + "\"\n",
		"internal/grammar/sous/x.go": "package sous\n\nconst X = `" + imp + "/types`\n",
		"types/t.go":                 "package types\n\nimport _ \"embed\"\n\n//go:embed data/t.tsv\nvar T string\n",
		"types/data/t.tsv":           table,
	}
	racine := t.TempDir()
	ecrireFixture(t, filepath.Join(racine, "go.mod"), "module "+chemin+"\n\ngo 1.22\n")
	for nom, contenu := range fichiers {
		ecrireFixture(t, filepath.Join(racine, filepath.FromSlash(path.Join(dossier, nom))), contenu)
	}
	return racine
}

// ecrireFixture ecrit un fichier de fixture, dossiers compris.
func ecrireFixture(t *testing.T, chemin, contenu string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(chemin), 0o750); err != nil {
		t.Fatalf("mkdir %s : %v", chemin, err)
	}
	if err := os.WriteFile(chemin, []byte(contenu), 0o600); err != nil {
		t.Fatalf("ecrire %s : %v", chemin, err)
	}
}

// TestCheminModuleCourantEstLaRacineDeLaBrique : la lecture de [cheminModuleCourant] suppose que
// `revision` vit DIRECTEMENT sous la racine de la brique. Un deplacement du paquet (sous
// `film/internal/`, par exemple) ferait lire un autre prefixe, et le cadre ne reconnaitrait plus
// les imports de la brique le jour ou elle change de module.
func TestCheminModuleCourantEstLaRacineDeLaBrique(t *testing.T) {
	paquet := reflect.TypeFor[Resultat]().PkgPath()
	if paquet != cheminModuleCourant+"/revision" || path.Base(cheminModuleCourant) != "film" {
		t.Errorf("chemin du paquet %s, racine de brique lue %s : `revision` doit vivre directement "+
			"sous la racine `film` de la brique", paquet, cheminModuleCourant)
	}
	m, err := ModuleDe(".")
	if err != nil {
		t.Fatalf("module : %v", err)
	}
	for _, c := range CouchesRevisees() {
		if _, err := os.Stat(m.SurDisque(c.Racine)); err != nil {
			t.Errorf("la racine historique %s ne se resout pas sur le disque du module %s : %v",
				c.Racine, m.Chemin, err)
		}
	}
}
