package replay

// placement_des_vies.go — OÙ ÉTAIT CHAQUE VIE PAR RAPPORT À SON ÉQUIPE, ET CE QU'ELLE A RAPPORTÉ.
//
// # LA QUESTION (plan `.ai/PLAN_EMPRISE_VIES_2026-09-28.md`, décision V3)
//
// Le graphe « Placement et rendement de chaque vie » pose un point par vie : sa distance médiane
// au coéquipier le plus proche, et ses frags. Ce fichier mesure une vie, sans rien lire : on lui
// passe ce que la passe de positions du collecteur a déjà en main (registre d'identité, positions
// monde), les camps de la base, les morts du journal, les intervalles de port d'objectif et la
// portée du radar de la variante. Il rend une ligne par vie nommée — la même vie que la ligne de
// `match_lives`, même registre, même horloge du MATCH.
//
// # LA MESURE, INSTANT PAR INSTANT
//
// Une grille de [PasDeLaGrilleDesViesMs] sur `[début, fin]` de la vie, bornes incluses. Chaque
// instant reçoit UNE cause, examinée DANS CET ORDRE — l'ordre est une décision, pas un détail :
//
//	porteur              le joueur porte l'objectif : il n'est ni groupé ni isolé, il fait le mode ;
//	équipe à terre       aucun coéquipier vivant : il n'y a personne de qui être proche ;
//	non situé            le joueur n'a pas de position de moins d'une seconde (véhicule, lecture
//	                     manquée — V0.1 du plan : 96 % du temps à bord n'est pas situé) ;
//	coéquipier non situé un coéquipier VIVANT n'est pas situé : la distance au plus proche ne se
//	                     calcule pas sans lui, le plus proche est peut-être celui-là ;
//	MESURÉ               la distance au coéquipier vivant le plus proche.
//
// « Vivant » et « situé » sont LES prédicats du contexte des morts (`death_context.go`),
// `vivantA` et `visibleA`, appelés tels quels : un coéquipier mort depuis 500 ms a encore une
// position fraîche, et c'est la vitalité qui l'écarte. La distance est [distanceHorizontale], la
// même que celle du contexte de mort.
//
// SEULS LES INSTANTS MESURÉS entrent dans la médiane et le temps hors radar ; chaque cause
// d'exclusion est cumulée en millisecondes sur la ligne, pour que le lecteur dise ce que la
// mesure n'a pas vu au lieu de le taire.
//
// # LES FRAGS D'UNE VIE
//
// Un frag publiable, entre deux camps DIFFÉRENTS, appartient à la vie du tueur qui couvre
// `[début de la vie, début de sa vie suivante)`. La borne de droite n'est PAS la fin de la vie :
// une grenade ou un échange tue après la mort du tireur (1,5 % des frags du 22/09), et ce frag
// est celui de la vie qui vient de finir. Un frag antérieur à la première vie du tueur n'est
// rattaché à aucune : il est compté dans le bilan, que l'appelant journalise.
//
// # CE QUE CE FICHIER REFUSE
//
// Un registre dont le pont n'est pas publiable (`PontPubliable`) : les positions s'attribuent par
// les vies, et une identité lue de deux façons rend le nommage lui-même faux. Même refus, même
// raison que [ContextesDesMorts] — une ligne écrite en base n'a pas d'écran pour montrer sa
// réserve.

