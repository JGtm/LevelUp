package replay

// published_tracks.go — LE FILTRE UNIQUE « ce calque n'existe que là où une trajectoire
// existe ».
//
// POURQUOI UN SEUL ENDROIT. Quatre calques (tirs, lancers, armes portées, inventaire)
// posaient chacun sa copie du même geste : indexer les slots publiés, garder ce qui tombe
// dedans, rendre nil si tout tombe. Quatre copies d'une règle, c'est quatre occasions de
// la faire diverger — et une divergence ici ne se voit PAS à l'écran : un calque garderait
// des éléments qu'un autre écarte, sur la même fiche, sans qu'aucun compteur ne bouge.
//
// LA RÈGLE, ELLE, N'EST PAS COSMÉTIQUE : un tir posé sur un slot sans trajectoire n'a
// aucune fiche où s'afficher ; le client le dessinerait dans le vide.
//
// GARDE-RAIL : `published_tracks_guard_test.go` interdit qu'un second endroit du paquet
// reconstruise un ensemble de slots publiés — ou de JOUEURS publiés. Une factorisation sans
// garde-rail re-diverge (règle n°6 du dépôt), et c'est exactement ce qui est arrivé : le
// garde-rail d'origine ne filtrait que le motif keyé par `.Slot`, si bien que les deux seuls
// filtres keyés par XUID (`objectives.go`, `neutral_deaths.go`) ont divergé des onze autres
// sans qu'il les voie — la divergence INVISIBLE que ce fichier existe pour empêcher.

import "strconv"

// publishedSlots indexe les slots qui portent une trajectoire publiée.
//
// C'est LE SEUL endroit du paquet où cet ensemble se construit (cf. garde-rail).
func publishedSlots(tracks []Track) map[uint32]bool {
	out := make(map[uint32]bool, len(tracks))
	for _, t := range tracks {
		out[t.Slot] = true
	}
	return out
}

// xuidOfPublishedTrack rend le joueur d'une piste publiée : son nom LU, sinon celui que le PONT
// CANONIQUE donne à son slot. Chaîne vide = la vie est anonyme ET le pont ne nomme pas son slot.
//
// UNE VIE ANONYME N'EST PAS UNE ABSENCE. Depuis le schéma 36 (« une track = une vie ») un slot
// recyclé publie PLUSIEURS pistes, et le fil des morts n'en nomme pas toutes : `Track.XUID` vide
// dit que la vie n'a pas été NOMMÉE, jamais que le joueur n'était pas là (contrat de
// `Track.XUID`, document.go — 15 vies sur 105 sur le film de référence, dont 6 survivants de fin
// de partie que le film ne clôt par aucun événement). Cadencer un filtre « le joueur a-t-il une
// trajectoire publiée » sur le seul nom LU supprime donc TOUTES les données d'un joueur dont
// aucune vie n'est nommée — mesuré : `3372e7eb`, 35 actions d'objectif sur 76 (46 %).
//
// L'IDENTITÉ VIENT DU PONT, PAS D'UNE DÉDUCTION LOCALE : `slotXUID` (`OwnerReport`,
// `ResolveSlotXUID`) est le même pont qui nomme les marques de portage, les ramassages et les
// positions de porteur de drapeau (`tracksByXUID`), et sa règle de collision refuse déjà un slot
// que deux joueurs se partagent. Un slot que le pont ne nomme pas reste écarté : on n'invente
// aucun joueur.
//
// C'est LE SEUL endroit du paquet où cette résolution s'écrit (cf. garde-rail).
func xuidOfPublishedTrack(t Track, slotXUID map[uint32]uint64) string {
	if t.XUID != "" {
		return t.XUID
	}
	if x, ok := slotXUID[t.Slot]; ok && x != 0 {
		return strconv.FormatUint(x, 10)
	}
	return ""
}

// publishedXUIDs indexe les JOUEURS qui portent une trajectoire publiée — nom lu ou pont.
//
// C'est LE SEUL endroit du paquet où cet ensemble se construit (cf. garde-rail).
func publishedXUIDs(tracks []Track, slotXUID map[uint32]uint64) map[string]bool {
	out := make(map[string]bool, len(tracks))
	for _, t := range tracks {
		if x := xuidOfPublishedTrack(t, slotXUID); x != "" {
			out[x] = true
		}
	}
	return out
}

// keepOfPublishedTracks filtre un calque sur les trajectoires publiées.
//
// `keep` reçoit l'élément ET l'ensemble des slots publiés : c'est ce qui permet aux
// lancers de grenade de garder leur exemption (une position lue sur le PROJECTILE ne
// dépend d'aucune trajectoire) sans dupliquer la boucle.
//
// CONTRAT PRÉSERVÉ À L'IDENTIQUE des quatre copies qu'elle remplace : entrée vide -> nil,
// filtrage EN PLACE (réutilise le tableau d'entrée), sortie vide -> nil (et non un tableau
// vide, que `omitempty` ne distinguerait pas de l'absence de mesure).
func keepOfPublishedTracks[T any](items []T, tracks []Track, keep func(T, map[uint32]bool) bool) []T {
	if len(items) == 0 {
		return nil
	}
	published := publishedSlots(tracks)
	out := items[:0]
	for _, it := range items {
		if keep(it, published) {
			out = append(out, it)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
