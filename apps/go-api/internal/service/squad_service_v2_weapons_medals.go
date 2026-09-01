// Package service — squad_service_v2_weapons_medals.go : helpers Tableau armes
// + Galerie medailles pour la page Squad V2 (cf. PLAN_SQUAD_GO_PORTAGE Phase
// P9, sections 21 + 22 audit).
//
//	Tableau armes  : aggregation kills par (joueur, arme), tri desc, slider
//	                 min kills cote front. Inclut grenades/melee si demande.
//	Galerie medailles : pour chaque match partage, liste des medailles par
//	                    joueur (xuid). Le tri par match suit la chronologie
//	                    pour une grille match-par-match.
//
// Les helpers sont purs : ils consomment des port.WeaponKillRow / port.MedalRow
// deja charges. Le repo DuckDB associe sera livre dans un chunk parallele
// (cf. S1→S1b pattern).
package service

import (
	"sort"

	"levelup/go-api/internal/port"
)

// WeaponsTableRow est une ligne du tableau armes du squad. Une ligne par arme,
// colonnes par joueur (kills) + total + grenade/melee separes.
//
// Le wrapper <WeaponsTable> rend ca avec tri par total, slider min kills,
// et colonne "is_grenade_melee" pour separer visuellement.
type WeaponsTableRow struct {
	WeaponID       int64          `json:"weapon_id"`
	Label          string         `json:"label,omitempty"`
	KillsByXUID    map[string]int `json:"kills_by_xuid"` // xuid -> kills
	Total          int            `json:"total"`
	IsGrenadeMelee bool           `json:"is_grenade_melee,omitempty"`
}

