# Plan — Régression sync sur matchs récents (post-23 avril 2026)

**Branche** : `feat/token-pool-parallel-sync`
**Date** : 2026-05-08
**Statut** : Diagnostic complet, plan validé, implémentation en cours.

Ce document est destiné à être lu par plusieurs agents qui ont identifié la même
cause racine sous des angles différents. Il consolide diagnostic, frontière de
responsabilité (ce qui est résolu par le fix parser vs ce qui reste), et plan
d'action pour la partie indépendante du parser.

> **Articulation avec [`PLAN_HIGHLIGHT_EVENTS_BACKFILL.md`](PLAN_HIGHLIGHT_EVENTS_BACKFILL.md) (mis à jour 2026-05-08)** :
>
> - Le plan jumeau couvre la chaîne sync highlight events (parser bit-aligné, replay tool, bitmasks honnêtes events+kvp, audit lecture seule des autres bits, HTTP `/backfill/start`, CLI `levelup replay-events`, golden fixture E2E).
> - **Phase D de ce plan a été réduite** : les invariants 1, 2, 3, 5 sont absorbés par le golden test (Phase 4) et les tests bitmasks (Phase 1bis) du plan jumeau. Seul l'invariant 4 (home/match-view consistency) reste ici, lié à RC4.
> - **Ordre d'exécution recommandé** : exécuter le plan jumeau d'abord. Une fois les bitmasks honnêtes et le replay terminé, certains symptômes RC4/RC5/RC6 ci-dessous peuvent partiellement disparaître (les fallbacks UX déclenchés par des bits menteurs deviennent inutiles). Tu pourras alors arbitrer la priorité de Phase A/B/C avec des données fraîches.
> - **Pas de dépendance code stricte** : Phase A, B, C de ce plan sont indépendantes du plan jumeau et peuvent en théorie être lancées en parallèle si tu n'as pas le luxe d'attendre.

---

## 0. TL;DR pour agents qui arrivent en cours de route

- **Cause primaire** (résolue côté parser, ne pas dupliquer) : le parser des
  highlight events scannait les bytes au lieu des bits. Conséquence : tous les
  matchs synchronisés par la pipeline parallèle après la mise en service du
  parser bogué ont 0 ligne dans `highlight_events`, `weapon_kills`,
  `killer_victim_pairs`. **Parser fix livré** (commits `64f6720b` + `34c7f646`),
  **replay tool livré** (`cmd/replay_highlight_events`, 27 matchs JGtm
  re-syncés). Voir [`PLAN_HIGHLIGHT_EVENTS_BACKFILL.md`](PLAN_HIGHLIGHT_EVENTS_BACKFILL.md)
  pour la suite (industrialisation HTTP + CLI + golden fixture E2E).

- **Causes secondaires indépendantes** (objet de ce plan) :
  - **RC4** — `asset_translations` n'est plus alimentée pour les UUID
    récents : le `catalog_fetcher_service` upsert ses résolutions dans
    `playlists_catalog`/`maps_catalog`/etc. **mais ne propage pas** les
    `Names` multilingues vers `asset_translations`. Conséquence : home et
    match-view échouent toutes les deux à résoudre — sauf que le home a un
    fallback `lang='en-US'` qui parfois trouve une vieille entrée résiduelle.
    **C'est un bug architectural, pas une asymétrie de fallback à équilibrer**.
    La Phase A est une refonte : propager `Names` au catalog upsert, puis
    introduire un resolver unifié `ResolveAssetName`.
  - **RC5** — gamertags absents de `match_participants` même quand
    `xuid_aliases` les contient (pas de backfill cross-match).
  - **RC6** — handler match retourne 404 au lieu de dégrader sur données
    partielles.
  Ces bugs pré-existaient et resteront après le fix parser. Phases A/B/C
  ci-dessous.

- **Tests insuffisants** : aucun test d'invariant n'aurait capté la régression
  parser. La Phase 4 du plan jumeau (golden fixture E2E) corrige ce manque
  pour la chaîne sync. Phase D **réduite** ici : seul l'invariant 4
  (home/match-view consistency) reste, lié à RC4.

---

## 1. Diagnostic empirique

### 1.1. Outillage

Outil créé : [apps/go-api/cmd/diag_recent_match_sync/main.go](apps/go-api/cmd/diag_recent_match_sync/main.go)

