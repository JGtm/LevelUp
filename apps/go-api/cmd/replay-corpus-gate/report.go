package main

// report.go — LE TABLEAU RECAPITULATIF ET LE VERDICT.
//
// Une perte, sur QUELQUE AXE QUE CE SOIT, fait sortir le gate en code 1 — c'est le seul signal
// qu'un merge qui touche au decodeur ou au constructeur de rejeu doit bloquer sur. Un gain
// (calque neuf, schema bumpe) n'est jamais un echec : c'est l'evolution attendue du format.
//
// LE DETAIL DES PERTES EST IMPRIME, PAS SEULEMENT LEUR COMPTE : un operateur qui lit « 27
// pertes » sans savoir LESQUELLES ne peut pas distinguer un correctif deja documente (bornes
// de scene assainies, drapeau neutre...) d'une regression neuve — exactement le risque que ce
// gate existe pour eliminer (CLAUDE.md, anti-pattern « rapporter, pas masquer »).

import (
	"fmt"
	"io"
	"time"

	"levelup/go-api/internal/replaydiff"
)

// ligneRapport est le resultat d'UN temoin, pret a s'imprimer.
type ligneRapport struct {
	Temoin      Temoin
	Absent      bool // vrai : temoin introuvable au parc (chunks ou reference manquants)
	AbsentCause string
	Erreur      error // non nil : la cuisson ou la comparaison a echoue (distinct d'« absent »)
	SchemaParc  int
	SchemaHEAD  int
	Gains       int
	Pertes      int
	Duree       time.Duration
	// PertesDetail : les differences de sens PERTE ou DISPARU seulement (jamais les gains ni
	// les changements) — c'est LE FAIT a rapporter, jamais a resumer en un seul compte.
	PertesDetail []replaydiff.Difference
}

// aUnePerte dit si CE temoin porte au moins une perte sur un axe quelconque.
func (l ligneRapport) aUnePerte() bool { return l.Pertes > 0 }

// bilanDepuisRapport peuple gains/pertes/schemas depuis un replaydiff.Rapport, et extrait le
// DETAIL des pertes (sens Perte ou Disparu uniquement).
func bilanDepuisRapport(rap replaydiff.Rapport) (schemaParc, schemaHEAD, gains, pertes int, detail []replaydiff.Difference) {
	for _, b := range rap.Bilans {
		gains += b.Gains
		pertes += b.Pertes
	}
	for _, d := range rap.Differences {
		if d.Sens == replaydiff.SensPerte || d.Sens == replaydiff.SensDisparu {
			detail = append(detail, d)
		}
	}
	return rap.SchemaAncien, rap.SchemaNouveau, gains, pertes, detail
}

// imprimerTableau ecrit le recapitulatif — un temoin par ligne, dans l'ordre du manifeste.
func imprimerTableau(w io.Writer, lignes []ligneRapport) {
	fmt.Fprintf(w, "%-12s %-16s %6s %6s %8s %8s %10s  %s\n",
		"temoin", "famille", "parc", "HEAD", "gains", "pertes", "duree", "statut")
	for _, l := range lignes {
		switch {
		case l.Absent:
			fmt.Fprintf(w, "%-12s %-16s %6s %6s %8s %8s %10s  ABSENT DU PARC (%s)\n",
				l.Temoin.ID, l.Temoin.Famille, "-", "-", "-", "-", "-", l.AbsentCause)
		case l.Erreur != nil:
			fmt.Fprintf(w, "%-12s %-16s %6s %6s %8s %8s %10s  ERREUR : %v\n",
				l.Temoin.ID, l.Temoin.Famille, "-", "-", "-", "-", "-", l.Erreur)
		default:
			statut := "ok"
			if l.aUnePerte() {
				statut = "PERTE"
			}
			fmt.Fprintf(w, "%-12s %-16s %6d %6d %8d %8d %10s  %s\n",
				l.Temoin.ID, l.Temoin.Famille, l.SchemaParc, l.SchemaHEAD,
				l.Gains, l.Pertes, l.Duree.Round(10*time.Millisecond), statut)
		}
	}
}

// imprimerDetailPertes ecrit, POUR CHAQUE TEMOIN EN PERTE, la liste nommee de ses ecarts —
// axe, metrique, ancien -> nouveau. Rien n'est resume : « rapporter, pas masquer ».
func imprimerDetailPertes(w io.Writer, lignes []ligneRapport) {
	var enPerte []ligneRapport
	for _, l := range lignes {
		if l.aUnePerte() {
			enPerte = append(enPerte, l)
		}
	}
	if len(enPerte) == 0 {
		return
	}
	fmt.Fprintf(w, "\nDETAIL DES PERTES (%d temoin(s)) :\n", len(enPerte))
	for _, l := range enPerte {
		fmt.Fprintf(w, "\n  [%s] %s (schema %d -> %d)\n", l.Temoin.ID, l.Temoin.Famille, l.SchemaParc, l.SchemaHEAD)
		for _, d := range l.PertesDetail {
			fmt.Fprintf(w, "    %-9s %-16s %-50s %10s -> %-10s\n",
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

// codeSortie rend 1 si un temoin CUIT porte une perte ou une erreur de cuisson/comparaison ;
// 0 sinon. Un temoin ABSENT du parc n'est PAS un echec (avertissement deja emis en slog) —
// sans quoi un poste sans parc local ferait toujours echouer le gate, ce que le prompt exclut
// explicitement.
func codeSortie(lignes []ligneRapport) int {
	for _, l := range lignes {
		if l.Absent {
			continue
		}
		if l.Erreur != nil || l.aUnePerte() {
			return 1
		}
	}
	return 0
}
