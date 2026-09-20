//go:build research

package main

// verdict_rapport_research_test.go — L'ECRITURE DU DOCUMENT DE VERDICT.
//
// Le test est la PREUVE REJOUABLE, le document est la PIECE : c'est lui que le pilote lit et
// sur lequel il tranche. Tout ce qui est imprime ici sort des bilans calcules par
// `verdict_oracle_research_test.go` — aucune valeur n'y est saisie a la main, aucune n'y est
// arrondie a l'avantage de qui que ce soit.

import (
	"fmt"
	"sort"
	"strings"

	"levelup/go-api/internal/analysis/powerpos"
)

// verdictRendu rassemble tout ce que le document imprime (une struct et non cinq
// parametres qui voyagent ensemble).
type verdictRendu struct {
	Doc         SortiePositions
	Ordre       []string
	Reels       map[string]bilanCarte
	Temoins     map[string]bilanCarte
	Diagnostics []diagnosticZone
}

// rendVerdict assemble le document complet.
func rendVerdict(v verdictRendu) string {
	var b strings.Builder
	eligibles, piegesPurs := tranche(v.Ordre, v.Reels)
	entete(&b, v.Doc, eligibles, len(eligibles) >= cartesExigees && piegesPurs == 0)
	tableParCarte(&b, v.Ordre, v.Reels, v.Temoins)
	tableTemoin(&b, v.Ordre, v.Reels, v.Temoins)
	tableContreExemples(&b, v.Ordre, v.Reels)
	tableFautes(&b, v.Ordre, v.Reels)
	tablePositions(&b, v.Ordre, v.Reels)
	tableDiagnostic(&b, v.Diagnostics)
	synthese(&b, v.Diagnostics)
	return b.String()
}

// synthese compte les causes d'echec et les rend en une conclusion. Les chiffres sont
// COMPTES sur les diagnostics, jamais saisis : le document se regenere avec la mesure.
func synthese(b *strings.Builder, diags []diagnosticZone) {
	fmt.Fprintf(b, "## 7. Ce que ces chiffres disent\n\n")
	if len(diags) == 0 {
		fmt.Fprintln(b, "Aucune zone `forte` manquee : rien a expliquer.")
		return
	}
	causes := map[string]int{}
	for _, d := range diags {
		causes[strings.SplitN(d.Cause, " :", 2)[0]] += 1
	}
	var cles []string
	for c := range causes {
		cles = append(cles, c)
	}
	sort.Strings(cles)
	fmt.Fprintf(b, "Sur **%d zones `forte` manquees**, la repartition des causes est :\n\n", len(diags))
	for _, c := range cles {
		fmt.Fprintf(b, "- **%s** : %d\n", c, causes[c])
	}
	fmt.Fprintf(b, "\nLe signal qui manque n'est donc PAS le meme partout, et c'est le fait"+
		" saillant de l'etape : une part des zones attendues EST detectee par le score (des"+
		" cellules y passent le seuil) mais ne forme pas d'amas de %d cellules 4-connexes, et"+
		" une autre part tombe a quelques millemes sous le p90 de sa carte. Ni l'une ni l'autre"+
		" ne se corrige en etape 2 : toute retouche de seuil apres le verdict l'invalide"+
		" (protocole du plan). L'arbitrage revient au pilote.\n", powerpos.ReglageV1().TailleMiniComposante)
}

// tranche rend les cartes eligibles comme preuve et le nombre de pieges PURS colores.
//
// LIVE FIRE EST EXCLUE DES DEUX COMPTES. Le document de mesure de l'etape 1 l'ecrit ainsi :
// ses positions de kill sont decalees de 9,88 m entre ses deux variantes, donc « ses chiffres
// ne comptent pas comme preuve, DANS UN SENS COMME DANS L'AUTRE ». Lui faire porter un
// contre-exemple serait la compter a charge apres l'avoir retiree des preuves.
func tranche(ordre []string, reels map[string]bilanCarte) ([]string, int) {
	var eligibles []string
	piegesPurs := 0
	for _, nom := range ordre {
		b := reels[nom]
		if nom == carteContaminee || b.NbResolues == 0 {
			continue
		}
		piegesPurs += len(b.ContreExemplesPurs)
		if b.Rappel >= seuilRappel && b.Precision >= seuilPrecision {
			eligibles = append(eligibles, nom)
		}
	}
	return eligibles, piegesPurs
}

