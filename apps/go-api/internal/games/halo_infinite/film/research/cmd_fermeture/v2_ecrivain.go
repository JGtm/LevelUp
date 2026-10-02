//go:build research

package main

// v2_ecrivain.go — CE QUE LA CARTE V2 DIT DES REGLES DE L ECRIVAIN, DES DENOMINATEURS ET DES
// TEMOINS DECALES (lot L0 de la campagne de grammaire). Trois TSV par film, ecrits par
// `v2_ecrivain_tsv.go` :
//
//	fermeture_ecrivain.tsv          fermes au bit pres contre fermes ; par regle de l ecrivain, les
//	                                fermetures qu elle retire (premiere regle), les paquets fermes au
//	                                bit qui la portent, et ceux, non fermes, dont la lecture la porte
//	fermeture_denominateurs.tsv     records utiles lus et fermes, part sur le denominateur VARIABLE
//	                                (les lus) et sur le denominateur FIXE (le maximum consolide donne
//	                                par `-denominateur-fixe`, releve par cette marche s il est depasse)
//	fermeture_temoins_decales.tsv   pour les paquets fermes, la part que la vue C referme depuis un
//	                                depart decale de k bits, k de -8 a 8 : le hasard de l oracle
//
// et, sous `-paquets`, `fermeture_paquets.tsv` : une ligne par paquet delta marche.

import (
	"strings"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// ecrivainV2 porte les mesures du lot L0 d un film (ou d un cumul).
type ecrivainV2 struct {
	paquetsFermesAuBit, paquetsFermes          int
	utilesLus, utilesFermesAuBit, utilesFermes int
	listesLocalisees, listesNonLocalisees      int
	// retires : les fermetures au bit que les regles retirent, par PREMIERE regle.
	retires map[string]*compte
	// portees : par regle, les paquets fermes au bit qui la portent ; nonFermes : les paquets non
	// fermes au bit, vue B atteinte, dont la lecture la porte.
	portees, nonFermes [grammar.NombreDInvariants]int
	// temoins : pour les paquets fermes, ceux que la vue C referme depuis chaque depart decale
	// (index : [grammar.BitDeDecalage]).
	temoins       [2 * grammar.DecalageTemoinMax]int
	temoinsFermes int
}

func nouvelEcrivainV2() ecrivainV2 { return ecrivainV2{retires: map[string]*compte{}} }

// compterEcrivain range un paquet dans les mesures du lot L0.
func (e *ecrivainV2) compterEcrivain(p grammar.PaquetDeCarte) {
	switch {
	case p.ListeNonLocalisee:
		e.listesNonLocalisees++
	case p.ListeLocalisee:
		e.listesLocalisees++
	}
	e.utilesLus += p.UtilesLus
	e.utilesFermes += p.UtilesLus - p.UtilesEnJeu
	if p.FermeeAuBit {
		e.paquetsFermesAuBit++
		e.utilesFermesAuBit += p.UtilesLus
	}
	for v := range grammar.NombreDInvariants {
		if p.Invariants&(uint32(1)<<uint(v)) == 0 {
			continue
		}
		if p.FermeeAuBit {
			e.portees[v]++
		} else if p.DebutVueB >= 0 {
			e.nonFermes[v]++
		}
	}
	if p.FermeeAuBit && !p.Fermee {
		ajouter(e.retires, p.Invariant.String(), p.UtilesLus)
	}
	if !p.Fermee {
		return
	}
	e.paquetsFermes++
	e.temoinsFermes++
	for b := range e.temoins {
		if p.TemoinsDecales&(uint16(1)<<uint(b)) != 0 {
			e.temoins[b]++
		}
	}
}

// cumuler ajoute les mesures d un film.
func (e *ecrivainV2) cumuler(f ecrivainV2) {
	e.paquetsFermesAuBit += f.paquetsFermesAuBit
	e.paquetsFermes += f.paquetsFermes
	e.utilesLus += f.utilesLus
	e.utilesFermesAuBit += f.utilesFermesAuBit
	e.utilesFermes += f.utilesFermes
	e.listesLocalisees += f.listesLocalisees
	e.listesNonLocalisees += f.listesNonLocalisees
	sommer(e.retires, f.retires)
	for v := range e.portees {
		e.portees[v] += f.portees[v]
		e.nonFermes[v] += f.nonFermes[v]
	}
	for b := range e.temoins {
		e.temoins[b] += f.temoins[b]
	}
	e.temoinsFermes += f.temoinsFermes
}

// decalageDuBit rend le decalage k que porte le bit `b` de [grammar.PaquetDeCarte.TemoinsDecales].
func decalageDuBit(b int) int {
	if b < grammar.DecalageTemoinMax {
		return b - grammar.DecalageTemoinMax
	}
	return b - grammar.DecalageTemoinMax + 1
}

// nomsDesRegles rend les noms des regles d un ensemble, joints par « + » ; « - » s il est vide.
func nomsDesRegles(ensemble uint32) string {
	var noms []string
	for v := range grammar.NombreDInvariants {
		if ensemble&(uint32(1)<<uint(v)) != 0 {
			noms = append(noms, grammar.InvariantEcrivain(v).String())
		}
	}
	if len(noms) == 0 {
		return "-"
	}
	return strings.Join(noms, " + ")
}
