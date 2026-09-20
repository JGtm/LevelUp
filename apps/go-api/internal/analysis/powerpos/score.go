package powerpos

// score.go — LE SCORE D'UNE CELLULE, et la SELECTION des positions de force.
//
// # LA MESURE QUI COMMANDE TOUT (2026-09-20, 12 cartes, .ai/V7.5/positions_de_force/)
//
// Le rapport de duel d'une cellule de 0,5 m — `kills_depuis / engagements` — a la MEME
// dispersion sur les douze cartes mesurees : p10 = 0,29, p25 = 0,38, p50 = 0,50, p75 = 0,62,
// p90 = 0,71. Douze cartes, douze fois la meme courbe, au centieme. Ce n'est pas du terrain,
// c'est du BRUIT BINOMIAL : la mediane du nombre d'engagements par cellule vaut 1 a 4, et une
// piece equilibree tiree six fois rend exactement cet etalement. Un seuil pose sur cette
// grandeur selectionnerait des cellules chanceuses, et il en selectionnerait autant sur une
// carte tiree au hasard.
//
// D'OU LA PREMIERE DECISION : LE SCORE NE SE CALCULE PAS SUR LA CELLULE, MAIS SUR SON
// VOISINAGE. Un disque de `RayonLissageM` autour de la cellule rassemble une cinquantaine de
// cellules, donc quelques dizaines a quelques centaines d'engagements — de quoi que le
// rapport de duel dise quelque chose. La cellule reste l'unite d'ADRESSAGE et de
// publication (D1 du plan) ; elle n'est plus l'unite de COMPTAGE.
//
// # LES QUATRE SIGNAUX RETENUS, ET LEUR POIDS
//
//   - AVANTAGE (0,50) — le rapport de duel du disque, RETRECI vers 0,5. Sans
//     retrecissement, un disque a 12 engagements pese autant qu'un disque a 300. C'est le
//     signal central : une position de force est un endroit d'ou l'on gagne ses duels.
//   - INTENSITE (0,25) — le volume de kills partis du disque, en echelle logarithmique,
//     rapporte au p95 de la carte. Sans lui, un recoin ou trois duels ont bien tourne bat
//     l'angle d'ou partent cent kills. Une position de force est un endroit OU IL SE PASSE
//     QUELQUE CHOSE.
//   - HAUTEUR (0,15) — le denivele median des kills partis du disque. Mesure : le denivele
//     median par cellule vaut 0,0 m et son p90 0,6 a 1,6 m selon la carte ; l'echelle est
//     donc petite, et `DeniveleReferenceM` la normalise sur 1,5 m. C'est la seule mesure
//     empirique de « tenir la hauteur ».
//   - PORTEE (0,10) — la portee mediane des kills partis du disque, normalisee entre le p50
//     et le p90 de la carte (4 a 12 m selon la carte). Une position de force voit loin.
//
// # CE QUI A ETE MESURE PUIS ECARTE
//
// L'OCCUPATION PAR EQUIPE NE PESE PAS DANS LE SCORE (poids zero, et ce n'est pas un oubli).
// Trois raisons, toutes mesurees : (a) six des douze cartes n'ont pas trois artefacts de
// rejeu, donc AUCUNE cellule au plancher — le signal est absent la ou il faudrait trancher ;
// (b) la mediane de l'ecart gagnants-perdants est POSITIVE partout (+0,03 a +0,20) : elle
// mesure d'abord que les vainqueurs vivent plus longtemps, un biais global et non un lieu ;
// (c) a un a six matchs par carte, ce qu'on lit est le cote de depart des equipes de ces
// matchs-la. Les colonnes restent au CSV et l'image de controle reste produite : le jour ou
// le parc d'artefacts se compte en centaines, le signal se reevalue — il ne se devine pas
// aujourd'hui.

import (
	"math"
	"sort"

	"levelup/go-api/internal/analysis/tactical"
)

