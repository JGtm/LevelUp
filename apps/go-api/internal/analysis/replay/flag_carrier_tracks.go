package replay

// flag_carrier_tracks.go — LA POSITION DU PORTEUR SUR LES PISTES PUBLIEES.
//
// Deux helpers, et une seule question : ou dessiner l objet porte a un instant donne ? Ils sont
// PARTAGES par les deux consommateurs du calque drapeau — `attachFlagCarryPositions`
// (flag_carries.go, la position de prise et de lacher) et `closeByFreeLives`/`flagFreeDropInside`
// (flag_objects.go, le lacher volontaire date par la vie libre de l objet). Sortis de
// flag_carries.go le 2026-09-06, deplacement PUR (le fichier franchissait les 500 lignes).

// tracksByXUID range les pistes publiees par joueur.
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
// (flag_carries_marker.go), les ramassages et les frags sous equipement actif ; sa regle de
// collision refuse deja un slot que deux joueurs se partagent. Une piste anonyme dont le pont
// ne nomme pas le slot reste ecartee : on n'invente aucun porteur.
//
// LA RESOLUTION ELLE-MEME VIT DANS `published_tracks.go` (`xuidOfPublishedTrack`), avec les deux
// autres lecteurs qui la partagent (le filtre des actions d'objectif et celui des morts neutres) :
// trois copies de la meme regle, c'est trois occasions de la faire diverger — et c'est deja
// arrive une fois (regle n°6 du depot, garde-rail dans `published_tracks_guard_test.go`).
func tracksByXUID(tracks []Track, slotXUID map[uint32]uint64) map[string][]Track {
	out := map[string][]Track{}
	for _, t := range tracks {
		xuid := xuidOfPublishedTrack(t, slotXUID)
		if xuid == "" {
			continue
		}
		out[xuid] = append(out[xuid], t)
	}
	return out
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
