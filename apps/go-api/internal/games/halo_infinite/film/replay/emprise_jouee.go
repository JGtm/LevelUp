package replay

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
