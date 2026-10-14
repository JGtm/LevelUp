package mappings

// loader_regulation_timeline_quality_test.go — CONTRÔLE DE QUALITÉ de la règle déclarée
// `[score_timeline]` : toute variante JOUÉE qui marque au frag doit être déclarée `hidden`.
//
// POURQUOI UN CONTRÔLE, ET POURQUOI ICI. La page de rejeu ne décide plus de la piste « Score »
// par une comparaison de rendu : la règle déclarée (`header.score_timeline_kind`) décide SEULE.
// Une variante de Slayer absente de la table retombe sur `curve` (le repli sûr) et réaffiche,
// sous la dominance, une piste qui la répète. Ce test rend cette absence ROUGE au lieu de la
// laisser se découvrir à l'écran.
//
// DÉFINITIONS, TOUTES LUES DANS LES DONNÉES DU TITRE (aucun nom de titre, aucun slug) :
//   - variante JOUÉE : une clé de `[score_target]` du même regulation.toml (la table des
//     variantes mesurées sur le registre) ;
//   - variante à OBJECTIF : son libellé porte un jeton de `objective_roles.toml` (`[[modes]]
//     match`), ou un jeton déclaré `events` dans `[score_timeline]` — l'objectif, pas le frag,
//     fait le score ;
//   - variante qui MARQUE AU FRAG : jouée, sans objectif. Elle doit résoudre `hidden`.
//
// LES OBJECTIFS SANS ENTRÉE DE CATALOGUE sont une donnée de test explicite (`objectifsHorsCatalogue`)
// et non un contournement : leur lecture reste volontairement au repli `curve` (décision produit
// consignée dans regulation.toml), ils ne marquent pas au frag pour autant.

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"

	"levelup/go-api/internal/analysis/modelabel"
)

// objectifsHorsCatalogue : jetons de modes à OBJECTIF qu'aucune entrée de objective_roles.toml
// ne nomme (rôles non décodés ou sans décision produit). Un jeton de cette liste exclut la
// variante du contrôle « marque au frag ». Toute entrée ici doit nommer un mode dont le score
// est un compteur d'objectif.
var objectifsHorsCatalogue = []string{"VIP", "Total Control"}

// regulationTimelineTOML projette ce que le contrôle lit de regulation.toml.
type regulationTimelineTOML struct {
	ScoreTarget   map[string]int    `toml:"score_target"`
	ScoreTimeline map[string]string `toml:"score_timeline"`
}

// objectiveTokensTOML projette les jetons de objective_roles.toml.
type objectiveTokensTOML struct {
	Modes []struct {
		Match []string `toml:"match"`
	} `toml:"modes"`
}

// variantesFragSansRegle rend les variantes jouées qui marquent au frag (aucun jeton
// d'objectif) et dont la règle déclarée n'est PAS `hidden`, triées.
func variantesFragSansRegle(
	reg *RegulationSet, played []string, timeline map[string]string, objectiveTokens []string,
) []string {
	tokens := append([]string(nil), objectiveTokens...)
	tokens = append(tokens, objectifsHorsCatalogue...)
	for token, kind := range timeline {
		if kind == ScoreTimelineEvents {
			tokens = append(tokens, token)
		}
	}
	var out []string
	for _, variant := range played {
		if modelabel.ExtractKnownMode(variant, tokens) != "" {
			continue // mode à objectif : le frag ne fait pas le score
		}
		if reg.ScoreTimelineKind(variant) != ScoreTimelineHidden {
			out = append(out, variant)
		}
	}
	sort.Strings(out)
	return out
}

func TestScoreTimeline_VariantesMarquantAuFragDeclarees(t *testing.T) {
	t.Parallel()
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "..")
	files, err := filepath.Glob(filepath.Join(repoRoot, "config", "titles", "*", "mappings", "regulation.toml"))
	if err != nil || len(files) == 0 {
		t.Fatalf("aucun regulation.toml sous config/titles (err=%v)", err)
	}
	for _, file := range files {
		file := file
		t.Run(filepath.Base(filepath.Dir(filepath.Dir(file))), func(t *testing.T) {
			reg, err := LoadRegulationFromFile(file)
			if err != nil {
				t.Fatalf("chargement: %v", err)
			}
			raw, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("lecture: %v", err)
			}
			var doc regulationTimelineTOML
			if err := toml.Unmarshal(raw, &doc); err != nil {
				t.Fatalf("décodage: %v", err)
			}
			if len(doc.ScoreTarget) == 0 {
				t.Skip("titre sans variante mesurée : rien à contrôler")
			}
			played := make([]string, 0, len(doc.ScoreTarget))
			for variant := range doc.ScoreTarget {
				played = append(played, variant)
			}
			var objective []string
			if rawRoles, err := os.ReadFile(filepath.Join(filepath.Dir(file), "objective_roles.toml")); err == nil {
				var roles objectiveTokensTOML
				if err := toml.Unmarshal(rawRoles, &roles); err != nil {
					t.Fatalf("objective_roles.toml: %v", err)
				}
				for _, m := range roles.Modes {
					objective = append(objective, m.Match...)
				}
			}
			if missing := variantesFragSansRegle(reg, played, doc.ScoreTimeline, objective); len(missing) > 0 {
				t.Errorf("variantes jouées qui marquent au frag sans règle `hidden` dans [score_timeline] : %s",
					strings.Join(missing, ", "))
			}
		})
	}
}

// Le contrôle SAIT ÊTRE ROUGE : une variante de frag absente de la table est rapportée, une
// variante à objectif ou déclarée `hidden` ne l'est pas.
func TestVariantesFragSansRegle_DetecteLeManque(t *testing.T) {
	t.Parallel()
	reg, err := LoadRegulationFromBytes("regulation.toml", []byte(`
[meta]
title_slug     = "synthetique"
schema_version = 8

[score_timeline]
"Slayer" = "hidden"
"CTF"    = "events"
`))
	if err != nil {
		t.Fatalf("chargement: %v", err)
	}
	played := []string{"Arena:Slayer", "Arena:CTF", "Arena:Bounty", "Arena:Hunt", "Arena:VIP"}
	got := variantesFragSansRegle(reg, played, map[string]string{"Slayer": "hidden", "CTF": "events"}, nil)
	want := []string{"Arena:Bounty", "Arena:Hunt"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("variantesFragSansRegle = %v, want %v", got, want)
	}
}