// Reglage porte tous les seuils du score et de la selection. UNE struct et non huit
// parametres : le seuil du depot est de cinq, et ces valeurs voyagent ensemble.
type Reglage struct {
	// RayonLissageM : rayon du disque de voisinage, en metres.
	RayonLissageM float64
	// PlancherMatchs : matchs distincts (par kill) exiges de la cellule CENTRALE.
	PlancherMatchs int
	// MinEngagementsDisque : engagements exiges dans le disque pour que la cellule soit
	// scorable du tout.
	MinEngagementsDisque int
	// ForcePrior : force du retrecissement du rapport de duel vers 0,5, en engagements
	// virtuels.
	ForcePrior float64
	// DeniveleReferenceM : denivele qui vaut 1,0 sur l'axe hauteur.
	DeniveleReferenceM float64

	PoidsAvantage  float64
	PoidsIntensite float64
	PoidsHauteur   float64
	PoidsPortee    float64
	// PoidsCouverture / PoidsAbri : les deux axes ANGULAIRES (cf. angles.go), ajoutes par
	// la v2. Zero dans le reglage v1, qui ne les connaissait pas.
	PoidsCouverture float64
	PoidsAbri       float64

	// MinDirections : nombre de directions exige dans le disque pour qu'un axe angulaire
	// soit lu. En dessous, l'axe vaut 0,5 (neutre) et ne departage rien : la correction de
	// biais de `SommeAngulaire` traite le petit echantillon, elle ne cree pas de la mesure
	// la ou il n'y en a pas.
	MinDirections int

	// UtiliseRangPondere : lire `KillsPonderes` / `MortsPonderees` (poids du rang du
	// tueur) plutot que les comptes bruts pour l'axe AVANTAGE. L'intensite reste comptee
	// en evenements bruts : c'est un volume, pas une opinion.
	UtiliseRangPondere bool

	// QuantileSeuil : quantile du score, sur les cellules scorables de la carte, au-dessus
	// duquel une cellule est retenue.
	QuantileSeuil float64
	// SeuilScoreMin : plancher ABSOLU du score. Sans lui, le quantile retiendrait toujours
	// 10 % des cellules, y compris sur une carte ou rien ne ressort.
	//
	// IL NE MORD SUR AUCUNE DES DOUZE CARTES MESUREES, ET C'EST SON ROLE : le p90 du score
	// va de 0,662 (fortress) a 0,726 (lattice), et le plancher est pose juste en dessous du
	// plus bas. Il ne se declenche que sur une carte dont le decile superieur serait plus
	// plat que tout ce qui a ete mesure — un filet, pas un reglage.
	SeuilScoreMin float64
	// TailleMiniComposante : nombre minimal de cellules d'une composante retenue.
	TailleMiniComposante int
	// MaxComposantes : nombre maximal de positions publiees par carte.
	MaxComposantes int

	// --- Selection v2 (cf. selection.go) : a zero, la selection est celle de la v1. ---

	// QuantileAmorce : quantile du score au-dessus duquel une cellule est une AMORCE. A
	// zero, la selection retombe sur le seuil unique de la v1.
	QuantileAmorce float64
	// QuantileCroissance : quantile du score au-dessus duquel une cellule agrege une
	// composante deja amorcee (seuil d'hysteresis, plus bas que l'amorce).
	QuantileCroissance float64
	// FermetureRayonCellules : rayon (en cellules) de la fermeture morphologique appliquee
	// aux cellules retenues avant le comptage des composantes. Zero = pas de fermeture.
	FermetureRayonCellules int
	// Connexite8 : compter les composantes en 8-connexite (apres fermeture) plutot qu'en
	// 4-connexite.
	Connexite8 bool
}

// ReglageV1 rend le reglage FIGE le 2026-09-20 (cf. l'en-tete pour la justification de
// chaque valeur, et MESURE_EMPIRIQUE_2026-09-20.md pour les distributions qui les ont
// choisies). Il ne se retouche pas apres le verdict contre l'oracle : une retouche de seuil
// apres coup invalide le verdict.
func ReglageV1() Reglage {
	return Reglage{
		RayonLissageM:        2.0,
		PlancherMatchs:       3,
		MinEngagementsDisque: 40,
		ForcePrior:           40,
		DeniveleReferenceM:   1.5,
		PoidsAvantage:        0.50,
		PoidsIntensite:       0.25,
		PoidsHauteur:         0.15,
		PoidsPortee:          0.10,
		QuantileSeuil:        0.90,
		SeuilScoreMin:        0.65,
		TailleMiniComposante: 12,
		MaxComposantes:       8,
	}
}

