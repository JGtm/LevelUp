package replay

// zone_states_hill_activation.go — QUAND LA PREMIERE COLLINE APPARAIT.
//
// L ASSEMBLAGE N EST PAS UN PARCOURS SEQUENTIEL DU FILM : une fois la colline de la 1re periode
// connue (par sa garde, zone_states_hill_garde.go), sa periode peut commencer a l instant ou la
// colline existe en jeu, et non au premier contact d un joueur.
//
// CE QUE LE FILM DIT DE CET INSTANT (mesure sur les films a colline du parc, journal du plan
// `.ai/PLAN_KOTH_REJEU_2026-10-09.md`) :
//
//	images-cles   l objet de mode (le bloc designateur, proprietaire, pousseur, jauge) est ABSENT
//	              d une image-cle et PRESENT a la suivante, avec la designation de la 1re colline.
//	              Les images-cles tombent toutes les 20 s : elles BORNENT la creation de l objet
//	              sans la dater. Rapportee au coup d envoi, la fenetre va de -17,8 s a +13,4 s
//	              selon le film, et l intersection des fenetres de onze films est ]-6,6 s ;
//	              +2,3 s] : toutes contiennent le coup d envoi.
//	trames delta  aucune lecture CHAINEE du bloc avant le premier contact ; la creation de
//	              l objet est un record que l ancrage par masque ne reconnait pas.
//
// LA REGLE : la 1re periode commence au COUP D ENVOI du match (`T0FilmMs`, premier mouvement des
// pistes), ramene dans la fenetre des images-cles — repli NOMME et COMPTE
// (`repli_colline_premiere_au_coup_d_envoi`), le film ne datant pas la creation. Sans coup
// d envoi mesure, elle commence a la premiere image-cle qui porte le bloc (une LECTURE, borne
// haute). Sans l une ni l autre, au premier contact. Jamais apres le premier contact.

import (
	"slices"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// zoneCles est ce que les images-cles disent des slots ti=13 : les frames des images-cles qui
// portent au moins un record ti=13 (triees ; -1 pour une image-cle anterieure a la frame 0), et
// la premiere image-cle ou chaque slot apparait.
type zoneCles struct {
	frames  []int
	premier map[uint32]int
}

// zoneClesDesSlots pose les lectures d image-cle sur la grille de frames.
func zoneClesDesSlots(keyReads []grammar.ManagedPropertyRead, c zoneCtx) zoneCles {
	out := zoneCles{premier: map[uint32]int{}}
	for _, r := range keyReads {
		f := zoneKeyFrameOf(r.TimestampUS, c)
		if !slices.Contains(out.frames, f) {
			out.frames = append(out.frames, f)
		}
		if p, ok := out.premier[r.Slot]; !ok || f < p {
			out.premier[r.Slot] = f
		}
	}
	slices.Sort(out.frames)
	return out
}

// zoneKeyFrameOf convertit l horodatage d une image-cle en frame : -1 avant la frame 0, la
// derniere frame au-dela de l axe.
func zoneKeyFrameOf(ts uint64, c zoneCtx) int {
	if ts < c.origin || c.step == 0 {
		return -1
	}
	return min(int((ts-c.origin)/c.step), c.frames-1) //nolint:gosec // ecart borne par l axe
}

// fenetre rend la fenetre ]lo-1 ; hi] des images-cles qui encadrent l apparition du premier des
// `slots` : `lo` est la frame qui suit la derniere image-cle sans aucun d eux (0 sans elle), `hi`
// la premiere image-cle qui en porte un. Faux quand aucune image-cle ne les porte.
func (k zoneCles) fenetre(slots []uint32) (lo, hi int, ok bool) {
	hi = -2
	for _, s := range slots {
		if f, found := k.premier[s]; found && (hi == -2 || f < hi) {
			hi = f
		}
	}
	if hi == -2 {
		return 0, 0, false
	}
	for _, f := range k.frames {
		if f < hi {
			lo = f + 1
		}
	}
	return max(lo, 0), max(hi, 0), true
}

// hillActivationCtx porte ce dont la datation de la 1re colline a besoin.
type hillActivationCtx struct {
	cles zoneCles
	// kickoff est la frame du coup d envoi, et hasKickoff dit qu elle est mesuree.
	kickoff    int
	hasKickoff bool
	fb         *fallback.Compteur
}

// hillFirstActivation rend la frame ou commence la 1re periode (cf. l en-tete) : le coup d envoi
// ramene dans la fenetre des images-cles du bloc, a defaut la premiere image-cle qui le porte, a
// defaut le premier contact — jamais au-dela du premier contact.
func hillFirstActivation(d hillDesignator, bloc []uint32, a hillActivationCtx) int {
	lo, hi, ok := a.cles.fenetre(append([]uint32{d.slot}, bloc...))
	if !ok {
		return d.first
	}
	start := hi
	if a.hasKickoff {
		a.fb.Declenche(fallback.NomCollinePremiereAuCoupDEnvoi)
		start = min(max(a.kickoff, lo), hi)
	}
	return min(start, d.first)
}
