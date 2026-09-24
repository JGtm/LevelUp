package grammar

// player_teams.go — L'EQUIPE DE CHAQUE JOUEUR, LUE DANS LE FILM (lot 1.7 du PLAN_DECODEUR_FILM).
//
// # OU ELLE EST, ET PAR QUELLES DEUX CHAINES ON LE SAIT
//
// Elle est dans la trame d'etat (paquets de type 2), portee par le composant
// `managed-player-team-designator-component` — le composant i0 de l'archetype ti=9
// (« managed-player ») — sur QUATRE bits. Source :
// `.ai/V7.5/film_re/NOTE_EQUIPE_FILM_2026-09-12.md`, deux chaines SANS etape commune :
//
//	L'EXECUTABLE : descripteur `0x143d08ad0` (nom en +0x08, ecrivain en +0x18, lecteur en
//	  +0x30) ; le lecteur `0x140f581e8` n'appelle que `FUN_1407ef804`, dont le desassemblage
//	  donne la largeur (`ADD dword ptr [RCX+0x2c],0x4` = 4 bits) ET la convention de valeur
//	  (`SHR R9,0x3c ; DEC R9B` = la valeur du jeu vaut le BRUT MOINS UN). L'enumeration de
//	  script `mp_team_designator` compte neuf noms (`First`..`Eighth`, `Neutral`), ce qui ferme
//	  avec le test de validite du lecteur de la composante globale (`valeur + 1 < 10`).
//	LES OCTETS : 16 films sur 18 en accord EXACT terme a terme avec `match_participants.team_id`,
//	  160 slots sur 176, dont DEUX Grandes batailles a 24/24 ; 0 faux positif sur 7 292 positions
//	  de record confrontees, 0 touche sur 576 decalages voisins.
//
// # LA POSITION SE DERIVE, ELLE NE SE CABLE PAS
//
// La note publie « 186 bits du debut du record », et ce nombre N'APPARAIT NULLE PART ICI : il
// est la SOMME d'une grammaire qui se rejoue. `WalkKeyframeFullState` (lot 1.4) lit l'en-tete
// par entite de 108 bits, le mot de taille `n1`, l'etat par defaut de l'archetype, puis `n2`,
// et OUVRE la boucle de composants. Le composant i0 commence donc a
// `108 + 32 + etat(ti=9) + 32`, et l'etat de ti=9 (`V ; R(6) ; R(6) ; R(1)`) vaut 14 bits quand
// la porte de version est fermee : 186. Un build qui changerait l'une des trois largeurs
// deplacerait le champ, et ce lecteur suivrait — un `186` en dur ne suivrait pas.
//
// La MEME regle vaut pour le NOM du composant : i0 est relu dans le registre DU FILM par le
// walk de production, jamais dans une table du depot. Un film dont ti=9 i0 porte un autre nom
// n'est pas lu, il est COMPTE.
//
// # L'APPARIEMENT ENTITE -> JOUEUR EST ECRIT PAR LE FILM (mesure du 2026-09-14)
//
// La note pariait sur un appariement ORDINAL (« la i-eme entite ti=9 est le i-eme siege de
// `chunk_00` »), et prevenait qu'il fallait indexer par slot des que le roster bouge. La mesure
// du lot 1.7 ferme la question autrement, et mieux : l'etat par defaut de ti=9 porte l'INDEX DE
// JOUEUR de l'entite dans son premier `R(6)` (cf. [readManagedPlayerDefaultState]). Il n'y a
// donc AUCUN appariement a faire — « l'index c'est l'index » (decision utilisateur du
// 2026-09-07). L'ordinal reste le CONTROLE, verifie sur le corpus par les tests de ce paquet.
//
// C'est ce qui rend les REMPLACANTS lisibles : un joueur arrive en cours de partie n'a pas de
// siege dans `chunk_00` (table du DEBUT du film, lot 1.6) mais son entite ti=9 porte son index
// et son equipe comme les autres. Mesure : 34 arrivees sur 18 films, toutes a designateur
// STABLE ; deux d'entre elles REPRENNENT l'index d'un partant (`11de8353` index 23,
// `51101d1d` index 6).
//
// # CE QUI SE REFUSE, ET CE QUI SE COMPTE
//
// Rien ne s'invente : un index hors du domaine de la table (0..31), une valeur brute hors du
// domaine de l'enumeration (0..9), un i0 qui n'est pas le designateur, une marche qui n'atteint
// pas i0 — chacun est un COMPTEUR de [TeamScanReport], jamais une valeur. Un index dont deux
// lectures ne s'accordent pas n'est pas publie : la divergence se compte et le relecteur la lit.
//
// HORS LIGNE — jamais depuis un chemin de requete. Aucune ecriture, aucun schema.

