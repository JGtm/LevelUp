// Package duckdb — highlight_events_repo.go : implementation DuckDB du loader
// unifie des events filmes (port.HighlightEventsRepository).
//
// Per-PlayerDB : un HighlightEventsRepo est lie a un PlayerDB precis. La lecture
// passe par SharedReadDB().Get() qui retourne (ADR 0016) une connexion DIRECTE
// a shared_matches_v2.duckdb — les tables sont a la racine du catalogue, sans
// alias `shared`. Les queries referencent donc highlight_events / match_registry
// en bare (PAS `shared.*`, qui ne resout que sur la topologie de test legacy).
//
// Capability gating : laisse au service appelant pour cette implementation.
// Le repo execute la requete telle quelle ; si le titre n'a pas la capability
// "match.detail.events", c'est au service de retourner games.ErrCapabilityNotSupported
// avant d'appeler le repo.
package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// HighlightEventsRepo charge les events filmes depuis shared.highlight_events.
type HighlightEventsRepo struct {
	pdb *PlayerDB
}

// NewHighlightEventsRepo cree un HighlightEventsRepo lie a un PlayerDB.
func NewHighlightEventsRepo(pdb *PlayerDB) *HighlightEventsRepo {
	return &HighlightEventsRepo{pdb: pdb}
}

