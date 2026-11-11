//go:build research

package divergence

// rapport.go : les comptes d un film et de l ensemble, et les deux sorties lisibles :
// `divergence_resume.md` (par film) et `resume_global.md` (les dix films, par build).

import (
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
)

const (
	exemplesParFilm = 6
	tailleControle  = 100
	graineControle  = 20261110
	premiersRangs   = 15
)

// Les quatre causes d arret principales de notre marche (noms de la carte de fermeture).
var causesPrincipales = []string{
	grammar.InvariantSortieParRejet.String(),
	grammar.CauseHorsCadre,
	grammar.CauseListeNonLocalisee,
	"vue C : kind non porte",
}

type config struct {
	racine, temoin, sortie, niveau string
	carte                          *profile.MapQuantCatalog // nil : sans carte (voir ouvrir.go)
}

// cleClassement : un premier desaccord, range par la cause d arret de notre marche et le lecteur.
type cleClassement struct{ build, cause, vue, lecteur string }

// candidatControle : un paquet que les deux lectures ferment (sans reprise du temoin).
type candidatControle struct {
	film          string
	chunk, paquet int
	desaccord     bool
	detail        string
}

// agregat : ce que tous les films reunis ont rendu.
type agregat struct {
	classement map[cleClassement]int
	decalages  map[string]int
	films      []*statsFilm
	// controle : echantillon uniforme (reservoir) de tailleControle candidats, et le compte de tous.
	controle                        []candidatControle
	candidats, candidatsEnDesaccord int
	rng                             *rand.Rand
}

func nouvelAgregat() *agregat {
	return &agregat{classement: map[cleClassement]int{}, decalages: map[string]int{}, rng: rand.New(rand.NewPCG(graineControle, 1))}
}

// candidat tire au sort (algorithme R) un paquet pour le controle de l instrument.
func (g *agregat) candidat(c candidatControle) {
	g.candidats++
	if c.desaccord {
		g.candidatsEnDesaccord++
	}
	if len(g.controle) < tailleControle {
		g.controle = append(g.controle, c)
		return
	}
	if k := g.rng.IntN(g.candidats); k < tailleControle {
		g.controle[k] = c
	}
}

// statsFilm : les comptes d un film.
type statsFilm struct {
	id, build                                     string
	paquets, nonMarches, sansTemoin, temoinSeul   int
	fenetreKO, verdictsDiscordants                int
	nousFermes, temoinFermes, deuxFermes          int
	deuxFermesEnDesaccord, nousSeuls, temoinSeuls int
	nonFermesSansDesaccord                        int
	classement                                    map[cleClassement]int
	nousSeulsParTemoin                            map[string]int
	decalages                                     map[string]int
	exemplesRejet                                 []string
}

func (f *statsFilm) compter(n paquetNous, t *paquetTemoin, d, suite desaccord, recupere, fenetreOK bool, g *agregat) {
	f.paquets++
	if recupere {
		f.decalages[d.lecteur]++
		g.decalages[d.lecteur]++
	}
	if !fenetreOK {
		f.fenetreKO++
	}
	if (n.p.Fermeture.Verdict == 1) != n.fermee {
		f.verdictsDiscordants++
	}
	if n.fermee {
		f.nousFermes++
	}
	if t.ferme() {
		f.temoinFermes++
	}
	switch {
	case n.fermee && t.ferme():
		f.deuxFermes++
		if !d.aucun() {
			f.deuxFermesEnDesaccord++
		}
		if t.reprises == 0 {
			g.candidat(candidatControle{film: f.id, chunk: n.p.Chunk, paquet: n.p.Index, desaccord: !d.aucun(),
				detail: fmt.Sprintf("%s / %s : %s | %s", d.vue, d.lecteur, d.luNous, d.luTemoin)})
		}
	case n.fermee:
		f.nousSeuls++
		cle := t.stop
		if t.repriseDepart >= 0 {
			cle += " + reprise " + t.repriseCause
		}
		f.nousSeulsParTemoin[cle]++
	case t.ferme():
		f.temoinSeuls++
	}
	if n.fermee {
		return
	}
	d = effectif(d, suite, recupere)
	k := cleClassement{build: f.build, cause: n.cause, vue: d.vue, lecteur: d.lecteur}
	if d.aucun() {
		k.vue, k.lecteur = "-", "aucun desaccord de position, le temoin ferme"
		if !t.ferme() {
			k.lecteur = "aucun desaccord de position, le temoin ne ferme pas non plus (" + t.stop + ")"
		}
		f.nonFermesSansDesaccord++
	}
	f.classement[k]++
	g.classement[k]++
}

