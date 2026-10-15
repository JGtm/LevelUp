package weapons

// forerunner_tint_test.go — GARDE-RAIL : la teinte de tir `forerunner` suit la FACTION du registre.
//
// POURQUOI. Le registre (`registry.go`) dit à quelle faction appartient chaque arme ; la teinte
// de l'éclair de bouche (`[shot_tints]` de config/titles/{titre}/mappings/replay_labels.toml)
// est une seconde déclaration, écrite à la main. Elles ont divergé : deux armes classées
// Forerunner au registre étaient teintées plasma. Ce test lie les deux, pour chaque titre qui
// déclare des teintes de tir :
//   - toute arme du titre de faction `forerunner` porte la teinte `forerunner` ;
//   - toute arme teintée `forerunner` est de faction `forerunner` (pas de teinte orpheline).
//
// TITLE-AGNOSTIC : les titres sont les dossiers de config/titles ; une arme appartient à un
// titre par le champ `title` du registre. Un titre sans `[shot_tints]` (ou sans
// replay_labels.toml) n'est pas contrôlé — il rend des tirs à la teinte neutre, ce qui est un
// rendu entier.

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

const teinteForerunner = "forerunner"

type shotTintsTOML struct {
	ShotTints map[string]string `toml:"shot_tints"`
}

// ecartsForerunner rend, triés, les désaccords entre la faction du registre et la teinte
// déclarée pour les armes d'un titre.
func ecartsForerunner(titre string, teintes map[string]string) []string {
	var out []string
	for _, w := range weaponRegistryWeapons {
		if w.title != titre {
			continue
		}
		teinte, declaree := teintes[w.key]
		switch {
		case w.faction == facForerunner && teinte != teinteForerunner:
			out = append(out, w.key+" : faction forerunner, teinte "+quoteOuAbsente(teinte, declaree))
		case w.faction != facForerunner && teinte == teinteForerunner:
			out = append(out, w.key+" : teinte forerunner, faction "+w.faction)
		}
	}
	sort.Strings(out)
	return out
}

func quoteOuAbsente(teinte string, declaree bool) string {
	if !declaree {
		return "absente"
	}
	return "\"" + teinte + "\""
}

func TestTeinteForerunner_SuitLaFactionDuRegistre(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "..")
	files, err := filepath.Glob(filepath.Join(repoRoot, "config", "titles", "*", "mappings", "replay_labels.toml"))
	if err != nil || len(files) == 0 {
		t.Fatalf("aucun replay_labels.toml sous config/titles (err=%v)", err)
	}
	for _, file := range files {
		titre := filepath.Base(filepath.Dir(filepath.Dir(file)))
		t.Run(titre, func(t *testing.T) {
			raw, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("lecture: %v", err)
			}
			var doc shotTintsTOML
			if err := toml.Unmarshal(raw, &doc); err != nil {
				t.Fatalf("décodage: %v", err)
			}
			if len(doc.ShotTints) == 0 {
				t.Skip("titre sans [shot_tints] : rien à contrôler")
			}
			for _, e := range ecartsForerunner(titre, doc.ShotTints) {
				t.Errorf("%s", e)
			}
		})
	}
}

// Le contrôle SAIT ÊTRE ROUGE : une arme Forerunner mal teintée, absente ou une teinte
// orpheline sont rapportées.
func TestEcartsForerunner_DetecteLesDeuxSens(t *testing.T) {
	// Les trois armes Forerunner du titre Infinite et un témoin d'une autre faction, par le registre.
	teintes := map[string]string{
		"hinf_sentinel_beam": teinteForerunner,
		"hinf_cindershot":    "plasma_cool", // mal teintée
		// hinf_heatwave absente
		"hinf_ma40_ar": teinteForerunner, // teinte orpheline (arme humaine)
	}
	got := ecartsForerunner(titleHINF, teintes)
	want := []string{
		"hinf_cindershot : faction forerunner, teinte \"plasma_cool\"",
		"hinf_heatwave : faction forerunner, teinte absente",
		"hinf_ma40_ar : teinte forerunner, faction human",
	}
	if len(got) != len(want) {
		t.Fatalf("ecartsForerunner = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("écart %d = %q, want %q", i, got[i], want[i])
		}
	}
}
