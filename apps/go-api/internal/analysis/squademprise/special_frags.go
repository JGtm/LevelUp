package squademprise

// special_frags.go — LES FRAGS AUX ARMES SPÉCIALES D'UN MATCH, sous la MÊME définition que leurs
// prises : une arme spéciale est une arme apparue sur un socle de niveau `puissance` de CE match
// (D3), jamais une classe d'arme.
//
// DEUX SOURCES, UNE RÈGLE DE CHOIX PAR MATCH :
//
//	journal du film   match aux niveaux de socle mesurés, au journal des morts publiable et au
//	                  catalogue d'armes lu : les frags dont la source de dégât désigne la clé de
//	                  registre d'une famille COMPTÉE (ci-dessous), crédités au camp du tueur ;
//	feuille de match  partout ailleurs (sans film, Halo 5 : D10 ; niveaux non mesurés ; journal
//	                  non publiable ; catalogue illisible) : `power_weapon_kills`, la liste d'armes
//	                  de puissance du JEU, qui n'est pas celle des socles (le Needler d'un socle de
//	                  puissance n'y compte pas ; une arme de la liste posée sur un râtelier y compte).
//
// UN SEUL PÉRIMÈTRE POUR LE RENDEMENT « frags par prise » : une famille est COMPTÉE quand elle a au
// moins une prise nommée sur un socle de puissance du match, aucune prise nommée à un autre niveau
// (râtelier, départ, emplacement non identifié : ses frags ne se séparent pas entre emplacements)
// et une clé de registre (sinon aucun frag ne se reconnaît). Les frags ET les prises du rendement
// ne portent que sur les familles comptées (matchTally.pwkPrises) ; le bilan des prises, lui, garde
// toutes les prises de puissance. Une famille sans prise nommée n'a pas de ligne de niveau : ni
// frag ni prise.
//
// LE CAMP DU TUEUR. Un suicide (tueur = victime) et une trahison (tueur et victime du même camp)
// ne sont pas des frags. Un tueur BOT n'a pas d'identité au journal : son camp se déduit de sa
// victime — l'adversaire d'une victime de notre camp ; le nôtre pour une victime de l'autre camp
// dans un match à DEUX camps. Ailleurs (victime bot ou inconnue, plus de deux camps), le frag ne
// se range dans aucun camp et n'est pas compté.

import (
	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/analysis/squadformes"
	"levelup/go-api/internal/domain"
)

// campsDuDuel : un match où le camp d'un bot se déduit de celui de sa victime.
const campsDuDuel = 2

// JournalKillRow — les frags d'un tueur sur une victime avec une clé de registre d'arme, sur un
// match, lus au journal des morts du film (lignes publiables). XUID vide = tueur bot ; VictimXUID
// vide = victime bot ou non résolue.
type JournalKillRow struct {
	MatchID    string
	XUID       string
	VictimXUID string
	WeaponKey  string
	Kills      int
}

// JournalRead — la lecture du journal des morts du périmètre. Read : les matchs dont la passe de
// décodage est publiable ligne à ligne ; un match absent se lit sur la feuille de match.
type JournalRead struct {
	Rows []JournalKillRow
	Read map[string]bool
}

// armesComptees — les familles comptées d'un match (famille -> clé de registre) et l'ensemble de
// leurs clés.
type armesComptees struct {
	families map[string]string
	keys     map[string]bool
}

// countedWeapons rend les familles comptées d'un match (en-tête du fichier).
func countedWeapons(rows []sessionusage.PadTierRow, weapons map[string]squadformes.WeaponInfo) armesComptees {
	power, other := map[string]bool{}, map[string]bool{}
	for _, r := range rows {
		if r.Pickups <= 0 || r.WeaponFamily == "" {
			continue
		}
		if r.Tier == domain.PadTierPower {
			power[r.WeaponFamily] = true
		} else {
			other[r.WeaponFamily] = true
		}
	}
	out := armesComptees{families: map[string]string{}, keys: map[string]bool{}}
	for fam := range power {
		k := weapons[fam].WeaponKey
		if other[fam] || k == "" {
			continue
		}
		out.families[fam], out.keys[k] = k, true
	}
	return out
}

// countedPrises compte les prises de puissance des familles comptées, par camp.
func countedPrises(c camp, rows []sessionusage.PadTierRow, a armesComptees) *domain.SquadEmpriseCount {
	var n domain.SquadEmpriseCount
	for _, r := range rows {
		if r.Tier != domain.PadTierPower || r.Pickups <= 0 {
			continue
		}
		if _, ok := a.families[r.WeaponFamily]; !ok {
			continue
		}
		if us, _ := c.side(r.XUID); us {
			n.Us += r.Pickups
		} else {
			n.Them += r.Pickups
		}
	}
	return &n
}

// journalSpecialFrags compte les frags du journal aux armes comptées, par camp du tueur.
func journalSpecialFrags(c camp, a armesComptees, rows []JournalKillRow) *domain.SquadEmpriseCount {
	var n domain.SquadEmpriseCount
	for _, r := range rows {
		if !a.keys[r.WeaponKey] {
			continue
		}
		us, ok := killerSide(c, r)
		switch {
		case !ok:
			continue
		case us:
			n.Us += r.Kills
		default:
			n.Them += r.Kills
		}
	}
	return &n
}

// killerSide dit si le tueur d'une ligne est de notre camp ; faux en second retour quand la ligne
// n'est pas un frag rangeable (suicide, trahison, bot au camp indéductible).
func killerSide(c camp, r JournalKillRow) (us, ok bool) {
	vt, vKnown := c.teamOf[r.VictimXUID]
	if r.XUID == "" {
		if !vKnown {
			return false, false
		}
		if vt == c.ours {
			return false, true
		}
		return true, campCount(c) == campsDuDuel
	}
	if r.XUID == r.VictimXUID {
		return false, false
	}
	if kt, kKnown := c.teamOf[r.XUID]; kKnown && vKnown && kt == vt {
		return false, false
	}
	us, _ = c.side(r.XUID)
	return us, true
}

// campCount — le nombre de camps distincts des participants du match.
func campCount(c camp) int {
	seen := map[int]bool{}
	for _, t := range c.teamOf {
		seen[t] = true
	}
	return len(seen)
}
