package main

// cmd_spawn_model_rapport.go — LES SECTIONS DE `spawn-model evaluate`. Chacune imprime des
// tableaux en Markdown, a recopier tels quels dans l'annexe des mesures.

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"sort"

	"levelup/go-api/internal/analysis/tactical"
	"levelup/go-api/internal/config"
)

// Valeurs de la mesure de depart (Assassin 4c4 hors Fiesta et BTB), contre lesquelles le temoin
// se compare.
const (
	refRayon1Hasard, refRayon1Freq, refRayon1Modele = 14.1, 15.6, 44.9
	refRayon3Freq, refRayon3Modele                  = 38.4, 62.5
	refTop1Freq, refTop3Freq                        = 4.5, 12.8
	refTop1Modele, refTop3Modele                    = 11.0, 27.7
	refReapparitions, refCartes                     = 43966, 45
	// ecartTemoinMax : l'ecart toléré au temoin, en points.
	ecartTemoinMax = 1.0
)

// familleTemoin est la famille de la mesure de depart : Assassin 4c4 hors Fiesta et BTB. C'est un
// identifiant de la table des familles du titre ; un titre qui ne le declare pas n'a pas de temoin.
const familleTemoin = "assassin"

// rapporteur porte ce que les sections partagent.
type rapporteur struct {
	ctx      context.Context
	cfg      *config.AppConfig
	o        spawnModelOptions
	jeu      *jeuReapparitions
	db       *sql.DB
	familles map[string]bool
	zones    map[string][]tactical.ZoneNommee
}

func imprimerBilan(w io.Writer, b bilanLecture) {
	_, _ = fmt.Fprintf(w, "## Bilan de lecture\n\n")
	_, _ = fmt.Fprintf(w, "| artefacts | lus | sans registre | hors familles | illisibles |\n|---|---|---|---|---|\n")
	_, _ = fmt.Fprintf(w, "| %d | %d | %d | %d | %d |\n\n", b.artefacts, b.lus, b.sansRegistre, b.horsFamille, b.echecs)
	_, _ = fmt.Fprintf(w, "| reapparitions (source film) | reapparitions (repli ecart 4-15 s) | debuts de manche | sans equipe | non finies |\n|---|---|---|---|---|\n")
	_, _ = fmt.Fprintf(w, "| %d | %d | %d | %d | %d |\n\n", b.reapFilm, b.reapRepli, b.debutsManche, b.sansEquipe, b.nonFinis)
	a := b.accord
	_, _ = fmt.Fprintf(w, "| documents lus au film | accord film et repli | film seul | repli seul | dont debut de manche au film | comptes a rebours sans joueur |\n|---|---|---|---|---|---|\n")
	_, _ = fmt.Fprintf(w, "| %d | %d | %d | %d | %d | %d |\n\n", a.documentsFilm, a.accord, a.filmSeul, a.repliSeul, a.repliSeulManche, b.comptesSansJoueur)
	fam := make([]string, 0, len(b.parFamille))
	for f := range b.parFamille {
		fam = append(fam, f)
	}
	sort.Strings(fam)
	for _, f := range fam {
		_, _ = fmt.Fprintf(w, "- %s : %d matchs lus\n", f, b.parFamille[f])
	}
	_, _ = fmt.Fprintln(w)
}

// resultat est ce que mesure une famille : le hasard, la frequence seule et le modele, en
// validation croisee par match.
// n, cartes et matchs se comptent sur les echantillons RETENUS (meme regle que poidsDeGroupe) :
// jamais sur les matchs lus avant filtrage.
type resultat struct {
	n, cartes    int
	matchs       int
	kMoyen       float64
	freq, modele tactical.Metriques
	modeleVue    *tactical.Metriques
}

// evaluerEchantillons note un jeu d'echantillons : frequence seule, modele a six variables et,
// si `avecVue`, modele a sept.
func evaluerEchantillons(echs []tactical.Echantillon, avecVue bool) resultat {
	r := resultat{n: len(echs)}
	cartes, matchs, k := map[string]bool{}, map[string]bool{}, 0.0
	for _, s := range echs {
		cartes[s.Carte], matchs[s.Match] = true, true
		k += float64(len(s.Feats))
	}
	r.cartes, r.matchs = len(cartes), len(matchs)
	if len(echs) == 0 {
		return r
	}
	r.kMoyen = k / float64(len(echs))
	r.freq = tactical.Evaluer(echs, tactical.ScoresValides(echs, tactical.FrequenceSeule))
	r.modele = tactical.Evaluer(echs, tactical.ScoresValides(echs, tactical.ToutesLesVariables))
	if avecVue {
		m := tactical.Evaluer(echs, tactical.ScoresValides(echs, tactical.AvecVisibilite))
		r.modeleVue = &m
	}
	return r
}

