# Dette assumée — 2026-Q3

> Bilan des items **PLANIFIÉS** différés du plan `PLAN_TRAITEMENT_AUDITS_2026-07.md`
> (N5, créé 2026-07-05). Ne liste QUE les reports intentionnels statués `[!]`/`[~]`
> (chantiers, activations phasées, follow-ups measure-first). Les **découvertes
> incidentes** hors périmètre vont dans la §7 « Découvertes » du plan (backlog
> séparé), PAS ici. Chaque entrée : pourquoi différé + condition de reprise.

## 1. Chantier K — Structure & couches (le plus gros)

- **K (tous sous-lots)** — la racine `api/` cesse d'être une 2e couche service ;
  god functions/packages découpés ; chemins via PathResolver. Calibré comme le seul
  lot aux comptes EXACTS (143/127/112/40) — juste énorme (4-6 j). Chantier dédié,
  commits par sous-lot. Inclut :
  - **F12** — migration film 18 fichiers `analysis/` → `games/halo_infinite/film/`
    (extraction d'un sous-système délicat, pas mécanique).
  - **J5** — `LoadAll` full-history par hit → cache joueur invalidé post-sync.
    **DÉCISION reco 2026-07-05 (à confirmer au démarrage)** : invalidation ÉVÉNEMENTIELLE
    par le chemin de complétion sync (après CHECKPOINT append-only) via un
    `InvalidatePlayerHistoryCache(xuid, titleSlug)` — **PAS de TTL temporel** (un TTL laisse
    voir des données périmées ; l'événement sync est le vrai signal, cohérent doctrine
    convergente append-only). Propriétaire = le sync, pas le lecteur.
  - **L2-(1)** — ratchet « pas de SQL/`Open*` dans api/ » (dépend de K ; baseline
    décroissante datée une fois K livré).
  - **K1g** — fusion de la double-lecture asset-drawer au boot (server.go), croisé L7.
- **Reprise** : session dédiée ; PLAN_TITLE_AGNOSTIC + calibration K du plan font foi.

## 2. i18n manuel — I1/I2/I4 (décision utilisateur A, 2026-07-05)

Le gate lint (I5, règle `no-hardcoded-strings` en `error`) est ATTEINT (I3+I5 livrés).
La règle est volontairement ciblée (texte JSX + attributs) et NE couvre PAS les args
de fonction ni les libellés courts. Reste donc un chantier de couverture bilingue réel
mais NON exigé par le gate :

- **I1** — ✅ **LIVRÉ (2026-07-05)**. 5 composants onboarding/auth bilingues via
  `common.toml` (commits 6462887f4, 81fed7aad, d39cc5d1a).
- **I2 labels** — ✅ **LIVRÉ (2026-07-05)**. Scoreboard, heatmaps (+ centralisation
  `calendar.ts` + garde-rail), filtres, breakdowns synthesis.
- **I2b — figement `toLocaleString('fr-FR')` (RESTE ~24 sites)** : pont canonique
  `lib/formatters/intlLocale.ts` créé + SynthesisPage (15 sites) migré. Résiduel = helpers
  PURS / builders ECharts / consts module SANS `locale` en scope :
  `career/CareerChartsSection.gauges.tsx` (2, buildRankGaugeOption/buildHeroGaugeOption),
  `session-detail/SessionDamageComposite.tsx` (fmtInt const), `session-detail/_shared.ts`,
  `session-detail/SessionMatchesTable.tsx` (toExplorerRow), `synthesis/SynthesisWeapon{Kills,Accuracy}Chart.tsx`
  (buildOption ECharts), `media/{MediaViewer,CoverFlowModal,MediaMatchPicker}.tsx` (dates),
  `home/HomeSessionCarousel.tsx` (formatSessionDate/Duration), `prestige/LeaderboardPP.tsx`,
  `timeseries/TimeseriesSquadAdapted.tsx`, `match-view/MatchScoreboard.logic.ts` (fmtScore).
  **Fix** : threader `locale` (param signature / prop) puis `X.toLocaleString(intlLocale(locale))`.
  Effort mécanique mais par-site (surface rendu chart). Impact = cosmétique (séparateurs
  nombre, ordre date EN). **Garder** : valeurs objet `fr` d'i18n (légitimes), `formatDateShort`
  (verrou chart DD/MM documenté). Poser un garde-rail avec allowlist des exceptions.
- **I4** — **155 ternaires** `locale === 'en'/'fr' ?` (vérifié 2026-07-05 ; audit disait 88),
  **tous DÉJÀ bilingues** → refactor d'ORGANISATION (pas de lacune user-facing). Deux lots :
  (a) **40 ponts** `? 'en-US' : 'fr-FR'` → `intlLocale(locale)` (helper prêt, I2b) — dédup #6 ;
  garder les 6 `'en-GB'` (date EU) ; import aliasé si collision `const intlLocale`.
  (b) **~115 libellés** → i18n.ts par feature ; concentration AscensionProfileTab(16),
  ArcPresetPicker(10), MatchViewPage(9), MatchEncountersTable(6), ExplorerEncounterBriefing(5),
  longue traîne 1-2/fichier tolérable. Non commencé (priorité basse : déjà bilingue).
- **Reprise** : tâche front dédiée ; système cible = manifests TOML + `build_i18n_manifests.mjs`
  (labels) et `intlLocale()` (formatage nombre/date).

## 3. Infra de test — M1, M5

- **M1** — test intégration `RecomputeLUSRCanonicalForPlayer`. Le replay délégué est
  déjà couvert (30+ `TestRunLUSRV2Shadow_*`) ; reste la sentinelle `is_reset`, qui exige
  une fixture au schéma COURANT (le scaffolding `openShadowTestDB` est en retard, sans
  `is_reset`). Ratio effort/valeur faible → follow-up ciblé.
- **M5** — goldens par slug : exige de GÉNÉRER des captures d'endpoints H5 (infra
  substantielle), pas juste `goldenPathForSlug` + `t.Run(slug)`.
