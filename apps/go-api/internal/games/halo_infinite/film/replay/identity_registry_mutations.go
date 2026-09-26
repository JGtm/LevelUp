package replay

// identity_registry_mutations.go — LES POSEURS DU REGISTRE : LES SEULS A MUTER LE PONT.
//
// # POURQUOI UN FICHIER A PART (lot P2-bis, 2026-09-08)
//
// `identity_registry.go` porte le TYPE, la construction et les ACCESSEURS ; il etait a 471 L
// pour un seuil de 500 (decouverte 3 du lot P2). Les poseurs en sortent avant que la deuxieme
// voie de deduction ne le fasse franchir le seuil. La coupure n'est pas arbitraire : ce fichier
// est le seul endroit ou les trois tables du pont (`lives`, `SlotXUID`, `Owner`, plus
// `SlotAmbiguous`) BOUGENT — et elles y bougent toujours ENSEMBLE. Deux d'entre elles qui
// divergeraient diraient deux choses du meme slot, l'invariant qu'`ownersFromLives` impose deja
// aux lectures.
//
// # LES DECIDEURS RESTENT DEHORS
//
// `identity_registry_elimination.go` (l'unicite sur le match) et
// `identity_registry_exclusion.go` (l'unicite sur l'intervalle) DECIDENT ; ce fichier POSE.
// Aucune regle de nommage n'est ecrite ici, et c'est ce qui permet d'en ajouter une sans
// toucher aux tables.
//
// # LA CAUSE DE FIN N'EST JAMAIS TOUCHEE
//
// Une deduction dit a QUI la vie appartient, jamais COMMENT elle s'est terminee. Confondre les
// deux fabrique une mort pour un survivant — le P0 de la ronde 2 du 2026-09-07.

// poserIdentiteDeVie pose une identite DEDUITE sur UNE vie designee par son indice, et fait
// suivre les tables du pont. Rend faux quand la vie n'existe pas, est deja nommee, ou que le
// xuid est nul.
//
// # LE SLOT QUE DEUX JOUEURS SE PARTAGENT DEVIENT AMBIGU, IL N'EST PAS ECRASE
//
// Le pont aplati (`SlotXUID`) ne retient qu'UN occupant par slot. Si la vie deduite tombe sur un
// slot que le pont attribue deja a QUELQU'UN D'AUTRE, ecraser publierait un nom arbitraire sur
// les lecteurs du pont aplati : on marque le slot AMBIGU, ce qui fait taire `XUIDAt`,
// `PontDeSlot` et `PontEpure` — exactement ce que `ownersFromLives` fait des collisions de
// lecture. La VIE, elle, garde son identite : elle est bornee dans le temps, elle ne ment pas.
func (r *IdentityRegistry) poserIdentiteDeVie(i int, xuid uint64, pi int, piConnu bool,
	nomPar string) bool {
	if xuid == 0 || i < 0 || i >= len(r.own.lives) || r.own.lives[i].xuid != 0 {
		return false
	}
	if r.own.lives[i].bid != "" {
		// LA VIE PORTE DEJA UN BOT (lot 4.3) : une deduction ne remplace pas une source. Sans
		// cette garde, l'elimination sur le roster — qui raisonne sur un SLOT entier — reprenait
		// les vies que le tableau de l'API venait d'attribuer a un bot, et le siege partage
		// retombait sur un seul occupant.
		return false
	}
	slot := r.own.lives[i].slot
	r.own.lives[i].xuid = xuid
	r.own.lives[i].nomPar = nomPar
	r.deducedLives[i] = true
	if deja, connu := r.own.SlotXUID[slot]; connu && deja != xuid {
		if r.own.SlotAmbiguous == nil {
			r.own.SlotAmbiguous = map[uint32]bool{}
		}
		r.own.SlotAmbiguous[slot] = true
		r.own.SlotCollisions = len(r.own.SlotAmbiguous)
		return true
	}
	if r.own.SlotXUID != nil {
		r.own.SlotXUID[slot] = xuid
	}
	if piConnu && r.own.Owner != nil {
		if _, deja := r.own.Owner[slot]; !deja {
			r.own.Owner[slot] = pi
		}
	}
	return true
}

// poserBidDeVie pose l'identifiant d'un BOT sur UNE vie. Rend faux quand la vie n'existe pas ou
// porte deja une identite.
//
// # POURQUOI IL NE TOUCHE NI `SlotXUID` NI `Owner`
//
// Les deux tables du pont aplati sont indexees par XUID et par index de JOUEUR : un bot n'a ni
// l'un ni l'autre a y mettre. `Owner` porte deja le siege du bot — `ownersFromCreations` l'y pose
// depuis le lot E2, et c'est par la que `nameBotTracks` nomme ses pistes. Ecrire ici reviendrait
// donc soit a inventer une valeur, soit a repeter ce que la lecture directe a deja pose.
//
// LA VIE, ELLE, EST BORNEE DANS LE TEMPS : c'est le seul endroit ou l'identite d'un bot peut
// vivre sans mentir sur un siege que deux occupants successifs se partagent.
func (r *IdentityRegistry) poserBidDeVie(i int, bid string) bool {
	if bid == "" || i < 0 || i >= len(r.own.lives) {
		return false
	}
	if r.own.lives[i].xuid != 0 || r.own.lives[i].bid != "" {
		return false
	}
	r.own.lives[i].bid = bid
	r.own.lives[i].nomPar = NomParTableauAPI
	return true
}

// poserIdentiteDeduite pose une identite DEDUITE sur TOUTES les vies anonymes d'un slot. C'est
// la forme qu'appelle l'elimination sur le roster : elle raisonne sur un slot ENTIER (« un seul
// slot sans aucune vie nommee »), donc toutes ses vies reviennent au meme joueur.
//
// Rend le nombre de vies nommees.
func (r *IdentityRegistry) poserIdentiteDeduite(slot uint32, xuid uint64, pi int, piConnu bool) int {
	n := 0
	for i := range r.own.lives {
		if r.own.lives[i].slot != slot {
			continue
		}
		if r.poserIdentiteDeVie(i, xuid, pi, piConnu, NomParElimination) {
			n++
		}
	}
	return n
}
