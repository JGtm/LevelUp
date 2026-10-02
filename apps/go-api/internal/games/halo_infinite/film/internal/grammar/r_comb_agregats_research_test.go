//go:build research

package grammar

// r_comb_agregats_research_test.go — compagnon de `r_comb_research_test.go` (RECHERCHE R-COMB,
// 2026-10-02) : les agregats. Lit la table brute `r_comb_configs.tsv` et les TSV des mesures de
// la campagne (pour le denominateur FIXE), ecrit les tables par build, par film et les
// contributions marginales. Aucune marche, aucun film lu : de l arithmetique sur des TSV.
//
//	CAMPAGNE_RCOMB=<r_comb_configs.tsv> CAMPAGNE_TSV=<campagne_grammaire_2026-10-01> \
//	CAMPAGNE_HORS_CORPUS=81c02726 CAMPAGNE_SORTIE=<dir hors data> \
//	  go test -tags=research -count=1 -run '^TestRCombAgregats$' ./internal/games/halo_infinite/film/internal/grammar/
//
// Denominateurs (PLAN §6.0 « Pourcentages ») :
//
//	variable  records utiles lus par la marche elle-meme ;
//	fixe      par film, le maximum des records utiles lus sur { les configurations R-COMB, les 14
//	          marches de bis 1, les marches de bis 3 hors `largeur-*`, les marches `ti=3` hors
//	          temoin `ti=4`, les marches `ti=43`, les A/B de position (instruments et
//	          production) } ; les temoins NEGATIFS (largeur fausse, `ti=4` a 26 bits) sont exclus ;
//	fixe14    le maximum des 14 marches de bis 1 seules (celui du RAPPORT §3), pour continuite.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// rcaLigne : une ligne de `r_comb_configs.tsv` (colonnes nommees).
type rcaLigne map[string]string

func (l rcaLigne) n(col string) int {
	v, _ := strconv.Atoi(l[col])
	return v
}

// rcaLire lit un TSV a en-tete.
func rcaLire(t *testing.T, chemin string) []rcaLigne {
	t.Helper()
	brut, err := os.ReadFile(chemin) //nolint:gosec // chemin d instrument
	if err != nil {
		t.Fatalf("%s : %v", chemin, err)
	}
	ls := strings.Split(strings.TrimRight(strings.ReplaceAll(string(brut), "\r", ""), "\n"), "\n")
	tete := strings.Split(ls[0], "\t")
	var out []rcaLigne
	for _, l := range ls[1:] {
		c := strings.Split(l, "\t")
		m := rcaLigne{}
		for i, h := range tete {
			if i < len(c) {
				m[h] = c[i]
			}
		}
		out = append(out, m)
	}
	return out
}

// rcaOrdreDesBuilds : ordre de publication.
var rcaOrdreDesBuilds = []string{"HI_1_13_0", "HI_1_12_0", "HI_1_11_0", "HI_1_10_0", "HI_1_9_0", "HI_1_8_0",
	"HI_1_4_1", "version-33", "version-31"}

func rcaRang(b string) int {
	for i, x := range rcaOrdreDesBuilds {
		if x == b {
			return i
		}
	}
	return len(rcaOrdreDesBuilds)
}

// rcaSomme : les comptes d une configuration sommes sur un groupe de films.
type rcaSomme struct {
	paquets, fermes, sains, utiles, utilesSains, lus, fixe, fixe14, hc int
	gRef, pRef, gsRef, psRef                                           int
	entrees, entreesSaines                                             int
}

func (s *rcaSomme) ajouter(l rcaLigne, fixe, fixe14 int) {
	s.paquets += l.n("paquets")
	s.fermes += l.n("fermes")
	s.sains += l.n("fermes_sains")
	s.utiles += l.n("utiles_fermes")
	s.utilesSains += l.n("utiles_fermes_sains")
	s.lus += l.n("utiles_lus")
	s.hc += l.n("hors_cadre")
	s.fixe += fixe
	s.fixe14 += fixe14
	s.gRef += l.n("g_ref")
	s.pRef += l.n("p_ref")
	s.gsRef += l.n("gs_ref")
	s.psRef += l.n("ps_ref")
	s.entrees += l.n("entrees_utiles_fermees")
	s.entreesSaines += l.n("entrees_utiles_saines")
}

func rcaPct(a, b int) string {
	if b == 0 {
		return "-"
	}
	return strconv.FormatFloat(100*float64(a)/float64(b), 'f', 1, 64)
}