Modes :
- `go run -tags cgo ./cmd/diag_recent_match_sync <match_id>` — détail d'un match
- `go run -tags cgo ./cmd/diag_recent_match_sync --recent N` — détail des N derniers
- `go run -tags cgo ./cmd/diag_recent_match_sync --summary N` — tableau résumé

Le diag ouvre `shared_matches_v2.duckdb` et `data/global/xbox_aliases.duckdb` en
read-only et tolère que le serveur tienne le verrou sur le global (skip
gracieux dans ce cas).

### 1.2. Frontière de régression observée

Pivot : entre **2026-04-06 23:00** et **2026-04-23 22:16**.

Avant la régression :
```
2026-04-06 23:00 | he=213 wk=86 kv=86 | map_name=Banished Narrows | pair_name=Community:Team Slayer… | bf=3211247
2026-04-06 23:11 | he=218 wk=92 kv=92 | map_name=Curfew           | pair_name=Community:Team Slayer… | bf=3211247
```

Après la régression :
```
2026-04-23 22:16 | he=0  wk=0  kv=0  | map_name=63d634be-… | pair_name=2ec67322-…    | bf=2162688
2026-05-05 23:16 | he=0  wk=0  kv=0  | map_name=2890782c-… | pair_name=9cb45f34-…    | bf=2162688
```

