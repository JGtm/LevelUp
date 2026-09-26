# Volet 1 — Cause racine de la corruption LUSR `h5_arena` (26-28 juin 2026)

Enquête en lecture seule. Aucun fichier Go/TS modifié, aucune commande `go`, aucun accès
aux bases DuckDB. Preuves : code versionné (`git show` / `git log -p`), `db_profiles.json`,
et les journaux applicatifs persistants (`logs/sync.log`, `logs/scheduler.log`,
`logs/general.log` du dépôt principal, qui couvrent le 2026-06-02 → 2026-08-27).

---

## 1. Résumé (5 lignes)

- **Cause NOMMÉE et PROUVÉE** : le pipeline de sync V2 (ADR 0027) tirait le titre de
  **deux sources indépendantes** — les handles DB depuis `deps.TitleSlug` (titre du cycle,
  `halo_infinite`) et le `SyncEngine` depuis `p.TitleSlug` (titre du profil, `halo_5`).
  Les 4 joueurs étant déclarés dans `db_profiles.json` **sous les deux titres** depuis le
  2026-06-25, chaque cycle les synchronisait deux fois ; la passe « profil halo_5 » a
  écrit la chaîne `h5_arena` dans les player DB **halo_infinite**.
- **Niveau de preuve** : PROUVÉ. Signature complète dans `logs/sync.log` du 2026-06-26
  11:20-11:28 (comptes 471 / 31 / 902 / 1064 = les comptes de corruption, WARN
  `"group":"h5_arena"` sur des matchs Infinite 16v16), plus le commit correctif ultérieur
  `b30eb9fe5` (2026-07-03) qui énonce le défaut mot pour mot.
- **Le chemin de 2026-06 est fermé** (partition `livesync.HandlesTitle`, `b30eb9fe5`).
- **La CLASSE de défaut ne l'est PAS** : la double source de titre subsiste
  (`sync_v2_wiring.go` L93/109/122/150 vs L283) et un second site vivant existe
  (`registry_lusr_gaps.go:145`).
- **Garde-rail recommandé : OUI**, et ce n'est PAS d'abord l'assertion d'intégration
  évoquée par le pilote (détecteur *a posteriori*) mais un **fail-loud structurel au
  câblage** ; l'assertion de données vient en second, comme filet.

---

## 2. Chronologie du 20 juin au 5 juillet 2026

| Date / heure | Élément | Nature |
|---|---|---|
| 2026-06-20 18:40 | `b111b5d33` `feat(h5)` : câblage live-sync h5 (dispatch registry) | commit |
| 2026-06-21 14:32 | `4cd2f3f84` `feat(lusr)` : seam **title-aware** `SetLUSRChainClassifierForTitle` / `GetLUSRChainForTitle` ; création de `halo_5/lusr_chain.go` (`h5_arena`) ; pose du classifier h5 dans `cmd/server/main.go` ; `processOneShadowMatch` lit `ctxkeys.TitleSlug(ctx)` | **commit armant le mécanisme** |
| 2026-06-23 19:52 | `6c8fd260a` : `cmd/h5-lusr-backfill` + `cmd/h5-enrich` (chemins halo_5 uniquement) | commit |
| 2026-06-24 21:51 | `5be99a2c3` C6 : `GetPerformanceChain` / `computeSkillRatingsBatch` / `RunFormulaSim` deviennent title-aware via `ctxkeys.TitleSlug(ctx)` (v1 LUSR + chaîne de perf) | commit (élargit la surface) |
| **2026-06-25 09:37** | **`db_profiles.json` modifié** (mtime) : ajout du bloc `"halo_5"` contenant **exactement** Chocoboflor, JGtm, Madina97294, XxDaemonGamerxX | **déclencheur** |
| 2026-06-25 / 26 | rounds « hardening H5 » (journal `.ai/archive/thought_log_2026-Q2.md`) | contexte |
| 2026-06-26 11:05:14 | boot serveur : `LUSR mode: v2 CANONICAL (v1 skippé)` (`logs/sync.log`) | journal |
| **2026-06-26 11:20:13 → 11:28:32** | **`auto_sync: tick démarré` → `sync.v2: cycle démarré` → `phase post_sync` → CORRUPTION** | **fait** |
| 2026-06-26 11:21:32 | `post-sync: LUSR v2 shadow OK gamertag=Chocoboflor processed=471 canonical=true` | journal |
| 2026-06-26 11:22:02 | idem `XxDaemonGamerxX processed=31` | journal |
| 2026-06-26 11:23:35 | idem `JGtm processed=902` | journal |
| 2026-06-26 11:28:30 | idem `Madina97294 processed=1064` | journal |
| 2026-06-26 16:05 | `33f40f269` patch ART sur `cmd/h5-lusr-backfill` (DELETE → `RecomputeLUSRCanonicalForPlayer`) — **sans rapport causal**, même journée | commit |
| 2026-06-27 19:11 → 20:30 | cycles suivants : `processed=5`, `processed=1` (les nouveaux matchs du jour reçoivent aussi une ligne `h5_arena`) | journal |
| 2026-06-28 / 29 | 186 puis 264 lignes `LUSR v2 shadow terminé` — la double passe par joueur persiste | journal |
| 2026-07-01 | `e2f2cda20` : pipeline V2 par défaut (il était déjà utilisé en local) | commit |
| **2026-07-03 14:36** | **`b30eb9fe5` `audit(D1c-1)` : partition `livesync.HandlesTitle` — les joueurs live-only ne passent plus par l'orchestrator V2. Ferme le chemin.** | **correctif (fortuit vis-à-vis du LUSR)** |
| 2026-07-21 / 22 | `71b7c97b3` / `ee53afd11` : fuite de titre H5→Infinite côté **lecture** (header `X-LevelUp-Title`) — même famille, autre surface | commits |
| 2026-08-28 | découverte + réparation à 99,9 % (lot 4 « note de perf », `.ai/PLAN_PERF_NOTE_OBJECTIFS.md:515`) | fait |

