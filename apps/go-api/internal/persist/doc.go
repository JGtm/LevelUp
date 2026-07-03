// Package persist — architecture Collect → Persist anti-corruption ART DuckDB.
//
// **Problème résolu** : DuckDB columnar implémente UPDATE comme DELETE+INSERT
// en interne ; sous concurrence intensive sur PK VARCHAR, l'index ART se
// corrompt. Verdict empirique 2026-05-23 : aucun pattern SQL ne contourne le
// bug (UPSERT, UPDATE pur, INSERT OR REPLACE).
//
// **Solution** : séparer la **collecte** (parallèle, en mémoire) de la
// **persistance** (séquentielle, INSERT batch uniquement). Aucun UPDATE/UPSERT
// sur les tables critiques → pas de pression sur l'index ART → bug ART
// impossible par construction.
//
// **Architecture** :
//
//	┌──── CALLERS (sync engine, backfill CLI, scripts) ────┐
//	│                                                       │
//	│  builder := persist.NewBatchBuilder(player, xuid)     │
//	│  builder.AddMatch(...)                                │
//	│  builder.AddEnrichment(...)                           │
//	│  queue.Submit(builder.Build())  ← non-bloquant        │
//	└───────────────┬───────────────────────────────────────┘
//	                ▼
//	┌────  BatchQueue  ────────────────────────────────────┐
//	│  - WAL JSON sur disque AVANT push channel             │
//	│  - 1 channel UNIQUE partagé (pas de routage DBTarget) │
//	│  - 1 worker + CombinedPersister (shared+player/batch) │
//	│  - Recovery au boot : relit WAL et re-pousse         │
//	└───────────────┬───────────────────────────────────────┘
//	                ▼
//	┌────  Persister (par DB target)  ─────────────────────┐
//	│  BEGIN TX                                             │
//	│    INSERT batch sur N tables (jamais UPDATE)          │
//	│  COMMIT                                               │
//	│  → ACK = delete WAL file                              │
//	└───────────────────────────────────────────────────────┘
//
// **Granularité** : 1 batch = 1 match COMPLET (data API + tous les
// enrichments locaux). Cf. §6.bis de REFACTOR_COLLECT_PERSIST.md.
//
// **Cache fetch intermédiaire** : activé par défaut. Les responses API
// brutes sont écrites dans `data/sync_cache/{cycle_id}/match_{id}_*.json`
// pour debug + tests + recovery sans re-fetch. Désactivable via
// LEVELUP_PERSIST_NO_FETCH_CACHE=1.
//
// **Extensibilité — ajouter un enrichment local sur player_match_enrichment** :
//
//  1. Migration DB : ajouter la colonne dans `internal/migration/steps_player*.go`
//     via `addColumnIfMissing(db, "player_match_enrichment", "X", "DOUBLE")`.
//  2. Ajouter le champ pointer dans `EnrichmentRow` (rows.go).
//  3. Ajouter une entrée dans la liste de champs de
//     `player_persister.go::enrichmentFields()` :
//     `if row.X != nil { append(fields, fieldEntry{"X", *row.X}) }`.
//  4. Implémenter le compute pur dans `internal/analysis/*` ou `internal/sync/*`.
//  5. Brancher dans l'orchestrator : `builder.SetEnrichment(enrichmentRow)`.
//  6. Test TDD dans `player_persister_test.go` qui assigne le champ et vérifie
//     qu'il est persisté en DB.
//
// **Extensibilité — ajouter une NOUVELLE TABLE shared** :
//
//  1. Migration DB : `CREATE TABLE IF NOT EXISTS` dans `steps_shared*.go`.
//  2. Ajouter un struct `XxxInsert` dans rows.go.
//  3. Ajouter `[]XxxInsert` dans `SharedBatch` (batch.go).
//  4. Ajouter `AddXxx()` setter dans `BatchBuilder` (builder.go).
//  5. Ajouter `persistXxx()` helper dans `shared_persister.go` + appel
//     dans `Persist()`.
//  6. Test TDD dans `shared_persister_test.go`.
//
// **Parité legacy en live sync — PVE/Metadata non câblés** :
//
// `MatchBatch` peut transporter `PVE` et `Metadata` sous-batches (Phase 2.2
// du refactor a ajouté l'extraction au fetch), MAIS `submitMatchAsBatch`
// (orchestrateur Phase 2.3 du chemin Collect→Persist) NE LES PERSISTE PAS
// pour préserver la parité avec le legacy `insertFetchedMatch` :
//   - `pve_match_stats` est écrit par les backfills CLI (`--pve`), pas par
//     le live sync delta. Wirer PVEPersister ici ajouterait un comportement.
//   - `mode_name_tr` est seedé par CLI (`cmd/seed-*`), pas par le sync.
//
// → Décision : PVEPersister/MetadataPersister restent disponibles (testés
// en isolation, 8 tests GREEN) mais non câblés dans `submitMatchAsBatch`.
// À câbler quand une feature future le nécessitera (ex : auto-extract PVE
// stats au sync, écriture de traductions dynamiques).
//
// **Hors scope MatchBatch** — écritures NON liées à un match précis. Ces
// workflows restent en dehors du flux Collect→Persist (ad-hoc writes via
// repo dédié ou post-sync direct sur la DB). À documenter ici pour ne pas
// se faire piéger lors du prochain refactor :
//
//   - `player_csr_snapshots` (player DB) : snapshot CSR officiel Waypoint par
//     playlist+saison. 1 fois par cycle de sync, pas per-match.
//   - `engagement_coefficients` (player DB) : coefs team_share/lobby_share
//     par (xuid, mode_category). Recompute aggregate post-sync.
//   - `player_assists_model` (player DB) : coefs régression expected_assists
//     personnalisés par (joueur × game_variant). Recompute via
//     `--assists-model` CLI sur l'historique entier.
//   - `assists_model_coefs` (metadata DB) : coefs populationnels, fallback si
//     pas assez de données joueur. Peuplé par `cmd/seed-assists-model`.
//   - `media_files` / `media_match_associations` (player DB) : workflow
//     d'indexation de medias, ne suit pas le flux sync de matchs.
//   - `match_exclusion` (`is_excluded`) (player DB) : action utilisateur via
//     PATCH /players/{slug}/matches/{id}/exclusion. Pas de relation avec
//     le sync. Écrit par `match_exclusion_repo.go`.
//
// Cf. `.ai/REFACTOR_COLLECT_PERSIST.md` pour le design complet,
// `.ai/ENRICHMENTS_CATALOG.md` pour l'inventaire exhaustif des données.
package persist
