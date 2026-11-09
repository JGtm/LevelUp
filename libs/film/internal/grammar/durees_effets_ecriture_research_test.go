//go:build research

package grammar

// durees_effets_ecriture_research_test.go — LES SORTIES de l instrument `durees_effets_research_test.go` : les TSV
// (usages, candidats, fenetres, ti_extra, archetypes) et la surveillance de la memoire. DEPLACEMENT PUR depuis ce
// fichier pour tenir le seuil de 500 lignes : aucune ligne de logique n a change.

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

// deMemoire echantillonne le tas et la memoire obtenue de l OS pendant l instrument.
type deMemoire struct {
	stop, done     chan struct{}
	maxTas, maxSys uint64
}

func deSurveillerLaMemoire() *deMemoire {
	m := &deMemoire{stop: make(chan struct{}), done: make(chan struct{})}
	go func() {
		defer close(m.done)
		tk := time.NewTicker(200 * time.Millisecond)
		defer tk.Stop()
		for {
			var s runtime.MemStats
			runtime.ReadMemStats(&s)
			m.maxTas, m.maxSys = max(m.maxTas, s.HeapAlloc), max(m.maxSys, s.Sys)
			select {
			case <-m.stop:
				return
			case <-tk.C:
			}
		}
	}()
	return m
}

func (m *deMemoire) arreter() (tas, sys uint64) {
	close(m.stop)
	<-m.done
	return m.maxTas, m.maxSys
}
func envOu(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func deEcrire(t *testing.T, chemin string, entete string, ecrire func(w *bufio.Writer)) {
	f, err := os.Create(chemin)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	fmt.Fprintln(w, entete)
	ecrire(w)
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
}

// deEcrireUsages : une ligne par usage.
func deEcrireUsages(t *testing.T, out, id string, origine uint64, usages []*deUsage) {
	deEcrire(t, filepath.Join(out, id+".usages.tsv"),
		"film\tusage\tfamille\trang\tobjet\tporteur\tt_ms\tn_suivis\tsuivi_ti\tsuivi_nomme_ms\tsuivi_fin_ms\t"+
			"suivi_duree_ms\tcreation_x\tcreation_y\tcreation_z\trepos_etat\trepos_x\trepos_y\trepos_z",
		func(w *bufio.Writer) {
			for _, u := range usages {
				nomme, fin, duree, ti := "", "", "", ""
				cr, rp, etat := "\t\t", "\t\t", ""
				if len(u.suivis) > 0 {
					s := u.suivis[0]
					ti, nomme = strconv.Itoa(s.ti), fmt.Sprintf("%.1f", deMs(s.nomme, origine))
					if s.fin > 0 {
						fin = fmt.Sprintf("%.1f", deMs(s.fin, origine))
						duree = fmt.Sprintf("%.0f", float64(s.fin-s.nomme)/1000)
					}
					cr, etat = dePos(s.creation, s.aCreation), s.reposEtat
					rp = dePos(s.repos, s.reposEtat != "absent")
				}
				fmt.Fprintf(w, "%s\t%d\t%s\t%d\t%d\t%d\t%.1f\t%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", id, u.idx, u.famille,
					u.rang, u.objet.Slot, u.porteur, deMs(u.ts, origine), len(u.suivis), ti, nomme, fin, duree, cr,
					etat, rp)
			}
		})
}

// deEcrireCandidats : une ligne par (usage, instant, vie candidate).
func deEcrireCandidats(t *testing.T, out, id string, origine uint64, cands []deCandidat,
	pos map[deCleNaissance][3]float32) {
	deEcrire(t, filepath.Join(out, id+".candidats.tsv"),
		"film\tusage\tfamille\trole\tautres_usages\tti\tslot\tgen\toff_usage_ms\toff_fin_ms\tduree_ms\tx\ty\tz\t"+
			"dist_repos_3d\tdist_repos_h\tdist_creation_suivi_3d",
		func(w *bufio.Writer) {
			for _, c := range cands {
				v, u := c.vie, c.u
				decal := int64(0)
				switch c.role {
				case "temoin+":
					decal = deTemoinUS
				case "temoin-":
					decal = -deTemoinUS
				}
				offF, duree := "", "-1"
				var s *deSuivi
				if len(u.suivis) > 0 {
					s = u.suivis[0]
					if s.fin > 0 {
						offF = fmt.Sprintf("%.0f", float64(int64(v.ne)-int64(s.fin)-decal)/1000)
					}
				}
				if v.fin > 0 {
					duree = fmt.Sprintf("%.0f", float64(v.fin-v.ne)/1000)
				}
				p, aP := pos[deCleNaissance{v.slot, v.gen, v.ne}]
				dr3, drh, dc := "", "", ""
				if aP && s != nil && c.role == "reel" {
					if s.reposEtat != "absent" {
						d3, dh := deDist(p, s.repos)
						dr3, drh = fmt.Sprintf("%.2f", d3), fmt.Sprintf("%.2f", dh)
					}
					if s.aCreation {
						d3, _ := deDist(p, s.creation)
						dc = fmt.Sprintf("%.2f", d3)
					}
				}
				fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%d\t%d\t%d\t%d\t%.0f\t%s\t%s\t%s\t%s\t%s\n", id, u.idx, u.famille,
					c.role, c.autres, v.ti, v.slot, v.gen, float64(int64(v.ne)-int64(u.ts)-decal)/1000, offF, duree,
					dePos(p, aP), dr3+"\t"+drh, dc)
			}
		})
}

