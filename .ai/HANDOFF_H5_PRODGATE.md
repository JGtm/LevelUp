# HANDOFF — Halo 5 prod-gate (sessions 2026-06-22 → 06-23)

Reprise de la mise en production de Halo 5. Document de reprise : état, commits,
findings non-évidents, reste à faire, commandes opérationnelles.

> **Les sections 2/4/5/8 ci-dessous datent du 2026-06-22. La § 0 MISE À JOUR ci-après
> les SUPERSEDE (état réel au 2026-06-23).**

---

## 0. MISE À JOUR 2026-06-23 — état réel (autoritatif)

**Branche `integration/h5-x-livefetch`. Bilan des 5 axes prod-gate :**

| Axe | État | Détail |
|-----|------|--------|
| **A** match.history | ✅ | livré (35a311130) |
| **B** commendations natives | ✅ | match-detail (Go+front) + **définitions** (table+seed **121/121** nom/icône/catégorie via haloapi, join **vérifié 35/35** sur carnage live) + **totaux à vie** (read source + endpoint `GET /commendations/totals` + front `/players/$slug/commendations`) |
| **C** career rank SR | ✅ | livré (5c6c95d3d) |
| **D** media | ✅ config | capability activée + regex `Halo_5_Guardians` **testée sur les vrais fichiers** (88 entrées, 84 mp4) + `media_captures_base_dir` configuré (`…/Videos/Captures`→`/JGtm`). **Reste** : valider la corrélation des 84 captures (gaté backfill). |
| **E** première synchro | ✅ front | UX « sync en cours » : détection autoritative `hero.kpis.total_matches<=0` + écran rassurant + **auto-poll 30 s** (bascule auto quand la synchro finit) ; backend `/pages/home` renvoie total_matches=0 gracieusement (TestHomeService_GetHomePage_Empty). **Reste** : notif PUSH away-case = suite MT-19. |
| **F** world leaderboard | ✅ | rien (h5 ne déclare pas `world.leaderboard`) |

**Commits de la session 06-23 (au-dessus de d314422c3/018481d40)** :
`466a355fa` media capability · `f6adff6c5` commendations match-detail (Go+front) ·
`6a62cd811` fix harness persist (weapon_kills) · `d67e306e7` thought_log ·
`f632e99a6` définitions natives (table+fetch+read-join) · `ef2060678` persiste Progress
absolu · `cd9d28feb` read layer totaux · `31cf08a73` endpoint totaux ·
`b2fed4019` front totaux · `ba7fbd31d` Axe E front first-sync.

**Findings clés ajoutés** :
1. **Totaux commendations = dernier `progress` ABSOLU** (carnage) du match le plus récent par
   commendation, PAS `SUM(count)` (sous-compterait la baseline pré-sync). On stockait
   seulement le delta `count` → ajout colonne `progress` (ART-safe, INSERT-only) + re-fetch
   requis (INSERT OR IGNORE ne rétro-remplit pas).
2. **Définitions natives** : `GET haloapi.com/metadata/h5/metadata/commendations` (**réponse
   GZIP** — `--compressed` côté curl ; Go décompresse seul). 121 défs, clé `id` = UUID =
   carnage `ProgressiveCommendationDeltas[].Id`. Seed via `cmd/h5-metadata-fetch` (ajout
   `seedCommendations`). Read-join : `commendation_definitions` (metadata) joint en lecture.
3. **Axe E détection** : signal fiable = `hero.kpis.total_matches`, PAS `recent_matches.length`
   (fenêtre ; l'accueil agrège des sources indépendantes — média via RecentMediaRail, identité
   live).
4. **Endpoint title-agnostic SANS slug** : la factory `CommendationTotalsCtx` type-asserte
   l'adapter à `LoadCommendationTotals` (seul *halo_5.DataAdapter l'implémente) → titres sans
   commendations natives → réponse vide. Pattern à réutiliser pour toute surface h5-spécifique.