// BuildWeaponsTable agrege []port.WeaponKillRow en un tableau armes × joueurs.
//
//	rows       : pre-charge par WeaponKillsRepository.LoadWeaponKillsAggregated
//	             (avec MatchIDs = matchs partages, XUIDs = xuids du squad).
//	xuidToGT   : si non nil, KillsByXUID est rekey par gamertag (front-friendly).
//	             Si nil, conserve les xuids bruts.
//	minKills   : exclut les armes avec total < N (slider audit § 21).
//
// Tri : total desc, fallback weapon_id asc en cas d'egalite.
func BuildWeaponsTable(
	rows []port.WeaponKillRow,
	xuidToGT map[string]string,
	minKills int,
) []WeaponsTableRow {
	if len(rows) == 0 {
		return nil
	}
	// Aggreger par (arme, xuid). La cle d arme est le COUPLE (identifiant, cle de
	// registre) — cf. port.WeaponKillRow.AggregateKey : les objets hors arsenal n ont
	// aucun identifiant numerique et fusionneraient sinon en une seule ligne.
	type key struct {
		weaponID       int64
		weaponKey      string
		isGrenadeMelee bool
	}
	type weaponData struct {
		killsByID map[string]int
		total     int
		label     string
	}
	agg := make(map[key]*weaponData, len(rows))
	for _, r := range rows {
		k := key{weaponID: r.WeaponID, weaponKey: r.WeaponKey, isGrenadeMelee: r.IsGrenadeMelee}
		w, ok := agg[k]
		if !ok {
			w = &weaponData{killsByID: make(map[string]int)}
			agg[k] = w
		}
		w.killsByID[r.XUID] += r.Kills
		w.total += r.Kills
		if w.label == "" && r.Label != "" {
			w.label = r.Label
		}
	}

	out := make([]WeaponsTableRow, 0, len(agg))
	for k, w := range agg {
		if w.total < minKills {
			continue
		}
		// Optionnel : remap xuid -> gamertag.
		killsByOut := w.killsByID
		if xuidToGT != nil {
			remapped := make(map[string]int, len(w.killsByID))
			for xuid, kills := range w.killsByID {
				gt := xuidToGT[xuid]
				if gt == "" {
					gt = xuid
				}
				remapped[gt] += kills
			}
			killsByOut = remapped
		}
		out = append(out, WeaponsTableRow{
			WeaponID:       k.weaponID,
			Label:          w.label,
			KillsByXUID:    killsByOut,
			Total:          w.total,
			IsGrenadeMelee: k.isGrenadeMelee,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Total != out[j].Total {
			return out[i].Total > out[j].Total
		}
		return out[i].WeaponID < out[j].WeaponID
	})
	return out
}

// MedalsGalleryEntry est l'entree par match de la galerie medailles.
//
//	MatchID         : identifiant match.
//	StartedAtUTC    : pour ordonner la grille (recents en premier).
//	MedalsByXUID    : map xuid -> liste de medailles (gardes l'ordre par
//	                  count desc puis medal_id pour stabilite).
//
// Le wrapper <MedalsGallery> rend une carte par match avec un agregat
// visuel (rangee de medailles par joueur).
type MedalsGalleryEntry struct {
	MatchID      string                  `json:"match_id"`
	MedalsByXUID map[string][]MedalEntry `json:"medals_by_xuid"`
}

// MedalEntry decrit une medaille gagnee par un joueur sur un match.
type MedalEntry struct {
	MedalID int64  `json:"medal_id"`
	Count   int    `json:"count"`
	Label   string `json:"label,omitempty"`
}

// BuildMedalsGallery agrege []port.MedalRow en MedalsGalleryEntry par match.
//
//	rows         : pre-charge par MedalsByXUIDRepository.LoadMedalsForMatchesByXUID.
//	xuidToGT     : si non nil, MedalsByXUID est rekey par gamertag.
//	matchOrder   : ordre des MatchID retournes (par defaut, ordre alpha
//	               match_id asc — le service amont peut fournir un ordre
//	               chrono via SquadSharedMatch tri).
//
// Tri interne : pour chaque match × xuid, les medailles sont triees count
// desc puis medal_id asc.
func BuildMedalsGallery(
	rows []port.MedalRow,
	xuidToGT map[string]string,
	matchOrder []string,
) []MedalsGalleryEntry {
	if len(rows) == 0 {
		return nil
	}
	// Aggreger par (match_id, xuid|gt) -> []MedalEntry.
	type key struct {
		matchID, who string
	}
	agg := make(map[key]map[int64]*MedalEntry)
	for _, r := range rows {
		who := r.XUID
		if xuidToGT != nil {
			if gt := xuidToGT[r.XUID]; gt != "" {
				who = gt
			}
		}
		k := key{matchID: r.MatchID, who: who}
		byMedal, ok := agg[k]
		if !ok {
			byMedal = make(map[int64]*MedalEntry)
			agg[k] = byMedal
		}
		entry, exists := byMedal[r.MedalID]
		if !exists {
			entry = &MedalEntry{MedalID: r.MedalID, Label: r.Label}
			byMedal[r.MedalID] = entry
		}
		entry.Count += r.Count
		if entry.Label == "" && r.Label != "" {
			entry.Label = r.Label
		}
	}

	// Decider de l'ordre des matchs : matchOrder si fourni, sinon collecte.
	var orderedMatches []string
	if len(matchOrder) > 0 {
		orderedMatches = append(orderedMatches, matchOrder...)
	} else {
		seen := make(map[string]bool)
		for k := range agg {
			if !seen[k.matchID] {
				orderedMatches = append(orderedMatches, k.matchID)
				seen[k.matchID] = true
			}
		}
		sort.Strings(orderedMatches)
	}

	out := make([]MedalsGalleryEntry, 0, len(orderedMatches))
	for _, matchID := range orderedMatches {
		entry := MedalsGalleryEntry{
			MatchID:      matchID,
			MedalsByXUID: make(map[string][]MedalEntry),
		}
		// Collecte des who pour ce match.
		whos := make([]string, 0)
		for k := range agg {
			if k.matchID == matchID {
				whos = append(whos, k.who)
			}
		}
		sort.Strings(whos)
		for _, who := range whos {
			byMedal := agg[key{matchID: matchID, who: who}]
			medals := make([]MedalEntry, 0, len(byMedal))
			for _, m := range byMedal {
				medals = append(medals, *m)
			}
			sort.SliceStable(medals, func(i, j int) bool {
				if medals[i].Count != medals[j].Count {
					return medals[i].Count > medals[j].Count
				}
				return medals[i].MedalID < medals[j].MedalID
			})
			entry.MedalsByXUID[who] = medals
		}
		// Skipper les matchs sans medaille (evite d'inclure le match dans la
		// galerie si rien a montrer).
		if len(entry.MedalsByXUID) > 0 {
			out = append(out, entry)
		}
	}
	return out
}
