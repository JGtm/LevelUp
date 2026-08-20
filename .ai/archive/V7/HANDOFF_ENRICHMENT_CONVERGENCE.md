# HANDOFF — Stabilisation sync + enrichissement convergent

> Date : 2026-06-04. Branche : `fix/enrichment-convergence` (non commitée, en attente d'accord). Statut : livré + vérifié (build/vet/test verts), reste l'observation prod + affinage.

## 1. Problème initial

Enrichissements incomplets après sync : FDA « faux », Madina sans LUSR, Flo (Chocoboflor) sans perfs, section « Rang & MMR équipe » sans LUSR, « Frags par arme » vide. Diagnostic complet (logs + code + lecture DuckDB, serveur arrêté) : voir `.ai/thought_log.md` [2026-06-04].

## 2. Causes racines confirmées

- **RC-1** — 401 auth sans recovery : `MarkUnhealthy`/refresh jamais câblés sur le bon slot (`pinnedGamertag` vide pour `PolicyAnyPublic`), `doGet` ne retry pas sur 401 → `matches_inserted:0`.
- **RC-2** — player DB legacy de Chocoboflor SANS PRIMARY KEY (`player_match_enrichment`, `match_citations`, `match_skill_rank`) : `CREATE TABLE IF NOT EXISTS (...PK...)` no-op sur table préexistante → `INSERT OR IGNORE`/`ON CONFLICT` échouent. De plus `RunForDB(TargetPlayer)` n'était JAMAIS appelé au boot serveur.
- **RC-3** — reader LUSR profil triait/fenêtrait sur `match_skill_rank.start_time` qui est 100 % NULL pour les rows LUSR (colonne dénormalisée jamais remplie ; seul le CSR la remplit).
- **RC-5** — events film jamais récupérés sur le chemin live : le watcher (`Trigger`, `cmd/server/main.go`) était construit sans `WithHighlightEvents` (zéro-value false). Le heal qui masquait ça a été retiré le 2026-06-01 → `highlight_events`/`weapon_kills`/`killer_victim_pairs` arrêtés au 30/05.
- **RC-4 (LUSR équipe)** — PAS un trou d'archi : le scoreboard lit déjà le LUSR de chaque coéquipier SUIVI depuis SA player DB (`match_view_builders_team.go` → `GetMatchSkillRank` → `Q22aMatchSkillRankPlayer` qui lit `match_skill_rank_latest`). Le « manque » récent = conséquence de l'enrichment cassé. Réparé par RC-1/RC-2/RC-5. **CSR et LUSR sont exclusifs par match** (classé/non classé) → aucune table shared dédiée nécessaire.

## 3. Livré (tout build/vet/test vert ; `go test ./...` = 81 pkg ok / 0 fail ; aucune modif front)

| Fix | Fichiers |
|---|---|
| Boot migrations player | `cmd/server/main.go` (appelle `RunPlayerMigrations` par profil, après `runMigrations`, gardé `!DemoMode`) |
| Migrations PK correctives | `internal/migration/steps_player_repair_pk.go` (+ `_test.go`), `order.go` (canonicalOrder) — `repair_player_match_enrichment_primary_key` (réutilise `RebuildPlayerMatchEnrichmentART`, gardé `!hasPrimaryKey`) + `repair_match_citations_primary_key` (CTAS dynamique + dédup `(match_id, citation_name_norm)`). `match_skill_rank` NON touchée (append-only, laissé au step idempotent). |
| RC-1 recovery auth | `internal/sync/pooled_client.go` (+ `_test.go`) : 401/403 → `MarkUnhealthy(lease.Gamertag)` + retry borné 1× via `doPublic` ; 429/503 cooldown global inchangé. |
| RC-5 watcher events 1er passage | `cmd/server/main.go` : `WithHighlightEvents: true` sur le `Trigger`. |
| Convergence | `internal/sync/convergence.go` (+ `_test.go`), `engine_postsync.go` : gate `len(insertedIDs)>0` délié → `insertedIDs ∪ selectMatchesMissingWeapons`, + étape events convergence AVANT (réutilise `FindMatchesMissingData`). Déclenchée même sans insert (`hasConvergenceBacklog`). |
| Attente film 1er passage | `internal/sync/engine_highlight_events.go` (`fetchHighlightChunkResilient`) + `engine_fetch.go`. Voir §5. |
| RC-3 reader LUSR | `internal/progression/profile/service.go` : `match_skill_rank_latest` + `COALESCE(start_time, written_at)`. |
| Observabilité | `engine_postsync.go` : expvar `convergence_{events,weapons}_pending_total` + `_processed_total` (passif, `/debug/vars`). |

## 4. La convergence n'est PAS un heal (garde-fous)

Le heal tué (01/06) réécrivait `match_participants` en `ON CONFLICT DO UPDATE` → corruption ART. La convergence ici : (1) rappelle UNIQUEMENT les persisters INSERT-pur / DELETE-then-INSERT sérialisés existants (garde `no_art_patterns_test` verte) ; (2) un seul chemin d'écriture (pas de duplication) ; (3) idempotente + terminale (no-film 30j) → work-set prouvé décroissant vers zéro (`convergence_test.go`). C'est un filet EXCEPTIONNEL, pas une roue de secours permanente.

## 5. POINT D'AFFINAGE — attente film au 1er passage (à observer)

Objectif produit : un match s'affiche COMPLET (events → killer_victim → frags par arme), pas à moitié puis rattrapé un cycle plus tard.

Mécanisme : `fetchHighlightChunkResilient` — pour un match FRAIS (< `freshFilmWaitWindow` = 10 min) dont le film n'est pas prêt au 1er fetch, re-essaie à intervalle borné jusqu'à publication. Le watcher détecte en ~10s mais le film se publie en ~1 min.

**Valeur de départ : 30s × 3 (re-essais à +30s/+60s/+90s).** Réglable EN PROD SANS REDÉPLOIEMENT via `LEVELUP_FRESH_FILM_RETRY` :
- `"0"` → désactive l'attente (retombe sur la convergence)
- `"30,30,30"` ou `"45,45"` → CSV de secondes
- absent → défaut 30s × 3

**À observer en prod** (puis affiner) :
1. Les matchs fraîchement joués s'affichent-ils COMPLETS (frags par arme présents) du premier coup ? Si non → augmenter le délai (le film met plus d'1 min).
2. Compteurs `/debug/vars` `convergence_*_pending_total` : doivent PLAFONNER en régime stationnaire (la convergence ne tourne qu'en exceptionnel). S'ils croissent en continu → le 1er passage fuit, augmenter le délai ou auditer le timing watcher.
3. Latence d'affichage : un match frais apparaît ~30-90s après la partie (le temps que le film se publie). Acceptable car complet ; si jugé trop lent, baisser le délai (au prix d'un risque de complétude).

