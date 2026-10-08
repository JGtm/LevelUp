// Package persist — life_placement_persister.go : ecriture INSERT-ONLY de `match_life_placement`,
// le placement et le rendement de chaque vie nommee (plan `.ai/V7.5/PLAN_EMPRISE_VIES_2026-09-28.md`,
// lot V2).
//
// MEME PATRON QUE [LivesPersister], ET POUR LA MEME RAISON : la table nait d'une passe de
// decodage d'un film, sur un match DEJA insere au registre. Le chemin builder serait un no-op.
//
// ANTI-ART (ADR 0019/0026/0030) : INSERT purs, aucun UPDATE, aucun DELETE, aucun ON CONFLICT.
// « Remplacer » une passe consiste a en ecrire une nouvelle sous un `decode_pass` neuf ; la vue
// `match_life_placement_latest` ne rend que la DERNIERE PASSE PAR MATCH. Rien a inscrire dans
// l'allowlist de `internal/sync/no_art_patterns_test.go`.
//
// UN PERSISTER SEPARE DE CELUI DES VIES, PAS UNE TROISIEME TABLE DANS SA TRANSACTION : le
// placement s'ecrit APRES les vies et sous son propre lease, pour que son echec ne coute jamais
// les vies (meme doctrine que les faits d'isolement face aux positions). Les deux passes ont donc
// deux `decode_pass` differents ; ce qui les joint est la vie, `(match_id, xuid, start_ms)`.
//
// PRE-REQUIS : le caller tient le lease RW sur shared_matches_v2.duckdb.
package persist

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

// LifePlacementBatch porte UNE passe de placement : une ligne par vie nommee du match.
type LifePlacementBatch struct {
	MatchID string
	// DecoderRev : la revision du placement (`PlacementRev` du collecteur). Requise — c'est la
	// seule facon de savoir quels matchs recalculer apres un changement de la mesure.
	DecoderRev string
	Rows       []LifePlacementInsert
}

// LifePlacementInsert — la mesure d'une vie, sur l'horloge du MATCH (decision V3 du plan).
type LifePlacementInsert struct {
	XUID       string `json:"xuid"`
	StartMS    int64  `json:"start_ms"`
	EndMS      int64  `json:"end_ms"`
	DurationMS int64  `json:"duration_ms"`
	MeasuredMS int64  `json:"measured_ms"`
	// MedianM : nil = vie non mesuree (temps mesure sous le minimum du calcul).
	MedianM *float64 `json:"median_m,omitempty"`
	// BeyondMS et RadarM : nil ENSEMBLE = variante sans portee de radar connue.
	BeyondMS           *int64   `json:"beyond_ms,omitempty"`
	RadarM             *float64 `json:"radar_m,omitempty"`
	CarrierMS          int64    `json:"carrier_ms"`
	TeamDownMS         int64    `json:"team_down_ms"`
	UnplacedMS         int64    `json:"unplaced_ms"`
	TeammateUnplacedMS int64    `json:"teammate_unplaced_ms"`
	Kills              int      `json:"kills"`
}

// LifePlacementPersister ecrit une passe de placement dans shared_matches_v2.duckdb.
type LifePlacementPersister struct {
	db txBeginner
}

// NewLifePlacementPersister construit un persister. `db` doit tenir le lease RW sur shared.
func NewLifePlacementPersister(db txBeginner) *LifePlacementPersister {
	return &LifePlacementPersister{db: db}
}

// PersistPass ecrit la passe d'UN match en 1 transaction, en INSERT purs. Toutes les lignes
// portent le MEME `decode_pass` et le MEME `written_at`.
//
// UNE PASSE VIDE N'ECRIT RIEN : sans ligne, il n'y a pas de passe a retenir, et une passe sans
// ligne ne se distingue pas en base d'un match jamais mesure. Elle est journalisee.
func (p *LifePlacementPersister) PersistPass(ctx context.Context, in LifePlacementBatch) error {
	if err := validateLifePlacementBatch(in); err != nil {
		return err
	}
	if len(in.Rows) == 0 {
		slog.WarnContext(ctx, "persist: passe de placement vide, aucune ligne ecrite",
			"match_id", in.MatchID, "decoder_rev", in.DecoderRev)
		return nil
	}
	pass, err := newDecodePassID()
	if err != nil {
		return fmt.Errorf("persist: %s: %w", in.MatchID, err)
	}
	return p.insertPlacementPass(ctx, in, pass, time.Now().UTC())
}

// insertPlacementPass ouvre la transaction et ecrit les lignes.
func (p *LifePlacementPersister) insertPlacementPass(
	ctx context.Context, in LifePlacementBatch, pass string, now time.Time,
) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("persist: BeginTx match_life_placement %s: %w", in.MatchID, err)
	}
	defer func() { _ = tx.Rollback() }() // no-op apres Commit

	for i := range in.Rows {
		if err := insertPlacementRow(ctx, tx, in, &in.Rows[i], pass, now); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("persist: Commit match_life_placement %s: %w", in.MatchID, err)
	}
	return nil
}