**En cours (background)** : re-backfill **progress-aware** (`/tmp/h5-backfill.exe JGtm 25`,
log `/tmp/h5_backfill_progress.log`) — fresh DB purgée pour capturer le `progress` absolu (le
1er backfill était count-only). Au 06-23 09:59 : page 34, atteint **2018-08-03** (couvre les 84
captures). À la fin : (a) vérifier `LoadCommendationTotals(JGtm)` non vide + `match_commendations.xuid==pdb.XUID`,
(b) corréler les 84 captures aux matchs (Axe D), (c) go/no-go prod.

**Vérifié 06-23** : full Go test suite `./internal/...` verte ; typecheck + eslint + tests front
(home 34, handlers, service) verts ; drift OpenAPI + contract routes verts.

**Reste après vérif backfill** : (1) notif push Axe E (MT-19) ; (2) nav front vers
`/commendations` (gating coarse-vs-fine : pas de capability h5-only propre — route accessible
directe en attendant) ; (3) décision land `integration`→`main` (auto-deploy, accord explicite).

---

## 1. Topologie & branche

- **Code** : worktree `c:/Users/Guillaume/Downloads/Scripts/levelup-multititre`,
  branche **`integration/h5-x-livefetch`** (NON poussée, NON sur main).
- **Données runtime** : repo `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration`
  (db_profiles.json, auth tokens, `data/titles/halo_5/...`). C'est le `LEVELUP_REPO_ROOT`
  à pointer pour les outils.
- Le **merge h5 × live-fetch** (46 conflits) est committé + **E2E validé** sur données
  réelles (migrations propres, h5 actif, Infinite non régressé, live-fetch OK). NON déployé.
- CGO obligatoire : `export PATH="/c/msys64/ucrt64/bin:$PATH"; export CGO_ENABLED=1` ;
  builds/tests via `go -C apps/go-api ...`.

## 2. Commits livrés (au-dessus du merge, branche integration)

| Hash | Objet |
|------|-------|
| `e0b331e60` | merge h5 × live-fetch (2 parents) |
| `4f1b6a653` | docs thought_log E2E + correction note career_ranks |
| `505a1efe7` | gate `career.rank_catalog` (n'ouvre la DB pour images de rang que si le titre déclare la capability ; supprime le bruit ERROR h5 + future-proof) |
| `92d9cec91` | **fix `include-times=true`** sur GetPlayerMatches → heures de match précises (+ test) |
| `5c6c95d3d` | **Career rank Spartan Rank** (SR 1..152) : fallback 152 garanti + front title-aware (axe C) |
| `35a311130` | **match.history** (axe A) : LoadMatchSummaries lit la DB shared locale |
| `01eef3ae1` | **Commendations natives per-match** (axe B) : ingestion + canonical + match detail |

## 3. Findings clés (NON-évidents — à ne pas re-découvrir)

1. **`include-times=true` REQUIS** sur `/h5/players/{gt}/matches`. Sans lui, l'API met
   `MatchCompletedDate` à `00:00:00` (fidelity 1) → `StartedAtUTC = minuit − durée` = heures
   FAUSSES (toutes ancrées à minuit). Avec, fidelity 2, horodaté à la ms. Corrigé dans
   `internal/games/halo_5/client.go:GetPlayerMatches`. Ce N'EST PAS une limite API (erreur
   de paramètre, pas de dégradation 343).
