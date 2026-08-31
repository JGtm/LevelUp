package migrations

// steps_appendonly_misc.go — 2 conversions append-only CONSOMMATRICES (match_csrs,
// pve_match_stats), déplacées depuis internal/migration/steps_*_append_only*.go
// (Phase 1.5 b22, voie B — regroupe les ex-b22/b24/b25 du plan, structurellement identiques).
// + kill_positions (G.2, 2026-08-30) : même mécanique, table SHARED (pas
// SharedPvE), ajoutée ici plutôt qu'un fichier dédié pour rester à côté de ses
// deux sœurs structurellement identiques (mécanisme written_at, dernière
// version par clé fonctionnelle).
//
// SQUASH v1 (chantier N4, 2026-07-12) : la 3e conversion, player_append_only_csr_snapshots_v1,
// faisait partie du bloc squashé dans create_baseline_player_v1 (elle no-opait sur DB vierge —
// player_csr_snapshots n'existe pas encore dans le bloc) → retirée ici (archive player_v1).
//
// Chaque step DÉLÈGUE au helper unique migration.ApplyAppendOnlyRebuild (cf.
// append_only_rebuild.go, doctrine « helper unique » ADR 0026) : swap CTAS
// TRANSACTIONNEL (rollback intégral sur erreur) + garde anti-perte rebuilt==before
// AVANT le DROP de l'ancienne table + recoverOrphan (réparation d'un crash mid-swap)
// + idempotence par présence de la colonne `id`. Spécifique à chaque table : la
// séquence backing `id`, les index PostSwap et la vue _latest (passés verbatim).
// Consommateurs de tables créées par des racines (player god-file global ; match_csrs
// via add_shared_match_csrs déjà title b3 ; pve_match_stats via add_pve_schema déjà
// title b3).
//
// Note d'historique : avant la délégation, ces 3 conversions étaient des copies inline
// NON transactionnelles (DROP TABLE avant toute vérification de cardinalité, sans
// recoverOrphan) → perte de données possible sur crash/erreur mid-swap. La délégation
// au helper supprime cette asymétrie de sûreté (parité exacte avec les conversions
// in-package du registre global), à ART éradiqué identique (PK technique id + written_at
// + vue _latest, écritures = INSERT pur).

import (
	"database/sql"

	"levelup/go-api/internal/migration"
)

// appendOnlyMiscSteps retourne les conversions append-only title-owned (b22 + G.2).
func appendOnlyMiscSteps() []migration.Migration {
	return []migration.Migration{
		{
			Name:        "shared_append_only_match_csrs_v1",
			TargetDB:    migration.TargetShared,
			Description: "Rebuild shared.match_csrs en append-only (id PK + written_at + vue latest) — élimine bug ART par construction",
			ApplySchema: applyAppendOnlyMatchCSRs,
		},
		{
			Name:        "shared_pve_append_only_v1",
			TargetDB:    migration.TargetSharedPvE,
			Description: "Rebuild pve_match_stats en append-only (id PK + written_at + vue latest)",
			ApplySchema: applyAppendOnlyPveMatchStats,
		},
		{
			Name:        "shared_append_only_kill_positions_v1",
			TargetDB:    migration.TargetShared,
			Description: "Rebuild shared.kill_positions en append-only (id PK + written_at + vue kill_positions_latest) — G.2, préalable au branchement de la capture Infinite (un re-décodage réécrirait sinon des doublons silencieux, cf. killer_victim_pairs 46,8 %)",
			ApplySchema: applyAppendOnlyKillPositions,
		},
	}
}

