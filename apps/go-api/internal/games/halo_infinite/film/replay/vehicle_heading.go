package replay

// vehicle_heading.go — LE CAP D UN VEHICULE : ce que le film ECRIT, et ce qu on DEDUIT faute
// de mieux.
//
// Extrait de `vehicle_tracks.go` le 2026-09-21 (lot 5.4.3), quand la seconde source est apparue
// et que le fichier a franchi le seuil de 500 lignes. La coupe suit une frontiere reelle :
// l ASSEMBLAGE des vies reste la-bas, le CAP et sa provenance viennent ici.

import (
	"log/slog"
	"math"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// vehicleMinSpeedMPS est la vitesse au-dela de laquelle la direction de la velocite `i1` vaut un
// CAP. C est le seuil de l oracle V1a.3 (rapport `V1A_RAPPORT_2026-08-31.md` § 3.1), sous lequel
// la mesure a valide `i1` : ecart median au deplacement de 1,7 a 2,1 deg sur quatre films
// (R = 0,992 a 0,997), contre 51 a 88 deg pour le temoin par melange deterministe.
//
// SOUS CE SEUIL, LA VELOCITE NE REND AUCUN CAP : la direction d un vecteur quasi nul est du
// bruit. Le cap du dernier echantillon mobile est alors REPORTE.
//
// CE SEUIL NE COMMANDE PLUS QUE LE REPLI (lot 5.4.3, 2026-09-21). La phrase qui tenait ici —
// « la seule honnete : `i2` est REFUTE » — datait du lot 5.2b.2, qui avait mesure le vecteur
// HAUT d `i2` en croyant mesurer son avant. L avant EST dans le film : c est la perpendiculaire
// reconstruite du couple (haut, angle de roulis), et sur le mode dont elle est prouvee elle est
// publiee a TOUT regime, seuil ou pas. Cf. [vehicleHeadingOf].
const vehicleMinSpeedMPS = 5.0

// vehicleHeadingOf rend le CAP en degres [0,360[ d un echantillon. Meme origine et meme sens que
// `atan2(Y, X)` des positions dequantifiees, donc la MEME convention que `Point.H` — le client n
// a qu une regle d orientation a connaitre.
//
// DEUX SOURCES, DANS CET ORDRE (lot 5.4.3) :
//
//  1. L AVANT ECRIT DANS LE FILM, reconstruit d `i2` (direction du HAUT + angle de roulis). Il
//     est juste EN TOUT REGIME — marche arriere, derapage, vol, arret — parce qu il ne deduit
//     rien du mouvement. Mesure a l appui : sur `4f77afc1`, 3,1 % des echantillons rapides ont
//     le nez a l OPPOSE de leur vitesse (mediane 154 deg), et le cap tenait donc faux sur eux ;
//     a l arret, le cap du film bouge de 0,29 deg d un echantillon au suivant (il est POSE, la
//     ou la velocite ne dit plus rien).
//  2. LA VELOCITE `i1`, en REPLI, au-dessus du seuil de l oracle.
//
// Le repli n est pas decoratif : le cap du film n est publie que sur le mode dont la
// reconstruction est PROUVEE ([grammar.FwdUpModeConfig]), et le chemin « delta » n ecrit aucun
// angle absolu. Sous les deux, `vehicleSamplesOf` reporte le dernier cap connu, comme avant.
func vehicleHeadingOf(p grammar.BipedPosition) (float32, bool) {
	if h, ok := vehicleFilmHeadingOf(p); ok {
		return h, true
	}
	return vehicleVelocityHeadingOf(p)
}

// vehicleFilmHeadingOf rend le cap LU DANS LE FILM, et seulement sur le chemin prouve.
func vehicleFilmHeadingOf(p grammar.BipedPosition) (float32, bool) {
	if !p.HasRoll || p.FwdMode != grammar.FwdUpModeConfig {
		return 0, false
	}
	f, ok := p.ChassisForwardVector()
	if !ok {
		return 0, false
	}
	return headingDegOf(float64(f[0]), float64(f[1])), true
}

// vehicleVelocityHeadingOf rend le cap DEDUIT du deplacement — le comportement d avant 5.4.3,
// conserve tel quel comme repli.
func vehicleVelocityHeadingOf(p grammar.BipedPosition) (float32, bool) {
	v, ok := p.VelocityVector()
	if !ok {
		return 0, false
	}
	if math.Hypot(float64(v[0]), float64(v[1])) < vehicleMinSpeedMPS {
		return 0, false
	}
	return headingDegOf(float64(v[0]), float64(v[1])), true
}

// logVehicleHeadingSource compte D OU sort le cap des echantillons du nuage. Il n ajoute AUCUN
// champ au document (le schema ne bouge pas) : c est un denominateur de journal, celui qui dira
// si la part du cap LU recule sur un build ou une carte donnes.
func logVehicleHeadingSource(pos []grammar.BipedPosition) {
	var film, velocite, aucun, modeNonPublie int
	for _, p := range pos {
		switch {
		case vehicleHeadingHasFilm(p):
			film++
		case vehicleHeadingHasVelocity(p):
			velocite++
			if p.HasRoll {
				modeNonPublie++
			}
		default:
			aucun++
		}
	}
	slog.Info("rejeu : source du cap des vehicules",
		"echantillons", len(pos), "capDuFilm", film, "capParVelocite", velocite,
		"sansCap", aucun, "roulisLuMaisModeNonPublie", modeNonPublie)
}

func vehicleHeadingHasFilm(p grammar.BipedPosition) bool {
	_, ok := vehicleFilmHeadingOf(p)
	return ok
}

func vehicleHeadingHasVelocity(p grammar.BipedPosition) bool {
	_, ok := vehicleVelocityHeadingOf(p)
	return ok
}

// headingDegOf ramene un vecteur du plan du sol a un cap en degres [0,360[.
func headingDegOf(x, y float64) float32 {
	deg := math.Atan2(y, x) * 180 / math.Pi
	if deg < 0 {
		deg += 360
	}
	return float32(deg)
}
