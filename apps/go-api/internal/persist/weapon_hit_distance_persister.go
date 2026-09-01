// Package persist — weapon_hit_distance_persister.go : ecriture INSERT-ONLY du NUMERATEUR film
// d Infinite (precision par arme + distance des touches), produit par une PASSE DE DECODAGE de
// film. Deux tables, une seule passe, un seul lease :
//
//	weapon_accuracy            (partagee, grain match x xuid x arme) : shots_fired = tirs
//	                           appariables, shots_landed = touches, drops = 0. MEME table que H5.
//	match_weapon_hit_distance  (soeur, meme grain) : histogramme des distances des touches +
//	                           effectif dist_n. Append-only (decode_pass + vue _latest).
//
// POURQUOI UN PERSISTER DEDIE, ET PAS LE CHEMIN BatchBuilder (persistWeaponAccuracy) : meme raison
// que WeaponShotsPersister. persistWeaponAccuracy n ecrit que si batch.Shared.Match != nil (le sync
// PRIMAIRE d un match) ; or une passe de FILM arrive un cycle plus tard, sur un match DEJA insere —
// SharedPersister y serait un no-op. Le numerateur film emprunte donc ce chemin direct, sous le
// lease RW de shared. La forme SQL du INSERT weapon_accuracy est celle de persistWeaponAccuracy
// (une seule regle d ecriture de cette table).
//
// IDEMPOTENCE (le point delicat) :
//   - match_weapon_hit_distance est APPEND-ONLY : re-executer la passe ecrit une nouvelle
//     generation (nouveau decode_pass), la vue _latest retient la derniere. Aucun doublon logique.
//   - weapon_accuracy N A NI decode_pass NI vue _latest (table partagee figee cote H5). Pour
//     Infinite elle n est peuplee QUE par cette passe film : un re-run (backfill) DOUBLERAIT donc
//     les lignes. GARDE SELECT-then-INSERT (pattern legacy documente CLAUDE.md) : si le match a
//     deja des lignes weapon_accuracy, la passe NE reecrit PAS l accuracy (la distance, elle, se
//     regenere par decode_pass). ART-safe : un SELECT puis des INSERT, jamais d UPDATE/DELETE.
//
// ANTI-ART (ADR 0019/0026/0030) : INSERT purs, aucun DELETE, aucun UPDATE, aucun ON CONFLICT —
// rien a faire figurer dans l allowlist de no_art_patterns_test.go.
//
// PRE-REQUIS : le caller tient le lease RW sur shared_matches_v2.duckdb (ADR 0013).

package persist

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"levelup/go-api/internal/migration"
)

// EvaluateHitsGate est LA porte d effectif du numerateur film, et sa copie UNIQUE. Une forme par
// arme (taux d accuracy sur ShotsPaired, histogramme de distance sur dist_n) n est SENSEE que si
// son effectif atteint le minimum mesure (migration.WeaponHitsMinShots = 8). Sous ce seuil, une
// poignee d observations n est que du bruit.
//
// USAGE : la ligne weapon_accuracy n est ECRITE que si EvaluateHitsGate(shots_fired) — une arme a
// 1-4 tirs ne fabrique pas de faux « 100 % ». Pour la distance, la table STOCKE tout (dist_n brut)
// et c est le LECTEUR (vue b, Lot 5) qui applique EvaluateHitsGate(dist_n) — le meme predicat.
func EvaluateHitsGate(effectif int) bool {
	return effectif >= migration.WeaponHitsMinShots
}

// WeaponHitDistancePersister ecrit le numerateur film (accuracy + distance) dans shared.
type WeaponHitDistancePersister struct {
	db txBeginner
}

// NewWeaponHitDistancePersister construit un persister. `db` doit tenir le lease RW sur shared.
func NewWeaponHitDistancePersister(db txBeginner) *WeaponHitDistancePersister {
	return &WeaponHitDistancePersister{db: db}
}

// PersistPass ecrit une passe (accuracy + distance) en 1 transaction, INSERT purs.
//
// `accuracy` porte shots_fired = ShotsPaired ; les lignes sous la porte d effectif sont ecartees.
// `distance` porte l histogramme brut (le lecteur tranche la publiabilite sur dist_n). Une passe
// entierement vide est ignoree et LOGGUEE (ecrire zero ligne serait indistinguable d un match sans
// tir, et la vue _latest continue de servir la passe precedente).
func (p *WeaponHitDistancePersister) PersistPass(
	ctx context.Context, accuracy []WeaponAccuracyInsert, distance WeaponHitDistanceBatch,
) error {
	if err := validateWeaponHitDistanceBatch(distance); err != nil {
		return err
	}
	if len(accuracy) == 0 && len(distance.Rows) == 0 {
		slog.WarnContext(ctx, "persist: passe numerateur film vide, aucune ligne ecrite",
			"match_id", distance.MatchID, "decoder_rev", distance.DecoderRev)
		return nil
	}

	pass, err := newDecodePassID()
	if err != nil {
		return fmt.Errorf("persist: %s: %w", distance.MatchID, err)
	}

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("persist: BeginTx numerateur film %s: %w", distance.MatchID, err)
	}
	defer func() { _ = tx.Rollback() }() // no-op apres Commit

	accWritten, accSkipped, err := p.insertAccuracy(ctx, tx, distance.MatchID, accuracy)
	if err != nil {
		return err
	}
	distWritten, err := p.insertDistance(ctx, tx, distance, pass, time.Now().UTC())
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("persist: Commit numerateur film %s: %w", distance.MatchID, err)
	}
	slog.InfoContext(ctx, "persist: numerateur film ecrit",
		"match_id", distance.MatchID, "decoder_rev", distance.DecoderRev,
		"accuracy_ecrites", accWritten, "accuracy_sous_porte", accSkipped,
		"distance_ecrites", distWritten)
	return nil
}

