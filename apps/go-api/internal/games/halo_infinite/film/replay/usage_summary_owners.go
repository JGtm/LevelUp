package replay

// usage_summary_owners.go — « A QUI ETAIT CE SLOT A CET INSTANT », et les deux replis qui
// repondent quand aucune vie ne couvre l'instant.
//
// SORTI DE `usage_summary.go` LE 2026-09-16 (revue de jalon M1) : le cablage des compteurs de
// repli aurait porte ce fichier de 538 a 592 lignes, au-dela du seuil de 500 du depot et dans
// le sens que la revue reproche justement au jalon (six fichiers deja trop longs ont GROSSI).
// Le decoupage est un DEPLACEMENT PUR : aucune regle d'attribution ne change, et le resolveur
// est un sujet a lui — les trois canaux qui l'appellent (`usage_summary.go`) restent la-bas.

import (
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/facts/fallback"
)

// usageOwners répond « à qui était ce slot À CET INSTANT ».
//
// POURQUOI PAS UNE SIMPLE TABLE slot -> joueur (constat C5 de la revue REG-R1, 2026-09-06).
// L'agrégat « dernier gagnant » créditait TOUS les gestes d'un slot recyclé à son SECOND
// occupant, y compris ceux de la vie du premier. Le cas existe au parc (`879a4dba`,
// `slotCollisions = 1`), et le correctif « une track = une vie » l'élargit : les épisodes et
// les tractions des vies non dernières n'existaient pas avant pour être mal attribués.
// L'instant est disponible à chacun des trois sites d'appel (`GrappleLine.T0`,
// `EquipmentEpisode.T0`, `EquipmentPlacement.T0`) : il n'y avait rien à deviner.
type usageOwners struct {
	// parVie : pour chaque slot, ses vies dans l'ordre chronologique.
	parVie map[uint32][]usageVie
	// dernier : le repli « dernier gagnant », pour un instant qu'aucune vie ne couvre.
	dernier map[uint32]string
	// fb : le compteur de la projection en cours. Nil est valide (il ne compte rien) — c'est
	// ce qui laisse un test construire un `usageOwners` à la main sans plomberie.
	fb *fallback.Compteur
}

// usageVie est une vie publiée réduite à ce que l'attribution consomme.
type usageVie struct {
	from, to int
	xuid     string // "" pour un bot ou une vie anonyme
}

// at rend le propriétaire du slot à cette image : la vie qui la couvre, sinon le dernier
// occupant connu.
//
// LE REPLI NE JOUE PAS SUR LES DEUX CANAUX QUI L'APPELLENT : les `T0` des tractions et des
// épisodes sont bornés à la fenêtre de leur vie par leurs assembleurs, et la mesure le
// confirme — 23/23 et 31/31 tractions, 7/7 et 15/15 épisodes tombent DANS une fenêtre publiée
// sur les films cuits. Il reste pour ne rien perdre si un jour un `T0` sortait, et il vaut
// alors exactement ce que rendait la table d'avant.
//
// LES POSES, ELLES, PASSENT PAR `atOrJustBefore` : leur `T0` n'est borné à aucune fenêtre.
func (o usageOwners) at(slot uint32, frame int) string {
	for _, v := range o.parVie[slot] {
		if frame >= v.from && frame <= v.to {
			return v.xuid
		}
	}
	return o.repliDernierOccupant(slot)
}

// repliDernierOccupant — LE REPLI NOMMÉ ET COMPTÉ (D14) des deux résolveurs : aucune vie du
// slot ne couvre l'image, le DERNIER occupant connu du slot sur tout le match est crédité.
//
// IL NE SE COMPTE QUE QUAND IL DÉCIDE. Un slot sans dernier occupant connu rend la chaîne vide,
// et l'appelant n'attribue alors RIEN : aucun fait n'est décidé, donc rien à compter. Compter
// ce cas gonflerait le compte d'un repli qui ne s'est pas appliqué, et D14 (d) fait ensuite
// lire ce compte comme la fréquence à laquelle un fait publié vient d'un repli.
func (o usageOwners) repliDernierOccupant(slot uint32) string {
	x := o.dernier[slot]
	if x != "" {
		o.fb.Declenche(fallback.NomGesteDernierOccupantDuMatch)
	}
	return x
}

