# ADR 0027 — Sync pipeline V2 : cycle orchestrator (parallélisation cross-player)

> **Note de renumérotation** : ce document a porté le numéro **0020** à sa création.
> Renuméroté **0027** pour lever la collision avec `0020-coach-prestige-bridge`. Les références
> code « ADR 0020 » mentionnant `pipeline V2` / `D6.x` / le package `internal/sync/v2` désignent
> ce document (0027).

**Statut** : Proposé (2026-05-25) — D0 préparation en cours, livraison phasée D1-D8.

**Branche source** : `fix/art-eradication-and-home-resilience` (préparation), puis branche dédiée si nécessaire pour les phases suivantes.

---

## Contexte

Le sync actuel orchestre au niveau **player** : N goroutines parallèles (une par joueur), chacune fait son cycle complet `loadKnown → paginateAPI → submitMatch → drain → postSync`. Mesures cycle de 2026-05-25 à 13:59 (4 joueurs) :

| Joueur | Insérés | Durée | dont attente shared lease |
|---|---|---|---|
| Chocoboflor | 21 | 148 s | 0 s |
| JGtm | 66 | 435 s | 196 s |
| XxDaemonGamerxX | 10 | 235 s | 106 s |
| Madina97294 | 0 | 511 s | 393 s |

**Wall time** : 8 min 30 s pour 97 inserts, soit ~5 s/match en moyenne. Madina97294 passe **77 % de son temps à attendre le shared writer lease** alors qu'il n'a 0 match à insérer.

### Causes racines identifiées

1. **Cross-player dedup cassée en parallèle**. Le BatchQueue (cf. ADR 0019) logue `match persisté` lors du WAL submit, **avant** que le worker async ait commit en DB. Quand un autre joueur appelle `loadKnownMatchIDs` 70 ms plus tard, sa lecture ne voit pas les writes du joueur précédent. JGtm refetche probablement les matchs déjà ingérés par Chocoboflor.

2. **Worker BatchQueue monothread**. Un seul `Worker.Run()` consomme `chMain`. Sur 97 batches en file, le drain prend ≥ 60 s (timeout fixe). Tous les drains du cycle expirent au même seuil.

3. **Contention shared lease post-drain**. Les 4 syncs releasent le lease juste avant drain puis le re-acquièrent pour le post-sync. Le worker async qui consomme la queue veut aussi ce lease. Sérialisation totale → 6 min 30 d'attente pour Madina97294.

4. **API fetch sérialisé entre joueurs**. Les 4 syncs démarrent en même temps mais XxDaemonGamerxX attend 24 s et Madina97294 28 s avant leur premier appel `/matches` — la sérialisation API est imposée par le shared lease writer pendant `loadKnownMatchIDs` (lecture qui prend pourtant un lease global).

---

## Décision

> **Orchestrer le sync au niveau cycle (process-wide) au lieu du niveau player.** Un `CycleOrchestrator` partitionne le travail en 6 phases dont les contention points sont explicites.

### Architecture cible

```
┌─── CycleOrchestrator.Run([]PlayerProfile) ───────────────────────────────────┐
│                                                                              │
│ Phase 1 — Discovery (parallèle N joueurs, read-only)                        │
│   pour chaque player en goroutine : loadKnown + paginate API → []unknownID │
│                                                                              │
│ Phase 2 — Dedup global (single)                                              │
│   uniqueMatches = ∪ unknownIDs                                               │
│   canonicalFetcher[matchID] = player choisi pour l'API call                 │
│   participants[matchID] = []player                                           │
│                                                                              │
│ Phase 3 — Fetch shared (errgroup N=8)                                        │
│   pour chaque matchID unique : GetMatchStats (1 call, tous les participants)│
│                                                                              │
│ Phase 4 — Fetch per-player (parallèle par player, errgroup interne)         │
│   pour chaque player : PersonalScores / awards qui requièrent son token     │
│                                                                              │
│ Phase 5 — Persist cycle batch (single writer, 1 méga-batch)                  │
│   1 seul Submit() sur la queue → shared + tous les player.* en une seule TX │
│                                                                              │
│ Phase 6 — Post-sync (parallèle N joueurs, plus de contention lease)         │
│   pour chaque player : heals + films + citations + dominance + LUSR         │
└──────────────────────────────────────────────────────────────────────────────┘
```

### Garanties

- **Cross-player dedup correcte** : Phase 2 dédoublonne avant tout fetch API. 1 match jamais fetché 2 fois dans le même cycle, indépendamment de qui le voit en premier.
- **API parallélisée** : Phase 3 errgroup(8), pas de lease shared pendant le fetch.
- **Worker BatchQueue moins sollicité** : 1 méga-batch par cycle au lieu de 97 batches/cycle. Le worker traite vite (1 acquire lease + 1 commit).
- **Post-sync sans contention** : Phase 6 démarre quand Phase 5 est terminée. Aucun lease shared en jeu pendant les heals.
- **Films chunks restent parallélisés** : `backfill_weapons.go` garde son `errgroup.SetLimit(24)` — pas touché.

---

## Compatibilité et migration

### Feature flag — SUPPRIMÉ (2026-07-03, lot D1c)

Le flag `LEVELUP_SYNC_PIPELINE` et le fallback automatique V2→V1 ont été supprimés
(audits 2026-07, DEC-2). V2 est désormais l'unique moteur de sync des joueurs moteur :
`AutoSyncScheduler` route les joueurs moteur (Infinite) vers `v2.CycleOrchestrator`
et les titres live-only (Halo 5) vers `syncPlayer`→`liveRunner` (D1c étape 1 : V2 est
mono-titre, il ne route pas les live-only). Si l'orchestrator n'est pas câblé au boot
(prérequis pool/queue/metaDB manquants), le cycle bascule sur un filet structurel
`syncPlayer` — ce n'est PLUS un rollback flag-sélectionnable.