// rcaFixes : par film, le denominateur fixe (et sa source) sur les TSV de la campagne et les
// configurations R-COMB, et le maximum des 14 marches de bis 1.
func rcaFixes(t *testing.T, dir string, rc []rcaLigne) (fixe, fixe14 map[string]int, src map[string]string) {
	fixe, fixe14, src = map[string]int{}, map[string]int{}, map[string]string{}
	prendre := func(source string, ls []rcaLigne, garder func(rcaLigne) bool) {
		for _, l := range ls {
			if garder != nil && !garder(l) {
				continue
			}
			if v := l.n("utiles_lus"); v > fixe[l["film"]] {
				fixe[l["film"]], src[l["film"]] = v, source+" : "+l["variante"]+l["config"]
			}
		}
	}
	bis1 := rcaLire(t, filepath.Join(dir, "mesures_bis_tsv", "mb_variantes.tsv"))
	for _, l := range bis1 {
		if v := l.n("utiles_lus"); v > fixe14[l["film"]] {
			fixe14[l["film"]] = v
		}
	}
	prendre("bis1", bis1, nil)
	prendre("bis3", rcaLire(t, filepath.Join(dir, "mesures_bis3_tsv", "mb3_variantes.tsv")),
		func(l rcaLigne) bool { return !strings.HasPrefix(l["variante"], "largeur-") })
	prendre("bis3 ti3", rcaLire(t, filepath.Join(dir, "mesures_bis3_tsv", "mb3_ti3.tsv")),
		func(l rcaLigne) bool { return !strings.HasPrefix(l["variante"], "ti4") })
	prendre("bis2 ti43", rcaLire(t, filepath.Join(dir, "mesures_bis2_tsv", "mb2_ti43.tsv")), nil)
	prendre("bis2 positions", rcaLire(t, filepath.Join(dir, "mesures_bis2_tsv", "mb2_positions.tsv")), nil)
	prendre("bis2 positions production", rcaLire(t, filepath.Join(dir, "mesures_bis2_tsv", "mb2_positions_production.tsv")), nil)
	prendre("r-comb", rc, nil)
	return fixe, fixe14, src
}

// TestRCombAgregats ecrit les agregats de R-COMB.
func TestRCombAgregats(t *testing.T) {
	chemin, dir, sortie := os.Getenv("CAMPAGNE_RCOMB"), os.Getenv("CAMPAGNE_TSV"), os.Getenv("CAMPAGNE_SORTIE")
	if chemin == "" || dir == "" || sortie == "" {
		t.Skip("CAMPAGNE_RCOMB, CAMPAGNE_TSV et CAMPAGNE_SORTIE requis")
	}
	hors := map[string]bool{}
	for _, x := range strings.Split(os.Getenv("CAMPAGNE_HORS_CORPUS"), ",") {
		hors[strings.TrimSpace(x)] = true
	}
	rc := rcaLire(t, chemin)
	if d := os.Getenv("CAMPAGNE_RCOMB_DERIVATION"); d != "" {
		rc = append(rc, rcaDerivation(t, d)...)
	}
	fixe, fixe14, src := rcaFixes(t, dir, rc)
	var films, configs []string
	vu, vuC := map[string]bool{}, map[string]bool{}
	build := map[string]string{}
	par := map[[2]string]rcaLigne{}
	for _, l := range rc {
		if !vu[l["film"]] {
			vu[l["film"]] = true
			films = append(films, l["film"])
		}
		if !vuC[l["config"]] {
			vuC[l["config"]] = true
			configs = append(configs, l["config"])
		}
		build[l["film"]] = l["build"]
		par[[2]string{l["film"], l["config"]}] = l
	}
	sort.Strings(films)
	sort.Strings(configs)
	// Denominateurs par film.
	den := []string{"film\tbuild\tdans_le_corpus\tfixe\tsource_du_fixe\tfixe14_bis1\tutiles_lus_reference\tutiles_lus_full"}
	for _, f := range films {
		den = append(den, fmt.Sprintf("%s\t%s\t%v\t%d\t%s\t%d\t%d\t%d", f, build[f], !hors[f], fixe[f], src[f], fixe14[f],
			par[[2]string{f, "reference"}].n("utiles_lus"), par[[2]string{f, "full"}].n("utiles_lus")))
	}
	b2Ecrire(t, sortie, "r_comb_denominateurs.tsv", den)
	// Sommes par groupe (build, corpus, hors corpus) et par configuration.
	groupes := map[string]map[string]*rcaSomme{}
	ajouter := func(g, c string, l rcaLigne, f string) {
		if groupes[g] == nil {
			groupes[g] = map[string]*rcaSomme{}
		}
		if groupes[g][c] == nil {
			groupes[g][c] = &rcaSomme{}
		}
		groupes[g][c].ajouter(l, fixe[f], fixe14[f])
	}
	for _, f := range films {
		for _, c := range configs {
			l, ok := par[[2]string{f, c}]
			if !ok {
				continue
			}
			if hors[f] {
				ajouter("hors corpus : "+f+" ("+build[f]+")", c, l, f)
				continue
			}
			ajouter(build[f], c, l, f)
			ajouter("corpus (20 films)", c, l, f)
		}
	}
	var gs []string
	for g := range groupes {
		gs = append(gs, g)
	}
	sort.Slice(gs, func(i, j int) bool {
		ri, rj := rcaRang(gs[i]), rcaRang(gs[j])
		if ri != rj {
			return ri < rj
		}
		return gs[i] < gs[j]
	})
	pb := []string{"groupe\tconfig\tpaquets\tfermes\tfermes_sains\tutiles_fermes\tutiles_fermes_sains\tutiles_lus\tfixe\tfixe14\t" +
		"pct_variable_brut\tpct_variable_sains\tpct_fixe_brut\tpct_fixe_sains\tpct_fixe14_sains\thors_cadre\tg_ref\tp_ref\tgs_ref\tps_ref\t" +
		"delta_sains_vs_ref\tdelta_utiles_sains_vs_ref\tentrees_utiles_fermees\tentrees_utiles_saines"}
	for _, g := range gs {
		ref := groupes[g]["reference"]
		for _, c := range configs {
			s := groupes[g][c]
			if s == nil || ref == nil {
				continue
			}
			pb = append(pb, fmt.Sprintf("%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%s\t%s\t%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%+d\t%+d\t%d\t%d",
				g, c, s.paquets, s.fermes, s.sains, s.utiles, s.utilesSains, s.lus, s.fixe, s.fixe14,
				rcaPct(s.utiles, s.lus), rcaPct(s.utilesSains, s.lus), rcaPct(s.utiles, s.fixe), rcaPct(s.utilesSains, s.fixe),
				rcaPct(s.utilesSains, s.fixe14), s.hc, s.gRef, s.pRef, s.gsRef, s.psRef,
				s.sains-ref.sains, s.utilesSains-ref.utilesSains, s.entrees, s.entreesSaines))
		}
	}
	b2Ecrire(t, sortie, "r_comb_par_build.tsv", pb)
	b2Ecrire(t, sortie, "r_comb_marginaux.tsv", rcaMarginaux(gs, groupes))
	b2Ecrire(t, sortie, "r_comb_par_film.tsv", rcaParFilm(films, build, hors, par, fixe))
}