// deEcrireExtra : toutes les vies des archetypes extra, avec l usage et l objet suivi les plus proches.
func deEcrireExtra(t *testing.T, out, id string, origine uint64, vies []*cfVie, extra map[int16]bool,
	usages []*deUsage, suivis []*deSuivi, pos map[deCleNaissance][3]float32) {
	deEcrire(t, filepath.Join(out, id+".ti_extra.tsv"),
		"film\tti\tslot\tgen\tne_ms\tduree_ms\tx\ty\tz\tusage_proche\tfamille\toff_usage_ms\tsuivi_proche_nomme_off_ms\t"+
			"suivi_proche_fin_off_ms",
		func(w *bufio.Writer) {
			for _, v := range vies {
				if !extra[v.ti] {
					continue
				}
				duree := "-1"
				if v.fin > 0 {
					duree = fmt.Sprintf("%.0f", float64(v.fin-v.ne)/1000)
				}
				p, aP := pos[deCleNaissance{v.slot, v.gen, v.ne}]
				bu, bf, bn := -1, "?", int64(math.MaxInt64)
				for _, u := range usages {
					if d := int64(v.ne) - int64(u.ts); deAbs64(d) < deAbs64(bn) {
						bu, bf, bn = u.idx, u.famille, d
					}
				}
				offU := ""
				if bu >= 0 {
					offU = fmt.Sprintf("%.0f", float64(bn)/1000)
				}
				sn, sf := "", ""
				var mn, mf int64 = math.MaxInt64, math.MaxInt64
				for _, s := range suivis {
					if d := int64(v.ne) - int64(s.nomme); deAbs64(d) < deAbs64(mn) {
						mn = d
					}
					if s.fin > 0 {
						if d := int64(v.ne) - int64(s.fin); deAbs64(d) < deAbs64(mf) {
							mf = d
						}
					}
				}
				if mn != math.MaxInt64 {
					sn = fmt.Sprintf("%.0f", float64(mn)/1000)
				}
				if mf != math.MaxInt64 {
					sf = fmt.Sprintf("%.0f", float64(mf)/1000)
				}
				fmt.Fprintf(w, "%s\t%d\t%d\t%d\t%.1f\t%s\t%s\t%d\t%s\t%s\t%s\t%s\n", id, v.ti, v.slot, v.gen,
					deMs(v.ne, origine), duree, dePos(p, aP), bu, bf, offU, sn, sf)
			}
		})
}

func deAbs64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

// deEcrireArchetypes : les composants des archetypes vus parmi les candidats et les extras.
func deEcrireArchetypes(t *testing.T, out, id string, reg *Registry, tiVus map[int16]int) {
	tis := make([]int, 0, len(tiVus))
	for ti := range tiVus {
		tis = append(tis, int(ti))
	}
	sort.Ints(tis)
	deEcrire(t, filepath.Join(out, id+".archetypes.tsv"), "film\tti\tcandidats\tcomposants", func(w *bufio.Writer) {
		for _, ti := range tis {
			a, ok := reg.Archetype(ti)
			if !ok {
				continue
			}
			fmt.Fprintf(w, "%s\t%d\t%d\t%s\n", id, ti, tiVus[int16(ti)], strings.Join(a.Components, ","))
		}
	})
}

// deEcrireFenetres : une ligne par fenetre de recherche, avec ou sans candidat (le denominateur des
// parts de la synthese).
func deEcrireFenetres(t *testing.T, out, id string, fens []deFen) {
	deEcrire(t, filepath.Join(out, id+".fenetres.tsv"),
		"film\tusage\tfamille\trole\tautres_usages\tautres_famille\tcandidats\tfenetre_ms",
		func(w *bufio.Writer) {
			for _, f := range fens {
				fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%d\t%d\t%d\t%.0f\n", id, f.u.idx, f.u.famille, f.role, f.autres,
					f.autresRang, f.n, f.dureeMs)
			}
		})
}