// temoin : M-1, la mesure de depart rejouee sur le depot.
func (r *rapporteur) temoin() {
	if !r.familles[familleTemoin] {
		fmt.Printf("## M-1 temoin : la famille %q n'est pas lue (table des familles ou --famille), pas de temoin\n", familleTemoin)
		return
	}
	fam := map[string]bool{familleTemoin: true}
	evts := r.jeu.evenementsDeFamilles(fam)
	echs := tactical.ConstruireEchantillons(evts, tactical.CandidatsParCarte(evts),
		tactical.ReglagesEchantillons{MatchsMinCarte: matchsMinCarteMesure})
	res := evaluerEchantillons(echs, false)
	w := tactical.Ajuster(echs, func(tactical.Echantillon) bool { return true }, tactical.ToutesLesVariables)
	fmt.Println("## M-1 temoin : Assassin hors Fiesta et BTB")
	fmt.Printf("\n%d reapparitions sur %d cartes (annexe : %d sur %d), hors candidats : %d\n\n",
		res.n, res.cartes, refReapparitions, refCartes, res.modele.HorsCandidats)
	fmt.Println("| mesure | depot | annexe | ecart |\n|---|---|---|---|")
	ligneTemoin("hasard, 1er tirage, a moins de 8 m", res.modele.HasardRayon1, refRayon1Hasard)
	fmt.Printf("| hasard, 3 tirages, a moins de 8 m (esperance exacte) | %.1f %% | | |\n", res.modele.HasardRayon3)
	ligneTemoin("frequence seule, 1er, a moins de 8 m", res.freq.Rayon1, refRayon1Freq)
	ligneTemoin("frequence seule, 3 premiers, a moins de 8 m", res.freq.Rayon3, refRayon3Freq)
	ligneTemoin("modele, 1er, a moins de 8 m", res.modele.Rayon1, refRayon1Modele)
	ligneTemoin("modele, 3 premiers, a moins de 8 m", res.modele.Rayon3, refRayon3Modele)
	ligneTemoin("frequence seule, point exact, 1er", res.freq.Top1, refTop1Freq)
	ligneTemoin("frequence seule, point exact, 3 premiers", res.freq.Top3, refTop3Freq)
	ligneTemoin("modele, point exact, 1er", res.modele.Top1, refTop1Modele)
	ligneTemoin("modele, point exact, 3 premiers", res.modele.Top3, refTop3Modele)
	fmt.Println("\n| variable | poids |\n|---|---|")
	for j, nom := range tactical.NomsVariables[:tactical.VarVisibles] {
		fmt.Printf("| %s | %+.2f |\n", nom, w[j])
	}
	fmt.Println()
	if r.o.parCarte {
		r.tableauParCarte(echs)
	}
}

func ligneTemoin(nom string, depot, annexe float64) {
	marque := ""
	if d := depot - annexe; d > ecartTemoinMax || d < -ecartTemoinMax {
		marque = " **> 1 point**"
	}
	fmt.Printf("| %s | %.1f %% | %.1f %% | %+.1f%s |\n", nom, depot, annexe, depot-annexe, marque)
}

// tableauParCarte : le temoin par carte (frequence seule puis modele, 1er et 3 premiers).
func (r *rapporteur) tableauParCarte(echs []tactical.Echantillon) {
	pred := tactical.ScoresValides(echs, tactical.ToutesLesVariables)
	predF := tactical.ScoresValides(echs, tactical.FrequenceSeule)
	var idx = map[string][]int{}
	for i, s := range echs {
		idx[s.Carte] = append(idx[s.Carte], i)
	}
	type ligne struct {
		nom        string
		n          int
		f1, m1     float64
		f3, m3, kk float64
	}
	lignes := make([]ligne, 0, len(idx))
	for carte, ii := range idx {
		var es []tactical.Echantillon
		var pm, pf [][]float64
		k := 0.0
		for _, i := range ii {
			es, pm, pf = append(es, echs[i]), append(pm, pred[i]), append(pf, predF[i])
			k += float64(len(echs[i].Feats))
		}
		mf, mm := tactical.Evaluer(es, pf), tactical.Evaluer(es, pm)
		lignes = append(lignes, ligne{r.jeu.noms[carte], len(es), mf.Top1, mm.Top1, mf.Top3, mm.Top3, k / float64(len(es))})
	}
	sort.Slice(lignes, func(a, b int) bool { return lignes[a].m3-lignes[a].f3 > lignes[b].m3-lignes[b].f3 })
	fmt.Println("| carte | n | top1 freq | top1 modele | top3 freq | top3 modele | candidats |\n|---|---|---|---|---|---|---|")
	for _, l := range lignes {
		fmt.Printf("| %s | %d | %.1f | %.1f | %.1f | %.1f | %.0f |\n", l.nom, l.n, l.f1, l.m1, l.f3, l.m3, l.kk)
	}
	fmt.Println()
}

