# HANDOFF — Sélection par titre + placement + activation Halo 5

> **Créé** : 2026-06-19 (session multi-titre / Halo 5 Phase 1).
> **Branche** : `feat/multititre-peripherie` (branch-only, PAS de PR).
> **Dernier commit** : `930a3d8c3`.
> **Worktree** : `c:/Users/Guillaume/Downloads/Scripts/levelup-multititre` (les données — db_profiles.json, data/, tokens — vivent dans le repo principal `LevelUp-go-migration` ; pour exécuter un CLI/serveur lisant les vraies données : `LEVELUP_REPO_ROOT=c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration`).
> Handoffs liés : `.ai/HANDOFF_HALO5_EXPERIMENTAL.md` (adapter h5, sonde), `.ai/AUDIT_MULTITITRE_COVERAGE.md`.

## 0. TL;DR — état

**Halo 5 Phase 1a (adapter read-only) = LIVRÉ + reviewé adversarialement + durci + active-ready.** Le caveat « 343 sert-il h5 ? » est levé (sonde live OK, SpartanToken v4 accepté). Avant d'ACTIVER Halo 5 (le servir live dans l'app), l'utilisateur a posé **2 pré-requis** :
1. **Nombre de matchs de placement classés par titre** (HINF=5, Halo 5=10) → **FAIT** (Pass A back `930a3d8c3` + Pass C front nettoyage `f6d8b0b87`).
2. **Sélection par titre** : onboarding (choisir jeux + nb matchs/titre) + Réglages (activer/pause, **min 1**) → **FAIT** (Pass B back `0adb4efaa`→`c42e4344d` ; Pass C front `f6d8b0b87`/`01c080c1f`/`882f6d7f9`).
3. **GATE 2026-06-19 — Canonical MatchEvents** : la sonde Halo 5 a prouvé une **timeline d'events NATIVE** (kill-feed + arme-par-kill + médailles + positions). Décision user : **canoniser ce modèle AVANT d'activer**. → **plan dédié `.ai/PLAN_CANONICAL_MATCH_EVENTS.md`**. **✅ FAIT 2026-06-20 — Phases 0→3b livrées/poussées (commits `f76d008da`→`8b4c451ba`), audit adversarial 4/4 « done ». GATE LEVÉ.**

**Gate MatchEvents levé** → place à **activation 1b** (registration générique + flip `status=active` + vérif live). **⚠ Nouveau pré-requis QUALITÉ (pas un blocage dur)** : **damage model par titre** — `.ai/PLAN_DAMAGE_MODEL_PER_TITLE.md` (la baseline `225` PV Infinite est câblée dans le compute rendement/résistance ; Halo 5 = `115` ; KPI mal échelonnés sinon). À traiter avec l'activation.

**État** : Pass B/C + **Canonical MatchEvents (Phases 0→3b)** = COMPLETS et poussés (`feat/multititre-peripherie`). **Reste avant/pendant activation : (a) damage model par titre (`PLAN_DAMAGE_MODEL_PER_TITLE.md`, qualité KPI), (b) activation 1b.** **Halo Infinite byte-identique** (tout le code multi-titre est additif/gaté).

## 1. Décisions produit (VERROUILLÉES — ne pas re-litiguer)

- **Désactiver un jeu pour un joueur = PAUSE** (garde les données sur disque, réactivable sans re-sync) **+ bouton PURGE séparé** (« supprimer les données de ce jeu »). → mécanisme = flag `sync_enabled` + action de purge distincte.
- **Les DEUX points en entier** (placement + sélection, back ET front) avant l'activation.
- **Halo 5 placement = 10** (valeur historique ; sonde montrait `MeasurementMatchesLeft:7` sur 10) ; **Halo Infinite = 5** (depuis Season 3, table par-saison qui prime).

## 2. INSIGHT CLÉ : `db_profiles.json` v3 EST le substrat d'activation par titre

