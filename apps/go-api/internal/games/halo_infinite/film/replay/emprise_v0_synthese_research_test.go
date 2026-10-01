package replay

// emprise_v0_synthese_research_test.go — LOT V0 DU PLAN `.ai/PLAN_EMPRISE_VIES_2026-09-28.md` :
// LE RAPPORT AGREGE. `TestEmpriseV0Rapport` relit les resultats poses par le volet collecteur
// (`col/`, dont la mesure V0.1) et par le volet porteurs (`porteurs/`), et imprime les tableaux
// colles au journal du lot, avec le verdict chiffre de chaque seuil. Il se saute sans
// `EMPRISE_V0_DIR`.
//
//	EMPRISE_V0_DIR=<scratch> go test ./internal/games/halo_infinite/film/replay/ \
//	  -run '^TestEmpriseV0Rapport$' -v -count=1

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/title"
)

// v0V01 : le resultat V0.1 pose par le volet collecteur (sous-ensemble).
type v0V01 struct {
	Episodes          int `json:"episodes"`
	EpisodesProximite int `json:"episodesProximite"`
	EpisodesSansXUID  int `json:"episodesSansXuid"`
	OccupantsSansVie  int `json:"episodesOccupantSansVieCollecteur"`
	Sondes            int `json:"sondes"`
	NonSitues         int `json:"nonSitues"`
	Sondes1s          int `json:"sondesPremiereSeconde"`
	NonSitues1s       int `json:"nonSituesPremiereSeconde"`
}

// v0ColV01 : la ligne du volet collecteur relue par le rapport.
type v0ColV01 struct {
	MatchID  string  `json:"matchId"`
	Variante string  `json:"variante"`
	V01      *v0V01  `json:"v01"`
	Refus    string  `json:"v01Refus"`
	Passe    float64 `json:"passeMedianeMs"`
}

func TestEmpriseV0Rapport(t *testing.T) {
	dir := os.Getenv("EMPRISE_V0_DIR")
	if dir == "" {
		t.Skip("EMPRISE_V0_DIR requis : rapport saute")
	}
	v0RapportV01(t, dir)
	v0RapportV02(t, dir)
}

// v0Fichiers liste les JSON d'un sous-repertoire du lot, tries.
func v0Fichiers(t *testing.T, dir, sous string) []string {
	t.Helper()
	noms, err := filepath.Glob(filepath.Join(dir, sous, "*.json"))
	if err != nil {
		t.Fatalf("%s : %v", sous, err)
	}
	sort.Strings(noms)
	return noms
}

func v0Relire(t *testing.T, chemin string, v any) {
	t.Helper()
	blob, err := os.ReadFile(chemin)
	if err != nil || json.Unmarshal(blob, v) != nil {
		t.Fatalf("%s illisible (%v)", chemin, err)
	}
}

// v0RapportV01 : le tableau V0.1, par film et agrege.
func v0RapportV01(t *testing.T, dir string) {
	t.Helper()
	var b strings.Builder
	b.WriteString("\nV0.1 | film | variante | episodes lus | sondes (100 ms) | non situes | part | 1re seconde non situee\n")
	var tot v0V01
	for _, f := range v0Fichiers(t, dir, "col") {
		var c v0ColV01
		v0Relire(t, f, &c)
		if c.V01 == nil || c.V01.Episodes == 0 {
			continue
		}
		v := c.V01
		fmt.Fprintf(&b, "V0.1 | %s | %s | %d (sans xuid %d, proximite ecartes %d) | %d | %d | %.2f %% | %d/%d\n",
			title.FilmShortMatchID(c.MatchID), c.Variante, v.Episodes, v.EpisodesSansXUID,
			v.EpisodesProximite, v.Sondes, v.NonSitues, 100*float64(v.NonSitues)/float64(v.Sondes),
			v.NonSitues1s, v.Sondes1s)
		tot.Episodes += v.Episodes
		tot.Sondes += v.Sondes
		tot.NonSitues += v.NonSitues
		tot.Sondes1s += v.Sondes1s
		tot.NonSitues1s += v.NonSitues1s
	}
	if tot.Sondes > 0 {
		fmt.Fprintf(&b, "V0.1 | AGREGE | | %d | %d | %d | %.2f %% | %d/%d (seuil >= 95 %%)\n", tot.Episodes,
			tot.Sondes, tot.NonSitues, 100*float64(tot.NonSitues)/float64(tot.Sondes),
			tot.NonSitues1s, tot.Sondes1s)
	}
	t.Log(b.String())
}

