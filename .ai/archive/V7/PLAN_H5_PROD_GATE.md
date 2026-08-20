# PLAN H5 PROD-GATE — Consolidation des 6 audits

> Source : 6 audits read-only (UI canonique-vs-legacy, citations natives, media,
> match-history+career rank, postsync/notifs, world leaderboard).
> Vérifié contre le code de `levelup-multititre` le 2026-06-22. Les écarts
> entre audits et code réel sont signalés `[VÉRIFIÉ]`.

---

## 1. Résumé exécutif

Halo 5 est **plus avancé que ne le laissent croire plusieurs audits**. L'adapter
canonique h5 sert déjà, et c'est confirmé dans `capabilities.toml` : `career.progression`,
`match.detail.core`, `match.events.timeline`, `match.killfeed.per_kill`,
`match.events.spatial`. Le centre de gravité du travail restant est **data/adapter
(Go)**, pas le front : le front consomme déjà le chemin canonique pour la majorité des
pages et bascule en `FeatureGate`/capability. Les vraies lacunes sont quatre câblages
adapter (`LoadMatchSummaries`, `LoadTimeseries`, citations natives, media) plus une
correction de plomberie (PostSyncRunner hardcodé `halo_infinite`). Le world leaderboard
est **non supportable en l'état** (aucun endpoint cross-joueurs H5) — verdict
`not_exposed` conditionné à une sonde de confirmation. Effort réaliste : **3 à 4
sprints**, dominé par le backend.

---

## 2. Inventaire pages : canonique-vs-legacy (synthèse)

Sur ~20 surfaces UI, voici la répartition pour H5. « Gratuit » = sert déjà le chemin
canonique, marche dès que l'adapter h5 remplit la méthode `Load*` correspondante (déjà
fait pour la plupart).

### Gratuites pour H5 (adapter déjà câblé, capability déjà `supported`)
- **Career** (`career_.tsx`, `CareerService`) — `LoadCareerSnapshot` câblé, SR 1..152
  via `career_sr.go`, CSR natif via service record. `[VÉRIFIÉ]`
- **Match Detail** (`matches/$matchId.tsx`, `MatchViewService`) — `LoadMatchDetail`
  (carnage) câblé, `match.detail.core=supported`. `[VÉRIFIÉ]`
- **Match Events / Timeline / Kill-feed** — `LoadMatchEvents` câblé (`halo_5/events.go`),
  natif `/h5/matches/{id}/events` avec arme-par-kill et positions monde. `[VÉRIFIÉ]`
- **Explorer** (`explorer/index.tsx`, `ExplorerService`) — chemin canonique
  (`LoadTargetRecentMatches`, `LoadParticipantStats`, `LoadPlayerIntersection`) ;
  gratuit dès que les impls h5 de ces interfaces existent.
- **Compare** (`compare.tsx`) — `LoadPlayerStats` + intersection canonique.

### À migrer / câbler (lacune adapter réelle)
- **Match History** (`MatchHistoryService`) — `LoadMatchSummaries` est un **stub**
  `ErrCapabilityNotSupported`. `[VÉRIFIÉ adapter_data.go:281]`
- **Timeseries** (`timeseries.tsx`, `TimeseriesService`) — `LoadTimeseries` jamais
  branché (côté HI comme h5). `analytics.timeseries=not_exposed`.
- **Citations / Commendations** (`citations.tsx`) — natif h5 mais aucun pipeline
  title-aware ; pas de `halo_5/citations_custom.go`. `[VÉRIFIÉ MISSING]`
- **Media** (`media.tsx`) — `CapMedia` absent des capabilities h5 ; pipeline
  Infinite production-ready et title-aware, mais le rail est gated off.

### Hors périmètre / dégradé assumé pour H5 (pas de données ou n/a)
- **Battlepass / Challenges** — n/a (REQ packs, pas de season pass/défis weeklies) →
  `not_exposed`, pages masquées par capability.
- **Squad** — agrégation legacy SQL HINF ; H5 ensemble-par-match inconnu → fallback
  léger (session-tagged) ou `not_exposed` Phase 2+.
- **Prestige** — Infinite-only → `not_exposed` pour h5.
- **World Leaderboard** — pas d'endpoint H5 → `not_exposed` (cf. axe dédié).

