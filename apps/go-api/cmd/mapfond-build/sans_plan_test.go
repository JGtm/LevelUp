package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/testutil"

	"github.com/JGtm/halo-infinite-map-images/himap"
)

// Une carte Forge dont le blob manque au dépôt cuit sans plan et rend la dégradation nommée, qu'elle
// prenne ou non sa référence sur le maillage.
func TestMaillageForgeSansBlobPoseLaDegradation(t *testing.T) {
	if len(himap.CartesForge) == 0 {
		t.Skip("aucune carte Forge declaree")
	}
	carte := himap.CartesForge[0]
	for nom, reglages := range map[string]map[string]reglageCarte{
		"sans reference": {},
		"avec reference": {carte.MapID: {NavmeshReference: true, GateLe: "2026-11-16"}},
	} {
		e := &environnement{repoRoot: t.TempDir(), reglages: &reglagesFond{Cartes: reglages}}
		arme, tout, degradation := e.maillageForge(context.Background(), carte)
		if arme != nil || tout != nil {
			t.Errorf("%s : un maillage est rendu sans blob", nom)
		}
		if degradation != degradationSansBlob {
			t.Errorf("%s : degradation = %q, attendu %q", nom, degradation, degradationSansBlob)
		}
	}
}

// Un blob présent mais illisible n'est pas une dégradation : la carte publiée sans plan doit se
// voir dans le journal, et ne dit pas « sans maillage » à tort.
func TestMaillageForgeBlobIllisibleNeDitPasSansMaillage(t *testing.T) {
	if len(himap.CartesForge) == 0 {
		t.Skip("aucune carte Forge declaree")
	}
	carte := himap.CartesForge[0]
	racine := t.TempDir()
	dossier := filepath.Join(racine, filepath.FromSlash(himap.DepotNavmesh))
	if err := os.MkdirAll(dossier, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dossier, carte.MapID+".blob"), []byte("pas un maillage"), 0o644); err != nil {
		t.Fatal(err)
	}
	e := &environnement{repoRoot: racine, reglages: &reglagesFond{Cartes: map[string]reglageCarte{}}}
	if _, _, degradation := e.maillageForge(context.Background(), carte); degradation != "" {
		t.Errorf("blob illisible : degradation = %q, attendu aucune", degradation)
	}
}

// fondsSansPlanAttendus : l'ensemble EXACT des fonds publiés sans plan des murs. Live Fire n'a pas de
// maillage dans son module ; les deux variantes de Thunderhead n'ont pas de blob au dépôt. Une carte
// qui entre ou sort de cette liste est un changement qui se décide, pas qui arrive.
var fondsSansPlanAttendus = []string{
	"28a3ac28-f69d-4fa9-9ebf-a0449c89c8da", // Thunderhead
	"37bc3df6-93e8-4d74-b16e-5ceaa30ebc23", // Thunderhead Heavies
	"sgh_interlock",                        // Live Fire
}

func TestFondsPublieSansPlanSontTroisEtNommes(t *testing.T) {
	racine, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot introuvable : %v", err)
	}
	dossier := title.NewPathResolver(racine).MapBackgroundDir(title.DefaultSlug)
	sidecars, err := filepath.Glob(filepath.Join(dossier, "*.json"))
	if err != nil || len(sidecars) == 0 {
		t.Skipf("aucun fond publie (%v)", err)
	}
	var sansPlan []string
	for _, chemin := range sidecars {
		blob, err := os.ReadFile(chemin)
		if err != nil {
			t.Fatal(err)
		}
		var s struct {
			Plan      json.RawMessage `json:"plan"`
			PlanImage string          `json:"planImage"`
		}
		if err := json.Unmarshal(blob, &s); err != nil {
			t.Fatalf("%s illisible : %v", chemin, err)
		}
		cle := strings.TrimSuffix(filepath.Base(chemin), ".json")
		if len(s.Plan) == 0 || string(s.Plan) == "null" {
			sansPlan = append(sansPlan, cle)
			if s.PlanImage != "" {
				t.Errorf("%s : image plan %q sans plan des murs", cle, s.PlanImage)
			}
		} else if s.PlanImage == "" {
			t.Errorf("%s : plan des murs sans image plan", cle)
		}
	}
	sort.Strings(sansPlan)
	if !reflect.DeepEqual(sansPlan, fondsSansPlanAttendus) {
		t.Errorf("fonds publiés sans plan = %v, attendu %v", sansPlan, fondsSansPlanAttendus)
	}
}
