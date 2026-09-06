package replay

import (
	"log/slog"
	"sort"
	"strconv"
)

// flag_carrier_tracks.go — LA POSITION DU PORTEUR SUR LES PISTES PUBLIEES.
//
// Deux helpers, et une seule question : ou dessiner l objet porte a un instant donne ? Ils sont
// PARTAGES par les deux consommateurs du calque drapeau — `attachFlagCarryPositions`
// (flag_carries.go, la position de prise et de lacher) et `closeByFreeLives`/`flagFreeDropInside`
// (flag_objects.go, le lacher volontaire date par la vie libre de l objet). Sortis de
// flag_carries.go le 2026-09-06, deplacement PUR (le fichier franchissait les 500 lignes).

// tracksByXUID range les pistes publiees par joueur. Rend aussi les SLOTS AMBIGUS : ceux dont les
// vies anonymes ont ete REFUSEES au repli ci-dessous (cf. la garde).
//
// UNE VIE ANONYME N'EST PAS UNE ABSENCE — MEME PRINCIPE QUE LE GATE DES PORTAGES DE CRANE
// (schema 43). Depuis le schema 36 (« une track = une vie ») un slot recycle publie PLUSIEURS
// pistes, et le fil des morts n'en nomme pas toujours toutes : une vie sans nom est une
// PRESENCE SANS IDENTITE PUBLIEE, pas une absence du porteur. N'indexer que les pistes NOMMEES
// faisait alors echouer `pointOfXUIDAt` sur des prises pourtant couvertes par une piste, et
// `attachFlagCarryPositions` les comptait `NoTrack` — mesure du corpus temoin : `bcb6d393`
// perd 9 prises sur 16 (toutes celles du slot 536 apres 2736, dont la vie est publiee ANONYME)
// alors que l'artefact du parc, ne les voyant qu'a travers une piste unique par slot, les
// portait toutes.
//
// L'IDENTITE VIENT DU PONT CANONIQUE, PAS D'UNE DEDUCTION LOCALE. `slotXUID` (OwnerReport,
// `ResolveSlotXUID`) est le MEME pont slot -> xuid que celui qui nomme les marques de portage
// (flag_carries_marker.go), les ramassages et les frags sous equipement actif. Une piste anonyme
// dont le pont ne nomme pas le slot reste ecartee : on n'invente aucun porteur.
//
// # LA GARDE, ET POURQUOI ELLE NE PEUT PAS VENIR DU PONT (revue DUREES-R1, constat C1)
//
// Le pont NE REFUSE PAS un slot que deux joueurs se partagent, contrairement a ce que ce
// commentaire a affirme du 2026-09-06 au 2026-09-06 : `ownersFromLives` (lives.go) compte la
// collision puis `continue` — le PREMIER nomme reste publie dans `SlotXUID`, choisi par l'ordre
// des vies et non par la proximite temporelle (owners.go le dit : « premiere nommee, collisions
// comptees »). Et `SlotCollisions` ne voit que les conflits entre vies NOMMEES : une vie ANONYME,
// exactement la population que ce repli croit, y est invisible.
//
// La garde est donc posee ICI, sur les vies PUBLIEES elles-memes — la seule matiere qui dise qui
// a occupe le slot et quand. Le repli est REFUSE des que les vies nommees du slot ne s'accordent
// pas avec le pont :
//
//	deux xuid nommes distincts sur le slot  -> refus (la vie anonyme peut etre de l'un ou de
//	                                          l'autre, rien ne tranche). Declenchement mesure :
//	                                          `084a804d` slot 734 — `[5872..6981]` nommee A,
//	                                          `[7123..7158]` ANONYME, `[7457..7591]` nommee B ;
//	                                          9 artefacts du parc sur 106 portent au moins un
//	                                          slot en collision ;
//	le pont nomme un joueur que les vies    -> refus (le pont et le document se contredisent, on
//	nommees du slot ne portent pas             ne choisit pas entre eux) ;
//	aucune vie nommee sur le slot           -> ACCEPTE : rien ne contredit le pont.
//
// CE QUE LA GARDE NE PEUT PAS ATTRAPER, et c'est ecrit pour que personne ne le croie : un slot
// occupe par A (vie nommee) puis par B dont AUCUNE vie n'est nommee sort avec un seul xuid nomme,
// et la vie de B est rangee sous A. Le document publie ne porte rien qui le distingue d'une vie
// de A coupee par un trou de replication ; le trancher demanderait de dater le pont, ce que
// `OwnerReport` ne fait pas.
func tracksByXUID(tracks []Track, slotXUID map[uint32]uint64) (map[string][]Track, []uint32) {
	nommes := namedXUIDsBySlot(tracks)
	out := map[string][]Track{}
	var ambigus []uint32
	vus := map[uint32]bool{}
	for _, t := range tracks {
		xuid := t.XUID
		if xuid == "" {
			x, ok := slotXUID[t.Slot]
			if !ok || x == 0 {
				continue // le pont ne nomme pas ce slot : aucun porteur a inventer
			}
			if slotAmbigu(nommes[t.Slot], x) {
				if !vus[t.Slot] {
					vus[t.Slot], ambigus = true, append(ambigus, t.Slot)
				}
				continue
			}
			xuid = strconv.FormatUint(x, 10)
		}
		if xuid == "" {
			continue
		}
		out[xuid] = append(out[xuid], t)
	}
	sort.Slice(ambigus, func(i, j int) bool { return ambigus[i] < ambigus[j] })
	return out, ambigus
}