// entete ecrit le verdict en une ligne, puis le rappel des regles.
func entete(b *strings.Builder, doc SortiePositions, eligibles []string, go2 bool) {
	mot := "NO-GO"
	if go2 {
		mot = "GO"
	}
	fmt.Fprintf(b, "# VERDICT — positions de force contre l'oracle pro (etape 2, 2026-09-20)\n\n")
	fmt.Fprintf(b, "> **%s** — %d carte(s) sur les %d exigees tiennent rappel >= %.2f ET precision >= %.2f"+
		" hors Live Fire (contaminee) : %s.\n\n",
		mot, len(eligibles), cartesExigees, seuilRappel, seuilPrecision, liste(eligibles))
	fmt.Fprintf(b, "> Genere par `go test -tags research ./cmd/mappower-build/ -run Verdict`"+
		" (`verdict_oracle_research_test.go`). Positions jugees : cuisson du %s,"+
		" reglage FIGE `powerpos.ReglageV1` (disque %.1f m, plancher %d matchs,"+
		" %d engagements, quantile %.2f, plancher de score %.2f, composante >= %d cellules).\n\n",
		doc.GenereLe, doc.Reglage.RayonLissageM, doc.Reglage.PlancherMatchs,
		doc.Reglage.MinEngagementsDisque, doc.Reglage.QuantileSeuil,
		doc.Reglage.SeuilScoreMin, doc.Reglage.TailleMiniComposante)
	fmt.Fprintf(b, "Regles appliquees, toutes ecrites avant la mesure (plan item 2.1-2.4,"+
		" relecture du pilote en §7 de l'oracle) :\n\n"+
		"- **retrouvee** : le polygone de la position couvre >= %.0f %% de l'aire de la zone attendue,"+
		" OU son barycentre tombe dans la zone ;\n"+
		"- **rappel** = zones `forte` retrouvees / zones `forte` RESOLUES dans le vocabulaire du depot ;\n"+
		"- **precision** = positions calculees qui retrouvent au moins une zone de l'oracle"+
		" (`forte` ou `faible`) / positions calculees ;\n"+
		"- les lignes `zone_en = ?` de l'oracle sont exclues des deux (sa regle de lecture) ;\n"+
		"- les zones `forte` de raison « arme » SEULE sont rapportees a part ;\n"+
		"- **Live Fire ne compte pas comme preuve** (decalage mesure de 9,88 m entre ses variantes) ;\n"+
		"- sur une carte a moins de %d zones `forte`, un rappel de 0,50 se lit"+
		" « indetermine », pas « echec » ;\n"+
		"- le GO exige en plus ZERO contre-exemple PUR colore.\n\n",
		seuilRecouvrement*100, fortesPourTrancher)
	fmt.Fprintf(b, "Les aires sont mesurees par echantillonnage au pas de %.1f m"+
		" (surfaces concaves, trouees et en plusieurs morceaux du catalogue de zones).\n\n",
		pasEchantillonM)
}

// tableParCarte : le tableau principal.
func tableParCarte(b *strings.Builder, ordre []string, reels, temoins map[string]bilanCarte) {
	fmt.Fprintf(b, "## 1. Rappel, precision, temoin — carte par carte\n\n")
	fmt.Fprintln(b, "| Carte | Cle oracle | `forte` (resolues/total) | Rappel | Rappel temoin |"+
		" Rappel hors « arme » seule | Positions | Precision | Precision temoin | Lecture |")
	fmt.Fprintln(b, "|---|---|---|---|---|---|---|---|---|---|")
	for _, nom := range ordre {
		r, tm := reels[nom], temoins[nom]
		fmt.Fprintf(b, "| %s | `%s` | %d/%d | %s | %s | %s (%d) | %d | %s | %s | %s |\n",
			nom, r.CarteCle, r.NbResolues, r.NbFortes,
			pourcent(r.Rappel), pourcent(tm.Rappel),
			pourcent(r.RappelHorsArme), r.NbHorsArme,
			len(r.Positions), pourcent(r.Precision), pourcent(tm.Precision), lecture(r))
	}
	fmt.Fprintln(b)
}

