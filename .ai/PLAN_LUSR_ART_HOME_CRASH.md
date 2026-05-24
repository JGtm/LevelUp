# Plan — Éradication exhaustive ART + crash home post-sync

**Date** : 2026-05-24
**Branche cible** : `fix/art-eradication-and-home-resilience` (depuis `refactor/collect-persist`)
**Niveau d'effort** : ~3 à 5 jours (scope élargi suite à audit complet)
**Documents liés** : [.ai/audit_art_writes.md](audit_art_writes.md), `INCIDENT_ART_CORRUPTION_DUCKDB.md` (ancien), ADR-0019

> **Changement de scope par rapport à la v1 du plan** : la v1 ne traitait que LUSR. L'audit exhaustif a identifié **9 sites CRITIQUES et ~20 sites HAUT** sur les mêmes DBs. Tant qu'on ne les corrige pas tous, le même crash peut ressurgir sur une autre table à tout moment. Ce plan élargi couvre l'éradication complète.

---

## 0. Pour comprendre sans jargon

### Le bug ART, en une phrase

DuckDB maintient un index interne (ART). Toute opération qui touche cet index par un DELETE — explicite (`DELETE FROM`) ou implicite (`UPDATE` d'une colonne indexée, `ON CONFLICT DO UPDATE`, `INSERT OR REPLACE`) — peut le corrompre. La base est alors marquée FATAL jusqu'au prochain restart.

### Ce qui s'est passé ce soir

1. Sync de Chocoboflor à 20:41:04 → crash ART sur `match_skill_rank` (player DB) via le chemin LUSR `DELETE + INSERT en TX`.
2. Cascade sur la même player DB : achievements, friends recompute, vues mat. — toutes invalidées par le FATAL.
3. Le sync se déclare quand même `status:"success"` (avec `lusr_updated:0` et `warnings:8`).
4. La handle SQL de la player DB devient zombie → les handlers HTTP qui la détiennent encore (`PlayerMatchesRepo` notamment) renvoient `sql: database is closed` → 500 sur la home page.

### Pourquoi je n'avais pas tout vu avant

- L'ADR-0019 et CLAUDE.md classaient **par hypothèse** plusieurs tables comme "non concernées" par le bug (LUSR, citations, engagement, achievements). Cette classification était basée sur l'observation que le bug se manifestait surtout sur `shared.match_participants`. **L'audit révèle ~30 sites qui auraient pu causer le même crash** — LUSR a juste été le premier servi.
- Le fichier [art_upsert_patterns_test.go](../apps/go-api/internal/sync/art_upsert_patterns_test.go) teste 3 patterns en `:memory:` et logge `0 errors` pour tous. Cela ne prouve rien (le bug ne se reproduit pas en mémoire), mais a pu donner une fausse impression de sécurité.

---

## 1. Faits validés (post-restart 20:18:30 uniquement)

| Heure | Joueur | Statut | Réalité |
|-------|--------|--------|---------|
| 20:36:48 | JGtm | success | OK |
| 20:38:20 | XxDaemonGamerxX | success | OK |
| 20:38 → 20:43+ | home JGtm | — | **500 répété** `sql: database is closed` |
| 20:39:49 | Madina97294 | success | OK (LUSR a écrit via INSERT-only) |
| **20:41:04** | **Chocoboflor** | — | **FATAL ART sur match_skill_rank** |
| 20:41:08 | Chocoboflor | success | KO : `lusr_updated:0`, achievements KO, mv KO |
| 20:49 → 20:52 | les 4 | success | no-op (`inserted:0`) |

Signature exacte du crash :
```
upsertLUSRRatingsBatch: PostSyncLUSRPersister.Upsert échoué
err="persist: Commit LUSR upsert: FATAL Error: Invalid Input Error:
     Failed to delete all rows from index. Only deleted 0 out of 1 rows.
     Chunk: [14 Columns]
     - FLAT VARCHAR: 1 = [ ec938fb4-a1ad-42c2-aa64-72df11b7256f ]  ← match_id
     - FLAT VARCHAR: 1 = [ LUSR ]                                   ← rating_type
     ..."
```

---

## 2. Cause racine triple

1. **Schema fragile** : des tables critiques (`match_skill_rank`, `match_csrs`, `player_match_enrichment`, `player_csr_snapshots`, `pve_match_stats`, etc.) reçoivent des écritures qui passent toutes par un DELETE (explicite ou implicite via UPSERT). Le bug ART est statistiquement inévitable.
2. **Handles non résilientes** : aucun mécanisme côté player DB pour rouvrir une handle invalidée. Les repos HTTP détiennent des `*sql.DB` qui peuvent devenir zombies sans préavis.
3. **Status sync menteur** : `status:"success"` est renvoyé même quand le post-sync a planté en FATAL. Aucun caller (UI, monitoring, rapport) ne peut détecter la dégradation.

---

## 3. Décisions de design (les plus pérennes possibles)

### Décision A — Schema append-only sur toutes les tables player DB et shared DB à écritures fréquentes

**Tables ciblées** (depuis l'audit `.ai/audit_art_writes.md`) :
- `match_skill_rank` (LUSR + CSR)
- `match_csrs` (shared)
- `player_csr_snapshots`
- `pve_match_stats`
- `player_match_enrichment` (cas particulier, voir Décision B)

**Recette uniforme** :
1. `ALTER TABLE x ADD COLUMN written_at TIMESTAMP NOT NULL DEFAULT now()`
2. PK élargie pour inclure `written_at` (jamais de conflit → aucune raison de DELETE)
3. Vue `CREATE OR REPLACE VIEW x_latest AS SELECT … QUALIFY ROW_NUMBER() OVER (PARTITION BY <clé fonctionnelle> ORDER BY written_at DESC) = 1`
4. Tous les writes deviennent INSERT pur
5. Toutes les lectures pointent sur `x_latest`

**Avantage** : bug ART **impossible par construction** sur ces tables.

### Décision B — `player_match_enrichment` : forcer le batch mode toujours

Cette table a déjà un chemin INSERT-only via `PlayerPersister` (BatchBuilder). Le path legacy (`LEVELUP_POSTSYNC_INSERT_ONLY=0`) existe encore par défensif et utilise `ON CONFLICT DO UPDATE` — c'est ce qui rend `sync/performance.go:641`, `sync/comeback.go:180`, `sync/engagement.go:498,519` à risque.

**Action** : supprimer le path legacy. Forcer toujours `batchMode = true`. Nettoyer les `if os.Getenv("LEVELUP_POSTSYNC_INSERT_ONLY") != "0"`.

### Décision C — Pattern SELECT-then-UPDATE-or-INSERT pour les handlers HTTP basse fréquence

**Tables ciblées** : `streaks`, `player_records`, `prestige_status`, `engagement_scores` (handler), `notification_prefs`, `match_exclusions`, `privacy_state`, etc.

**Recette** : remplacer `ON CONFLICT DO UPDATE` par :
```go
err := db.QueryRow(`SELECT 1 FROM t WHERE pk = ?`, pk).Scan(&exists)
if err == sql.ErrNoRows {
    db.Exec(`INSERT INTO t (...) VALUES (...)`, ...)
} else {
    db.Exec(`UPDATE t SET ... WHERE pk = ?`, ..., pk)
}
```

**Avantage** : pas de DELETE déclenché, pas de migration de schema. Coût : 2 round-trips par row, acceptable hors hot path sync.

### Décision D — Provider player DB résilient au FATAL

**Recette** :
- Ajouter `RecoverFromFATAL(ctx, playerSlug)` côté provider player DB.
- Sur erreur FATAL détectée : ferme proprement, attend les TX en cours, rouvre, invalide les références stale.
- Handler home : sur `database is closed` ou FATAL → retry 1× après recovery, sinon 503 + `Retry-After`.

### Décision E — Status sync honnête

**Recette** :
- Dans `engine.go:548`, si une étape post-sync renvoie FATAL OU si `lusr_updated == 0` alors qu'il y avait des candidats → `status: "partial"` ou `"degraded"`.
- Les logs WARN existants restent.

### Décision F — Guard-rail anti-régression

Test `apps/go-api/internal/sync/no_art_patterns_test.go` qui grep les patterns interdits dans `sync/`, `persist/`, `platform/duckdb/`. Allowlist explicite des sites tolérés (avec justification commentée). Fail si un nouveau site apparaît hors allowlist.

---

## 4. Plan d'exécution (5 jours estimés)

### Phase 1 — Reproduction ART en test (1/2 jour)

**Objectif** : test rouge avant tout fix, par DB et par catégorie de pattern.

Fichiers à créer dans `apps/go-api/internal/persist/` :
- `lusr_art_repro_test.go` : DuckDB sur fichier, schema `match_skill_rank` réel, batch DELETE+INSERT avec collisions → assert erreur "Failed to delete all rows from index"
- `csr_art_repro_test.go` : idem mais sur `match_skill_rank` rating_type=CSR + sur `match_csrs` shared
- Compléter `art_upsert_patterns_test.go` pour reproduire sur fichier persistant (pas `:memory:`)

**Critère go** : 3 tests rouges documentés, commités isolément.

### Phase 2 — Migration schema append-only (1 à 1.5 jour)

**Tables traitées par ce PR** : `match_skill_rank`, `match_csrs`, `player_csr_snapshots`, `pve_match_stats`. (Hors `player_match_enrichment` traité en Phase 3.)

**Pour chaque table** :
1. Migration SQL versionnée dans `apps/go-api/internal/platform/duckdb/migrations/`
2. Vue `<table>_latest` créée par la même migration
3. Persister INSERT-only correspondant dans `internal/persist/` (ou modification de l'existant pour retirer le DELETE)
4. Bascule des writes dans `internal/sync/*` vers le nouveau persister
5. Bascule des lectures dans `internal/platform/duckdb/*_repo.go` vers la vue `_latest`
6. Tests unitaires INSERT + re-INSERT du même key → lecture latest renvoie la version la plus récente
7. Grep automatique dans test : aucune occurrence de `DELETE FROM <table>` ni `ON CONFLICT.*<table>`

**Critère go** :
- Tests Phase 1 verts (le pattern qui crashait ne crashe plus).
- `grep -rE "DELETE FROM (match_skill_rank|match_csrs|player_csr_snapshots|pve_match_stats)|ON CONFLICT.*<table>"` → 0 hit hors tests et migrations.

### Phase 3 — `player_match_enrichment` : suppression du path legacy (1/2 jour)

**Objectif** : forcer le batch mode partout, supprimer les `ON CONFLICT DO UPDATE` legacy.

- Identifier tous les `if os.Getenv("LEVELUP_POSTSYNC_INSERT_ONLY") != "0"` dans `sync/comeback.go`, `sync/engagement.go`, `sync/performance.go`.
- Supprimer la branche legacy, garder uniquement le path batchMode (via `PlayerPersister`/`BatchBuilder`).
- Mettre à jour la doc `internal/persist/doc.go` pour acter la suppression.
- Test : régression sur les 3 calculs (comeback dominance, engagement, performance score) avec dataset de référence.

**Critère go** : `grep "LEVELUP_POSTSYNC_INSERT_ONLY"` retourne 0 hit (sauf docs). Tests existants passent.

### Phase 4 — Handlers HTTP : pattern SELECT-then-UPDATE-or-INSERT (1/2 jour)

**Tables traitées** : `streaks`, `player_records`, `engagement_scores` (handler), `match_exclusions`, `notification_prefs`, `prestige_status`.

Pour chaque : refactor du `ON CONFLICT DO UPDATE` en pattern C. Tests unitaires existants doivent passer (sémantique identique).

**Critère go** : tests verts, pas de régression handler HTTP.

### Phase 5 — Provider player DB résilient + status sync honnête (1 jour)

**Sous-étape 5.1 — investigation provider (2 h)** :
- Lire le code du provider player DB. Documenter findings dans ce doc.

**Sous-étape 5.2 — RecoverFromFATAL (3 h)** :
- Implémentation côté provider.
- Test : forcer FATAL sur DB Chocoboflor (via SQL invalide), vérifier que GET home JGtm reste 200.

**Sous-étape 5.3 — Handler home retry/503 (1 h)** :
- Détection `database is closed` ou FATAL → 1 retry après recovery → sinon 503 + Retry-After.

**Sous-étape 5.4 — Status sync (1 h)** :
- `engine.go:548` retourne `partial` sur FATAL post-sync.
- Test sur 3 cas.

### Phase 6 — Guard-rail + audit metadata cache (1/2 jour)

**Sous-étape 6.1 — Guard-rail** :
- `internal/sync/no_art_patterns_test.go` : grep + allowlist + fail sur nouveau hit.
- Allowlist initiale = sites MOYENS de l'audit non traités dans ce PR (cache metadata, handler basse fréquence).

**Sous-étape 6.2 — Audit metadata cache et social** :
- Lister les sites MOYENS restants (cache metadata, persist_sink, social) dans `.ai/audit_art_writes.md` section "post-PR".
- Tickets à créer pour la suite.

### Phase 7 — Livraison (1/2 jour)

- Update CLAUDE.md : retirer la phrase fausse, ajouter une note "voir audit_art_writes.md pour le périmètre exhaustif".
- Update ADR-0019 : compléter avec les tables ajoutées et le pattern append-only.
- Nouvelle ADR-0020 (optionnel mais recommandé) : "Pattern append-only + vue latest pour éliminer le bug ART".
- Entrée `.ai/thought_log.md` finale `[YYYY-MM-DD] Éradication ART exhaustive` — Complété — avec liste des sites fixés et impact mesuré.
- Grille `delivery-checklist` :
  - `go test ./...` + `go vet ./...` verts
  - Aucune fonction > 80 lignes ajoutée
  - Aucun fichier > 500 lignes
  - Logs structurés via `slog.ErrorContext`
- PR depuis `fix/art-eradication-and-home-resilience` vers `refactor/collect-persist`.

---

## 5. Critères go/no-go globaux

Le fix est livrable quand **tous** ces critères sont verts :

- [ ] 3 tests ART repro de Phase 1 passent (le pattern fixé ne crashe plus)
- [ ] Grep retourne 0 hit sur les tables critiques pour `DELETE FROM` / `ON CONFLICT DO UPDATE` / `INSERT OR REPLACE` (hors tests et migrations)
- [ ] 10 syncs back-to-back sur les 4 joueurs sans aucun WARN/ERROR ART dans les logs
- [ ] Test d'intégration : FATAL provoqué sur player DB A → home de B reste 200
- [ ] Test d'intégration : FATAL provoqué sur player DB A → home de A retourne 503 puis 200 après retry
- [ ] Test : sync avec post-sync FATAL → status renvoyé = `partial`, pas `success`
- [ ] Guard-rail `no_art_patterns_test.go` vert (allowlist déclarée)
- [ ] CLAUDE.md à jour (phrase fausse retirée)
- [ ] ADR-0019 mis à jour, ADR-0020 (optionnelle) rédigée
- [ ] thought_log à jour
- [ ] `go test ./...` + `go vet ./...` verts depuis `apps/go-api/`

---

## 6. Hors périmètre — à traiter dans des PR suivants

Les sites MOYENS de l'audit ne sont pas tous fixés dans ce PR (sinon 2 semaines de travail). Sont reportés :

- Cache metadata (`metadata_repo.go`, `asset_cache_repo.go`, `medal_cache_repo.go`, `map_cache_repo.go`, `persist_sink.go`) — risque faible car volumétrie basse, mais à fixer.
- Prestige metadata et social (`prestige_metadata_repo.go`, `prestige_social_repo.go`, `prestige_player_repo.go`) — handler HTTP basse fréquence, fixable par pattern C.
- Notifications repo (`notifications_repo.go`) — basse fréquence, pattern C ou append-only.
- Media repo (`media_repo.go`) — basse fréquence, mix UPDATE/DELETE/ON CONFLICT, à analyser cas par cas.
- Auth tokens (`queries_auth.go`) — basse fréquence et structure simple, ON CONFLICT à remplacer par C.

Ces sites sont **listés explicitement** dans `.ai/audit_art_writes.md` section 2.4/2.5 et seront tracés dans la allowlist du guard-rail Phase 6.

---

## 7. Glossaire

- **ART** : Adaptive Radix Tree, l'index interne DuckDB. Le bug `Failed to delete all rows from index` est un crash connu de ce moteur.
- **FATAL DuckDB** : état où DuckDB marque la base comme invalidée. Aucune requête ne passe plus jusqu'au redémarrage. Symptôme : `database has been invalidated because of a previous fatal error`.
- **Player DB** : la DuckDB de stats d'un joueur, `data/players/{gamertag}/stats.duckdb`.
- **Shared DB** : la DuckDB de matchs partagés, `data/warehouse/shared_matches_v2.duckdb`.
- **Post-sync** : pipeline qui tourne après l'insertion d'un match — calcule LUSR, citations, vues mat., friends, achievements.
- **BatchBuilder** : composant de `internal/persist/` qui sérialise les écritures via INSERT batch, prescrit par l'ADR-0019.
- **Append-only** : pattern de schema où on n'écrase jamais, on ajoute toujours. Lecture via vue qui prend la version la plus récente.
- **Pattern C** : SELECT-then-UPDATE-or-INSERT — alternative à `ON CONFLICT DO UPDATE` qui évite le DELETE implicite. Cf. `art_upsert_patterns_test.go`.
