//go:build gamefiles

package himap

import (
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/himodule"
)

// POURQUOI LIVE FIRE NE PLACE AUCUNE DE SES 28 ANCRES SUR UN SOL.
//
// Le balayage du 2026-08-09 donne 0/14 sur `live_fire_sgh_interlock` ET sur sa variante
// classee, quand les trois autres cartes `sgh` (recharge/blueprint, prism/crystalcaves,
// streets/streets) rendent 13/14, 8/14 et 10/14. Le defaut est donc propre a cette carte, pas
// a la famille.
//
// Ce diagnostic compare, pour une carte fautive et une carte temoin, les deux reperes qui
// doivent coincider : l'emprise declaree du sbsp et l'emprise des ancres. S'ils sont
// disjoints, la question n'est plus « pourquoi le sol manque » mais « pourquoi les ancres sont
// ailleurs », ce qui est un tout autre probleme.
func TestDiagnosticLiveFire(t *testing.T) {
	if _, err := DeployRoot(); err != nil {
		t.Skip(err)
	}
	for _, module := range []string{"live_fire_sgh_interlock", "recharge_sgh_blueprint", "cliffhanger_ridgeline"} {
		chemin, ok := ChercheModuleInstalle(module)
		if !ok {
			t.Logf("%-26s module introuvable", module)
			continue
		}
		bsps, err := ReadModuleInstances(chemin)
		if err != nil {
			t.Logf("%-26s lecture KO : %v", module, err)
			continue
		}

		lo := [3]float64{math.Inf(1), math.Inf(1), math.Inf(1)}
		hi := [3]float64{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
		n := 0
		for _, e := range ancresDuModule(t, module) {
			for _, o := range e.Objectives {
				n++
				p := [3]float64{o.Pos.X, o.Pos.Y, o.Pos.Z}
				for k := 0; k < 3; k++ {
					lo[k] = math.Min(lo[k], p[k])
					hi[k] = math.Max(hi[k], p[k])
				}
			}
		}
		t.Logf("%s — %d bsp · %d ancres", module, len(bsps), n)
		t.Logf("   ancres  X [%+7.1f ; %+7.1f]  Y [%+7.1f ; %+7.1f]  Z [%+7.1f ; %+7.1f]",
			lo[0], hi[0], lo[1], hi[1], lo[2], hi[2])
		for i, b := range bsps {
			t.Logf("   bsp %d  X [%+7.1f ; %+7.1f]  Y [%+7.1f ; %+7.1f]  Z [%+7.1f ; %+7.1f]  %d instances",
				i, b.Bounds.Min[0], b.Bounds.Max[0], b.Bounds.Min[1], b.Bounds.Max[1],
				b.Bounds.Min[2], b.Bounds.Max[2], len(b.Instances))
		}
	}
}

// TestInventaireModuleLiveFire — QUE CONTIENT LE MODULE DE LIVE FIRE, exactement.
//
// L'utilisateur joue Live Fire regulierement : la carte est donc bien livree. La conclusion
// « geometrie non installee », tiree le 2026-08-27 de la seule taille du fichier (0,21 Mo
// contre 478 pour sgh_streets), demande donc a etre verifiee par le CONTENU et non par le
// poids. Deux hypotheses a departager :
//
//	a. le module est un talon et ne porte presque aucun tag -> installation partielle ;
//	b. il porte ses tags mais AUCUN sbsp, sa geometrie vivant dans un autre module -> Live
//	   Fire serait une variante batie sur le decor d'une autre carte, et le poids leger
//	   serait celui du seul DELTA.
//
// Ce test ne conclut pas : il compte les tags par groupe pour la carte fautive et pour deux
// temoins, et laisse les chiffres decider.
func TestInventaireModuleLiveFire(t *testing.T) {
	if _, err := DeployRoot(); err != nil {
		t.Skip(err)
	}
	racine, _ := DeployRoot()
	cibles := []string{"live_fire_sgh_interlock", "recharge_sgh_blueprint", "streets_sgh_streets"}
	// Les modules PARTAGES : si la geometrie de Live Fire n est pas chez elle, elle est
	// peut-etre la — multiplayer-rtx-new pese 955 Mo et n appartient a aucune carte.
	partages := []string{
		filepath.Join(racine, "pc", "globals", "multiplayer-rtx-new.module"),
		filepath.Join(racine, "pc", "globals", "common-rtx-new.module"),
		// LES CANEVAS FORGE. Question de l utilisateur du 2026-08-27 : « les cartes Forge ont
		// une base sans doute, sur laquelle dessiner, et nous on ne doit avoir que cette base.
		// Ou inversement. » La cuisson Forge ne pose QUE les objets de la variante et n a
		// jamais dessine le canevas — savoir ce que celui-ci porte tranche la question.
		filepath.Join(racine, "pc", "levels", "multi", "fo08_wetland", "fo08_wetland-rtx-new.module"),
		filepath.Join(racine, "pc", "levels", "multi", "fo11_blank", "fo11_blank-rtx-new.module"),
	}
	for _, module := range append(cibles, partages...) {
		chemin := module
		if !strings.HasSuffix(module, ".module") {
			var ok bool
			chemin, ok = ChercheModuleInstalle(module)
			if !ok {
				t.Logf("%-26s module introuvable", module)
				continue
			}
		}
		m, err := himodule.Open(chemin)
		if err != nil {
			t.Logf("%-26s ouverture KO : %v", module, err)
			continue
		}
		parGroupe := map[string]int{}
		for _, f := range m.Files("") {
			parGroupe[f.Group]++
		}
		groupes := make([]string, 0, len(parGroupe))
		for g := range parGroupe {
			groupes = append(groupes, g)
		}
		sort.Slice(groupes, func(i, j int) bool { return parGroupe[groupes[i]] > parGroupe[groupes[j]] })
		if len(groupes) > 12 {
			groupes = groupes[:12]
		}
		var detail []string
		for _, g := range groupes {
			detail = append(detail, fmt.Sprintf("%s=%d", g, parGroupe[g]))
		}
		t.Logf("%-30s %d fichiers, %d groupes — %s",
			filepath.Base(module), len(m.Files("")), len(parGroupe), strings.Join(detail, " "))
	}
}

// TestLevlLiveFireDesigneQuoi — LE TALON POINTE-T-IL AILLEURS ?
//
// L'inventaire montre que `sgh_interlock` ne porte QUE six fichiers dont un `levl`. Si la
// carte est jouable — et elle l'est, l'utilisateur y joue —, sa geometrie est ailleurs. Ce
// test extrait le `levl` et en tire les chaines lisibles : c'est la que se nomment les
// scenarios et les zones. Un nom de dossier autre que `sgh_interlock` y designerait le
// module reellement porteur.
func TestLevlLiveFireDesigneQuoi(t *testing.T) {
	if _, err := DeployRoot(); err != nil {
		t.Skip(err)
	}
	for _, module := range []string{"live_fire_sgh_interlock", "recharge_sgh_blueprint"} {
		chemin, ok := ChercheModuleInstalle(module)
		if !ok {
			continue
		}
		m, err := himodule.Open(chemin)
		if err != nil {
			t.Logf("%-26s ouverture KO : %v", module, err)
			continue
		}
		for _, f := range m.Files("levl") {
			blob, err := m.Extract(f)
			if err != nil {
				t.Logf("%-26s extraction levl KO : %v", module, err)
				continue
			}
			t.Logf("%-26s levl de %d octets — chaines : %s",
				module, len(blob), strings.Join(chainesLisibles(blob, 6, 24), " | "))
		}
	}
}

// chainesLisibles rend les suites d'ASCII imprimables d'au moins minLen caracteres, au plus
// maxOut d'entre elles.
func chainesLisibles(b []byte, minLen, maxOut int) []string {
	var out []string
	cur := make([]byte, 0, 64)
	vide := func() {
		if len(cur) >= minLen && len(out) < maxOut {
			out = append(out, string(cur))
		}
		cur = cur[:0]
	}
	for _, c := range b {
		if c >= 0x20 && c < 0x7f {
			cur = append(cur, c)
			continue
		}
		vide()
	}
	vide()
	return out
}

// TestGeometrieLiveFireDansCommon — L'HYPOTHESE DE L'UTILISATEUR, MISE A L'EPREUVE.
//
// « Live Fire j'y joue regulierement donc c'est bizarre, ce doit etre une variante d'une autre
// map et le poids leger doit etre la diff. » L'inventaire lui donne raison sur le fait : le
// module de la carte ne porte QU'UN levl de 2,3 Mo et cinq fichiers sans groupe — aucune
// geometrie. Or `common-rtx-new.module` porte QUATRE sbsp, ce qu'aucune carte n'explique.
//
// Le depart se fait par l'emprise, l'oracle deja employe par TestDiagnosticLiveFire : si l'un
// de ces quatre bsp contient les ancres d'objectif de Live Fire, c'est sa geometrie.
func TestGeometrieLiveFireDansCommon(t *testing.T) {
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	lo := [3]float64{math.Inf(1), math.Inf(1), math.Inf(1)}
	hi := [3]float64{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
	n := 0
	for _, e := range ancresDuModule(t, "live_fire_sgh_interlock") {
		for _, o := range e.Objectives {
			n++
			for k, v := range [3]float64{o.Pos.X, o.Pos.Y, o.Pos.Z} {
				lo[k] = math.Min(lo[k], v)
				hi[k] = math.Max(hi[k], v)
			}
		}
	}
	t.Logf("Live Fire — %d ancres  X [%+7.1f ; %+7.1f]  Y [%+7.1f ; %+7.1f]  Z [%+7.1f ; %+7.1f]",
		n, lo[0], hi[0], lo[1], hi[1], lo[2], hi[2])

	for _, nom := range []string{"common-rtx-new.module", "multiplayer-rtx-new.module"} {
		chemin := filepath.Join(racine, "pc", "globals", nom)
		bsps, err := ReadModuleInstances(chemin)
		if err != nil {
			t.Logf("%s : lecture KO : %v", nom, err)
			continue
		}
		for i, b := range bsps {
			dedans := lo[0] >= b.Bounds.Min[0] && hi[0] <= b.Bounds.Max[0] &&
				lo[1] >= b.Bounds.Min[1] && hi[1] <= b.Bounds.Max[1]
			t.Logf("%s bsp %d  X [%+8.1f ; %+8.1f]  Y [%+8.1f ; %+8.1f]  %5d instances  contient les ancres : %v",
				nom, i, b.Bounds.Min[0], b.Bounds.Max[0], b.Bounds.Min[1], b.Bounds.Max[1],
				len(b.Instances), dedans)
		}
	}
}