**Bilan** : ~10 surfaces gratuites/déjà câblées, **4 câblages adapter réels**
(history, timeseries, citations, media), le reste assumé `not_exposed` documenté.

---

## 3. Les 5 axes du prod-gate (+ leaderboard)

### AXE A — `match.history`

**État** : données présentes (`livesync/persist.go` écrit `match_registry` +
`match_participants` dans `shared_matches_v2.duckdb`), mapper
`mapMatchSummaries()`/`mapOneMatchSummary()` existe et est testé, mais
`LoadMatchSummaries` est un **stub**. `match.history=not_exposed`. `[VÉRIFIÉ]`

| # | Work-item | Layer | Effort | Dépend de |
|---|-----------|-------|--------|-----------|
| A1 | Câbler `LoadMatchSummaries(ctx, ids)` : query `match_registry`+`match_participants` (player-scoped) → réutiliser `mapOneMatchSummary` ; gérer `ids==nil` = N derniers | adapter | M | — |
| A2 | Test `TestLoadMatchSummaries_Shared` (dataset réaliste PVP, ordering) | adapter | S | A1 |
| A3 | Passer `match.history` à `supported` dans `capabilities.toml` | config | S | A1 |
| A4 | `MatchHistoryService.GetPage()` : tenter `dataAdapter.LoadMatchSummaries`, fallback repo ; mapper `canonical.MatchSummary`→`MatchHistoryRawRow` (delta_mmr/perf_score=0) | service | M | A1 |
| A5 | Front : activer page Historique sous `<FeatureGate capability="match.history">` (réutiliser query `matchHistoryAll`) | front | M | A3 |

**Done** : `curl /api/v1/players/{h5-slug}/pages/match-history` renvoie les matchs du
shared h5 (pas le repo legacy), page front s'affiche sous capability, champs
non-canoniques à 0 documentés.

---

### AXE B — Citations / Commendations natives

**État** : assets présents (180+ PNG `static/commendations/halo_5_guardians/`),
167 medals en metadata h5, mais **aucun pipeline title-aware** : pas de
`halo_5/citations_custom.go` `[VÉRIFIÉ MISSING]`, pas de `title_id` dans
`citation_mappings`/`match_citations`, dispatcher Infinite-only. `citations.engine=not_exposed`.

> **Décision produit à trancher en tête (Décision B)** : H5 a des commendations
> NATIVES. Faut-il **rejouer le moteur de citations** (port des fonctions custom) ou
> **afficher les commendations natives telles quelles** (pas de reconstruction) ?
> Recommandation : **natif tel quel** d'abord (effort S, fidèle à la donnée), moteur
> custom seulement si on veut la progression par tier/composite parité Infinite.

| # | Work-item (voie « natif tel quel ») | Layer | Effort | Dépend de |
|---|-----------|-------|--------|-----------|
| B0 | Trancher Décision B (natif vs moteur) — sonde lecture commendations carnage | probe | S | — |
| B1 | Handler title-aware `pages/citations` lisant les commendations natives h5 (medals_earned + medal_definitions h5) sans custom dispatch | handler | M | B0 |
| B2 | Vérifier couverture `medal_id` cités vs `medal_definitions` h5 (degrade 0 si absent) | data | S | B0 |
| B3 | Doc `internal/games/halo_5/CITATIONS_LIMITATIONS.md` (no event rules en FF, weapon_kills format, composites) | config | S | B1 |

| # | Work-item additionnel (voie « moteur custom », SI Décision B = moteur) | Layer | Effort | Dépend de |
|---|-----------|-------|--------|-----------|
| B4 | `title_id` dans `citation_mappings` + `match_citations` (migration) | data | L | B0 |
| B5 | `halo_5/citations_seed.go` (extraire entrées H5 de `seed_citation_data.go`) | data | M | B4 |
| B6 | `halo_5/citations_custom.go` + `RegisterCustomDispatcher` title-aware | adapter | M | B5 |
| B7 | `BackfillMatchCitations` title-aware (param `title_id`) | data | L | B6 |

**Done (voie natif)** : page citations h5 affiche les commendations natives avec assets,
0 gracieux pour medals manquants, limitations documentées. **Done (voie moteur)** : en
plus, dispatch h5≠Infinite, progression par tier validée sur 3-5 joueurs réels.

---

### AXE C — Career Rank (Spartan Rank 1..152)