// insertPlacementRow ecrit une ligne. INSERT pur, table append-only.
func insertPlacementRow(
	ctx context.Context, tx *sql.Tx, in LifePlacementBatch, r *LifePlacementInsert, pass string, now time.Time,
) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO match_life_placement (
			match_id, decode_pass, decoder_rev, written_at,
			xuid, start_ms, end_ms, duration_ms, measured_ms, median_m, beyond_ms, radar_m,
			carrier_ms, team_down_ms, unplaced_ms, teammate_unplaced_ms, kills
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		in.MatchID, pass, in.DecoderRev, now,
		r.XUID, r.StartMS, r.EndMS, r.DurationMS, r.MeasuredMS,
		nullableFloat(r.MedianM), nullableInt64(r.BeyondMS), nullableFloat(r.RadarM),
		r.CarrierMS, r.TeamDownMS, r.UnplacedMS, r.TeammateUnplacedMS, r.Kills)
	if err != nil {
		return fmt.Errorf("persist: INSERT match_life_placement %s/%s/%d: %w",
			in.MatchID, r.XUID, r.StartMS, err)
	}
	return nil
}

// nullableFloat / nullableInt64 : un pointeur nil s'ecrit NULL, jamais zero.
func nullableFloat(v *float64) any {
	if v == nil {
		return nil
	}
	return *v
}

func nullableInt64(v *int64) any {
	if v == nil {
		return nil
	}
	return *v
}

// validateLifePlacementBatch refuse ce qui ne peut pas etre une passe.
//
// LES REFUS SONT DES ERREURS, PAS DES CORRECTIONS (meme doctrine que [validateLivesBatch]) : une
// ligne incoherente est le symptome d'un producteur casse, la corriger ici publierait une donnee
// inventee sous l'autorite de la base.
func validateLifePlacementBatch(in LifePlacementBatch) error {
	if in.MatchID == "" {
		return errors.New("persist: LifePlacementPersister.PersistPass: matchID vide")
	}
	if in.DecoderRev == "" {
		return fmt.Errorf("persist: LifePlacementBatch.DecoderRev vide (%s) — sans elle, aucun "+
			"recalcul cible n'est possible apres un changement de la mesure", in.MatchID)
	}
	for i := range in.Rows {
		if err := validatePlacementRow(&in.Rows[i]); err != nil {
			return fmt.Errorf("persist: match_life_placement %s ligne #%d: %w", in.MatchID, i, err)
		}
	}
	return nil
}

// validatePlacementRow : les invariants d'une ligne.
func validatePlacementRow(r *LifePlacementInsert) error {
	if r.XUID == "" {
		return errors.New("xuid vide — une vie anonyme ne s'ecrit pas")
	}
	if r.EndMS < r.StartMS || r.DurationMS != r.EndMS-r.StartMS {
		return fmt.Errorf("bornes incoherentes (debut %d, fin %d, duree %d)", r.StartMS, r.EndMS, r.DurationMS)
	}
	if err := validatePlacementCumuls(r); err != nil {
		return err
	}
	if r.MedianM != nil && (*r.MedianM < 0 || r.MeasuredMS == 0) {
		return fmt.Errorf("mediane %v sans instant mesure ou negative", *r.MedianM)
	}
	if (r.RadarM == nil) != (r.BeyondMS == nil) {
		return errors.New("portee du radar et temps hors radar vont ENSEMBLE (tous deux nuls sans portee)")
	}
	if r.RadarM != nil && (*r.RadarM <= 0 || *r.BeyondMS < 0 || *r.BeyondMS > r.MeasuredMS) {
		return fmt.Errorf("portee %v m / hors radar %d ms incoherents avec %d ms mesurees",
			*r.RadarM, *r.BeyondMS, r.MeasuredMS)
	}
	return nil
}

// validatePlacementCumuls : aucun cumul negatif, aucun frag negatif, et les cinq cumuls COUVRENT
// la vie — la grille est fermee, chaque instant recoit une cause : une somme inferieure a la duree
// cacherait des instants sans cause.
func validatePlacementCumuls(r *LifePlacementInsert) error {
	cumuls := []int64{r.MeasuredMS, r.CarrierMS, r.TeamDownMS, r.UnplacedMS, r.TeammateUnplacedMS}
	var somme int64
	for _, c := range cumuls {
		if c < 0 {
			return fmt.Errorf("cumul negatif (%v)", cumuls)
		}
		somme += c
	}
	if somme < r.DurationMS {
		return fmt.Errorf("les cinq cumuls (%d ms) ne couvrent pas la vie (%d ms)", somme, r.DurationMS)
	}
	if r.Kills < 0 {
		return fmt.Errorf("frags negatifs (%d)", r.Kills)
	}
	return nil
}