// lecture qualifie la ligne selon les regles du pilote.
func lecture(b bilanCarte) string {
	switch {
	case b.NbResolues == 0:
		return "hors verdict (aucune zone `forte` resolue)"
	case b.Carte == carteContaminee:
		return "rapporte, NE COMPTE PAS (contamination)"
	case len(b.Positions) == 0:
		return "aucune position calculee"
	case b.Rappel >= seuilRappel && b.Precision >= seuilPrecision:
		return "TIENT"
	case b.NbResolues < fortesPourTrancher && b.Rappel >= 0.5:
		return "indetermine (oracle a gros grain)"
	default:
		return "ECHOUE"
	}
}

// tableTemoin : l'ecart entre le reel et la permutation circulaire.
func tableTemoin(b *strings.Builder, ordre []string, reels, temoins map[string]bilanCarte) {
	fmt.Fprintf(b, "## 2. Temoin negatif — permutation circulaire des zones\n\n")
	fmt.Fprintf(b, "La zone attendue en Z_i est reputee en Z_i+1 dans la liste TRIEE des zones"+
		" nommees de la carte. Si le rappel ne s'effondre pas, l'appariement ne mesure que la"+
		" densite des zones, pas leur identite — et un GO ne vaudrait rien.\n\n")
	fmt.Fprintln(b, "| Carte | Rappel reel | Rappel temoin | Delta | Precision reelle |"+
		" Precision temoin | Delta | Decalage applique aux zones `forte` |")
	fmt.Fprintln(b, "|---|---|---|---|---|---|---|---|")
	var sr, st, pr, pt float64
	n := 0
	for _, nom := range ordre {
		r, tm := reels[nom], temoins[nom]
		if r.NbResolues == 0 {
			continue
		}
		fmt.Fprintf(b, "| %s | %s | %s | %+.2f | %s | %s | %+.2f | %s |\n", nom,
			pourcent(r.Rappel), pourcent(tm.Rappel), tm.Rappel-r.Rappel,
			pourcent(r.Precision), pourcent(tm.Precision), tm.Precision-r.Precision,
			liste(tm.Permutation))
		sr, st, pr, pt = sr+r.Rappel, st+tm.Rappel, pr+r.Precision, pt+tm.Precision
		n++
	}
	if n > 0 {
		fmt.Fprintf(b, "| **moyenne** | **%s** | **%s** | **%+.2f** | **%s** | **%s** | **%+.2f** | |\n",
			pourcent(sr/float64(n)), pourcent(st/float64(n)), (st-sr)/float64(n),
			pourcent(pr/float64(n)), pourcent(pt/float64(n)), (pt-pr)/float64(n))
	}
	if n == 0 {
		fmt.Fprintln(b)
		return
	}
	fmt.Fprintf(b, "\n**Reserve sur la force de ce temoin, a lire dans la derniere colonne.**"+
		" La liste des zones est TRIEE PAR NOM, et le vocabulaire des cartes est ainsi fait que"+
		" le voisin alphabetique est souvent le voisin GEOGRAPHIQUE (`Dried Rat Hole` ->"+
		" `Dried Rat Tunnel`, `Whirlpool Dam` -> `Whirlpool Ledge`, `Subway Balcony` ->"+
		" `Subway Bend`, `Main Street` -> `Main Street Alley`). Le decalage deplace donc souvent"+
		" l'attente de quelques metres seulement : le temoin est PLUS FAIBLE que ne le laisse"+
		" croire son principe. Il reste concluant dans ce sens-ci — le rappel reel (%s) n'est"+
		" pas meilleur que le rappel temoin (%s). L'appariement ne porte quasiment aucun signal,"+
		" ce qui est le constat du NO-GO et non une objection a lui.\n\n",
		pourcent(sr/float64(n)), pourcent(st/float64(n)))
}