// namedXUIDsBySlot rend, par slot, l'ensemble des identites que ses vies PUBLIEES portent.
func namedXUIDsBySlot(tracks []Track) map[uint32]map[string]struct{} {
	out := map[uint32]map[string]struct{}{}
	for _, t := range tracks {
		if t.XUID == "" {
			continue
		}
		if out[t.Slot] == nil {
			out[t.Slot] = map[string]struct{}{}
		}
		out[t.Slot][t.XUID] = struct{}{}
	}
	return out
}

// slotAmbigu dit si les vies NOMMEES d'un slot interdisent de preter ses vies anonymes au joueur
// que le pont designe : plusieurs occupants nommes, ou un occupant nomme qui n'est pas celui-la.
func slotAmbigu(noms map[string]struct{}, pont uint64) bool {
	if len(noms) == 0 {
		return false // rien ne contredit le pont
	}
	if len(noms) > 1 {
		return true // deux joueurs se partagent le slot : la vie anonyme n'appartient a personne
	}
	_, accord := noms[strconv.FormatUint(pont, 10)]
	return !accord
}

// logFlagAmbiguousSlots journalise les slots dont les vies anonymes ont ete refusees au repli.
// Un AVERTISSEMENT, pas une information : c'est de la matiere que le calque renonce a exploiter,
// et le seul endroit ou la limite ci-dessus se voit en production.
func logFlagAmbiguousSlots(slots []uint32) {
	if len(slots) == 0 {
		return
	}
	slog.Warn("rejeu : vies anonymes refusees au calque drapeau — slot partage par plusieurs joueurs",
		"slots", slots, "nombre", len(slots))
}

// pointOfXUIDAt rend le point PUBLIE le plus proche de la frame demandee, parmi les pistes d'un
// joueur. Rend (_, false) si aucune piste n'a de point a moins d'une frame — le drapeau n'aurait
// alors pas de position a dessiner, et on prefere ne rien poser.
func pointOfXUIDAt(tracks []Track, frame int) (Point, bool) {
	best, bd, found := Point{}, 0, false
	for _, tr := range tracks {
		for _, p := range tr.Points {
			d := p.T - frame
			if d < 0 {
				d = -d
			}
			if !found || d < bd {
				best, bd, found = p, d, true
			}
		}
	}
	return best, found && bd <= 1
}
