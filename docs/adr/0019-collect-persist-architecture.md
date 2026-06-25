# ADR 0019 — Architecture Collect → Persist (fix ART DuckDB définitif)

**Statut** : Accepté (2026-05-23) — implémenté Phase 1-2, activation progressive Phase 3 en cours.

**Branche source** : `refactor/collect-persist`

---

## Contexte

DuckDB est un moteur columnar. Sa stratégie d'UPDATE est `DELETE + INSERT` interne — chaque UPDATE touche l'index ART de la table. Sous **concurrence intensive** sur PK VARCHAR (cas `shared.match_participants` lors de syncs multi-joueurs en parallèle), l'index ART se corrompt :

```
FATAL Error: Invalid Input Error: Failed to delete all rows from index.
Only deleted 0 out of N rows
```

Cycle 19:33:20 (2026-05-22) — verdict empirique après 3 sprints de tentatives :

| Pattern testé | Résultat ART |
|---|---|
| `ON CONFLICT DO UPDATE` | FATAL |
| `INSERT OR REPLACE` | FATAL |
| `UPDATE` pur (then INSERT fallback) | FATAL |
| `singleflight` per-(match_id, xuid) | FATAL (réduit la fréquence mais ne supprime pas) |
| `CHECKPOINT` post-sync (Plan J) | Inefficace (FATAL avant CHECKPOINT) |
| `RebuildART` runtime (auto-heal) | Cache l'incident mais corruption revient |

**Conclusion** : tant qu'on fait des UPSERT/UPDATE concurrents sur DuckDB, le bug ART persiste. **Toutes** les tentatives de workaround SQL ont échoué.

---

## Décision

> **Séparer la collecte (parallèle, mémoire) de la persistance (séquentielle, batch INSERT-only)**.

Plus aucun UPDATE/UPSERT sur les tables critiques pendant le live sync. Les écritures se font exclusivement via des `INSERT` en transaction, avec un mécanisme d'idempotence (vérification `EXISTS(match_id)` en début de TX → skip silently si batch déjà persisté).

### Architecture

```
┌──── CALLERS (sync engine, backfill CLI, scripts) ────┐
│  builder := persist.NewBatchBuilder(player, xuid)     │
│  builder.AddMatch(...) / SetEnrichment(...)           │
│  queue.Submit(builder.Build())  ← non-bloquant       │
└───────────────┬───────────────────────────────────────┘
                ▼
┌────  BatchQueue (durable)  ──────────────────────────┐
│  - Submit → WAL JSON sur disque AVANT push channel    │
│  - 1 channel buffered par DBTarget                    │
│  - Drain(ctx) attend pending == 0                     │
│  - RecoverPending() relit le WAL au boot              │
└───────────────┬───────────────────────────────────────┘
                ▼
┌────  Persister (par DB target)  ─────────────────────┐
│  BEGIN TX                                             │
│    SELECT EXISTS(match_id) → skip ACK si déjà         │
│    INSERT batch sur N tables (jamais UPDATE)          │
│  COMMIT                                               │
│  → ACK = delete WAL file                              │
└───────────────────────────────────────────────────────┘
```

**4 Persisters** isolent les 4 DBs cibles :

| Persister | DB cible | Idempotence |
|---|---|---|
| `SharedPersister` | `shared_matches_v2.duckdb` | `EXISTS(match_registry.match_id)` |
| `PlayerPersister` | `stats.duckdb` (par joueur) | `EXISTS(player_match_enrichment.match_id)` |
| `PVEPersister` | `shared_pve.duckdb` | `INSERT OR IGNORE` sur PK (match_id, xuid) |
| `MetadataPersister` | `metadata.duckdb` | `INSERT OR IGNORE` sur PK (mode_en, lang) |

---

## Modes d'activation (3 niveaux)

| Mode | Flag | Path | Risque |
|---|---|---|---|
| **Legacy** | (défaut) | `insertFetchedMatch` (UPSERT direct sur shared) | ART possible |
| **Sync batch** | `LEVELUP_PERSIST_BATCH=1` | `submitMatchAsBatch` direct Persister (INSERT-only, pas de WAL) | ART supprimé, pas de crash-recovery |
| **Async batch** | + `WithBatchQueue` (pas de flag env pour l'instant) | `queue.Submit` + Worker async + Drain à fin de cycle | ART supprimé, WAL durable |

**Phase 3 (activation prog)** active le mode **sync batch** d'abord. L'async sera ajouté en Phase 4 si bénéfice observé.

---

## Property "ajout facile d'enrichment"

Le PlayerPersister utilise un **INSERT dynamique** sur `player_match_enrichment` : seuls les champs pointer non-nil de `EnrichmentRow` sont INSERTés.

**Pour ajouter un enrichment local** (3 étapes seulement) :

1. Migration DB : `addColumnIfMissing(db, "player_match_enrichment", "X", "DOUBLE")`
2. Champ pointer dans `internal/persist/rows.go::EnrichmentRow` : `X *float64`
3. 1 if-block dans `internal/persist/player_persister.go::enrichmentFields()`

→ pas de SQL à modifier, pas d'INSERT à toucher.

---

## Hors scope (à faire séparément)

- **Migration Postgres/SQLite** : on garde DuckDB, juste on l'utilise correctement.
- **Refactor des reads** : reste inchangé (DuckDB reste OLAP-friendly pour les query analytics).
- **Workflows non per-match** : `player_csr_snapshots`, `engagement_coefficients`, `player_assists_model`, `media_*`, `is_excluded`, `known_teammates_count` restent ad-hoc post-sync. Cf. `internal/persist/doc.go`.
- **Cleanup anti-ART** (singleflight, CHECKPOINT, UPDATE-then-INSERT migrations) : à supprimer une fois Phase 3 activée et validée 10 cycles sans FATAL ART.

---

## Critères de validation Phase 3

1. ✅ 10 cycles consécutifs prod avec `LEVELUP_PERSIST_BATCH=1` sans `FATAL Error: Invalid Input Error`.
2. ✅ `art_corruption_detected_*` reste à 0 sur 24h.
3. ✅ Cycle ≤ 8 min sur 3 joueurs.
4. ✅ Backfill CLI fonctionne avec la même archi (futur sprint).
5. ✅ Crash mid-persist → recovery au boot sans perte (vérifié `TestE2E_Pipeline_CrashRecovery` en async mode).

---

## Références

- `.ai/V7/REFACTOR_COLLECT_PERSIST.md` — design complet (577 lignes, 13 sections, 11 décisions validées)
- `.ai/V7/INCIDENT_ART_CORRUPTION_DUCKDB.md` — verdict empirique des tentatives SQL
- `.ai/ENRICHMENTS_CATALOG.md` — inventaire exhaustif des données (audit Phase 1.4)
- `internal/persist/doc.go` — checklist d'extension + workflows hors scope
- ADR 0017 — `rebuild-art-corruption-pattern.md` (auto-heal, devenu obsolète)
- ADR 0018 — `concurrent-write-model.md` (singleflight, devenu obsolète)
