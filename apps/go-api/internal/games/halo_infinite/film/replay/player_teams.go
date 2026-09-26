package replay

// player_teams.go — L'EQUIPE PUBLIEE VIENT DU FILM, ET DE LUI SEUL (lot 1.7.2).
//
// # LA REGLE, ET QUI L'A TRANCHEE
//
// Decision utilisateur du 2026-09-13 (V4 du PLAN_DECODEUR_FILM) : « si le decodeur est fiable,
// pas besoin du repli ». L'equipe d'un joueur vient du DESIGNATEUR que le film ecrit
// ([grammar.ScanPlayerTeams]) ; la base n'en pose AUCUNE. Elle entre ici comme CONTROLE, et
// uniquement comme tel : `coverage.teams.{accord, contradiction, silence}` disent ce qu'elle
// aurait dit, sans jamais le publier. Une contradiction ne se corrige pas en silence — le film
// fait foi, l'ecart se compte, et le relecteur le lit.
//
// # CE QUE CELA REPARE, ET IL EST MESURE
//
// Avant ce lot, `Track.Team` valait -1 sur toutes les vies de tous les artefacts, et l'equipe
// d'un porteur de drapeau venait de `match_participants.team_id` par l'appelant. Deux
// consequences, toutes deux chiffrees :
//
//	LE REJEU HORS LIGNE N'AVAIT AUCUNE EQUIPE. Le constructeur n'ouvre aucune base ; sans lignes
//	  de match la table etait nil, l'invariant du drapeau (« on ne porte jamais son propre
//	  drapeau ») se taisait, et `coverage.flagCarries.carrierTeamUnknown` comptait exactement
//	  cette perte.
//	LES ARRIVANTS EN COURS DE PARTIE N'AVAIENT PAS DE SIEGE. La table des joueurs de `chunk_00`
//	  est celle du DEBUT du film (lot 1.6) ; un remplacant n'y figure pas. Son entite ti=9, elle,
//	  porte son index ET son equipe comme les autres — 34 arrivees mesurees sur 18 films le
//	  2026-09-14, toutes a designateur stable.
//
// # CE QUE LA VALEUR VEUT DIRE
//
// Le document publie le DESIGNATEUR du jeu, pas la valeur brute du flux : `0..8` sont les huit
// camps de l'enumeration `mp_team_designator`, et `-1` est « AUCUNE EQUIPE » — ce que le moteur
// ecrit sur un mode sans camps (FFA), mesure sur les deux films FFA du cache. Une vie que le
// film ne nomme pas garde ce meme `-1`, et c'est `coverage.teams.unread` qui dit combien : les
// deux etats se distinguent par la couverture, pas par une seconde sentinelle.
//
// # L'EQUIPE EST CELLE DE L'OCCUPANT, PAS CELLE DE L'INDEX (lot M2.3, 2026-09-23)
//
// Le lot 1.7 lisait l'equipe PAR INDEX de joueur. Sonde P4 (`b1ad85eb`) : trois bots de DEUX
// equipes se relaient sur l'index 8 — l'index diverge, et aucun des trois n'avait d'equipe. Depuis
// ce lot, chaque entree du roster prend le designateur de SES entites `ti=9` (occupants.go) ; la
// table par index n'est plus que le CONTROLE, et le repli des entrees qu'aucune entite ne porte.
// Les vies suivent l'entree qui les porte (xuid, ou nom de bot), avant le pont par index.

