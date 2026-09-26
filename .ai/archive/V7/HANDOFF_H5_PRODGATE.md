# HANDOFF — Halo 5 prod-gate (sessions 2026-06-22 → 06-23)

Reprise de la mise en production de Halo 5. Document de reprise : état, commits,
findings non-évidents, reste à faire, commandes opérationnelles.

> **Les sections 2/4/5/8 ci-dessous datent du 2026-06-22. La § 0 MISE À JOUR ci-après
> les SUPERSEDE (état réel au 2026-06-23).**

---

## 0. MISE À JOUR 2026-06-23 — état réel (autoritatif)

> **ÉTAT FIN DE SESSION 2026-06-23 (capstone — LIRE EN PREMIER).** Les 5 axes prod-gate sont VERTS et la
> branche `integration/h5-x-livefetch` est POUSSÉE (HEAD `503660cce`, tous tests verts). TOUT le périmètre
> « tout traiter » est LIVRÉ : MT-19 notif push (`title_ready`) · damage model par titre (hp **115** +
> OC P80 **1.264** calibré data & câblé display + KDA natif `h5NetFDA` + DR N/A car cryptum sans
> `damage_taken`) · backlog UI · activation 1b / LUSR v2 temps-joué / recalibration combat (déjà DONE
> in-branch, vérifié par audit 5 agents) · TZ first_joined (0 décalé) · campagne ART (in-branch) ·
> ratchets pre-push débloqués (dette branche). **SEUL CHANTIER CODE RESTANT = weapon family canonical /
> weapon v3** (STOP user, attend GO) → plan `.ai/PLAN_WEAPON_ATTRIBUTION_V3.md` + mémoire
> `project_weapon_attribution_v3_status`. Ensuite (NON-code) : sanity-check runtime LUSR/combat (app qui
> tourne) ; puis **land `integration`→`main`** (auto-deploy prod, accord explicite requis ; gros merge —
> branche ~280 commits derrière main). **Micro-dette damage** : radar session-compare (axe Impact) gardé
> sur la const P80 Infinite (threading 3-niveaux + 13 fixtures disproportionné ; surface secondaire h5
> not_exposed) — cf. PLAN_DAMAGE_MODEL §0.
>
> **Commits session 06-23** (au-dessus de `edba37b38`) : `145303f4f` damage(86 ERRONÉ→reverté) ·
> `b78211253` MT-19 notif push · `918b6c5ee` ratchet unblock · `10621333b` backlog empty-state ·
> `48bcecb92` docs bilan · `7ceace421` damage **115** (revert 86) + littéraux front · `a37328fdf` P80
> config+getter · `503660cce` P80 câblage display. **8 commits, branche poussée.**

> **CADRAGE PASSE « TOUT TRAITER » (user 2026-06-23)** : autonomie complète — commit ET push
> sans demander, sur la branche `integration/h5-x-livefetch` (pas d'auto-deploy, safe). Ordre
> lourd→léger. **STOP impératif AVANT le weapon family canonical / weapon v3** (dernier item, à
> faire séparément). Le **land integration→main** (auto-deploy prod) n'est PAS exécuté dans cette
> passe — il vient après weapon family + accord explicite. Vérifié in-branch : campagne ART
> (`ba3cb4608`) déjà portée ; code TZ (`cmd/backfill_first_joined_tz`, `analysis/timeline`) présent.

### BILAN PASSE « TOUT TRAITER » 2026-06-23 (commits `145303f4f`→`503660cce`, branche POUSSÉE)

- ✅ **MT-19 / axe E notif push** (DERNIER item prod-gate ouvert) — LIVRÉ (`b78211253`). Notif
  `title_ready` first-sync title-aware émise dans `Runner.RunDelta` (funnel HTTP+scheduler+watcher),
  idempotence watermark `sync_meta`, posée dans le flux du titre PAR DÉFAUT (shared_social per-titre),
  ZÉRO contact pipeline progression/prestige. Backend+front+tests verts. → **les 5 axes prod-gate VERTS.**
- ✅ **Damage model par titre** : h5 `effective_hp_to_kill` = **115** (valeur DESIGN PV-pour-tuer : bouclier
  70 + armure 45) — l'autorité, PAS la moyenne empirique dégâts/kill (facteur overkill title-spécifique non
  constant). Le compute/SQL/barres OC-DR étaient DÉJÀ title-aware (audit). NB : un essai « scale-match » à 86
  (commit `145303f4f`) était une ERREUR, **reverté à 115**. Littéraux front rendus title-aware (jeton `{{HP}}`, ci-dessous).
- ✅ **Activation 1b / LUSR v2 temps-joué / recalibration combat** : DÉJÀ DONE in-branch (audit d'état
  5 agents, preuves file:line). Activation 1b dépassée (live sync) ; LUSR temps-joué = code+tests+backfill
  05-30 ; recalibration combat = `e1f021cbb` ancêtre HEAD (5 bandes + engagement absolu).