Un « profil joueur » est en réalité un **couple (titre, gamertag)** :
```json
{ "version":"3.0", "admin":"<gt>",
  "profiles": { "<title_slug>": { "<Gamertag>": { "db_path","xuid","waypoint_player" } } } }
```
- `loadPlayersV3` (`internal/config/config_players.go:139-171`) aplatit cette map en N `PlayerSummary`, chacun portant **UN** `TitleSlug` (`internal/domain/bootstrap.go:42`). Le couple est matérialisé à `config_players.go:166`.
- **« Activer un titre pour un joueur » = présence d'une entrée `profiles[<slug>][<gamertag>]`.** Le sync suit nativement : le scheduler (`internal/scheduler/auto_sync.go:568` `LoadPlayers()` sans filtre, boucle `:618`) ET le watcher (`cmd/server/main.go:1824` → `daemon.Start` → `daemon.go:368 initPlayers`) consomment la même liste de couples.
- `registry.Active()` (côté TITRE, infra/provisioning, `registry.go:220`, `main.go:419/:1487 provisionAdditionalActiveTitles`) est **totalement découplé** de `LoadPlayers` (côté JOUEUR). Activer un titre provisionne ses DB mais n'enrôle AUCUN joueur tant qu'aucun couple `(joueur, slug2)` n'existe.

→ La feature « sélection par titre » se greffe naturellement : étendre l'entrée v3 + un writer + un filtre au chokepoint `loadPlayersV3`.

## 3. PASS B — Sélection par titre (BACK)