// atOrJustBefore rend le propriétaire du slot à cette image, ou À DÉFAUT celui de la vie qui
// vient de s'y achever. Jumeau exact d'`ownerAtFrameOrLast` (rosterLogic.ts).
//
// POURQUOI IL EXISTE (constat N-3 de la revue REG-R2, 2026-09-06). Un objet LÂCHÉ à la mort
// porte `t0 = finVie + 1` — le poseur n'occupe déjà plus le slot —, et rien côté Go ne borne
// `EquipmentPlacement.T0` à une fenêtre publiée. Le repli « dernier occupant du match » de `at`
// créditait donc ces poses au joueur SUIVANT sur un slot repris, ce qui est précisément la
// règle que le correctif du 2026-09-06 déclare avoir supprimée. Et ce n'est pas un cas de bord :
// 32 à 95 % des poses d'un film tombent hors de toute fenêtre publiée (153/351, 443/466,
// 34/105 sur trois films).
//
// L'ORDRE EST CHRONOLOGIQUE (`usageSlotOwners` trie), donc « la dernière vie vue avant l'image »
// est bien la plus récente qui s'est achevée. Si AUCUNE ne précède — une pose datée avant la
// première vie publiée du slot —, la PREMIÈRE vie du slot répond : sur un slot mono-identité
// c'est le même joueur qu'avant ce correctif, et sur un slot recyclé c'est son premier occupant,
// jamais le dernier. Ainsi ce canal ne perd aucune ligne au passage.
func (o usageOwners) atOrJustBefore(slot uint32, frame int) string {
	vies := o.parVie[slot]
	dernierVu, vu := "", false
	for _, v := range vies {
		if frame < v.from {
			break // triées : ni celle-ci ni les suivantes ne couvrent ni ne précèdent
		}
		if frame <= v.to {
			return v.xuid
		}
		dernierVu, vu = v.xuid, true
	}
	if vu {
		return dernierVu
	}
	if len(vies) > 0 {
		return o.repliPremiereVieDuSlot(vies)
	}
	return o.repliDernierOccupant(slot)
}

// repliPremiereVieDuSlot — LE REPLI NOMMÉ ET COMPTÉ (D14) de la pose datée AVANT la première
// vie publiée du slot : faute de vie précédente, la PREMIÈRE vie du slot est créditée.
//
// MÊME RÈGLE DE COMPTE QUE [usageOwners.repliDernierOccupant] : une première vie anonyme (bot,
// vie que le pont n'a pas nommée) rend la chaîne vide et ne décide aucun fait.
func (o usageOwners) repliPremiereVieDuSlot(vies []usageVie) string {
	x := vies[0].xuid
	if x != "" {
		o.fb.Declenche(fallback.NomGestePremiereVieDuSlot)
	}
	return x
}

