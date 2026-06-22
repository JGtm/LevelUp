package halo_5

// mapping_commendations.go — projection PURE des commendations NATIVES Halo 5 d'un
// carnage report (AXE B prod-gate) vers []persist.CommendationInsert (compteur
// par-match shared.match_commendations).
//
// Source per-match CONFIRMÉE (sonde live) : PlayerStats[].ProgressiveCommendationDeltas.
// Le COMPTE gagné CE match = Progress − PreviousProgress (analogue au compteur
// par-match de medals_earned). On NE reconstruit RIEN par tier/composite : c'est la
// donnée NATIVE telle quelle (décision produit AXE B).
//
// MetaCommendationDeltas : même forme, vide dans la sonde → inclus sur le même
// chemin si peuplé (best-effort, AXE B Phase 1 : « inclure si pertinent »). Les deux
// listes partagent la PK (match_id, xuid, commendation_id) : si un même UUID
// apparaît dans les deux, le 1er gagne (INSERT OR IGNORE côté persister).
//
// Vit dans le package halo_5 (et pas dans ingest) : ingest ne peut pas importer
// halo_5 (cycle), et le mapper réutilise le type carnage privé + produit
// directement le type persist consommé par ingest.CollectMatchBatch.AddCommendations.

import (
	"levelup/go-api/internal/persist"
)

// mapCarnageCommendations projette les deltas de commendations natives du carnage
// → []persist.CommendationInsert. resolveXUID(gamertag) → xuid Xbox résolu.
//
// RESOLVE-OR-SKIP : un joueur dont l'xuid ne résout pas ("") est SAUTÉ — la PK
// match_commendations (match_id, xuid, commendation_id) ferait collisionner ≥2
// joueurs non résolus sur xuid="" (parité avec mapCarnageParticipants).
//
// FILTRE : seuls les deltas à count = Progress − PreviousProgress > 0 produisent une
// row (une commendation listée sans progression réelle n'est pas « gagnée CE match »).
// L'ordre suit l'ordre du payload (déterminisme des tests).
func mapCarnageCommendations(matchID string, carnage *H5CarnageResponse, resolveXUID func(gamertag string) string) []persist.CommendationInsert {
	if carnage == nil || len(carnage.PlayerStats) == 0 {
		return nil
	}
	var out []persist.CommendationInsert
	for i := range carnage.PlayerStats {
		p := &carnage.PlayerStats[i]
		xuid := resolveXUID(p.Player.Gamertag)
		if xuid == "" {
			continue // resolve-or-skip (cf. godoc : PK xuid="" collisionnerait)
		}
		out = appendCommendationDeltas(out, matchID, xuid, p.ProgressiveCommendationDeltas)
		// MetaCommendationDeltas : inclus sur le même chemin (vide dans la sonde).
		out = appendCommendationDeltas(out, matchID, xuid, p.MetaCommendationDeltas)
	}
	return out
}

// appendCommendationDeltas convertit une liste de deltas (count = Progress −
// PreviousProgress) en rows, en filtrant les count ≤ 0 et les Id vides.
func appendCommendationDeltas(out []persist.CommendationInsert, matchID, xuid string, deltas []H5CommendationDelta) []persist.CommendationInsert {
	for j := range deltas {
		d := deltas[j]
		if d.Id == "" {
			continue
		}
		count := d.Progress - d.PreviousProgress
		if count <= 0 {
			continue // pas de progression réelle CE match → pas « gagnée »
		}
		out = append(out, persist.CommendationInsert{
			MatchID:        matchID,
			XUID:           xuid,
			CommendationID: d.Id,
			Count:          count,
		})
	}
	return out
}
