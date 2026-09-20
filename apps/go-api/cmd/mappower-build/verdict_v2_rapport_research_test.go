//go:build research

package main

// verdict_v2_rapport_research_test.go — L'ECRITURE DU DOCUMENT DE VERDICT V2 : le verdict
// en une ligne, les regles, la table par carte (vrai / temoin, rappel par groupe de
// raison), le temoin geographique, les contre-exemples, les fautes nommees.
//
// Comme en v1 : le test est la preuve rejouable, le document est la piece. Tout ce qui est
// imprime sort des bilans calcules ; aucune valeur n'est saisie a la main.

import (
	"fmt"
	"strings"
)

// verdictV2Rendu rassemble tout ce que le document imprime.
type verdictV2Rendu struct {
	CheminPositions string
	// Lignee : le nom de la lignee jugee, deduit du nom du fichier ; ReglageBrut : son
	// objet `reglage` tel quel (chaque lignee a le sien).
	Lignee      string
	ReglageBrut string
	Doc         SortiePositions
	Ordre       []string
	Reels       map[string]bilanV2
	Temoins     map[string]bilanV2
	// Lignees : toutes les lignees du chantier jugees aux memes regles.
	Lignees []ligneeJugee

	Diagnostics    []diagnosticV2
	Fidelites      []fideliteRejeu
	DiagnosticsGeo []diagnosticGeo
}

// rendVerdictV2 assemble le document complet.
func rendVerdictV2(v verdictV2Rendu) string {
	var b strings.Builder
	enteteV2(&b, v)
	tableParCarteV2(&b, v.Ordre, v.Reels, v.Temoins)
	tableTemoinGeographique(&b, v.Ordre, v.Reels, v.Temoins)
	tableContreExemplesV2(&b, v.Ordre, v.Reels)
	tableFautesV2(&b, v.Ordre, v.Reels)
	tablePositions(&b, v.Ordre, bilansCarte(v.Reels))
	tableCouloirs(&b, v.Ordre, v.Reels)
	tableDiagnosticV2(&b, v)
	tableLignees(&b, v)
	sectionGeoEtFusion(&b, v)
	syntheseV2(&b, v.Diagnostics)
	return b.String()
}

// trancheV2 rend les cartes de VALIDATION eligibles comme preuve, et le nombre de pieges
// PURS colores sur les cartes de validation. Les cartes de calibrage ne comptent ni dans un
// sens ni dans l'autre.
func trancheV2(ordre []string, reels map[string]bilanV2) ([]string, int) {
	var eligibles []string
	piegesPurs := 0
	for _, nom := range ordre {
		b := reels[nom]
		if b.Role != roleValidation || b.NbResolues == 0 {
			continue
		}
		piegesPurs += len(b.ContreExemplesPurs)
		if tient(b) {
			eligibles = append(eligibles, nom)
		}
	}
	return eligibles, piegesPurs
}

// tient dit si une carte passe les deux seuils du plan.
func tient(b bilanV2) bool {
	return b.NbResolues > 0 && len(b.Positions) > 0 &&
		b.Rappel >= seuilRappel && b.PrecisionForte >= seuilPrecision
}

// bilansCarte projette les bilans v2 sur le type v1, pour reutiliser ses tables.
func bilansCarte(reels map[string]bilanV2) map[string]bilanCarte {
	out := make(map[string]bilanCarte, len(reels))
	for nom, b := range reels {
		out[nom] = b.bilanCarte
	}
	return out
}

// motVerdict rend GO ou NO-GO selon le critere du plan.
func motVerdict(eligibles []string, piegesPurs int) string {
	if len(eligibles) >= cartesExigees && piegesPurs == 0 {
		return "GO"
	}
	return "NO-GO"
}