import (
	"log/slog"
	"strconv"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// TeamCoverage est ce que la lecture de l'equipe a couvert, et ce que la base en dit.
//
// ELLE PUBLIE LES DEUX MOITIES SEPAREMENT, et c'est le point : `film` dit ce que l'artefact
// TIENT du film, `accord` / `contradiction` / `silence` disent ce qu'une source EXTERIEURE en
// pense. Confondre les deux rendrait invisible le jour ou le film se tromperait.
type TeamCoverage struct {
	// Read dit si le balayage a produit une lecture. Faux = le film n'a pas ete lu, et
	// `Refusal` dit pourquoi.
	Read bool `json:"read"`
	// Refusal nomme la cause d'une couverture sans lecture : `archetype_absent` (le registre ne
	// porte pas ti=9 — bobine partielle), `composant_inattendu` (l'i0 de ti=9 porte un autre nom
	// que le designateur : une grammaire a bouge, et on ne lit PAS le composant voisin), ou
	// `non_balaye` (l'appelant assemble depuis des positions deja decodees et n'a fourni aucune
	// lecture d'equipe).
	Refusal string `json:"refusal,omitempty"`
	// Records est le denominateur : les records ti=9 rencontres dans la trame.
	Records int `json:"records"`
	// Rejected : ceux qu'un domaine (index hors de la table de 32, valeur brute hors de 0..9)
	// ou une marche inachevee a ecartes. Un rejet se compte, il ne se tait pas.
	Rejected int `json:"rejected"`
	// Divergences : les entites ou les index dont deux lectures ne s'accordent pas. Un index
	// divergent n'est PAS publie — entre deux equipes pour un meme joueur, il n'y a rien a
	// choisir.
	Divergences int `json:"divergences"`
	// Film : les joueurs du roster dont le FILM donne l'equipe.
	Film int `json:"film"`
	// NoTeam : parmi eux, ceux a qui il donne « aucune equipe » (FFA). Sous-ensemble de `Film`,
	// parce que c'est une LECTURE et non un silence.
	NoTeam int `json:"noTeam"`
	// Unread : les joueurs du roster que le film ne nomme pas. C'est le SILENCE du film, a ne
	// pas confondre avec celui du controle.
	Unread int `json:"unread"`
	// Accord / Contradiction / Silence : LE CONTROLE, c'est-a-dire la table de la base.
	// `accord` = elle dit la meme equipe ; `contradiction` = elle en dit une autre (le film est
	// publie quand meme) ; `silence` = elle ne porte pas ce joueur.
	Accord        int `json:"accord"`
	Contradiction int `json:"contradiction"`
	Silence       int `json:"silence"`
	// Tracks / TracksNamed : les vies publiees, et celles que le film NOMME — « aucune equipe »
	// comprise, parce que c'est une lecture. Le rapport des deux est ce qu'un lecteur voit.
	Tracks      int `json:"tracks"`
	TracksNamed int `json:"tracksNamed"`
	// TracksSlotAmbiguous : les vies sans xuid dont le SLOT a porte deux joueurs nommes
	// d'equipes differentes, et sur lesquelles le pont s'est ABSTENU.
	//
	// IL DIT POURQUOI UNE VIE N'EST PAS NOMMEE, ET C'EST LE CORRECTIF DE LA REVUE DE JALON M1
	// (lentille L4). `equipeDuSlot` lisait `IndexParSlot()`, qui garde le PREMIER occupant nomme
	// d'un slot recycle (`ownersFromLives`) : sur un slot que deux joueurs d'equipes differentes
	// se partagent, la vie du SECOND recevait l'equipe du PREMIER, et `tracksNamed` l'affirmait
	// nommee. C'est la meme abstention que [IdentityRegistry.PontDeSlot], et pour la meme raison
	// — le pont aplati a d'ailleurs ete retire au lot 6.1 exactement pour ce defaut.
	TracksSlotAmbiguous int `json:"tracksSlotAmbiguous,omitempty"`
}

// teamRefusalArchetype / teamRefusalComponent / teamRefusalNotScanned nomment les trois causes
// d'une couverture sans lecture. LA TROISIEME EXISTE PARCE QU'UN REFUS SE NOMME (D14) : un
// appelant qui assemble depuis des positions deja decodees (`BuildFromPositions`, le chemin du
// collecteur de kills) ne balaye AUCUN film, et sa couverture doit le dire au lieu de se lire
// comme un balayage qui n'a rien trouve.
const (
	teamRefusalArchetype  = "archetype_absent"
	teamRefusalComponent  = "composant_inattendu"
	teamRefusalNotScanned = "non_balaye"
)

// teamPublication porte ce qu'il faut pour poser l'equipe partout : la table du film par index,
// sa projection par xuid, le pont slot -> index (les bots n'ont pas de xuid), et le CONTROLE.
type teamPublication struct {
	byIndex map[int]int
	byXUID  map[uint64]int
	bySlot  map[uint32]int
	// parIdentite : l'equipe PAR ENTREE du roster (lot M2.3), cle de roster -> designateur. Elle
	// passe devant le pont par index : un bot d'un index partage y a son equipe.
	parIdentite map[string]int
	// slotsAmbigus : les slots que deux joueurs nommes se sont partages. `bySlot` y garde le
	// PREMIER occupant, donc le pont s'y tait (cf. [TeamCoverage.TracksSlotAmbiguous]).
	slotsAmbigus map[uint32]bool
	rep          grammar.TeamScanReport
	controle     map[string]int
}

// newTeamPublication projette la table du film sur les xuids que le registre d'identite connait.
//
// LA PROJECTION PASSE PAR LA TABLE D'INDEX EFFECTIVE DU REGISTRE (lot 1.6) : c'est la seule
// table du film, et deux tables du meme film divergeraient.
func newTeamPublication(reg IdentityRegistry, byIndex map[int]int, rep grammar.TeamScanReport,
	controle map[string]int) teamPublication {
	p := teamPublication{byIndex: byIndex, bySlot: reg.IndexParSlot(),
		slotsAmbigus: reg.SlotsAmbigus(), rep: rep, controle: controle}
	if len(byIndex) == 0 {
		return p
	}
	table := reg.TableDIndex()
	p.byXUID = make(map[uint64]int, len(table.ByXUID))
	for x, idx := range table.ByXUID {
		if t, ok := byIndex[idx]; ok {
			p.byXUID[x] = t
		}
	}
	return p
}

// equipeDuXUID rend l'equipe d'un xuid numerique.
func (p teamPublication) equipeDuXUID(x uint64) (int, bool) {
	t, ok := p.byXUID[x]
	return t, ok
}

// equipeDuSlot rend l'equipe de l'occupant d'un slot de bipede, par le pont slot -> index. C'est
// la seule voie pour un BOT : il n'a pas de xuid.
//
// ELLE SE TAIT SUR UN SLOT AMBIGU, et c'est la meme abstention que [IdentityRegistry.PontDeSlot].
// `IndexParSlot` garde le PREMIER occupant nomme d'un slot recycle (`ownersFromLives` compte les
// suivants en collision et ne tranche pas) : servir ce pont sur un slot que deux joueurs
// d'equipes differentes se partagent publie l'equipe du premier sur la vie du second. Le second
// retour distingue les deux silences — `false, false` = le pont ne connait pas ce slot,
// `false, true` = il le connait mais deux joueurs s'en disputent l'identite.
func (p teamPublication) equipeDuSlot(slot uint32) (equipe int, lue, ambigu bool) {
	if p.slotsAmbigus[slot] {
		return 0, false, true
	}
	idx, ok := p.bySlot[slot]
	if !ok {
		return 0, false, false
	}
	t, ok := p.byIndex[idx]
	return t, ok, false
}

// poserEquipesParEntree pose sur le roster l'equipe PAR ENTREE que la liaison aux entites a lue
// (occupants.go), et la retient pour les vies et le drapeau. Sans entite lue, la liaison rend
// l'equipe par index : rien ne change.
func (p *teamPublication) poserEquipesParEntree(roster []RosterEntry, occ occupants) {
	p.parIdentite = map[string]int{}
	for i := range roster {
		roster[i].Team = occ.parEntree[i].equipe
		t := roster[i].Team
		if t == nil {
			continue
		}
		if cle := cleDeRoster(roster[i]); cle != "" {
			p.parIdentite[cle] = *t
		}
		if x, err := strconv.ParseUint(roster[i].XUID, 10, 64); err == nil && occ.balaye {
			if p.byXUID == nil {
				p.byXUID = map[uint64]int{}
			}
			p.byXUID[x] = *t
		}
	}
}

// poserSurLesTraces pose l'equipe du film sur chaque vie publiee et rend les deux comptes.
//
// L'ORDRE EST FIXE ET IL N'EST PAS ARBITRAIRE : l'equipe de l'ENTREE qui porte la vie d'abord (lot
// M2.3 — xuid ou nom de bot), le xuid par index ensuite (le lien direct du lot 1.6), le pont
// slot -> index enfin. Une vie que rien ne nomme garde `-1`, et le compte le dit. Le troisieme
// retour isole les vies que le pont REFUSE de nommer parce que leur slot a porte deux joueurs
// (cf. [teamPublication.equipeDuSlot]) : sans lui, une abstention se lirait comme un film muet.
func (p teamPublication) poserSurLesTraces(tracks []Track) (total, nommees, slotAmbigu int) {
	for i := range tracks {
		total++
		if t, ok := p.parIdentite[cleDePiste(tracks[i])]; ok {
			tracks[i].Team, nommees = t, nommees+1
			continue
		}
		if x, err := strconv.ParseUint(tracks[i].XUID, 10, 64); err == nil {
			if t, ok := p.equipeDuXUID(x); ok {
				tracks[i].Team, nommees = t, nommees+1
				continue
			}
		}
		switch t, lue, ambigu := p.equipeDuSlot(tracks[i].Slot); {
		case lue:
			tracks[i].Team, nommees = t, nommees+1
		case ambigu:
			slotAmbigu++
		}
	}
	return total, nommees, slotAmbigu
}

// equipeDuRoster rend l'equipe d'une entree de roster, par son index de joueur. NIL quand le
// film ne nomme pas cet index : l'absence et « aucune equipe » ne sont pas la meme chose, et
// c'est le pointeur qui les separe (cf. [RosterEntry.Team]).
func (p teamPublication) equipeDuRoster(e RosterEntry) *int {
	t, ok := p.byIndex[e.FilmIndex]
	if !ok {
		return nil
	}
	return &t
}

// couverture rend ce que la lecture a couvert, controle compris.
func (p teamPublication) couverture(vies, viesNommees, viesSlotAmbigu int,
	entrees []RosterEntry) TeamCoverage {
	cov := TeamCoverage{
		Read:        p.rep.Lu(),
		Records:     p.rep.Records,
		Rejected:    p.rep.Unreached + p.rep.OutOfDomainIndex + p.rep.OutOfDomainValue,
		Divergences: p.rep.EntityDivergences + p.rep.IndexDivergences,
	}
	switch {
	case p.rep.ArchetypeAbsent:
		cov.Refusal = teamRefusalArchetype
	case p.rep.ComponentMismatch:
		cov.Refusal = teamRefusalComponent
	case p.rep.Component == "":
		// Aucun balayage n'a tourne : `ScanPlayerTeams` pose TOUJOURS `Component` des qu'il lit
		// le registre, meme quand il ne trouve aucun record.
		cov.Refusal = teamRefusalNotScanned
	}
	// LE DENOMINATEUR EST L'EQUIPE PAR ENTREE (lot M2.3) : celle que le roster publie.
	for _, e := range entrees {
		if e.Team == nil {
			cov.Unread++
			continue
		}
		cov.Film++
		if *e.Team == grammar.TeamNone {
			cov.NoTeam++
		}
		p.controler(&cov, e, *e.Team)
	}
	cov.Tracks, cov.TracksNamed, cov.TracksSlotAmbiguous = vies, viesNommees, viesSlotAmbigu
	return cov
}

// controler confronte une equipe LUE a ce que la base en dit. Elle ne change rien : elle compte.
func (p teamPublication) controler(cov *TeamCoverage, e RosterEntry, lue int) {
	if e.XUID == "" {
		return // un bot n'a pas de ligne de participant joignable par xuid
	}
	base, connu := p.controle[e.XUID]
	switch {
	case !connu:
		cov.Silence++
	case base == lue:
		cov.Accord++
	default:
		cov.Contradiction++
	}
}

// tableDesEquipesPourLesDrapeaux rend la table `xuid -> equipe` que le calque du drapeau
// consomme, construite du FILM SEUL. Les joueurs a « aucune equipe » n'y entrent pas :
// l'invariant « jamais son propre drapeau » ne se refuse que sur une equipe LUE et REELLE.
func (p teamPublication) tableDesEquipesPourLesDrapeaux() map[string]int {
	if len(p.byXUID) == 0 {
		return nil
	}
	out := make(map[string]int, len(p.byXUID))
	for x, t := range p.byXUID {
		if t == grammar.TeamNone {
			continue
		}
		out[strconv.FormatUint(x, 10)] = t
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// logTeamCoverage journalise ce que la lecture a couvert et ce que le controle en dit. Une
// contradiction et un refus se DISENT, avant toute degradation (regle n° 3 du depot).
func logTeamCoverage(matchID string, cov TeamCoverage) {
	slog.Info("rejeu : equipes lues dans le film", "match_id", matchID, "lue", cov.Read,
		"refus", cov.Refusal, "records", cov.Records, "rejetes", cov.Rejected,
		"divergences", cov.Divergences, "film", cov.Film, "sansEquipe", cov.NoTeam,
		"nonLus", cov.Unread, "accord", cov.Accord, "contradiction", cov.Contradiction,
		"silence", cov.Silence, "vies", cov.Tracks, "viesAvecEquipe", cov.TracksNamed,
		"viesSlotAmbigu", cov.TracksSlotAmbiguous)
	if cov.TracksSlotAmbiguous > 0 {
		slog.Warn("rejeu : le pont slot -> index s'ABSTIENT sur des vies dont le slot a porte "+
			"deux joueurs nommes — leur equipe reste inconnue plutot qu'empruntee au premier "+
			"occupant", "match_id", matchID, "vies", cov.TracksSlotAmbiguous)
	}
	if cov.Contradiction > 0 {
		slog.Warn("rejeu : la base CONTREDIT le film sur l'equipe de joueurs — le film fait foi, "+
			"l'ecart est compte", "match_id", matchID, "contradictions", cov.Contradiction,
			"accords", cov.Accord)
	}
	if !cov.Read {
		slog.Warn("rejeu : equipes NON LUES dans le film — aucune vie ne portera d'equipe",
			"match_id", matchID, "refus", cov.Refusal)
	}
}