// usageSlotOwners construit ce résolveur. L'ordre des joueurs reste celui de la construction
// web (buildPlayers + indexBySlot de rosterLogic.ts) : roster du film puis pistes, vies de
// chacun triées par frame de début — c'est lui qui décide du repli « dernier gagnant ». La clé
// rendue est le xuid, ou "" pour une vie de BOT ou anonyme (un bot n'a pas de xuid : ses gestes
// n'entrent dans aucune ligne persistée, mais il OCCUPE ses slots — les attribuer au précédent
// occupant humain serait faux).
func usageSlotOwners(doc *ReplayDocument, fb *fallback.Compteur) usageOwners {
	type joueur struct {
		xuid  string // "" pour un bot : identité non persistable
		lives []*Track
	}
	index := map[string]int{}
	var ordre []*joueur
	// viesSansNom : les vies que ni le fil des morts, ni le pont, ni le relais n'ont nommees.
	// Elles n'ouvrent AUCUNE ligne (une ligne est keyee par xuid) mais elles OCCUPENT leur slot.
	var viesSansNom []*Track
	ajouter := func(cle, xuid string) *joueur {
		if i, ok := index[cle]; ok {
			return ordre[i]
		}
		index[cle] = len(ordre)
		j := &joueur{xuid: xuid}
		ordre = append(ordre, j)
		return j
	}
	for i := range doc.Roster {
		e := &doc.Roster[i]
		switch {
		case e.XUID != "":
			ajouter(e.XUID, e.XUID)
		case e.Bot && e.Name != "":
			ajouter("bot:"+e.Name, "")
		}
	}
	for i := range doc.Tracks {
		tr := &doc.Tracks[i]
		var j *joueur
		switch {
		case tr.XUID != "":
			j = ajouter(tr.XUID, tr.XUID)
		case tr.Bot != "":
			j = ajouter("bot:"+tr.Bot, "")
		default:
			// UNE VIE QUE LE NOMMAGE N'A PAS RESOLUE OCCUPE QUAND MEME SON SLOT (residu B,
			// instruit le 2026-09-06). L'ecarter la faisait retomber `at()` sur
			// `dernier[slot]` — le DERNIER occupant du match — pour tout instant qu'elle
			// couvre : un geste mesure pendant cette vie etait credite a la LIGNE d'un autre
			// joueur. Un faux positif nomme est plus couteux qu'une ligne manquante : la vie
			// entre avec un xuid VIDE, ce qui rend l'instant non attribuable au lieu de
			// l'attribuer a tort. C'est deja le traitement des vies de BOT, pour la meme
			// raison (« les attribuer au precedent occupant humain serait faux »).
			viesSansNom = append(viesSansNom, tr)
			continue
		}
		j.lives = append(j.lives, tr)
	}
	out := usageOwners{parVie: map[uint32][]usageVie{}, dernier: map[uint32]string{}, fb: fb}
	for _, j := range ordre {
		sort.SliceStable(j.lives, func(a, b int) bool {
			return j.lives[a].StartFrame < j.lives[b].StartFrame
		})
		for _, tr := range j.lives {
			out.parVie[tr.Slot] = append(out.parVie[tr.Slot],
				usageVie{from: tr.StartFrame, to: tr.EndFrame, xuid: j.xuid})
			out.dernier[tr.Slot] = j.xuid
		}
	}
	for _, tr := range viesSansNom {
		out.parVie[tr.Slot] = append(out.parVie[tr.Slot],
			usageVie{from: tr.StartFrame, to: tr.EndFrame, xuid: ""})
	}
	for slot := range out.parVie {
		vies := out.parVie[slot]
		sort.SliceStable(vies, func(a, b int) bool { return vies[a].from < vies[b].from })
	}
	return out
}

// usageFilmIndexOwners — index de film -> xuid, via le roster. Même garde que le
// web : seul un joueur dont AU MOINS UNE VIE est publiée reçoit des lancers (une
// entrée de roster sans piste n'a été mesurée sur aucun canal).
//
// LA GARDE SE LIT SUR TOUTES LES VIES, PAS SUR LE DERNIER OCCUPANT DE CHAQUE SLOT
// (constat N-4 de la revue REG-R2, 2026-09-06). Bâtie sur `dernier`, elle effaçait un joueur
// dont TOUTES les vies sont sur des slots repris ensuite par un autre : il perdait la totalité
// de ses lancers alors que l'en-tête ci-dessus lui en promet — il a bien une vie publiée. Le
// défaut est antérieur au correctif par vie, mais celui-ci apporte la table qui le ferme.
func usageFilmIndexOwners(doc *ReplayDocument, slotOwner usageOwners) map[int]string {
	avecVie := make(map[string]bool, len(slotOwner.dernier))
	for _, vies := range slotOwner.parVie {
		for _, v := range vies {
			if v.xuid != "" {
				avecVie[v.xuid] = true
			}
		}
	}
	out := make(map[int]string, len(doc.Roster))
	for i := range doc.Roster {
		e := &doc.Roster[i]
		if e.XUID != "" && avecVie[e.XUID] {
			out[e.FilmIndex] = e.XUID
		}
	}
	return out
}