func exempleRejet(id string, n paquetNous, d desaccord) string {
	return fmt.Sprintf("%s\t%d\t%d\t%d\t%d\t%s\t%s", id, n.p.Chunk, n.p.Index, d.bitNous, d.bitTemoin, nettoyer(d.luNous), nettoyer(d.luTemoin))
}

// ecrire range les sorties d un film : divergence_resume.md et les exemples de rejet.
func (f *statsFilm) ecrire(dir string, g *agregat) error {
	g.films = append(g.films, f)
	var b strings.Builder
	fmt.Fprintf(&b, "# Divergence de lecture contre le temoin : film %s (%s)\n\n", f.id, f.build)
	f.ecrireComptes(&b)
	fmt.Fprintf(&b, "\n## Premiers desaccords, paquets non fermes chez nous, par lecteur en cause\n\n")
	tableau(&b, f.classement, "", premiersRangs)
	b.WriteString("\n## Decalages de frontiere rattrapes (premier desaccord recupere au record suivant)\n\n")
	liste(&b, f.decalages)
	fmt.Fprintf(&b, "\n## A part : paquets que nous fermons et que le temoin ne lit pas jusqu'au bout (%d)\n\n", f.nousSeuls)
	for _, kv := range trier(f.nousSeulsParTemoin) {
		fmt.Fprintf(&b, "- %d : arret ou reprise du temoin = %s\n", kv.n, kv.k)
	}
	if err := os.WriteFile(filepath.Join(dir, "divergence_resume.md"), []byte(b.String()), 0o644); err != nil { //nolint:gosec // sortie de l operateur
		return err
	}
	if len(f.exemplesRejet) > 0 {
		txt := "film\tchunk\tpaquet\tbit_nous\tbit_temoin\tlu_nous\tlu_temoin\n" + strings.Join(f.exemplesRejet, "\n") + "\n"
		return os.WriteFile(filepath.Join(dir, "divergence_exemples_rejet.tsv"), []byte(txt), 0o644) //nolint:gosec // idem
	}
	return nil
}

func (f *statsFilm) ecrireComptes(b *strings.Builder) {
	fmt.Fprintf(b, "- paquets compares : %d (non marches par le temoin, moins de 2 octets : %d ; sans ligne temoin : %d ; temoin seul : %d)\n",
		f.paquets, f.nonMarches, f.sansTemoin, f.temoinSeul)
	fmt.Fprintf(b, "- appariement : taille ou horodatage differents sur %d paquets ; verdicts des deux marches differents sur %d\n", f.fenetreKO, f.verdictsDiscordants)
	fmt.Fprintf(b, "- fermes chez nous : %d ; fermes chez le temoin : %d ; fermes par les deux : %d (dont en desaccord de position : %d)\n",
		f.nousFermes, f.temoinFermes, f.deuxFermes, f.deuxFermesEnDesaccord)
	fmt.Fprintf(b, "- fermes chez nous seulement : %d ; chez le temoin seulement : %d\n", f.nousSeuls, f.temoinSeuls)
	fmt.Fprintf(b, "- non fermes chez nous sans aucun desaccord de position : %d\n", f.nonFermesSansDesaccord)
}

type compte struct {
	k string
	n int
}

func trier(m map[string]int) []compte {
	out := make([]compte, 0, len(m))
	for k, n := range m {
		out = append(out, compte{k, n})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].n > out[j].n || out[i].n == out[j].n && out[i].k < out[j].k })
	return out
}

// tableau ecrit les `max` premiers lecteurs en cause (vue + lecteur) de la cause donnee ("" : toutes),
// restreints au build (cle.build) quand le classement porte plusieurs builds.
func tableau(b *strings.Builder, m map[cleClassement]int, cause string, max int) {
	agr := map[string]int{}
	total := 0
	for k, n := range m {
		if cause != "" && k.cause != cause {
			continue
		}
		agr[k.vue+" | "+k.lecteur] += n
		total += n
	}
	fmt.Fprintf(b, "Total : %d paquets\n\n| rang | paquets | part | vue | lecteur en cause |\n|---|---|---|---|---|\n", total)
	for i, kv := range trier(agr) {
		if i >= max {
			break
		}
		vue, lecteur, _ := strings.Cut(kv.k, " | ")
		fmt.Fprintf(b, "| %d | %d | %.1f %% | %s | %s |\n", i+1, kv.n, 100*float64(kv.n)/float64(max0(total)), vue, lecteur)
	}
}