// chargerZones resout les zones nommees des cartes lues, une fois.
func (r *rapporteur) chargerZones() {
	if r.zones != nil {
		return
	}
	var cartes []string
	for id := range r.jeu.noms {
		cartes = append(cartes, id)
	}
	sort.Strings(cartes)
	r.zones = zonesDesCartes(r.ctx, r.cfg.PathResolver().ResourcesRoot(), r.o.titleSlug, r.db, cartes)
}

// lignesParFamille ordonne les familles lues.
func (r *rapporteur) lignesParFamille() []string {
	var out []string
	for _, f := range r.o.table.Families() {
		if r.familles[f] {
			out = append(out, f)
		}
	}
	return out
}

// parMode : M-3, famille par famille.
func (r *rapporteur) parMode() {
	r.chargerZones()
	reglages := tactical.ReglagesEchantillons{MatchsMinCarte: r.o.matchsMinCarte, Zone: fonctionZone(r.zones)}
	tous := make([]ligneFamille, 0, len(r.o.table.Families()))
	for _, f := range r.lignesParFamille() {
		evts := r.jeu.evenementsDeFamilles(map[string]bool{f: true})
		echs := tactical.ConstruireEchantillons(evts, tactical.CandidatsParCarte(evts), reglages)
		tous = append(tous, ligneFamille{f, evaluerEchantillons(echs, false)})
	}
	fmt.Println("## M-3 par mode (validation croisee par match, 5 plis, cartes vues dans au moins",
		r.o.matchsMinCarte, "matchs)")
	imprimerParMode(tous)
}

// ligneDeVue : M-2, la septieme variable (ennemis a vue) contre les six premieres, sur les SEULES
// cartes dont les artefacts portent la structure (les autres n'ont pas de geometrie a lire).
func (r *rapporteur) ligneDeVue() {
	r.chargerZones()
	reglages := tactical.ReglagesEchantillons{
		MatchsMinCarte: r.o.matchsMinCarte, Zone: fonctionZone(r.zones), Vue: fonctionVue(r.jeu.obstacles),
	}
	cartes := map[string]bool{}
	tous := make([]ligneFamille, 0, len(r.o.table.Families()))
	for _, f := range r.lignesParFamille() {
		var evts []tactical.EvenementReapparition
		for _, e := range r.jeu.evenementsDeFamilles(map[string]bool{f: true}) {
			if r.jeu.obstacles[e.CarteID] != nil {
				evts, cartes[e.CarteID] = append(evts, e), true
			}
		}
		echs := tactical.ConstruireEchantillons(evts, tactical.CandidatsParCarte(evts), reglages)
		tous = append(tous, ligneFamille{f, evaluerEchantillons(echs, true)})
	}
	fmt.Println("## M-2 gain de la ligne de vue (7e variable) sur les 6 variables")
	fmt.Println()
	fmt.Printf("Cartes dont les artefacts portent la structure : %d sur %d lues.\n", len(r.jeu.obstacles), len(r.jeu.noms))
	for id := range r.jeu.obstacles {
		fmt.Printf("- %s (%d boites)\n", r.jeu.noms[id], r.jeu.obstacles[id].NbBoites())
	}
	fmt.Println()
	imprimerGainVue(tous)
}

// ligneFamille est une famille mesuree, prete a etre imprimee.
type ligneFamille struct {
	nom string
	res resultat
}