// insertAccuracy ecrit les lignes weapon_accuracy passant la porte d effectif, SAUF si le match a
// deja des lignes (garde d idempotence — voir l en-tete). Rend (ecrites, ecartees_sous_porte).
func (p *WeaponHitDistancePersister) insertAccuracy(
	ctx context.Context, tx *sql.Tx, matchID string, rows []WeaponAccuracyInsert,
) (int, int, error) {
	if len(rows) == 0 {
		return 0, 0, nil
	}
	var existing int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM weapon_accuracy WHERE match_id = ?`, matchID).Scan(&existing); err != nil {
		return 0, 0, fmt.Errorf("persist: garde weapon_accuracy %s: %w", matchID, err)
	}
	if existing > 0 {
		// Deja peuple (passe anterieure) : ne pas doubler. La distance, elle, se regenere par
		// decode_pass ci-dessous — les deux tables ont des regimes d idempotence differents.
		slog.InfoContext(ctx, "persist: weapon_accuracy deja peuple pour ce match, accuracy non reecrite",
			"match_id", matchID, "lignes_existantes", existing)
		return 0, 0, nil
	}
	written, skipped := 0, 0
	for _, r := range rows {
		if !EvaluateHitsGate(r.ShotsFired) {
			skipped++
			continue
		}
		// weapon_id en CHAINE DECIMALE (piege ubigintArg : un id filmshell dont le bit de poids
		// fort vaut 1 serait NEGATIF en int64 signe et le CAST AS UBIGINT echouerait).
		if _, err := tx.ExecContext(ctx, insertWeaponAccuracyFilmSQL,
			r.MatchID, r.XUID, strconv.FormatUint(r.WeaponID, 10),
			int64(r.ShotsFired), int64(r.ShotsLanded), int64(r.Drops)); err != nil {
			return 0, 0, fmt.Errorf("persist: INSERT weapon_accuracy %s/%s/%d: %w",
				r.MatchID, r.XUID, r.WeaponID, err)
		}
		written++
	}
	return written, skipped, nil
}

// insertDistance ecrit les lignes match_weapon_hit_distance de la passe (append-only). Rend le
// nombre de lignes ecrites.
func (p *WeaponHitDistancePersister) insertDistance(
	ctx context.Context, tx *sql.Tx, in WeaponHitDistanceBatch, pass string, now time.Time,
) (int, error) {
	if len(in.Rows) == 0 {
		return 0, nil
	}
	stmt, err := tx.PrepareContext(ctx, insertWeaponHitDistanceSQL)
	if err != nil {
		return 0, fmt.Errorf("persist: prepare match_weapon_hit_distance %s: %w", in.MatchID, err)
	}
	defer func() { _ = stmt.Close() }()
	for _, r := range in.Rows {
		if _, err := stmt.ExecContext(ctx,
			in.MatchID, pass, in.DecoderRev, now,
			r.XUID, strconv.FormatUint(r.WeaponID, 10), r.DistBucketJSON, int64(r.DistN)); err != nil {
			return 0, fmt.Errorf("persist: INSERT match_weapon_hit_distance %s/%s/%d: %w",
				in.MatchID, r.XUID, r.WeaponID, err)
		}
	}
	return len(in.Rows), nil
}

const insertWeaponAccuracyFilmSQL = `
	INSERT INTO weapon_accuracy
		(match_id, xuid, weapon_id, shots_fired, shots_landed, drops)
	VALUES (?, ?, CAST(? AS UBIGINT), ?, ?, ?)`

const insertWeaponHitDistanceSQL = `
	INSERT INTO match_weapon_hit_distance
		(match_id, decode_pass, decoder_rev, written_at, xuid, weapon_id, dist_bucket_json, dist_n)
	VALUES (?, ?, ?, ?, ?, CAST(? AS UBIGINT), ?, CAST(? AS INTEGER))`

// validateWeaponHitDistanceBatch : ce que le persister REFUSE au niveau de la passe. La validation
// passe AVANT la transaction : un refus ne laisse aucune ligne derriere lui.
func validateWeaponHitDistanceBatch(in WeaponHitDistanceBatch) error {
	if in.MatchID == "" {
		return errors.New("persist: WeaponHitDistanceBatch.MatchID vide")
	}
	if in.DecoderRev == "" {
		return fmt.Errorf("persist: WeaponHitDistanceBatch.DecoderRev vide (%s) — "+
			"une passe doit dire quel numerateur l a produite", in.MatchID)
	}
	seen := make(map[[2]string]bool)
	for _, r := range in.Rows {
		if r.XUID == "" {
			return fmt.Errorf("persist: %s ligne distance sans xuid — la distance exige un "+
				"joueur resolu (deux positions monde)", in.MatchID)
		}
		if r.WeaponID == 0 {
			return fmt.Errorf("persist: %s arme nulle en distance (xuid %s)", in.MatchID, r.XUID)
		}
		if r.DistN <= 0 {
			return fmt.Errorf("persist: %s dist_n <= 0 (xuid %s, arme %d) — une ligne distance "+
				"sans touche resolue n est pas une mesure", in.MatchID, r.XUID, r.WeaponID)
		}
		key := [2]string{r.XUID, strconv.FormatUint(r.WeaponID, 10)}
		if seen[key] {
			return fmt.Errorf("persist: %s doublon (xuid %s, arme %d) dans la meme passe distance",
				in.MatchID, r.XUID, r.WeaponID)
		}
		seen[key] = true
	}
	return nil
}