// liste ecrit un classement de lecteurs, du plus frequent au moins frequent.
func liste(b *strings.Builder, m map[string]int) {
	if len(m) == 0 {
		b.WriteString("aucun\n")
	}
	for _, kv := range trier(m) {
		fmt.Fprintf(b, "- %d : %s\n", kv.n, kv.k)
	}
}

func max0(n int) int {
	if n < 1 {
		return 1
	}
	return n
}

// ecrireGlobal ecrit resume_global.md : population, controle de l instrument, classements par build.
func (g *agregat) ecrireGlobal(dir string) error {
	var b strings.Builder
	b.WriteString("# Divergence de lecture contre le temoin : dix films\n\n## Films\n\n")
	b.WriteString("| film | build | compares | fermes nous | fermes temoin | fermes les deux | en desaccord | nous seuls | temoin seul | appariement KO | verdicts differents |\n|---|---|---|---|---|---|---|---|---|---|---|\n")
	for _, f := range g.films {
		fmt.Fprintf(&b, "| %s | %s | %d | %d | %d | %d | %d | %d | %d | %d | %d |\n", f.id, f.build, f.paquets, f.nousFermes, f.temoinFermes,
			f.deuxFermes, f.deuxFermesEnDesaccord, f.nousSeuls, f.temoinSeuls, f.fenetreKO, f.verdictsDiscordants)
	}
	b.WriteString("\n## Decalages de frontiere entre composants rattrapes au record suivant (tous les paquets)\n\n")
	liste(&b, g.decalages)
	b.WriteString("\n## Controle de l'instrument\n\n")
	enDesaccord := 0
	for _, c := range g.controle {
		if c.desaccord {
			enDesaccord++
		}
	}
	fmt.Fprintf(&b, "- echantillon : %d paquets tires au hasard (graine %d) parmi les %d fermes par les deux lectures sans reprise du temoin : %d desaccords de position\n",
		len(g.controle), graineControle, g.candidats, enDesaccord)
	fmt.Fprintf(&b, "- population entiere de ces paquets : %d desaccords sur %d\n", g.candidatsEnDesaccord, g.candidats)
	for _, c := range g.controle {
		if c.desaccord {
			fmt.Fprintf(&b, "  - %s %d/%d : %s\n", c.film, c.chunk, c.paquet, c.detail)
		}
	}
	builds := map[string]bool{}
	for _, f := range g.films {
		builds[f.build] = true
	}
	noms := []string{""}
	for k := range builds {
		noms = append(noms, k)
	}
	sort.Strings(noms[1:])
	for _, build := range noms {
		titre := "Tous builds"
		if build != "" {
			titre = "Build " + build
		}
		fmt.Fprintf(&b, "\n## %s : premiers desaccords par lecteur en cause\n", titre)
		sous := map[cleClassement]int{}
		for k, n := range g.classement {
			if build == "" || k.build == build {
				sous[k] += n
			}
		}
		for _, c := range causesPrincipales {
			fmt.Fprintf(&b, "\n### Cause d'arret : %s\n\n", c)
			tableau(&b, sous, c, premiersRangs)
		}
		fmt.Fprintf(&b, "\n### Autres causes d'arret\n\n")
		autres := map[cleClassement]int{}
		for k, n := range sous {
			if !contient(causesPrincipales, k.cause) {
				autres[k] += n
			}
		}
		agr := map[string]int{}
		for k, n := range autres {
			agr[k.cause] += n
		}
		for _, kv := range trier(agr) {
			fmt.Fprintf(&b, "- %d : %s\n", kv.n, kv.k)
		}
	}
	return os.WriteFile(filepath.Join(dir, "resume_global.md"), []byte(b.String()), 0o644) //nolint:gosec // sortie de l operateur
}

func contient(l []string, s string) bool {
	for _, x := range l {
		if x == s {
			return true
		}
	}
	return false
}
