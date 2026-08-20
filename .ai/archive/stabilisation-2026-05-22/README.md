# Archive — Plan de stabilisation 2026-05-22

Cette archive regroupe les 5 documents d'audit/incident qui ont déclenché le plan de stabilisation post-merge `fix/citations-progression-semantic`, ainsi que leurs résolutions associées.

## Documents

| Fichier | Type | Résolution |
|---------|------|------------|
| [INCIDENT_2026-05-20_match_participants_index.md](INCIDENT_2026-05-20_match_participants_index.md) | Incident | **Phase 1** — rebuild ART migration appliqué sur `match_participants` + filet de garde au boot + workaround `\|\| ''` retiré ; [ADR 0017](../../../docs/adr/0017-rebuild-art-corruption-pattern.md) |
| [INCIDENT_2026-05-21_metadata_duckdb_lock_air_hot_reload.md](INCIDENT_2026-05-21_metadata_duckdb_lock_air_hot_reload.md) | Incident | **Phase 2** — DumpCachedLeaks + démotion `slog.Error → Debug` + fix fuite refCount `PrestigeBundle` (cause racine confirmée) |
| [AUDIT_DUCKDB_ATTACH_2026-05-21.md](AUDIT_DUCKDB_ATTACH_2026-05-21.md) | Audit | **Phase 3** — `sync.Once` ATTACH global + migration 3/4 régressions home_repo vers SharedReader (Q26 split reporté Phase 3.bis) |
| [HANDOFF_db_open_audit_2026-05-20.md](HANDOFF_db_open_audit_2026-05-20.md) | Handoff | **Phase 3** — vérifié, le fix `shared_social` mix RO/RW était déjà appliqué via merge `fix/citations-progression-semantic` ; reste un audit pour mémoire |
| [AUDIT_ASCENSION_PIPELINE_DISCONNECTED_2026-05-21.md](AUDIT_ASCENSION_PIPELINE_DISCONNECTED_2026-05-21.md) | Audit | **Phase 4** — solution A (seed milestone_catalog idempotent) + solution B1 (refactor `SyncEngine` + interface `port.PostSyncRunner`) ; [ADR 0014 mis à jour](../../../docs/adr/0014-progression-tracking-v2-ascension.md#2026-05-22--wiring-corrigé-phase-4-plan-stabilisation) |

## Branches livrées

| Phase | Branche | Statut |
|-------|---------|--------|
| 1 | `fix/duckdb-art-corruption-rebuild` | Pushed (rebuild appliqué prod : Madina 1/10→10/10) |
| 2 | `chore/metadata-shutdown-cleanup` | Pushed |
| 3 | `fix/duckdb-pool-hardening` | Pushed (Q26 split reporté Phase 3.bis) |
| 4 + 1.bis | `feat/ascension-pipeline-v2-wiring` | Pushed (5 commits Phase 4 + 1 commit Phase 1.bis carry-adj) |
| 5 | `docs/post-stabilisation` | Cette doc |

## Suite

- **Phase 1.bis carry-adj** : modifs formule appliquées (Elo dynamique muOpp + carry asymétrique sur enemyAvgKE). Réduit la régression mais ne ferme pas l'écart Madina (1286 vs cible 1700+). Investigation gap résiduelle ouverte — sigma stagnant ? splitParticipantKEs ? breakdown composantes ?
- **Phase 3.bis** : Q26 LoadHomeMatches split cross-DB (player + shared) à refactorer en 2 phases Go-side. Modèle : `Q26CareerTopEncountersTpl` dans `career_repo.go`.
- **Investigation upstream `duckdb-go`** : ouvrir une issue avec repro minimal de la corruption ART après prochaine montée de version DuckDB. Cf. [ADR 0017 §Investigation upstream](../../../docs/adr/0017-rebuild-art-corruption-pattern.md#investigation-upstream).

## Cibles LUSR de validation (pour réf future)

- **Madina97294** : fin Platine / début-milieu Diamant (μ ≈ 1700-1900)
- **Chocoboflor** : milieu/bas Or (μ ≈ 1400-1500) — atteint ✓
- **JGtm** : milieu/bas Or (μ ≈ 1400-1500) — atteint ✓
