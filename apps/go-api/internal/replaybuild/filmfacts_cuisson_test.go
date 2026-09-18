package replaybuild

// filmfacts_cuisson_test.go — LES GARDES DE LA BASCULE (lot 4.1.2, 2026-09-17).
//
// AUCUN OCTET DE FILM : ces tests lisent le SOURCE et des structures fabriquees a la main.

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/port"
)

// ecrivainsDArtefactAutorises — LES SEULS appelants de `writeArtifactBytes`, avec leur raison.
// Chemins relatifs a `internal/replaybuild`. 2026-09-17, lot 4.1.2.
var ecrivainsDArtefactAutorises = map[string]string{
	"replaybuild.go": "`BuildMatch` : la composition IN-PROCESSUS (CLI unitaire, enfant de " +
		"backfill, action admin) — construit puis range",
	"artifact_store.go": "`StoreArtifact` : le chemin du POST-SYNC, ou l enfant construit et le " +
		"serveur range ; plus la definition de `writeArtifactBytes` elle-meme",
	"artifact_digest.go": "`StoreArtifactBytes` : la porte des appelants qui tiennent deja les " +
		"octets et veulent le digest en retour",
}

// TestBrancheDesFaitsTraverseLeMemePuits : LE PUITS D OCTETS D ARTEFACT RESTE UNIQUE.
//
// # CE QUE CE TEST GARDE, ET CE QU IL CORRIGE
//
// L item 4.1.2 du plan cite `no_second_artifact_sink_test` comme garde de cette unicite. C EST
// FAUX, et c est une decouverte du lot : ce ratchet-la compte les cablages du PUITS DE
// NOTIFICATION (`SetArtifactStoredSink`, notification Discord groupee, deux appelants autorises).
// Il ne dit rien des octets ecrits sur le disque.
//
// Ce que la bascule « relire les faits » doit preserver est que TOUT artefact — quelle que soit la
// branche qui l a produit — traverse `writeArtifactBytes` (`artifact_store.go`) : trois refus
// avant ecriture, garde anti-regression, ecriture atomique. Ce test le garde de deux cotes :
//
//  1. les appelants de `writeArtifactBytes` sont EXACTEMENT ceux de l allowlist datee ;
//  2. les trois fonctions de la bascule n en appellent AUCUN — elles rendent des octets, elles
//     ne rangent rien, et les deux branches reviennent donc au meme point d ecriture.
func TestBrancheDesFaitsTraverseLeMemePuits(t *testing.T) {
	vus := map[string]bool{}
	for _, chemin := range fichiersDeProductionDuPaquet(t) {
		blob, err := os.ReadFile(chemin) //nolint:gosec // chemin du paquet courant
		if err != nil {
			t.Fatalf("%s : %v", chemin, err)
		}
		if !strings.Contains(string(blob), "writeArtifactBytes(") {
			continue
		}
		vus[filepath.Base(chemin)] = true
	}
	for nom := range vus {
		if _, ok := ecrivainsDArtefactAutorises[nom]; !ok {
			t.Errorf("%s ecrit des octets d artefact hors allowlist : un SECOND puits d octets "+
				"perdrait les trois refus, la garde anti-regression et l ecriture atomique de "+
				"`writeArtifactBytes`. L inscrire avec sa raison, ou passer par la porte.", nom)
		}
	}
	for nom, raison := range ecrivainsDArtefactAutorises {
		if !vus[nom] {
			t.Errorf("%s est dans l allowlist des ecrivains d artefact (%q) mais n ecrit plus : "+
				"entree perimee, la retirer.", nom, raison)
		}
	}
	for _, fn := range []string{"BuildBytes", "documentDeLaCuisson", "serialiserDocument"} {
		if corps := corpsDeFonction(t, fn); strings.Contains(corps, "writeArtifactBytes(") {
			t.Errorf("%s ecrit des octets d artefact : la moitie CONSTRUCTION ne range rien (lot "+
				"BUILDALL), et une ecriture ici doublerait le puits sur la branche des faits.", fn)
		}
	}
}

