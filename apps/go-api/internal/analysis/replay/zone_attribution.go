package replay

// ATTRIBUTION DE ZONE — « quel joueur est DANS quelle zone a l'instant d'une prise ».
//
// Le chantier d'origine repondait « Quelle zone : NON ». Les trois pieces existaient
// pourtant, chacune de son cote et aucune ne repondant seule :
//
//	les FORMES     — 63 zones de Bastion versionnees, avec dimensions et orientation
//	                 (objectives_catalog.go, mapvar/shape.go) ;
//	les INSTANTS   — les actions d'objectif, datees a la milliseconde et attribuees a un
//	                 xuid (objectives.go) ;
//	les POSITIONS  — les trajectoires du rejeu, sur la MEME horloge que les actions
//	                 (build.go, ecart mesure < 4 ms sur 573 s de film).
//
// Ce fichier est le croisement, et rien d'autre : une fonction pure, sans I/O.
//
// # Ce qu'il attribue, et ce qu'il n'attribue PAS
//
// Il rend l'IDENTITE de la zone (InstanceID du jeu, et le rang spatial stable qui va
// avec). Il ne rend PAS la lettre A/B/C affichee en jeu : elle n'est ni dans la variante
// de carte ni dans le film, et la deduire de l'ordre du fichier serait exactement la
// fausse decouverte contre laquelle mapvar/objectives.go met en garde. Etablir la lettre
// demande un temoin externe (releve Theater) ; le champ existera alors A COTE de
// SpatialRank, sans l'ecraser.
//
// # Pourquoi la couverture est publiee avec le resultat
//
// Une attribution sans son denominateur ne se lit pas : « 200 actions posees dans une
// zone » ne veut rien dire tant qu'on ne sait pas si le total etait 210 ou 2 100. Chaque
// action non attribuee tombe donc dans un compteur NOMME, et les compteurs somment
// exactement au total.

