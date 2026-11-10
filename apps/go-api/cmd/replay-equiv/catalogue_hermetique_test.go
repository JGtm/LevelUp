package main

// catalogue_hermetique_test.go — LE CATALOGUE DU HARNAIS EXPLIQUE LES REFERENCES, SANS LE POSTE.
//
// Une carte doit figurer dans (versionne + fixture) si et seulement si la reference figee du
// film ne dit pas `map_absent`. Les deux sens comptent : une carte en trop ferait diverger un
// film dont la reference est `map_absent`, une carte en moins un film dont la reference porte
// ses points. Le test ne lit aucun overlay de poste.

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/replaybuild"
	"levelup/go-api/internal/testutil"

	"github.com/JGtm/LevelUp/libs/film/replay"
)

const (
	titreDuHarnais     = "halo_infinite"
	etapeEtatDesSocles = "spawnPointsState"
)

// etatDeReference lit l'etat des socles que la reference TSV d'un film a fige. Le TSV ne
// porte la longueur de la chaine (`digest.Of`) : l'etat se retrouve par cette longueur.
func etatDeReference(t *testing.T, tsv string) string {
	t.Helper()
	raw, err := os.ReadFile(tsv)
	if err != nil {
		t.Fatalf("reference illisible : %v", err)
	}
	for _, l := range strings.Split(string(raw), "\n") {
		champs := strings.Split(strings.TrimSpace(l), "\t")
		if len(champs) != 3 || champs[0] != etapeEtatDesSocles {
			continue
		}
		return etatParLongueur(t, champs[1], tsv)
	}
	t.Fatalf("%s : pas de ligne %s", tsv, etapeEtatDesSocles)
	return ""
}

// etatParLongueur : les trois etats ont des longueurs distinctes (10, 11, 15) ; une quatrieme
// valeur ferait echouer ici plutot que de se laisser lire comme l'une des trois.
func etatParLongueur(t *testing.T, longueur, tsv string) string {
	t.Helper()
	for _, etat := range []string{
		replay.SpawnPointsMapAbsent, replay.SpawnPointsEstablished, replay.SpawnPointsNotEstablished,
	} {
		if longueur == strconv.Itoa(len(etat)) {
			return etat
		}
	}
	t.Fatalf("%s : longueur d'etat %s inconnue", tsv, longueur)
	return ""
}

func TestCatalogueDuHarnaisExpliqueLesReferences(t *testing.T) {
	racine, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot : %v", err)
	}
	cat, err := catalogueDesSocles(context.Background(), racine, titreDuHarnais)
	if err != nil {
		t.Fatal(err)
	}
	dossier, err := dossierEquivalence()
	if err != nil {
		t.Fatalf("dossier des references : %v", err)
	}
	faitsFichiers, err := filepath.Glob(filepath.Join(dossier, "*.facts.json"))
	if err != nil || len(faitsFichiers) == 0 {
		t.Fatalf("aucun fichier de faits sous %s (err=%v)", dossier, err)
	}
	for _, faitsPath := range faitsFichiers {
		film := strings.TrimSuffix(filepath.Base(faitsPath), ".facts.json")
		t.Run(film, func(t *testing.T) {
			faits, err := replaybuild.ReadFactsFile(faitsPath)
			if err != nil {
				t.Fatal(err)
			}
			_, presente := cat.Maps[faits.MapID]
			attendue := etatDeReference(t, filepath.Join(dossier, film+".tsv")) != replay.SpawnPointsMapAbsent
			if presente != attendue {
				t.Errorf("carte %q : presente=%v au catalogue du harnais, la reference l'exige %v",
					faits.MapID, presente, attendue)
			}
		})
	}
}

// TestFixtureDesSoclesEstObligatoire : sans fixture, `LoadMapWeaponPadsMerged` rendrait le
// versionne seul en silence ; le harnais doit refuser.
func TestFixtureDesSoclesEstObligatoire(t *testing.T) {
	racine, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot : %v", err)
	}
	// Une racine qui porte le catalogue versionne mais pas la fixture.
	vide := t.TempDir()
	copie := filepath.Join(vide, "data", "titles", titreDuHarnais, "reference", "map_weapon_pads.json")
	if err := os.MkdirAll(filepath.Dir(copie), 0o750); err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile(filepath.Join(racine, "data", "titles", titreDuHarnais, "reference", "map_weapon_pads.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(copie, src, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := catalogueDesSocles(context.Background(), vide, titreDuHarnais); err == nil {
		t.Fatal("une fixture absente doit etre refusee")
	}
}
