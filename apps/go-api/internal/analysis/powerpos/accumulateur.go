package powerpos

import (
	"math"
	"sort"

	"levelup/go-api/internal/analysis/tactical"
)

// Accumulateur agrege des eliminations et des occupations sur la grille tactique.
//
// Table CREUSE, comme `tactical.Raster` : une cellule jamais atteinte n'y figure pas.
type Accumulateur struct {
	grille tactical.Grille
	pond   Ponderation

	cellules map[tactical.Cellule]*celluleAcc

	// matchs : tous les matchs qui ont alimente au moins un echantillon, toutes sources
	// confondues. Sert au rapport (« combien de matchs derriere cette carte »).
	matchs map[string]struct{}

	killsIgnores      int
	presencesIgnorees int
}

// celluleAcc est l'etat mutable d'une cellule pendant l'accumulation. Les portees et les
// denivelees sont gardees en LISTE parce que la mesure retenue est une MEDIANE : une
// moyenne serait tiree par les kills de trop longue portee (un tir de sniper a travers la
// carte) et par les chutes (denivele de plusieurs dizaines de metres).
type celluleAcc struct {
	killsDepuis int
	mortsDedans int
	portees     []float64
	denivelees  []float64

	matchsKills    map[string]struct{}
	matchsPresence map[string]struct{}

	occupGagnants float64
	occupPerdants float64

	dirSortante SommeAngulaire
	dirEntrante SommeAngulaire

	killsPonderes  float64
	mortsPonderees float64
	killsRangConnu int
	mortsRangConnu int
	killsTueurFort int
	mortsTueurFort int
}

// NouvelAccumulateur construit un accumulateur sur la grille donnee, SANS ponderation de
// rang (tout pese 1,0).
func NouvelAccumulateur(g tactical.Grille) *Accumulateur {
	return NouvelAccumulateurPondere(g, PonderationNeutre())
}

// NouvelAccumulateurPondere construit un accumulateur qui penche chaque elimination selon
// le rang de son tueur. Les bornes de la ponderation se mesurent sur le corpus AVANT
// l'accumulation (quantiles des rangs vus) : c'est l'appelant qui les etablit, ce paquet
// ne lit rien.
func NouvelAccumulateurPondere(g tactical.Grille, pond Ponderation) *Accumulateur {
	return &Accumulateur{
		grille:   g,
		pond:     pond,
		cellules: make(map[tactical.Cellule]*celluleAcc),
		matchs:   make(map[string]struct{}),
	}
}

// PasM rend le pas de la grille, en metres.
func (a *Accumulateur) PasM() float64 { return a.grille.PasM() }

// AjouteKill compte une elimination : un `KillsDepuis` dans la cellule du tueur, un
// `MortsDedans` dans celle de la victime, et la portee comme le denivele dans celle du
// TUEUR (ce sont des proprietes du poste de tir).
//
// L'echantillon est ECARTE EN ENTIER si l'une des deux extremites n'est pas une position
// finie — cf. doc.go, une mort sans tueur n'est pas un duel.
func (a *Accumulateur) AjouteKill(k KillSample) {
	cTueur, okT := a.grille.Cellule(k.KillerX, k.KillerY)
	cVictime, okV := a.grille.Cellule(k.VictimX, k.VictimY)
	if !okT || !okV || !finis(k.KillerZ, k.VictimZ) {
		a.killsIgnores++
		return
	}
	a.matchs[k.MatchID] = struct{}{}
	poids := a.pond.Poids(k.RangTueur)
	fort := a.pond.EstFort(k.RangTueur)

	tueur := a.cellule(cTueur)
	tueur.killsDepuis++
	tueur.matchsKills[k.MatchID] = struct{}{}
	tueur.portees = append(tueur.portees, distance3D(k))
	tueur.denivelees = append(tueur.denivelees, k.KillerZ-k.VictimZ)
	// La direction part de la CELLULE, pas de la position exacte du tueur : c'est la
	// cellule qu'on publie, et deux kills partis des deux coins d'une meme cellule ne
	// pointent pas la meme voie a courte portee.
	cxT, cyT := a.grille.Centre(cTueur)
	tueur.dirSortante.Ajoute(k.VictimX-cxT, k.VictimY-cyT)
	tueur.compteRang(poids, fort, k.RangTueur != nil, true)

	victime := a.cellule(cVictime)
	victime.mortsDedans++
	victime.matchsKills[k.MatchID] = struct{}{}
	cxV, cyV := a.grille.Centre(cVictime)
	victime.dirEntrante.Ajoute(k.KillerX-cxV, k.KillerY-cyV)
	victime.compteRang(poids, fort, k.RangTueur != nil, false)
}

