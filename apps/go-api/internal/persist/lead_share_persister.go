// Package persist — lead_share_persister.go : écriture INSERT-ONLY de LA MAINMISE d'un match (temps
// passé en tête du score par camp, retournements), calculée de la courbe de score de l'artefact
// de rejeu.
//
// Une destination, une transaction :
//
//	shared.match_lead_share   une ligne par camp publié + UNE ligne « match » par passe (append-only,
//	                          vue match_lead_share_latest — ADR 0026)
//
// # LA LIGNE « MATCH », ET POURQUOI ELLE EXISTE
//
// Une passe s'écrit TOUJOURS, même refusée : la ligne `match` porte la couverture (mesurée ou non,
// et la raison du refus) et fait que la vue `_latest` retient la passe entière. Sans elle, un match
// dont la re-projection est devenue un refus servirait à jamais les lignes de la passe précédente
// (l'arbitrage par clé, piège ADR 0026), et un match refusé serait indistinguable d'un match que
// personne n'a jamais regardé. « Non mesuré » ne se confond jamais avec zéro.
//
// Les colonnes du match (durée mesurée, égalité, retournements, camps muets, couverture) sont
// écrites sur CHAQUE ligne de la passe : toute ligne se lit seule.
//
// # ANTI-ART (ADR 0019/0026/0030)
//
// INSERT purs. Aucun DELETE, aucun UPDATE, aucun ON CONFLICT : rien à figurer dans l'allowlist de
// `no_art_patterns_test.go`, où `match_lead_share` entre au contraire dans les tables PROTÉGÉES
// (ainsi que dans `sync/append_only_state_guard_test.go`, la regexp de
// `duckdb/no_raw_rating_reads_test.go` et le registre de compaction).
//
// PRÉ-REQUIS : le caller doit tenir le lease RW sur shared_matches_v2.duckdb.

package persist

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Les deux natures de ligne de `match_lead_share`. Valeurs de DONNÉE, jamais des libellés.
const (
	LeadShareRowKindCamp  = "camp"
	LeadShareRowKindMatch = "match"
)

// LeadShareRow : le temps passé SEUL en tête par UN camp.
type LeadShareRow struct {
	// Camp : identifiant d'équipe du registre (0 à 8).
	Camp int `json:"camp"`
	// LeadMS : temps passé seul en tête, en millisecondes de la partie jouée.
	LeadMS int64 `json:"lead_ms"`
}

// LeadShareBatch est LE RÉSULTAT D'UNE PASSE DE LECTURE D'UN ARTEFACT, côté Mainmise. L'unité de
// production est le MATCH ENTIER.
type LeadShareBatch struct {
	MatchID string `json:"match_id"`
	// Measured faux = « Mainmise non mesurée » : Reason dit pourquoi, aucune ligne de camp.
	Measured bool   `json:"measured"`
	Reason   string `json:"reason,omitempty"`
	// DocSchema : version de schéma de l'artefact lu (diagnostic).
	DocSchema int `json:"doc_schema"`
	// MeasuredMS : durée de la partie jouée ; TieMS : temps sans meneur ; LeadChanges : retournements ;
	// SilentCamps : camps du roster sans série de score (concourent à zéro, ne mènent jamais).
	MeasuredMS  int64 `json:"measured_ms"`
	TieMS       int64 `json:"tie_ms"`
	LeadChanges int   `json:"lead_changes"`
	SilentCamps int   `json:"silent_camps"`
	// Rows : un camp par série publiée. Vide quand Measured est faux.
	Rows []LeadShareRow `json:"rows,omitempty"`
}

// LeadSharePersister écrit une passe de la Mainmise dans shared_matches_v2.duckdb.
type LeadSharePersister struct {
	db txBeginner
}

// NewLeadSharePersister construit un persister. `db` doit tenir le lease RW sur shared.
func NewLeadSharePersister(db txBeginner) *LeadSharePersister {
	return &LeadSharePersister{db: db}
}

// Persist écrit le sous-batch `batch.Shared.LeadShare` s'il existe. No-op sinon. Chemin
// BatchBuilder ; le chemin direct (lecture tardive d'un artefact, backfill) passe par
// [LeadSharePersister.PersistPass].
func (p *LeadSharePersister) Persist(ctx context.Context, batch *MatchBatch) error {
	if batch == nil {
		return errors.New("persist: LeadSharePersister.Persist: batch nil")
	}
	if batch.Shared.LeadShare == nil {
		return nil
	}
	return p.PersistPass(ctx, *batch.Shared.LeadShare)
}

