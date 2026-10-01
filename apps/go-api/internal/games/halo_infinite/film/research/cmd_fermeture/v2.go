//go:build research

package main

// v2.go — LA CARTE DE FERMETURE V2 (campagne de recherche sur la grammaire, phase 1, etape 1,
// 2026-10-01), mode `-mode v2` : la carte de `fermeture` (memes TSV, memes colonnes, memes
// valeurs : la mesure passe par `grammar.FrameClosureDetaillee`, qui rend la carte de
// `grammar.FrameClosure`), PLUS ce que le detail de chaque paquet permet de ventiler :
//
//   - item 1.1 : « vue C : terminateur hors cadre » par sortie de la vue B, par vue C (vide ou
//     nombre d entrees), par reste du payload, et par dernier composant lu ;
//   - item 1.2 : pour chaque sortie de vue B par REJET, l etat du slot rejete dans le bloc de
//     type 1 du chunk, et la naissance non lue de l eid quand le bloc du chunk SUIVANT la montre ;
//   - item 1.3 : le denominateur des entrees de controle UTILES (celles que le tir continu lit) ;
//   - item 1.5 : le compte d evenements DECLARE du chunk des temps forts contre ceux trouves ;
//   - item 1.6 : le mode borne — les lectures qui depassent la fin du payload.
//
// Ce fichier tient les mesures d UN film ; `v2_rapport.go` les ecrit et les cumule par build.

