package replay

// pont_shims_test.go — LES DEUX PORTES QUE LES INSTRUMENTS DU PAQUET EMPRUNTENT.
//
// Le lot E2 (2026-09-08) a declasse le pont par morts : `nameLivesByDeaths` n'existe plus, parce
// que NOMMER n'est plus son travail — il apparie, il pose la cause de fin, il verifie
// (`identity_registry_bridge.go`). Et `buildOwners` prend desormais une [IdentityInput] entiere,
// parce qu'il compose deux lectures au lieu d'une.
//
// UNE VINGTAINE D'INSTRUMENTS DE RECHERCHE du paquet appelaient les deux anciennes formes pour
// obtenir « des vies nommees a partir d'un fil des morts », sans rien mesurer du pont lui-meme.
// Les deux shims ci-dessous leur rendent EXACTEMENT cette forme, et ils vivent dans un fichier
// de test : la production, elle, n'a plus de porte qui nomme par les morts.
//
// CE QU'ILS NE SONT PAS : un repli. Rien en production ne les appelle, et
// `TestPontParMortsNeNommePasQuandLeFilmNomme` verifie que le chemin de production, lui, ne
// nomme jamais une vie par le pont des qu'une creation est lue.

// nameLivesByDeaths rejoue l'ancienne forme : apparier, poser la cause de mort, nommer. Rend le
// nombre de vies nommees.
func nameLivesByDeaths(lives []lifeSpan, deaths []Death, off int64) int {
	paires := apparierMortsEtVies(lives, deaths, off)
	marquerCauseDeMort(lives, paires)
	return nommerParLesMorts(lives, deaths, paires)
}

// buildOwnersDeTest rejoue l'ancienne signature de `buildOwners` : des trajectoires, un fil des
// morts, une table d'index, des tirs — et AUCUNE creation, donc le pont nomme (degradation
// declaree, cf. identity_registry_bridge.go).
func buildOwnersDeTest(tracks map[uint32]slotTrack, deaths []Death, idx PlayerIndexTable,
	fire []FireEventRef) OwnerReport {
	rep, _, _ := buildOwnersFromTracks(tracks,
		IdentityInput{Deaths: deaths, PlayerIndices: idx, Fire: fire})
	return rep
}
