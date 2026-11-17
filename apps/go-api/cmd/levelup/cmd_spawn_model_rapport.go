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
	_, _ = fmt.Fprintf(w, "| reapparitions lues au film | documents sans lecture du film | sans equipe | non finies | comptes a rebours sans joueur |\n|---|---|---|---|---|\n")
	_, _ = fmt.Fprintf(w, "| %d | %d | %d | %d | %d |\n\n", b.reapparitions, b.sansLectureFilm, b.sansEquipe, b.nonFinis, b.comptesSansJoueur)
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
}

// evaluerEchantillons note un jeu d'echantillons : frequence seule et modele a six variables.
func evaluerEchantillons(echs []tactical.Echantillon) resultat {
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
	return r
}

// reglages rend les reglages des echantillons de la lecture, avec l'indice de zone nommee des
// cartes lues.
func (r *rapporteur) reglages() tactical.ReglagesEchantillons {
	r.chargerZones()
	return tactical.ReglagesEchantillons{
		MatchsMinCarte: r.o.matchsMinCarte,
		Zone:           func(carte string, p tactical.Point3) int { return tactical.IndiceDeZone(r.zones[carte], p) },
	}
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
	reglages := r.reglages()
	tous := make([]ligneFamille, 0, len(r.o.table.Families()))
	for _, f := range r.lignesParFamille() {
		evts := r.jeu.evenementsDeFamilles(map[string]bool{f: true})
		echs := tactical.ConstruireEchantillons(evts, tactical.CandidatsParCarte(evts), reglages)
		tous = append(tous, ligneFamille{f, evaluerEchantillons(echs)})
	}
	fmt.Println("## M-3 par mode (validation croisee par match, 5 plis, cartes vues dans au moins",
		r.o.matchsMinCarte, "matchs)")
	imprimerParMode(tous)
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

// instant : M-4, la prevision a l'instant de la mort contre celui de la reapparition.
func (r *rapporteur) instant() {
	reglages := r.reglages()
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
