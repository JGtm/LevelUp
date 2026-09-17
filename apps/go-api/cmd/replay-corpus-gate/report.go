package main

// report.go — LE TABLEAU RECAPITULATIF ET LE VERDICT.
//
// En mode BASE (defaut), une perte sur QUELQUE AXE QUE CE SOIT fait sortir le gate en code
// `codePerte` — c'est le seul signal qu'un merge qui touche au decodeur ou au constructeur de
// rejeu doit bloquer sur : la reference est une cuisson du MEME code, un commit plus tot, donc
// toute perte est necessairement due au diff en cours de revue. En mode PARC (balayage de
// release, `--reference=parc`), le parc n'est jamais a jour et une perte peut n'etre que l'age
// de la reference — le tableau reste informatif SAUF `--strict`. Un gain (calque neuf, schema
// bumpe) n'est jamais un echec dans aucun mode : c'est l'evolution attendue du format.
//
// LE DETAIL DES PERTES ET DES CHANGEMENTS EST IMPRIME, PAS SEULEMENT LEUR COMPTE : un operateur
// qui lit « 27 pertes » sans savoir LESQUELLES ne peut pas distinguer un correctif deja
// documente (bornes de scene assainies, drapeau neutre...) d'une regression neuve — exactement
// le risque que ce gate existe pour eliminer (CLAUDE.md, anti-pattern « rapporter, pas
// masquer »).
//
// # LA REGLE DE PRIORITE DES STATUTS (2026-09-17, lot 2.8.1)
//
// Un temoin porte UN SEUL statut, et il se choisit dans CET ORDRE — le premier qui s'applique
// gagne, les suivants ne sont meme pas consultes :
//
//	1. ABSENT      le temoin n'a pas ete compare du tout (film, faits ou artefact manquants).
//	   ERREUR      la cuisson ou la comparaison a echoue. ABSENT et ERREUR s'excluent par
//	               construction (orchestrate.go les pose dans des branches disjointes) ; ils
//	               partagent ce rang parce qu'ils disent la meme chose : PAS DE MESURE.
//	2. PERTE       au moins une mesure a baisse ou disparu. PRIME SUR `CHANGEMENT` : un temoin
//	               qui porte les deux est un temoin en perte, et c'est la perte qu'on instruit.
//	3. CHANGEMENT  aucune perte, mais au moins une valeur publiee a BOUGE (reattribution, voie
//	               de nommage qui cede a une autre — `replaydiff/polarite.go`). Statut DISTINCT
//	               de la perte depuis le 2026-09-17 : jusque-la un changement sortait `PERTE`,
//	               ce qui envoyait chercher une regression la ou une valeur avait seulement
//	               change de main. Il reste BLOQUANT (`estBloquant`) : un changement se
//	               justifie (divergence prouvee) ou il se corrige, jamais il ne se tait.
//	4. ok          ni perte ni changement. Des GAINS peuvent s'y trouver : un gain n'est jamais
//	               un echec.
//
// Les gains n'entrent nulle part dans ce choix — ils se lisent dans leur colonne.

import (
	"fmt"
	"io"
	"strings"
	"time"

	"levelup/go-api/internal/replaydiff"
)

// Les codes de sortie du gate, NOMMES — le seul canal qu'une CI, un agregateur ou un pilote
// lisent sans parser le tableau. Avant le 2026-09-17 ils etaient des litteraux `1` et `2`
// disperses, et le `2` disait DEUX choses incomparables : « le manifeste est invalide » et
// « un temoin du corpus est absent ». Un appelant ne pouvait pas distinguer « ce gate n'a pas
// demarre » de « ce gate a demarre mais n'a pas tout compare ».
const (
	// codeOK : tout le manifeste a ete compare, aucun temoin bloquant.
	codeOK = 0
	// codePerte : au moins un temoin compare porte une PERTE ou un CHANGEMENT bloquant
	// (`estBloquant`) — le verdict que ce gate existe pour rendre.
	codePerte = 1
	// codeUsage : le gate n'a pas pu DEMARRER (drapeau invalide, manifeste illisible, racine
	// introuvable, capability absente, worktree de base impossible). Aucun temoin n'a ete
	// compare, et rien n'a ete mesure du diff sous revue.
	codeUsage = 2
	// codeErreurCuisson : le gate a demarre, mais au moins un temoin CUIT a echoue a la
	// cuisson ou a la comparaison. Distinct de `codePerte` : le gate n'a pas pu poser la
	// question sur ce temoin, il n'a pas repondu « il a perdu ».
	codeErreurCuisson = 3
	// codeCouvertureIncomplete : au moins un temoin du manifeste est ABSENT (film, faits ou
	// artefact de reference manquants) et `--allow-missing` n'a pas ete passe. Distinct de
	// `codePerte` (D2 (cloture M1), 2026-09-17 : l'export des faits peut tomber sur la base
	// tenue en ecriture, et un pilote doit pouvoir relancer les seuls absents — `--temoins` —
	// au lieu de chercher une regression qui n'existe pas) ET de `codeUsage` (le manifeste,
	// lui, est valide).
	codeCouvertureIncomplete = 4
)

