// Package duckdb — prestige_social_recovery.go : helpers d'accès résilient à
// shared_social.duckdb (partagés par les repos de prestige_social_repo.go).
//
// Routage des lectures (V721-08b). shared_social.duckdb est tenue en RW par le
// serveur pour toute sa durée de vie ET écrite par le sync engine (post-sync).
// Tant que les lectures mono-ligne passaient par `db.QueryRow` PLAT, une
// invalidation FATAL DuckDB (bug ART) ou une handle périmée (« sql: database is
// closed » après un Close/Reopen concurrent) rendait ces lectures
// définitivement en erreur jusqu'au prochain restart — donc un 500 permanent
// sur le prestige du joueur (GET /prestige/user), le détail d'escouade et le
// détail d'un défi d'escouade (join / abandon / évaluation). Les variantes
// `*Recovered` (duckdb/db_query.go → WithReopenOnInvalidated) rouvrent et
// rejouent une fois. Même trou, même correctif que sur metadata.duckdb
// (V721-08.1, prestige_metadata_repo.go).
//
// La contention fichier (un AUTRE détenteur du fichier) est en plus traduite en
// dblease.ErrDBLocked par translateSocialLockErr → 503 + Retry-After côté
// handler, au lieu d'un 500 masqué non retryable.

package prestige

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/platform/dblease"
	"levelup/go-api/internal/platform/duckdb"
)

// execCheckpointed exécute une écriture mutative sur shared_social (avec reopen
// auto sur invalidation) PUIS flushe le WAL via CHECKPOINT immédiat (non-fatal).
//
// Les writes Prestige tournent déjà sous le lease KindSharedSocial (acquis par
// LazyPrestigeService pour le HTTP, tenu par le sync engine pour le post-sync) :
// le CHECKPOINT s'exécute donc sous le lease, sérialisé avec le sync engine.
// Sans lui, l'INSERT/UPDATE/DELETE reste dans le WAL et est perdu si la recovery
// #7659 quarantine un WAL orphelin au restart — même classe de bug que les
// mutations notifications (ADR 0022). CHECKPOINT non-fatal : la donnée est déjà
// commit, le scheduler 5 min fera fallback.
func execCheckpointed(ctx context.Context, db *duckdb.DB, query string, args ...any) error {
	if _, err := db.ExecRecovered(ctx, query, args...); err != nil {
		return err
	}
	_ = duckdb.CheckpointSharedSocial(ctx, db)
	return nil
}

// translateSocialLockErr traduit une erreur de contention fichier DuckDB (un
// AUTRE détenteur tient shared_social.duckdb : CLI backfill, 2e instance
// serveur, handle pas encore libérée après un hot-reload — cf.
// duckdb.IsFileLockError) en sentinelle dblease.ErrDBLocked. C'est la seule
// erreur de cette chaîne de lecture qui soit TRANSITOIRE et retryable :
// PrestigeHandler.serviceError la mappe en 503 + `Retry-After: 5`
// (api/handlers/prestige_squads.go:453), au lieu de la branche `default` = 500
// masqué non retryable.
//
// Le cas se produit notamment quand Reopen() (déclenché par les variantes
// *Recovered sur invalidation) ne peut pas ré-ouvrir le fichier parce qu'un
// autre process l'a saisi entre-temps.
//
// Toute autre erreur — y compris sql.ErrNoRows — est rendue TELLE QUELLE : le
// mapping domaine des callers (« défi/escouade introuvable », prestige vide)
// est inchangé.
//
// 2e copie ; à la 3e, centraliser + garde-rail (CLAUDE.md règle 6). Jumelle :
// translateLockErr (prestige_metadata_repo.go) — même forme, base et message
// de log différents (metadata vs shared_social).
func translateSocialLockErr(ctx context.Context, err error) error {
	if err == nil || !duckdb.IsFileLockError(err) {
		return err
	}
	// Jamais d'erreur avalée en silence (CLAUDE.md règle 3) : le 503 ne loggue
	// rien côté handler, la cause opérationnelle doit apparaître ici.
	slog.WarnContext(ctx, "prestige: shared_social verrouillée par un autre writer, dégradation en 503",
		"err", err)
	return fmt.Errorf("prestige shared_social locked: %w", errors.Join(dblease.ErrDBLocked, err))
}