// CelluleScoree est une cellule SCORABLE : elle passe le plancher de matchs et son disque
// porte assez d'engagements.
type CelluleScoree struct {
	Cellule

	// KillsDisque / MortsDisque : les comptes du VOISINAGE, pas de la cellule.
	KillsDisque int
	MortsDisque int

	// Les axes, chacun dans [0, 1] apres normalisation (l'avantage est ramene de [0, 1]
	// brut a [0, 1] par une transformation affine autour de 0,5, cf. Score).
	Avantage  float64
	Intensite float64
	Hauteur   float64
	Portee    float64
	// Couverture : dispersion angulaire des directions d'ou l'on TUE depuis le disque.
	// 1 = on tient toutes les voies, 0 = une seule ligne de tir.
	Couverture float64
	// Abri : 1 - dispersion angulaire des directions d'ou l'on MEURT dans le disque.
	// 1 = on ne s'y fait prendre que par un cote, 0 = on y meurt de partout.
	Abri float64

	// DirSortanteDisque / DirEntranteDisque : les sommes angulaires du disque, gardees
	// pour que le rapport de mesure puisse en donner les distributions sans recalculer.
	DirSortanteDisque SommeAngulaire
	DirEntranteDisque SommeAngulaire

	Score float64
}

// Score calcule le score de chaque cellule scorable d'une carte. Rend les cellules TRIEES
// par score decroissant puis par adresse (stabilite du CSV et du catalogue).
//
// Les normalisations (p95 des kills, p50 et p90 de la portee) sont PAR CARTE : une carte
// d'arene et une carte BTB n'ont ni les memes volumes ni les memes distances, et un
// etalonnage global ferait passer toute une carte sous le seuil.
func Score(g tactical.Grille, cellules []Cellule, r Reglage) []CelluleScoree {
	disques := agregeDisques(g, cellules, r)
	if len(disques) == 0 {
		return nil
	}
	ref := etalonne(disques)
	out := make([]CelluleScoree, 0, len(disques))
	for _, d := range disques {
		out = append(out, note(d, ref, r))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		if out[i].Col != out[j].Col {
			return out[i].Col < out[j].Col
		}
		return out[i].Lig < out[j].Lig
	})
	return out
}

// disque est l'agregat du voisinage d'une cellule.
type disque struct {
	centre           Cellule
	kills, morts     int
	killsPond        float64 // kills au poids du rang du tueur
	mortsPond        float64 // morts au poids du rang du tueur
	deniveleP        float64 // denivele moyen PONDERE par les kills
	porteeP          float64 // portee moyenne PONDEREE par les kills
	killsPourMoyenne int
	dirSortante      SommeAngulaire
	dirEntrante      SommeAngulaire
}

// agregeDisques somme les accumulateurs du voisinage de chaque cellule qui passe le
// plancher de matchs. Les cellules du voisinage n'ont PAS a passer ce plancher : elles
// apportent leurs engagements, pas leur droit d'exister.
func agregeDisques(g tactical.Grille, cellules []Cellule, r Reglage) []disque {
	index := make(map[tactical.Cellule]Cellule, len(cellules))
	for _, c := range cellules {
		index[tactical.Cellule{Col: c.Col, Lig: c.Lig}] = c
	}
	offsets := offsetsDisque(g.PasM(), r.RayonLissageM)

	out := make([]disque, 0, len(cellules))
	for _, c := range cellules {
		if c.MatchsKills < r.PlancherMatchs {
			continue
		}
		d := disque{centre: c}
		for _, o := range offsets {
			v, ok := index[tactical.Cellule{Col: c.Col + o.Col, Lig: c.Lig + o.Lig}]
			if !ok {
				continue
			}
			d.kills += v.KillsDepuis
			d.morts += v.MortsDedans
			d.killsPond += v.KillsPonderes
			d.mortsPond += v.MortsPonderees
			d.dirSortante = d.dirSortante.Plus(v.DirSortante)
			d.dirEntrante = d.dirEntrante.Plus(v.DirEntrante)
			if v.KillsDepuis > 0 {
				d.deniveleP += v.DeniveleMedianM * float64(v.KillsDepuis)
				d.porteeP += v.PorteeMedianeM * float64(v.KillsDepuis)
				d.killsPourMoyenne += v.KillsDepuis
			}
		}
		if d.kills+d.morts < r.MinEngagementsDisque {
			continue
		}
		if d.killsPourMoyenne > 0 {
			d.deniveleP /= float64(d.killsPourMoyenne)
			d.porteeP /= float64(d.killsPourMoyenne)
		}
		out = append(out, d)
	}
	return out
}

// offsetsDisque rend les deplacements de cellule contenus dans un disque de rayon `rayonM`.
func offsetsDisque(pasM, rayonM float64) []tactical.Cellule {
	portee := int(math.Ceil(rayonM / pasM))
	var out []tactical.Cellule
	for dc := -portee; dc <= portee; dc++ {
		for dl := -portee; dl <= portee; dl++ {
			if math.Hypot(float64(dc)*pasM, float64(dl)*pasM) <= rayonM {
				out = append(out, tactical.Cellule{Col: dc, Lig: dl})
			}
		}
	}
	return out
}