// Les statuts d'un temoin, tels qu'ils s'impriment au tableau et s'ecrivent au JSON. Ils sont
// nommes ici parce que le tableau, le JSON et les tests les partagent : trois litteraux
// separes divergeraient au premier renommage.
const (
	statutOK         = "ok"
	statutChangement = "CHANGEMENT"
	statutPerte      = "PERTE"
	statutAbsent     = "ABSENT"
	statutErreur     = "ERREUR"
)

// ligneRapport est le resultat d'UN temoin, pret a s'imprimer. `SchemaReference` porte le
// schema de l'artefact COMPARE — celui d'une cuisson a la base (mode base) ou celui deja cuit
// dans le parc (mode parc) : le meme champ sert les deux modes, seul le LIBELLE de colonne
// change (cf. imprimerTableau).
type ligneRapport struct {
	Temoin          Temoin
	Absent          bool // vrai : temoin introuvable (parc, ou base sans ce commit)
	AbsentCause     string
	Erreur          error // non nil : la cuisson ou la comparaison a echoue (distinct d'« absent »)
	SchemaReference int
	SchemaHEAD      int
	Gains           int
	Pertes          int
	// Changements : les valeurs publiees qui BOUGENT sans etre ni un gain ni une perte.
	// AJOUTE LE 2026-09-16 (lot 1.9.1 bis, cloture) : `replaydiff.BilanAxe` les compte depuis
	// toujours, et ce rapport les jetait — ni le tableau ni le JSON ne les montraient. Un gate
	// aveugle a une valeur qui bouge est exactement le compteur lu a l envers que le depot
	// interdit.
	Changements int
	Duree       time.Duration
	// PertesDetail : les differences de sens PERTE ou DISPARU seulement — c'est LE FAIT a
	// rapporter, jamais a resumer en un seul compte.
	PertesDetail []replaydiff.Difference
	// TelemetrieDetail : les feuilles de TELEMETRIE qui ont bouge (lot 3.3.3) — les revisions du
	// decodeur, le build, l empreinte de registre. Elles ne comptent NI en gain, NI en perte, NI
	// en changement (cf. verdict_metriques.go) et s impriment dans leur propre section : une
	// revision qui monte est attendue a chaque lot, la voir est utile, en faire un verdict non.
	TelemetrieDetail []replaydiff.Difference
	// ChangementsDetail : les differences de sens CHANGEMENT, symetrique de PertesDetail
	// (2026-09-17, D5 (1.9.9)). Sans lui, le JSON du gate disait « 2 changements » et le
	// pilote devait relancer `replay-diff` a la main sur les artefacts conserves pour savoir
	// LESQUELS — ce qui est arrive a la cloture M1 (plan §5, les 7 changements nommes).
	ChangementsDetail []replaydiff.Difference
}

// aUnePerte dit si CE temoin porte au moins une mesure qui a baisse ou disparu.
func (l ligneRapport) aUnePerte() bool { return l.Pertes > 0 }

// aUnChangement dit si CE temoin porte au moins une valeur publiee qui a BOUGE sans etre ni un
// gain ni une perte.
func (l ligneRapport) aUnChangement() bool { return l.Changements > 0 }

// estBloquant dit si CE temoin doit faire echouer le gate : une perte OU un changement. Les
// deux bloquent, et ils ne se confondent pas pour autant — cf. la regle de priorite en tete.
func (l ligneRapport) estBloquant() bool { return l.aUnePerte() || l.aUnChangement() }

// statut rend le statut du temoin selon la regle de priorite de l'en-tete de ce fichier.
func (l ligneRapport) statut() string {
	switch {
	case l.Absent:
		return statutAbsent
	case l.Erreur != nil:
		return statutErreur
	case l.aUnePerte():
		return statutPerte
	case l.aUnChangement():
		return statutChangement
	}
	return statutOK
}

