package main

// sante.go — LA METRIQUE DE SANTE, ET CE QU ELLE NE VOIT PAS.
//
// CE QU ELLE SURVEILLE : des hypotheses qui peuvent cesser d etre vraies SANS produire la moindre
// erreur — un catalogue de sources perime par une saison neuve, un roster plus grand que prevu, un
// mode de jeu inconnu. Aucune ne leve d exception : elles se paient en couverture silencieuse.
//
// LA SORTIE AFFICHE SON POINT AVEUGLE, et ce n est pas de la modestie decorative : le compteur
// principal ne voit un identifiant perime que si la voie SEQUENTIELLE porte cet identifiant. Un
// identifiant servi par le seul balayage ne le fait pas sonner — c est mesure. Dans ce mode, le
// plancher de couverture est le seul filet, et un operateur doit le savoir en lisant le verdict.

import (
	"fmt"
	"os"
	"text/tabwriter"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/games/halo_infinite/film/killsource"
)

func afficherSante(r *rapport) error {
	h := r.result.Health
	fmt.Printf("SANTE DU DECODAGE — film %s\n\n", r.film)
	fmt.Printf("VERDICT : %s\n", h.Verdict())
	for _, a := range h.Alerts() {
		fmt.Printf("   ALERTE : %s\n", a)
	}
	if len(h.Alerts()) == 0 {
		fmt.Println("   aucune alerte")
	}
	blocDomaine(h)
	blocVentilation(h)
	blocVoies(r.result.Stats)
	blocSonde(r.result)
	blocPointAveugle()
	blocCompteurs(h)
	return nil
}

// blocDomaine : un verdict n est PAS un jugement sur les etiquettes publiees. C est un jugement
// sur le DOMAINE : ce film ressemble-t-il a ceux sur lesquels le decodeur a ete mesure ?
func blocDomaine(h filmdec.KillSourceHealth) {
	fmt.Println("\nLE VERDICT PORTE SUR LE DOMAINE, PAS SUR LES ETIQUETTES")
	fmt.Println("  Il dit << ce film ressemble-t-il a ceux sur lesquels le decodeur a ete mesure >>.")
	fmt.Println("  Un film HORS DOMAINE n est pas casse : ses lignes se ponderent, voila tout.")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "  candidats inexpliques\t%.1f %%\tseuil de sortie de domaine %.1f %% · alerte %.1f %%\t\n",
		100*h.UnexplainedRatio(), 100*filmdec.UnexplainedWarnRatio, 100*filmdec.UnexplainedAlertRatio)
	fmt.Fprintf(w, "  couverture\t%.1f %%\tplancher %.1f %% (la serie de reference est exacte)\t\n",
		100*h.CoverageRatio(), 100*filmdec.CoverageWarnRatio)
	_ = w.Flush()
	fmt.Println("  Seuils tires de la distribution de CINQ films, pas d une intuition :")
	fmt.Println("     4 films a 8 joueurs : 7.0 / 9.4 / 11.6 / 17.8 % d inexpliques, couverture 100 %")
	fmt.Println("     1 film BTB          : 26.0 % d inexpliques, couverture 76.5 % — CONTROLE POSITIF,")
	fmt.Println("                           il DOIT sortir du domaine, et il en sort par trois criteres.")
}

// blocVentilation : les candidats que rien ne publie. ILS NE SORTENT PAS et ne coutent rien au
// consommateur — c est leur TAUX qui informe, jamais leur existence.
func blocVentilation(h filmdec.KillSourceHealth) {
	fmt.Println("\nLES CANDIDATS QUE RIEN NE PUBLIE — ils ne sortent pas, c est leur TAUX qui informe")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "  candidats consultes\t%d\t\n", h.Candidates)
	fmt.Fprintf(w, "  lignes publiees\t%d\t\n", h.Published)
	fmt.Fprintf(w, "  inexpliques (sur les candidats)\t%d\t= %s\t\n",
		h.UnexplainedTotal(), pct(h.UnexplainedTotal(), h.Candidates))
	fmt.Fprintf(w, "     couple sans equivalent au kill-feed\t%d\t\n", h.UnexplainedPair)
	fmt.Fprintf(w, "     source appartenant a la victime, non retenue\t%d\t\n", h.UnexplainedSelf)
	fmt.Fprintf(w, "     indice de bot non resolu\t%d\t\n", h.UnexplainedBotIdx)
	_ = w.Flush()
	fmt.Printf("  hors roster : %d — COMPTEUR SEPARE, et c est un choix.\n", h.OutOfRoster)
	fmt.Println("     Il se compte sur une AUTRE population que les trois ci-dessus (le filtre de")
	fmt.Println("     credibilite ecarte ces lignes avant qu elles ne deviennent des candidats), donc")
	fmt.Println("     il n entre pas dans le taux. Un ratio dont le numerateur deborde du denominateur")
	fmt.Println("     ne veut rien dire. Non nul = un participant n est pas compte.")
}