Ordre conseillé (chaque étape = commit vert ; B.1-B.3 sont additifs/inertes → Halo byte-identique tant qu'aucun endpoint ne pose `sync_enabled=false`).

### B.1 — Modèle de données (additif)
- Étendre `dbProfileEntry` dans les DEUX structs miroir (dupliquées) :
  - `internal/config/config_players.go:26`
  - `internal/service/profile_service.go:40`
  ```go
  SyncEnabled       *bool `json:"sync_enabled,omitempty"`        // nil/true = actif, false = pause
  InitialMaxMatches int   `json:"initial_max_matches,omitempty"` // 0 = défaut 200
  ```
- Propager sur `domain.PlayerSummary` (`internal/domain/bootstrap.go:35`) : `SyncEnabled bool` (résolu nil→true) + `InitialMaxMatches int`.

### B.2 — Writer `db_profiles.json` (LE CODE NE FAIT QUE LIRE aujourd'hui)
- Nouveau store, ex. `internal/platform/dbprofiles/store.go` : lecture v3 + **écriture ATOMIQUE** (write temp + rename) + lock fichier (`sync.Mutex` process + lock OS si multi-process). API : `AddEntry(slug, gamertag, entry)`, `RemoveEntry(slug, gamertag)`, `SetSyncEnabled(slug, gamertag, bool)`, `Get/List`.
- ⚠️ Préserver les champs inconnus / la structure (round-trip fidèle), comme le fait `platform/settings/store.go` avec son `raw map[string]json.RawMessage`.

### B.3 — Filtre au chokepoint
- `loadPlayersV3` (`config_players.go:155`, juste après `for gamertag, p := range titleProfiles`) : **skip si `SyncEnabled != nil && *SyncEnabled == false`** → couvre AUTOMATIQUEMENT scheduler + watcher + provisioning boot (un seul point).
- Alternative (si on veut garder le couple visible mais non synchronisé) : filtrer dans `checkSyncPreconditions` (`auto_sync.go:840`) au lieu de `loadPlayersV3`. **Recommandé : `loadPlayersV3`** (chokepoint unique).

### B.4 — Onboarding multi-titre (API)
- `POST /setup/players` (`internal/api/handlers/setup.go:106 handleCreatePlayer`, injecte `titleSlug` du ctx à `:140`) → étendre pour accepter une **liste de titres + max_matches par titre**. Modifier `domain.CreatePlayerProfileRequest` (`internal/domain/settings.go:139` : `Gamertag/XUID/ProfileMode/TitleSlug`) → ajouter `Titles []TitleSyncChoice{Slug string, MaxMatches int}` (ou le front appelle N fois). `ProfileService.CreatePlayer` (`internal/service/profile_service.go:49`) écrit une entrée par titre choisi (via le writer B.2).
- Sync initial : `POST /sync/initial` (`internal/api/handlers/sync_handler.go:308 StartInitialSync`, `max_matches` défaut **200** borné 1-2000 à `:334-339`, `InitialSyncStartRequest` `settings.go:154`) → **ajouter `TitleSlug`** au request (sinon ambigu si le gamertag existe sous 2 titres) + utiliser `InitialMaxMatches` du profil au lieu du 200. Poser le slug au ctx (`ctxkeys.WithTitleSlug`, déjà utilisé par auto_sync) pour cibler la bonne DB.

### B.5 — Réglages (API)
- Nouvel endpoint **toggle** `sync_enabled` par `(player, title)` (ex. `POST /settings/titles/toggle {gamertag, title_slug, enabled}`) → writer B.2.
- Nouvel endpoint **purge** (ex. `DELETE /settings/titles/{slug}/data?gamertag=`) → retire l'entrée + supprime la player DB du titre (`PathResolver.PlayerDBPath(slug, gamertag)` + le dossier `data/titles/<slug>/players/<gt>/`).
- **Validation « min 1 »** (back, 409/400) : refuser de désactiver/purger le DERNIER titre actif d'un gamertag.
- Note : `app_settings.json` est GLOBAL — ne PAS y mettre la sélection (qui est par joueur). L'overlay per-titre PMT-4 (`settings.go:172 extractPerTitleOverlay`, limité à `show_progression`/`outcome_exclude_*`) varie par titre, pas par joueur → inadapté. La voie = db_profiles.json (B.2).

### B.6 — Tests
- `config_players` : round-trip writer + filtre `sync_enabled`. `setup`/`sync_handler` : httptest multi-titre + min-1. `profile_service` : création N titres.

### B.7 (⚠️ RISQUÉ, séparable) — Fixes watcher pour le LIVE multi-titre
Bugs réels trouvés (n'impactent que le watch LIVE simultané de 2 titres pour un même joueur — donc **post-activation**) :
- **Map keyée par gamertag SEUL** (`internal/watcher/daemon.go:113`, `:334 AddPlayer`, `:385 initPlayers`) → `(GT, titreA)` et `(GT, titreB)` s'écrasent (un seul watcher survit). **Re-keyer par `gamertag|titleSlug`** (+ `playerCancels`, `UpdateSubscriptions`, `broadcastPresenceActive`).
- **`watcherSlug := title.DefaultSlug` hardcodé** (`cmd/server/main.go:1881`) pour le `LiveRefreshFactory` (BP/challenges) → écrit dans les DB halo_infinite même pour un autre titre. Dériver `p.TitleSlug` par joueur.
- **`MatchPresence` ne vérifie pas le titre du joueur** (`daemon.go:443`) → un joueur enregistré halo_infinite qui lance Halo 5 ferait sync dans le mauvais titre. Garde : `if td.Slug != pw.titleSlug → OnPresenceInactive`.
- À mener **en passe dédiée** (concurrence, sensible). Le path scheduler, lui, est déjà correct (un couple = un sync).

## 4. PASS C — Sélection par titre (FRONT) + nettoyage placement

- **Onboarding** : nouveau step « sélection des titres » entre `StepPlayer` et `StepInitialSync` dans `apps/web/src/features/setup/` (états du wizard : `SetupPage.tsx` ; `StepInitialSync.tsx:37` a le `200` hardcodé à remplacer par la valeur par titre). Source des titres : `BootstrapResponse.available_titles`. Hooks : `features/setup/queries.ts`.
- **Réglages** : nouvel onglet « Jeux/Titres » (ou bloc dans `GeneralTab.tsx`) — `apps/web/src/routes/settings.tsx` (tableau `SETTINGS_TABS:7`) + `features/settings/SettingsPage.tsx` (liste d'onglets `:137`). Toggle par titre via `ToggleRow` (`_settingsShared.tsx`), **griser le décochage du dernier coché** (min 1) + bouton **purge**. Nouveau hook mutation dans `features/settings/queries.ts`. Data : `availableTitles` (`appShellStore`) × profils du joueur.
- **i18n** : libellés FR+EN (le projet impose un manifest i18n + linter ; cf. `features/settings/i18n`).
- **Nettoyage placement (point 1, reste)** : retirer les fallbacks hardcodés `?? 10` / `: 10` une fois `placement_total` toujours fourni par le back :
  - `apps/web/src/features/home/HomeSkillPeakCard.tsx:141` (`peak.placement_total ?? 10`)
  - `apps/web/src/features/career/CareerRankingBlock.tsx:33` (`rank.placement_total > 0 ? ... : 10`)
  - `ExplorerTargetSeasonCSR.tsx:80-88` est le plus en retard (état binaire « Non classé ») — afficher « X restants » si le DTO le porte.

## 5. ACTIVATION 1b (APRÈS Pass B + C + **Canonical MatchEvents**)

> ✅ **GATE LEVÉ (2026-06-20)** : **Canonical MatchEvents** (Phases 0→3b) livré/poussé/audité (4/4 « done »). Halo 5 peut s'activer avec sa timeline d'events native. **⚠ Pré-requis QUALITÉ restant (pas un blocage dur)** : **damage model par titre** (`.ai/PLAN_DAMAGE_MODEL_PER_TITLE.md`). **MISE À JOUR 2026-06-20 (audit code)** : le compute rendement/résistance est **déjà paramétré** (`games.EffectiveHpToKill(slug)` ; baseline `225` Infinite externalisée, `115` Halo 5 dans `constants.toml`) — Phases 0→2 + copy d'aide title-aware = **DÉJÀ FAIT** (cf. `PLAN_DAMAGE_MODEL_PER_TITLE.md` §0). Ne reste, à l'activation, que **3 items activation-couplés** : (1) **valider `115`** vs vraies données h5 (JGtm) ; (2) **P80 par titre** (recalibrer/dégrader) ; (3) front `ONE_LIFE_DAMAGE` (charts escouade, `not_exposed` h5 Phase 1). Les littéraux `225` restants (Q23 legacy, LUSR Infinite-only, diag offline) **ne sont pas des gaps**.

L'adapter h5 est **active-ready** (`internal/games/halo_5/`) : `NewDataAdapter(NewSpartanTokenSource, logger).WithCapabilities(caps).WithPlacementTotal(desc.PlacementMatches)`. `NewSpartanTokenSource(ctx)` lit le SpartanToken du ctx (`ctxkeys.HaloTokens`). Étapes :
1. **Registration générique au boot** (`internal/api/server.go` ; bloc HI hardcodé `:257-345`) : boucle registry-driven sur `titleRegistry.NonArchived()` (PAS `if slug=="halo_5"` — archlint), enregistrer `RegisterSemantic(games.NewGenericSemanticAdapter(slug, fields, ranks, assets, outcomes))` + `RegisterData(...)` + `RegisterPlayerDataBuilder(slug, func(*PlayerDB) → halo_5.NewDataAdapter(...))` pour les titres ≠ DefaultSlug ayant un FieldMappingSet. **Inerte tant que coming_soon** (RequireActiveTitle `:999` → 503).
2. **Flip `status = "active"`** dans `config/titles/halo_5/title.toml`.
3. **Provisioning** : `provisionAdditionalActiveTitles` (`main.go:1487`) créera les DB warehouse h5 — vérifier qu'un titre **live-only** (adapter sans DuckDB) ne casse pas (migrations h5 vides OK, ou skip).
4. **clearance** : déjà géré (`clearance_url=""` optionnel — fix commit `8438dfd81`).
5. **Vérif LIVE (oracle)** : démarrer le serveur (`LEVELUP_REPO_ROOT=...LevelUp-go-migration`), créer un profil h5 pour JGtm (via onboarding Pass B), switcher sur h5, **confirmer la page Carrière = vrai rang CSR de JGtm** + **le kill-feed (onglet Détails d'un match) = events Halo 5 natifs (arme/positions, capabilities `supported`)** ; surfaces `not_exposed` masquées proprement. ⚠ Si des KPI rendement/résistance Halo 5 sont affichés, ils seront sur l'**échelle Infinite (`225`)** tant que `PLAN_DAMAGE_MODEL_PER_TITLE.md` n'est pas livré — c'est le moment où le mauvais cadrage devient visible (cf. ce plan).
6. **DURABILITÉ events Halo 5 (Phase 4a, `PLAN_CANONICAL_MATCH_EVENTS.md` §4a) — à faire AVEC cette activation** : les events Halo 5 ne viennent QUE de l'API cryptum (fragile, irremplaçable) ; les **persister en append-only capture-on-fetch** dans le warehouse Halo 5 (créé à l'étape 3) au moment où `LoadMatchEvents` fetch — sinon une coupure API = perte définitive du kill-feed/arme-par-kill/positions. Lecture = table d'abord, API en refresh. (Infinite déjà à l'abri via `highlight_events`.) Doctrine append-only : `project_append_only_eradication_campaign`.

## 6. Pièges / rappels

- **Sonde h5** : `cmd/probe-h5` conservé (re-sonde). Recette : host `spartanstats.svc.halowaypoint.com`, header `X-343-Authorization-Spartan` (v4), `User-Agent: cpprestsdk/2.4.0`, query `?auth=st`, **PAS de 343-clearance**, **gamertag brut** (pas `xuid()`).
- **Identité h5 = gamertag** (`Player.Xuid` toujours null) — l'adapter indexe par gamertag (le param `xuid` des Load* = le gamertag côté h5).
- **Capabilities h5 = HONNÊTES** (`config/titles/halo_5/mappings/capabilities.toml` : seul `career.progression` supported en Phase 1a ; le reste `not_exposed` remonte en Phase 2 à mesure du câblage des méthodes). La matrice optimiste cible vit dans `HANDOFF_HALO5_EXPERIMENTAL.md` §2.
- **MÉDIA HALO 5 (captures locales) — distinction à NE PAS confondre (relevé 2026-06-20, user a des médias h5)** : `title.toml` h5 EXCLUT `"media"` de `capabilities` avec le commentaire « forge/media (UGC HINF-shaped) ». ⚠ Ça **conflate deux choses distinctes** : (a) le **média UGC** (maps/variants forge via l'API discovery HINF-shaped) — légitimement exclu ; (b) les **captures LOCALES** du joueur (clips/screenshots Xbox) — pipeline **title-agnostic** (stockage `data/media/{gamertag}`, DB `shared_social` par-titre, déjà title-aware via gap2/gap3) qui **ne dépend PAS** de l'API UGC. Les routes média gatent sur `CapMedia` (`server.go:1115`) → tant que h5 n'a pas `"media"`, la galerie/captures h5 = 403, **les médias h5 du joueur n'apparaissent pas**. **Rien à faire côté user** (fichiers déjà à l'abri sur disque, partagés entre titres). **À l'activation, pour que les captures h5 s'affichent** : (1) ajouter `"media"` aux `capabilities` h5 (title.toml) ; (2) h5 actif + **matchs h5 ingérés** (pour l'association clip↔match par timestamp `delta_seconds` — c'est l'ingestion de cette session, à câbler live) ; (3) un **scan média** en contexte h5 (indexe dans `shared_social` h5 via gap2, associe aux matchs h5). Les libellés map/mode dégradent (COALESCE) tant que la metadata h5 n'est pas peuplée — clips visibles quand même. **Décision à acter** : réviser l'exclusion `media` (séparer UGC-media de captures-locales).
- **GenericSemanticAdapter** (`internal/games/semantic_adapter.go`) : impl PARTAGÉE (le semantic adapter était du boilerplate dupliqué). Migration de `halo_infinite`/`synthetic_title_b` vers ce générique = dette DRY documentée (review #12), non faite (touche du code testé).
- **Contraintes session** : répondre en FR ; pas de Python ; pas de git stash (WIP commit) ; pas d'emoji versionné ; demander avant chaque commit SAUF autorisation autonome explicite du tour ; vitest hors sandbox (`dangerouslyDisableSandbox`) ; CGO = `PATH=/c/msys64/ucrt64/bin` + `CGO_ENABLED=1` ; archlint `no_slug_comparison` interdit `slug == "halo_5"` (utiliser const de package / capability).
- **Outils RE/MCP** : workflows utilisés pour understand/review (chemins ABSOLUS worktree obligatoires dans les prompts d'agents — leur cwd par défaut est le repo principal, autre branche).

## 7. Phase 2 (post-activation, différée)

Adapter h5 : carnage report (scoreboard étendu + CSR pré/post → `match.detail.core`/`scoreboard.extra`), historique (`LoadMatchSummaries` via `GetPlayerMatches` + `mapMatchSummaries` déjà testé → `match.history`), commendations natives (décision B : découpler `citations.engine` surface/mécanisme), warzone/PvE, REQ packs, recalibration engagement. + sync/ingestion DB h5 (Phase 1 = live read-only).