## 6. Différé (non bloquant)

- **Phase 3a — écrire `start_time` sur l'INSERT LUSR** : ABANDONNÉE en l'état. La colonne `match_skill_rank.start_time` est une dénormalisation redondante (le `match_id` → `match_registry` détient le temps canonique). Le reader (3b) gère via `COALESCE(start_time, written_at)`. Si un jour on veut nettoyer : soit JOIN `match_registry` au reader, soit supprimer la colonne. Ne PAS la remplir (empire la dénormalisation).
- **Self-alert convergence** : option non retenue par défaut (l'utilisateur ne consulte pas les logs/jauge). Si besoin : exposer un champ « enrichissement en retard » dans `/health` (déjà monitoré par le VPS) plutôt qu'un nouveau canal.
- **Retry events intra-sync** : livré (§5) — c'était l'élément qui transforme « solide à ~90 % » en « complet au 1er passage ».

## 7. À valider après déploiement

- Re-sync des 4 joueurs (tokens Madina/XxDaemon depuis le store `data/auth/watcher_tokens/`, cf. ADR 0023) → vérifier `matches_inserted>0` (RC-1 réparé).
- Au reboot : les migrations PK s'appliquent sur la DB legacy de Chocoboflor (logs `repair_*_primary_key`) → ses perfs/citations se peuplent.
- Niveaux LUSR attendus (sanity check, cf. mémoire `reference_lusr_target_levels`) : Madina97294 fin Platine/début Diamant, Chocoboflor + JGtm milieu/bas Or.
- Backlog events 30/05 → convergence le résorbe sur quelques cycles (horizon 50/cycle).

## 8. Commit

Non commité (accord requis). Plan : 3 commits logiques sur `fix/enrichment-convergence` — (1) foundations (migrations PK + boot + auth + watcher events), (2) convergence + attente film + observabilité, (3) reader LUSR. `cmd/diag_q/` (runner SQL diag) optionnel. EXCLURE `cmd/rdata_weapon_scan/` (scratch d'un agent, hors scope).
