// Package duckdb — squad_repo_kv_fallback.go : fallback title-agnostic de synthèse
// d'events kill/death depuis killer_victim_pairs (kvPairs). Découpe de squad_repo.go
// (god-file split, dépassement limite 500 L). LoadImpactEvents reste dans squad_repo.go ;
// ce fichier porte la lecture batch des kvPairs et la reconstruction/fusion des events.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// LoadKVPairs charge les paires killer→victim horodatées (killer_victim_pairs)
// en batch pour une liste de match_ids. Source du fallback title-agnostic de
// synthèse d'events kill/death. Retourne nil si matchIDs est vide.
func (r *SquadRepo) LoadKVPairs(ctx context.Context, matchIDs []string) ([]domain.KVPairRaw, error) {
	if len(matchIDs) == 0 {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("LoadKVPairs: shared reader: %w", err)
	}
	defer release()
	return r.loadKVPairsOn(ctx, db, matchIDs)
}

// loadKVPairsOn exécute Q32c sur une connexion shared déjà acquise. Factorise la
// lecture pour LoadKVPairs (public) et le fallback interne de LoadImpactEvents
// (qui réutilise la connexion déjà ouverte, évitant un second Get sous timeout).
func (r *SquadRepo) loadKVPairsOn(ctx context.Context, db *sql.DB, matchIDs []string) ([]domain.KVPairRaw, error) {
	placeholders := make([]string, len(matchIDs))
	args := make([]interface{}, len(matchIDs))
	for i, id := range matchIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	query := fmt.Sprintf(Q32cSquadKVPairsTemplate, strings.Join(placeholders, ","))

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("loadKVPairsOn: %w", err)
	}
	defer rows.Close()

	var result []domain.KVPairRaw
	for rows.Next() {
		var (
			matchID string
			kv      domain.KVPairRaw
		)
		if err := rows.Scan(&matchID, &kv.KillerXUID, &kv.VictimXUID, &kv.KillCount, &kv.TimeMS); err != nil {
			return nil, fmt.Errorf("loadKVPairsOn scan: %w", err)
		}
		kv.MatchID = matchID
		result = append(result, kv)
	}
	return result, rows.Err()
}

// impactRowsHaveKillOrDeath indique si le lot d'events highlight escouade contient
// au moins un kill ou death. Miroir de analysis.HasCanonicalKillOrDeath pour la
// forme domain.ImpactEventRow (mêmes valeurs canoniques "kill"/"death"). Sert à
// décider du fallback synthétique kvPairs → events (titres dont highlight_events
// ne porte que des médailles, ex. Halo 5).
func impactRowsHaveKillOrDeath(rows []domain.ImpactEventRow) bool {
	for _, r := range rows {
		switch canonical.HighlightEventType(r.EventType) {
		case canonical.EventKill, canonical.EventDeath:
			return true
		}
	}
	return false
}

// synthesizeImpactRowsFromKVPairs reconstruit des ImpactEventRow kill/death (1
// paire → 1 kill acteur=tueur + 1 death acteur=victime) depuis les paires
// killer→victim batch, en respectant le regroupement par match (kv.MatchID). La
// règle de synthèse est centralisée dans analysis.SynthesizeKillEventsFromKVPairs
// (source unique partagée avec engagement.go / match-view) — appelée par match.
// On reconvertit l'event canonique en ImpactEventRow (XUID = acteur) attendu par
// les builders Escouade. Gamertag laissé vide : résolu en aval depuis le xuid.
func synthesizeImpactRowsFromKVPairs(pairs []domain.KVPairRaw) []domain.ImpactEventRow {
	if len(pairs) == 0 {
		return nil
	}
	// Regrouper par match (préserve l'ordre d'apparition des match_ids).
	byMatch := make(map[string][]analysis.KVSyntheticInput, len(pairs))
	order := make([]string, 0)
	for _, kv := range pairs {
		if _, seen := byMatch[kv.MatchID]; !seen {
			order = append(order, kv.MatchID)
		}
		byMatch[kv.MatchID] = append(byMatch[kv.MatchID], analysis.KVSyntheticInput{
			KillerXUID: kv.KillerXUID,
			VictimXUID: kv.VictimXUID,
			TimeMS:     kv.TimeMS,
			KillCount:  kv.KillCount,
		})
	}
	out := make([]domain.ImpactEventRow, 0, len(pairs)*2)
	for _, mid := range order {
		canon := analysis.SynthesizeKillEventsFromKVPairs(byMatch[mid], mid)
		for i := range canon {
			out = append(out, domain.ImpactEventRow{
				MatchID:   canon[i].MatchID,
				XUID:      canon[i].XUID,
				EventType: canon[i].EventType,
				TimeMS:    canon[i].TimeMS,
			})
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// mergeImpactRowsByTime fusionne deux listes d'ImpactEventRow (médailles
// existantes + kill/death synthétiques) et les trie par TimeMS croissant, stable.
// Aligne le lot d'events sur l'ordre chronologique attendu par les builders
// (Q32 ORDER BY he.match_id, he.time_ms ; ici on conserve un tri global par
// TimeMS, suffisant car les builders regroupent par match_id en aval).
func mergeImpactRowsByTime(existing, synth []domain.ImpactEventRow) []domain.ImpactEventRow {
	if len(existing) == 0 {
		return synth
	}
	if len(synth) == 0 {
		return existing
	}
	out := make([]domain.ImpactEventRow, 0, len(existing)+len(synth))
	out = append(out, existing...)
	out = append(out, synth...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].TimeMS < out[j].TimeMS })
	return out
}