// enteteV2 ecrit les deux verdicts en tete (strict sur la validation, elargi avec le
// calibrage — nomme comme tel), puis le rappel des regles.
func enteteV2(b *strings.Builder, v verdictV2Rendu) {
	strict, piegesStrict := trancheV2(v.Ordre, v.Reels)
	elargi, piegesElargi := trancheElargi(v.Ordre, v.Reels)
	fmt.Fprintf(b, "# VERDICT — %s contre l'oracle pro v2 (item 2bis.D, 2026-09-20)\n\n", v.Lignee)
	fmt.Fprintf(b, "> **%s — STRICT : %s** — %d carte(s) de VALIDATION sur les %d exigees tiennent"+
		" rappel >= %.2f ET precision >= %.2f (%s) ; %d piege(s) PUR(S) colore(s) sur les cartes de validation.\n>\n",
		v.Lignee, motVerdict(strict, piegesStrict), len(strict), cartesExigees, seuilRappel, seuilPrecision,
		liste(strict), piegesStrict)
	fmt.Fprintf(b, "> **%s — ELARGI (validation + calibrage, N'EST PAS un GO strict) : %s** — %d carte(s)"+
		" sur les %d exigees tiennent (%s) ; %d piege(s) PUR(S) sur ces cartes. Les cartes de calibrage"+
		" ont servi a choisir les reglages : un GO elargi dit que la methode tient sur ce qu'elle a vu,"+
		" pas qu'elle generalise.\n\n",
		v.Lignee, motVerdict(elargi, piegesElargi), len(elargi), cartesExigees, liste(elargi), piegesElargi)
	fmt.Fprintf(b, "> Genere par `go test -tags research ./cmd/mappower-build/ -run VerdictV2`"+
		" (`verdict_v2_*_research_test.go`, `verdict_lignees_research_test.go`). Fichier juge : `%s`"+
		" (cuisson du %s). Reglage serialise dans le fichier : `%s`. Oracle : `%s` (§6 positions,"+
		" §7 contre-exemples).\n\n",
		v.CheminPositions, v.Doc.GenereLe, v.ReglageBrut, verdictV2OracleNom)
	fmt.Fprintf(b, "Regles appliquees, toutes ecrites avant la mesure (plan D11, oracle v2 §10) :\n\n"+
		"- **retrouvee** : le polygone de la position couvre >= %.0f %% de l'aire de la zone attendue,"+
		" OU son barycentre tombe dans la zone ;\n"+
		"- **rappel** = zones `forte` retrouvees / zones `forte` RESOLUES au vocabulaire du depot ;\n"+
		"- **precision v2** = positions qui TOUCHENT une zone `forte` (meme relation que « retrouvee ») /"+
		" positions calculees — les zones `faible` ne comptent plus ;\n"+
		"- rappel rapporte A PART sur les fortes de raison « hauteur » ou « lignes de vue » (HV), et sur"+
		" celles de raison « arme » ou « objectif » sans hauteur ni vue (AO) ;\n"+
		"- les lignes `zone_en = ?` de l'oracle sont exclues ;\n"+
		"- **temoin GEOGRAPHIQUE** : chaque zone attendue est remplacee par la zone nommee de la carte"+
		" dont le centroide est le plus eloigne ;\n"+
		"- **calibrage** (%s) : rapportees, NON comptees ; **validation** : %s, plus toute carte ou"+
		" l'oracle v2 porte une forte resolue ; sans forte : « hors oracle » ;\n"+
		"- sur une carte a %d fortes, un rappel de 0,50 se lit « indetermine » ;\n"+
		"- contre-exemples (§7) : barycentre dans une zone deconseillee ; piege PUR si l'oracle ne"+
		" decrit jamais la zone comme une position, PARTAGE sinon ; le GO exige ZERO piege PUR.\n\n",
		seuilRecouvrement*100, liste(clesDeMap(cartesCalibrage)), liste(clesDeMap(cartesValidation)),
		fortesIndeterminees)
	fmt.Fprintf(b, "**Live Fire** : les positions v2 (`%s/%s`) ont ete mesurees SANS la variante"+
		" classee « Live Fire - Ranked » (positions de kill fausses en base, decouverte 1 du plan ;"+
		" `--exclure-variantes`, 3 854 kills ecartes selon le document de mesure v2). Elle compte"+
		" donc comme carte de validation ici, contrairement au verdict v1 ou elle etait contaminee.\n\n",
		verdictV2Mesures, verdictV2PositionsNom)
	fmt.Fprintf(b, "Les aires sont mesurees par echantillonnage au pas de %.1f m.\n\n", pasEchantillonM)
}

