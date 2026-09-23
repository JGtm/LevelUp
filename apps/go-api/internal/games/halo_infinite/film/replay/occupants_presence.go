package replay

// occupants_presence.go — LA PRESENCE D'UN OCCUPANT, DE SES ENTITES, DE BOT_METADATA ET DE SES
// VIES (lot M2.3, 2026-09-23). Les regles sont ecrites en tete de occupants.go ; ce fichier les
// applique, et porte les fenetres d'entite que la liaison et le nommage des corps partagent.

import (
	"math"
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// instantDeFrame rend l'instant, en microsecondes de film, du debut d'une frame du document.
func instantDeFrame(h replayClock, f int) uint64 {
	if f < 0 {
		f = 0
	}
	return h.origin + uint64(f)*h.step
}

// frameDInstant rend la frame d'un instant du film, bornee a la grille du document.
func frameDInstant(h replayClock, us uint64) int {
	if h.step == 0 {
		return 0
	}
	return clampFrame(frameOf(us, h.origin, h.step), h.frames)
}

// fenetreStricteUS rend la fenetre STRICTE d'une entite : de sa premiere a sa derniere image-cle
// porteuse, bornes incluses.
func fenetreStricteUS(scan grammar.PlayerEntityScan, e grammar.PlayerEntity) (de, a uint64) {
	de, _ = scan.KeyframeUS(e.FirstKF)
	a, _ = scan.KeyframeUS(e.LastKF)
	return de, a
}

// fenetreLarge est la fenetre LARGE d'une entite, bornes EXCLUES : de l'image-cle porteuse qui
// precede sa premiere a celle qui suit sa derniere. OUVERTE au bord du film — l'occupant d'une
// entite lue a la premiere image-cle etait la avant elle, et celui lu a la derniere apres elle.
type fenetreLarge struct {
	de, a                      uint64
	ouverteAvant, ouverteApres bool
}

// contient dit si un instant tombe STRICTEMENT dans la fenetre.
func (f fenetreLarge) contient(t uint64) bool {
	return (f.ouverteAvant || t > f.de) && (f.ouverteApres || t < f.a)
}

// fenetreLargeDe rend la fenetre large d'une entite.
func fenetreLargeDe(scan grammar.PlayerEntityScan, e grammar.PlayerEntity) fenetreLarge {
	f := fenetreLarge{ouverteAvant: true, ouverteApres: true}
	if v, ok := scan.KeyframeUS(e.FirstKF - 1); ok {
		f.de, f.ouverteAvant = v, false
	}
	if v, ok := scan.KeyframeUS(e.LastKF + 1); ok {
		f.a, f.ouverteApres = v, false
	}
	return f
}

// entitesDuBot rend les entites de l'index d'un bot que ses declarations BOT_METADATA designent.
func entitesDuBot(scan grammar.PlayerEntityScan, idx int, decls [][2]uint64) []int {
	var out []int
	for k, e := range scan.Entities {
		if e.Index == idx && !e.Unstable && entiteDeclareeParLeBot(scan, e, decls) {
			out = append(out, k)
		}
	}
	return out
}

// entiteDeclareeParLeBot dit si la fenetre STRICTE d'une entite croise une des declarations
// BOT_METADATA d'un bot (`[de, a)`, `a == 0` = declare jusqu'au bout).
func entiteDeclareeParLeBot(scan grammar.PlayerEntityScan, e grammar.PlayerEntity, decls [][2]uint64) bool {
	de, a := fenetreStricteUS(scan, e)
	for _, d := range decls {
		fin := d[1]
		if fin == 0 {
			fin = math.MaxUint64
		}
		if de < fin && a >= d[0] {
			return true
		}
	}
	return false
}

// declarationsDe rend les declarations BOT_METADATA d'une entree de bot. Le roster nomme ses bots
// par le nom que BOT_METADATA porte (cf. `buildRoster`) : c'est la cle, avec l'index.
func declarationsDe(e RosterEntry, bots []BotIdentity) [][2]uint64 {
	var out [][2]uint64
	for _, b := range bots {
		if b.Name == e.Name && b.FilmIndex == e.FilmIndex {
			out = append(out, b.Declarations...)
		}
	}
	return out
}

// presenceDe rend la presence d'une entree et dit si elle est LUE (entite ou BOT_METADATA) plutot
// que tiree des seules vies.
func presenceDe(e RosterEntry, occ occupantDuRoster, in entreesDesOccupants) ([]intervalleDePresence, bool) {
	ents := make([]intervalleDePresence, 0, len(occ.entites))
	for _, k := range occ.entites {
		ents = append(ents, presenceDeLEntite(in.scan, in.scan.Entities[k], in.horloge))
	}
	var decls []intervalleDePresence
	if e.Bot {
		for _, d := range declarationsDe(e, in.bots) {
			decls = append(decls, presenceDeLaDeclaration(d, in.horloge))
		}
	}
	switch {
	case len(decls) > 0:
		// LE RETRAIT DU BOT EST EXACT : les entites ne font qu'avancer son arrivee, jamais
		// reculer son depart au pas des images-cles.
		return etendreParLesVies(avancerParLesEntites(decls, ents), occ.vies), true
	case len(ents) > 0:
		return etendreParLesVies(fusionnerLesPresences(ents), occ.vies), true
	default:
		return enveloppeDesVies(occ.vies), false
	}
}

// presenceDeLEntite traduit une entite en presence : de la frame 0 si elle est lue au coup
// d'envoi, sinon de sa premiere image-cle ; certaine jusqu'a sa derniere image-cle, affichee
// jusqu'a la veille de la suivante (Q22) — jusqu'au bout si elle est lue a la fin.
func presenceDeLEntite(scan grammar.PlayerEntityScan, e grammar.PlayerEntity, h replayClock) intervalleDePresence {
	derniere := h.frames - 1
	de, a := fenetreStricteUS(scan, e)
	iv := intervalleDePresence{de: frameDInstant(h, de), a: frameDInstant(h, a)}
	if scan.AtStart(e) {
		iv.de = 0
	}
	iv.aMax = iv.a
	if scan.AtEnd(e) {
		iv.a, iv.aMax = derniere, derniere
	} else if suivante, ok := scan.KeyframeUS(e.LastKF + 1); ok {
		iv.aMax = max(iv.a, frameDInstant(h, suivante)-1)
	}
	return iv
}

// presenceDeLaDeclaration traduit une declaration BOT_METADATA en presence EXACTE : du paquet qui
// declare le bot a la veille du paquet qui ne le declare plus.
func presenceDeLaDeclaration(d [2]uint64, h replayClock) intervalleDePresence {
	iv := intervalleDePresence{de: frameDInstant(h, d[0])}
	fin := h.frames - 1
	if d[1] != 0 {
		fin = max(iv.de, frameDInstant(h, d[1])-1)
	}
	iv.a, iv.aMax = fin, fin
	return iv
}

// avancerParLesEntites avance l'arrivee de chaque declaration a celle des entites qui la croisent
// (un bot lu au coup d'envoi est la des la frame 0, meme si le premier paquet le declare plus tard).
func avancerParLesEntites(decls, ents []intervalleDePresence) []intervalleDePresence {
	for i := range decls {
		for _, e := range ents {
			if e.de <= decls[i].aMax && e.aMax >= decls[i].de && e.de < decls[i].de {
				decls[i].de = e.de
			}
		}
	}
	return fusionnerLesPresences(decls)
}

// etendreParLesVies etend chaque presence aux vies qui la touchent ; une vie hors de toute presence
// en ouvre une (un corps qui bouge prouve la presence de son occupant).
func etendreParLesVies(ivs []intervalleDePresence, vies [][2]int) []intervalleDePresence {
	for _, v := range vies {
		pose := false
		for i := range ivs {
			if v[0] <= ivs[i].aMax && v[1] >= ivs[i].de {
				ivs[i].de = min(ivs[i].de, v[0])
				ivs[i].a, ivs[i].aMax = max(ivs[i].a, v[1]), max(ivs[i].aMax, v[1])
				pose = true
				break
			}
		}
		if !pose {
			ivs = append(ivs, intervalleDePresence{de: v[0], a: v[1], aMax: v[1]})
		}
	}
	return fusionnerLesPresences(ivs)
}

// enveloppeDesVies est la presence SANS ENTITE : de la premiere vie a la fin de la derniere. C'est
// le repli (cf. l'en-tete de occupants.go) ; nil sans vie.
func enveloppeDesVies(vies [][2]int) []intervalleDePresence {
	if len(vies) == 0 {
		return nil
	}
	iv := intervalleDePresence{de: vies[0][0], a: vies[0][1]}
	for _, v := range vies[1:] {
		iv.de, iv.a = min(iv.de, v[0]), max(iv.a, v[1])
	}
	iv.aMax = iv.a
	return []intervalleDePresence{iv}
}

// fusionnerLesPresences trie les intervalles et fond ceux qui se recouvrent ou se touchent.
func fusionnerLesPresences(ivs []intervalleDePresence) []intervalleDePresence {
	if len(ivs) < 2 {
		return ivs
	}
	sort.Slice(ivs, func(i, j int) bool { return ivs[i].de < ivs[j].de })
	out := []intervalleDePresence{ivs[0]}
	for _, iv := range ivs[1:] {
		dernier := &out[len(out)-1]
		if iv.de <= dernier.aMax+1 {
			dernier.a, dernier.aMax = max(dernier.a, iv.a), max(dernier.aMax, iv.aMax)
			continue
		}
		out = append(out, iv)
	}
	return out
}
