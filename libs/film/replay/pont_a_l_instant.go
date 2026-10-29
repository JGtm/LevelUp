package replay

// pont_a_l_instant.go — LES DEUX ADAPTATEURS D'HORLOGE DU REGISTRE A L'INSTANT (lot 6.1).
//
// # POURQUOI ILS EXISTENT
//
// `IdentityRegistry.XUIDNumAt` prend un instant en MICROSECONDES DE L'HORLOGE DU FILM — la seule
// horloge sur laquelle les vies sont datees. Or les calques qui l'interrogent portent leur instant
// dans deux autres unites :
//
//   - la FRAME du document (`EquipmentEpisode.T0/T1`, `Pickup.T`, `Inventory.T`) : l'axe publie,
//     dont l'origine et le pas vivent dans `replayClock` ;
//   - la MILLISECONDE DE L'HORLOGE DU MATCH (`HeldObjectEvent.TimeMS`, le fil des morts) : celle
//     que `DeathOffsetMS` relie a l'horloge du film, dans le sens `film = match + offset` (meme
//     derivation qu'en tete de bomb_carries.go).
//
// Ecrire la conversion sur chaque site rejouerait a l'identique le defaut que ce lot corrige :
// deux copies d'un meme calcul d'horloge divergent au premier ajustement. Elles vivent donc ici,
// une fois, et les calques recoivent une FONCTION — ce qui les garde testables sans registre.
//
// # POURQUOI UNE FONCTION ET PAS LE REGISTRE
//
// Les cinq lecteurs migres sont des fonctions PURES, testees sur des donnees synthetiques. Leur
// passer le registre les obligerait a en fabriquer un pour chaque cas ; leur passer un
// `func(slot, instant) uint64` laisse le test dire directement « a cet instant, ce siege est a ce
// joueur », qui est exactement la propriete exercee.

// occupantParFrame rend « qui occupe ce siege a cette FRAME du document ».
//
// UNE FRAME NEGATIVE NE DESIGNE PERSONNE, et la garde n'est pas theorique : les instants recales
// sur l'origine du document (`attachEpisodeKills`) peuvent tomber avant la frame 0, et convertir
// un negatif en `uint64` le ferait boucler vers le haut de l'axe — un occupant tire au hasard.
func occupantParFrame(reg IdentityRegistry, clk replayClock) func(slot uint32, frame int) uint64 {
	return func(slot uint32, frame int) uint64 {
		if frame < 0 || clk.step == 0 {
			return 0
		}
		return reg.XUIDNumAt(slot, clk.origin+uint64(frame)*clk.step)
	}
}

// occupantParMatchMS rend « qui occupe ce siege a cet instant de l'horloge du MATCH ».
//
// LE CALAGE EST CELUI QUE LE PONT A MESURE (`DeathOffsetMS`), applique dans le sens
// `horlogeFilm = horlogeMatch + DeathOffsetMS` — le meme que `deathTimesByVictimMS` et que
// `bombHeldEventsOf`, en sens inverse. Un instant qui retombe avant l'origine du film ne designe
// personne, pour la meme raison que ci-dessus.
func occupantParMatchMS(reg IdentityRegistry) func(slot uint32, matchMS int) uint64 {
	offset := reg.DeathOffsetMS()
	return func(slot uint32, matchMS int) uint64 {
		filmMS := int64(matchMS) + offset
		if filmMS < 0 {
			return 0
		}
		return reg.XUIDNumAt(slot, uint64(filmMS)*1000)
	}
}