// clesDeMap rend les cles d'un ensemble.
func clesDeMap(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// tableParCarteV2 : le tableau principal.
func tableParCarteV2(b *strings.Builder, ordre []string, reels, temoins map[string]bilanV2) {
	fmt.Fprintf(b, "## 1. Rappel, precision, temoin — carte par carte\n\n")
	fmt.Fprintln(b, "| Carte | Role | Cle oracle | `forte` (resolues/total) | Rappel | Rappel temoin |"+
		" Rappel HV (n) | Rappel AO (n) | Positions | Precision (fortes) | Precision temoin | Lecture |")
	fmt.Fprintln(b, "|---|---|---|---|---|---|---|---|---|---|---|---|")
	for _, nom := range ordre {
		r, tm := reels[nom], temoins[nom]
		fmt.Fprintf(b, "| %s | %s | `%s` | %d/%d | %s | %s | %s (%d) | %s (%d) | %d | %s | %s | %s |\n",
			nom, r.Role, r.CarteCle, r.NbResolues, r.NbFortes,
			pourcent(r.Rappel), pourcent(tm.Rappel),
			pourcent(r.RappelHauteurVue), r.NbHauteurVue, pourcent(r.RappelArmeObjectif), r.NbArmeObjectif,
			len(r.Positions), pourcent(r.PrecisionForte), pourcent(tm.PrecisionForte), lectureV2(r))
	}
	fmt.Fprintln(b)
	for _, nom := range ordre {
		if r := reels[nom]; r.NbFortesAutreRaison > 0 {
			fmt.Fprintf(b, "- %s : %d forte(s) ni HV ni AO, comptees au rappel total seulement : %s\n",
				nom, r.NbFortesAutreRaison, liste(r.FortesAutreRaisonNoms))
		}
	}
	fmt.Fprintln(b)
}

// lectureV2 qualifie la ligne selon le role de la carte et les regles du pilote.
func lectureV2(b bilanV2) string {
	switch {
	case b.Role == roleHorsOracle:
		return "hors oracle (aucune forte)"
	case b.NbResolues == 0:
		return b.Role + " — aucune forte resolue au vocabulaire"
	case len(b.Positions) == 0:
		return b.Role + " — aucune position calculee : ECHOUE"
	}
	mot := "ECHOUE"
	switch {
	case tient(b):
		mot = "TIENT"
	case b.NbResolues <= fortesIndeterminees && b.Rappel == 0.5:
		mot = "indetermine (rappel 0,5 sur " + fmt.Sprint(b.NbResolues) + " fortes)"
	}
	if b.Role == roleCalibrage {
		return "calibrage — rapportee, NON comptee (" + mot + ")"
	}
	return mot
}

// tableTemoinGeographique : l'ecart entre le reel et le remplacement par la zone la plus
// eloignee.
func tableTemoinGeographique(b *strings.Builder, ordre []string, reels, temoins map[string]bilanV2) {
	fmt.Fprintf(b, "## 2. Temoin negatif GEOGRAPHIQUE — chaque forte remplacee par la zone la plus eloignee\n\n")
	fmt.Fprintf(b, "Le temoin v1 (permutation alphabetique) deplacait souvent l'attente de quelques metres."+
		" Ici la zone attendue est remplacee par celle dont le centroide est LE PLUS ELOIGNE sur la"+
		" carte : si rappel et precision ne s'effondrent pas, l'appariement ne mesure que la densite"+
		" des zones, pas leur identite.\n\n")
	fmt.Fprintln(b, "| Carte | Rappel reel | Rappel temoin | Delta | Precision reelle | Precision temoin | Delta | Remplacements des fortes (distance des centroides) |")
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
			pourcent(r.PrecisionForte), pourcent(tm.PrecisionForte), tm.PrecisionForte-r.PrecisionForte,
			liste(tm.Permutation))
		sr, st, pr, pt = sr+r.Rappel, st+tm.Rappel, pr+r.PrecisionForte, pt+tm.PrecisionForte
		n++
	}
	if n > 0 {
		fmt.Fprintf(b, "| **moyenne** | **%s** | **%s** | **%+.2f** | **%s** | **%s** | **%+.2f** | |\n",
			pourcent(sr/float64(n)), pourcent(st/float64(n)), (st-sr)/float64(n),
			pourcent(pr/float64(n)), pourcent(pt/float64(n)), (pt-pr)/float64(n))
	}
	fmt.Fprintln(b)
}

