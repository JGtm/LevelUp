package grammar

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

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
// SANS preuve, pour eprouver le principe des doutes sur de vrais octets. La lecture est un canal de
// la phase des images-cles ([canalDesEquipes]) qui parcourt les corps des records ti=9.
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
	c := &canalDesEquipes{rep: rep, arch: arch, ctx: fc.ContexteDeLecture(), entites: nouvelAccumulateurDEntites(),
		parIndex: map[int]map[int]int{}}
	distribuerLaDemande(fc, demandeDImagesCles{marche: &marche}, []Canal{c})
	c.rep.Entities = len(c.entites.entites)
	c.rep.EntityDivergences = c.entites.divergences()
	return publierEquipes(c.parIndex, &c.rep), c.rep, c.entites.publier()
}

// canalDesEquipes lit l'equipe de chaque joueur gere dans la phase des images-cles : il interprete
// le designateur d'equipe, i0 de ti=9, et la phase parcourt donc les corps de ti=9.
type canalDesEquipes struct {
	rep  TeamScanReport
	arch Archetype
	// ctx est le contexte de lecture du film sous lequel l'etat par defaut de ti=9 se relit (cadre,
	// controle de corruption) : celui du balayage, pris avant la phase.
	ctx      ContexteDeLecture
	entites  *accumulateurDEntites
	parIndex map[int]map[int]int // index de joueur -> designateur -> compte
}

func (*canalDesEquipes) Interets() []Interet {
	return []Interet{{Phase: PhaseImagesCles, TI: managedPlayerTypeIndex, Composant: teamDesignatorComponent}}
}

func (*canalDesEquipes) Clore(BilanDeMarche) {}

// ImageCle lit les records ti=9 d'UN paquet d'image-cle. Chaque refus est compte.
//
// L'IMAGE-CLE N'EST INSCRITE QUE SI ELLE EST PORTEUSE (au moins un record ti=9) : c'est le pas
// des presences, et une image-cle qui ne porte aucun occupant (le preambule) ne dit l'absence de
// personne.
//
// CE QU'ELLE NE PROUVE PAS SE NOTE (lot D-fix, 2026-09-24) : le slot dont l'en-tete EXACT de
// record ti=9 apparait dans le payload sans que la marche l'ait LU — record perdu par n'importe quel
// chemin de la marche, ou atteint mais illisible — est un occupant peut-etre present ; son absence a
// cette image-cle n'est pas prouvee ([PlayerEntityScan.AbsenceProuvee],
// `player_entities_entetes.go`).
func (c *canalDesEquipes) ImageCle(p *lecture.Paquet, _ *MarcheDistribuee) {
	rang := -1
	lus := map[int]bool{}
	for i := range p.Records {
		r := &p.Records[i]
		if int(r.TI) != managedPlayerTypeIndex {
			continue
		}
		if rang < 0 {
			rang = c.entites.ouvrirImageCle(p.TS)
			c.rep.Packets++
		}
		c.rep.Records++
		slot := int(r.Vie.Slot)
		idx, brut, ok := lireEquipeALEtendue(p, r, c.arch, c.ctx)
		switch {
		case !ok:
			c.rep.Unreached++
		case idx < 0 || idx >= playerTableSlots:
			c.rep.OutOfDomainIndex++
		case brut < 0 || brut > teamDesignatorRawMax:
			c.rep.OutOfDomainValue++
		default:
			c.rep.Read++
			lus[slot] = true
			c.entites.noter(rang, slot, idx, brut-1)
			noter(c.parIndex, idx, brut-1)
		}
	}
	if rang >= 0 {
		c.entites.douterDe(rang, lus, slotsDEntetesExacts(p.Payload))
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
// designateur. C'est la forme des instruments, qui marchent leurs records hors de la phase des
// images-cles ; la production lit le record que la phase a parcouru ([lireEquipeALEtendue]).
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
	return lireEquipeA(pay, recBit, tr.Comps[0].StartBit, ctx)
}

// lireEquipeALEtendue rend l'index de joueur et la valeur BRUTE du designateur du record ti=9 `r`
// du paquet d'image-cle `p`, dont la phase a parcouru le corps : i0 est la premiere occurrence du
// record, l'etendue que la marche lui a donnee. Memes derivations, meme concordance que
// [lireEquipeDuRecord].
func lireEquipeALEtendue(p *lecture.Paquet, r *lecture.Record, arch Archetype, ctx ContexteDeLecture) (idx, brut int, ok bool) {
	if r.Comps[1] == r.Comps[0] {
		return 0, 0, false
	}
	c0 := p.Comps[r.Comps[0]]
	if int(c0.Index) >= len(arch.Components) || arch.Components[c0.Index] != teamDesignatorComponent ||
		c0.Etat == lecture.EtatInfranchissable {
		return 0, 0, false
	}
	return lireEquipeA(p.Payload, int(r.Debut), int(c0.Debut), ctx)
}

// lireEquipeA relit l'etat par defaut du record ti=9 de premier bit `recBit` pour son index de
// joueur, verifie que son composant i0 commence bien en `i0`, et y lit le designateur BRUT.
func lireEquipeA(pay []byte, recBit, i0 int, ctx ContexteDeLecture) (idx, brut int, ok bool) {
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
	return idx, int(source.BitsBourres(pay, i0, teamDesignatorBits)), true
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