// rcaDerivation lit `r_comb_derivation.tsv` (interaction L1 x L9) et en rend deux configurations
// de plus : la combinaison complete et la paire L1+L9, l oracle L1 etant DERIVE SANS L9.
func rcaDerivation(t *testing.T, chemin string) []rcaLigne {
	var out []rcaLigne
	for _, l := range rcaLire(t, chemin) {
		if l["config"] != "L1(sans L9)+L9" {
			continue
		}
		m := rcaLigne{}
		for k, v := range l {
			m[k] = v
		}
		m["config"] = rcaFullSansL9
		if l["composants"] == "aucun" {
			m["config"] = "L1+L9 (L1 derive sans L9)"
		}
		out = append(out, m)
	}
	return out
}

// rcaFullSansL9 : la combinaison complete dont l oracle L1 est derive sans L9.
const rcaFullSansL9 = "full (L1 derive sans L9)"

// rcaLeviers : levier ->(configuration « seul », configuration « combinaison privee du levier »).
var rcaLeviers = [][3]string{{"L1", "L1", "full-L1"}, {"L8", "L8", "full-L8"}, {"L2", "L2", "full-L2"},
	{"L9", "L9", "full-L9"}, {"L6a", "L6a", "full-L6a"}, {"L6b", "L6b", "full-L6b"}}

// rcaMarginaux : par groupe et par levier, le gain seul (X - reference), le gain marginal dans la
// combinaison (full - (full sans X)) et l interaction (marginal - seul), en paquets sains, records
// utiles fermes sains et paquets fermes bruts. Pour la combinaison dont l oracle L1 est derive sans
// L9, seuls L1 et L9 ont leur combinaison privee mesuree (`full-L1`, `full-L9` : memes oracles).
func rcaMarginaux(gs []string, groupes map[string]map[string]*rcaSomme) []string {
	out := []string{"groupe\tcombinaison\tlevier\tseul_sains\tseul_utiles_sains\tseul_fermes_brut\tmarginal_sains\t" +
		"marginal_utiles_sains\tmarginal_fermes_brut\tinteraction_sains\tinteraction_utiles_sains\tmarginal_pts_fixe_sains"}
	for _, g := range gs {
		for _, comb := range []struct {
			nom     string
			leviers [][3]string
		}{{"full", rcaLeviers}, {rcaFullSansL9, rcaLeviers[:4]}} {
			out = append(out, rcaMarginauxDe(g, comb.nom, comb.leviers, groupes[g])...)
		}
	}
	return out
}