// blocVoies : le cout PAR VOIE. A lire comme une VENTILATION DU COUT, jamais comme deux precisions
// directement comparables — la bijection est ajustee sur l union des deux, et la marche en fournit
// 91 % : le decoupage n est pas neutre vis-a-vis de cet ajustement.
func blocVoies(s killsource.Stats) {
	fmt.Println("\nLE COUT, VENTILE PAR VOIE DE LECTURE")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "  voie\tproposes\tapparies au couple exact\ttaux\tpublies\t")
	fmt.Fprintf(w, "  sequentielle\t%d\t%d\t%s\t%d\t\n",
		s.Walk.Population, s.Walk.Matched, pct(s.Walk.Matched, s.Walk.Population), s.Walk.Published)
	fmt.Fprintf(w, "  balayage (rattrapage)\t%d\t%d\t%s\t%d\t\n",
		s.Scan.Population, s.Scan.Matched, pct(s.Scan.Matched, s.Scan.Population), s.Scan.Published)
	_ = w.Flush()
	fmt.Println("  VENTILATION DU COUT, PAS DEUX PRECISIONS COMPARABLES : la bijection indice -> joueur")
	fmt.Println("  est ajustee sur l UNION des deux voies, dont la sequentielle fournit 91 % des")
	fmt.Println("  candidats — le decoupage d un objectif maximise conjointement n est pas neutre.")
	fmt.Printf("\n  concordance : %d enregistrement(s) lu(s) par les deux voies · accord %d · DESACCORD %d\n",
		s.Redundant, s.Agree, s.Disagree)
	if s.Disagree == 0 {
		fmt.Println("     DESACCORD nul : les deux voies lisent le meme champ au meme bit. La preference")
		fmt.Println("     de la voie sequentielle est donc une PREFERENCE, pas un arbitrage.")
	} else {
		fmt.Println("     DESACCORD NON NUL : la preference est devenue un ARBITRAGE. A documenter avant")
		fmt.Println("     toute publication — ce cas ne s est jamais produit sur la serie de reference.")
	}
	fmt.Printf("  morts a plusieurs candidats en concurrence : %d\n", s.MultiCandidate)
	fmt.Printf("  paquets a evenements localises : %d / %d = %s\n",
		s.PacketsLocated, s.PacketsWithEvents, pct(s.PacketsLocated, s.PacketsWithEvents))
}

// blocSonde : la sonde a porte de catalogue relachee. PUBLIEE, EXCLUE DES ALERTES, et c est mesure.
func blocSonde(res *killsource.Result) {
	if res.Probe == nil {
		fmt.Println("\nSONDE A PORTE DE CATALOGUE RELACHEE : non executee (couverture complete).")
		fmt.Println("  Elle ne porte de l information que sur les morts NON COUVERTES : a 100 % de")
		fmt.Println("  couverture elle vaut zero par construction, et elle coute cher.")
		return
	}
	p := res.Probe
	fmt.Printf("\nSONDE A PORTE DE CATALOGUE RELACHEE : %d candidat(s), %d hors catalogue, %d apparie(s),\n",
		p.Candidates, p.OutOfCatalogue, p.Paired)
	fmt.Printf("  dont %d sur une mort NON COUVERTE", p.Uncovered)
	if len(p.Tags) > 0 {
		// Troncature volontaire : hors domaine mesure (BTB), cette liste depasse le millier
		// d identifiants et rend la sortie illisible dans un terminal. L outil de RE tronque
		// deja (`tmp_deadstate hyhealth`) ; ne pas le faire ici etait une regression.
		const maxTagsAffiches = 12
		fmt.Printf(" — identifiants : ")
		for i, t := range p.Tags {
			if i >= maxTagsAffiches {
				fmt.Printf("... (%d au total)", len(p.Tags))
				break
			}
			if i > 0 {
				fmt.Print(" ")
			}
			fmt.Printf("%08x", t)
		}
	}
	fmt.Println()
	fmt.Println("  ELLE INFORME, ELLE N ALERTE PAS : son rapport signal/hasard mesure vaut 1.10 a 1.72")
	fmt.Println("  selon le film. Seule la ligne << morts NON COUVERTES >> est exploitable.")
}

// blocPointAveugle : CE QUE LA METRIQUE NE VOIT PAS. Mesure, pas suppose.
func blocPointAveugle() {
	fmt.Println("\nCE QUE CETTE METRIQUE NE VOIT PAS — point aveugle MESURE")
	fmt.Println("  Le compteur principal (identifiants hors catalogue vus par la voie sequentielle)")
	fmt.Println("  ne sonne QUE si la voie sequentielle porte l identifiant concerne. Un identifiant")
	fmt.Println("  dont la seule occurrence publiee vient du balayage passe INAPERCU du compteur :")
	fmt.Println("     ablation mesuree : compteur = 0, aucune alerte, couverture 99 -> 98.")
	fmt.Println("  C EST POURQUOI LE PLANCHER DE COUVERTURE VAUT 1.00 : dans ce mode, il est le SEUL")
	fmt.Println("  filet. Ce n est pas une severite a assouplir, c est le compteur de secours.")
	fmt.Println("  Portee juste du controle positif : 20 ablations sur 20 declenchent l alerte SUR LES")
	fmt.Println("  IDENTIFIANTS QUE LA VOIE SEQUENTIELLE PORTE — et non << sur un catalogue perime >>.")
	fmt.Println("  Consequence sur la verite terrain : 5 des 30 ancres Theater sont servies par le seul")
	fmt.Println("  balayage (4 sur 9b191a7f, 1 sur 78919882) et ne sont donc pas protegees.")
}

