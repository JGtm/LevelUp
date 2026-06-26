package ingest

import (
	"strconv"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/persist"
)

// MapWeaponAccuracy agrège la PRÉCISION PAR ARME depuis les events `weapon_drop`
// d'une timeline Halo 5. Chaque WeaponDrop porte ShotsFired/ShotsLanded pour l'arme
// lâchée ; on somme par (xuid, weapon_id) sur le match → 1 row weapon_accuracy par
// arme effectivement tirée par un joueur.
//
// Pourquoi les events et pas le carnage : le carnage `WeaponStats[]` (arsenal complet
// par joueur) est servi VIDE par 343 ; la timeline WeaponDrop est la SEULE source de
// la précision par arme. La somme par joueur reconstitue exactement le carnage
// TotalShotsFired/TotalShotsLanded (validé 8/8 joueurs, cf. .ai/H5_EXPLORATION).
//
// Sont ignorés : les drops sans arme (weapon_id 0 / non numérique), sans joueur, et
// les drops à 0 tir tiré ET 0 touché (arme ramassée puis lâchée sans usage → bruit,
// n'affecte pas la précision ; la somme des tirs reste inchangée).
func MapWeaponAccuracy(
	matchID string,
	events []canonical.MatchEvent,
	resolveXUID func(gamertag string) string,
) []persist.WeaponAccuracyInsert {
	type key struct {
		xuid   string
		weapon uint64
	}
	agg := map[key]*persist.WeaponAccuracyInsert{}
	order := make([]key, 0)

	for i := range events {
		ev := events[i]
		if ev.Type != canonical.MatchEventWeaponDrop || ev.Player == nil || ev.Weapon == nil {
			continue
		}
		sf, sl := 0, 0
		if ev.ShotsFired != nil {
			sf = *ev.ShotsFired
		}
		if ev.ShotsLanded != nil {
			sl = *ev.ShotsLanded
		}
		if sf == 0 && sl == 0 {
			continue // arme non tirée → hors périmètre proficience
		}
		wid, err := strconv.ParseUint(ev.Weapon.ID, 10, 64)
		if err != nil || wid == 0 {
			continue
		}
		k := key{xuid: resolveXUID(ev.Player.Gamertag), weapon: wid}
		row := agg[k]
		if row == nil {
			row = &persist.WeaponAccuracyInsert{MatchID: matchID, XUID: k.xuid, WeaponID: wid}
			agg[k] = row
			order = append(order, k)
		}
		row.ShotsFired += sf
		row.ShotsLanded += sl
		row.Drops++
	}

	if len(order) == 0 {
		return nil
	}
	out := make([]persist.WeaponAccuracyInsert, 0, len(order))
	for _, k := range order {
		out = append(out, *agg[k])
	}
	return out
}
