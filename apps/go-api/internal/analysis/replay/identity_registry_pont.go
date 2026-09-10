package replay

// identity_registry_pont.go — LA CONSTRUCTION DU PONT BRUT, ET SES DEUX ACCESSEURS GARDES.
//
// TROISIEME MOITIE DU MEME PRODUCTEUR (lot E2, 2026-09-08). `identity_registry.go` porte les
// types, l'ordre des etapes et les accesseurs ; `identity_registry_mutations.go` porte les
// poseurs (lot P2-bis) ; ce fichier-ci porte la CONSTRUCTION du pont brut — `buildOwners`, ce
// qu'il compose, et les deux methodes d'`OwnerReport` qui portent une garde (`xuidNumAt`,
// `NamingBridge`).
//
// LA SEPARATION EST ARITHMETIQUE, PAS DOCTRINALE : le lien direct du lot E2 a porte
// `identity_registry.go` a 505 lignes pour un seuil de 500. Aucune REGLE de nommage ne vit ici
// non plus — les trois decideurs (`_creation.go`, `_elimination.go`, `_exclusion.go`) restent
// hors allowlist et passent par les poseurs.

// buildOwners construit le pont : le lien DIRECT du record de creation d'abord, le fil des morts
// en temoin.
//
// PAS DE REPLI. Si le film ne porte ni lien direct ni fil des morts, le pont est VIDE et aucun
// tir n'est publie — c'est le comportement voulu. Un rejeu muet se voit ; un rejeu qui pose des
// tirs sur le mauvais joueur ne se voit pas, et c'est bien pire.
//
// APPELANT UNIQUE : [BuildIdentityRegistry]. Le garde-rail `archlint` l'exige.
func buildOwners(in IdentityInput) (OwnerReport, creationReport, bridgeVerification) {
	return buildOwnersFromTracks(indexBySlot(in.Positions), in)
}

// buildOwnersFromTracks est [buildOwners] sur des trajectoires DEJA indexees par slot.
//
// LA SEPARATION EXISTE POUR LES INSTRUMENTS : une vingtaine de tests de recherche du paquet
// construisent leurs trajectoires a la main plutot que de fabriquer des positions. Leur faire
// passer par `indexBySlot` demanderait de reconstituer des `BipedPosition` a partir de vies —
// c'est-a-dire d'inventer la donnee que l'instrument mesure.
func buildOwnersFromTracks(tracks map[uint32]slotTrack,
	in IdentityInput) (OwnerReport, creationReport, bridgeVerification) {
	rep := OwnerReport{Owner: map[uint32]int{}, SlotXUID: map[uint32]uint64{}}
	deaths, idx := in.Deaths, in.PlayerIndices
	if len(tracks) == 0 || len(idx.ByXUID) == 0 {
		return rep, creationReport{}, bridgeVerification{}
	}
	lives := buildLifeSpans(tracks)
	rep.LivesTotal = len(lives)
	rep.lives = lives
	// LE LIEN DIRECT EN PREMIER, ET SANS CONDITION : c'est la doctrine « l'index est l'index ».
	crea := nommerViesParCreations(lives, in.BipedCreations, idx, in.Bots)
	// LE PONT PAR MORTS ENSUITE, EN TEMOIN. Son appariement pose la CAUSE de fin — la seule qui
	// dise « ce joueur est mort » — puis confronte la victime au joueur que le film ecrit.
	off, matched, second := bestDeathOffset(lives, deaths)
	rep.DeathOffsetMS, rep.DeathOffsetMatches = off, matched
	rep.DeathOffsetRunnerUp = second
	paires := apparierMortsEtVies(lives, deaths, off)
	marquerCauseDeMort(lives, paires)
	verif := verifierParLesMorts(lives, deaths, paires, in.MatchID)
	if crea.Slots == 0 {
		// AUCUNE LECTURE DIRECTE RECUE : degradation complete et declaree (cf.
		// identity_registry_bridge.go). Sur un film dont les creations sont lues, cette
		// branche ne s'execute pas et `BridgeNamedLives` reste a zero.
		verif.NamedByBridge = nommerParLesMorts(lives, deaths, paires)
	}
	rep.DeathsNamed = verif.Matched
	if crea.Lues() == 0 && verif.NamedByBridge == 0 {
		return rep, crea, verif
	}
	rep.IndexReadings = idx.Readings
	rep.IndexDisagreements = idx.Disagreements
	owners, byXUID, ambigus := ownersFromLives(lives, idx.ByXUID)
	// LE PONT SLOT -> INDEX DES CREATIONS COMPLETE CELUI DES VIES, et il le fait pour les corps
	// que les vies nommees ne peuvent PAS porter : un bot n'a pas de xuid, donc `ownersFromLives`
	// ignore son slot. Le record de creation, lui, porte l'index quel qu'il soit.
	for s, pi := range ownersFromCreations(in.BipedCreations) {
		if _, connu := owners[s]; !connu {
			owners[s] = pi
		}
	}
	rep.SlotAmbiguous = ambigus
	rep.SlotCollisions = len(ambigus)
	rep.FromDeaths = len(owners)
	// LES FERMETURES VIENNENT APRES LA LECTURE, JAMAIS A SA PLACE (cf. closures.go). Elles ne
	// touchent que les vies que le fil des morts n'a pas nommees, et elles s'abstiennent des que
	// deux candidats subsistent. `FromDeaths` est fige AVANT, pour que l'ecart entre lui et
	// `len(Owner)` reste lisible : c'est exactement ce que les fermetures ont ajoute.
	rep.Owner, rep.Closures = closeBridge(tracks, owners, lives, deaths, off, idx.ByXUID, in.Fire)
	rep.SlotXUID = extendSlotXUID(byXUID, rep.Owner, idx.ByXUID)
	// LES FERMETURES NOMMENT AUSSI LA VIE (lot identite des vies, 2026-09-02) : le nommage des
	// tracks se fait desormais PAR VIE, et une vie fermee sans identite redeviendrait anonyme a
	// l'ecran alors que le pont la connait. C'est LA VIE QUE LA FERMETURE A DESIGNEE qui est
	// nommee (`closureReport.closedLife`), pas « l'unique vie anonyme du slot ».
	nameClosedLives(rep.lives, rep.Owner, rep.Closures.closedLife, idx.ByXUID)
	return rep, crea, verif
}