// blocCompteurs : la publication expvar, telle que le brancheur l ecrira.
func blocCompteurs(h filmdec.KillSourceHealth) {
	fmt.Println("\nCOMPTEURS PRETS POUR expvar (ADR 0009 — entiers, snake_case, aucun ratio publie)")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, p := range h.ExpvarPairs() {
		fmt.Fprintf(w, "  %s\t%d\t\n", p.Name, p.Value)
	}
	_ = w.Flush()
	fmt.Println("  Aucun ratio n est publie : une agregation multi-films doit sommer des numerateurs")
	fmt.Println("  et des denominateurs, jamais moyenner des pourcentages.")
	fmt.Println("  Cablage cote appelant :")
	fmt.Println("     for _, p := range res.Health.ExpvarPairs() { observability.AddInt(p.Name, p.Value) }")
}

// santeDeJSON : la meme mesure, pour la sortie JSON.
func santeDeJSON(res *killsource.Result) santeJSON {
	h := res.Health
	s := santeJSON{
		Verdict: h.Verdict(), Alertes: h.Alerts(),
		TauxInexpliques: h.UnexplainedRatio(), TauxCouverture: h.CoverageRatio(),
		GateParVoie: map[string]gateJSON{
			"sequentielle": gateDeJSON(res.Stats.Walk),
			"balayage":     gateDeJSON(res.Stats.Scan),
		},
		Concordance: concordanceJSON{
			Redondants: res.Stats.Redundant, Accord: res.Stats.Agree,
			Desaccord: res.Stats.Disagree, MortsPlusieursCands: res.Stats.MultiCandidate,
			Note: "desaccord nul = les deux voies lisent le meme champ au meme bit, donc " +
				"preferer la voie sequentielle est une PREFERENCE et non un arbitrage",
		},
		NoteSondeCatalogue: "publiee mais EXCLUE des alertes : rapport signal/hasard mesure de " +
			"1.10 a 1.72 ; non executee quand la couverture est complete",
	}
	if s.Alertes == nil {
		s.Alertes = []string{}
	}
	for _, p := range h.ExpvarPairs() {
		s.Compteurs = append(s.Compteurs, compteurJSON{
			Nom: p.Name, Valeur: p.Value, Alerte: compteurAlerte(p.Name), Comment: remarqueCompteur(p.Name),
		})
	}
	if res.Probe != nil {
		tags := make([]string, 0, len(res.Probe.Tags))
		for _, t := range res.Probe.Tags {
			tags = append(tags, hex8(t))
		}
		s.SondeCatalogue = &sondeJSON{
			Candidats: res.Probe.Candidates, HorsCatalogue: res.Probe.OutOfCatalogue,
			Apparies: res.Probe.Paired, NonCouvertes: res.Probe.Uncovered, TagsConcernees: tags,
		}
	}
	return s
}

func gateDeJSON(p killsource.PathStats) gateJSON {
	return gateJSON{Population: p.Population, Apparies: p.Matched, Publiees: p.Published, Taux: p.Ratio()}
}

// compteurAlerte : lesquels des compteurs publies declenchent une alerte dure.
func compteurAlerte(nom string) bool {
	return nom == "killsource_tag_out_of_catalogue_walk" || nom == "killsource_out_of_roster"
}

// remarqueCompteur : la portee des deux compteurs porteurs, attachee au compteur lui-meme pour
// qu un tableau de bord ne la perde pas en route.
func remarqueCompteur(nom string) string {
	switch nom {
	case "killsource_tag_out_of_catalogue_walk":
		return "compteur principal de catalogue perime ; bruit MESURE nul sur 5 films ; AVEUGLE aux " +
			"identifiants servis par le seul balayage — le plancher de couverture est alors le filet"
	case "killsource_out_of_roster":
		return "population distincte des trois compteurs `unexplained_` : ne pas l y additionner"
	case "killsource_tag_out_of_catalogue_scan":
		return "informe, n alerte pas (rapport signal/hasard 1.10 a 1.72)"
	default:
		return ""
	}
}

func publicationDeJSON(res *killsource.Result) publicationJSON {
	p := publicationJSON{
		LigneParLigneAutorisee: res.LineByLinePublishable(),
		MargeDeBijection:       res.BijectionMargin,
	}
	if p.LigneParLigneAutorisee {
		return p
	}
	if res.BijectionMargin <= 0 {
		p.Motif = "marge de bijection nulle : au moins deux joueurs sont interchangeables, les " +
			"attributions individuelles sont fausses meme si l agregat est juste"
		return p
	}
	p.Motif = "la metrique de sante est en ALERTE"
	return p
}
