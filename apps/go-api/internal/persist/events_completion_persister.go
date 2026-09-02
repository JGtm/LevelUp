// Package persist — events_completion_persister.go : écriture atomique de la
// COMPLÉTION combat (highlight_events + killer_victim_pairs + match_kill_events
// + bits de complétude) dans shared_matches_v2.duckdb.
//
// Pourquoi un persister dédié (et pas SharedPersister) :
//
//   - SharedPersister.Persist est no-op si batch.Shared.Match == nil et skip si
//     le match_registry existe déjà (idempotence sur création de match). Or la
//     complétion écrit des events SUR UN MATCH DÉJÀ INSÉRÉ (le film n'était pas
//     prêt au sync primaire, il arrive un cycle plus tard). SharedPersister est
//     donc structurellement inutilisable ici.
//
// Ce persister remplace les écritures `db.Exec` directes que faisait la
// complétion legacy (sync.InsertHighlightEvents + sync.InsertKillerVictimPairs-
// FromEvents + sync.MarkEventsLoaded/MarkKillerVictimLoaded), 3 opérations NON
// atomiques. Ici tout passe en UNE transaction sur le writer RW (cf. règle
// absolue : zéro écriture shared hors package persist ; cf. ADR 0019 +
// .ai/HANDOFF_sync_combat_completion.md).
//
// Schéma killer_victim_pairs : on écrit la forme **par-kill** (1 row par kill,
// avec gamertags + time_ms) — c'est ce que lit le match-view (tug-of-war, KD
// timeline, antagonistes). NE PAS confondre avec persistKillerVictim
// (shared_persister.go) qui écrit la forme agrégée (kill_count, sans gamertags
// ni time_ms) destinée au flux primaire — lossy pour la complétion.
//
// TROISIÈME TABLE ÉCRITE DEPUIS LE 2026-08-02 : `match_kill_events`, la table
// canonique des morts (double écriture datée, cf. persistKillerVictim). Les
// mêmes couples y entrent par l'unique traduction du dépôt
// (CreditKillEventsFromPairs → persistCreditKillEvents), DANS LA MÊME
// TRANSACTION, et sous la préséance du film. C'est le chemin par lequel les
// matchs dont le film arrive un cycle après le sync primaire obtiennent leurs
// morts — l'omettre de cet en-tête a masqué qu'aucun test ne l'assertait
// (constat J4R-2 ; couvert depuis par
// TestEventsCompletionPersister_EcritLaTableCanonique).
//
// ⚠️ ART-safety (asymétrie volontaire, alignée sur la complétion legacy) :
//   - highlight_events possède des index ART (PK id seq + idx_highlight_match +
//     idx_events_match_type) → INSERT-only, JAMAIS de DELETE (un DELETE sur une
//     table ART-indexée est le trigger exact du bug `Failed to delete all rows
//     from index`, cf. ADR 0019). La non-duplication est garantie en amont par
//     le gate events_loaded (un match déjà complété ne repasse pas par ici).
//   - killer_victim_pairs n'a aucun index/PK → DELETE-then-INSERT par match_id
//     est sûr (rien à corrompre) et rend la réécriture idempotente (c'est déjà
//     ce que faisait la complétion legacy, writes.go).
// L'UPDATE match_registry sur backfill_completed/events_loaded suit le pattern
// bitmask existant (sync.MarkEventsLoaded).

package persist

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// HLEventCompletion — une row highlight_events pour la complétion.
//
// Mappe sur (match_id, event_type, time_ms, xuid, type_hint, raw_json). match_id
// est porté par l'input (commun à toutes les rows). XUID est déjà sérialisé en
// décimal par le caller (strconv.FormatUint), conformément au schéma VARCHAR de
// la colonne.
//
// RawJSON porte l'identité de la médaille (`{"medal_name":"..."}`) sur les events
// `medal`, nil partout ailleurs. Il existe parce que cette voie est le SECOND
// écrivain vivant de la table : le flux primaire (collect→persist) n'est pas seul,
// la complétion/convergence réécrit les events des matchs dont le film n'était pas
// publié au sync primaire, ou qui étaient déjà en registry via un coéquipier. Tant
// que ce chemin n'écrivait pas `raw_json`, chaque cycle rouvrait le trou que le
// flux primaire venait de fermer (revue adversariale du 2026-09-02).
type HLEventCompletion struct {
	XUID      string
	EventType string
	TimeMS    int
	TypeHint  int
	RawJSON   *string
}