// nameClosedLives pose l'identite d'une fermeture sur LA VIE QU'ELLE A DESIGNEE.
//
// `closed` vient des fermetures elles-memes (slot -> indice de vie ; -1 = deux vies designees,
// donc abstention). Une vie deja nommee par le fil des morts n'est jamais reecrite : la lecture
// prime sur la deduction, comme partout dans ce pont.
func nameClosedLives(lives []lifeSpan, after, closed map[uint32]int, xuidToIndex map[uint64]int) {
	if len(closed) == 0 {
		return
	}
	indexToXUID := indexToXUIDOf(xuidToIndex)
	for slot, life := range closed {
		if life < 0 || life >= len(lives) || lives[life].xuid != 0 {
			continue
		}
		pi, known := after[slot]
		if !known {
			continue
		}
		if x, ok := indexToXUID[pi]; ok {
			lives[life].xuid = x
			// LA FERMETURE NOMME, ELLE NE TERMINE PAS. Elle dit « un autre corps est reapparu,
			// donc celui-ci etait celui-la » — rien sur la facon dont la vie s'est terminee.
			// `cause` reste donc ce que la decoupe a etabli (fin du film, ou coupure), et c'est
			// exactement ce qui empeche de refabriquer une mort pour un survivant.
			lives[life].nomPar = NomParFermeture
		}
	}
}

// extendSlotXUID pose l'identite sur les slots que les fermetures ont attribues. Sans cela, un
// slot deduit porterait des tirs sans que le client puisse nommer son joueur — les deux tables
// diraient deux choses differentes du meme pont, ce que `ownersFromLives` interdit deja.
func extendSlotXUID(byXUID map[uint32]uint64, owner map[uint32]int,
	xuidToIndex map[uint64]int) map[uint32]uint64 {
	indexToXUID := indexToXUIDOf(xuidToIndex)
	out := make(map[uint32]uint64, len(owner))
	for s, x := range byXUID {
		out[s] = x
	}
	for s, pi := range owner {
		if _, ok := out[s]; ok {
			continue
		}
		if x, ok := indexToXUID[pi]; ok {
			out[s] = x
		}
	}
	return out
}

