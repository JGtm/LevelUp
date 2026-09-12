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
// Il vit dans `games/halo_infinite/film/replay` parce que c'est le paquet qui tient déjà les positions et le
// pont slot->xuid. Le collecteur, lui, se contente de lui passer ce qu'il a déjà lu.
//
// # LES TROIS ÉTATS APPLIQUÉS EN V1, ET LEUR ORDRE
//
// L'ordre d'examen est une DÉCISION, pas un détail d'implémentation :
//
//	1. VISIBLE        une position répliquée dans la dernière seconde ET une vie nommée qui
//	                  couvre l'instant. LES DEUX SONT NÉCESSAIRES : la réplication s'arrête
//	                  ~34 ms APRÈS la mort, donc un joueur mort depuis 500 ms a encore une
//	                  position fraîche — sans le test de vitalité, il sortait « visible », avec
//	                  une distance, et sa mort se lisait « accompagnée ». C'est le SEUL état qui
//	                  autorise une distance.
//	2. EN ATTENTE     sa dernière mort au journal précède l'instant, et rien ne l'a montré
//	                  depuis. Il attend sa réapparition, il ne peut pas accompagner.
//	3. HORS DE VUE    tout le reste. IL EST VIVANT — typiquement en véhicule, où le biped cesse
//	                  d'être répliqué. Le compter mort ferait sortir « équipe à terre » une mort
//	                  survenue à trois mètres d'un coéquipier en Warthog : c'est exactement le
//	                  défaut P0 de la revue ronde 1 (2026-09-07).
//
// V1 : ABSENT et PARTI ne sont JAMAIS appliqués (horloge API ≠ horloge du film). Un coéquipier
// est présent tout au long du match ; le champ `teammates_left` vaut toujours 0.
//
// Mettre « en attente » avant « visible » inverserait le poids de la preuve : une position
// répliquée est une OBSERVATION, une attente est une déduction, et l'observation gagne.

import (
	"math"
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/filmdec"
)

// DecimalesDeDistance : la précision à laquelle une distance en mètres est publiée.
//
// DEUX DÉCIMALES, LA MÊME CONVENTION QUE LES COORDONNÉES DE L'ARTEFACT (`replay.round2`) et que
// le sidecar d'occupation. Le centimètre est déjà bien en deçà de l'incertitude de la mesure
// (un joueur au sprint parcourt 5 m dans la fenêtre de visibilité) : publier plus de chiffres
// donnerait à la valeur une précision qu'elle n'a pas.
const DecimalesDeDistance = 100

// arrondiMetres applique cette convention.
func arrondiMetres(v float64) float64 {
	return math.Round(v*DecimalesDeDistance) / DecimalesDeDistance
}

// FenetreVisibiliteMs borne l'âge d'une position pour qu'elle compte comme « il était là ».
//
// UNE SECONDE, ET C'EST UN CHOIX BORNÉ PAR LA MESURE : le pas de réplication est d'environ
// 16 ms, donc une seconde représente une soixantaine d'échantillons manqués — bien au-delà d'un
// trou de transmission, bien en deçà des 5 s qui découpent une vie (`lifeGapUS`). Un joueur au
// sprint (~5 m/s en Halo) parcourt 5 m dans cette fenêtre : l'incertitude reste inférieure au
// rayon de radar le plus court (18 m en Arène).
const FenetreVisibiliteMs = 1000

// Les trois états actifs d'un coéquipier à l'instant d'une mort.
//
// Un quatrième état (« parti », déduit d'un calage horloge API/film sur `ArriveeMS`/
// `DepartMS`) a existé puis a été retiré par 7C.9 (2026-09-07) : le calage produisait un
// FAIT FAUX POSSIBLE (cf. .ai/DECOUVERTES_TACTIQUE_2026-09-07.md, section 7C) — un
// coéquipier présent en début de partie pouvait sortir « parti » avant même d'être
// entré, gonflant le compte de morts isolées. `EtatParti` n'était plus jamais produit
// depuis ce retrait ; supprimé en clôture (Q8) avec son `case` et le champ `Partis`.
const (
	EtatVisible   = "visible"
	EtatEnAttente = "waiting"
	EtatHorsDeVue = "out_of_sight"
)

