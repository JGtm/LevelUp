package grammar

// entete.go — L EN-TETE DE LA MARCHE : ses parametres hors flux, resolus AVANT elle, chacun avec sa
// provenance (ADR 0037 IR-7 ; decision DT2-3 du plan de l etape 2).
//
// Un parametre hors flux est une valeur que la marche ne lit pas dans le paquet, mais SOUS laquelle
// elle le lit : la largeur de l identifiant bas des records, le decoupage du bloc MPP, le decoupage
// d i0. La marche est construite depuis l en-tete ([FilmContext.EnTete]) et le distributeur le rend
// a ses canaux ([Distribuer]). L en-tete ne lit aucun bit de trame : un parametre que la grammaire
// ne determine qu en balayant le film (le decoupage d i0 auto-detecte) n y est pas resolu.

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// idLowBitsPresume est la largeur de l identifiant bas d un record sous laquelle la marche lit les
// trames : la valeur de l IMAGE STATIQUE du binaire (categorie 7 de `FUN_1406d3140`,
// `DAT_144706100` = 0x1FFF, `varwidth.go`). Deux ecrivains du jeu la reecrivent au runtime : c est
// une valeur PRESUMEE, pas une constante du format — la marche des morts d objet la calibre de 10 a
// 15 (`object_deaths_calibrate.go`).
const idLowBitsPresume = 13

// EnTete porte les parametres hors flux de la marche d un film.
type EnTete struct {
	// IDLowBits : la largeur de l identifiant bas des records des trames delta.
	IDLowBits lecture.Parametre[int]
	// MPP : le decoupage du bloc MPP de la VERSION DE FORMAT du film, que la phase des images-cles
	// pose (`profile.MPPPourFormat`). Non renseigne pour un format inconnu ou de largeur
	// indeterminee : la phase garde alors le decoupage du contexte.
	MPP lecture.Parametre[profile.MPPWidths]
	// I0 : le decoupage d i0 que le contexte impose — force par l appelant, ou celui du catalogue
	// de la carte (cf. `resolveI0Layout`). Non renseigne sans l un ni l autre : la grammaire le
	// detecte alors sur le film, a la demande ([FilmContext.I0Layout]).
	I0 lecture.Parametre[profile.I0Layout]
}

// EnTete rend l en-tete de la marche de ce film. Contexte nil : l en-tete d un film sans format ni
// carte, la seule largeur presumee.
func (c *FilmContext) EnTete() EnTete {
	h := EnTete{IDLowBits: lecture.Parametre[int]{Valeur: idLowBitsPresume,
		Provenance: lecture.ProvenancePresumee}}
	if w, ok, err := mppDuFormat(c.Film()); err == nil && ok {
		h.MPP = lecture.Parametre[profile.MPPWidths]{Valeur: w, Provenance: lecture.ProvenancePresumee}
	}
	if c != nil && c.impose != nil {
		h.I0 = *c.impose
	}
	return h
}

// mppDuFormat rend le decoupage MPP de la version de format du film : (largeurs, vrai) quand le
// format est connu et sa largeur determinee ; (zero, faux, nil) pour un format connu de largeur
// indeterminee (les formats <= 25) ; l erreur d un format inconnu ou illisible.
func mppDuFormat(film *source.Film) (profile.MPPWidths, bool, error) {
	format, ok := FilmFormatVersion(film)
	if !ok {
		return profile.MPPWidths{}, false, profile.ErreurFormatInconnu(FilmFormatVersionUnknown)
	}
	w, ok := profile.MPPPourFormat(format)
	if !ok {
		return profile.MPPWidths{}, false, profile.ErreurFormatInconnu(format)
	}
	return w, w.Valid(), nil
}
