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
//	│  - 1 channel par DB target (shared, player, pve)      │
//	│  - 1 worker goroutine par channel                     │
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
// **Extensibilité — ajouter un enrichment local** :
//  1. Migration DB : ajouter la colonne dans `internal/migration/`.
//  2. Ajouter le champ pointer dans `EnrichmentRow` (ce package).
//  3. Ajouter une branche `if row.NewField != nil { ... }` dans
//     `buildEnrichmentInsertSQL` (sql_builder.go).
//  4. Implémenter le compute pur dans `internal/sync/my_enrichment.go`.
//  5. Appeler le compute + assigner dans le batch dans l'orchestrator.
//  6. Test E2E qui valide que le nouveau champ est persisté.
//
// Cf. `.ai/REFACTOR_COLLECT_PERSIST.md` pour le design complet.
package persist