// indexToXUIDOf renverse la table identite -> index. Un helper plutot que deux boucles
// identiques a vingt lignes d'ecart : la troisieme copie derive.
func indexToXUIDOf(xuidToIndex map[uint64]int) map[int]uint64 {
	out := make(map[int]uint64, len(xuidToIndex))
	for x, i := range xuidToIndex {
		out[i] = x
	}
	return out
}

// xuidNumAt rend le joueur qui OCCUPE ce slot à cet instant : la vie qui couvre l'instant si elle
// est nommée, sinon le pont par slot. Zéro = ni l'une ni l'autre ne le nomme.
//
// POURQUOI L'INSTANT COMPTE (correctif du 2026-09-06, constat P1-7). `SlotXUID` est une identité
// UNIQUE PAR SLOT pour tout le match : `ownersFromLives` garde la PREMIÈRE vie nommée et jette
// les suivantes en collision, et `buildLifeSpans` trie par slot puis chronologiquement — c'est
// donc le PREMIER occupant, quel que soit l'instant demandé. Sur un slot de biped recyclé entre
// deux joueurs nommés (9 artefacts du parc portent `slotCollisions > 0`), tout lecteur qui
// interroge le pont sans son instant crédite le premier occupant.
//
// LE MOTIF EST CELUI DU DÉPÔT — « par vie d'abord, pont en repli » (cf. `tracksByXUID`) — et
// c'est ici qu'il vit pour tous ses lecteurs : la table par vie est déjà DANS cet objet.
func (r OwnerReport) xuidNumAt(slot uint32, tUS uint64) uint64 {
	t := int64(tUS)
	for _, l := range r.lives {
		if l.slot != slot || l.xuid == 0 || t < l.from || t > l.to {
			continue
		}
		return l.xuid
	}
	// LE REPLI PAR SLOT S'ABSTIENT SUR UN SLOT AMBIGU (2026-09-07). `SlotXUID` y garde le
	// PREMIER occupant nommé, par ordre des vies : le servir à un instant que sa vie ne couvre
	// pas reviendrait à publier un nom arbitraire, et c'est exactement ce que cette méthode
	// existe pour éviter. Sans vie couvrante ET sur un slot à plusieurs occupants, on se tait.
	if r.SlotAmbiguous[slot] {
		return 0
	}
	return r.SlotXUID[slot]
}

// NamingBridge rend le pont slot -> joueur DÉBARRASSÉ DES SLOTS AMBIGUS — celui que doit
// employer tout lecteur qui s'en sert pour NOMMER une piste.
//
// POURQUOI IL EXISTE (constat C2 de la revue VIES-R1, 2026-09-07). `SlotXUID` garde le PREMIER
// occupant nommé d'un slot que deux joueurs se partagent : c'est un choix par l'ORDRE DES VIES.
// `xuidAt` et `bridgeOfSlot` s'en abstiennent déjà, mais le helper partagé `xuidOfPublishedTrack`
// ne le pouvait pas — il ne reçoit qu'une map. Résultat mesuré sur `084a804d` slot 734 : la passe
// de nommage REFUSE (`contested = 1`) et le helper servait quand même `2535430265968559`, si bien
// que `samplesByXUID` indexait les positions de la piste contestée sous le premier occupant —
// une capture de zone pouvait être géolocalisée sur la trajectoire d'un AUTRE joueur.
//
// PLUTÔT QUE DE FAIRE DESCENDRE `SlotAmbiguous` DANS QUATRE CHAÎNES d'appel (les zones, les
// pistes de porteur de drapeau, les actions d'objectif, les morts neutres), on retire les slots
// ambigus À LA SOURCE : le lecteur ne peut plus oublier la garde, puisqu'il n'a plus de quoi
// l'enfreindre.
//
// `SlotXUID` RESTE INCHANGÉ pour ses autres consommateurs (ramassages, marques de portage, frags
// sous équipement actif) : leur exemption est explicite au cadrage de l'audit, et la modifier
// sortirait du périmètre de cette revue.
func (r OwnerReport) NamingBridge() map[uint32]uint64 {
	if len(r.SlotAmbiguous) == 0 {
		return r.SlotXUID
	}
	out := make(map[uint32]uint64, len(r.SlotXUID))
	for s, x := range r.SlotXUID {
		if !r.SlotAmbiguous[s] {
			out[s] = x
		}
	}
	return out
}