// managedPlayerTypeIndex est l'archetype « managed-player » : une entite par joueur.
const managedPlayerTypeIndex = 9

// teamDesignatorComponent est le nom que le registre du film donne au composant i0 de ti=9.
const teamDesignatorComponent = "managed-player-team-designator-component"

// teamDesignatorBits est la largeur du champ, lue chez `FUN_1407ef804`, pas supposee.
const teamDesignatorBits = 4

// teamDesignatorRawMax est la borne du domaine BRUT : `mp_team_designator` compte neuf
// designateurs (0..8), ecrits VALEUR PLUS UN, plus le zero d'« aucune equipe ».
const teamDesignatorRawMax = 9

// TeamNone est la valeur que le jeu ecrit pour « aucune equipe » (brut 0, donc -1 apres le
// `DEC R9B` du lecteur). C'est la valeur des parties SANS equipes (FFA), et elle est une
// LECTURE : le film dit « aucune equipe », il ne se tait pas.
const TeamNone = -1

// TeamScanReport est ce que la lecture a vu, et ce qu'elle a refuse.
//
// AUCUN DE CES COMPTEURS N'EST DECORATIF : ils sont la seule facon de distinguer « le film ne
// porte pas d'equipe » de « on n'a pas su la lire », et la couverture de l'artefact les publie.
type TeamScanReport struct {
	// ArchetypeAbsent : le registre du film ne porte pas ti=9 (bobine partielle, autre jeu).
	ArchetypeAbsent bool
	// ComponentMismatch : ti=9 existe mais son i0 n'est PAS le designateur. Le film n'est pas lu
	// « au composant voisin » : la lecture s'arrete.
	ComponentMismatch bool
	// Component est le nom REEL de l'i0 de ti=9 dans le registre du film, pour que le refus se
	// diagnostique sans rouvrir le film.
	Component string
	// Packets / Records : les paquets d'image-cle porteurs de ti=9, et les records lus.
	Packets, Records int
	// Read : les records dont le designateur a ete lu (marche arrivee a i0, domaines tenus).
	Read int
	// Unreached : les records dont la marche n'a pas atteint i0 (desynchronisation en tete,
	// archetype inconnu, record tronque).
	Unreached int
	// OutOfDomainIndex / OutOfDomainValue : les records dont l'index de joueur sort de la table
	// de 32, ou dont la valeur brute sort de `0..9`. Mesure du 2026-09-14 : UN seul sur les
	// 18 films du corpus (`111fa685`, index 59, un unique paquet, valeur brute 0).
	OutOfDomainIndex, OutOfDomainValue int
	// Entities : les entites ti=9 distinctes (par slot de replication) rencontrees.
	Entities int
	// EntityDivergences : les entites dont le designateur CHANGE au cours du film. La note en
	// mesure ZERO sur 22 films — une entite ne change pas d'equipe, c'est l'appariement qui
	// bougeait.
	EntityDivergences int
	// IndexDivergences : les index de joueur que DEUX lectures nomment differemment (deux
	// entites successives sur le meme index, ou une entite instable). Un index divergent n'est
	// PAS publie.
	IndexDivergences int
	// Indices : les index de joueur publies, c'est-a-dire la taille de la table rendue.
	Indices int
	// NoTeam : les index publies a « aucune equipe » (FFA). Ce n'est pas un silence.
	NoTeam int
}

// Lu dit si la lecture a produit quelque chose d'exploitable.
func (r TeamScanReport) Lu() bool { return !r.ArchetypeAbsent && !r.ComponentMismatch && r.Read > 0 }