**État** : **quasi terminé.** `LoadCareerSnapshot` câblé, `career_sr.go` remplit
`RankMax=152`/`XPMax`, et `career_service.go:215` route déjà
`buildHeroProgress(..., heroRankMax(rank))` → `domain.HeroProgress.TotalRanks`.
`[VÉRIFIÉ]` Le « X/272 » survit uniquement comme **fallback front**
(`CareerSummaryCard.tsx:21 HERO_RANK_TOTAL_FALLBACK = 272`) ET quand
`enrichSpartanRank()` échoue (best-effort via dernier match `XpInfo`) → `RankMax` reste
nil → 272.

| # | Work-item | Layer | Effort | Dépend de |
|---|-----------|-------|--------|-----------|
| C1 | Durcir `enrichSpartanRank()` : fallback déterministe (si pas de SR enrichi, fixer `RankMax=152` pour h5 par titre plutôt que laisser nil) | adapter | S | — |
| C2 | Front : remplacer le fallback dur 272 par `heroProgress?.total_ranks ?? (titleSlug==='halo_5' ? 152 : 272)` (filet) | front | S | — |
| C3 | Valider catalogue rangs h5 = labels « SR 1 ».. « SR 152 » (pas labels HINF) ; images SR par level absentes = OK (texte seul) | adapter | S | — |
| C4 | Test `career_service_test.go` : `GetCareerPage()` h5 → `total_ranks=152` | service | S | C1 |

**Done** : page career h5 affiche « SR X/152 », labels « SR N », pas d'images HINF,
CSR Onyx affiché à côté quand `ranked`.

---

### AXE D — Media (upload + timezone)

**État** : pipeline Infinite **production-ready et déjà title-aware** (`PathResolver`
scope par titre, `MediaPathStore` portable, associations append-only). Le rail h5 est
**gated off** : `CapMedia` absent des capabilities h5. `[VÉRIFIÉ title.toml]`

