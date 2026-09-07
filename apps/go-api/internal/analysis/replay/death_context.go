package replay

// death_context.go — QUI ENTOURAIT LA VICTIME, A L'INSTANT DE SA MORT.
//
// # LA QUESTION, ET POURQUOI ELLE SE REPOND ICI
//
// « Où je meurs isolé » demande si un coéquipier était assez près pour peser sur l'échange. La
// réponse tient à quatre choses que personne ne détient ensemble : les MORTS (le journal de la
// base), les POSITIONS (le film), les ÉQUIPES et les DÉPARTS (la base). Ce fichier ne fait que
// TRANCHER — il ne connaît ni base, ni fichier, ni carte : on lui passe les quatre, il rend un
// état par coéquipier.
//
// Il vit dans `analysis/replay` parce que c'est le paquet qui tient déjà les positions et le
// pont slot->xuid. Le collecteur, lui, se contente de lui passer ce qu'il a déjà lu.
//
// # LES QUATRE ÉTATS, ET LEUR ORDRE
//
// L'ordre d'examen est une DÉCISION, pas un détail d'implémentation :
//
//	1. PARTI          la base fait foi. Un joueur dont le départ est enregistré n'est plus dans
//	                  la partie, quoi que le film montre encore de lui.
//	2. VISIBLE        une position répliquée dans la dernière seconde. C'est le SEUL état qui
//	                  autorise une distance : les autres n'ont pas de position crédible.
//	3. EN ATTENTE     sa dernière mort au journal précède l'instant, et rien ne l'a montré
//	                  depuis. Il attend sa réapparition, il ne peut pas accompagner.
//	4. HORS DE VUE    tout le reste. IL EST VIVANT — typiquement en véhicule, où le biped cesse
//	                  d'être répliqué. Le compter mort ferait sortir « équipe à terre » une mort
//	                  survenue à trois mètres d'un coéquipier en Warthog : c'est exactement le
//	                  défaut P0 de la revue ronde 1 (2026-09-07).
//
// Mettre « en attente » avant « visible » inverserait le poids de la preuve : une position
// répliquée est une OBSERVATION, une attente est une déduction, et l'observation gagne.