// KVPairCompletion — une row killer_victim_pairs (forme par-kill).
//
// Distincte de KillerVictimInsert (forme agrégée du flux primaire). Porte les
// gamertags + time_ms requis par le match-view.
type KVPairCompletion struct {
	KillerXUID     string
	KillerGamertag string
	VictimXUID     string
	VictimGamertag string
	TimeMS         int64
}

// EventsCompletionInput — payload complet d'une complétion combat pour 1 match.
//
// EventsBit / KillerVictimBit sont passés par le caller (package sync :
// MBitEvents / MBitKillerVictim) car persist ne doit pas importer sync.
// MarkKV : marquer le bit killer_victim même si Pairs est vide (parité legacy —
// la complétion legacy marquait le bit dès que le calcul des paires réussissait,
// y compris 0 paire).
type EventsCompletionInput struct {
	MatchID         string
	Events          []HLEventCompletion
	Pairs           []KVPairCompletion
	MarkKV          bool
	EventsBit       int64
	KillerVictimBit int64
}

// EventsCompletionPersister écrit une complétion combat en INSERT atomique sur
// le writer RW shared.
type EventsCompletionPersister struct {
	db txBeginner
}

// NewEventsCompletionPersister construit un persister. `db` doit pointer vers
// une connexion ayant un write lease actif sur shared_matches_v2.duckdb (le
// writer RW du SharedDBProvider tenu par le post-sync).
func NewEventsCompletionPersister(db txBeginner) *EventsCompletionPersister {
	return &EventsCompletionPersister{db: db}
}

// Persist écrit la complétion en 1 transaction :
//
//	(1) si len(Events) > 0  : DELETE highlight_events WHERE match_id + INSERT × N
//	(2) si len(Pairs)  > 0  : DELETE killer_victim_pairs WHERE match_id + INSERT × N
//	(3) UPDATE match_registry : backfill_completed |= mask, events_loaded |= flag
//
// mask = (EventsBit si des events ont été insérés) | (KillerVictimBit si MarkKV).
// events_loaded passe TRUE uniquement si des events ont été réellement insérés.
//
// Retourne le nombre d'events insérés (pour result.EventsInserted côté caller).
// En cas d'erreur, ROLLBACK (defer) — aucun état partiel.
func (p *EventsCompletionPersister) Persist(ctx context.Context, in EventsCompletionInput) (int, error) {
	if in.MatchID == "" {
		return 0, errors.New("persist: EventsCompletionPersister: MatchID vide")
	}

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("persist: EventsCompletion BeginTx: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // no-op après Commit

	eventsInserted, err := insertCompletionHighlightEvents(ctx, tx, in.MatchID, in.Events)
	if err != nil {
		return 0, err
	}

	if err := insertCompletionKillerVictim(ctx, tx, in.MatchID, in.Pairs); err != nil {
		return 0, err
	}

	if err := markCompletionRegistry(ctx, tx, in, eventsInserted); err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("persist: EventsCompletion Commit %s: %w", in.MatchID, err)
	}
	return eventsInserted, nil
}