// compteRang range une elimination du cote « kills partis d'ici » (depuis vrai) ou du cote
// « morts subies ici ». Le poids et la force sont ceux du TUEUR dans les deux cas : c'est
// le meme evenement, vu des deux bouts.
func (c *celluleAcc) compteRang(poids float64, fort, rangConnu, depuis bool) {
	if depuis {
		c.killsPonderes += poids
		if rangConnu {
			c.killsRangConnu++
		}
		if fort {
			c.killsTueurFort++
		}
		return
	}
	c.mortsPonderees += poids
	if rangConnu {
		c.mortsRangConnu++
	}
	if fort {
		c.mortsTueurFort++
	}
}

// AjoutePresence compte un segment d'occupation du cote de l'issue du match. Une duree
// nulle ou negative, ou une position non finie, est ECARTEE et comptee.
func (a *Accumulateur) AjoutePresence(p PresenceSample) {
	c, ok := a.grille.Cellule(p.X, p.Y)
	if !ok || !(p.DurMS > 0) || math.IsInf(p.DurMS, 0) {
		a.presencesIgnorees++
		return
	}
	a.matchs[p.MatchID] = struct{}{}
	cell := a.cellule(c)
	cell.matchsPresence[p.MatchID] = struct{}{}
	if p.Gagnant {
		cell.occupGagnants += p.DurMS
		return
	}
	cell.occupPerdants += p.DurMS
}

// cellule rend (en la creant au besoin) l'accumulateur d'une cellule.
func (a *Accumulateur) cellule(c tactical.Cellule) *celluleAcc {
	if existante := a.cellules[c]; existante != nil {
		return existante
	}
	neuve := &celluleAcc{
		matchsKills:    make(map[string]struct{}),
		matchsPresence: make(map[string]struct{}),
	}
	a.cellules[c] = neuve
	return neuve
}

// NbMatchs rend le nombre de matchs distincts ayant alimente au moins un echantillon.
func (a *Accumulateur) NbMatchs() int { return len(a.matchs) }

// NbCellules rend le nombre de cellules alimentees, avant tout plancher.
func (a *Accumulateur) NbCellules() int { return len(a.cellules) }

// KillsIgnores rend le nombre d'eliminations ecartees faute de deux extremites finies.
func (a *Accumulateur) KillsIgnores() int { return a.killsIgnores }

// PresencesIgnorees rend le nombre de segments d'occupation ecartes.
func (a *Accumulateur) PresencesIgnorees() int { return a.presencesIgnorees }

// Cellules rend les cellules alimentees, TRIEES par colonne puis ligne — un parcours de
// map est aleatoire, et deux lectures du meme corpus doivent rendre le meme CSV.
func (a *Accumulateur) Cellules() []Cellule {
	out := make([]Cellule, 0, len(a.cellules))
	for adr, acc := range a.cellules {
		out = append(out, a.projette(adr, acc))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Col != out[j].Col {
			return out[i].Col < out[j].Col
		}
		return out[i].Lig < out[j].Lig
	})
	return out
}

// projette fige l'etat mutable d'une cellule en valeur publiee.
func (a *Accumulateur) projette(adr tactical.Cellule, acc *celluleAcc) Cellule {
	cx, cy := a.grille.Centre(adr)
	return Cellule{
		Col:                  adr.Col,
		Lig:                  adr.Lig,
		CentreX:              cx,
		CentreY:              cy,
		KillsDepuis:          acc.killsDepuis,
		MortsDedans:          acc.mortsDedans,
		PorteeMedianeM:       Mediane(acc.portees),
		DeniveleMedianM:      Mediane(acc.denivelees),
		MatchsKills:          len(acc.matchsKills),
		MatchsPresence:       len(acc.matchsPresence),
		MatchsDistincts:      unionMatchs(acc),
		OccupationGagnantsMS: acc.occupGagnants,
		OccupationPerdantsMS: acc.occupPerdants,
		DirSortante:          acc.dirSortante,
		DirEntrante:          acc.dirEntrante,
		KillsPonderes:        acc.killsPonderes,
		MortsPonderees:       acc.mortsPonderees,
		KillsRangConnu:       acc.killsRangConnu,
		MortsRangConnu:       acc.mortsRangConnu,
		KillsTueurFort:       acc.killsTueurFort,
		MortsTueurFort:       acc.mortsTueurFort,
	}
}

// unionMatchs compte les matchs distincts des deux sources reunies.
func unionMatchs(acc *celluleAcc) int {
	union := make(map[string]struct{}, len(acc.matchsKills)+len(acc.matchsPresence))
	for m := range acc.matchsKills {
		union[m] = struct{}{}
	}
	for m := range acc.matchsPresence {
		union[m] = struct{}{}
	}
	return len(union)
}

// distance3D rend la distance entre les deux extremites d'une elimination.
func distance3D(k KillSample) float64 {
	dx := k.KillerX - k.VictimX
	dy := k.KillerY - k.VictimY
	dz := k.KillerZ - k.VictimZ
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

// finis dit si toutes les valeurs sont des nombres finis.
func finis(vs ...float64) bool {
	for _, v := range vs {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}
	return true
}
