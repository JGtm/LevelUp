package main

// reglages_acces.go — L'ACCESSEUR DES LEVIERS : lire, pour une carte, ce que son entrée arme.

import (
	"log/slog"
	"path/filepath"

	"github.com/JGtm/halo-infinite-map-images/himap"
)

// levier décrit un levier par carte : comment il se lit dans une entrée et comment il se journalise.
// Tous les leviers qui se lisent par une valeur passent par `armeDe` : une carte sans entrée, ou dont
// l'entrée ne l'arme pas, reçoit la valeur nulle, c'est-à-dire la valeur de production.
type levier[T any] struct {
	// libelle : message journalisé quand le levier s'applique à la carte. Supprimer de la matière ou
	// peindre un aplat ne doit jamais passer inaperçu.
	libelle string
	// attribut : nom de la valeur journalisée avec le message ; vide pour un levier booléen.
	attribut string
	// lit rend la valeur du levier dans l'entrée et dit s'il y est armé.
	lit func(reglageCarte) (T, bool)
}

// levierBool : armé quand le champ est vrai.
func levierBool(libelle string, champ func(reglageCarte) bool) levier[bool] {
	return levier[bool]{libelle: libelle, lit: func(c reglageCarte) (bool, bool) { v := champ(c); return v, v }}
}

// levierMesure : armé quand `arme` accepte la valeur du champ (strictement positive, non nulle ou
// strictement négative selon le levier) ; la valeur journalisée porte le nom `attribut`.
func levierMesure(libelle, attribut string, champ func(reglageCarte) float64, arme func(float64) bool) levier[float64] {
	return levier[float64]{libelle: libelle, attribut: attribut,
		lit: func(c reglageCarte) (float64, bool) { v := champ(c); return v, arme(v) }}
}

func positif(v float64) bool       { return v > 0 }
func nonNul(v float64) bool        { return v != 0 }
func strictNegatif(v float64) bool { return v < 0 }

// entreeDe rend l'entrée de réglage de la carte et dit si elle existe.
func (e *environnement) entreeDe(cle string) (reglageCarte, bool) {
	if e.reglages == nil {
		return reglageCarte{}, false
	}
	c, ok := e.reglages.Cartes[cle]
	return c, ok
}

// armeDe est l'accesseur de tous les leviers par valeur : il rend la valeur du levier pour la carte
// et dit s'il y est armé, et le journalise avec la date de son gate. Sans entrée ou levier non armé,
// il rend la valeur nulle.
func armeDe[T any](e *environnement, cle string, l levier[T]) (T, bool) {
	var nul T
	c, ok := e.entreeDe(cle)
	if !ok {
		return nul, false
	}
	v, arme := l.lit(c)
	if !arme {
		return nul, false
	}
	args := []any{"carte", cle}
	if l.attribut != "" {
		args = append(args, l.attribut, v)
	}
	slog.Info("mapfond: "+l.libelle, append(args, "gateLe", c.GateLe)...)
	return v, true
}

// valeurDe rend la valeur du levier pour la carte, nulle quand il n'y est pas armé.
func valeurDe[T any](e *environnement, cle string, l levier[T]) T {
	v, _ := armeDe(e, cle, l)
	return v
}

// styleDe rend l'habillage de cette carte : le sien s'il est déclaré, sinon celui de la
// cuisson. Le choix retenu est JOURNALISÉ — un fond publié dans un habillage qu'on n'a pas vu
// passer est un fond qu'on ne saura pas expliquer.
func (e *environnement) styleDe(cle string) himap.StyleFond {
	c, ok := e.entreeDe(cle)
	if !ok || c.Style == "" {
		return e.style
	}
	s := himap.StyleFond(c.Style)
	if s != e.style {
		slog.Info("mapfond: habillage propre a la carte", "carte", cle, "style", string(s),
			"gateLe", c.GateLe)
	}
	return s
}

// echelleDe rend l'échelle de cette carte : la sienne si elle est déclarée, sinon celle de la
// cuisson (elle-même à la valeur de production si aucune n'est demandée).
func (e *environnement) echelleDe(cle string) float64 {
	if v, ok := armeDe(e, cle, levierEchelle); ok {
		return v
	}
	return e.echelle
}

