package replay

import "levelup/go-api/internal/games/halo_infinite/film/internal/grammar"

// emprise_jouee.go — L EMPRISE JOUEE D UN FILM, ECRITE UNE FOIS (lot M1 des retours du rejeu,
// 2026-09-23).
//
// # TROIS USAGES, UNE SEULE REGLE
//
// La garde « p1..p99 plus douze etendues centrales, par axe » est nee au cadrage (`boundsOf`,
// 2026-09-08) : elle y ecartait l echantillon aberrant des BORNES, et le laissait publie. Le lot M1
// l applique aussi a ce qui est PUBLIE — points de trace, echantillons et naissances de vehicule
// (repli `repli_position_hors_emprise_ecartee`, cf. positions_porte.go). Trois usages de la meme
// regle : elle vit ici, et nulle part ailleurs. Le garde-rail
// `TestUneSeuleEcritureDeLEmpriseJouee` interdit toute autre construction de garde d axe, et donc
// toute recopie de la constante `boundsRejectSpreads` ou des centiles — une recopie divergerait au
// premier recalibrage (regle n° 6 du depot).
//
// # CE QU ELLE EST, ET CE QU ELLE N EST PAS
//
// Une HEURISTIQUE MESUREE, pas une grammaire : sur le parc (64 artefacts, 2026-09-08), l ecart des
// artefacts de decodage va de 17,7 a 105,3 etendues centrales, celui du jeu legitime ne depasse
// pas 9,5 (cf. geometry.go). Une carte a zone jouee tres etendue et peu echantillonnee pourrait lui
// faire ecarter une position reelle : c est pourquoi tout ce qu elle ecarte est COMPTE, jamais tu.
//
// # LA CONTINUITE : UNE CHUTE REELLE N EST PAS UN ARTEFACT
//
// Hors de l emprise ne suffit pas. Le corpus temoin (`50247b26`, 2026-09-23) montre une Wasp, un
// Ghost et un Warthog qui TOMBENT dans un vide, z de -59 a -77 m, un echantillon par image de
// 100 ms : la fin de leur chute sort de la garde, et elle est vraie. Un faux en-tete, lui, est
// ISOLE : il surgit a des centaines de metres de la position precedente (le Mongoose de
// 81c02726 : 193 m apres 70 s de silence ; la Wraith de 0a44c6cc : trois echantillons identiques
// a 350 m). La regle ecarte donc une position hors de l emprise seulement quand AUCUNE chaine
// d instants voisins (au plus `continuiteMaxEcartUS` d ecart, a au plus `continuiteVitesseMaxMPS`)
// ne la relie a une position DANS l emprise du meme slot — dans un sens ou dans l autre (une chute
// hors de la carte, un vehicule largue d en haut).

// empriseJouee est la garde des trois axes. `armee` est faux quand l echantillon est trop maigre
// pour que des centiles aient un sens (`boundsMinSamples`) : elle n ecarte alors rien.
type empriseJouee struct {
	axes  [3]axisGuard
	armee bool
}

// empriseDesAxes construit la garde depuis les trois axes TRIES (la forme que rend `axisValues`).
// C EST LE SEUL APPELANT DE `guardOf` (garde-rail).
func empriseDesAxes(xs, ys, zs []float32) empriseJouee {
	if len(xs) < boundsMinSamples {
		return empriseJouee{}
	}
	return empriseJouee{axes: [3]axisGuard{guardOf(xs), guardOf(ys), guardOf(zs)}, armee: true}
}

// rejette dit si une position tombe hors de l emprise. Une garde desarmee ne rejette rien.
func (e empriseJouee) rejette(x, y, z float32) bool {
	if !e.armee {
		return false
	}
	return e.axes[0].rejects(x) || e.axes[1].rejects(y) || e.axes[2].rejects(z)
}

// continuiteMaxEcartUS : l ecart maximal, en microsecondes, entre deux positions d un meme slot pour
// qu elles se relient (cinq images de 100 ms). Au-dela, le film s est tu, et un silence ne relie
// rien (cf. le repli F-2 des vehicules).
const continuiteMaxEcartUS = 500_000

// continuiteVitesseMaxMPS : la vitesse, en m/s, au-dela de laquelle deux positions voisines ne se
// relient pas. MESURE : les vehicules atteignent 26,1 m/s (V1a.3), une chute du corpus temoin
// ~24 m/s ; un faux en-tete saute de 190 a 1 100 m. 60 m/s laisse plus du double de marge aux
// mouvements reels et reste d un ordre de grandeur sous les sauts d artefact.
const continuiteVitesseMaxMPS = 60.0

// continuitePlancherS : la duree plancher de la borne de distance — deux positions du MEME instant
// (ou d instants tres proches) se relient a moins de 6 m, jamais a une distance nulle imposee.
const continuitePlancherS = 0.1

// seRelient dit si deux positions (ou une naissance et une position) d un meme slot sont physiquement
// voisines : ecart d au plus `continuiteMaxEcartUS`, distance compatible avec
// `continuiteVitesseMaxMPS`.
func seRelient(a, b [3]float32, ta, tb uint64) bool {
	dt := int64(tb) - int64(ta)
	if dt < 0 {
		dt = -dt
	}
	if dt > continuiteMaxEcartUS {
		return false
	}
	return dist3(a, b) <= continuiteVitesseMaxMPS*max(float64(dt)/1e6, continuitePlancherS)
}

// rejetsIsoles rend, pour les positions d UN slot TRIEES par instant, celles que le repli F-1
// ecarte : hors de l emprise ET sans chaine de continuite vers une position dans l emprise. Une
// position sans coordonnee monde n est jamais ecartee, et ne relie rien.
func (e empriseJouee) rejetsIsoles(pos []grammar.BipedPosition) []bool {
	n := len(pos)
	garde := make([]bool, n)
	for i, p := range pos {
		garde[i] = !p.HasWorld || !e.rejette(p.X, p.Y, p.Z)
	}
	relie := func(i, j int) bool {
		a, b := pos[i], pos[j]
		return a.HasWorld && b.HasWorld && seRelient([3]float32{a.X, a.Y, a.Z}, [3]float32{b.X, b.Y, b.Z},
			a.TimestampUS, b.TimestampUS)
	}
	for i := 1; i < n; i++ {
		if !garde[i] && garde[i-1] && pos[i-1].HasWorld && relie(i-1, i) {
			garde[i] = true
		}
	}
	for i := n - 2; i >= 0; i-- {
		if !garde[i] && garde[i+1] && pos[i+1].HasWorld && relie(i, i+1) {
			garde[i] = true
		}
	}
	rejets := make([]bool, n)
	for i := range garde {
		rejets[i] = !garde[i]
	}
	return rejets
}
