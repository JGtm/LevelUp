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
// reconstruise un ensemble de slots publiés. Une factorisation sans garde-rail re-diverge
// (règle n°6 du dépôt).

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