- **Reprise** : tâches ciblées ; corriger d'abord le schéma du scaffolding (M1).

## 4. Perf DuckDB — J1(2), J2, J3, J4, J6, J7, J9 (measure-first)

J1(1) (instrumentation expvar `duckdb_pool_stats`) + J8 (constantes de pool) livrés.
Les optimisations DÉPENDENT d'une mesure avant/après SOUS CHARGE (runtime) que J1(1)
rend possible — les faire à l'aveugle optimiserait un chemin non mesuré + risque de
changement de résultat (J3/J4/J7) ou de wiring provider (J9).

- **J1(2)** pool lecture player DBs · **J2** budgets mémoire/threads · **J3**
  `GetHistoryForAvgBulk` · **J4** `LoadSquadMatchesBulk` · **J6** 8 N+1 batchables ·
  **J7** CTE Q26 bornée · **J9** emprunt cross-titre B-swap-safe.
- **J2 — ✅ LIVRÉ (2026-07-05)**. Mesure VPS (ssh) : 2 vCPU/2 Go no-swap, conteneur 845 Mo
  au repos, ~256 Mo dispo. DuckDB n'avait AUCUNE limite → défaut ~1.5 Go = risque OOM. Fix :
  `SET memory_limit + threads` sur chaque connexion (hook connector `openSQLDBFor`), défaut
  `512MB`/`2`, override env. Suite intégration duckdb verte. (J1(2)/J3/J4/J6/J7/J9 restent.)
- **Reprise** : lire `/debug/vars` `duckdb_pool_stats` sous charge, puis optimiser +
  valider avant/après.

## 5. Gouvernance — L1 (re-scope), L2-(3/4/5), L5

- **L1** — drift OpenAPI : approche « bulk-résoudre les 22 DIVERGENT via emit Huma »
  INVALIDÉE (Huma dérive `string`, perdant les énums/nullabilité ajoutés à la main →
  dégrade le contrat + casse le typecheck front). Re-scope : catégoriser dérive-réelle
  vs enrichissement-voulu, durcir avec allowlist (PAS à 0).
- **L2-(3/4/5)** — parités capability (invariant F15-14 confirmé RÉEL) : tests exigeant
  de charger 2 sous-systèmes de config.
- **L5** — ✅ **LIVRÉ (2026-07-05, commit 91492e360)**. 7 registres feature-local + 8
  littéraux inline centralisés dans `queryKeys` (clés identiques au byte) + garde-rail
  `keys.guard.test.ts`. Le « 180 » sur-comptait (la plupart consommaient déjà `queryKeys`).
- **Reprise** : tests config-integration (L2) ; re-scope (L1). (L5 clos.)

## 6. Front structurel — N1, N2, N3(d)

- **N1** — `LeaderboardBlock` `<table>` native (576 L, colonnes conditionnelles, rendu
  de cellules riche) → TanStack Table.
- **N2** — `SquadLayout` god component (~630 L) → 3 hooks + `SquadFilterBar`.
- **N3(d)** — `MatchCard` ~470 L découpé par responsabilité.
- **Reprise** : session front avec REVUE VISUELLE (Gate N) — non faisable à l'aveugle.

## 7. Reports de lots antérieurs (déjà planifiés avant l'audit 2026-07)

- **E7** — télémétrie `legacy_source_used` puis suppression des fallbacks legacy (auth).
- **F7** — activation engagement H5 (canonicalisation adapter + calibration cold-start ;
  chantier futur Halo 7).
- **F8/F9** — Phase 1b activation multi-titre.
- **D2** — ADR 0023 Phase 5 (différé ≥7 j après mise en prod de D1a).

---

_Mis à jour au fil des sessions. Les items livrés sortent de cette liste ; les nouveaux
reports planifiés y entrent avec leur condition de reprise._