// TestLaBasculeNeTouchePasAuVerrouSolo : le verrou solo reste chez les points d entree.
//
// Un rejeu depuis les faits ne decompresse aucun chunk et ne merite pas le verrou ; le sortir du
// verrou deplacerait pourtant une garantie MEMOIRE dans une boucle — la porte par laquelle quatre
// sinistres RAM sont passes. La decision du lot 4.1 est de NE RIEN CHANGER (note §2.5), et ce test
// la tient : `replaybuild` ne prend ni ne relache le verrou.
func TestLaBasculeNeTouchePasAuVerrouSolo(t *testing.T) {
	for _, chemin := range fichiersDeProductionDuPaquet(t) {
		blob, err := os.ReadFile(chemin) //nolint:gosec // chemin du paquet courant
		if err != nil {
			t.Fatalf("%s : %v", chemin, err)
		}
		// SUR LES LIGNES DE CODE SEULEMENT : ce fichier-ci EXPLIQUE la decision en commentaire,
		// et un balayage du texte entier mordrait sa propre documentation. `filmproc` reste
		// importe par `refus_de_cuisson.go` pour le PROTOCOLE DE CODE DE SORTIE, qui n est pas le
		// verrou — c est le nom du verrou qu on interdit, pas le paquet.
		for i, ligne := range strings.Split(string(blob), "\n") {
			nue := strings.TrimSpace(ligne)
			if strings.HasPrefix(nue, "//") || !strings.Contains(nue, "AcquireSolo") {
				continue
			}
			t.Errorf("%s:%d prend le verrou solo (%s) : il appartient aux SIX points d entree "+
				"(cmd/*, internal/replaychild), jamais a ce paquet — le prendre ici le prendrait "+
				"dans une boucle.", filepath.Base(chemin), i+1, nue)
		}
	}
}

// TestLesDeuxBranchesNommentLeurOrigine : l etape de branche est declaree, pas devinee.
func TestLesDeuxBranchesNommentLeurOrigine(t *testing.T) {
	if EtapeRejeuDepuisLesFaits == "" {
		t.Fatal("l etape de branche n a pas de nom : le harnais d equivalence ne pourrait pas " +
			"distinguer « relu » de « decode »")
	}
	if len(BuildBytesStepsAfter) == 0 || BuildBytesStepsAfter[0] != EtapeRejeuDepuisLesFaits {
		t.Errorf("BuildBytesStepsAfter = %v : l etape de branche doit PRECEDER `artifact` — le "+
			"harnais doit savoir quelle branche a servi avant de comparer les octets.",
			BuildBytesStepsAfter)
	}
}

// fichiersDeProductionDuPaquet rend les `.go` non-test du paquet courant, tries.
func fichiersDeProductionDuPaquet(t *testing.T) []string {
	t.Helper()
	entrees, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("lecture du paquet : %v", err)
	}
	var out []string
	for _, e := range entrees {
		nom := e.Name()
		if e.IsDir() || !strings.HasSuffix(nom, ".go") || strings.HasSuffix(nom, "_test.go") {
			continue
		}
		out = append(out, nom)
	}
	if len(out) < 10 {
		t.Fatalf("%d fichier(s) de production lus : le balayage ne mesure plus rien", len(out))
	}
	sort.Strings(out)
	return out
}

// corpsDeFonction rend le source du corps d une fonction du paquet, cherchee dans les fichiers de
// la suite observee (`observe_test.go`).
func corpsDeFonction(t *testing.T, nom string) string {
	t.Helper()
	for _, fichier := range fichiersDeLaSuiteObservee {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, fichier, nil, 0)
		if err != nil {
			t.Fatalf("%s illisible : %v", fichier, err)
		}
		blob, err := os.ReadFile(fichier) //nolint:gosec // chemin du paquet courant
		if err != nil {
			t.Fatalf("%s : %v", fichier, err)
		}
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Name.Name != nom || fn.Body == nil {
				continue
			}
			return string(blob[fset.Position(fn.Body.Pos()).Offset:fset.Position(fn.Body.End()).Offset])
		}
	}
	t.Fatalf("fonction %q introuvable dans %v : ce garde-rail ne garde plus rien", nom,
		fichiersDeLaSuiteObservee)
	return ""
}