- ✅ **Backlog UI** : empty-state `SquadMatchHistoryTable` (`10621333b`) ; 3 autres évalués no-op
  (TimeseriesKdaTrend délègue déjà à ChartFromOption ; donut edge-rare en grille ; squad/v2 hors route live).
- ✅ **TZ first_joined** re-vérifié **0 décalé** (résorbé) · **ART** déjà in-branch · **chore ratchet**
  (`…`) débloque le 1er push (knip exports 88→90 + LabWaypoint baseline = dette PRÉ-EXISTANTE branche,
  à réconcilier au merge main — PAS introduite par cette passe).

- ✅ **Damage front littéraux 225 → title-aware** (LIVRÉ 2026-06-23) : `offensiveDamageGradient`/
  `defensiveDamageGradient`/`damageAxisBounds` prennent un param `oneLife` (défaut 225 Infinite) ;
  `TimeseriesEfficiency` + `EfficiencyTooltipText` résolvent le PV du titre via `useEffectiveHpToKill()`
  (`lib/damage/effectiveHp.ts`) + jeton `{{HP}}` (`efficiency_tooltip` + `ref_one_life`). Charts escouade
  gardent 225 (Infinite-only). Tests verts (param oneLife + substituteHpToken). Bars déjà correctes (hp=115).

**RESTE (non bloquant prod)** :
- ✅ **P80 OC h5 = 1.264 calibré + CÂBLÉ display** : config + getter `games.OffensiveConversionP80` +
  exposé bootstrap (`TitleSummary`/openapi) + barres OC front (combat-yield-bar/Timeseries/SessionOcdr via
  hook) + radars backend match-view & squad. LUSR garde la const (Infinite-only). Seule exception : radar
  session-compare gardé sur const (threading 3-niveaux + 13 fixtures disproportionné, surface secondaire).
  **KDA h5 = `h5NetFDA` (k+a/3)−d déjà correct** · **DR h5 = N/A** (cryptum sans `damage_taken`, vérifié
  carnage brut 0/13241). → **Damage model par titre ENTIÈREMENT réglé.** Cf. PLAN_DAMAGE_MODEL §0.
- 🟡 **Sanity-check terrain LUSR/combat** (code DONE, PAS un gap) : vérif runtime niveaux (Madina Platine/Diamant…)
  + profils combat distincts — relève de l'exécution (service tournant / `cmd/diag_lusr_*`), pas du code.

**STOP : weapon family canonical / weapon v3 = NON commencé (consigne user). Land main = après weapon family + accord.**

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
`b2fed4019` front totaux · `ba7fbd31d` Axe E front first-sync ·
`d6fd5ec0d` nav h5 Citations→Commendations · `30c85609f` docs vérif end-to-end. **13 commits.**

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

**VÉRIFICATION END-TO-END (06-23) — PASSE sur la vraie DB h5** (re-backfill arrêté à 1001
matchs 2018-05→2023, DB libérée le temps des vérifs) :
- `match_commendations` : 83 756 rows, **100 % avec `progress`** (re-fetch OK), 972 matchs.
- **xuid OK** : JGtm `2533274823110022` (match_commendations) == `db_profiles.json` == `pdb.XUID`
  → le filtre `LoadCommendationTotals` est correct.
- **Totaux JGtm** non vides + cohérents : Spartan Slayer **27 058**, Headshot 7 154, Assistant
  6 606, Magnum 4 321, Smash 3 341… noms+catégories+icônes (CDN testées HTTP 200) résolus.
- **Media** : **84/84 captures** corrèlent à une fenêtre de match JGtm (Paris→UTC DST-aware
  DuckDB `AT TIME ZONE 'Europe/Paris'`).
- Code : full Go test suite `./internal/...` + typecheck + eslint + tests front (home 34,
  handlers, service, duckdb) + drift OpenAPI + contract routes → tous verts.
- **CONCLUSION : prod-gate VERT sur les 5 axes (A/B/C/D/E + F).**