### Runs identifiés dans la fenêtre

Aucune commande `cmd/h5-*` n'est en cause : **toutes** les CLI h5 résolvent leurs chemins
par `PathResolver(halo5.TitleSlug)` (vérifié une par une : `h5-lusr-backfill:89-90`,
`h5-enrich:88-89`, `h5-csr-match-backfill:99,104`, `h5-appearance-backfill:87`,
`h5-teamscore-backfill:82`, `h5-events-backfill:73`, `h5-roster-*` via `--title`). Aucune
ne peut ouvrir une base `halo_infinite`.

Le seul écrivain est le **serveur local**, via son cron `auto_sync` → pipeline V2.

---

## 3. Reconstitution de l'état du code au 26 juin 2026

Binaire en vol identifié par les numéros de ligne des journaux
(`engine_postsync_scoring.go:141`, `skill_v2_shadow.go:174`, `skill_v2_shadow.go:303`) :
ils correspondent exactement à l'arbre `33f40f269` (2026-06-26). La chaîne d'écriture était
donc la suivante.

**(a) Deux profils par joueur.** `apps/go-api/internal/config/config_players.go`
(`loadPlayersV3`, L164-186 à `33f40f269`, inchangé aujourd'hui) parcourt la map
`title_slug → gamertag → entry` et produit **une `PlayerSummary` par (titre, gamertag)** :

```go
for titleSlug, titleProfiles := range file.Profiles {
    if filter != "" && titleSlug != filter { continue }
    for gamertag, p := range titleProfiles {
        players = append(players, domain.PlayerSummary{ ..., TitleSlug: titleSlug, ... })
```

`AutoSyncScheduler` appelle `s.cfg.LoadPlayers()` **sans filtre** (auto_sync.go:601 à
`33f40f269`) → 8 profils pour 4 gamertags après le 2026-06-25.

**(b) Le pipeline V2 recevait TOUS les profils.**
`runOnceV2` (auto_sync.go:1036-1047 à `33f40f269`) mappait chaque `PlayerSummary` en
`syncv2.PlayerProfile{ TitleSlug: resolveTitleSlug(p) }` **sans aucune partition par
titre**. C'est précisément ce que corrigera `b30eb9fe5`, dont le message de commit est un
aveu explicite :

> « L'orchestrator V2 (ADR 0027) est mono-titre (Infinite) et ne route pas les titres
> live-only : **sous V2 (defaut prod) les joueurs Halo 5 etaient traites comme Infinite.** »

**(c) La double source de titre — le cœur du défaut.**
`apps/go-api/cmd/server/sync_v2_wiring.go` (numéros de ligne d'aujourd'hui ; identiques en
substance au 26 juin) :

```go
 93:  sharedPath := deps.PathResolver.SharedDBPath(deps.TitleSlug)               // titre du CYCLE
109:  path := deps.PathResolver.PlayerDBPath(deps.TitleSlug, gamertag)           // titre du CYCLE
122:  path := deps.PathResolver.PlayerDBPath(deps.TitleSlug, gamertag)           // titre du CYCLE
150:  persister = syncv2.NewCycleBatchPersister(deps.TitleSlug, ...)             // titre du CYCLE
283:  engine := syncpkg.NewSyncEngineForTitle(deps.Cfg.RepoRoot, p.TitleSlug, ...) // titre du PROFIL
```

`deps.TitleSlug` est posé UNE fois au boot (`cmd/server/main.go:1111`, `TitleSlug: titleSlug`)
= `halo_infinite`. `p.TitleSlug` vient de `db_profiles.json`. Pour le profil
`(halo_5, JGtm)` : **handles = bases Infinite, moteur = titre halo_5**.

**(d) Le titre du moteur pilote le classifier, pas les bases.**
`apps/go-api/internal/sync/engine_postsync_scoring.go:162` (L136 au 26/06) :

```go
scoringCtx := ctxkeys.WithTitleSlug(ctx, e.titleSlug)   // "halo_5"
RunLUSRV2ShadowOwnerOnly(scoringCtx, playerDB /* Infinite */, sharedDB /* Infinite */, e.xuid)
```

puis `apps/go-api/internal/sync/skill/skill_v2_shadow.go:255` (L~228 au 26/06) :

```go
group := GetLUSRChainForTitle(ctxkeys.TitleSlug(ctx), m.pairName)
```

et `apps/go-api/internal/sync/skill/skill_chain_provider.go:69-70` :

```go
if f, ok := lusrChainClassifiersByTitle[titleSlug]; ok { return f(pairName) }
```

Le classifier h5 est posé au boot du serveur (`cmd/server/main.go:1478` à `33f40f269`) et
`apps/go-api/internal/games/halo_5/lusr_chain.go:22-26` retourne **inconditionnellement**
`"h5_arena"` :

```go
const LUSRChainArena = "h5_arena"
func ClassifyLUSRChain(_ string) string { return LUSRChainArena }
```

**(e) Écriture.** `state.PlaylistGroup` (= `h5_arena`) est recopié tel quel dans
`persist.LUSRRatingInsert.PlaylistGroup` par
`apps/go-api/internal/sync/skill/skill_v2_canonical.go:112` (`writeCanonicalLUSRRow`), pour
**deux lignes par match** (`rating_type` `LUSR` + `LUSR_V2`), en INSERT pur append-only.

**(f) Pourquoi TOUT l'historique et pas seulement le delta.** Le watermark LUSR v2 est
porté **par chaîne** (`player_skill_state_v2(xuid, playlist_group)`). La chaîne `h5_arena`
n'existait pas dans le shared Infinite → `LoadState` renvoie nil → aucun match n'est
`skippedAlready` → replay complet de l'historique éligible sous la mauvaise chaîne.

### Preuve directe dans `logs/sync.log`

1. **Le groupe fautif est nommé sur des matchs Infinite.** 35 WARN du 2026-06-26 portent
   `"group":"h5_arena"` avec des tailles d'équipe **16v16, 15v12, 12v14, 15v14** — du BTB
   Halo Infinite ; Halo 5 (Arena 4v4 ; Warzone exclu du LUSR par le filtre 2-équipes) ne
   produit pas ces effectifs :

```
2026-06-26T11:22:38 WARN processOneShadowMatch "LUSR v2 shadow: compute échoué"
  match_id=3efe4592-… group=h5_arena team_a_size=16 team_b_size=16
```

2. **Les volumes du run correspondent aux volumes de corruption** (plan lot 4 : JGtm 895 /
   Madina 1064 / Chocoboflor 471 / Daemon 31) :

```
11:21:32 post-sync: LUSR v2 shadow OK gamertag=Chocoboflor      processed=471  canonical=true
11:22:02 post-sync: LUSR v2 shadow OK gamertag=XxDaemonGamerxX  processed=31   canonical=true
11:23:35 post-sync: LUSR v2 shadow OK gamertag=JGtm             processed=902  canonical=true
11:28:30 post-sync: LUSR v2 shadow OK gamertag=Madina97294      processed=1064 canonical=true
```

(JGtm 902 vs 895 : le journal compte les matchs traités, le plan compte les lignes restées
gagnantes dans `match_skill_rank_latest` ; 7 lignes ont été réécrites depuis.)

3. **La double passe par joueur est visible et datée.** Chaque xuid apparaît **deux fois
   par cycle** à partir du 26/06, avec des cardinales voisines mais distinctes (les deux
   moteurs lisent le même shared Infinite mais tiennent deux watermarks) :

```
2026-06-26T11:22:00 xuid=2533274858283686 processed=0    already=1141   <- profil halo_infinite
2026-06-26T11:28:30 xuid=2533274858283686 processed=1064 already=0      <- profil halo_5
2026-06-26T11:21:32 xuid=2535469190789936 processed=471  already=0
2026-06-26T11:25:48 xuid=2535469190789936 processed=0    already=496
```

4. **Avant le 25/06, la double passe n'existe pas.** Dernier cycle antérieur au journal :

```
2026-06-19T13:33:54 … 13:34:29 : 4 runs, un par xuid, processed=0
```

5. **Le déclencheur est daté par le système de fichiers** : `db_profiles.json` mtime
   **2026-06-25 09:37**, et son bloc `"halo_5"` contient exactement les 4 gamertags
   corrompus — les 5 profils `auth_only` (Trimbutton, DankerGlue, QuiteSiren, UppedJoker,
   GeleJugefi) ne sont déclarés que sous `halo_infinite` et **ne sont pas corrompus**,
   ce qui referme la corrélation.

### Chaîne causale complète (PROUVÉE)

```
db_profiles.json v3 : 4 gamertags déclarés sous halo_infinite ET halo_5 (25/06 09:37)
  → LoadPlayers() sans filtre  → 8 PlayerSummary (config_players.go:165-186)
  → runOnceV2 sans partition   → 8 PlayerProfile envoyés à l'orchestrator (avant b30eb9fe5)
  → sync_v2_wiring : handles = deps.TitleSlug (halo_infinite)
                     moteur  = p.TitleSlug   (halo_5)          [L93/109/122 vs L283]
  → engine_postsync_scoring : scoringCtx = WithTitleSlug(ctx, "halo_5")
  → GetLUSRChainForTitle("halo_5", …) → halo5.ClassifyLUSRChain → "h5_arena"
  → watermark h5_arena inexistant dans le shared Infinite → replay complet
  → writeCanonicalLUSRRow → 2 lignes/match dans match_skill_rank des player DB INFINITE
```

---

## 4. Hypothèses : retenue et écartées

### RETENUE (prouvée)

**H0 — Double source de titre dans le pipeline V2 + profils dupliqués dans `db_profiles.json`.**
Preuves : section 3 (a)-(f) + les 5 preuves journalisées + le message de `b30eb9fe5`.
Niveau : **PROUVÉ** (code + journaux datés + corrélation exacte des volumes et de la
population affectée).

### ÉCARTÉES

| Hypothèse | Réfutation |
|---|---|
| **H1 — « un binaire h5 exécuté sur les données Infinite »** (hypothèse de travail du lot 4) | ÉCARTÉE. Les 13 CLI `cmd/h5-*` résolvent toutes leurs bases par `PathResolver(halo5.TitleSlug)` — vérifié ligne par ligne. Aucune ne peut ouvrir `data/titles/halo_infinite/`. L'écrivain est le **serveur** : `source=engine_postsync_scoring.go:141` dans les journaux. L'intuition « signature h5 » était bonne, le vecteur non. |
| **H2 — classifier global fuité entre titres dans le même process** | ÉCARTÉE. `lusrChainClassifiersByTitle` est une map **keyée par slug** (`skill_chain_provider.go:42-44`) ; la seule pose h5 est `SetLUSRChainClassifierForTitle(halo5.TitleSlug, …)`. Il n'y a jamais eu de pose sous `halo_infinite` ni sous `""`. Le problème n'est pas la clé, c'est la valeur de `titleSlug` présentée à la lecture. |
| **H3 — défaut de classifier (nil → repli h5)** | ÉCARTÉE. Le repli est fail-loud et Infinite : `GetLUSRChainForTitle` panique si le classifier par défaut est nil, et `ctxkeys.TitleSlug` retourne `"halo_infinite"` quand la clé est absente du ctx. Aucun chemin ne dégrade vers h5. |
| **H4 — CLI `--player` sans `--title`** | ÉCARTÉE. `cmd/lusr_v2_canonical_backfill` (le seul outil ciblant nommément les 4 joueurs, via `defaultPlayers`) code ses chemins en dur sur `halo_infinite`, n'enregistre **que** le classifier par défaut et tourne sur `context.Background()` → chaîne Infinite par construction. `cmd/levelup backfill --lusr` construit moteur ET chemins depuis le même slug. |
| **H5 — recompute itérant sur tous les titres avec le mauvais PathResolver** | ÉCARTÉE pour juin : `NewSyncEngineForTitle` (engine_options.go:51-71) dérive `playerDBPath` / `sharedDBPath` / `metadataDBPath` du **même** `titleSlug` que `e.titleSlug`. Le moteur seul est cohérent ; c'est le **wiring V2 qui contourne ses chemins** en lui passant des handles ouverts ailleurs. |
| **H6 — LUSR v1 (`batchComputeLUSR`) devenu title-aware par C6 `5be99a2c3`** | ÉCARTÉE factuellement, mais **reste un vecteur armé**. Le boot du 2026-06-26 11:05 journalise `LUSR mode: v2 CANONICAL (…, v1 skippé)` et `engine_postsync_scoring` saute v1 sous `IsLUSRV2Canonical()`. v1 n'a donc pas tourné. Si le mode canonical avait été OFF, `batchComputeLUSR(ctx, …)` — qui lit `ctxkeys.TitleSlug(ctx)` **sans stamp du titre du moteur** — aurait produit la même corruption. |
| **H7 — copie / transfert d'une base h5 par-dessus une base Infinite** | ÉCARTÉE. Une copie n'aurait pas produit des volumes exactement égaux au nombre de matchs **Infinite** éligibles par joueur, ni des WARN 16v16 pendant un cycle de sync horodaté. |

---

## 5. Verdict sur les gardes actuelles + spécification du garde-rail manquant

### 5.1 Verdict : NON, le trou n'est pas fermé

**Ce qui est fermé.** `b30eb9fe5` partitionne les joueurs avant l'orchestrator
(`auto_sync_run.go:123-128`) :

```go
if livesync.HandlesTitle(resolveTitleSlug(p)) { livePlayers = append(...) } else { orchPlayers = append(...) }
```

`livesync.HandlesTitle("halo_5")` est vrai → le profil halo_5 ne peut plus entrer dans le
cycle V2. Test associé : `TestRunOnceTrigger_V2_RoutesH5ToLiveRunner_NotOrchestrator`
(`internal/scheduler/auto_sync_h5_test.go`). Le scénario exact de juin est donc mort.

**Ce qui ne l'est pas — trois trous vivants, vérifiés sur pièces.**

**T1 — la double source de titre existe toujours.** `cmd/server/sync_v2_wiring.go` :
handles depuis `deps.TitleSlug` (L93/109/122/150), moteur depuis `p.TitleSlug` (L283),
**sans aucune assertion d'égalité**. Le prédicat de partition est
`livesync.HandlesTitle` = « ce titre a-t-il un runner live dédié ? », **pas** « ce titre
est-il celui du cycle ? ». Un **troisième titre piloté par le SyncEngine** (donc
`HandlesTitle == false`) tomberait dans `orchPlayers` et reproduirait la corruption à
l'identique. La protection actuelle est **incidente**, pas structurelle.

**T2 — action admin « replay LUSR » : titre du moteur ≠ titre du ctx.**
`internal/api/wire/registry_lusr_gaps.go` stampe le ctx pour le **scan** (L35, L165) mais
**pas pour le replay** :

```go
141: engine := sync_pkg.NewSyncEngineForTitle(r.cfg.RepoRoot, titleSlug, p.Gamertag, p.XUID, nil, r.provider)
145: updated, err := engine.RecomputeLUSRCanonical(ctx)   // ctx NON stampé
```

`SyncEngine.RecomputeLUSRCanonical` (`engine_backfills.go:187-208`) transmet `ctx` tel quel
à `RecomputeLUSRCanonicalForPlayer`. Or `titleSlug` vient du **corps de requête**
(`handlers/admin_lusr_gaps.go:62`, `titleOrDefaultSlug(in.Title)`) tandis que le ctx vient
du **middleware** (`middleware/title.go` : header `X-LevelUp-Title` > session > défaut).
Les deux peuvent diverger : un admin dont l'onglet est sur Halo 5 qui déclenche le replay
d'un joueur `halo_infinite` **réécrit `h5_arena` dans la base Infinite, aujourd'hui**.
L'auto-heal cron (`cmd/server/main.go:1142`) produit la corruption **miroir** : ctx de fond
sans titre → défaut `halo_infinite` → chaînes `arena_*` écrites dans une base `halo_5`
(retenu seulement par le kill-switch `LEVELUP_LUSR_AUTOHEAL_ENABLED`, défaut OFF).

**T3 — `openspartan_post_import_service.go:152`** appelle
`RecomputeLUSRCanonicalForPlayer(ctx, …)` avec le ctx de requête brut ; même schéma.

**Les gardes du lot 4 ne couvrent pas ce risque.** `cmd/recompute_perfnote` est sûr, mais
pour une raison accidentelle : il n'importe pas `internal/games/halo_5` et ne pose que le
classifier par défaut (`wireClassifiers`, main.go:100-112) — il ne peut donc pas écrire
`h5_arena`. Ses 3 gardes anti-no-op portent sur `SlugHasLUSR` / `IsLUSRV2Enabled` /
`IsLUSRV2Canonical`, pas sur la cohérence titre ↔ base.

### 5.2 Le garde-rail proposé par le pilote : bon, mais en second

L'assertion « aucune chaîne `h5_*` dans une base non-h5 » est **utile mais insuffisante** :
c'est un détecteur *a posteriori*, qui ne se déclenche qu'après que 2 461 lignes ont été
écrites, et qui doit énumérer les chaînes de chaque titre (liste à maintenir ; 3e titre =
oubli garanti). Le garde-rail qui **ferme** le trou est le fail-loud au câblage. Je
recommande les trois, dans cet ordre.

### 5.3 Spécification exécutable

#### G1 — PRIORITAIRE : fail-loud sur la divergence de titre au câblage V2

- **Fichier de production** : `apps/go-api/cmd/server/sync_v2_wiring.go`, dans
  `buildSyncEngineFactoryParityComplete`, **avant** la ligne 283.
- **Assertion** : si `p.TitleSlug != "" && p.TitleSlug != deps.TitleSlug`, la fabrique
  retourne une erreur et loggue en `ErrorContext` :
  `fmt.Errorf("sync.v2: profil %q du titre %q soumis au cycle du titre %q — les handles DB sont ceux du cycle, refus (corruption LUSR h5_arena 2026-06-26)", p.Gamertag, p.TitleSlug, deps.TitleSlug)`.
  La signature `syncv2.SyncEngineFactory` retourne déjà `(*SyncEngine, error)` : aucun
  changement de contrat. Un profil refusé remonte en `failed` dans le `CycleResult` —
  bruyant, visible, non destructeur.
- **Fichier de test** : `apps/go-api/cmd/server/sync_v2_wiring_title_test.go` (nouveau).
- **Nom du test** : `TestBuildSyncEngineFactory_RefusesForeignTitleProfile`.
- **Cas** : (1) `deps.TitleSlug="halo_infinite"`, `p.TitleSlug="halo_infinite"` → engine
  non nil, err nil ; (2) `p.TitleSlug=""` → engine non nil (défaut historique préservé) ;
  (3) `p.TitleSlug="halo_5"` → engine nil, err non nil contenant `"halo_5"` et
  `"halo_infinite"`.
- **Coût** : ~15 lignes de production, ~40 lignes de test, 0 requête DB, 0 accès disque.

#### G2 — Ratchet : une seule source de titre dans le wiring V2

- **Fichier** : `apps/go-api/cmd/server/sync_v2_wiring_title_test.go` (second test du même
  fichier ; ou le paquet archlint si le pilote préfère y regrouper les ratchets grep).
- **Nom du test** : `TestSyncV2WiringHasSingleTitleSource`.
- **Assertion** : lire `cmd/server/sync_v2_wiring.go` ; pour **chaque** occurrence de
  `p.TitleSlug`, exiger que la ligne contienne aussi `deps.TitleSlug` (donc : une
  comparaison de garde, jamais une source de chemin / de persister / de construction de
  moteur). Message d'échec citant l'incident du 2026-06-26 et le présent rapport.
- **Coût** : ~25 lignes, exécution en microsecondes, aucun tag de build.

#### G3 — Détecteur de données : invariant « chaîne étrangère au titre »

Le foyer naturel existe déjà : `apps/go-api/internal/sync/invariants/` (« contrats de
données déclarés du pipeline de sync », consommé par le gate d'intégration
`internal/sync/invariants_gate_integration_test.go` et destiné à une sentinelle runtime).
Le paquet est volontairement sans dépendance interne : l'ensemble autorisé doit donc être
**injecté par l'appelant**, pas énuméré dans `invariants`.

- **Fichier** : `apps/go-api/internal/sync/invariants/invariants.go` (ajout) — nouvelle
  fonction exportée à côté de `CheckPlayer` / `CheckShared` :

  ```
  func CheckPlayerLUSRChains(ctx context.Context, playerDB *sql.DB, allowedChains []string) (Report, error)
  ```

  (fonction séparée plutôt qu'un `checkFn` de plus, pour ne pas casser la signature
  `checkFn` partagée par les 13 invariants existants).
- **Clé de violation** : `lusr_chain_foreign_title` — **`SeverityFail`**.
- **Description** : « lignes match_skill_rank portant une chaîne LUSR qui n'appartient pas
  au titre de cette base (corruption cross-titre, incident 2026-06-26) ».
- **Requête exacte** (sur la **table brute**, pas la vue `_latest` — voir section 6 :
  `Q8LUSRHistoryPlayer` lit la table brute, donc une corruption y survit à tout replay
  append-only) :

  ```sql
  SELECT DISTINCT playlist_group
  FROM match_skill_rank
  WHERE rating_type IN ('LUSR','LUSR_V2')
    AND playlist_group IS NOT NULL
    AND playlist_group <> ''
    AND playlist_group NOT IN (<allowedChains>)
  ```

  `Count` = `SELECT COUNT(*)` sur le même prédicat ; `Sample` = jusqu'à 5 `playlist_group`
  distincts (`capSample`).
- **Source de l'ensemble autorisé** : le gate d'intégration (et, à terme, la sentinelle
  post-sync) passe la liste du titre de la base. `skillchain` n'expose pas aujourd'hui
  d'énumération de ses chaînes ; la variante sans nouvelle API est de passer la liste en
  dur **côté test uniquement**, avec un commentaire pointant
  `internal/games/halo_infinite/skillchain/classify.go` comme source de vérité.
  **À VÉRIFIER SUR PIÈCES avant écriture** : la liste exacte des chaînes Infinite après le
  lot 1 « scission ranked par famille » (`d3081524c`) — ne pas la recopier depuis ce
  rapport.
- **Fichier de test** : `apps/go-api/internal/sync/invariants/invariants_violation_test.go`
  (existant) — ajouter `TestCheckPlayerLUSRChains_FlagsForeignChain` : insérer dans une
  player DB fixture une ligne `('m1','LUSR',…,'arena_slayer')` et une ligne
  `('m2','LUSR',…,'h5_arena')` ; `allowed={"arena_slayer"}` → 1 violation
  `lusr_chain_foreign_title`, `Count=1`, `Severity=fail` ;
  `allowed={"arena_slayer","h5_arena"}` → 0 violation.
- **Coût** : ~45 lignes de production, ~40 lignes de test unitaire (fixture DuckDB déjà
  outillée dans ce fichier), 1 scan par player DB au gate d'intégration
  (`match_skill_rank` ≈ 2 lignes par match — négligeable).

#### G4 — Correctif de cohérence sur les sites vivants T2 / T3 (hors garde-rail)

Non traité ici (enquête en lecture seule), à ouvrir en lot dédié. La forme minimale :
stamper `ctxkeys.WithTitleSlug(ctx, titleSlug)` avant `engine.RecomputeLUSRCanonical` dans
`registry_lusr_gaps.go:145` (symétrie avec le scan déjà stampé L165) et dans
`openspartan_post_import_service.go:152`. **La forme correcte** : supprimer la double
source en faisant stamper le ctx par `SyncEngine.RecomputeLUSRCanonical` lui-même
(`engine_backfills.go:187`), qui connaît `e.titleSlug` — un seul point, tous les appelants
couverts. C'est le miroir exact de ce que fait déjà `engine_postsync_scoring.go:162`.

---

## 6. Statut des lignes résiduelles Madina — et une découverte plus large

### 6.1 Les 2 lignes de `match_skill_rank_latest` : NUISIBLES, lues par des lecteurs vivants

Les 2 matchs non rejouables (équipes non binaires → `skippedNonTwoTeam` dans
`processOneShadowMatch`) n'ont reçu aucune ligne corrective : leur ligne `h5_arena` reste
la gagnante de la vue `_latest`. Elle est lue par :

- `Q5PlayerSkillRankHistoryTpl` (`platform/duckdb/queries_career.go:115-128`) — sélectionne
  `playlist_group` depuis `match_skill_rank_latest` et « alimente le module *Classement*
  segmenté PAR CHAÎNE » ;
- `queries_match_detail.go:11`, `queries_home_citations.go:110/295`,
  `queries_career_encounters.go:484`, `leaderboard_repo.go:183`
  (`playlist_name = COALESCE(mr.playlist_name, msr.playlist_group, '')`).

Côté web, `apps/web/src/features/career/lusr-chains.ts` :
`resolveLusrGroupsForDisplay(titleSlug, dataGroups)` **ajoute** les groupes présents dans la
donnée mais absents des connus du titre, et `career.toml:387-389` fournit le libellé
(`career.lusr.chain.h5_arena` = « Arène » / « Arena »). La page Carrière de Madina97294
**sous Halo Infinite** affiche donc une 5e chaîne « Arène » à côté d'Assassin / Objectif /
BTB / Chaos.

### 6.2 Découverte plus grave : le résidu n'est pas de 2 lignes pour tous les lecteurs

`match_skill_rank` est append-only : la réparation du lot 4 a **ajouté** des lignes
correctes, elle n'a **rien supprimé** (le plan le dit : « aucune perte de ligne,
arithmétique append-only exacte »). Or **un lecteur vivant lit la table BRUTE**, pas la
vue `_latest` :