// tableContreExemplesV2 : les positions posees sur un lieu que les guides deconseillent.
func tableContreExemplesV2(b *strings.Builder, ordre []string, reels map[string]bilanV2) {
	fmt.Fprintf(b, "## 3. Contre-exemples (§7 de l'oracle v2)\n\n")
	fmt.Fprintf(b, "Une position est un contre-exemple quand son BARYCENTRE tombe dans une zone"+
		" deconseillee. PURS : l'oracle ne decrit la zone nulle part comme une position. PARTAGES :"+
		" la zone est les deux a la fois dans l'oracle (Long Hall, Market, Pit, East / West Tower...)."+
		" **Le GO porte sur les PURS des cartes de validation.**\n\n")
	fmt.Fprintln(b, "| Carte | Role | Pieges PURS colores | Pieges aussi decrits comme positions |")
	fmt.Fprintln(b, "|---|---|---|---|")
	total, validation := 0, 0
	for _, nom := range ordre {
		r := reels[nom]
		total += len(r.ContreExemplesPurs)
		if r.Role == roleValidation {
			validation += len(r.ContreExemplesPurs)
		}
		fmt.Fprintf(b, "| %s | %s | %s | %s |\n", nom, r.Role, liste(r.ContreExemplesPurs),
			liste(r.ContreExemplesPartage))
	}
	fmt.Fprintf(b, "\n**Total des pieges PURS colores : %d**, dont **%d sur les cartes de validation**"+
		" (c'est ce compte-la qui engage le GO).\n\n", total, validation)
}

// tableFautesV2 : faux negatifs et faux positifs nommes, par groupe de raison.
func tableFautesV2(b *strings.Builder, ordre []string, reels map[string]bilanV2) {
	fmt.Fprintf(b, "## 4. Faux positifs et faux negatifs, nommes\n\n")
	fmt.Fprintln(b, "| Carte | Faux negatifs HV (hauteur / lignes de vue) | Faux negatifs AO (arme / objectif) |"+
		" Faux positifs (positions sans forte, avec leur zone dominante) | `forte` NON RESOLUES au vocabulaire |")
	fmt.Fprintln(b, "|---|---|---|---|---|")
	for _, nom := range ordre {
		r := reels[nom]
		fmt.Fprintf(b, "| %s | %s | %s | %s | %s |\n", nom, liste(r.FauxNegatifsHauteur),
			liste(r.FauxNegatifsArmeObj), liste(r.FauxPositifs), liste(r.FortesNonResolu))
	}
	fmt.Fprintf(b, "\nUne forte NON RESOLUE est absente du catalogue de zones du depot pour cette"+
		" carte : l'algorithme ne peut ni la trouver ni la manquer ; elle est hors du denominateur"+
		" du rappel. Un faux positif est une position qui ne touche aucune forte — il peut"+
		" toucher une zone `faible` de l'oracle, ce qui ne compte plus.\n\n")
}