import (
	"math"
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// PasDeLaGrilleDesViesMs est le pas de la grille sur laquelle une vie est mesurée : chaque
// instant de la grille vaut ce nombre de millisecondes dans les cumuls de la ligne.
//
// 100 MS, LE PAS DU DOCUMENT DE REJEU ([DefaultFrameIntervalMS]) : assez fin pour qu'un échange
// de quelques secondes compte plusieurs instants, assez gros pour qu'une vie de deux minutes
// reste à 1 200 instants — et la fenêtre de visibilité (une seconde) en couvre dix.
const PasDeLaGrilleDesViesMs = 100

// MesureMinimaleDUneVieMs est le temps MESURÉ sous lequel une vie n'a pas de médiane publiée.
//
// DEUX SECONDES, DÉCISION V3 DU PLAN : une médiane de moins de vingt instants dirait une
// distance que la vie n'a pas eu le temps d'avoir. La vie reste publiée — durée, causes, frags —
// avec une médiane NULL : « vie non mesurée », ni tracée ni classée par le lecteur.
const MesureMinimaleDUneVieMs = 2000

// IntervalleDePort est un portage d'objectif, sur l'horloge du MATCH, bornes INCLUSES.
type IntervalleDePort struct {
	DebutMS, FinMS int64
}

// FragDuJournal est une mort du journal telle que la base la porte : son tueur, sa victime, son
// instant (horloge du MATCH) et la publiabilité de sa passe. Un xuid nul veut dire « non résolu »
// (bot, nom inconnu du roster).
type FragDuJournal struct {
	TueurXUID, VictimeXUID uint64
	TempsMS                int64
	Publiable              bool
}

// EntreePlacement porte ce que la passe de positions du collecteur a déjà lu, plus ce que la base
// et le mode savent.
type EntreePlacement struct {
	// Positions et Registre : le matériau de la passe de positions (positions monde des bipèdes,
	// registre d'identité qui porte les vies nommées et le calage d'horloge).
	Positions []grammar.BipedPosition
	Registre  IdentityRegistry
	// Equipes : xuid -> camp, depuis la base (`match_participants`). Le film ne porte aucun camp.
	Equipes map[uint64]int
	// Journal : toutes les morts du match, telles que le journal les a écrites.
	Journal []FragDuJournal
	// Portages : les intervalles de port d'objectif par xuid. Vide hors mode à porteur.
	Portages map[uint64][]IntervalleDePort
	// RadarM : la portée du radar de la variante, en mètres. nil = variante absente de la table
	// des portées : aucune part hors radar n'est calculée.
	RadarM *float64
}

// InstantMesure est un instant mesuré de la vie et sa distance au coéquipier vivant le plus
// proche (mètres, deux décimales).
type InstantMesure struct {
	TempsMS   int64
	DistanceM float64
}

// PlacementVie est la mesure d'une vie nommée.
type PlacementVie struct {
	XUID           uint64
	DebutMS, FinMS int64
	DureeMS        int64
	// MesureMS : le temps mesuré (instants dont la cause est MESURÉ), en ms.
	MesureMS int64
	// MedianeM : médiane de la distance au coéquipier vivant le plus proche sur les instants
	// mesurés, en mètres à deux décimales. nil sous [MesureMinimaleDUneVieMs] mesurées.
	MedianeM *float64
	// HorsRadarMS : temps mesuré où cette distance DÉPASSE la portée du radar. nil sans portée.
	HorsRadarMS *int64
	// RadarM : la portée employée, recopiée sur la ligne (nil sans portée).
	RadarM *float64
	// Les quatre causes d'exclusion, cumulées en ms.
	PorteurMS, EquipeATerreMS, NonSitueMS, CoequipierNonSitueMS int64
	// Frags : les frags rattachés à la vie (cf. l'en-tête).
	Frags int
	// DerniereMesure : le dernier instant mesuré de la vie, nil si aucun. C'est l'instant que le
	// contexte de mort mesure lui aussi pour une vie qui finit par une mort : les deux producteurs
	// se confrontent dessus (témoin du lot V1).
	DerniereMesure *InstantMesure
}

// BilanPlacement dit ce que la passe a écarté, pour que l'appelant le compte et le journalise.
type BilanPlacement struct {
	// PontNonPubliable : le registre refuse son pont — aucune ligne n'est rendue.
	PontNonPubliable bool
	// FragsRattaches : frags attribués à une vie.
	FragsRattaches int
	// FragsAvantLaPremiereVie : frags d'un tueur antérieurs à sa première vie nommée.
	FragsAvantLaPremiereVie int
	// FragsSansVie : frags d'un tueur qui n'a aucune vie nommée.
	FragsSansVie int
	// FragsNonPubliables : morts d'une passe non publiable.
	FragsNonPubliables int
	// FragsTueurInconnu : morts dont le tueur n'est pas résolu.
	FragsTueurInconnu int
	// FragsCampInconnu : le camp du tueur ou de la victime est absent de la base — ni frag
	// prouvé ni trahison prouvée, écarté.
	FragsCampInconnu int
	// Trahisons : tueur et victime du même camp (suicides compris).
	Trahisons int
}

// PlacementDesVies mesure chaque vie nommée du registre, dans l'ordre de `ViesNommees` (celui
// des lignes de `match_lives`).
func PlacementDesVies(e EntreePlacement) ([]PlacementVie, BilanPlacement) {
	if !e.Registre.PontPubliable() {
		return nil, BilanPlacement{PontNonPubliable: true}
	}
	vies := e.Registre.ViesNommees()
	if len(vies) == 0 {
		return nil, BilanPlacement{}
	}
	m := mesureDesVies{
		pos:         indexerParXUID(e.Positions, e.Registre),
		vivantes:    viesParXUID(e.Registre),
		equipes:     e.Equipes,
		coequipiers: coequipiersParCamp(e.Equipes),
		portages:    e.Portages,
		radar:       e.RadarM,
	}
	out := make([]PlacementVie, 0, len(vies))
	for _, v := range vies {
		out = append(out, m.mesurer(v))
	}
	return out, rattacherLesFrags(out, e.Journal, e.Equipes)
}

// causeDInstant : la cause attribuée à un instant de la grille.
type causeDInstant int

const (
	instantMesure causeDInstant = iota
	instantPorteur
	instantEquipeATerre
	instantNonSitue
	instantCoequipierNonSitue
)

// mesureDesVies porte les index que toutes les vies d'un match partagent.
type mesureDesVies struct {
	pos         positionsParXUID
	vivantes    map[uint64][]vieMatch
	equipes     map[uint64]int
	coequipiers map[int][]uint64
	portages    map[uint64][]IntervalleDePort
	radar       *float64
}

// mesurer parcourt la grille d'une vie et cumule ses causes.
func (m mesureDesVies) mesurer(v VieNommee) PlacementVie {
	p := PlacementVie{XUID: v.XUID, DebutMS: v.DebutMS, FinMS: v.FinMS, DureeMS: v.FinMS - v.DebutMS}
	var distances []float64
	var horsRadar int64
	var derniere InstantMesure
	for t := v.DebutMS; t <= v.FinMS; t += PasDeLaGrilleDesViesMs {
		cause, d := m.classer(v.XUID, t)
		switch cause {
		case instantPorteur:
			p.PorteurMS += PasDeLaGrilleDesViesMs
		case instantEquipeATerre:
			p.EquipeATerreMS += PasDeLaGrilleDesViesMs
		case instantNonSitue:
			p.NonSitueMS += PasDeLaGrilleDesViesMs
		case instantCoequipierNonSitue:
			p.CoequipierNonSitueMS += PasDeLaGrilleDesViesMs
		default:
			p.MesureMS += PasDeLaGrilleDesViesMs
			distances = append(distances, d)
			if m.radar != nil && d > *m.radar {
				horsRadar += PasDeLaGrilleDesViesMs
			}
			derniere = InstantMesure{TempsMS: t, DistanceM: arrondiMetres(d)}
		}
	}
	p.MedianeM = medianeDUneVie(distances, p.MesureMS)
	if m.radar != nil {
		r := *m.radar
		p.RadarM, p.HorsRadarMS = &r, &horsRadar
	}
	if p.MesureMS > 0 {
		p.DerniereMesure = &derniere
	}
	return p
}

// classer rend la cause d'un instant, dans l'ordre de l'en-tête, et la distance quand il est
// mesuré.
func (m mesureDesVies) classer(xuid uint64, t int64) (causeDInstant, float64) {
	if porteA(m.portages[xuid], t) {
		return instantPorteur, 0
	}
	vivants := m.coequipiersVivants(xuid, t)
	if len(vivants) == 0 {
		return instantEquipeATerre, 0
	}
	lieu, situe := m.pos.visibleA(xuid, t)
	if !situe {
		return instantNonSitue, 0
	}
	plusProche := math.Inf(1)
	for _, autre := range vivants {
		p, ok := m.pos.visibleA(autre, t)
		if !ok {
			return instantCoequipierNonSitue, 0
		}
		plusProche = math.Min(plusProche, distanceHorizontale(lieu, p))
	}
	return instantMesure, plusProche
}

// coequipiersVivants rend les coéquipiers du joueur qu'une vie nommée couvre à l'instant. Un
// joueur sans camp en base n'a aucun coéquipier.
func (m mesureDesVies) coequipiersVivants(xuid uint64, t int64) []uint64 {
	camp, connu := m.equipes[xuid]
	if !connu {
		return nil
	}
	var out []uint64
	for _, autre := range m.coequipiers[camp] {
		if autre != xuid && m.pos.vivantA(m.vivantes, autre, t) {
			out = append(out, autre)
		}
	}
	return out
}

// porteA dit si un intervalle de port (bornes incluses) couvre l'instant.
func porteA(ports []IntervalleDePort, t int64) bool {
	for _, iv := range ports {
		if t >= iv.DebutMS && t <= iv.FinMS {
			return true
		}
	}
	return false
}

// coequipiersParCamp regroupe les xuids par camp, triés pour un parcours déterministe.
func coequipiersParCamp(equipes map[uint64]int) map[int][]uint64 {
	out := map[int][]uint64{}
	for x, camp := range equipes {
		out[camp] = append(out[camp], x)
	}
	for _, l := range out {
		sort.Slice(l, func(i, j int) bool { return l[i] < l[j] })
	}
	return out
}

// medianeDUneVie rend la médiane des distances mesurées, arrondie comme toute distance publiée,
// ou nil sous le temps mesuré minimal.
func medianeDUneVie(d []float64, mesureMS int64) *float64 {
	if mesureMS < MesureMinimaleDUneVieMs || len(d) == 0 {
		return nil
	}
	sort.Float64s(d)
	n := len(d)
	med := d[n/2]
	if n%2 == 0 {
		med = (d[n/2-1] + d[n/2]) / 2
	}
	r := arrondiMetres(med)
	return &r
}

// rattacherLesFrags pose chaque frag publiable sur la vie de son tueur (cf. l'en-tête) et compte
// ceux qu'il écarte. `vies` est dans l'ordre de `ViesNommees` : par xuid, les débuts croissent.
func rattacherLesFrags(vies []PlacementVie, journal []FragDuJournal, equipes map[uint64]int) BilanPlacement {
	parTueur := map[uint64][]int{}
	for i := range vies {
		parTueur[vies[i].XUID] = append(parTueur[vies[i].XUID], i)
	}
	var b BilanPlacement
	for _, f := range journal {
		if !fragRecevable(f, equipes, &b) {
			continue
		}
		idx := parTueur[f.TueurXUID]
		if len(idx) == 0 {
			b.FragsSansVie++
			continue
		}
		// Première vie dont le début DÉPASSE l'instant : le frag appartient à celle d'avant.
		k := sort.Search(len(idx), func(i int) bool { return vies[idx[i]].DebutMS > f.TempsMS })
		if k == 0 {
			b.FragsAvantLaPremiereVie++
			continue
		}
		vies[idx[k-1]].Frags++
		b.FragsRattaches++
	}
	return b
}

// fragRecevable applique les trois refus d'un frag — passe non publiable, tueur inconnu, camps
// inconnus ou identiques — et compte celui qui s'applique.
func fragRecevable(f FragDuJournal, equipes map[uint64]int, b *BilanPlacement) bool {
	if !f.Publiable {
		b.FragsNonPubliables++
		return false
	}
	if f.TueurXUID == 0 {
		b.FragsTueurInconnu++
		return false
	}
	campT, okT := equipes[f.TueurXUID]
	campV, okV := equipes[f.VictimeXUID]
	if !okT || !okV {
		b.FragsCampInconnu++
		return false
	}
	if campT == campV {
		b.Trahisons++
		return false
	}
	return true
}