// ScanPlayerTeams lit l'equipe de chaque joueur dans la trame d'etat du film.
//
// Rend la table `index de joueur -> DESIGNATEUR` (la valeur du jeu : [TeamNone] pour « aucune
// equipe », 0..8 sinon), le rapport de lecture, et les ENTITES (cf. player_entities.go). Un index
// absent de la table est un SILENCE — le film ne l'a pas dit ici — et jamais un « pas d'equipe » :
// les deux se distinguent.
//
// UNE SEULE PASSE POUR LES DEUX VUES (lot M2.1, 2026-09-23) : la table par index et les entites
// sortent des MEMES lectures. La table n'est plus que le CONTROLE — un index repris par deux
// occupants d'equipes differentes y diverge et ne se publie pas, alors que chaque entite garde
// le sien (sonde P4, `b1ad85eb` : trois bots d'index 8, deux equipes).
//
// LA MARCHE D IMAGE-CLE EST CELLE DU FILM ([FilmContext.MarcheDImageCle], lot D-fix) : elle ne perd
// plus le joueur gere qu une fausse ancre elue effacait, et ce qu elle ecarte encore par repli se
// note en DOUTE (cf. player_entities.go).
//
// HORS LIGNE (parcourt les chunks de donnees du film) ; un appel par cuisson.
func ScanPlayerTeams(fc *FilmContext) (map[int]int, TeamScanReport, PlayerEntityScan) {
	return scanPlayerTeamsAvec(fc, fc.MarcheDImageCle())
}

// scanPlayerTeamsAvec est [ScanPlayerTeams] sous une marche donnee : les tests y rejouent la marche
// SANS preuve, pour eprouver le principe des doutes sur de vrais octets.
func scanPlayerTeamsAvec(fc *FilmContext, marche MarcheDImageCle) (map[int]int, TeamScanReport, PlayerEntityScan) {
	var rep TeamScanReport
	reg, err := fc.Registry()
	if err != nil {
		rep.ArchetypeAbsent = true
		return nil, rep, PlayerEntityScan{}
	}
	arch, ok := reg.Archetype(managedPlayerTypeIndex)
	if !ok || len(arch.Components) == 0 {
		rep.ArchetypeAbsent = true
		return nil, rep, PlayerEntityScan{}
	}
	rep.Component = arch.Components[0]
	if rep.Component != teamDesignatorComponent {
		rep.ComponentMismatch = true
		return nil, rep, PlayerEntityScan{}
	}
	entites := nouvelAccumulateurDEntites()
	parIndex := map[int]map[int]int{} // index de joueur -> designateur -> compte
	for _, c := range fc.ChunkNumbers() {
		raw, paquets, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range paquets {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			scanPaquetEquipes(pk.Payload(raw), pk.TimestampUS, reg, &rep, lecturesDEquipe{
				entites: entites, parIndex: parIndex, ctx: fc.ContexteDeLecture(), marche: marche})
		}
	}
	rep.Entities = len(entites.entites)
	rep.EntityDivergences = entites.divergences()
	return publierEquipes(parIndex, &rep), rep, entites.publier()
}

// lecturesDEquipe porte ce que chaque paquet alimente : les entites, la table de controle par
// index, le contexte de lecture et la marche d'ancres du film. Une structure plutot que quatre
// parametres de plus : le depot borne a cinq, et les quatre voyagent toujours ensemble.
type lecturesDEquipe struct {
	entites  *accumulateurDEntites
	parIndex map[int]map[int]int
	ctx      ContexteDeLecture
	marche   MarcheDImageCle
}