// v0RapportV02 : les tableaux V0.2, par film et agreges, et les verdicts par seuil.
func v0RapportV02(t *testing.T, dir string) {
	t.Helper()
	var b strings.Builder
	b.WriteString("\nV0.2 | film | variante | famille | registre | passe ms | (a) ms | (a) % | (b) ms | (b) % | pic passe | pic base | pic (a) | pic (b) Gio\n")
	agg := map[string]map[string]v0Fid{"a": {}, "b": {}, "b-sans-monde": {}}
	var sa, sb float64
	var n int
	var ra, rb float64
	for _, f := range v0Fichiers(t, dir, "porteurs") {
		var r v0Resultat
		v0Relire(t, f, &r)
		if r.Famille == "" {
			// HORS MODE A PORTEUR : la garde de mode ne lit rien, aucune des deux voies ne paie.
			fmt.Fprintf(&b, "V0.2 | %s | %s | - | %s | %.0f | non lu | 0 | non lu | 0 | %.2f | %.2f | - | -\n",
				title.FilmShortMatchID(r.MatchID), r.Variante, v0OuiNon(r.RegistreEcart), r.PasseMS,
				v0G(r.PicPasse), v0G(r.PicBase))
			continue
		}
		fmt.Fprintf(&b, "V0.2 | %s | %s | %s | %s | %.0f | %.0f | %.1f | %.0f | %.1f | %.2f | %.2f | %.2f | %.2f\n",
			title.FilmShortMatchID(r.MatchID), r.Variante, r.Famille, v0OuiNon(r.RegistreEcart),
			r.PasseMS, r.CoutA, r.SurcoutA, r.CoutB, r.SurcoutB, v0G(r.PicPasse), v0G(r.PicBase),
			v0G(r.PicA), v0G(r.PicB))
		if r.Famille == "drapeau" {
			fmt.Fprintf(&b, "V0.2 |   %s detail (b) : statborg %.0f ms, pont %.0f ms, equipes %.0f ms, objets du monde %.0f ms, assemblage %.0f ms ; sensibilite (b) sans objets du monde %.0f ms (%.1f %%)\n",
				title.FilmShortMatchID(r.MatchID), r.CoutStatborg, r.CoutPont, r.CoutEquipes, r.CoutMonde,
				r.CoutAssemblage, r.CoutBSansMonde, 100*r.CoutBSansMonde/r.PasseMS)
		}
		n++
		sa, sb = sa+r.SurcoutA, sb+r.SurcoutB
		ra, rb = max(ra, float64(r.PicA)/float64(r.PicBase)), max(rb, float64(r.PicB)/float64(r.PicBase))
		for _, voie := range []string{"a", "b", "b-sans-monde"} {
			for _, fam := range []string{"drapeau", "crane", "bombe", "vip"} {
				v, ok := r.Fid[voie][fam]
				if !ok {
					continue
				}
				x := agg[voie][fam]
				x.ajouter(v)
				agg[voie][fam] = x
				fmt.Fprintf(&b, "V0.2 |   %s voie (%s) %s : reference %d ms, identique +-100 ms %d ms, en trop %d ms -> %.2f %%\n",
					title.FilmShortMatchID(r.MatchID), voie, fam, v.RefMS, v.OkMS, v.ExtraMS, 100*v.Taux())
			}
		}
	}
	for _, voie := range []string{"a", "b", "b-sans-monde"} {
		var tot v0Fid
		for _, fam := range []string{"drapeau", "crane", "bombe", "vip"} {
			if v, ok := agg[voie][fam]; ok {
				tot.ajouter(v)
				fmt.Fprintf(&b, "V0.2 | AGREGE voie (%s) %s : reference %d ms, identique %d ms, en trop %d ms -> %.2f %%\n",
					voie, fam, v.RefMS, v.OkMS, v.ExtraMS, 100*v.Taux())
			}
		}
		fmt.Fprintf(&b, "V0.2 | AGREGE voie (%s) TOUTES FAMILLES : %.2f %% (seuil >= 98 %%)\n", voie, 100*tot.Taux())
	}
	if n > 0 {
		fmt.Fprintf(&b, "V0.2 | SURCOUT MOYEN sur %d matchs a porteur : (a) %.1f %%, (b) %.1f %% (seuil <= 25 %%)\n",
			n, sa/float64(n), sb/float64(n))
		fmt.Fprintf(&b, "V0.2 | PIC MEMOIRE, pire rapport a la base : (a) %.2f x, (b) %.2f x (seuil <= 1,5 x)\n", ra, rb)
	}
	t.Log(b.String())
}