// Load charge les events selon les filtres fournis.
//
// L'appelant doit avoir valide les filtres via filters.Validate() en amont
// (rejette notamment les combinaisons trop larges qui declencheraient un scan
// complet de shared.highlight_events). Le repo re-applique sa propre validation
// defensive.
//
// Mapping XUID -> KillerXUID/VictimXUID/PlayerXUID : la table shared.highlight_events
// stocke un seul `xuid` par row dont le sens depend du event_type :
//
//	kill, first_kill, finisher, clutch -> xuid = tueur (KillerXUID)
//	death, first_death                 -> xuid = victime (VictimXUID)
//	medal, assist                      -> xuid = joueur centrant l'event (PlayerXUID)
//
// Le champ legacy XUID est conserve (compat). Les autres pointeurs (KillerXUID,
// VictimXUID, PlayerXUID) sont peuples selon ce mapping pour permettre aux
// consommateurs (analysis/narrative) d'utiliser la nouvelle API.
func (r *HighlightEventsRepo) Load(
	ctx context.Context,
	filters port.HighlightEventFilters,
) ([]canonical.HighlightEvent, error) {
	if err := filters.Validate(); err != nil {
		return nil, fmt.Errorf("HighlightEventsRepo.Load: %w", err)
	}

	q, args, err := buildHighlightEventsQuery(filters)
	if err != nil {
		return nil, fmt.Errorf("HighlightEventsRepo.Load: build query: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("HighlightEventsRepo.Load: shared reader: %w", err)
	}
	defer release()

	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("HighlightEventsRepo.Load: query: %w", err)
	}
	defer rows.Close()

	var out []canonical.HighlightEvent
	for rows.Next() {
		ev, err := scanHighlightEvent(rows)
		if err != nil {
			return nil, fmt.Errorf("HighlightEventsRepo.Load: scan: %w", err)
		}
		out = append(out, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("HighlightEventsRepo.Load: rows: %w", err)
	}

	// Fallback title-agnostic (centralisé au niveau lecture) : highlight_events ne
	// porte pas forcément les kills selon le titre. Infinite y stocke
	// kill/death/medal ; Halo 5 n'y stocke QUE des médailles, les kills horodatés
	// vivant dans killer_victim_pairs. Si le lot ne contient AUCUN kill/death, on
	// synthétise les kill/death depuis killer_victim_pairs (même DB shared) via le
	// helper partagé analysis.SynthesizeKillEventsFromKVPairs (source unique) et on
	// les fusionne, triés par TimeMS. NO-OP sur Infinite (kills déjà présents →
	// fallback jamais pris). Couvre la timeseries solo (chart .11 first events +
	// chart .21 intensité), tous deux vides en H5 sans ce fallback.
	if !analysis.HasCanonicalKillOrDeath(out) {
		synth := r.loadSyntheticKillEvents(ctx, db, filters.MatchIDs)
		if len(synth) > 0 {
			out = analysis.MergeAndSortCanonicalEvents(out, synth)
		}
	}
	return out, nil
}

// loadSyntheticKillEvents charge killer_victim_pairs (batch sur matchIDs) et
// synthétise des events canoniques kill/death via le helper partagé
// analysis.SynthesizeKillEventsFromKVPairs (1 paire → 1 kill + 1 death, source
// unique de la règle). Best-effort : matchIDs vide ou table absente → nil sans
// erreur (l'absence de fallback dégrade en charts vides, pas en 500). Réutilise
// la connexion shared déjà acquise par Load (pas de second Get).
func (r *HighlightEventsRepo) loadSyntheticKillEvents(
	ctx context.Context, db *sql.DB, matchIDs []string,
) []canonical.HighlightEvent {
	if len(matchIDs) == 0 {
		return nil
	}
	placeholders := make([]string, len(matchIDs))
	args := make([]any, len(matchIDs))
	for i, id := range matchIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	// Source canonique depuis le 2026-08-03. Bots (xuid NULL) écartés au SQL et non
	// normalisés en chaîne vide — même raison qu'en Q32c (cf. Q32cSquadKVPairsTemplate).
	q := fmt.Sprintf(`
SELECT match_id, feed_killer_xuid, victim_xuid, 1, time_ms
FROM `+KillEventsCanonicalTable+`
WHERE match_id IN (%s)
  AND feed_killer_xuid IS NOT NULL
  AND victim_xuid      IS NOT NULL
ORDER BY match_id, time_ms ASC`, strings.Join(placeholders, ","))

	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	// Regrouper par match (préserve l'ordre d'apparition) puis synthétiser par
	// match — SynthesizeKillEventsFromKVPairs pose MatchID sur chaque event.
	byMatch := make(map[string][]analysis.KVSyntheticInput)
	order := make([]string, 0, len(matchIDs))
	for rows.Next() {
		var (
			matchID string
			in      analysis.KVSyntheticInput
		)
		if err := rows.Scan(&matchID, &in.KillerXUID, &in.VictimXUID, &in.KillCount, &in.TimeMS); err != nil {
			continue
		}
		if _, seen := byMatch[matchID]; !seen {
			order = append(order, matchID)
		}
		byMatch[matchID] = append(byMatch[matchID], in)
	}
	if err := rows.Err(); err != nil {
		return nil
	}
	var out []canonical.HighlightEvent
	for _, mid := range order {
		out = append(out, analysis.SynthesizeKillEventsFromKVPairs(byMatch[mid], mid)...)
	}
	return out
}

// buildHighlightEventsQuery compose le SELECT et les WHERE dynamiques selon
// les filtres. Les valeurs scalaires passent par placeholders ; les fragments
// structurels (IN-list, ORDER BY) sont assembles via whitelist.
func buildHighlightEventsQuery(f port.HighlightEventFilters) (string, []any, error) {
	var sb strings.Builder
	sb.WriteString(highlightEventsBaseSelect)

	var args []any
	var whereParts []string

	if len(f.MatchIDs) > 0 {
		placeholders := make([]string, 0, len(f.MatchIDs))
		for _, id := range f.MatchIDs {
			placeholders = append(placeholders, "?")
			args = append(args, id)
		}
		whereParts = append(whereParts,
			fmt.Sprintf("he.match_id IN (%s)", strings.Join(placeholders, ",")))
	}
	if f.PlayerXUID != nil {
		whereParts = append(whereParts, "he.xuid = ?")
		args = append(args, *f.PlayerXUID)
	}
	if len(f.EventTypes) > 0 {
		placeholders := make([]string, 0, len(f.EventTypes))
		for _, t := range f.EventTypes {
			placeholders = append(placeholders, "?")
			args = append(args, string(t))
		}
		whereParts = append(whereParts,
			fmt.Sprintf("he.event_type IN (%s)", strings.Join(placeholders, ",")))
	}
	if f.Since != nil {
		whereParts = append(whereParts, StartTimeCanonicalSQL("r")+" >= ?")
		args = append(args, *f.Since)
	}

	if len(whereParts) > 0 {
		sb.WriteString(" WHERE ")
		sb.WriteString(strings.Join(whereParts, " AND "))
	}

	orderBy, err := highlightEventsOrderBy(f.OrderBy)
	if err != nil {
		return "", nil, err
	}
	sb.WriteString(" ORDER BY ")
	sb.WriteString(orderBy)

	if f.Limit > 0 {
		sb.WriteString(" LIMIT ?")
		args = append(args, f.Limit)
	}

	return sb.String(), args, nil
}

// highlightEventsBaseSelect porte les colonnes du SELECT + le LEFT JOIN sur
// match_registry (pour le filtre Since via start_time).
const highlightEventsBaseSelect = `
SELECT
    he.match_id,
    he.event_type,
    COALESCE(he.time_ms, 0) AS time_ms,
    he.xuid
FROM highlight_events he
LEFT JOIN match_registry r ON r.match_id = he.match_id`

// highlightEventsOrderBy traduit l'OrderBy filtre en expression SQL safe
// (whitelist fermee). Vide -> ordre par defaut (match_id ASC, time_ms ASC).
func highlightEventsOrderBy(s string) (string, error) {
	switch strings.TrimSpace(s) {
	case "", "match_id ASC, time_ms ASC":
		return "he.match_id ASC, he.time_ms ASC", nil
	case "match_id DESC, time_ms ASC":
		return "he.match_id DESC, he.time_ms ASC", nil
	case "time_ms ASC":
		return "he.time_ms ASC", nil
	case "time_ms DESC":
		return "he.time_ms DESC", nil
	}
	return "", fmt.Errorf("%w: %q", ErrUnknownHighlightEventsOrderBy, s)
}

// ErrUnknownHighlightEventsOrderBy est retournee si OrderBy n'est pas dans la
// whitelist.
var ErrUnknownHighlightEventsOrderBy = errors.New("HighlightEventsRepo: unknown OrderBy")

// scanHighlightEvent scanne une row SQL en canonical.HighlightEvent et applique
// le mapping XUID -> KillerXUID/VictimXUID/PlayerXUID selon event_type.
func scanHighlightEvent(rows *sql.Rows) (canonical.HighlightEvent, error) {
	var (
		matchID, eventType string
		timeMS             int64
		xuid               sql.NullString
	)
	if err := rows.Scan(&matchID, &eventType, &timeMS, &xuid); err != nil {
		return canonical.HighlightEvent{}, err
	}

	ev := canonical.HighlightEvent{
		MatchID:   matchID,
		EventType: eventType,
		TimeMS:    timeMS,
	}
	if xuid.Valid {
		ev.XUID = xuid.String
		assignXUIDByEventType(&ev, xuid.String, eventType)
	}
	return ev, nil
}

// assignXUIDByEventType peuple KillerXUID / VictimXUID / PlayerXUID selon
// le sens semantique de l'event. Voir docstring de Load() pour le mapping.
func assignXUIDByEventType(ev *canonical.HighlightEvent, xuid, eventType string) {
	x := xuid
	switch eventType {
	case string(canonical.EventKill),
		string(canonical.EventFirstKill),
		string(canonical.EventFinisher),
		string(canonical.EventClutch):
		ev.KillerXUID = &x
	case string(canonical.EventDeath),
		string(canonical.EventFirstDeath):
		ev.VictimXUID = &x
	case string(canonical.EventMedal),
		string(canonical.EventAssist):
		ev.PlayerXUID = &x
	}
}