// TestHorlogeDesChunksEstToujoursNonNil : une horloge vide n est pas une horloge absente.
func TestHorlogeDesChunksEstToujoursNonNil(t *testing.T) {
	if got := horlogeDesChunks(nil); got == nil {
		t.Fatal("horlogeDesChunks(nil) rend nil : un film sans chunk datable doit rendre une " +
			"horloge VIDE, pas absente — le balayage de l anneau lit `hasStart` dessus")
	}
}

// TestSectionStatborgVideNeFabriquePasDeCalque : la section vide reste un refus, pas un zero.
func TestSectionStatborgVideNeFabriquePasDeCalque(t *testing.T) {
	st := assemblerFilmStats(context.Background(), "000d5950", replay.FilmStatborg{},
		port.MatchFacts{}, filmDeaths{})
	if st.score != nil {
		t.Error("une section statborg VIDE a produit une courbe de score : le document dirait " +
			"« zero point » la ou rien n a ete lu")
	}
}

// TestCompleterLesFaitsPoseLesDEUXSectionsDeLAssemblage : LA MOITIE `replaybuild` DU GARDE-RAIL
// DU TROU QUE S8 A TROUVE (2026-09-18).
//
// # CE QU IL GARDE
//
// La chaine de production d un fichier de faits traverse DEUX paquets. `replay.faitsDuBalayage`
// pose ce que la couche de publication sait ; les sections STATBORG et KILLSOURCE, elles, naissent
// ICI (`statborgDuFilm`, `decodeKillSource`) et la couche de publication ne les voit jamais
// passer. Personne ne les posait : le fichier ecrit portait deux sections VIDES, l en-tete restait
// FRAIS, et un rejeu publiait un document sans courbe de score, sans actions d objectif, sans
// drapeau, sans couronne, sans crane et sans armement. Seul le gate S8 pouvait l attraper, et il
// l a attrape a la PREMIERE ecriture.
//
// L AUTRE MOITIE DU GARDE-RAIL VIT CHEZ `replay` :
// `TestFaitsDuBalayageLaisseExactementDeuxSectionsAuLAssemblage` y nomme les sections que la
// publication laisse volontairement vides. Une section NEUVE que personne ne remplirait rougit
// la-bas ; celle-ci verifie que les deux nommees arrivent bien.
//
// POURQUOI DEUX TESTS ET PAS UN : `film/internal/*` n est pas importable d ici — c est la
// frontiere du decodeur (ADR 0034) —, donc ce paquet ne peut pas fabriquer une identite de film ni
// un rapport de replis. Chaque moitie vit dans le paquet qui voit son cote.
func TestCompleterLesFaitsPoseLesDEUXSectionsDeLAssemblage(t *testing.T) {
	var f replay.FilmFactsFile
	if !reflect.ValueOf(f.Statborg).IsZero() || f.Kills != nil {
		t.Fatal("le temoin part deja rempli : le test ne mesure rien")
	}
	src := entreesDeCuisson{
		statborg: statborgTemoin(),
		kills:    &decfilm.Result{Calibration: "temoin"},
	}
	completerLesFaits(&f, src)
	if reflect.ValueOf(f.Statborg).IsZero() {
		t.Error("FilmFactsFile.Statborg sort VIDE : un rejeu reconstruirait ses entrees de calque " +
			"sur une section vide et publierait un document appauvri, en-tete FRAIS a l appui.")
	}
	if f.Kills == nil {
		t.Error("FilmFactsFile.Kills sort NIL : les morts neutres, les references de frag, les " +
			"identites de bot et le profil de balayage calibre seraient perdus au rejeu.")
	}
	// UN NIL NE DOIT PAS PANIQUER : le chemin decode peut ne rien avoir a persister (decoupage
	// d i0 illisible), et `ecrireLesFaits` le journalise deja.
	completerLesFaits(nil, src)
}

// statborgTemoin : une section statborg non vide, sans un octet de film.
func statborgTemoin() replay.FilmStatborg {
	return replay.FilmStatborg{
		Records:      []decfilm.StatRecord{{TimeMS: 1, Slot: 10, Comps: map[int]decfilm.StatValue{0: {A: 1}}}},
		BurstMS:      []int{1200},
		Truncated:    true,
		ChunkStartMS: map[int]int{0: 0},
	}
}