// etalon porte les reperes de normalisation d'une carte.
type etalon struct {
	killsP95  float64
	porteeP50 float64
	porteeP90 float64
}

// etalonne mesure les reperes sur les disques scorables de la carte.
func etalonne(disques []disque) etalon {
	kills := make([]float64, 0, len(disques))
	portees := make([]float64, 0, len(disques))
	for _, d := range disques {
		kills = append(kills, float64(d.kills))
		if d.killsPourMoyenne > 0 {
			portees = append(portees, d.porteeP)
		}
	}
	return etalon{
		killsP95:  Quantile(kills, 0.95),
		porteeP50: Quantile(portees, 0.50),
		porteeP90: Quantile(portees, 0.90),
	}
}

// note applique la formule a un disque.
func note(d disque, ref etalon, r Reglage) CelluleScoree {
	intensite := 0.0
	if ref.killsP95 > 0 {
		intensite = borne01(math.Log1p(float64(d.kills)) / math.Log1p(ref.killsP95))
	}
	portee := 0.5
	if ref.porteeP90 > ref.porteeP50 {
		portee = borne01(0.5 + (d.porteeP-ref.porteeP50)/(2*(ref.porteeP90-ref.porteeP50)))
	}

	c := CelluleScoree{
		Cellule: d.centre, KillsDisque: d.kills, MortsDisque: d.morts,
		Avantage:  avantageDe(d, r),
		Intensite: intensite,
		Hauteur:   borne01(0.5 + d.deniveleP/(2*r.DeniveleReferenceM)),
		Portee:    portee,
		// Couverture : plus les kills partent dans des directions variees, plus le lieu
		// tient de voies. Abri : l'inverse cote morts — on ne s'y fait prendre que par
		// quelques angles. Les deux retombent a 0,5 sous le minimum de directions.
		Couverture:        axeAngulaire(d.dirSortante, r, false),
		Abri:              axeAngulaire(d.dirEntrante, r, true),
		DirSortanteDisque: d.dirSortante,
		DirEntranteDisque: d.dirEntrante,
	}
	c.Score = r.PoidsAvantage*c.Avantage + r.PoidsIntensite*c.Intensite +
		r.PoidsHauteur*c.Hauteur + r.PoidsPortee*c.Portee +
		r.PoidsCouverture*c.Couverture + r.PoidsAbri*c.Abri
	return c
}

// avantageDe rend le rapport de duel du disque, RETRECI vers 0,5 avec `ForcePrior`
// engagements virtuels puis ramene sur [0, 1] par une transformation affine — le score
// final reste ainsi lisible comme une note.
//
// Le rapport se calcule sur les comptes PONDERES par le rang du tueur quand le reglage le
// demande. La force du retrecissement, elle, reste comptee en engagements BRUTS : c'est la
// taille d'echantillon qui dit combien on peut croire le rapport, et un poids ne cree pas
// d'observation.
func avantageDe(d disque, r Reglage) float64 {
	kills, morts := float64(d.kills), float64(d.morts)
	if r.UtiliseRangPondere {
		kills, morts = d.killsPond, d.mortsPond
	}
	total := kills + morts
	if total <= 0 {
		return 0.5
	}
	// Le retrecissement est proportionne a l'echantillon BRUT : on ramene les comptes
	// ponderes a l'echelle des comptes bruts avant d'y ajouter le prior.
	echelle := float64(d.kills+d.morts) / total
	brut := (kills*echelle + 0.5*r.ForcePrior) / (float64(d.kills+d.morts) + r.ForcePrior)
	return borne01(0.5 + (brut-0.5)*2)
}

// axeAngulaire convertit une somme de directions en axe de score. `inverse` rend l'abri
// (1 - dispersion) plutot que la couverture (dispersion).
//
// Sous `MinDirections`, l'axe vaut 0,5 : neutre, il ne departage rien. Une cellule sans
// mort n'est pas un abri parfait, c'est une cellule sur laquelle on ne sait rien.
func axeAngulaire(s SommeAngulaire, r Reglage, inverse bool) float64 {
	if s.N < r.MinDirections || s.N < 2 {
		return 0.5
	}
	dispersion := s.Dispersion()
	if inverse {
		return borne01(1 - dispersion)
	}
	return borne01(dispersion)
}

// borne01 ramene une valeur dans [0, 1].
func borne01(v float64) float64 {
	switch {
	case math.IsNaN(v):
		return 0
	case v < 0:
		return 0
	case v > 1:
		return 1
	}
	return v
}