// scanPaquetEquipes lit les records ti=9 d'UN payload d'image-cle. Chaque refus est compte.
//
// L'IMAGE-CLE N'EST INSCRITE QUE SI ELLE EST PORTEUSE (au moins un record ti=9) : c'est le pas
// des presences, et une image-cle qui ne porte aucun occupant (le preambule) ne dit l'absence de
// personne.
//
// CE QU'ELLE NE PROUVE PAS SE NOTE (lot D-fix, 2026-09-24) : le slot d'un record ti=9 que sa
// marche a ecarte par REPLI ([MarcheDePayload.Ecartes]), ou qu'elle a atteint sans pouvoir le lire,
// est un occupant peut-etre present — son absence a cette image-cle n'est pas prouvee
// ([PlayerEntityScan.AbsenceProuvee]).
func scanPaquetEquipes(pay []byte, ts uint64, reg *Registry, rep *TeamScanReport, l lecturesDEquipe) {
	rang := -1
	mp := l.marche.Marcher(pay)
	lus := map[int]bool{}
	var illisibles []int
	for _, b := range keyframeBornesDe(mp.Records) {
		if b.TI != managedPlayerTypeIndex {
			continue
		}
		if rang < 0 {
			rang = l.entites.ouvrirImageCle(ts)
			rep.Packets++
		}
		rep.Records++
		idx, brut, ok := lireEquipeDuRecord(pay, b.Bit, reg, l.ctx)
		switch {
		case !ok:
			rep.Unreached++
			illisibles = append(illisibles, b.Slot)
		case idx < 0 || idx >= playerTableSlots:
			rep.OutOfDomainIndex++
			illisibles = append(illisibles, b.Slot)
		case brut < 0 || brut > teamDesignatorRawMax:
			rep.OutOfDomainValue++
			illisibles = append(illisibles, b.Slot)
		default:
			rep.Read++
			lus[b.Slot] = true
			l.entites.noter(rang, b.Slot, idx, brut-1)
			noter(l.parIndex, idx, brut-1)
		}
	}
	if rang >= 0 {
		l.entites.douterDe(rang, lus, illisibles, mp.Ecartes)
	}
}

// noter incremente le compte d'une valeur pour une cle.
func noter(m map[int]map[int]int, cle, valeur int) {
	if m[cle] == nil {
		m[cle] = map[int]int{}
	}
	m[cle][valeur]++
}

// lireEquipeDuRecord rejoue UN record ti=9 et rend l'index de joueur et la valeur BRUTE du
// designateur.
//
// DEUX DERIVATIONS INDEPENDANTES, ET LEUR CONCORDANCE EST VERIFIEE. L'index sort de l'etat par
// defaut rejoue a `en-tete + n1` ; le designateur sort de la boucle de composants de
// PRODUCTION, qui nomme i0 depuis le registre du film. Si la position du composant ne tombe pas
// la ou la grammaire la place, la lecture est REFUSEE : c'est le signe qu'une largeur a bouge.
func lireEquipeDuRecord(pay []byte, recBit int, reg *Registry, ctx ContexteDeLecture) (idx, brut int, ok bool) {
	tr := WalkKeyframeFullState(pay, recBit, reg, ctx)
	if len(tr.Comps) == 0 || tr.Comps[0].Name != teamDesignatorComponent || !tr.Comps[0].Ported {
		return 0, 0, false
	}
	i0 := tr.Comps[0].StartBit
	br := LecteurSur(pay)
	br.PoserContexte(ctx)
	// LE CADRE VIENT DU PROFIL QUE LE LECTEUR PORTE (lot 2.2.c) : cette lecture REJOUE le cadre
	// d'`walkKeyframeFullState` pour retrouver le premier composant, et les deux doivent donc
	// tenir leur en-tete et leur mot de taille du MEME endroit — sinon la garde ci-dessous
	// refuserait des records valides le jour ou l'un des deux bouge.
	cadre := br.cadre()
	br.SetBitPos(recBit + cadre.EnTeteBits + cadre.MotDeTailleBits)
	idx = readManagedPlayerDefaultState(br)
	attendu := br.BitPos() + cadre.MotDeTailleBits
	if ctx.Profil.Grammaire.ControleDeCorruption {
		attendu += cadre.MotDeTailleBits
	}
	if attendu != i0 || i0+teamDesignatorBits > len(pay)*8 {
		return 0, 0, false
	}
	return idx, int(kfReadBits(pay, i0, teamDesignatorBits)), true
}

// publierEquipes ne garde que les index dont TOUTES les lectures s'accordent. Un index
// divergent se compte et ne se publie pas : entre deux equipes pour un meme joueur, il n'y a
// rien a choisir.
func publierEquipes(parIndex map[int]map[int]int, rep *TeamScanReport) map[int]int {
	if len(parIndex) == 0 {
		return nil
	}
	out := make(map[int]int, len(parIndex))
	for idx, vus := range parIndex {
		if len(vus) != 1 {
			rep.IndexDivergences++
			continue
		}
		for v := range vus {
			out[idx] = v
			if v == TeamNone {
				rep.NoTeam++
			}
		}
	}
	rep.Indices = len(out)
	if len(out) == 0 {
		return nil
	}
	return out
}
