package main

// report.go — LE TABLEAU RECAPITULATIF ET LE VERDICT.
//
// En mode BASE (defaut), une perte sur QUELQUE AXE QUE CE SOIT fait sortir le gate en code 1 —
// c'est le seul signal qu'un merge qui touche au decodeur ou au constructeur de rejeu doit
// bloquer sur : la reference est une cuisson du MEME code, un commit plus tot, donc toute perte
// est necessairement due au diff en cours de revue. En mode PARC (balayage de release,
// `--reference=parc`), le parc n'est jamais a jour et une perte peut n'etre que l'age de la
// reference — le tableau reste informatif SAUF `--strict`. Un gain (calque neuf, schema bumpe)
// n'est jamais un echec dans aucun mode : c'est l'evolution attendue du format.
//
// LE DETAIL DES PERTES EST IMPRIME, PAS SEULEMENT LEUR COMPTE : un operateur qui lit « 27
// pertes » sans savoir LESQUELLES ne peut pas distinguer un correctif deja documente (bornes
// de scene assainies, drapeau neutre...) d'une regression neuve — exactement le risque que ce
// gate existe pour eliminer (CLAUDE.md, anti-pattern « rapporter, pas masquer »).

import (
	"fmt"
	"io"
	"strings"
	"time"

	"levelup/go-api/internal/replaydiff"
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
	Duree           time.Duration
	// PertesDetail : les differences de sens PERTE ou DISPARU seulement (jamais les gains ni
	// les changements) — c'est LE FAIT a rapporter, jamais a resumer en un seul compte.
	PertesDetail []replaydiff.Difference
}

// aUnePerte dit si CE temoin porte au moins une perte sur un axe quelconque.
func (l ligneRapport) aUnePerte() bool { return l.Pertes > 0 }

// bilanDepuisRapport peuple gains/pertes/schemas depuis un replaydiff.Rapport, et extrait le
// DETAIL des pertes (sens Perte ou Disparu uniquement).
func bilanDepuisRapport(rap replaydiff.Rapport) (schemaReference, schemaHEAD, gains, pertes int, detail []replaydiff.Difference) {
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
// `refLabel` nomme la colonne de reference ("base" ou "parc") — c'est la SEULE chose qui
// distingue l'affichage des deux modes, la structure de ligneRapport est commune aux deux.
func imprimerTableau(w io.Writer, lignes []ligneRapport, refLabel string) {
	_, _ = fmt.Fprintf(w, "%-12s %-16s %6s %6s %8s %8s %10s  %s\n",
		"temoin", "famille", refLabel, "HEAD", "gains", "pertes", "duree", "statut")
	for _, l := range lignes {
		switch {
		case l.Absent:
			_, _ = fmt.Fprintf(w, "%-12s %-16s %6s %6s %8s %8s %10s  ABSENT (%s)\n",
				l.Temoin.ID, l.Temoin.Famille, "-", "-", "-", "-", "-", l.AbsentCause)
		case l.Erreur != nil:
			_, _ = fmt.Fprintf(w, "%-12s %-16s %6s %6s %8s %8s %10s  ERREUR : %v\n",
				l.Temoin.ID, l.Temoin.Famille, "-", "-", "-", "-", "-", l.Erreur)
		default:
			statut := "ok"
			if l.aUnePerte() {
				statut = "PERTE"
			}
			_, _ = fmt.Fprintf(w, "%-12s %-16s %6d %6d %8d %8d %10s  %s\n",
				l.Temoin.ID, l.Temoin.Famille, l.SchemaReference, l.SchemaHEAD,
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
	_, _ = fmt.Fprintf(w, "\nDETAIL DES PERTES (%d temoin(s)) :\n", len(enPerte))
	for _, l := range enPerte {
		_, _ = fmt.Fprintf(w, "\n  [%s] %s (schema %d -> %d)\n", l.Temoin.ID, l.Temoin.Famille, l.SchemaReference, l.SchemaHEAD)
		for _, d := range l.PertesDetail {
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

// codeSortie rend 1 si un temoin CUIT porte une erreur de cuisson/comparaison (TOUJOURS
// bloquant, quel que soit le mode : le gate n'a alors pas pu faire son travail), ou si
// `pertesBloquent` est vrai et qu'un temoin porte une perte. `pertesBloquent` vaut
// `reference == "base"` (toujours) ou `--strict` (mode parc) — cf. l'en-tete du fichier. Un
// temoin ABSENT n'est jamais un echec ICI (avertissement deja emis en slog) : la COUVERTURE
// (au moins un temoin absent = un gate qui ne compare rien) est verifiee separement par
// [verifierCouverture], AVANT ce calcul — les deux ne se substituent pas l'une a l'autre.
func codeSortie(lignes []ligneRapport, pertesBloquent bool) int {
	for _, l := range lignes {
		if l.Absent {
			continue
		}
		if l.Erreur != nil {
			return 1
		}
		if pertesBloquent && l.aUnePerte() {
			return 1
		}
	}
	return 0
}

// verifierCouverture impose qu'AUCUN temoin du manifeste ne soit ABSENT, sauf si `allowMissing`
// est vrai — CORPUS-R1 C3 (L6, P0) : un cache de film purge ou partiel rend TOUS les temoins
// ABSENT, et `codeSortie` ci-dessus les saute tous (`continue`) sans jamais rencontrer ni
// erreur ni perte — le gate sortait alors en 0 SANS RIEN COMPARER, le silence le plus dangereux
// qu'un gate de non-regression puisse rendre. Par defaut, un seul temoin absent est donc une
// ERREUR DE COUVERTURE (distincte d'une perte ou d'une erreur de cuisson) — `--allow-missing`
// restaure l'ancien comportement (avertissement seul) pour un usage delibere (par exemple un
// manifeste dont un temoin vient d'etre ajoute avant que son film soit copie localement).
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