import (
	"math"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

// FenetreVisibiliteMs borne l'âge d'une position pour qu'elle compte comme « il était là ».
//
// UNE SECONDE, ET C'EST UN CHOIX BORNÉ PAR LA MESURE : le pas de réplication est d'environ
// 16 ms, donc une seconde représente une soixantaine d'échantillons manqués — bien au-delà d'un
// trou de transmission, bien en deçà des 5 s qui découpent une vie (`lifeGapUS`). Un joueur au
// sprint (~5 m/s en Halo) parcourt 5 m dans cette fenêtre : l'incertitude reste inférieure au
// rayon de radar le plus court (18 m en Arène).
const FenetreVisibiliteMs = 1000

// Les quatre états d'un coéquipier à l'instant d'une mort.
const (
	EtatVisible   = "visible"
	EtatEnAttente = "waiting"
	EtatHorsDeVue = "out_of_sight"
	EtatParti     = "left"
)

// MortDuJournal est une mort telle que la BASE la porte : une victime et un instant, sur
// l'horloge du match. C'est la clé de `match_kill_events`.
type MortDuJournal struct {
	VictimeXUID uint64
	TempsMS     int64
}

// EntreeContexteMorts porte les quatre sources, déjà lues par l'appelant.
type EntreeContexteMorts struct {
	// Positions : les positions bipeds du film, avec leurs coordonnées MONDE (l'appelant a
	// fourni les bornes de la carte à `ScanBipedPositions`). Sans monde, aucune distance.
	Positions []filmdec.BipedPosition
	// SlotXUID : le pont slot -> xuid, tel que `ResolveSlotXUID` le rend.
	SlotXUID map[uint32]uint64
	// DecalageMS convertit l'horloge du FILM en horloge du MATCH
	// (`horlogeFilm = horlogeMatch + DecalageMS`), soit `OwnerReport.DeathOffsetMS`.
	DecalageMS int64
	// Journal : toutes les morts du match, y compris celles des coéquipiers — elles servent à
	// savoir qui attendait sa réapparition.
	Journal []MortDuJournal
	// Equipes : xuid -> numéro d'équipe, depuis `match_participants`. JAMAIS depuis le film,
	// qui ne porte aucun camp.
	Equipes map[uint64]int
	// DepartMS : xuid -> instant du départ, en ms depuis le début du match. Une entrée ABSENTE
	// veut dire « jamais parti » — et c'est le cas normal.
	DepartMS map[uint64]int64
}

// ContexteMort est ce que la lecture saura d'une mort : combien de coéquipiers dans chaque état,
// et à quelle distance était le plus proche de ceux qu'on voyait.
type ContexteMort struct {
	VictimeXUID uint64
	TempsMS     int64
	// PlusProcheM : distance 2D HORIZONTALE au coéquipier VISIBLE le plus proche, en mètres
	// monde, arrondie à 2 décimales. nil = aucun coéquipier visible, ou victime sans position
	// — une absence de mesure, jamais une distance infinie ni un zéro.
	PlusProcheM                            *float64
	Visibles, EnAttente, HorsDeVue, Partis int
	Total                                  int
}

// ContextesDesMorts rend un contexte par mort du journal dont la VICTIME A UNE POSITION connue.
//
// UNE MORT SANS LIEU NE SORT PAS. Elle a bien eu lieu, mais le film ne montre pas où : les
// distances à ses coéquipiers ne se calculent pas, et écrire une ligne à `nearest_teammate_m`
// NULL la ferait lire « aucun coéquipier à portée », c'est-à-dire ISOLÉE. Une absence de mesure
// deviendrait un verdict.
func ContextesDesMorts(e EntreeContexteMorts) []ContexteMort {
	pos := indexerParXUID(e)
	mortsPar := mortsParVictime(e.Journal)

	out := make([]ContexteMort, 0, len(e.Journal))
	for _, m := range e.Journal {
		lieu, connu := pos.visibleA(m.VictimeXUID, m.TempsMS)
		if !connu {
			continue
		}
		son, dansUneEquipe := e.Equipes[m.VictimeXUID]
		if !dansUneEquipe {
			continue
		}
		out = append(out, contexteDUneMort(e, pos, mortsPar, m, son, lieu))
	}
	return out
}

// contexteDUneMort classe chaque coéquipier de la victime et retient le plus proche des visibles.
func contexteDUneMort(e EntreeContexteMorts, pos positionsParXUID,
	mortsPar map[uint64][]int64, m MortDuJournal, son int, lieu point,
) ContexteMort {
	c := ContexteMort{VictimeXUID: m.VictimeXUID, TempsMS: m.TempsMS}
	plusProche := math.Inf(1)
	for autre, equipe := range e.Equipes {
		if autre == m.VictimeXUID || equipe != son {
			continue
		}
		c.Total++
		switch etatDUnCoequipier(e, pos, mortsPar, autre, m.TempsMS) {
		case EtatParti:
			c.Partis++
		case EtatVisible:
			c.Visibles++
			if p, ok := pos.visibleA(autre, m.TempsMS); ok {
				if d := math.Hypot(p.x-lieu.x, p.y-lieu.y); d < plusProche {
					plusProche = d
				}
			}
		case EtatEnAttente:
			c.EnAttente++
		default:
			c.HorsDeVue++
		}
	}
	if !math.IsInf(plusProche, 1) {
		arrondi := math.Round(plusProche*100) / 100
		c.PlusProcheM = &arrondi
	}
	return c
}

// etatDUnCoequipier applique les quatre règles, dans l'ordre documenté en tête de fichier.
func etatDUnCoequipier(e EntreeContexteMorts, pos positionsParXUID,
	mortsPar map[uint64][]int64, xuid uint64, tMS int64,
) string {
	if depart, parti := e.DepartMS[xuid]; parti && depart <= tMS {
		return EtatParti
	}
	if _, vu := pos.visibleA(xuid, tMS); vu {
		return EtatVisible
	}
	if derniere, mort := derniereMortAvant(mortsPar[xuid], tMS); mort && !pos.vuEntre(xuid, derniere, tMS) {
		return EtatEnAttente
	}
	return EtatHorsDeVue
}

// derniereMortAvant rend l'instant de la dernière mort STRICTEMENT antérieure. Les instants sont
// triés par `mortsParVictime`.
func derniereMortAvant(morts []int64, tMS int64) (int64, bool) {
	derniere, trouve := int64(0), false
	for _, d := range morts {
		if d >= tMS {
			break
		}
		derniere, trouve = d, true
	}
	return derniere, trouve
}

// mortsParVictime indexe le journal par victime, instants triés.
func mortsParVictime(journal []MortDuJournal) map[uint64][]int64 {
	out := make(map[uint64][]int64, len(journal))
	for _, m := range journal {
		out[m.VictimeXUID] = append(out[m.VictimeXUID], m.TempsMS)
	}
	for _, v := range out {
		sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })
	}
	return out
}

// point est une position monde, en mètres, sur l'horloge du match.
type point struct {
	tMS  int64
	x, y float64
}

// positionsParXUID répond à « où était ce joueur, et l'a-t-on vu récemment ».
type positionsParXUID map[uint64][]point

// indexerParXUID projette les positions du film sur les xuids, en convertissant l'horloge.
//
// SEULES LES POSITIONS MONDE ENTRENT : sans les bornes de la carte, un quantum n'est pas une
// coordonnée, et une distance calculée dessus n'aurait pas d'unité.
func indexerParXUID(e EntreeContexteMorts) positionsParXUID {
	out := positionsParXUID{}
	for i := range e.Positions {
		p := &e.Positions[i]
		if !p.HasWorld {
			continue
		}
		xuid, connu := e.SlotXUID[p.Slot]
		if !connu || xuid == 0 {
			continue
		}
		out[xuid] = append(out[xuid], point{
			tMS: int64(p.TimestampUS)/1000 - e.DecalageMS,
			x:   float64(p.X), y: float64(p.Y),
		})
	}
	for _, v := range out {
		sort.Slice(v, func(i, j int) bool { return v[i].tMS < v[j].tMS })
	}
	return out
}

// visibleA rend la position la plus RÉCENTE dans la fenêtre de visibilité, et si elle existe.
func (p positionsParXUID) visibleA(xuid uint64, tMS int64) (point, bool) {
	best, trouve := point{}, false
	for _, pt := range p[xuid] {
		if pt.tMS > tMS {
			break
		}
		if tMS-pt.tMS <= FenetreVisibiliteMs {
			best, trouve = pt, true
		}
	}
	return best, trouve
}

// vuEntre dit si le joueur a été observé APRÈS `depuis` et jusqu'à `jusqua` — c'est-à-dire s'il a
// réapparu depuis sa mort.
func (p positionsParXUID) vuEntre(xuid uint64, depuis, jusqua int64) bool {
	for _, pt := range p[xuid] {
		if pt.tMS > jusqua {
			break
		}
		if pt.tMS > depuis {
			return true
		}
	}
	return false
}
