package squademprise

// special_frags.go — LES FRAGS AUX ARMES SPÉCIALES D'UN MATCH, sous la MÊME définition que leurs
// prises : une arme spéciale est une arme apparue sur un socle de niveau `puissance` de CE match
// (D3), jamais une classe d'arme.
//
// DEUX SOURCES, UNE RÈGLE DE CHOIX PAR MATCH :
//
//	journal du film   match aux niveaux de socle mesurés ET au journal des morts publiable : les
//	                  frags dont la source de dégât désigne la clé de registre d'une famille
//	                  posée sur un socle de puissance du match, crédités au tueur du kill-feed ;
//	feuille de match  partout ailleurs (sans film, Halo 5 : D10 ; niveaux non mesurés ; journal
//	                  non publiable) : `power_weapon_kills`, la liste d'armes de puissance du JEU,
//	                  qui n'est pas celle des socles (le Needler d'un socle de puissance n'y
//	                  compte pas ; une arme de la liste posée sur un râtelier y compte).
//
// Le rendement « frags par prise » divise donc, sur un match aux niveaux mesurés, des frags et des
// prises qui portent sur les mêmes armes.
//
// UNE FAMILLE QU'UN AUTRE NIVEAU CONFIRMÉ PORTE AUSSI DANS LE MATCH (râtelier, départ) n'est pas
// comptée : ses frags ne se séparent pas entre les deux emplacements. Une famille sans clé de
// registre ne se reconnaît dans aucun frag. Une famille dont aucune prise n'est nommée n'a pas de
// ligne de niveau, donc pas de frag compté ici, comme elle n'a pas de prise comptée.

import (
	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/analysis/squadformes"
	"levelup/go-api/internal/domain"
)

// JournalKillRow — les frags d'un tueur avec une clé de registre d'arme, sur un match, lus au
// journal des morts du film (lignes publiables, tueur du kill-feed nommé).
type JournalKillRow struct {
	MatchID   string
	XUID      string
	WeaponKey string
	Kills     int
}

// JournalRead — la lecture du journal des morts du périmètre. Read : les matchs dont la passe de
// décodage est publiable ligne à ligne ; un match absent se lit sur la feuille de match.
type JournalRead struct {
	Rows []JournalKillRow
	Read map[string]bool
}

// specialWeaponKeys rend les clés de registre des armes spéciales d'un match : les familles d'un
// socle de puissance (au moins une prise nommée) qu'aucun autre niveau confirmé ne porte.
func specialWeaponKeys(rows []sessionusage.PadTierRow, weapons map[string]squadformes.WeaponInfo) map[string]bool {
	power, other := map[string]bool{}, map[string]bool{}
	for _, r := range rows {
		if r.Pickups <= 0 || r.WeaponFamily == "" {
			continue
		}
		switch r.Tier {
		case domain.PadTierPower:
			power[r.WeaponFamily] = true
		case domain.PadTierGround, domain.PadTierBase:
			other[r.WeaponFamily] = true
		}
	}
	keys := map[string]bool{}
	for fam := range power {
		if other[fam] {
			continue
		}
		if k := weapons[fam].WeaponKey; k != "" {
			keys[k] = true
		}
	}
	return keys
}

// journalSpecialFrags compte les frags du journal aux armes `keys`, par camp du tueur.
func journalSpecialFrags(c camp, keys map[string]bool, rows []JournalKillRow) *domain.SquadEmpriseCount {
	var n domain.SquadEmpriseCount
	for _, r := range rows {
		if !keys[r.WeaponKey] {
			continue
		}
		if us, _ := c.side(r.XUID); us {
			n.Us += r.Kills
		} else {
			n.Them += r.Kills
		}
	}
	return &n
}