// echelleImposeeOuDe rend `imposee` si elle est positive, sinon l'échelle de la carte (`echelleDe`) :
// `-relief-seul` impose celle du fond publié, que ni le défaut, ni un levier, ni `-mpp` ne doivent
// déplacer. Une échelle explicite l'emporte aussi sur la cible de cadre automatique de la brique
// (`EchellePourCadre`).
func (e *environnement) echelleImposeeOuDe(cle string, imposee float64) float64 {
	if imposee > 0 {
		return imposee
	}
	return e.echelleDe(cle)
}

// zonesNommeesDe rend les polygones des callouts d'une carte : le contour principal de chaque
// zone, plus ses PARTIES (provenance « decoupe »).
//
// LES TROUS NE SONT PAS SOUSTRAITS, et c'est délibéré : le masque sert à décider ce qu'on
// GARDE. Un trou rempli garde un peu plus de matière — l'erreur va dans le sens prudent. Les
// soustraire demanderait une règle pair-impair globale et ferait disparaître, au moindre défaut
// de découpe, du terrain que personne n'a jugé.
func (e *environnement) zonesNommeesDe(cle string) [][][2]float64 {
	if e.callouts == nil {
		return nil
	}
	entree, ok := e.callouts.Maps[cle]
	if !ok {
		return nil
	}
	var out [][][2]float64
	for _, z := range entree.Zones {
		if len(z.Polygon) >= 3 {
			out = append(out, z.Polygon)
		}
		for _, p := range z.Parts {
			if len(p) >= 3 {
				out = append(out, p)
			}
		}
	}
	return out
}

// boiteUtileDe rend le rectangle monde declaré pour cette carte, ou zéro s'il n'y en a pas.
func (e *environnement) boiteUtileDe(cle string) [4]float64 {
	var out [4]float64
	c, ok := e.entreeDe(cle)
	if !ok || len(c.BoiteUtile) != 4 {
		return out
	}
	copy(out[:], c.BoiteUtile)
	return out
}

// moduleGeometrieDe rend le chemin ABSOLU du module qui porte la geometrie de cette carte,
// ou la chaine vide quand c'est le sien — le cas de toutes les cartes sauf une.
func (e *environnement) moduleGeometrieDe(cle string) string {
	rel, ok := armeDe(e, cle, levierModuleGeometrie)
	if !ok {
		return ""
	}
	return filepath.Join(e.racineJeu, filepath.FromSlash(rel))
}

// rogneAuNavmeshDe dit si cette carte se rogne a son maillage de navigation.
func (e *environnement) rogneAuNavmeshDe(cle string) bool {
	return valeurDe(e, cle, levierRogneAuNavmesh)
}

// rogneAuNavmeshNatifDe dit si une carte native se rogne a son maillage : oui, sauf si son
// reglage dit `rogneAuNavmesh: false`.
func (e *environnement) rogneAuNavmeshNatifDe(cle string) bool {
	return !valeurDe(e, cle, levierSansRogneAuNavmesh)
}

// rogneAuxAltitudesProchesDe dit si cette carte n affiche que ce qui est proche du niveau de
// jeu. Journalise : c est une voie qui SUPPRIME de la matiere.
func (e *environnement) rogneAuxAltitudesProchesDe(cle string) (bool, float64, float64) {
	c, ok := e.entreeDe(cle)
	if !ok || !c.RogneAuxAltitudesProches {
		return false, 0, 0
	}
	slog.Info("mapfond: rognage a l altitude du niveau de jeu arme pour cette carte", "carte", cle,
		"seuil", c.SeuilAltitude, "marge", c.MargeAltitude, "gateLe", c.GateLe)
	return true, c.SeuilAltitude, c.MargeAltitude
}

// positionsJoueesDe rend les positions réellement jouées de la carte, ou nil si le levier n'est
// pas armé pour elle. Journalisé : ce rognage s'appuie sur un corpus observé, et le nombre de
// positions retenues explique à lui seul l'agressivité du masque.
func (e *environnement) positionsJoueesDe(cle string) ([]himap.PositionJouee, float64) {
	c, ok := e.entreeDe(cle)
	if !ok || !c.RogneAuxPositionsJouees {
		return nil, 0
	}
	pos := e.positions[cle]
	if len(pos) == 0 {
		slog.Warn("mapfond: rognage aux positions demande mais catalogue vide pour cette carte",
			"carte", cle, "gateLe", c.GateLe)
		return nil, 0
	}
	slog.Info("mapfond: rognage aux positions jouees arme pour cette carte", "carte", cle,
		"positions", len(pos), "rayon", c.RayonPositions, "gateLe", c.GateLe)
	return pos, c.RayonPositions
}