const insertLeadShareSQL = `
	INSERT INTO match_lead_share (
		match_id, decode_pass, written_at, row_kind, camp, lead_ms,
		measured, unmeasured_reason, doc_schema, measured_ms, tie_ms, lead_changes, silent_camps
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

// PersistPass écrit UNE passe en 1 transaction, en INSERT purs : la ligne match, puis les camps.
func (p *LeadSharePersister) PersistPass(ctx context.Context, in LeadShareBatch) error {
	if err := ValidateLeadShareBatch(in); err != nil {
		return err
	}
	// Le tirage AVANT la transaction : un decode_pass non distinguable ferait rendre à la vue
	// _latest un MÉLANGE de passes (cf. newDecodePassID).
	pass, err := newDecodePassID()
	if err != nil {
		return err
	}
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("persist: BeginTx match_lead_share %s: %w", in.MatchID, err)
	}
	defer func() { _ = tx.Rollback() }() // no-op après Commit

	stmt, err := tx.PrepareContext(ctx, insertLeadShareSQL)
	if err != nil {
		return fmt.Errorf("persist: prepare match_lead_share %s: %w", in.MatchID, err)
	}
	defer func() { _ = stmt.Close() }()

	now := time.Now().UTC()
	all := make([]LeadShareRow, 0, len(in.Rows)+1)
	all = append(all, LeadShareRow{Camp: -1}) // la ligne match : camp -1
	all = append(all, in.Rows...)
	for i, r := range all {
		kind := LeadShareRowKindCamp
		if i == 0 {
			kind = LeadShareRowKindMatch
		}
		if _, err := stmt.ExecContext(ctx,
			in.MatchID, pass, now, kind, r.Camp, r.LeadMS,
			in.Measured, in.Reason, in.DocSchema, in.MeasuredMS, in.TieMS, in.LeadChanges, in.SilentCamps,
		); err != nil {
			return fmt.Errorf("persist: INSERT match_lead_share %s/%s/%d: %w", in.MatchID, kind, r.Camp, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("persist: Commit match_lead_share %s: %w", in.MatchID, err)
	}
	return nil
}

// ValidateLeadShareBatch : ce que le persister REFUSE. EXPOSÉE pour que le projecteur prouve sa
// sortie sans ouvrir de base. La validation passe AVANT la transaction.
//
//	(1) un match sans identifiant ;
//	(2) une passe « non mesurée » sans raison, avec des lignes ou avec des temps non nuls ;
//	    une passe mesurée avec une raison ;
//	(3) une passe mesurée sans durée, avec un compte négatif, ou sans aucun camp ;
//	(4) une ligne de camp hors 0..8, au temps négatif, ou un DOUBLON de camp ;
//	(5) une somme des temps (égalité + camps) qui n'est pas la durée mesurée : la lecture d'une
//	    part divisée par cette durée n'aurait plus de sens.
func ValidateLeadShareBatch(in LeadShareBatch) error {
	if in.MatchID == "" {
		return errors.New("persist: LeadShareBatch.MatchID vide")
	}
	if !in.Measured {
		return validateLeadShareUnmeasured(in)
	}
	switch {
	case in.Reason != "":
		return fmt.Errorf("persist: %s: passe mesurée avec une raison de non mesure %q", in.MatchID, in.Reason)
	case in.MeasuredMS <= 0:
		return fmt.Errorf("persist: %s: durée mesurée %d ms", in.MatchID, in.MeasuredMS)
	case in.TieMS < 0 || in.LeadChanges < 0 || in.SilentCamps < 0:
		return fmt.Errorf("persist: %s: compte négatif", in.MatchID)
	case len(in.Rows) == 0:
		return fmt.Errorf("persist: %s: passe mesurée sans aucun camp", in.MatchID)
	}
	total := in.TieMS
	vus := make(map[int]bool, len(in.Rows))
	for i, r := range in.Rows {
		if r.Camp < 0 || r.Camp > 8 {
			return fmt.Errorf("persist: %s ligne #%d: camp %d hors 0..8", in.MatchID, i, r.Camp)
		}
		if r.LeadMS < 0 {
			return fmt.Errorf("persist: %s ligne #%d: temps en tête négatif", in.MatchID, i)
		}
		if vus[r.Camp] {
			return fmt.Errorf("persist: %s: doublon du camp %d dans la même passe", in.MatchID, r.Camp)
		}
		vus[r.Camp] = true
		total += r.LeadMS
	}
	if total != in.MeasuredMS {
		return fmt.Errorf("persist: %s: la somme des temps (%d ms) n'est pas la durée mesurée (%d ms)",
			in.MatchID, total, in.MeasuredMS)
	}
	return nil
}

// validateLeadShareUnmeasured : le contrôle (2) d'une passe non mesurée.
func validateLeadShareUnmeasured(in LeadShareBatch) error {
	switch {
	case in.Reason == "":
		return fmt.Errorf("persist: %s: passe non mesurée sans raison — dire pourquoi est le contrat", in.MatchID)
	case len(in.Rows) > 0:
		return fmt.Errorf("persist: %s: passe non mesurée avec des lignes de camp", in.MatchID)
	case in.MeasuredMS != 0 || in.TieMS != 0 || in.LeadChanges != 0 || in.SilentCamps != 0:
		return fmt.Errorf("persist: %s: passe non mesurée avec des temps non nuls", in.MatchID)
	}
	return nil
}
