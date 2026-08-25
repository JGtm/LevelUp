package main

// identite.go — QUELLE CARTE A ETE JOUEE DANS CE FILM.
//
// L'artefact de rejeu ne nomme NI sa carte NI son mode (cf. `replay.Document`, champ
// MapObjectives) : en production c'est le registre des matchs qui la nomme. Un instrument de
// MESURE n'a pas a ouvrir une base DuckDB pour ca — la regle est reprise telle quelle de
// `internal/mapdecoupe/oracle_corpus_test.go` (2026-08-16), avec ses constantes et leurs
// justifications chiffrees, parce qu'elle y est deja etalonnee sur le meme corpus.
//
// LA REGLE, en une phrase : la carte d'un film est celle dont les PAVES DU DESIGNER (les zones
// nommees du tag `levl`, catalogue `map_callouts.json`) contiennent la plus grande part de ses
// positions, ALTITUDE COMPRISE, et dont le cadre publie les contient aussi.
//
// POURQUOI L'ALTITUDE EST INDISPENSABLE, mesure du 2026-08-16 : sans elle un match de FORGE
// (canevas pose vers z = +61) se faisait attribuer `btb_exiled` a 87 % — les paves, vus du
// dessus, se recouvrent d'une carte a l'autre.

import (
	"fmt"
	"math"

	"levelup/go-api/internal/analysis/replay"
)

// Reglages de la reconnaissance — VALEURS DE `internal/mapdecoupe`, ne pas les re-regler ici.
const (
	// pasEchantillonID : une position sur N suffit a reconnaitre une carte.
	pasEchantillonID = 50
	// margeZID : de combien un joueur peut sortir de la tranche verticale de sa zone (saut,
	// chute, plateforme mince). Serre, parce que c'est LUI qui separe deux cartes dont les
	// paves se recouvrent vus du dessus.
	margeZID = 2.0
	// scoreMinID / ecartMinID : le meilleur candidat doit contenir 80 % des positions ET
	// devancer le suivant de 15 points. Sinon le film est ECARTE — aucune carte ne lui est
	// attribuee, plutot qu'une carte plausible.
	scoreMinID = 0.80
	ecartMinID = 0.15
	// ecartNiveauMax : ecart tolere, en metres, entre l'altitude MEDIANE d'un film et le sol
	// joue publie par le fond de la carte.
	ecartNiveauMax = 20.0
)

// prisme est un pave du designer : un contour au sol, la tranche verticale qu'il habite, et
// le nom joueur de la zone (celui du tag `uslg`, resolu hors ligne au catalogue).
type prisme struct {
	nom    string
	poly   [][2]float64
	zBas   float64
	zHaut  float64
	grande bool
}

// carte rassemble, pour un module installe, tout ce que l'instrument lit a son sujet : le
// calage et le sol joue de son fond publie, ses paves de designer, ses ancres d'objectifs et
// le chemin de son `.module`.
type carte struct {
	module    string
	noms      []string
	chemin    string
	ancres    [][3]float64
	calage    replay.MapBackgroundCalibration
	niveauJeu float64
	prismes   []prisme
	grandes   int
}

// reconnait rend le module de la carte jouee, ou une explication de l'ECART quand aucune
// carte ne l'emporte assez nettement. Une explication vaut mieux qu'une attribution douteuse :
// un film mal rattache polluerait la hauteur frequentee d'une carte qu'il n'a jamais visitee.
func reconnait(cartes []carte, pts [][3]float64) (module, ecart string) {
	zmed := medianeAltitude(pts)
	meilleur, second, nom, nom2 := 0.0, 0.0, "", ""
	for i := range cartes {
		if math.Abs(zmed-cartes[i].niveauJeu) > ecartNiveauMax {
			continue
		}
		s := cartes[i].score(pts)
		if s > meilleur {
			meilleur, second, nom, nom2 = s, meilleur, cartes[i].module, nom
			continue
		}
		if s > second {
			second, nom2 = s, cartes[i].module
		}
	}
	if meilleur >= scoreMinID && meilleur-second >= ecartMinID {
		return nom, ""
	}
	return "", fmt.Sprintf("z=%.1f ; %s %.0f%% contre %s %.0f%%",
		zmed, ouAucune(nom), 100*meilleur, ouAucune(nom2), 100*second)
}

func ouAucune(s string) string {
	if s == "" {
		return "(aucune)"
	}
	return s
}

// score rend la part des positions echantillonnees qui tombent DANS LE CADRE publie de la
// carte ET dans un de ses prismes, minoree par la part des GRANDES zones visitees.
//
// Le minimum des deux couvertures est ce qui rend le score decisif : « la carte contient-elle
// le match » seule ne separe rien (une grande carte contient une petite arene tout entiere et
// marque 100 %) ; « le match a-t-il visite la carte » la disqualifie.
func (c *carte) score(pts [][3]float64) float64 {
	vues := make([]bool, len(c.prismes))
	n, dedans := 0, 0
	for k := 0; k < len(pts); k += pasEchantillonID {
		n++
		p := pts[k]
		if _, _, ok := c.calage.MondeVersPixel(p[0], p[1]); !ok {
			continue
		}
		if c.touche(p, vues) {
			dedans++
		}
	}
	if n == 0 {
		return 0
	}
	part := float64(dedans) / float64(n)
	if c.grandes == 0 {
		return part
	}
	visitees := 0
	for i := range c.prismes {
		if c.prismes[i].grande && vues[i] {
			visitees++
		}
	}
	return math.Min(part, float64(visitees)/float64(c.grandes))
}

// touche marque les prismes qui contiennent le point et dit si l'un d'eux l'a contenu.
func (c *carte) touche(p [3]float64, vues []bool) bool {
	dedans := false
	for i := range c.prismes {
		z := &c.prismes[i]
		if p[2] < z.zBas-margeZID || p[2] > z.zHaut+margeZID {
			continue
		}
		if len(z.poly) >= 3 && dansPolygone(p[0], p[1], z.poly) {
			vues[i], dedans = true, true
		}
	}
	return dedans
}

// medianeAltitude rend l'altitude mediane d'un film — robuste aux quelques positions en vol
// (chute, vehicule, grappin) que la moyenne ou les bornes deplaceraient.
func medianeAltitude(pts [][3]float64) float64 {
	h := nouvelHistogramme()
	for k := 0; k < len(pts); k += pasEchantillonID {
		h.ajoute(pts[k][2])
	}
	return h.centile(0.5)
}

// dansPolygone : test de parite par lancer de rayon horizontal.
func dansPolygone(x, y float64, poly [][2]float64) bool {
	dedans := false
	for i, j := 0, len(poly)-1; i < len(poly); j, i = i, i+1 {
		if (poly[i][1] > y) != (poly[j][1] > y) &&
			x < (poly[j][0]-poly[i][0])*(y-poly[i][1])/(poly[j][1]-poly[i][1])+poly[i][0] {
			dedans = !dedans
		}
	}
	return dedans
}