// MortDuJournal est une mort telle que la BASE la porte : une victime et un instant, sur
// l'horloge du match. C'est la clé de `match_kill_events`.
type MortDuJournal struct {
	VictimeXUID uint64
	TempsMS     int64
}

// EntreeContexteMorts porte les trois sources, déjà lues par l'appelant.
type EntreeContexteMorts struct {
	// Positions : les positions bipeds du film, avec leurs coordonnées MONDE (l'appelant a
	// fourni les bornes de la carte à `ScanBipedPositions`). Sans monde, aucune distance.
	Positions []filmdec.BipedPosition
	// Registre est LE REGISTRE D IDENTITE du film. Il porte les vies nommees (slot, bornes,
	// occupant) et le calage d'horloge.
	//
	// PAS `SlotXUID`, ET C'EST LA CORRECTION P0-2 (2026-09-07) : ce pont aplati donne tout
	// l'intervalle d'un slot RECYCLE a son PREMIER porteur nomme (`ownersFromLives`). Le
	// second occupant d'un slot se voyait donc crediter les positions du premier —
	// `teammates_visible` et `nearest_teammate_m` faux, et rien pour le signaler. C'est le
	// bug historique que `nameTracksByLives` a corrige pour les traces le 2026-09-02 ; ici
	// l'attribution se fait par la VIE QUI COUVRE L'INSTANT.
	Registre IdentityRegistry
	// Journal : toutes les morts du match, y compris celles des coéquipiers — elles servent à
	// savoir qui attendait sa réapparition.
	Journal []MortDuJournal
	// Equipes : xuid -> numéro d'équipe, depuis `match_participants`. JAMAIS depuis le film,
	// qui ne porte aucun camp.
	Equipes map[uint64]int
}

// ContexteMort est ce que la lecture saura d'une mort : combien de coéquipiers dans chaque état,
// et à quelle distance était le plus proche de ceux qu'on voyait.
type ContexteMort struct {
	VictimeXUID uint64
	TempsMS     int64
	// PlusProcheM : distance 2D HORIZONTALE au coéquipier VISIBLE le plus proche, en mètres
	// monde, arrondie à 2 décimales. nil = aucun coéquipier visible, ou victime sans position
	// — une absence de mesure, jamais une distance infinie ni un zéro.
	PlusProcheM                    *float64
	Visibles, EnAttente, HorsDeVue int
	Total                          int
}

// LES DEUX CRITÈRES DE `IdentityRegistry.PontPubliable` — le NOMMAGE des vies est-il assez sûr
// pour qu on en tire des faits écrits en
// base.
//
// # `IndexDisagreements` REFUSE, ET C'EST LA MÊME RAISON QUE LE REJEU
//
// Une identité lue de deux façons d'un chunk à l'autre (`coverage.go`, `verdictOfBridge`) rend
// le nommage lui-même faux : ce n'est pas une ambiguïté à arbitrer, c'est le symptôme d'une
// lecture cassée. Le rejeu refuse alors de publier ; écrire quand même des faits d'isolement en
// base serait pire — ils n'ont pas d'écran pour montrer leur réserve, ils seront lus comme des
// mesures.
//
// # `SlotCollisions` NE REFUSE PAS ICI, ET C'EST UNE DIFFÉRENCE ASSUMÉE AVEC LE REJEU
//
// Ce compteur dit qu'un SLOT a porté deux joueurs — autrement dit qu'il a été RECYCLÉ. Il
// invalide le pont APLATI (`SlotXUID`, un xuid par slot), que le document de rejeu publie ; il
// n'invalide pas les VIES, qui portent chacune leur occupant et leurs bornes.
//
// Or c'est précisément par les vies que ce fichier attribue les positions (cf.
// `indexerParXUID`). Refuser sur ce critère écarterait exactement les films que la correction
// P0-2 existe pour traiter : ceux où un slot change de porteur. On perdrait la couverture sans
// gagner la moindre sûreté.
// Elle vit sur le REGISTRE depuis le lot P2 : les deux criteres lisent les tables brutes du
// pont, que seul `identity_registry.go` a le droit de toucher (garde-rail `archlint`).