**COMMANDE BACKFILL (pour faire d'AUTRES joueurs)** — le binaire `/tmp/h5-backfill.exe` capture
désormais le `progress` absolu (colonne `match_commendations.progress`). Resumable (commits par
page, skip-known). Le joueur doit être dans `db_profiles.json` (avec `xuid`) AVANT.

```bash
export PATH="/c/msys64/ucrt64/bin:$PATH"; export CGO_ENABLED=1
MAIN=/c/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration
# 1. (Re)build le binaire après toute modif capture/persist/mapper :
go -C apps/go-api build -o /tmp/h5-backfill.exe ./cmd/h5-backfill
# 2. Lancer pour un joueur (ajoute SES matchs au shared partagé ; PAS de purge —
#    purger supprimerait les autres joueurs). 25 = taille de page.
LEVELUP_REPO_ROOT="$MAIN" LEVELUP_LOG_LEVEL=warn /tmp/h5-backfill.exe <Gamertag> 25
# log : stdout (rediriger vers un fichier). Lock RW shared pendant le run → 1 seul à la fois.
```

- **NE PAS purger** la shared DB pour ajouter un joueur (la purge n'était nécessaire QUE pour
  re-fetcher JGtm dont les rows count-only n'avaient pas de `progress` ; `INSERT OR IGNORE` ne
  rétro-remplit pas). Un NOUVEAU joueur (rows absents) capture le `progress` directement.
- **Note** : `match_commendations` contient déjà des rows (avec progress) pour TOUS les xuids
  des rosters des matchs de JGtm (la carnage mappe tous les joueurs) — mais seulement sur les
  matchs où JGtm était présent. Pour des totaux COMPLETS d'un autre joueur, lancer SON backfill
  (ajoute ses propres matchs). Re-seed metadata (defs) inutile : `commendation_definitions` est
  partagé (déjà 121 seedées).
- Seed/refresh des définitions (une fois, partagé) :
  `LEVELUP_HALOAPI_KEY=<clé .env.local> LEVELUP_REPO_ROOT="$MAIN" go -C apps/go-api run ./cmd/h5-metadata-fetch`

**PROCHAIN CHANTIER — Axe E notif PUSH (away-case), = MT-19** : l'auto-poll front couvre le cas
on-page ; reste la notif quand le user est AILLEURS. L'émetteur `reg.NotificationsEmitter(ctx,
slug)` EST title-aware (pas le blocage). Le blocage = le post-sync hook (`buildPostSyncDeltaHook`,
`EvaluateProgressionAfterSync`) est câblé sur le sync_handler HInf + `defaultProgressionTitleSlug()`
hardcodé `halo_infinite` + `PrestigeBundle` singleton ; la livesync h5 ne passe PAS par ce hook.
Chemin minimal : émettre une notif « titre prêt » à la complétion du 1er sync h5 (détecter
total_matches 0→N), via l'émetteur title-aware, SANS toucher la pipeline progression/prestige
(qui reste HInf). Réf : `internal/api/post_sync_deltas.go`, finding 7 § 3.

**Reste (hors prod-gate)** : notif push (ci-dessus) ; **land `integration`→`main`** (prêt
techniquement, **auto-deploy** → accord user explicite requis, cf. [[feedback_sync_local_main_on_merge]]).

**DETTES / CHANTIERS CONNEXES À NE PAS OUBLIER AVANT PROD** (hors périmètre h5, tracés ailleurs
mais épinglés ici car flaggés prod) :
1. ✅ **Dette TZ `first_joined_time`** — RÉSORBÉE côté données. Re-check dry-run lecture seule
   2026-06-23 sur `data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb` = **0 match
   décalé** (CET 0 / CEST 0) → la correction de base (05-29 + résiduels 06-02) tient, pas de
   réapparition. Code TZ présent in-branch (`cmd/backfill_first_joined_tz`, `analysis/timeline`).
   **Reste** : propagation au recalcul LUSR v2 — FOLDÉE dans le chantier « LUSR v2 pondération
   temps-joué » (son re-backfill relit le `last_leave_time` corrigé). Doc : `project_data_quality_first_joined_tz`.
2. 🟠 **Campagne append-only / ART** — éradication des surfaces ART (UPDATE/DELETE indexés),
   branche `fix/metadata-art-battlepass-appendonly`, « déployer à la fin ». Doc :
   `.ai/HANDOFF_APPEND_ONLY_ART_CAMPAIGN.md` + mémoire `project_append_only_eradication_campaign`.
Autres suites (non bloquantes prod), traitées dans CETTE passe (ordre lourd→léger) : damage model par
titre (h5=115, `project_damage_model_per_title_225`) ; activation multi-titre 1b
(`project_multititre_activation_handoff`) ; LUSR v2 pondération temps-joué ; recalibration profil de
combat (`PLAN_COMBAT_PROFILE_RECALIBRATION.md`) ; backlog UI session solo+squad ; **weapon family
canonical / weapon v3 = EN DERNIER** (décision user 2026-06-23).

> **HORS PÉRIMÈTRE DE CE PROJET** : le *kill-feed frame decoder* (gros RE offline) est **en pause et
> retiré de ce chantier** (décision user 2026-06-23) — il n'a aucun lien avec la mise en prod h5/multi-titre.
> Suivi séparé dans ses propres docs/mémoires (`project_kill_feed_frame_decoder`), à ne PAS reprendre ici.

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