// remplirBilan peuple schemas, comptes et LES TROIS DETAILS depuis un `replaydiff.Rapport`.
//
// C'etait une fonction a six valeurs de retour ; la septieme (le detail des changements)
// l'aurait rendue illisible a l'appel. Une methode qui peuple la ligne dit la meme chose sans
// aligner sept resultats anonymes (CLAUDE.md n°5).
//
// ELLE NE SOMME PLUS `rap.Bilans` DEPUIS LE LOT 3.3.3, ET C'EST LE POINT : les bilans par axe
// portent le sens que `replaydiff` a MESURE, et le gate en retient un autre sur deux familles —
// la TELEMETRIE (jamais comptee) et les COMPTEURS DE REJET dont le denominateur a bouge
// (`verdict_metriques.go`). Les comptes se recalculent donc DEPUIS LES ECARTS, qui sont la meme
// population que les bilans (`replaydiff.Rapport.ajouter` alimente les deux d'un seul geste) :
// aucune mesure n'est perdue, seule la CLASSIFICATION change.
func (l *ligneRapport) remplirBilan(rap replaydiff.Rapport) {
	l.SchemaReference, l.SchemaHEAD = rap.SchemaAncien, rap.SchemaNouveau
	l.Gains, l.Pertes, l.Changements = 0, 0, 0
	l.PertesDetail, l.ChangementsDetail, l.TelemetrieDetail = nil, nil, nil
	parMetrique := make(map[string]replaydiff.Difference, len(rap.Differences))
	for _, d := range rap.Differences {
		parMetrique[d.Metrique] = d
	}
	for _, d := range rap.Differences {
		switch classerPourLeVerdict(d, parMetrique) {
		case sensTelemetrie:
			l.TelemetrieDetail = append(l.TelemetrieDetail, d)
		case replaydiff.SensPerte, replaydiff.SensDisparu:
			l.Pertes++
			l.PertesDetail = append(l.PertesDetail, d)
		case replaydiff.SensChangement:
			l.Changements++
			l.ChangementsDetail = append(l.ChangementsDetail, d)
		default:
			l.Gains++
		}
	}
}

// imprimerTableau ecrit le recapitulatif — un temoin par ligne, dans l'ordre du manifeste.
// `refLabel` nomme la colonne de reference ("base" ou "parc") — c'est la SEULE chose qui
// distingue l'affichage des deux modes, la structure de ligneRapport est commune aux deux.
func imprimerTableau(w io.Writer, lignes []ligneRapport, refLabel string) {
	_, _ = fmt.Fprintf(w, "%-12s %-16s %6s %6s %8s %8s %8s %10s  %s\n",
		"temoin", "famille", refLabel, "HEAD", "gains", "pertes", "chang.", "duree", "statut")
	for _, l := range lignes {
		switch {
		case l.Absent:
			_, _ = fmt.Fprintf(w, "%-12s %-16s %6s %6s %8s %8s %8s %10s  %s (%s)\n",
				l.Temoin.ID, l.Temoin.Famille, "-", "-", "-", "-", "-", "-",
				statutAbsent, l.AbsentCause)
		case l.Erreur != nil:
			_, _ = fmt.Fprintf(w, "%-12s %-16s %6s %6s %8s %8s %8s %10s  %s : %v\n",
				l.Temoin.ID, l.Temoin.Famille, "-", "-", "-", "-", "-", "-",
				statutErreur, l.Erreur)
		default:
			_, _ = fmt.Fprintf(w, "%-12s %-16s %6d %6d %8d %8d %8d %10s  %s\n",
				l.Temoin.ID, l.Temoin.Famille, l.SchemaReference, l.SchemaHEAD,
				l.Gains, l.Pertes, l.Changements, l.Duree.Round(10*time.Millisecond), l.statut())
		}
	}
}

// imprimerDetailPertes ecrit, POUR CHAQUE TEMOIN EN PERTE, la liste nommee de ses ecarts —
// axe, metrique, ancien -> nouveau. Rien n'est resume : « rapporter, pas masquer ».
func imprimerDetailPertes(w io.Writer, lignes []ligneRapport) {
	imprimerDetail(w, lignes, "DETAIL DES PERTES",
		func(l ligneRapport) []replaydiff.Difference { return l.PertesDetail })
}

// imprimerDetailChangements ecrit, POUR CHAQUE TEMOIN QUI EN PORTE, la liste nommee de ses
// changements (2026-09-17, D5 (1.9.9)) — la section symetrique de celle des pertes. Un
// changement se JUSTIFIE (reattribution documentee, voie de nommage qui cede) ou il se
// corrige ; dans les deux cas il faut le NOMMER, et le compte ne le nomme pas.
func imprimerDetailChangements(w io.Writer, lignes []ligneRapport) {
	imprimerDetail(w, lignes, "DETAIL DES CHANGEMENTS",
		func(l ligneRapport) []replaydiff.Difference { return l.ChangementsDetail })
}

