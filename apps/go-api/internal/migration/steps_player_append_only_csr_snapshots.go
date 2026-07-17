package migration

// steps_player_append_only_csr_snapshots.go — la MIGRATION player_append_only_csr_snapshots_v1
// a été squashée dans create_baseline_player_v1 (chantier N4) et la conversion vit désormais
// dans le package title-owned (note d'historique dans steps_appendonly_misc.go). Le nom reste
// dans internal/migration/order.go (canonicalOrder) comme SENTINELLE DM-5.
//
// Ce fichier RÉINTRODUIT la conversion append-only sous la forme d'une RÉPARATION DE BOOT
// idempotente (PAS un step de registre) : filet M2/M3 de la revue 2026-07-17. Une player DB
// legacy (backup restauré pré-2026-05-24) porte encore l'ANCIEN schéma player_csr_snapshots
// (PK(playlist_id, season_id), sans colonnes `id`/`written_at`). Sur cette DB,
// EnsurePlayerSchema (sync/schema.go) crée ensuite l'index idx_pcs_lookup(... written_at) et la
// vue player_csr_snapshots_latest (QUALIFY ... written_at, id) : leur BIND échoue sur l'ancien
// schéma → OpenPlayerDB en échec PERMANENT (la player DB devient inouvrable, aucun step de
// migration restant ne répare puisque la conversion a été squashée).
//
// Décision D3 : la réparation est déclenchée par INTROSPECTION des colonnes (présence de `id`),
// JAMAIS par une sentinelle ni un numéro de version. Elle est 100 % idempotente (le helper
// applyAppendOnlyRebuild no-ope si le marqueur `id` est déjà présent, et no-ope si la table est
// absente — DB vierge). Elle DOIT s'exécuter AVANT la création de la vue → appelée en tête de
// EnsurePlayerSchema (cf. sync/schema.go).

import (
	"database/sql"
)

// EnsurePlayerCSRSnapshotsAppendOnly convertit player_csr_snapshots de l'ancien schéma
// (PK(playlist_id, season_id), sans id/written_at) vers l'append-only (PK technique `id` +
// `written_at`, écritures = INSERT pur, lecture via player_csr_snapshots_latest). Réparation
// de boot idempotente (finding M2/M3 — cf. godoc fichier) :
//   - table absente (DB vierge) → no-op (le schéma append-only est créé par playerSchemaSQL) ;
//   - colonne marqueur `id` déjà présente (déjà convertie) → no-op ;
//   - ancien schéma (id absent) → swap CTAS TRANSACTIONNEL (recoverOrphan + garde anti-perte
//     rebuilt==before + rollback intégral) via applyAppendOnlyRebuild (recette ADR 0026).
//
// ViewSQL laissé VIDE À DESSEIN : la vue player_csr_snapshots_latest reste possédée par
// playerSchemaSQL (sync/schema.go) — source unique, pas de duplication. La réparation garantit
// seulement les colonnes id/written_at + l'index, pour que l'index et la vue de playerSchemaSQL
// (exécutés juste après, dans le même EnsurePlayerSchema) bindent au lieu d'échouer.
//
// Exposé pour EnsurePlayerSchema (sync) et les fixtures de test.
func EnsurePlayerCSRSnapshotsAppendOnly(db *sql.DB) error {
	return applyAppendOnlyRebuild(db, appendOnlyRebuild{
		Table:         "player_csr_snapshots",
		IDSeq:         "pcs_seq",
		SyntheticCols: synthWrittenAt,
		PostSwap: []string{
			`ALTER TABLE player_csr_snapshots ALTER COLUMN written_at SET DEFAULT now()`,
			`CREATE INDEX IF NOT EXISTS idx_pcs_lookup ON player_csr_snapshots(playlist_id, season_id, written_at)`,
		},
	})
}