import (
	"math"
	"sort"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

// DefaultMaxGapFrames est la tolerance par defaut entre l'instant d'une action et
// l'echantillon de position utilise, en NOMBRE DE FRAMES de la grille du rejeu.
//
// Deux a 100 ms font 200 ms. Le choix vient de la donnee, pas du confort : le film
// replique une entite a ~60 Hz mais PAS a intervalle garanti, et le reechantillonnage
// laisse des trous quand l'entite n'a rien emis. Au-dela de deux frames, on interpolerait
// un joueur en mouvement sur plusieurs metres — la largeur d'une zone. En deca, on
// perdrait des actions pour un trou d'echantillonnage qui ne dit rien de la position.
const DefaultMaxGapFrames = 2

// ZoneAttribution est le verdict rendu pour UNE action d'objectif.
type ZoneAttribution struct {
	Action ObjectiveAction
	// Attributed dit si l'action a ete posee dans une zone. Quand il est faux, le motif
	// se lit dans ZoneCoverage — jamais dans les champs ci-dessous, qui restent a zero.
	Attributed bool
	// SpatialRank et InstanceID identifient la zone retenue. InstanceID est l'identite du
	// JEU ; SpatialRank est NOTRE rang stable (cf. Zone.SpatialRank), pas la lettre.
	SpatialRank int
	InstanceID  int32
	// SampleGapFrames est l'ecart, en frames, entre l'instant de l'action et
	// l'echantillon de position utilise. Zero = echantillon exactement sur la frame.
	// Le publier permet de mesurer si le taux d'attribution tient a la tolerance.
	SampleGapFrames int
	// Sample est la position effectivement employee, et HasSample dit si elle existe.
	//
	// POURQUOI LA PUBLIER. Un taux d'attribution faible a deux causes opposees : le
	// joueur n'etait pas dans la zone (fait de jeu, ou semantique de la statistique), ou
	// bien les deux reperes ne coincident pas (bug). La distance du joueur a la zone la
	// plus proche les separe en une mesure — et sans la position, l'appelant ne peut pas
	// la calculer.
	Sample    Point
	HasSample bool
	// DistanceM est la distance de Sample AU VOLUME de la zone la plus proche, en metres.
	// Zero quand le joueur est dedans. Renseignee des qu'une position existe, meme si
	// l'action n'est pas attribuee — c'est ce chiffre qui distingue « il etait a cote »
	// de « il etait a l'autre bout de la carte ».
	DistanceM float64
}

// ZoneCoverage compte ce qui a ete attribue et, surtout, ce qui ne l'a pas ete.
//
// INVARIANT : Attributed + NoPosition + Outside + Ambiguous == Actions.
type ZoneCoverage struct {
	Actions    int
	Attributed int
	// NoPosition : aucun echantillon de position de ce joueur a moins de MaxGapFrames de
	// l'action. Ce n'est PAS un echec du croisement mais un trou du rejeu (vie non
	// nommee, joueur sans track publiee, trou d'echantillonnage).
	NoPosition int
	// Outside : la position est connue, et la zone la plus proche est au-dela de
	// MaxDistanceM (donc, au seuil par defaut de zero : le joueur n'est dans aucune
	// zone). C'est le seul compteur qui parle vraiment du croisement — c'est lui qu'un
	// temoin negatif doit faire exploser.
	Outside int
	// Ambiguous : PLUSIEURS zones sont a la meme distance minimale — au seuil strict,
	// des zones qui se recouvrent. Trancher par « la premiere trouvee » donnerait un
	// resultat dependant de l'ordre de tri ; on refuse.
	Ambiguous int
}

// AttributeOptions regle le croisement.
type AttributeOptions struct {
	// MaxGapFrames : tolerance entre l'instant de l'action et l'echantillon de position.
	// <= 0 prend DefaultMaxGapFrames.
	MaxGapFrames int
	// MaxDistanceM : distance maximale AU VOLUME de la zone, en metres.
	//
	// ZERO = appartenance STRICTE (le joueur est dans la forme), et c'est le defaut. Une
	// valeur positive relache le test : le joueur peut etre a cote.
	//
	// A QUOI CA SERT, ET CE QUE CA COUTE. Mesure sur le corpus : a l'instant d'une action
	// de zone, 100 % des joueurs sont a moins de 20 m d'une zone, mediane 6,6 m, mais
	// seuls ~10 % sont DEDANS. Trois causes se superposent, aucune n'est un bug du
	// croisement : la statistique du statborg est repliquee a son propre rythme (donc
	// APRES l'action), la recompense d'equipe n'exige pas d'etre dans la zone, et la
	// forme du fichier n'est pas forcement le volume de capture du jeu.
	//
	// Relacher le test rend donc l'attribution exploitable, mais AFFAIBLIT la
	// proposition : on ne dit plus « il etait dedans », on dit « c'est de cette zone-la
	// qu'il s'agit ». Un seuil ajuste pour maximiser le taux serait une tautologie —
	// l'appelant doit publier la COURBE taux(seuil) avec son temoin, jamais un seuil seul.
	MaxDistanceM float64
}

func (o AttributeOptions) maxGapFrames() int {
	if o.MaxGapFrames > 0 {
		return o.MaxGapFrames
	}
	return DefaultMaxGapFrames
}

// AttributeZones croise les actions, les trajectoires et les zones.
//
// Les actions sont rendues dans leur ordre d'entree, une entree par action, y compris
// pour celles qui ne sont pas attribuees : l'appelant qui veut la liste complete des
// actions la garde sans re-joindre.
func AttributeZones(actions []ObjectiveAction, tracks []Track, zones []Zone,
	opt AttributeOptions) ([]ZoneAttribution, ZoneCoverage) {
	maxGap := opt.maxGapFrames()
	samples := samplesByXUID(tracks)
	out := make([]ZoneAttribution, 0, len(actions))
	cov := ZoneCoverage{Actions: len(actions)}
	for _, a := range actions {
		att := ZoneAttribution{Action: a}
		p, gap, ok := nearestSample(samples[a.XUID], a.T, maxGap)
		if !ok {
			cov.NoPosition++
			out = append(out, att)
			continue
		}
		att.SampleGapFrames, att.Sample, att.HasSample = gap, p, true
		best, hits := nearestZones(zones, p)
		if len(hits) > 0 {
			// Sans zone, `best` vaut l'infini : le publier ferait entrer +Inf dans les
			// quantiles de l'appelant. Une carte sans zone n'a pas de distance a une zone.
			att.DistanceM = best
		}
		switch {
		case len(hits) == 0 || best > opt.MaxDistanceM:
			cov.Outside++
		case len(hits) == 1:
			att.Attributed = true
			att.SpatialRank = hits[0].SpatialRank
			att.InstanceID = hits[0].InstanceID
			cov.Attributed++
		default:
			cov.Ambiguous++
		}
		out = append(out, att)
	}
	return out, cov
}

// nearestZones rend la plus petite distance au volume et TOUTES les zones qui l'atteignent.
//
// Rendre la liste des ex aequo plutot que la premiere trouvee est ce qui rend l'ambiguite
// visible au lieu de la resoudre en secret : deux zones qui se recouvrent donnent toutes
// les deux une distance nulle, et trancher par l'ordre de tri rendrait le resultat
// instable a la prochaine regeneration du catalogue.
func nearestZones(zones []Zone, p Point) (float64, []Zone) {
	w := mapvar.Vec3{X: float64(p.X), Y: float64(p.Y), Z: float64(p.Z)}
	best := math.Inf(1)
	// Une seule allocation : la liste des ex aequo est REMISE A ZERO quand une zone plus
	// proche apparait, jamais reallouee. Elle est appelee une fois par action.
	hits := make([]Zone, 0, len(zones))
	for _, z := range zones {
		switch d := z.Volume.DistanceTo(w); {
		case d < best:
			best, hits = d, append(hits[:0], z)
		case d == best:
			hits = append(hits, z)
		}
	}
	return best, hits
}

// samplesByXUID regroupe les points de TOUTES les vies d'un joueur sur un seul axe de
// temps, trie par frame.
//
// POURQUOI FUSIONNER LES VIES : une Track est une VIE, pas un joueur (document.go). Un
// joueur qui meurt et reapparait a plusieurs tracks ; chercher sa position dans une seule
// d'entre elles raterait toutes les actions des autres vies.
//
// Les vies sans xuid sont ECARTEES : le film ne nomme pas toutes les vies, et rattacher
// une position anonyme a un joueur serait exactement l'erreur que le pont d'identite
// existe pour eviter.
func samplesByXUID(tracks []Track) map[string][]Point {
	out := map[string][]Point{}
	for _, tr := range tracks {
		if tr.XUID == "" {
			continue
		}
		out[tr.XUID] = append(out[tr.XUID], tr.Points...)
	}
	for x := range out {
		pts := out[x]
		sort.SliceStable(pts, func(i, j int) bool { return pts[i].T < pts[j].T })
		out[x] = pts
	}
	return out
}

// nearestSample rend l'echantillon le plus proche de la frame demandee, et l'ecart en
// frames. ok est faux si le plus proche est au-dela de maxGap.
//
// A egalite d'ecart, l'echantillon ANTERIEUR l'emporte : une prise de zone est la
// consequence d'une presence, la position d'avant la decrit mieux que celle d'apres.
func nearestSample(pts []Point, frame, maxGap int) (Point, int, bool) {
	if len(pts) == 0 {
		return Point{}, 0, false
	}
	i := sort.Search(len(pts), func(k int) bool { return pts[k].T >= frame })
	best, bestGap := Point{}, maxGap+1
	if i > 0 {
		best, bestGap = pts[i-1], frame-pts[i-1].T
	}
	if i < len(pts) {
		if g := pts[i].T - frame; g < bestGap {
			best, bestGap = pts[i], g
		}
	}
	if bestGap > maxGap {
		return Point{}, 0, false
	}
	return best, bestGap, true
}

// TranslateZones rend une copie du jeu de zones deplacee du meme vecteur — le TEMOIN
// NEGATIF du croisement.
//
// Sans lui, « le joueur est dans la zone » n'est pas une mesure : des zones assez
// grandes attrapent tout le monde, et le chantier a deja du retirer un « critere en or »
// tautologique pour cette raison exacte. Le rang spatial est CONSERVE tel quel : on
// compare le meme jeu de zones a lui-meme, deplace.
func TranslateZones(zones []Zone, d mapvar.Vec3) []Zone {
	out := make([]Zone, len(zones))
	for i, z := range zones {
		z.Volume = z.Volume.Translate(d)
		z.Center = mapvar.Vec3{X: z.Center.X + d.X, Y: z.Center.Y + d.Y, Z: z.Center.Z + d.Z}
		out[i] = z
	}
	return out
}