// ContextesDesMorts rend un contexte par mort du journal dont la VICTIME A UNE POSITION connue.
//
// UNE MORT SANS LIEU NE SORT PAS. Elle a bien eu lieu, mais le film ne montre pas où : les
// distances à ses coéquipiers ne se calculent pas, et écrire une ligne à `nearest_teammate_m`
// NULL la ferait lire « aucun coéquipier à portée », c'est-à-dire ISOLÉE. Une absence de mesure
// deviendrait un verdict.
func ContextesDesMorts(e EntreeContexteMorts) []ContexteMort {
	if !e.Registre.PontPubliable() {
		return nil
	}
	pos := indexerParXUID(e)
	vies := viesParXUID(e.Registre)
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
		out = append(out, contexteDUneMort(e, pos, vies, mortsPar, m, son, lieu))
	}
	return out
}

// contexteDUneMort classe chaque coéquipier de la victime et retient le plus proche des visibles.
func contexteDUneMort(e EntreeContexteMorts, pos positionsParXUID, vies map[uint64][]vieMatch,
	mortsPar map[uint64][]int64, m MortDuJournal, son int, lieu point,
) ContexteMort {
	c := ContexteMort{VictimeXUID: m.VictimeXUID, TempsMS: m.TempsMS}
	plusProche := math.Inf(1)
	for autre, equipe := range e.Equipes {
		if autre == m.VictimeXUID || equipe != son {
			continue
		}
		c.Total++
		switch etatDUnCoequipier(pos, vies, mortsPar, autre, m.TempsMS) {
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
		arrondi := arrondiMetres(plusProche)
		c.PlusProcheM = &arrondi
	}
	return c
}

// etatDUnCoequipier applique les trois règles actives : visible, en attente, hors de vue.
// V1 : les départs ne sont plus appliqués (horloge API vs film) — cf. .ai/DECOUVERTES_TACTIQUE_2026-09-07.md
func etatDUnCoequipier(pos positionsParXUID, vies map[uint64][]vieMatch,
	mortsPar map[uint64][]int64, xuid uint64, tMS int64,
) string {
	if _, vu := pos.visibleA(xuid, tMS); vu && pos.vivantA(vies, xuid, tMS) {
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
//
// # L'ATTRIBUTION SE FAIT PAR LA VIE QUI COUVRE L'INSTANT, JAMAIS PAR LE SLOT
//
// Un slot est RECYCLÉ aux réapparitions : il désigne une vie, pas un joueur. Le pont aplati
// (`SlotXUID`) donne tout son intervalle au premier porteur nommé, si bien que le second
// occupant se voit créditer les positions du premier — c'est le défaut P0-2, et c'est
// exactement le bug que `nameTracksByLives` a corrigé pour les traces le 2026-09-02.
//
// UNE POSITION QUE NULLE VIE NOMMÉE NE COUVRE N'EST ATTRIBUÉE À PERSONNE. Elle est écartée,
// jamais rattachée au voisin le plus proche : mieux vaut un coéquipier « hors de vue » qu'un
// coéquipier placé au mauvais endroit.
func indexerParXUID(e EntreeContexteMorts) positionsParXUID {
	vies := viesParSlot(e.Registre)
	out := positionsParXUID{}
	for i := range e.Positions {
		p := &e.Positions[i]
		if !p.HasWorld {
			continue
		}
		xuid, connu := occupantA(vies[p.Slot], int64(p.TimestampUS))
		if !connu {
			continue
		}
		out[xuid] = append(out[xuid], point{
			tMS: int64(p.TimestampUS)/1000 - e.Registre.DeathOffsetMS(),
			x:   float64(p.X), y: float64(p.Y),
		})
	}
	for _, v := range out {
		sort.Slice(v, func(i, j int) bool { return v[i].tMS < v[j].tMS })
	}
	return out
}

// viesParSlot indexe les vies NOMMÉES par slot. Les vies anonymes n'entrent pas : elles
// n'attribuent rien.
func viesParSlot(r IdentityRegistry) map[uint32][]lifeSpan {
	out := make(map[uint32][]lifeSpan, len(r.Vies()))
	for _, l := range r.Vies() {
		if l.xuid == 0 {
			continue
		}
		out[l.slot] = append(out[l.slot], l)
	}
	return out
}

// occupantA rend l'occupant d'un slot à un instant du FILM (microsecondes), si une vie nommée
// le couvre. Bornes INCLUSIVES : `from` et `to` sont les premier et dernier échantillons de la
// vie, pas un intervalle ouvert.
func occupantA(vies []lifeSpan, tUS int64) (uint64, bool) {
	for _, l := range vies {
		if tUS >= l.from && tUS <= l.to {
			return l.xuid, true
		}
	}
	return 0, false
}

// vivantA dit qu'une vie NOMMÉE du joueur couvre l'instant (horloge du MATCH).
//
// ELLE DOUBLE LA VISIBILITÉ, ET CE N'EST PAS REDONDANT : la réplication d'un joueur s'arrête
// ~34 ms APRÈS sa mort (médiane mesurée, `deathMatchWindowMS`). Une position vieille de 500 ms
// est donc parfaitement possible pour un joueur MORT depuis 500 ms — et sans cette seconde
// condition il sortirait « visible », avec une distance, et sa mort se lirait « accompagnée ».
func (p positionsParXUID) vivantA(vies map[uint64][]vieMatch, xuid uint64, tMS int64) bool {
	for _, v := range vies[xuid] {
		if tMS >= v.debutMS && tMS <= v.finMS {
			return true
		}
	}
	return false
}

// vieMatch est une vie nommée ramenée à l'horloge du MATCH, pour le test de vitalité.
type vieMatch struct{ debutMS, finMS int64 }

// viesParXUID indexe les vies nommées par occupant, sur l'horloge du match.
func viesParXUID(r IdentityRegistry) map[uint64][]vieMatch {
	out := make(map[uint64][]vieMatch, len(r.Vies()))
	for _, l := range r.Vies() {
		if l.xuid == 0 {
			continue
		}
		out[l.xuid] = append(out[l.xuid], vieMatch{
			debutMS: l.from/1000 - r.DeathOffsetMS(),
			finMS:   l.to/1000 - r.DeathOffsetMS(),
		})
	}
	return out
}

// visibleA rend la position la plus RÉCENTE dans la fenêtre de visibilité, et si elle existe.
//
// RECHERCHE BINAIRE, PAS UN BALAYAGE. La tranche est triée par instant, et cette fonction est
// appelée pour CHAQUE coéquipier de CHAQUE mort : un balayage linéaire y coûtait
// morts × coéquipiers × échantillons, soit des centaines de millions d'itérations sur un BTB
// de 15 minutes à 24 joueurs. `sort.Search` ramène le facteur `échantillons` à son logarithme.
func (p positionsParXUID) visibleA(xuid uint64, tMS int64) (point, bool) {
	pts := p[xuid]
	// Premier indice dont l'instant DÉPASSE tMS : le candidat est celui juste avant.
	i := sort.Search(len(pts), func(k int) bool { return pts[k].tMS > tMS })
	if i == 0 {
		return point{}, false
	}
	pt := pts[i-1]
	if tMS-pt.tMS > FenetreVisibiliteMs {
		return point{}, false
	}
	return pt, true
}

// vuEntre dit si le joueur a été observé APRÈS `depuis` et jusqu'à `jusqua` — c'est-à-dire s'il a
// réapparu depuis sa mort.
func (p positionsParXUID) vuEntre(xuid uint64, depuis, jusqua int64) bool {
	pts := p[xuid]
	// Premier indice STRICTEMENT après `depuis` : s'il existe et tombe avant `jusqua`, le
	// joueur a été revu. Même raison qu'au-dessus de préférer la recherche binaire.
	i := sort.Search(len(pts), func(k int) bool { return pts[k].tMS > depuis })
	return i < len(pts) && pts[i].tMS <= jusqua
}