Historique (avant D1c) : `v1` (défaut) itérait sur les joueurs via l'engine legacy ;
`v2` (opt-in via l'env var) instanciait l'orchestrator, avec fallback auto V2→V1.

### Réutilisation

- V2 partage les Persisters (`internal/persist/`), le schéma DB et le WAL format.
- V2 réutilise les heals + le `SyncEngine` pour le post-sync (`internal/sync/engine_postsync.go`).
  `engine.run`/`RunDelta` reste PARTAGÉ (watcher, HTTP, CLI, admin convergence) — non supprimé.

### Shadow run (D7)

Avant de basculer prod en `v2`, runner V2 en parallèle de V1 en mode dry-run (Phase 1+2 seulement, aucun write). Logger la diff entre `unique_matches_v2` et `inserted_v1_total`. 1 semaine de zéro divergence → switch.

---

## Multi-titre

L'orchestrateur prend `titleSlug` en paramètre et le propage à tous les `Persister` et `OpenPlayerDB`. Le partitionnement par chemin FS (ADR 0008) est préservé. Aucun nouveau verrou cross-titre.

---

## Tests anti-régression (criticité MAX)

Le sync a été cassé 14 jours en mai 2026 (incident `xuid(NNN)` URL format). On ne peut pas se permettre une nouvelle régression silencieuse. Suite de tests obligatoire :

### Tests contract (V1 ET V2 doivent passer)

`internal/sync/contract_test.go` définit les invariants observables après un cycle :
1. Tous les matchs API non skippés sont en `shared.match_registry`.
2. Pour chaque participant tracké, `shared.match_participants` contient sa ligne avec xuid+match_id.
3. Pour chaque joueur, `player_match_enrichment` contient une ligne pour chaque match auquel il a participé.
4. Aucune ligne dupliquée (PK uniques respectées).
5. Cross-player dedup : pour un match joué par P1+P2, exactement 1 GetMatchStats API call.

### Tests intégration V2 spécifiques

`tests/integration/sync_v2/` :
- `test_full_cycle_4_players.go` — PvP escouade + PvE solo, dataset hétérogène
- `test_dedup_correctness.go` — Property : `count(unique) ≤ sum(unknownByPlayer)`
- `test_partial_failure.go` — 1 match retourne 500, les autres OK
- `test_token_expiry.go` — Token P1 expiré → ses enrichments skippés, P2..P4 OK
- `test_idempotence.go` — 2 cycles successifs sur mêmes données → state DB identique
- `test_halo_api_url_format.go` — Anti-régression `xuid(NNN)` format (incident mai 2026)
- `test_metadata_dsn_alignment.go` — Anti-régression `Can't open with different configuration` (incident citations 2026-05-25)
- `test_drain_visibility.go` — Phase 5 visible en Phase 6
- `test_soak_1h.go` — Pas de fuite mémoire / dégradation perf sur 1h

---

## Critères de validation (go/no-go pour switch prod)

1. Suite contract passe pour V1 ET V2 sur le même fixture.
2. Test parité V1/V2 GREEN sur 5 datasets différents.
3. Soak test 1 h sans crash, sans fuite mémoire détectable.
4. Shadow run 7 jours en prod : zéro divergence `unique_matches_v2 ≠ inserted_v1_total`.
5. Cycle complet 4 joueurs ≤ 3 min sur fake API (vs 8 min 30 actuel).
6. `no_art_patterns_test.go` passe sans nouvelle entrée allowlist.

---

## Hors scope

- Refactor des reads (DuckDB reste OLAP-friendly, aucun changement query).
- Migration schéma (aucun ALTER).
- Cleanup V1 : dans un PR séparé après 2 semaines de prod V2 stable (D8).
- Optimisation `backfill_weapons.go` errgroup limit (déjà optimal).
- Refonte heals : restent inchangés, juste appelés depuis Phase 6.

---

## Effort estimé

≈ 3 semaines focalisé (D0 → D7). Cleanup V1 (D8) +1j après période de stabilisation.

| Deliverable | Effort |
|---|---|
| D0 — Prep + ADR + suite contract V1 baseline | 2 j |
| D1 — Phase 1+2 (Discovery + Dedup) | 3 j |
| D2 — Phase 3 (Fetch shared parallèle) | 2 j |
| D3 — Phase 4 (Per-player enrichments) | 1.5 j |
| D4 — Phase 5 (Insert cycle batch) | 3 j |
| D5 — Phase 6 (Post-sync parallèle) | 2 j |
| D6 — Intégration scheduler + flag | 1 j |
| D7 — Shadow run | 7 j (élapsed, pas full-time) |
| D8 — Cleanup V1 | 1 j (post-D7) |

---

## Références

- ADR 0019 — Architecture Collect → Persist (Persisters, BatchQueue, WAL)
- ADR 0016 — SharedDBProvider RO↔RW swap
- ADR 0008 — Isolation multi-titre par chemin FS
- `internal/persist/queue.go` — implémentation BatchQueue actuelle (1 channel, 1 worker)
- `internal/sync/engine.go:478` — drain timeout 60 s
- `internal/sync/engine.go:465` — release lease avant drain (cause contention)
- `.ai/thought_log.md` 2026-05-25 — diagnostic complet du cycle 13:59
