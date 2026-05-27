// Package duckdb — pool_shared_social_recovery.go : ouverture de
// shared_social.duckdb avec récupération automatique d'un WAL orphelin
// non-rejouable (bug DuckDB upstream #7659).
//
// Contexte :
//
// Quand un ATTACH/DDL legacy s'écrit dans shared_social.duckdb.wal sans
// CHECKPOINT (cf. ADR 0016 + crash brutal Windows SIGKILL), le header de
// la DB principale marque "WAL replay needed" mais le replay échoue avec :
//
//	INTERNAL Error: Failure while replaying WAL file "...wal":
//	Calling DatabaseManager::GetDefaultDatabase with no default database set
//
// Cette erreur est non récupérable par DuckDB lui-même : le fichier reste
// définitivement inouvrable en RW jusqu'à intervention manuelle.
//
// Stratégie de récupération :
//
//  1. Tenter OpenReadWriteShared.
//  2. Si l'erreur correspond au pattern WAL replay → renommer le .wal
//     en .wal.orphan-<RFC3339-utc> (atomique sur NTFS via os.Rename).
//  3. Réessayer OpenReadWriteShared UNE seule fois.
//  4. Si l'erreur persiste → retourner nil pour dégradation graceful
//     (le comportement existant : socialDB=nil → galerie média vide
//     plutôt que crash du pool).
//
// Limites connues :
//
//   - Si la corruption est dans le HEADER du .duckdb (pas seulement le .wal),
//     le rename ne suffit pas : il faut un EXPORT/IMPORT via cmd/rebuild_shared_social.
//     Ce cas reste détectable (la 2e tentative échoue) et émet un slog.ErrorContext
//     avec un pointeur vers le runbook.
//   - Pas de boucle : un seul retry, on accepte la dégradation si ça refoire.
//
// Sentinelle : expvar.Int "levelup.wal_orphan_quarantine.shared_social"
// est incrémenté à chaque quarantaine pour permettre l'alerting.
package duckdb

import (
	"context"
	"errors"
	"expvar"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

// errWALReplayMarker est l'extrait du message d'erreur DuckDB upstream
// (bug #7659) qui signale un WAL non-rejouable. La détection se fait par
// strings.Contains pour rester robuste aux variations de wrapping
// (database/sql/driver wrappe le message dans son propre format).
const errWALReplayMarker = "Failure while replaying WAL file"

// metricWALOrphanQuarantineSocial compte les quarantaines effectuées pour
// shared_social. Visible via /debug/vars (ADR 0009).
var metricWALOrphanQuarantineSocial = expvar.NewInt("levelup.wal_orphan_quarantine.shared_social")

// metricSharedSocialOpenFailures compte les échecs d'ouverture de
// shared_social.duckdb, segmentés par cause (Phase 5.2 ADR 0021).
//
// Clés observées en pratique :
//   - "wal_replay" : bug DuckDB #7659 (le plus courant ; quarantine déclenchée)
//   - "wal_replay_after_quarantine" : retry échoue malgré quarantaine
//     (corruption .duckdb header — runbook rebuild)
//   - "other" : tout autre échec (permission, lock, fichier absent, etc.)
//
// Format expvar Map → exposé en JSON `{cause: count}` sur /debug/vars.
var metricSharedSocialOpenFailures = expvar.NewMap("levelup.shared_social.open_failures")

// metricSharedSocialCheckpointMs trace la durée du dernier CHECKPOINT
// shared_social (Phase 5.2). Utile pour détecter une dégradation
// progressive (CHECKPOINT > 1s = signe d'un WAL qui grossit anormalement).
//
// Note : expvar n'a pas d'histogram natif. On expose la dernière valeur
// (Float). Un wrapper de monitoring (Prometheus exporter) peut tracker
// les variations dans le temps.
var metricSharedSocialCheckpointMs = expvar.NewFloat("levelup.shared_social.checkpoint_duration_ms")

// openSharedSocialFn est l'opener injecté — pointé sur OpenReadWriteShared
// en production, surchargeable en test pour simuler une corruption WAL
// déterministiquement (cf. pool_shared_social_recovery_test.go).
//
// Pattern var-of-func plutôt que paramètre explicite pour préserver la
// signature publique simple de openSharedSocialWithWALRecovery.
var openSharedSocialFn = OpenReadWriteShared

// openSharedSocialWithWALRecovery ouvre shared_social.duckdb avec
// récupération auto en cas de WAL orphelin. Retourne (db, nil) en succès,
// (nil, nil) en cas de dégradation graceful (path vide, fichier absent,
// ou recovery échouée — tous non-bloquants pour le pool). (nil, err)
// uniquement pour les erreurs vraiment fatales (jamais aujourd'hui).
//
// Cette fonction est extraite du bloc d'ouverture historique
// (pool.go:252-269) pour permettre des tests unitaires indépendants.
func openSharedSocialWithWALRecovery(ctx context.Context, path, timezone, gamertag string) *DB {
	if path == "" {
		return nil
	}

	db, err := openSharedSocialFn(path, timezone)
	if err == nil {
		return db
	}

	// Cas non-WAL-replay : dégradation classique, log Warn (comportement
	// pré-2026-05-27 préservé). Aucune action corrective tentée.
	if !isWALReplayFailure(err) {
		metricSharedSocialOpenFailures.Add("other", 1)
		slog.WarnContext(ctx, "pool: ouverture SharedSocial échouée (dégradation: socialDB=nil)",
			"path", path, "gamertag", gamertag, "err", err)
		return nil
	}
	metricSharedSocialOpenFailures.Add("wal_replay", 1)

	// Cas WAL replay : tenter la quarantaine + retry.
	walPath := path + ".wal"
	quarantinePath, walSize, renameErr := quarantineOrphanWAL(walPath)
	if renameErr != nil {
		// Le rename a échoué (lock antivirus, permissions, etc.). On loggue
		// en Error et on dégrade comme avant — sans pollution supplémentaire.
		slog.ErrorContext(ctx, "pool: SharedSocial WAL non-rejouable + quarantaine échouée (dégradation: socialDB=nil)",
			"path", path, "wal_path", walPath, "gamertag", gamertag,
			"original_err", err, "quarantine_err", renameErr)
		return nil
	}

	metricWALOrphanQuarantineSocial.Add(1)
	slog.ErrorContext(ctx, "pool: SharedSocial WAL orphelin mis en quarantaine — retry open",
		"path", path, "quarantine_path", quarantinePath, "wal_size_bytes", walSize,
		"gamertag", gamertag, "original_err", err)

	// Retry une seule fois. Si le .duckdb header est aussi corrompu (bug DuckDB
	// upstream #7659 cas extrême), cette tentative échouera et on dégradera
	// vers socialDB=nil + log Error pour pointer vers le runbook rebuild.
	db, err = openSharedSocialFn(path, timezone)
	if err == nil {
		slog.InfoContext(ctx, "pool: SharedSocial récupéré après quarantaine WAL",
			"path", path, "quarantine_path", quarantinePath, "gamertag", gamertag)
		return db
	}

	metricSharedSocialOpenFailures.Add("wal_replay_after_quarantine", 1)
	slog.ErrorContext(ctx, "pool: SharedSocial reste inouvrable après quarantaine WAL — corruption probable dans le fichier principal, voir runbook cmd/rebuild_shared_social",
		"path", path, "quarantine_path", quarantinePath, "gamertag", gamertag,
		"retry_err", err)
	return nil
}

// isWALReplayFailure détecte le pattern d'erreur DuckDB upstream #7659.
// Utilise strings.Contains pour robustesse aux wrappings (database/sql/driver).
// Couvert par pool_shared_social_recovery_test.go.
func isWALReplayFailure(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), errWALReplayMarker)
}