// MarkNoFilmDefinitive marque un match dont le film est définitivement absent
// (404 + match plus vieux que la fenêtre de retry) : events_loaded=TRUE +
// backfill_completed |= eventsBit, pour le sortir du retry set. Aucune donnée
// combat n'est écrite (il n'y en a pas). Sur le writer RW, hors db.Exec direct.
//
// eventsBit = sync.MBitEvents (passé par le caller, persist n'importe pas sync).
func (p *EventsCompletionPersister) MarkNoFilmDefinitive(ctx context.Context, matchID string, eventsBit int64) error {
	if matchID == "" {
		return errors.New("persist: MarkNoFilmDefinitive: matchID vide")
	}
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("persist: MarkNoFilmDefinitive BeginTx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
		UPDATE match_registry
		SET backfill_completed = COALESCE(backfill_completed, 0) | ?,
		    events_loaded = TRUE
		WHERE match_id = ?`, eventsBit, matchID); err != nil {
		return fmt.Errorf("persist: MarkNoFilmDefinitive update %s: %w", matchID, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("persist: MarkNoFilmDefinitive Commit %s: %w", matchID, err)
	}
	return nil
}

// MarkEventsEmptyDefinitive marque un match dont le chunk film a bien été récupéré
// ET parsé sans erreur, mais qui ne contient AUCUN highlight event (vide LÉGITIME,
// ex. chunk minuscule sans marqueur). Statut DISTINCT de events_loaded (qui reste
// FALSE — aucun event n'a été chargé, on ne ment pas) : on pose events_empty=TRUE +
// backfill_completed |= eventsBit pour SORTIR le match du retry set (fin de la boucle
// parse_anomaly re-fetch/re-parse à chaque cycle). Les matchs events_empty=TRUE
// restent identifiables/auditables (SELECT ... WHERE events_empty) et re-processables
// via un backfill --force si un fix parser arrive plus tard. Writer RW, hors db.Exec direct.
//
// eventsBit = sync.MBitEvents (passé par le caller, persist n'importe pas sync).
func (p *EventsCompletionPersister) MarkEventsEmptyDefinitive(ctx context.Context, matchID string, eventsBit int64) error {
	if matchID == "" {
		return errors.New("persist: MarkEventsEmptyDefinitive: matchID vide")
	}
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("persist: MarkEventsEmptyDefinitive BeginTx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
		UPDATE match_registry
		SET backfill_completed = COALESCE(backfill_completed, 0) | ?,
		    events_empty = TRUE
		WHERE match_id = ?`, eventsBit, matchID); err != nil {
		return fmt.Errorf("persist: MarkEventsEmptyDefinitive update %s: %w", matchID, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("persist: MarkEventsEmptyDefinitive Commit %s: %w", matchID, err)
	}
	return nil
}

// insertCompletionHighlightEvents insère les highlight_events du match en
// INSERT-only (pas de DELETE — table ART-indexée, cf. en-tête). La colonne id
// est auto-générée (seq). Retourne le nombre de rows effectivement insérées.
//
// INSERT OR IGNORE : parité exacte avec la complétion legacy
// (sync.InsertHighlightEvents). Tolère les events dupliqués (l'API Halo peut
// répéter un event ; certains schémas ont une contrainte unique composite sur
// (match_id, xuid, time_ms, event_type)). En prod (PK id auto-seq) OR IGNORE est
// un no-op de dédup ; sur les schémas à clé composite il dédupe.
func insertCompletionHighlightEvents(ctx context.Context, tx *sql.Tx, matchID string, events []HLEventCompletion) (int, error) {
	if len(events) == 0 {
		return 0, nil
	}
	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR IGNORE INTO highlight_events (match_id, event_type, time_ms, xuid, type_hint, raw_json)
		VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, fmt.Errorf("persist: EventsCompletion prepare highlight_events %s: %w", matchID, err)
	}
	defer func() { _ = stmt.Close() }()

	inserted := 0
	for _, e := range events {
		res, execErr := stmt.ExecContext(ctx, matchID, e.EventType, e.TimeMS, e.XUID, e.TypeHint, e.RawJSON)
		if execErr != nil {
			return 0, fmt.Errorf("persist: EventsCompletion insert highlight_events %s: %w", matchID, execErr)
		}
		if n, _ := res.RowsAffected(); n > 0 {
			inserted++
		}
	}
	return inserted, nil
}