// applyAppendOnlyMatchCSRs délègue au helper commun (mécanisme written_at, dernière
// version par match_id+xuid).
func applyAppendOnlyMatchCSRs(db *sql.DB) error {
	return migration.ApplyAppendOnlyRebuild(db, migration.AppendOnlyRebuild{
		Table:         "match_csrs",
		IDSeq:         "mcsrs_seq",
		SyntheticCols: migration.SynthWrittenAt,
		PostSwap: []string{
			`ALTER TABLE match_csrs ALTER COLUMN written_at SET DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)`,
			`CREATE INDEX IF NOT EXISTS idx_match_csrs_lookup ON match_csrs(match_id, xuid, written_at)`,
			`CREATE INDEX IF NOT EXISTS idx_match_csrs_xuid    ON match_csrs(xuid)`,
			`CREATE INDEX IF NOT EXISTS idx_match_csrs_season  ON match_csrs(season_id)`,
			`CREATE INDEX IF NOT EXISTS idx_match_csrs_match   ON match_csrs(match_id)`,
		},
		ViewSQL: `CREATE OR REPLACE VIEW match_csrs_latest AS
			SELECT * FROM match_csrs
			QUALIFY ROW_NUMBER() OVER (PARTITION BY match_id, xuid ORDER BY written_at DESC, id DESC) = 1`,
	})
}

// applyAppendOnlyPveMatchStats délègue au helper commun (mécanisme written_at,
// dernière version par match_id+xuid).
func applyAppendOnlyPveMatchStats(db *sql.DB) error {
	return migration.ApplyAppendOnlyRebuild(db, migration.AppendOnlyRebuild{
		Table:         "pve_match_stats",
		IDSeq:         "pve_seq",
		SyntheticCols: migration.SynthWrittenAt,
		PostSwap: []string{
			`ALTER TABLE pve_match_stats ALTER COLUMN written_at SET DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)`,
			`CREATE INDEX IF NOT EXISTS idx_pve_lookup ON pve_match_stats(match_id, xuid, written_at)`,
			`CREATE INDEX IF NOT EXISTS idx_pve_match  ON pve_match_stats(match_id)`,
		},
		ViewSQL: `CREATE OR REPLACE VIEW pve_match_stats_latest AS
			SELECT * FROM pve_match_stats
			QUALIFY ROW_NUMBER() OVER (PARTITION BY match_id, xuid ORDER BY written_at DESC, id DESC) = 1`,
	})
}

// applyAppendOnlyKillPositions délègue au helper commun (mécanisme written_at, dernière
// version par match_id+killer_xuid+time_ms — LA clé fonctionnelle de la table depuis sa
// création, cf. steps_shared_kill_positions.go : aucune colonne victim_xuid n'existe pour
// affiner davantage). G.2 (2026-08-30) : la table est remplie par Halo 5 en prod
// (`ingest.MapKillPositions`) — le CTAS swap PRÉSERVE ces lignes (avant=après vérifié par le
// helper, rollback intégral sinon) ; Infinite est câblé ENSUITE (killcollector), sur ce même
// schéma désormais append-only — sans cette conversion PRÉALABLE, un re-décodage de film
// écrirait des doublons silencieux au lieu de superséder proprement (le défaut qui a coûté
// 46,8 % de doublons à killer_victim_pairs avant sa propre conversion).
func applyAppendOnlyKillPositions(db *sql.DB) error {
	return migration.ApplyAppendOnlyRebuild(db, migration.AppendOnlyRebuild{
		Table:         "kill_positions",
		IDSeq:         "kill_positions_seq",
		SyntheticCols: migration.SynthWrittenAt,
		PostSwap: []string{
			`ALTER TABLE kill_positions ALTER COLUMN written_at SET DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)`,
			`CREATE INDEX IF NOT EXISTS idx_kill_positions_lookup ON kill_positions(match_id, killer_xuid, time_ms, written_at)`,
		},
		ViewSQL: `CREATE OR REPLACE VIEW kill_positions_latest AS
			SELECT * FROM kill_positions
			QUALIFY ROW_NUMBER() OVER (PARTITION BY match_id, killer_xuid, time_ms ORDER BY written_at DESC, id DESC) = 1`,
	})
}