// CheckpointSharedSocial exécute CHECKPOINT sur la conn shared_social.
//
// Doit être appelé après chaque écriture qui ne passe pas par SocialPersister
// (qui CHECKPOINT déjà). Garantit que les writes runtime sont flushées sur
// disque immédiatement → fenêtre d'exposition au bug WAL orphelin = 0 sec
// (au lieu de 5 min — scheduler périodique dans cmd/server/main.go:614-637).
//
// Erreur retournée mais loguée systématiquement en ErrorContext : le caller
// peut choisir de la propager ou de continuer selon la criticité (DDL → propager,
// write isolé → continuer + accepter la fenêtre 5min de fallback scheduler).
//
// Phase 5.2 : met à jour la métrique expvar `checkpoint_duration_ms` pour
// monitoring (détection de WAL qui grossit anormalement).
//
// Couvert par pool_shared_social_recovery_test.go.
func CheckpointSharedSocial(ctx context.Context, db *DB) error {
	if db == nil {
		return nil
	}
	start := time.Now()
	_, err := db.Exec(ctx, "CHECKPOINT")
	metricSharedSocialCheckpointMs.Set(float64(time.Since(start).Milliseconds()))
	if err != nil {
		slog.ErrorContext(ctx, "shared_social: CHECKPOINT échoué",
			"path", db.Path(), "err", err)
		return fmt.Errorf("shared_social CHECKPOINT: %w", err)
	}
	return nil
}

// quarantineOrphanWAL renomme atomiquement <path>.wal en
// <path>.wal.orphan-<RFC3339-utc>. Retourne (chemin de quarantaine,
// taille du .wal originel, nil) en succès.
//
// Si le .wal n'existe pas (cas où le header .duckdb seul est corrompu),
// retourne ("", 0, nil) — le retry de l'ouverture est tenté quand même
// car DuckDB peut parfois s'en sortir sans le .wal.
//
// Couvert par pool_shared_social_recovery_test.go.
func quarantineOrphanWAL(walPath string) (string, int64, error) {
	stat, err := os.Stat(walPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Pas de .wal physique → corruption probablement dans le .duckdb
			// header. Pas de quarantaine à faire, mais on autorise le retry.
			return "", 0, nil
		}
		return "", 0, fmt.Errorf("stat wal: %w", err)
	}
	size := stat.Size()

	// Timestamp compatible avec un nom de fichier sur tous les FS (NTFS, ext4, APFS).
	ts := time.Now().UTC().Format("20060102-150405Z")
	quarantinePath := walPath + ".orphan-" + ts

	if err := os.Rename(walPath, quarantinePath); err != nil {
		return "", size, fmt.Errorf("rename wal -> quarantine: %w", err)
	}
	return quarantinePath, size, nil
}