// rcaMarginauxDe : les lignes d une combinaison d un groupe.
func rcaMarginauxDe(g, nomFull string, leviers [][3]string, m map[string]*rcaSomme) []string {
	ref, full := m["reference"], m[nomFull]
	if ref == nil || full == nil {
		return nil
	}
	var out []string
	var sSains, sUtiles, sBrut int
	for _, lv := range leviers {
		x, sans := m[lv[1]], m[lv[2]]
		if x == nil || sans == nil || (nomFull != "full" && lv[0] != "L1" && lv[0] != "L9") {
			continue
		}
		ss, su, sb := x.sains-ref.sains, x.utilesSains-ref.utilesSains, x.fermes-ref.fermes
		ms, mu, mb := full.sains-sans.sains, full.utilesSains-sans.utilesSains, full.fermes-sans.fermes
		sSains, sUtiles, sBrut = sSains+ss, sUtiles+su, sBrut+sb
		pts := "-"
		if full.fixe > 0 {
			pts = strconv.FormatFloat(100*float64(mu)/float64(full.fixe), 'f', 2, 64)
		}
		out = append(out, fmt.Sprintf("%s\t%s\t%s\t%+d\t%+d\t%+d\t%+d\t%+d\t%+d\t%+d\t%+d\t%s", g, nomFull, lv[0], ss, su, sb,
			ms, mu, mb, ms-ss, mu-su, pts))
	}
	if nomFull != "full" {
		return out
	}
	return append(out, fmt.Sprintf("%s\t%s\tsomme des seuls\t%+d\t%+d\t%+d\t\t\t\t\t\t", g, nomFull, sSains, sUtiles, sBrut),
		fmt.Sprintf("%s\t%s\tcombinaison complete (full - reference)\t%+d\t%+d\t%+d\t\t\t\t%+d\t%+d\t", g, nomFull,
			full.sains-ref.sains, full.utilesSains-ref.utilesSains, full.fermes-ref.fermes,
			(full.sains-ref.sains)-sSains, (full.utilesSains-ref.utilesSains)-sUtiles))
}

// rcaParFilm : par film, la reference, la combinaison complete et chaque combinaison privee d un
// levier, contre la reference (critere du gate 2 : aucune baisse saine sur aucun film).
func rcaParFilm(films []string, build map[string]string, hors map[string]bool, par map[[2]string]rcaLigne,
	fixe map[string]int) []string {
	out := []string{"film\tbuild\tdans_le_corpus\tconfig\tfermes\tfermes_sains\tutiles_fermes_sains\tfixe\tpct_fixe_sains\t" +
		"delta_sains_vs_ref\tdelta_utiles_sains_vs_ref\tgs_ref\tps_ref\tg_ref\tp_ref\tbaisse_saine"}
	cs := []string{"reference", "full", rcaFullSansL9, "full-L1", "full-L8", "full-L2", "full-L9", "full-L6a", "full-L6b",
		"L1", "L8", "L2", "L9", "L6a", "L6b", "L1+L9", "L1+L9 (L1 derive sans L9)", "full-L1-L9 (composants)"}
	for _, f := range films {
		ref := par[[2]string{f, "reference"}]
		for _, c := range cs {
			l, ok := par[[2]string{f, c}]
			if !ok {
				continue
			}
			ds, du := l.n("fermes_sains")-ref.n("fermes_sains"), l.n("utiles_fermes_sains")-ref.n("utiles_fermes_sains")
			baisse := "non"
			if l.n("ps_ref") > 0 || du < 0 {
				baisse = fmt.Sprintf("oui (%d paquets sains perdus, %+d utiles sains)", l.n("ps_ref"), du)
			}
			out = append(out, fmt.Sprintf("%s\t%s\t%v\t%s\t%d\t%d\t%d\t%d\t%s\t%+d\t%+d\t%d\t%d\t%d\t%d\t%s", f, build[f], !hors[f], c,
				l.n("fermes"), l.n("fermes_sains"), l.n("utiles_fermes_sains"), fixe[f], rcaPct(l.n("utiles_fermes_sains"), fixe[f]),
				ds, du, l.n("gs_ref"), l.n("ps_ref"), l.n("g_ref"), l.n("p_ref"), baisse))
		}
	}
	return out
}