import (
	"fmt"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// compte : des paquets, et les records utiles en jeu qu ils portent.
type compte struct{ paquets, enJeu int }

// cleHorsCadre ventile un paquet « hors cadre » (item 1.1).
type cleHorsCadre struct{ sortie, vueC, reste string }

// cleDernier ventile un paquet « hors cadre » par son dernier composant lu (item 1.1).
type cleDernier struct{ sortie, dernier string }

// cleRejet ventile une sortie de vue B par rejet (item 1.2).
type cleRejet struct{ sortie, paquet, etat, naissance string }

// sortieStat compte une sortie de vue B sur tous les paquets qui l atteignent.
type sortieStat struct{ paquets, fermes, horsCadre, autres, enJeuHorsCadre int }

// entreesV2 porte le denominateur des entrees de controle utiles (item 1.3).
type entreesV2 struct {
	paquets, nonFermes, vuesCFermees, entreesFermees, utilesFermees int
	// utilesLuesHorsFermeture : entrees a bloc lues dans une vue C qui NE ferme PAS le paquet —
	// lues a une position non prouvee, publiees pour memoire, jamais comptees.
	utilesLuesHorsFermeture int
}

// borneV2 porte le mode borne (item 1.6).
type borneV2 struct {
	records, debordants, composants, neufs, paquets, terminateursAuDela int
}

// mesureV2 est ce que la carte v2 a mesure d UN film.
type mesureV2 struct {
	sorties   map[string]*sortieStat
	horsCadre map[cleHorsCadre]*compte
	dernier   map[cleDernier]*compte
	rejets    map[cleRejet]*compte
	entrees   entreesV2
	borne     borneV2
	chunk3    chunk3
	// sansBloc / blocsIllisibles : chunks CONSULTES (celui d un rejet et son suivant) sans paquet de
	// type 1, et blocs de type 1 que `grammar.LireBlocDeDatums` refuse.
	sansBloc, blocsIllisibles int
}

func nouvelleMesureV2() *mesureV2 {
	return &mesureV2{sorties: map[string]*sortieStat{}, horsCadre: map[cleHorsCadre]*compte{},
		dernier: map[cleDernier]*compte{}, rejets: map[cleRejet]*compte{}}
}

// classeDeVueC rend la classe de la vue C d un paquet « hors cadre » : vide (son terminateur
// seul), ou son nombre d entrees de controle.
func classeDeVueC(c grammar.FluxVueC) string {
	n := len(c.Entrees)
	switch {
	case c.Vide:
		return "vide"
	case n == 0:
		return "0 entree (kind 3 seul)"
	case n >= 9:
		return "9+ entrees"
	}
	return fmt.Sprintf("%d entree(s)", n)
}

// classeDeReste rend la classe du reste du payload derriere la vue C.
func classeDeReste(reste int) string {
	switch {
	case reste < 0:
		return "negatif"
	case reste <= 7:
		return "0-7 (non nuls)"
	case reste <= 63:
		return "8-63"
	}
	return ">= 64"
}

// classeDePaquet resume le sort d un paquet pour la ventilation des rejets.
func classeDePaquet(p grammar.PaquetDeCarte) string {
	switch {
	case p.Fermee:
		return "ferme"
	case p.Cause == grammar.CauseHorsCadre:
		return "hors cadre"
	}
	return p.Cause
}

// compterPaquet range un paquet dans les mesures qui ne dependent pas du bloc de type 1.
func (m *mesureV2) compterPaquet(p grammar.PaquetDeCarte) {
	m.compterEntrees(p)
	m.compterBorne(p)
	if p.Sortie == grammar.SortieVueBNonAtteinte {
		return
	}
	s := m.sorties[p.Sortie.String()]
	if s == nil {
		s = &sortieStat{}
		m.sorties[p.Sortie.String()] = s
	}
	s.paquets++
	switch {
	case p.Fermee:
		s.fermes++
	case p.Cause == grammar.CauseHorsCadre:
		s.horsCadre++
		s.enJeuHorsCadre += p.UtilesEnJeu
		m.compterHorsCadre(p)
	default:
		s.autres++
	}
}

// compterHorsCadre ventile un paquet « hors cadre » (item 1.1).
func (m *mesureV2) compterHorsCadre(p grammar.PaquetDeCarte) {
	k := cleHorsCadre{sortie: p.Sortie.String(), vueC: classeDeVueC(p.VueC), reste: classeDeReste(p.Bits - p.Curseur)}
	ajouter(m.horsCadre, k, p.UtilesEnJeu)
	ajouter(m.dernier, cleDernier{sortie: p.Sortie.String(), dernier: p.DernierLu}, p.UtilesEnJeu)
}

// ajouter compte un paquet sous une cle.
func ajouter[K comparable](t map[K]*compte, k K, enJeu int) {
	c := t[k]
	if c == nil {
		c = &compte{}
		t[k] = c
	}
	c.paquets++
	c.enJeu += enJeu
}

// compterEntrees tient le denominateur des entrees de controle utiles (item 1.3). Une entree est
// UTILE quand elle porte le bloc de 0x68 octets : c est la seule que le tir continu lit
// (`tir_continu.go`, `collecteurTirContinu.fermer` : `if !e.Bloc { continue }`).
func (m *mesureV2) compterEntrees(p grammar.PaquetDeCarte) {
	m.entrees.paquets++
	utiles := 0
	for _, e := range p.VueC.Entrees {
		if e.Bloc {
			utiles++
		}
	}
	if p.Fermee {
		m.entrees.vuesCFermees++
		m.entrees.entreesFermees += len(p.VueC.Entrees)
		m.entrees.utilesFermees += utiles
		return
	}
	m.entrees.nonFermes++
	m.entrees.utilesLuesHorsFermeture += utiles
}

// compterBorne tient le mode borne (item 1.6).
func (m *mesureV2) compterBorne(p grammar.PaquetDeCarte) {
	b := &m.borne
	b.records += p.RecordsLus
	b.debordants += p.RecordsDebordants
	b.composants += p.ComposantsDebordants
	b.neufs += p.NeufsPropresDebordants
	if p.RecordsDebordants > 0 {
		b.paquets++
	}
	if p.Sortie == grammar.SortieVueBTerminateur && p.FinVueB > p.Bits {
		b.terminateursAuDela++
	}
}

// denominateurEstime rend le denominateur ESTIME des entrees utiles : celles des vues C fermees,
// plus, pour chaque paquet qui ne ferme pas, la moyenne d entrees utiles d une vue C fermee du
// MEME film. Une estimation, nommee comme telle : un paquet qui ne ferme pas ne dit pas combien
// d entrees il portait.
func (e entreesV2) denominateurEstime() float64 {
	if e.vuesCFermees == 0 {
		return 0
	}
	moy := float64(e.utilesFermees) / float64(e.vuesCFermees)
	return float64(e.utilesFermees) + moy*float64(e.nonFermes)
}
