package replay

// player_teams.go — L'EQUIPE PUBLIEE VIENT DU FILM, ET DE LUI SEUL (lot 1.7.2).
//
// # LA REGLE, ET QUI L'A TRANCHEE
//
// Decision utilisateur du 2026-09-13 (V4 du PLAN_DECODEUR_FILM) : « si le decodeur est fiable,
// pas besoin du repli ». L'equipe d'un joueur vient du DESIGNATEUR que le film ecrit
// ([filmdec.ScanPlayerTeams]) ; la base n'en pose AUCUNE. Elle entre ici comme CONTROLE, et
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
// # CE QUI NE CHANGE PAS
//
// Aucune regle d'affichage (§1.2 du plan) : le web colore les joueurs par `team_side` de la
// feuille de match (`rosterLogic.ts`), pas par l'artefact. Ce lot publie une DONNEE.

import (
	"log/slog"
	"strconv"

	"levelup/go-api/internal/games/halo_infinite/film/filmdec"
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
	byIndex  map[int]int
	byXUID   map[uint64]int
	bySlot   map[uint32]int
	rep      filmdec.TeamScanReport
	controle map[string]int
}

// newTeamPublication projette la table du film sur les xuids que le registre d'identite connait.
//
// LA PROJECTION PASSE PAR LA TABLE D'INDEX EFFECTIVE DU REGISTRE (lot 1.6) : c'est la seule
// table du film, et deux tables du meme film divergeraient.
func newTeamPublication(reg IdentityRegistry, byIndex map[int]int, rep filmdec.TeamScanReport,
	controle map[string]int) teamPublication {
	p := teamPublication{byIndex: byIndex, bySlot: reg.IndexParSlot(), rep: rep,
		controle: controle}
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
func (p teamPublication) equipeDuSlot(slot uint32) (int, bool) {
	idx, ok := p.bySlot[slot]
	if !ok {
		return 0, false
	}
	t, ok := p.byIndex[idx]
	return t, ok
}

// poserSurLesTraces pose l'equipe du film sur chaque vie publiee et rend les deux comptes.
//
// L'ORDRE EST FIXE ET IL N'EST PAS ARBITRAIRE : le xuid d'abord (le lien direct du lot 1.6),
// le pont slot -> index ensuite (la seule voie d'un bot). Une vie que ni l'un ni l'autre ne
// nomme garde `-1`, et le compte le dit.
func (p teamPublication) poserSurLesTraces(tracks []Track) (total, nommees int) {
	for i := range tracks {
		total++
		if x, err := strconv.ParseUint(tracks[i].XUID, 10, 64); err == nil {
			if t, ok := p.equipeDuXUID(x); ok {
				tracks[i].Team, nommees = t, nommees+1
				continue
			}
		}
		if t, ok := p.equipeDuSlot(tracks[i].Slot); ok {
			tracks[i].Team, nommees = t, nommees+1
		}
	}
	return total, nommees
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
func (p teamPublication) couverture(vies, viesNommees int, entrees []RosterEntry) TeamCoverage {
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
	for _, e := range entrees {
		t, lu := p.byIndex[e.FilmIndex]
		if !lu {
			cov.Unread++
			continue
		}
		cov.Film++
		if t == filmdec.TeamNone {
			cov.NoTeam++
		}
		p.controler(&cov, e, t)
	}
	cov.Tracks, cov.TracksNamed = vies, viesNommees
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
		if t == filmdec.TeamNone {
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
		"silence", cov.Silence, "vies", cov.Tracks, "viesAvecEquipe", cov.TracksNamed)
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