| # | Work-item | Layer | Effort | Dépend de |
|---|-----------|-------|--------|-----------|
| D1 | Ajouter `"media"` aux capabilities h5 (`title.toml`) — débloque `RequireCapability` | config | S | — |
| D2 | Vérifier `shared_matches_v2.duckdb` h5 a `start_time_utc`/`end_time_utc` TIMESTAMPTZ (migration `ADD COLUMN IF NOT EXISTS` sinon) — prérequis association | data | S | D1 |
| D3 | Audit propagation `titleSlug` dans `media_upload.go`/`media_index_service.go` (pas de fallback `halo_infinite`) + test isolation multi-titre | service | M | D1 |
| D4 | Brancher `BuildMediaScanHook` au post-sync h5 (vérifier settings store présent) | service | M | D1 |
| D5 | Test timezone-safety h5 (`parseCaptureTimeFromFilename` + isolation cross-titre) | probe | M | D4 |
| D6 | Doc setup utilisateur (UserTimezone = TZ de capture ; media h5 séparé d'Infinite) | config | S | — |

**Done** : upload media sur joueur h5 stocke sous `data/titles/halo_5/...`, indexation +
association temporelle OK, aucune fuite cross-titre, TZ filename correcte.

> **Dette à ne pas oublier** : `first_joined_tz` est global (`user.settings`), pas
> par-titre. Mitigation court terme = même TZ entre titres. Fix long terme = colonne
> per-player (work-item L, hors prod-gate).

---

### AXE E — PostSync / Notifications / First-Sync

**État** : PostSyncRunner **hardcodé** `defaultProgressionTitleSlug()=="halo_infinite"`
`[VÉRIFIÉ post_sync_deltas.go:115-119]`. Infra notifications riche (40+ catégories)
mais `player_notifications` PK `(xuid,id)` non title-aware, pas de catégorie
`sync_complete_for_title`, pas de bannière front « H5 chargement en cours ».

| # | Work-item | Layer | Effort | Dépend de |
|---|-----------|-------|--------|-----------|
| E1 | PostSyncRunner title-aware : passer `titleSlug` via `ctxkeys.TitleSlug`/param ; retirer `defaultProgressionTitleSlug()` hardcodé | service | M | — |
| E2 | `CategorySyncCompleteForTitle` dans `notifications/types.go` + `AllCategories()` + seed prefs | data | S | — |
| E3 | Détecter sync initial (`len(InsertedMatchIDs)>0 && MatchCountBefore==0`, ou flag `job_type=initial_sync`) | service | M | E1 |
| E4 | Émettre notif `sync_complete_for_title` (best-effort, `TargetRoute`/`TargetSearch.title`) + i18n keys | handler | M | E2, E3 |
| E5 | Front `LoadingBanner` (« Halo 5 chargement… » + bouton « Reviens sur {otherTitle} ») | front | S | — |
| E6 | Intégrer `LoadingBanner` + empty-state dans `NotificationsPage` | front | S | E5 |
| E7 | Navigation notif → `navigate` + `setCurrentTitleSlug` (guard `useEffect` anti-race router) | front | S | E4 |
| E8 | Test intégration multi-titre PostSyncRunner h5 (`-tags=integration`) | adapter | M | E1, E4 |

**Done** : sync initial h5 émet une notif title-aware, clic → home h5 + bascule titre,
page notifications affiche la bannière au lieu d'un vide muet, **zéro régression
notifications Infinite** (vérifier autres callers de `defaultProgressionTitleSlug`).

> **Risque** : détection « sync initial » sur COUNT avant/après est fragile en
> multi-réplica. Préférer un flag `job_type` explicite si dispo.

---

### AXE F — World Leaderboard (verdict conditionnel)

**État** : **non supportable.** Infinite scrape une page web publique paginée
(`leaderboard_scraper.go`). H5 n'expose le CSR **qu'en contexte joueur-individuel**
(service record `ArenaPlaylistStats[].Csr`, carnage `Pre/PostMatchRating`). Aucun
endpoint cross-joueurs top-N ; Waypoint n'a pas de route `/halo-5/leaderboards/` ;
HaloAPI officielle dégradée. Champion (`DesignationId=6`) non exposé dans les DTO.

| # | Work-item | Layer | Effort | Dépend de |
|---|-----------|-------|--------|-----------|
| F1 | **Sonde live** `cmd/probe-h5` : tester `/h5/leaderboards/{playlist}`, `/h5/rankings/{mode}`, batch `servicerecords?players=*` — trancher définitivement | probe | S | — |
| F2 | Audit cryptum-halodotapi (toute mention `leaderboard`/`rankings`/`top`) | probe | S | — |
| F3 | Figer `world.leaderboard` dans `capabilities.toml` : `not_exposed` (verdict probable) OU `degraded` (si batch-query trouvé) | config | S | F1, F2 |
| F4 | (Conditionnel, seulement si batch-query trouvé ET produit accepte UX dégradée) leaderboard synthétique seed-based, `max(Csr)` par playlist | adapter | M | F1 |

**Done** : capability figée avec rationale documenté en `.ai/`. **Verdict par défaut =
`not_exposed`** (exception prod-gate explicite, cf. §5).

---

## 4. Séquencement en phases livrables

### Phase 0 — Sondes & décisions (tête de pipeline, ~0.5 sprint)
Trancher avant de coder pour éviter le travail jeté :
- **F1/F2** : sonde leaderboard → verdict `not_exposed`/`degraded`.
- **B0** : Décision B citations (natif tel quel vs moteur custom).
- **D2** : vérifier schéma `start_time_utc` h5 (gate media).
- Audit grep `defaultProgressionTitleSlug` : recenser tous les callers avant E1.

### Phase 1 — Backend data/adapter (centre de gravité, ~1.5 sprint)
Indépendants entre eux, parallélisables :
- **AXE A** (A1→A4) match.history.
- **AXE C** (C1, C3, C4) career rank finition backend.
- **AXE E** (E1→E4) PostSyncRunner title-aware + notif first-sync (E1 débloque media scan hook D4 et tout multi-titre futur).
- **AXE B** voie retenue (B1→B3 natif, ou B4→B7 moteur).

### Phase 2 — Capabilities & front (~1 sprint)
Débloqué par Phase 1 :
- **A3/A5** activer history (capability + page).
- **C2** filet front 272→152.
- **D1/D3/D4** activer media h5.
- **E5→E7** LoadingBanner + navigation notif.

### Phase 3 — Timeseries + durcissement (~0.5-1 sprint)
- **Timeseries** : brancher `LoadTimeseries` dans `TimeseriesService` + impl adapter h5
  (depuis l'historique), ou figer `not_exposed` honnête avec UI dégradée annoncée.
- **D5/E8** tests intégration, **B/probe** validation 3-5 joueurs réels.
- **F3** figer capability leaderboard.

**Chaîne de déblocage** : E1 (title-aware) → D4 (media scan) + tout multi-titre ;
A1 → A3 → A5 ; B0 → toute la voie citations ; F1 → F3.

---

## 5. Checklist Go / No-Go prod

Conditions **bloquantes** (no-go si non remplies) :
- [ ] Career h5 affiche « SR X/152 » (pas 272) — AXE C done.
- [ ] Match history h5 servie depuis le shared h5 (pas repo legacy) OU `not_exposed`
      assumé avec page masquée proprement — AXE A.
- [ ] PostSyncRunner title-aware, **zéro régression progression/notifs Infinite**
      (test + grep callers) — AXE E1/E8.
- [ ] Capabilities `capabilities.toml` **honnêtes** : aucune capability `supported`
      dont la méthode `Load*` n'est pas réellement câblée (règle déjà inscrite en tête
      du fichier).
- [ ] Pages n/a (battlepass, challenges, prestige) **masquées** par capability, pas
      d'erreur 503 visible utilisateur.
- [ ] Si media activé : isolation cross-titre vérifiée (pas de fuite Infinite↔h5).

Exceptions **explicites** (no-go levé car donnée absente, documenté) :
- **World leaderboard** = `not_exposed` (aucun endpoint H5 cross-joueurs). Acceptable
  tant que le verdict sonde (F1/F2) le confirme.
- **Timeseries** = `not_exposed` toléré au lancement SI l'UI dégrade explicitement
  (pas d'écran vide muet) — sinon le brancher (Phase 3).
- **Citations** = voie « natif tel quel » acceptable pour prod ; moteur custom = post-prod.
- **Squad / Firefight / Forge** = `not_exposed` Phase 2+, documenté handoff.

---

## 6. Risques transverses

1. **Parité capability ↔ adapter** (risque le plus structurant) : la règle en tête de
   `capabilities.toml` impose qu'une capability `supported` ait sa méthode `Load*`
   câblée. Toute activation (A3, D1) doit être précédée du câblage, sinon `Has()==true`
   mène à un appel qui échoue toujours. Garde-fou : test capability↔méthode.

2. **TZ media `first_joined_tz` global** : `user.settings.UserTimezone` partagé entre
   titres. Si l'utilisateur a une TZ de capture différente entre Infinite et h5, le
   reindex cross-titre parse mal les filenames → media non associé. Mitigation prod =
   même TZ ; fix réel = colonne per-player (hors prod-gate, dette L).

3. **B-swap / écritures concurrentes DuckDB** : tout nouveau writer h5 (livesync
   persist, media index, notif) doit respecter la contrainte mono-process RO↔RW et le
   pattern Persister + CHECKPOINT sous lease (cf. ADR 0016/0019, mémoire shared_social
   durable writes). Ne pas réintroduire `OpenReadWrite` sans CHECKPOINT ni d'`UPDATE`
   indexé sur tables d'état (ART). Les écritures `match_skill_rank`/`match_csrs`/
   `player_csr_snapshots`/`pve_match_stats` sont append-only.

4. **Dette damage-model 225 (Infinite) vs 115 (Halo 5)** : les KPI rendement/résistance
   sont normalisés sur baseline 225 dupliquée (compute + SQL + front + P80). H5 = 115.
   Tant que non title-aware, les KPI engagement/damage h5 seront mal échelonnés. À
   traiter avec l'activation engagement h5 (plan `.ai/PLAN_DAMAGE_MODEL_PER_TITLE.md`).
   N'est PAS bloquant prod si `engagement.score` reste `not_exposed` pour h5.

5. **Détection sync-initial fragile** (COUNT avant/après, multi-réplica) : préférer un
   flag `job_type=initial_sync` explicite à la closure capturée.

6. **MatchSummary canonique incomplète** : pas de `delta_mmr`/`performance_score` →
   historique h5 affichera ces colonnes vides (acceptable Phase 2, à documenter dans le
   mapper).

7. **Stack traces split-ownership** : Match Detail = scoreboard + events sur deux voies
   ; en cas d'échec, l'origine (legacy vs canonique) doit rester lisible. Côté h5 c'est
   tout canonique (atténue le risque).