// imprimerDetail est LE rendu partage des deux sections de detail : meme en-tete, meme
// colonnes, meme silence quand il n'y a rien a dire. Ecrire deux fois ces quinze lignes les
// ferait diverger au premier ajout de colonne (CLAUDE.md n°6).
func imprimerDetail(w io.Writer, lignes []ligneRapport, titre string,
	detailDe func(ligneRapport) []replaydiff.Difference) {
	var concernes []ligneRapport
	for _, l := range lignes {
		if len(detailDe(l)) > 0 {
			concernes = append(concernes, l)
		}
	}
	if len(concernes) == 0 {
		return
	}
	_, _ = fmt.Fprintf(w, "\n%s (%d temoin(s)) :\n", titre, len(concernes))
	for _, l := range concernes {
		_, _ = fmt.Fprintf(w, "\n  [%s] %s (schema %d -> %d)\n",
			l.Temoin.ID, l.Temoin.Famille, l.SchemaReference, l.SchemaHEAD)
		for _, d := range detailDe(l) {
			_, _ = fmt.Fprintf(w, "    %-9s %-16s %-50s %10s -> %-10s\n",
				d.Sens, d.Axe, d.Metrique, vide(d.Ancien), vide(d.Nouveau))
		}
	}
}

func vide(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

// codeSortie rend `codeErreurCuisson` si un temoin CUIT porte une erreur de cuisson ou de
// comparaison (TOUJOURS bloquant, quel que soit le mode : le gate n'a alors pas pu faire son
// travail), `codePerte` si `pertesBloquent` est vrai et qu'un temoin est bloquant (perte OU
// changement), `codeOK` sinon. `pertesBloquent` vaut `reference == "base"` (toujours) ou
// `--strict` (mode parc) — cf. l'en-tete du fichier.
//
// L'ERREUR DE CUISSON PRIME SUR LA PERTE : elle se rencontre en premier dans la boucle et rend
// tout de suite, parce qu'un gate qui n'a pas pu cuire un temoin ne sait PAS si les autres
// auraient perdu. Un temoin ABSENT n'est jamais un echec ICI (avertissement deja emis en
// slog) : la COUVERTURE (au moins un temoin absent = un gate qui ne compare pas tout) est
// verifiee separement par [verifierCouverture] et rend `codeCouvertureIncomplete` — les deux ne
// se substituent pas l'une a l'autre.
func codeSortie(lignes []ligneRapport, pertesBloquent bool) int {
	code := codeOK
	for _, l := range lignes {
		if l.Absent {
			continue
		}
		if l.Erreur != nil {
			return codeErreurCuisson
		}
		if pertesBloquent && l.estBloquant() {
			code = codePerte
		}
	}
	return code
}

// verifierCouverture impose qu'AUCUN temoin du manifeste ne soit ABSENT, sauf si `allowMissing`
// est vrai — CORPUS-R1 C3 (L6, P0) : un cache de film purge ou partiel rend TOUS les temoins
// ABSENT, et `codeSortie` ci-dessus les saute tous (`continue`) sans jamais rencontrer ni
// erreur ni perte — le gate sortait alors en 0 SANS RIEN COMPARER, le silence le plus dangereux
// qu'un gate de non-regression puisse rendre. Par defaut, un seul temoin absent est donc une
// ERREUR DE COUVERTURE (`codeCouvertureIncomplete`, distincte d'une perte, d'une erreur de
// cuisson et d'un usage invalide) — `--allow-missing` restaure l'ancien comportement
// (avertissement seul) pour un usage delibere.
func verifierCouverture(lignes []ligneRapport, allowMissing bool) error {
	if allowMissing {
		return nil
	}
	var absents []string
	for _, l := range lignes {
		if l.Absent {
			absents = append(absents, fmt.Sprintf("%s (%s) : %s", l.Temoin.ID, l.Temoin.Famille, l.AbsentCause))
		}
	}
	if len(absents) == 0 {
		return nil
	}
	return fmt.Errorf("couverture incomplete : %d/%d temoin(s) du manifeste absent(s) — "+
		"un gate qui ne compare pas un temoin ne le garde pas (passer --allow-missing pour "+
		"tolerer deliberement) :\n  %s",
		len(absents), len(lignes), strings.Join(absents, "\n  "))
}