2. **Commendations per-match** = carnage (`GetMatchCarnage`) →
   `PlayerStats[].ProgressiveCommendationDeltas[]` (+ `MetaCommendationDeltas`) =
   `{Id (UUID), PreviousProgress, Progress}`. Le compte du match = `Progress − PreviousProgress`
   (>0). Confirmé non-vide live. (Les `Impulses` carnage sont VIDES — ne pas s'en servir.)
   Décision produit : affichage **NATIF** (pas le moteur de citations dérivé d'Infinite),
   mais **ingestion par match en BDD** (table `match_commendations`, INSERT OR IGNORE,
   ART-safe parité `medals_earned`).
3. **Corrélation media** : nom de capture en **heure locale Paris** → UTC (**CET = UTC+1**
   en hiver) → match où `start_time_utc ≤ ts ≤ end_time_utc`. **VALIDÉ** sur 2019-12-12
   (3 captures 22h27/37/49 ↔ 3 matchs 21:25/21:36/21:47 UTC, mapping 1:1). Ce n'est PAS
   un offset d'horloge à recalibrer, juste la conversion TZ. Captures : 84 fichiers
   `Halo_5_Guardians-YYYY-MM-DD_HHhMM.mp4` dans `C:/Users/Guillaume/Videos/Captures/JGtm`
   (2018-08 → 2019-12).
4. **Modèle tokens** (cf. [[feedback_token_model_rt_never_recapture]]) : **6 RT valides**
   (5 + JGtm : DankerGlue1131/GeleJugefi/QuiteSiren8545/TrimButton1352/UppedJoker1176). Les
   3 xuids en `AADSTS70000 different client id` (2533274833178266, 2533274858283686,
   2535469190789936) = RT d'une **vieille app Azure**, entrées store périmées → **IGNORER**,
   **JAMAIS re-capturer**. Le bruit `AADSTS70000` dans les logs de sync = ces 3 RT retentés.
5. **Backfill h5 = ALL-OR-NOTHING** : le runner (`livesync/runner.go` RunDelta) capture TOUT
   puis `PersistAll` à la FIN. Pour le full historique (JGtm = 2016→2023, beaucoup de matchs),
   ça veut dire des heures de capture sans rien persisté + risque de tout perdre. Un run full
   a tourné 2h sans rien persister. **À FAIRE : un backfill incrémental/paginé résumable**
   (persister par page) avant de relancer le full.
6. **Harness `persist`-intégration cassé (PRÉ-EXISTANT du merge)** : les tests
   `//go:build integration` du package `internal/persist` échouent dans le SETUP
   (`weapon_kills does not exist` — le harness `openSharedTestDB`/`e2eSetup` ne câble pas le
   provider de migrations title-owned, donc `weapon_kills` (créé title-owned) absent). Même
   classe que le bug que le merge a corrigé pour les tests `internal/migration` (relocation).
   Indépendant de tout code commendation. **À FAIRE : câbler `SetTitleStepsProvider` dans ce
   harness** (comme les tests migration relocalisés).
7. **MT-19 bloque l'axe E** : `defaultProgressionTitleSlug()` est hardcodé `halo_infinite`
   (`internal/api/post_sync_deltas.go:119`) et `PrestigeBundle` est un singleton global
   (server.go) → le PostSyncRunner / notifications ne sont pas title-aware. Rendre l'axe E
   (notif « titre prêt » first-sync) nécessite la refacto MT-19. Phase 1 Infinite-only = safe.

## 4. État des axes prod-gate (cf. `.ai/PLAN_H5_PROD_GATE.md`)

| Axe | État |
|-----|------|
| **A** match.history | ✅ LIVRÉ (`35a311130`) — lit la DB locale, capability supported, parité, tests |
| **B** commendations natives | ✅ CORE LIVRÉ (`01eef3ae1`) — ingestion+canonical+match detail. **Reste** : front display, endpoint lifetime `/players/{xuid}/commendations`, définitions natives (nom/icône via haloapi.com → table `commendation_definitions`) |
| **C** career rank SR | ✅ LIVRÉ (`5c6c95d3d`) |
| **D** media | 🔜 le pipeline est title-aware ; reste : (1) **regex filename** `Halo_5_Guardians-YYYY-MM-DD_HHhMM` à ajouter dans `internal/ops/media_filename.go` (les 2 regex existants attendent OBS/Xbox avec SECONDES, 6 groupes ; format h5 = 5 groupes, pas de sec → ajouter regex + `sec:=0 si len(m)<7`), (2) **activer la capability media** pour h5 (coarse `CapMedia` dans `config/titles/halo_5/title.toml`), (3) **valider** la corrélation sur les 84 captures |
| **E** postsync/first-sync notif | ⛔ BLOQUÉ par MT-19 (cf. finding 7) |
| **F** world leaderboard | ✅ rien à faire — h5 ne déclare pas `world.leaderboard` → déjà gaté ; rationale dans `.ai/HANDOFF_HALO5_EXPERIMENTAL.md §2` |

## 5. En cours (background à la fin de session)

- **Run borné 300** (`/tmp/h5-sync.exe JGtm 300`, binaire complet A+B) en capture — produit
  ~300 matchs récents (2019→2023) avec **commendations + heures précises** pour VALIDER le
  pipeline (corrélation media + commendations peuplées). DB =
  `LevelUp-go-migration/data/titles/halo_5/warehouse/shared_matches_v2.duckdb`.
- À la fin : vérifier `SELECT count(*) FROM match_commendations` > 0, et corréler quelques
  captures 2019 (timestamps précis désormais).

## 6. Opérationnel (commandes)

```bash
export PATH="/c/msys64/ucrt64/bin:$PATH"; export CGO_ENABLED=1
MAIN=/c/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration

# Rebuild le binaire de sync (après toute modif capture/persist/client h5)
go -C apps/go-api build -o /tmp/h5-sync.exe ./cmd/h5-sync

# Backfill borné (full historique nécessite delta-stop contourné = DB vide) :
rm -f "$MAIN/data/titles/halo_5/warehouse/shared_matches_v2.duckdb"* ; rm -rf "$MAIN/data/titles/halo_5/players"
LEVELUP_REPO_ROOT="$MAIN" LEVELUP_LOG_LEVEL=warn /tmp/h5-sync.exe JGtm <N>

# Requêter une DuckDB (NE PAS ouvrir la DB h5 pendant un backfill — verrou)
go -C apps/go-api run ./cmd/diag_exec "$MAIN/data/titles/halo_5/warehouse/shared_matches_v2.duckdb" "SELECT ..."

# Sonde live h5 brute (shapes JSON)
LEVELUP_REPO_ROOT="$MAIN" go -C apps/go-api run ./cmd/probe-h5 JGtm
```

- **delta-stop** : sur une DB non-vide, RunDelta s'arrête au 1er match connu → pour
  re-fetcher l'historique profond, purger la DB d'abord (d'où le besoin d'un vrai backfill
  paginé incrémental).
- Le **commit hook** (lefthook) lance gofmt/vet/check-merge-conflict ; les « build constraints
  exclude » des `cmd/*` sont informatifs (tags), pas des erreurs.

## 7. Règles & mémoires à respecter

- Demander avant tout `git commit` ([[feedback_ask_before_commit]]) ; jamais de re-capture
  de tokens ([[feedback_token_model_rt_never_recapture]]) ; pas de Python ([[feedback_no_python]]) ;
  pas d'emojis dans les fichiers ([[feedback_no_emojis]]) ; title-agnostic au max (capability,
  pas slug) ([[feedback_prefer_title_agnostic]]).
- Ne JAMAIS réintroduire une surface ART (UPDATE/DELETE sur colonne/table indexée ; UNIQUE sur
  colonne mutée) — gardes `TestNoARTSurfaceIndexInMigrations` (étendu au dossier title-owned) +
  `TestNoMediaFilesFilePathUnique`.

## 8. Prochaines étapes recommandées (ordre)

1. Run borné 300 fini → **valider** commendations + corrélation media.
2. **Axe D** (regex filename + capability media + validation 84 captures).
3. **Backfill incrémental résumable** → full historique 2016-2019 (couvre toutes les captures).
4. **B finition** (front commendations + lifetime + définitions natives).
5. Fix harness `persist`-intégration (weapon_kills).
6. Axe E quand MT-19 est traité.
7. Décision séparée : faire atterrir `integration` sur `main` (auto-déploie) — pas sans validation E2E complète + accord explicite.