`bf=2162688 = MBitEvents (1<<16) | MBitWeaponKills (1<<21)` — bits "loaded"
positionnés alors que **0 ligne n'a été insérée**. C'est un mensonge en DB qui
empêche `events_heal` de retenter (le filtre `WHERE events_loaded=FALSE` dans
[events_heal.go:37](apps/go-api/internal/sync/events_heal.go#L37) skippe
définitivement ces matchs).

### 1.3. Symptômes utilisateur observés

| Symptôme | Cause directe | Résolu par |
|---|---|---|
| Scoreboard et rencontres → xuid au lieu de gamertag | `match_participants.gamertag` NULL pour 5-7 sur 8 joueurs | parser fix (alias depuis events) + RC5 (cross-match backfill pour matchs sans film exploitable) |
| Armes pas dans l'expander | `weapon_kills` vide | parser fix |
| Antagonistes absents | `killer_victim_pairs` vide | parser fix |
| Onglet Combat vide | `highlight_events` vide | parser fix |
| Courbe engagement vide | `match_intensity` non calculé (faute d'events) | parser fix |
| Dernière session = 1 match | `sessions.session_id` non recalculé | parser fix (post-sync conditionnel se déclenchera) |
| Tuiles sans image map | `match_registry.map_id` peut être NULL | RC4 (résolution UUID→nom + map_images_registry) |
| Bannière joueur absente | xuid joueur sans alias | RC5 |
| « Match introuvable » en historique | handler 404 sur partial data | RC6 |
| Match-view affiche UUID au lieu de nom de map/mode | asymétrie fallback EN entre home et match-view | RC4 |

---

## 2. Root causes confirmées (file:line)

### RC1 (parser fix — user-owned)
Parser highlight events : scan bytes vs bits, retourne 0 events sur films
valides. → tous les downstream (weapon_kills, killer_victim_pairs,
match_intensity, alias coverage, sessions) reposent dessus.

### RC2 (lying bits — risque résiduel post-parser)
[events_heal.go:71-78](apps/go-api/internal/sync/events_heal.go#L71-L78),
[engine.go:1054-1064](apps/go-api/internal/sync/engine.go#L1054-L1064),
[backfill_weapons.go:74-76](apps/go-api/internal/sync/backfill_weapons.go#L74-L76)
appellent `MarkEventsLoaded` / `MarkWeaponKillsDone` sur erreur réseau ou data
vide. Une fois le parser corrigé, le risque devient marginal (uniquement
sur erreurs API transitoires), couvert par les tests d'invariants en RC7.

### RC3 (cascade dependency)
Sous-cas de RC2 : `getKillsForPlayer` lit depuis `killer_victim_pairs` ; si
events n'a rien inséré, les kills sont vides, `MarkWeaponKillsDone(false)` est
appelé quand même. Disparaît quand RC1 est résolu.

### RC4 (résolution noms d'asset cassée — racine architecturale) — INDÉPENDANT, à fixer

**Diagnostic révisé après investigation approfondie** : ce n'était pas une
asymétrie acceptable home/match-view, c'est un design défaillant qui touche
les deux pipelines.

Les `map_id`, `pair_id`, `playlist_id`, `game_variant_id` sont **invariants**.
Une fois résolus, leurs noms canoniques (multilingues) devraient vivre dans
`asset_translations` et être servis par un resolver unifié. Or :

1. **Au sync** ([transforms.go:100-144](apps/go-api/internal/sync/transforms.go#L100-L144)) :
   `MapVariant.PublicName` (qui peut être un UUID Discovery UGC) est écrit brut
   dans `match_registry.map_name`. **Aucun JOIN avec `asset_translations` à
   l'INSERT**. Fallback : `coalesceStrPtr(row.MapName, row.MapID)` — si
   `PublicName` est vide, on stocke l'UUID directement.

2. **Le `catalog_fetch_queue`**
   ([catalog_enqueue.go:27-77](apps/go-api/internal/sync/catalog_enqueue.go#L27-L77))
   enqueue bien les 4 types (`playlist`, `pair`, `map`, `game_variant`).

3. **Le consumer**
   ([catalog_fetcher_service.go:119-149](apps/go-api/internal/service/catalog_fetcher_service.go#L119-L149))
   appelle `TitleCatalogAdapter.FetchPlaylist/FetchMap/FetchPair/FetchGameVariant`
   et obtient un `CanonicalPlaylist/Map/Pair/GameVariant` qui contient
   `Names map[string]string` (multilingue, voir
   [canonical/catalog.go:67-76](apps/go-api/internal/games/canonical/catalog.go#L67-L76)).

4. **Le bug racine** : le consumer upsert dans `playlists_catalog`,
   `maps_catalog`, etc. avec **uniquement `name_canonical` (EN)**. Il
   **n'appelle jamais** `UpsertAssetTranslation`
   ([metadata_repo_assets.go:97-119](apps/go-api/internal/platform/duckdb/metadata_repo_assets.go#L97-L119))
   pour persister les `Names` dans `asset_translations`. Donc la table de
   lookup multilingue reste vide pour tous les UUID nouvellement résolus.

5. **Conséquence côté lecture** :
   - Home tile : a un fallback EN
     ([home_repo.go:737-777](apps/go-api/internal/platform/duckdb/home_repo.go#L737-L777))
     qui parfois récupère une vieille entrée `lang='en-US'` venant d'un sync
     antérieur — d'où l'apparence de fonctionner.
   - Match-view : ne fait que FR, échoue silencieusement, affiche l'UUID.
   - **Les deux sont buguées** : aucune ne va lire `playlists_catalog.name_canonical`
     ou ne demande au catalog de remplir `asset_translations`.

6. **Le "asset kinds" pattern**
   ([internal/assets/](apps/go-api/internal/assets/)) couvre les binaires
   (medals, weapons, ranks, images). Il n'a **aucun Kind** pour les noms
   d'assets et `Resolver` n'a pas de méthode pour ça —
   [resolver.go:8-29](apps/go-api/internal/assets/resolver.go#L8-L29) traite
   uniquement images + métadonnées JSON.

**Conclusion** : il manque un resolver unifié pour les noms d'asset, ET le
catalog_fetcher ne propage pas ses résolutions vers `asset_translations`.
Toute cascade FR→EN côté lecture est un patch sur un trou architectural plus
profond.

Concerne : `map_name`, `pair_name`, `playlist_name`, `game_variant_name`.

### RC5 (xuid_aliases partial coverage) — INDÉPENDANT, à fixer
[engine.go:944](apps/go-api/internal/sync/engine.go#L944) écrit
`xuid_aliases` uniquement si `p.Gamertag != nil && p.Gamertag != ""`. L'API
`GetMatchStats` ne retourne le gamertag que pour les joueurs « primaires » du
match — les autres viennent du parsing des highlight events
([engine.go:1085-1094](apps/go-api/internal/sync/engine.go#L1085-L1094)). Sur
les matchs dont le film a expiré côté Halo (matchs anciens), même après le fix
parser, l'alias ne pourra pas être récupéré. Or `xuid_aliases` peut contenir
l'alias pour ce xuid via un autre match plus récent (source mutualisée). On
ne fait jamais de backfill cross-match.

### RC6 (handler 404 sur partial data) — INDÉPENDANT, à fixer
Localisation à confirmer : probablement
[apps/go-api/internal/service/match_view_service.go:GetMatchView](apps/go-api/internal/service/match_view_service.go)
qui propage `ErrNoRows` quand une requête sœur (Q12 scoreboard, Q13 meta, etc.)
ne trouve rien. Le handler
[apps/go-api/internal/api/handlers/match_view_test.go](apps/go-api/internal/api/handlers/match_view_test.go)
référence `match_not_found`. Le front affiche "Match introuvable ou erreur de
chargement" ([MatchViewPage.tsx:77](apps/web/src/features/match-view/MatchViewPage.tsx#L77))
sur n'importe quel échec downstream. Un match avec `match_registry` OK mais
secondaires vides ne devrait pas crash : on doit dégrader.

---

## 3. Hors-scope de ce plan (autres agents / user)

- **Fix du parser highlight events** (bytes→bits) — réalisé par l'utilisateur.
- **Repair de la BDD existante** — réalisé par l'utilisateur (re-sync delta + heal).
- **Reset des bits menteurs** sur les matchs déjà cassés — fait dans la repair user.
- **Lying bits structuraux** (RC2/RC3) — supprimés par RC1 ; tests d'invariants
  en RC7 couvrent le risque résiduel sans modifier le code.

Si un autre agent veut toucher à ces sujets, vérifier d'abord la mémoire de
l'utilisateur et le commit log : ne pas dupliquer.

---

## 4. Plan d'action

### Phase A — RC4 : résolution unifiée des noms d'asset (refonte)

Cette phase est divisée en 4 sous-tâches qui s'enchaînent. Objectif : éliminer
les SQL `lang IN (...)` dispersés et garantir qu'`asset_translations` est
peuplée par le `catalog_fetcher` pour tous les UUID résolus.

#### A.1 — Catalog fetcher : propager `Names` dans `asset_translations`

**Fichier** :
[apps/go-api/internal/service/catalog_fetcher_service.go](apps/go-api/internal/service/catalog_fetcher_service.go)
— `processEntry()` (lignes 119-149).

**Changement** : après chaque `upsertPlaylist/upsertMap/upsertPair/upsertGameVariant`,
itérer sur `canonical.Names` et appeler
`UpsertAssetTranslation(assetID, assetType, lang, name)`. Le consumer connaît
les langs disponibles (cf.
[canonical/catalog.go:67-76](apps/go-api/internal/games/canonical/catalog.go#L67-L76))
— il faut les pousser vers la table de lookup mutualisée.

**Test** : `catalog_fetcher_service_test.go` — mock `TitleCatalogAdapter`
qui retourne `CanonicalPlaylist{Names: {"en-US":"Quick Play","fr-FR":"Partie rapide"}}`.
Après `Drain`, assertion sur `asset_translations` : 2 lignes (en-US, fr-FR)
avec les bons noms.

#### A.2 — Resolver canonical unifié

**Fichier** :
[apps/go-api/internal/platform/duckdb/metadata_repo_assets.go](apps/go-api/internal/platform/duckdb/metadata_repo_assets.go)

**Nouvelle fonction** :
```go
// ResolveAssetName retourne le meilleur nom disponible pour (assetType, assetID).
// preferredLangs : liste ordonnée des langs souhaitées (ex. ["fr-FR","fr","en-US","en"]).
// Renvoie name = "" si aucun match.
func (r *MetadataRepo) ResolveAssetName(ctx context.Context, assetType, assetID string, preferredLangs []string) (name, lang string, err error)
```

Implémentation : une seule requête `SELECT name, lang FROM asset_translations
WHERE asset_type = ? AND asset_id = ?` puis tri en mémoire selon
`preferredLangs`, fallback sur n'importe quelle lang trouvée si rien dans la
préférence.

**Test** : table-driven sur DuckDB :memory: — assets avec différentes combos
de langs présentes/absentes ; vérifier la priorité et le fallback.

#### A.3 — Migration des callers (home + match-view + canonical)

**Fichiers** :
- [home_repo.go](apps/go-api/internal/platform/duckdb/home_repo.go) :
  supprimer `loadHomeAssetTranslationNames` + `loadHomeAssetTranslationNamesEN`
  ; remplacer par un appel unique à `ResolveAssetName` pour chaque asset.
  Ajuster `applyCanonicalAssetFR/EN` en conséquence (peut devenir un seul
  `applyCanonicalAssetLabel`).
- [match_view_repo.go](apps/go-api/internal/platform/duckdb/match_view_repo.go) :
  supprimer `lookupMapNameFR`, `lookupModeNameFR`, `lookupPlaylistNameFR` ;
  remplacer par appel à `ResolveAssetName` avec `preferredLangs` selon locale
  utilisateur. Le service décide juste « locale = fr » → préférences
  `["fr-FR","fr","en-US","en"]`.
- [match_view_service.go:buildMatchHeader](apps/go-api/internal/service/match_view_service.go)
  : éliminer la cascade `MapNameFR → MapName brut`. Si `ResolveAssetName`
  retourne vide, dernier recours = `match_registry.map_name` brut, mais ne
  l'afficher que si **pas un UUID** (regex
  [halo_infinite/adapter_asset_urls.go:uuidRe](apps/go-api/internal/games/halo_infinite/adapter_asset_urls.go#L19)).
  Si UUID → afficher placeholder neutre (ex. "Carte inconnue") plutôt que
  l'UUID brut.

**Test** : nouveau test bout-en-bout dans `match_view_dominance_test.go` —
asset_translations contient seulement `lang='en-US'` ; locale FR ; le service
retourne le nom EN (pas l'UUID).

#### A.4 — Backfill one-shot pour réparer l'historique

**Fichier nouveau** :
[apps/go-api/cmd/repair_asset_translations/main.go](apps/go-api/cmd/repair_asset_translations/main.go)

Itère sur `playlists_catalog`, `maps_catalog`, `map_mode_pair_definitions`,
`game_variants_catalog` ; pour chaque ligne, lit `Names` (ou
`name_canonical` si `Names` absent en DB legacy) et upsert dans
`asset_translations`. Idempotent. Ne fait aucun appel API — juste un
re-routage des données déjà résolues vers la table de lookup unifiée.

**Test** : DuckDB :memory:, seed catalogs avec quelques entrées Names
multilingues, run main → asset_translations a toutes les entrées attendues.

**Critère de fin Phase A** :
- Le diag CLI sur les 5 matchs récents montre des `map_name` / `pair_name`
  affichés en clair (pas d'UUID) côté match-view ET côté home tile.
- Test unitaire qui aurait failli avant le fix : passe.
- `go vet ./...` clean. `loadHomeAssetTranslationNames*` et `lookupMapNameFR`
  supprimés (grep -r vide).

### Phase B — RC5 : xuid_aliases backfill cross-match

**Fichiers** :
- [internal/sync/aliases_backfill.go](apps/go-api/internal/sync/aliases_backfill.go) —
  nouveau, helper `BackfillMissingGamertagsFromAliases(sharedDB, globalDB)`.
  Requête :
  ```sql
  UPDATE match_participants AS mp
  SET gamertag = COALESCE(xa_shared.gamertag, xa_global.gamertag)
  FROM xuid_aliases xa_shared
  LEFT JOIN xuid_aliases_global.xuid_aliases xa_global ON xa_global.xuid = mp.xuid
  WHERE mp.xuid = xa_shared.xuid
    AND (mp.gamertag IS NULL OR mp.gamertag = '' OR mp.gamertag = mp.xuid)
    AND mp.xuid NOT LIKE 'bid(%'
    AND COALESCE(xa_shared.gamertag, xa_global.gamertag) IS NOT NULL
  ```
- [internal/sync/engine.go:runConditionalPostSync](apps/go-api/internal/sync/engine.go#L606) —
  appel post-sync (best-effort, log warn si erreur).
- [cmd/repair_aliases_backfill/main.go](apps/go-api/cmd/repair_aliases_backfill/main.go) —
  CLI one-shot pour rattraper l'historique.

**Test** : table-driven dans `aliases_backfill_test.go` — 2 matchs sur DuckDB
in-memory, alias présent dans `xuid_aliases` mais pas dans `match_participants`,
appel du backfill, assertion gamertag présent.

**Critère de fin** : après une sync, aucune ligne `match_participants` n'a
`gamertag` NULL/vide pour un xuid dont l'alias existe par ailleurs.

### Phase C — RC6 : match handler graceful sur partial data

**Étape 1 — investigation** : tracer le chemin exact qui retourne 404 / 500.
Lire `MatchViewService.GetMatchView` et le handler. Identifier les sql.ErrNoRows
qui remontent. Comprendre la sémantique actuelle.

**Étape 2 — design** :
- Ajouter `MatchViewResponse.IsPartial bool` et `PartialReasons []string` dans
  [domain/match_view.go](apps/go-api/internal/domain/match_view.go).
- Le service ne retourne 404 que si `match_registry` est totalement absent
  (vrai "match introuvable"). Sinon il dégrade : tabs vides + drapeau partiel.
- Front (`MatchViewPage.tsx`) : si `is_partial`, afficher un bandeau dégradé
  avec les raisons (ex. "events non chargés", "scoreboard partiel"), au lieu
  de l'écran d'erreur full.

**Test** : `match_view_test.go` — match avec `match_registry` OK mais 0
participants → 200 + `is_partial=true` + `partial_reasons=["scoreboard_empty"]`.

**Critère de fin** : scroller dans l'historique sur un match cassé n'affiche
plus "Match introuvable", mais une page dégradée explicite.

### Phase D — Tests d'invariants (réduite, 2026-05-08)

> **Mise à jour 2026-05-08** — Périmètre réduit après l'arrivée du plan jumeau
> [`PLAN_HIGHLIGHT_EVENTS_BACKFILL.md`](PLAN_HIGHLIGHT_EVENTS_BACKFILL.md). Les
> invariants 1, 2, 3, 5 listés initialement sont absorbés par :
>
> - **Golden test Phase 4** du plan jumeau (`TestSyncPipeline_GoldenMatch_AllTablesPopulated` + variante `NoFilm_CascadeRespected`) : couvre invariants 1 et 2 sur fixture API authentique.
> - **Tests Phase 1bis** du plan jumeau (`TestInsertHighlightEventsFromData_DoesNotMarkKVLoadedOnInsertFailure`, `TestHealEventsForRecentMatches_DoesNotMarkOnParseAnomaly`) : couvrent invariant 3 (lying bits detector).
> - **Golden test Phase 4** du plan jumeau : vérifie aussi l'invariant 5 (alias coverage post-sync) sur le match canonique. L'**implémentation** du backfill cross-match reste dans la Phase B ci-dessus (RC5).
>
> **Seul l'invariant 4 ci-dessous reste à charge de ce plan**, car il dépend de RC4 (cascade i18n match-view).

**Test home/match-view consistency** :
`internal/service/match_view_service_test.go` + un test dans
`match_view_dominance_test.go` — pour un même `match_id` avec un
`asset_translations` qui n'a que `lang='en-US'`, home et match-view
retournent **le même** label de map/mode. Capture l'asymétrie historique
identifiée en RC4.

---

## 5. Application de la grille plan-review

### Architecture
- RC4 → repo (lecture asset_translations) + service (cascade i18n) — couches respectées.
- RC5 → service (post-sync helper) + cmd (one-shot backfill) — pas de SQL inline dans handler.
- RC6 → service (sémantique dégradée) + handler (status code) + frontend (bandeau).

### Multi-titres
- RC4 : la cascade FR→EN→raw est title-agnostic (tous les titres ont la même
  table `asset_translations` dans `metadata.duckdb`). Pas de capability requise.
- RC5 : `xuid_aliases` est shared / global → multi-titres OK.
- RC6 : le drapeau `is_partial` est domain-level, applicable à tout titre.

### Tests
- Unitaires sur cascade i18n (RC4).
- DuckDB :memory: pour RC5.
- httptest pour RC6 handler.
- E2E (mock client) pour invariants.
- Frontend test pour bandeau partial.

### Logging
- RC4 : aucun log nouveau (pure lecture, fallback existant).
- RC5 : `slog.InfoContext(ctx, "aliases_backfill: rows", "updated", n)`.
- RC6 : `slog.WarnContext(ctx, "match_view: partial data", "match_id", id, "reasons", reasons)`.

### Frontend
- RC6 nécessite : nouveau composant bandeau, traductions FR + EN, tests
  vitest sur le rendu conditionnel.

---

## 6. Critères de complétion globaux

- [ ] Phase A merged, tests verts, page match récente affiche le bon nom de
      map/mode même quand `match_registry.map_name = UUID`.
- [ ] Phase B merged, tests verts, après sync delta + backfill aucun
      `match_participants.gamertag` NULL/xuid pour un xuid alias-couvert.
- [ ] Phase C merged, tests verts, plus de "Match introuvable" sur match
      partiel.
- [ ] Phase D (réduite) : invariant 4 (home/match-view consistency) ajouté et
      vert. Les invariants 1/2/3/5 sont délégués au plan jumeau
      [`PLAN_HIGHLIGHT_EVENTS_BACKFILL.md`](PLAN_HIGHLIGHT_EVENTS_BACKFILL.md)
      (Phase 4 golden test + Phase 1bis bitmasks).
- [ ] `.ai/thought_log.md` complété avec date 2026-05-08, statut Complété,
      résultats observés, lien vers ce plan.
- [ ] `go test ./... && go vet ./...` clean.
- [ ] `npm run typecheck && npm run lint && npm run test` clean côté
      `apps/web/`.

---

## 7. Notes pour cross-référence avec autres agents

Si un autre agent travaille sur :

- **Le parser highlight events / la chaîne sync events / les bitmasks** → c'est
  le scope du plan jumeau
  [`PLAN_HIGHLIGHT_EVENTS_BACKFILL.md`](PLAN_HIGHLIGHT_EVENTS_BACKFILL.md). Le
  parser bit-aligné est livré (commits `64f6720b` + `34c7f646`) ; les Phases 1
  à 5 du jumeau industrialisent le replay (HTTP + CLI + golden fixture E2E) et
  fixent les bitmasks menteurs events+kvp. Ce plan-ci ne touche plus à ces
  sujets.
- **Le catalog_fetch_queue / asset_translations** → **scope partagé avec ce
  plan**. La Phase A.1 modifie
  [catalog_fetcher_service.go:processEntry](apps/go-api/internal/service/catalog_fetcher_service.go#L119-L149)
  pour propager `CanonicalPlaylist.Names` (et équivalents map/pair/variant)
  dans `asset_translations` via `UpsertAssetTranslation`. Si un autre agent
  veut aussi toucher `processEntry`, coordonner pour ne pas dupliquer ; sinon
  contournement via la phase A.4 (backfill one-shot lit les catalogs et écrit
  asset_translations sans toucher au sync).
- **Le pattern asset/Kind dans `internal/assets/`** → ne pas confondre avec
  ce plan. `Kind` couvre les binaires (medals, ranks, weapons, images). Les
  noms d'assets (map/pair/playlist) ne sont **pas** un Kind dans la version
  actuelle ; le resolver `internal/assets/resolver.go:Resolver` n'a pas
  d'API pour ça. La Phase A.2 introduit `MetadataRepo.ResolveAssetName` au
  niveau platform/duckdb, pas au niveau assets/Kind. Si un agent juge
  pertinent d'élever ça à un Kind à terme, c'est un plan séparé.
- **La pipeline de sync parallel** → ne pas modifier les bits `MBit***` dans le
  cadre de ce plan. Les fixes events+kvp sont dans le jumeau (Phase 1bis).
  L'audit lecture seule des autres bits (`MBitAssets`, `MBitAliases`,
  `MBitPVEStats`, `MBitWeaponKills`, `PBit*`) est en Phase 1ter du jumeau ;
  si elle révèle des menteurs, un plan dédié sera ouvert.
- **Le handler match-view** → coordonner sur RC6 ; le contrat actuel (404 sur
  manque de data) est rompant pour le front, doit migrer vers 200 + flag
  partial.

---

## 8. Historique des changements liés (déjà commités sur cette branche)

- 2026-05-07 : fix mode/map résolution match-view (Q13 lit `pair_name_fr`,
  ajout `lookupMapImageURL` via `map_images_registry`). Fait sur cette
  conversation, déjà appliqué. Reste insuffisant pour les matchs avec
  `asset_translations` lang='en-US' uniquement → traité par Phase A.