// tableContreExemples : les positions posees sur un lieu que les guides deconseillent.
func tableContreExemples(b *strings.Builder, ordre []string, reels map[string]bilanCarte) {
	fmt.Fprintf(b, "## 3. Contre-exemples (§3.1 de l'oracle)\n\n")
	fmt.Fprintf(b, "Une position calculee est un contre-exemple quand son BARYCENTRE tombe dans"+
		" une zone que les guides deconseillent. Deux colonnes, parce que la §4 de l'oracle"+
		" (« desaccords entre sources ») nomme des zones qui sont a la fois position et piege"+
		" dans le MEME article : les exiger a zero demanderait a l'algorithme de trancher un"+
		" desaccord que l'oracle n'a pas tranche. **Le GO porte sur la colonne PURS.**\n\n")
	fmt.Fprintln(b, "| Carte | Pieges PURS colores | Pieges aussi decrits comme positions |")
	fmt.Fprintln(b, "|---|---|---|")
	purs := 0
	for _, nom := range ordre {
		r := reels[nom]
		purs += len(r.ContreExemplesPurs)
		fmt.Fprintf(b, "| %s | %s | %s |\n", nom, liste(r.ContreExemplesPurs),
			liste(r.ContreExemplesPartage))
	}
	fmt.Fprintf(b, "\n**Total des pieges PURS colores : %d** (dont %d hors Live Fire — c'est"+
		" ce compte-la qui engage le GO).\n\n", purs, horsContaminee(ordre, reels))
}

// horsContaminee compte les pieges purs des cartes qui comptent comme preuve.
func horsContaminee(ordre []string, reels map[string]bilanCarte) int {
	_, purs := tranche(ordre, reels)
	return purs
}

// tableFautes : faux positifs et faux negatifs nommes.
func tableFautes(b *strings.Builder, ordre []string, reels map[string]bilanCarte) {
	fmt.Fprintf(b, "## 4. Faux positifs et faux negatifs, nommes\n\n")
	fmt.Fprintln(b, "| Carte | Faux negatifs (`forte` manquees) | Faux positifs (positions sans zone de l'oracle) | `forte` NON RESOLUES au vocabulaire |")
	fmt.Fprintln(b, "|---|---|---|---|")
	for _, nom := range ordre {
		r := reels[nom]
		fmt.Fprintf(b, "| %s | %s | %s | %s |\n", nom, liste(r.FauxNegatifs),
			liste(r.FauxPositifs), liste(r.FortesNonResolu))
	}
	fmt.Fprintf(b, "\nUne zone `forte` NON RESOLUE est absente du catalogue de zones du depot"+
		" pour cette carte : l'algorithme ne peut ni la trouver ni la manquer. Elle est hors"+
		" du denominateur du rappel, et signalee ici pour que le defaut reste visible.\n\n")
}

// tablePositions : la table que le pilote lira, une ligne par position calculee.
func tablePositions(b *strings.Builder, ordre []string, reels map[string]bilanCarte) {
	fmt.Fprintf(b, "## 5. Chaque position calculee et sa zone nommee dominante\n\n")
	fmt.Fprintf(b, "« Part » = fraction de la position couverte par la zone dominante."+
		" « Aire zone » = aire de la zone dominante (union des zones du meme libelle sur les"+
		" cartes en miroir).\n\n")
	fmt.Fprintln(b, "| Carte | Position | Score moyen | Aire pos. m2 | Zone dominante | Part | Aire zone m2 | Appariee a |")
	fmt.Fprintln(b, "|---|---|---|---|---|---|---|---|")
	for _, nom := range ordre {
		for _, p := range reels[nom].Positions {
			apparie := "—"
			if p.Appariee {
				apparie = p.ApparieeA
			}
			fmt.Fprintf(b, "| %s | `%s` | %.3f | %.1f | %s | %s | %.1f | %s |\n",
				nom, p.ID, p.ScoreMoyen, p.AireM2, p.ZoneDominante,
				pourcent(p.PartDominante), p.AireDominanteM2, apparie)
		}
	}
	fmt.Fprintln(b)
}

// liste rend une enumeration lisible en cellule de tableau.
func liste(v []string) string {
	if len(v) == 0 {
		return "—"
	}
	out := append([]string{}, v...)
	sort.Strings(out)
	return strings.Join(out, " ; ")
}