func imprimerParMode(lignes []ligneFamille) {
	fmt.Println("\nZone a 8 m : un des candidats retenus est a moins de 8 m du point reel. Zone nommee : la zone du catalogue " +
		"(une zone recoit la somme des probabilites de ses candidats). Couverture : part des reapparitions dont le point reel a une zone.")
	fmt.Println()
	fmt.Println("| mode | matchs | reapparitions | cartes | candidats/carte | a 8 m, 3 premiers : hasard | freq | modele | a 8 m, 1er : freq | modele |" +
		" zone nommee, 3 premiers : hasard | freq | modele | couverture des zones | exact 1er : modele | exact 3 : modele |")
	fmt.Println("|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|")
	for _, l := range lignes {
		m, f := l.res.modele, l.res.freq
		cov := 0.0
		if m.N > 0 {
			cov = 100 * float64(m.ZonesMesurees) / float64(m.N)
		}
		fmt.Printf("| %s | %d | %d | %d | %.0f | %.1f | %.1f | %.1f | %.1f | %.1f | %.1f | %.1f | %.1f | %.0f %% | %.1f | %.1f |\n",
			l.nom, l.res.matchs, l.res.n, l.res.cartes, l.res.kMoyen, m.HasardRayon3, f.Rayon3, m.Rayon3,
			f.Rayon1, m.Rayon1, m.HasardZone3, f.Zone3, m.Zone3, cov, m.Top1, m.Top3)
	}
	fmt.Println()
}

func imprimerGainVue(lignes []ligneFamille) {
	fmt.Println("\n| mode | reapparitions | a 8 m, 3 premiers : 6 variables | 7 variables | gain | zone nommee, 3 premiers : 6 variables | 7 variables | gain | a 8 m, 1er : gain |")
	fmt.Println("|---|---|---|---|---|---|---|---|---|")
	var n, s8, sz, s1 float64
	for _, l := range lignes {
		if l.res.modeleVue == nil {
			continue
		}
		a, b := l.res.modele, *l.res.modeleVue
		fmt.Printf("| %s | %d | %.1f | %.1f | %+.1f | %.1f | %.1f | %+.1f | %+.1f |\n",
			l.nom, l.res.n, a.Rayon3, b.Rayon3, b.Rayon3-a.Rayon3, a.Zone3, b.Zone3, b.Zone3-a.Zone3, b.Rayon1-a.Rayon1)
		w := float64(l.res.n)
		n, s8, sz, s1 = n+w, s8+w*(b.Rayon3-a.Rayon3), sz+w*(b.Zone3-a.Zone3), s1+w*(b.Rayon1-a.Rayon1)
	}
	if n > 0 {
		fmt.Printf("| toutes familles (moyenne ponderee) | %.0f | | | %+.1f | | | %+.1f | %+.1f |\n", n, s8/n, sz/n, s1/n)
	}
	fmt.Println()
}

// instant : M-4, la prevision a l'instant de la mort contre celui de la reapparition.
func (r *rapporteur) instant() {
	r.chargerZones()
	reglages := tactical.ReglagesEchantillons{MatchsMinCarte: r.o.matchsMinCarte, Zone: fonctionZone(r.zones)}
	fmt.Println("## M-4 prevision a l'instant de la mort, contre l'instant de la reapparition")
	fmt.Println("\nMemes evenements (les deux instantanes ont au moins un adversaire), memes candidats ; chaque modele est ajuste sur son propre instant.")
	fmt.Println()
	fmt.Println("| mode | reapparitions | a 8 m, 3 premiers : reapparition | mort | ecart | a 8 m, 1er : reapparition | mort | ecart | zone nommee, 3 premiers : reapparition | mort | ecart |")
	fmt.Println("|---|---|---|---|---|---|---|---|---|---|---|")
	for _, f := range r.lignesParFamille() {
		reap, mort := r.jeu.evenementsAvecLesDeux(map[string]bool{f: true})
		cands := tactical.CandidatsParCarte(reap)
		ea := tactical.ConstruireEchantillons(reap, cands, reglages)
		eb := tactical.ConstruireEchantillons(mort, cands, reglages)
		if len(ea) == 0 || len(ea) != len(eb) {
			fmt.Printf("| %s | %d | (populations non comparables : %d / %d) | | | | | | | | |\n", f, len(reap), len(ea), len(eb))
			continue
		}
		a := tactical.Evaluer(ea, tactical.ScoresValides(ea, tactical.ToutesLesVariables))
		b := tactical.Evaluer(eb, tactical.ScoresValides(eb, tactical.ToutesLesVariables))
		fmt.Printf("| %s | %d | %.1f | %.1f | %+.1f | %.1f | %.1f | %+.1f | %.1f | %.1f | %+.1f |\n",
			f, a.N, a.Rayon3, b.Rayon3, b.Rayon3-a.Rayon3, a.Rayon1, b.Rayon1, b.Rayon1-a.Rayon1,
			a.Zone3, b.Zone3, b.Zone3-a.Zone3)
	}
	fmt.Println()
}