// insertCompletionKillerVictim écrit les couples tueur → victime de la complétion combat,
// DANS LES DEUX TABLES (double écriture datée du 2026-08-02, cf. persistKillerVictim).
//
// Côté table canonique, elle AJOUTE UNE PASSE : deux passes ne s'additionnent pas, la vue
// `_latest` ne retient que la dernière. C'est là que meurt le mécanisme des doublons — le
// DELETE-then-INSERT ci-dessous face à l'INSERT nu du flux primaire, 46,5 % de la table et un
// facteur 2,006 sur les agrégats carrière. Il subsiste tant que la table historique est lue,
// et il part avec elle.
func insertCompletionKillerVictim(ctx context.Context, tx *sql.Tx, matchID string, pairs []KVPairCompletion) error {
	if len(pairs) == 0 {
		return nil
	}
	if err := persistCreditKillEvents(ctx, tx,
		CreditKillEventsFromPairs(ctx, matchID, killerVictimRowsFromCompletion(matchID, pairs))); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM killer_victim_pairs WHERE match_id = ?`, matchID); err != nil {
		return fmt.Errorf("persist: EventsCompletion delete killer_victim_pairs %s: %w", matchID, err)
	}
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO killer_victim_pairs
			(match_id, killer_xuid, killer_gamertag, victim_xuid, victim_gamertag, time_ms, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("persist: EventsCompletion prepare killer_victim_pairs %s: %w", matchID, err)
	}
	defer func() { _ = stmt.Close() }()

	now := time.Now().UTC()
	for _, kv := range pairs {
		if _, err := stmt.ExecContext(ctx, matchID,
			kv.KillerXUID, kv.KillerGamertag, kv.VictimXUID, kv.VictimGamertag, kv.TimeMS, now,
		); err != nil {
			return fmt.Errorf("persist: EventsCompletion insert killer_victim_pairs %s: %w", matchID, err)
		}
	}
	return nil
}

// killerVictimRowsFromCompletion adapte la forme de la complétion à celle du pipeline
// primaire. Adaptation de FORME seulement : la traduction vers les morts de
// `match_kill_events` reste unique (CreditKillEventsFromPairs), sans quoi les trois états de
// l'assistant auraient deux chances d'être confondus.
func killerVictimRowsFromCompletion(matchID string, pairs []KVPairCompletion) []KillerVictimInsert {
	rows := make([]KillerVictimInsert, 0, len(pairs))
	for _, kv := range pairs {
		rows = append(rows, KillerVictimInsert{
			MatchID:        matchID,
			KillerXUID:     kv.KillerXUID,
			KillerGamertag: kv.KillerGamertag,
			VictimXUID:     kv.VictimXUID,
			VictimGamertag: kv.VictimGamertag,
			Count:          1,
			TimeMS:         kv.TimeMS,
		})
	}
	return rows
}

// markCompletionRegistry positionne les bits de complétude + events_loaded en un
// seul UPDATE (pattern bitmask existant, cf. sync.MarkEventsLoaded).
func markCompletionRegistry(ctx context.Context, tx *sql.Tx, in EventsCompletionInput, eventsInserted int) error {
	var mask int64
	if eventsInserted > 0 {
		mask |= in.EventsBit
	}
	if in.MarkKV {
		mask |= in.KillerVictimBit
	}
	eventsLoaded := eventsInserted > 0
	if mask == 0 && !eventsLoaded {
		return nil // rien à marquer
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE match_registry
		SET backfill_completed = COALESCE(backfill_completed, 0) | ?,
		    events_loaded = events_loaded OR ?
		WHERE match_id = ?`, mask, eventsLoaded, in.MatchID); err != nil {
		return fmt.Errorf("persist: EventsCompletion update match_registry %s: %w", in.MatchID, err)
	}
	return nil
}