`apps/go-api/internal/platform/duckdb/queries_career.go:204-214` —

```go
const Q8LUSRHistoryPlayer = `
SELECT msr.match_id, msr.rating_type, msr.rating_value, msr.tier_label, msr.playlist_group, …
FROM match_skill_rank msr
WHERE msr.rating_type <> 'LUSR_V2'`
```

(commentaire assumé en amont : « ce graphe d'évolution qui veut TOUS les checkpoints »).
Consommé par `career_repo_lusr.go:71` (`GetLUSRHistory`), regroupé par
`(rating_type, playlist_group)` (`career_repo_lusr.go:183-184`), puis rendu par :

- `apps/web/src/features/career/CareerChartsSection.lusrEvolution.tsx:31-50` — **une série
  par `(rating_type, playlist_group)`** donc une série « Arène (LUSR) » tracée à partir des
  lignes `rating_type='LUSR'` restées en table brute (environ la moitié des 2 461, l'autre
  moitié étant les lignes `LUSR_V2` filtrées par la requête) ;
- `apps/web/src/features/career/CareerRankingBlock.tsx:122` — `lusrByGroup` est dérivé des
  **mêmes checkpoints bruts**, donc la ligne « Arène » du module Classement provient de ce
  résidu pour les 4 joueurs, pas seulement pour Madina.

**Statut de cette découverte : PLAUSIBLE À TRÈS HAUTE CONFIANCE, NON MESURÉE** — la
vérification exigerait une lecture DuckDB, interdite par le périmètre. Elle est entièrement
déduite du code ; le seul contre-argument possible (une purge des lignes brutes au lot 4)
est réfuté par le plan lui-même, qui revendique zéro suppression.

### 6.3 Recommandation chiffrée

1. **Vérifier d'abord, en lecture seule** (serveur arrêté, CLI `duckdb`, 4 bases) :
   `SELECT rating_type, COUNT(*) FROM match_skill_rank WHERE playlist_group='h5_arena' GROUP BY 1;`
   puis la même chose sur `match_skill_rank_latest`. **Coût : 5 minutes.** C'est ce qui
   départage « 2 lignes cosmétiques » et « ~2 461 lignes encore servies au graphe
   d'évolution ».
2. **Si le résidu brut est confirmé — purge par reconstruction de table, PAS par DELETE.**
   `DELETE FROM match_skill_rank WHERE playlist_group='h5_arena'` est exactement le vecteur
   ART #23046 sur une table indexée (règle CLAUDE.md n°1, ADR 0026). La forme sûre est le
   **swap CTAS transactionnel** déjà éprouvé dans
   `apps/go-api/internal/migration/append_only_rebuild.go` (`rebuildAppendOnlyTx` : BeginTx,
   garde de cardinalité avant le DROP, `recoverOrphanAppendOnly`, recréation de la vue
   `_latest`). Outil jetable dédié (`cmd/purge_foreign_lusr_chain`, dry-run par défaut,
   `--commit`, une base à la fois), modelé sur `cmd/recompute_perfnote`.
   **Coût : ~150 lignes jetables + 1 test, ~1 h de développement, ~10 min d'exécution sur
   4 bases, backup préalable obligatoire, serveur arrêté.** Effet de bord voulu : les 2
   matchs Madina non rejouables se retrouvent sans ligne LUSR — ce qui est la vérité (le
   moteur les juge non notables), pas une perte.
3. **Ne rien faire n'est pas neutre** : l'artefact est visible en permanence sur la page
   Carrière des 4 joueurs (série et ligne « Arène » fantômes) et fausse le module
   Classement segmenté par chaîne.
4. **NULLer `playlist_group` est la plus mauvaise option** :
   `queries_home_citations.go:273-277` documente un bug de prod du 2026-05-20 causé par des
   `playlist_group` NULL ; la ligne migrerait simplement vers un groupe « vide ».
5. **Prod** : couverte par la décision utilisateur du 2026-08-28 (migration des BDD locales
   vers la prod). Si la purge est faite, elle doit l'être **avant** cette migration, sinon
   la prod héritera du résidu.

---

## 7. Découvertes hors périmètre (consignées, NON traitées)

1. **T2 / T3 — deux sites vivants de divergence titre ↔ ctx**
   (`registry_lusr_gaps.go:145`, `openspartan_post_import_service.go:152`) : voir 5.1 et le
   correctif G4. Le site admin est déclenchable **manuellement aujourd'hui**.
2. **`Q8LUSRHistoryPlayer` lit la table brute `match_skill_rank`** alors que la règle
   CLAUDE.md n°2 / ADR 0026 impose « lecture = vue `_latest` UNIQUEMENT ». L'exception est
   documentée en commentaire (le graphe veut tous les checkpoints), mais elle rend le
   graphe d'évolution LUSR **structurellement non réparable par un replay append-only** :
   toute erreur de chaîne y reste visible à vie. À arbitrer.
3. **`checkSyncPreconditions` (auto_sync_run.go:313 et 445)** court-circuite la précondition
   « player DB présente » pour les titres live-only. Correct aujourd'hui, mais c'est le
   même prédicat `HandlesTitle` porteur de la confusion « titre live-only » vs « titre du
   cycle ».
4. **La corruption miroir n'a pas été recherchée** : les player DB `halo_5` des 4 joueurs
   peuvent contenir des chaînes `arena_slayer` / `btb` / `chaos`. Le pipeline V2 de juin ne
   pouvait pas la produire (les handles étaient Infinite), mais l'auto-heal cron et
   l'action admin le peuvent. Le détecteur G3 appliqué au titre `halo_5` avec
   `allowed={"h5_arena"}` répondrait en une requête.
5. **`h5-lusr-smoke:29` et `h5-read-smoke:31` embarquent un chemin absolu en dur**
   (`c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/…`) dans du code versionné —
   hygiène, sans lien avec l'incident.
6. **Le commit `5be99a2c3` (C6)** a rendu title-aware le LUSR **v1** (`batchComputeLUSR`) et
   la chaîne de performance (`GetPerformanceChain`) **sans jamais stamper le ctx depuis
   `e.titleSlug`** : ces deux chemins lisent le titre de l'appelant. v1 est aujourd'hui
   mort (canonical = v2), mais `loadHistoryForPerf` → `GetPerformanceChain(ctxkeys.TitleSlug(ctx), …)`
   reste actif et alimente `performance_chain` — même exposition que le LUSR, sur une autre
   colonne. Non vérifié sur données (hors périmètre).
