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
  commits par sous-lot. **GROS DU LOT LIVRÉ 2026-07-06** (~40 commits, cf. plan parent §6
  Journal LOT K) — NewRouter 89 L, racine api 4 root files, god-files duckdb/service/sync
  découpés/exemptés. Ci-dessous les RÉSIDUS réels post-2026-07-06.
  Inclut (toujours différés) :
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
- **Résidus K réels (post-2026-07-06)** — reprise = sessions dédiées :
  - **K1a-cœur** (extraction `service/postsync/`) — `[~]` OPTIONNEL : LARGEMENT SUBSUMÉ par
    K3d (relocation CoachAdvisorBundle/PrestigeBundle + pipeline post-sync hors racine api/).
    Reste « couche métier stricte » (post_sync → `service/postsync/` + formules → `analysis/`).
    Non bloquant. Reprise : cosmétique de couche, faible ROI.
  - **K1b chemin legacy** — `[!]` : dédup du CHEMIN LEGACY de la cascade auth (sync_meta DuckDB
    + env var) NON faite (déléguer introduirait 3 divergences : marquage reauth, rotation→DuckDB
    si store nil, granularité télémétrie). COUPLÉ D2 : à réconcilier quand le legacy disparaît
    (Phase 5 ADR 0023). Le cœur (cascade store MSAL→OAuth) est livré (`2da454304`).
  - **K1d reste** — `[!]` : relocation `ExpandPlaylistChildren` hors racine api/ + DDL→migration
    + batch 3-req/entry (croisé J6). Couplé à la sortie du post-sync de la racine api/ (famille
    K1a). Le dédup upsert ART-safe est livré (`d442fb507`).
  - **K1h reste** — `[!]` : collection de handlers→services+ports (le slug SQL weapon-coverage
    est fait `de52897c3`/`b5a978e1e` ; reste la migration des handlers restants vers la couche
    service). Session dédiée.
  - **K1j reste** — `[!]` : `openspartan_post_import_service` reçoit encore un `*sql.DB` direct
    (D-MV2 : catalog/media repos ~250 L SQL + port + wiring, en partie fait `21d9154db`). Reste
    à router le post-import via un port propre.
  - **K1k reste** — `[!]` : la factory `career_live_fetcher` importe encore `sync` (les DTO
    career-live sont promus en domain `ad2efea74` mais l'inversion de dépendance factory→sync
    n'est pas complète). Découplage restant.
  - **K1l reste** — `[!]` : chemins legacy divers (stash friends, layout seed_demo, double
    mécanisme `config.go`). PlayersRootDir + CacheRootDir sont dédupliqués (`f271bbdc5`/
    `0316ef1c1`) ; reste la longue traîne de chemins non-PathResolver résiduels.
  - **K1n** — `[!]` : déplacements de couche à FAIBLE VALEUR (les dédups médiane/engagement
    sont faits `6b1662655`/`fad35380f`/`4208c9f85` ; reste des micro-relocations non prioritaires).
  - **K2b drain** — `[!]` INFAISABLE documenté : l'extraction du `drain` de `SyncEngine.run`
    est bloquée par un `defer` LIFO load-bearing (l'ordre de libération des ressources est
    sémantiquement significatif ; extraire casserait le lifecycle). La pagination, elle, EST
    extraite (`6aa7a260e`). Reste en l'état (documenté sur pièces au commit K2b).
  - **K2c nolint** — `[~]` : `BuildEngine`/`RunOnceTrigger` gardent un `//nolint:funlen`
    JUSTIFIÉ (fonctions cohésives factory/trigger) ; le run-loop est extrait (`02ad47ac2`).
    Rien à faire hors re-découpe cosmétique.
  - **K3a poursuite** — `[!]` OPEN-ENDED : scission des god-packages (duckdb 143 / service 127 /
    sync 111 fichiers) — halo5/prestige extraits (`1475192bc`/`29d69fb81`), mais la scission
    complète « 1 domaine = 1 commit » est mécanique-mais-volumineuse, sans fin nette.
  - **K3b ratchet imports croisés** — `[!]` : le ratchet interdisant les imports croisés entre
    features est à bâtir (la feature teammates est extraite via squadagg leaf `e7aea7e63`, mais
    le garde-rail générique anti-couplage inter-features n'existe pas encore).
  - **K3f décisions packages** — `[!]` DÉCISIONS structurelles à statuer (pas du code) :
    doublon `notify/` vs `notifications/` (renommer ou documenter) ; `prestige/`+`campaign/`
    → `progression/` (ou documenter le choix) ; `metadata/` renommé ; `openspartan/` → `platform/` ;
    échéance `legacymatch` [TRACKÉ]. Décisions de packages = suite dédiée, pas d'urgence.
- **Reprise** : sessions dédiées ; PLAN_TITLE_AGNOSTIC + calibration K du plan font foi.

## 2. i18n manuel — I1/I2/I4 (décision utilisateur A, 2026-07-05)

Le gate lint (I5, règle `no-hardcoded-strings` en `error`) est ATTEINT (I3+I5 livrés).
La règle est volontairement ciblée (texte JSX + attributs) et NE couvre PAS les args
de fonction ni les libellés courts. Reste donc un chantier de couverture bilingue réel
mais NON exigé par le gate :

- **I1** — LIVRÉ (2026-07-05), CLOS → voir §8 « Livrées depuis la rédaction ».
- **I2 labels** — LIVRÉ (2026-07-05), CLOS → voir §8. (I2b ci-dessous : sous-partie du chantier
  i18n en cours, conservée ici pour tracer couvert vs restant.)
- **I2b — figement `toLocaleString('fr-FR')` — ✅ LIVRÉ (2026-07-05)**. Pont canonique
  `lib/formatters/intlLocale.ts` + TOUS les sites figés composant/chart migrés (SynthesisPage,
  synthesis weapon charts, media ×3, career gauges/encounters, LeaderboardPP, session-detail
  ×2, HomeSessionCarousel, TimeseriesSquadAdapted) via `intlLocale(locale)` threadé. 2 sites
  figés étaient du **code mort** (`formatScore`, `formatShortDateTime`) → supprimés (CLAUDE.md
  n°7). **Exceptions conservées (légitimes)** : `formatDateShort` (verrou chart DD/MM documenté,
  date.ts) + valeurs objet `fr` des i18n.ts (branche FR d'un bilingue). Pas de garde-rail
  automatique posé : un guard lexical `'fr-FR'` flaguerait les exceptions légitimes (allowlist
  trop large, faible signal) — `intlLocale()` EST la convention. Gate : typecheck 0, eslint 0.
- **I4** — **155 ternaires** `locale === 'en'/'fr' ?` (vérifié 2026-07-05), **tous DÉJÀ
  bilingues** → refactor d'ORGANISATION. Deux lots :
  (a) **41 ponts** `? 'en-US' : 'fr-FR'` → `intlLocale(locale)` — ✅ **LIVRÉ (2026-07-05)**,
  dédup #6 complète (6 commits `refactor(I4)`), 6 `'en-GB'` conservés, typecheck/eslint/261
  tests verts.
  (b) **libellés** — sur-compté (le « 114 » incluait dict/locale-prop/data-selection). Vrais
  scattered ≈ 40. **26 migrés (2026-07-05)** : fichiers haute densité (AscensionProfileTab 16,
  MatchViewPage 3, PrestigeSquadProgress 3, Ascension{Realisations,Coaching}Tab 4) → feature
  i18n. **Reste** (accepté/scopé) : `ArcPresetPicker` dict local consolidé (pattern accepté) ;
  ~~3 composants dupliqués SplitBar+AllyEnemySplitBar+KDSplitBar~~ → ✅ **LIVRÉ (2026-07-05)** :
  extraits vers `features/_shared/EncounterSplitBars.tsx` (dédup #6, tooltips i18n résolus au
  passage) ; longue
  traîne 1-2/fichier (**tolérable** par règle plan). Data-selection `_fr:_en` = légitime, garder.
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

## 4. Perf DuckDB — restent J1(2), J4, J6 measure-first + J5 chantier (J2/J3/J7/J9 livrés)

J1(1) (instrumentation expvar `duckdb_pool_stats`) + J8 (constantes de pool) livrés.
Les optimisations DÉPENDENT d'une mesure avant/après SOUS CHARGE (runtime) que J1(1)
rend possible — les faire à l'aveugle optimiserait un chemin non mesuré + risque de
changement de résultat (J3/J4/J7) ou de wiring provider (J9).

- **LIVRÉS (2026-07-05/06)** : **J8** + **J1(1)** (instrumentation `duckdb_pool_stats`) ·
  **J2** budgets mémoire/threads (`df5832d60`+`305b6b959`) · **J3** `GetHistoryForAvgBulk`
  (`dfeb199f3`) · **J7** CTE Q26 bornée (`f6b8cce4a`) · **J9** emprunt cross-titre revu +
  documenté (`8d385cc70`, sûr en opération normale, aucun changement de code justifié).
- **RESTENT différés measure-first** (VPS requis) :
  - **J1(2)** — pool lecture player DBs 2-4 conns : BLOQUÉ RUNTIME (lecture des stats SOUS
    CHARGE + audit des UPSERT reposant sur MaxOpenConns(1)). Impossible en test local.
  - **J4** — `LoadSquadMatchesBulk` : petit-N (1-4 coéquipiers sélectionnés), refacto LOURD +
    correctness-sensible → measure-first (optimiser à l'aveugle proscrit).
  - **J6** — 8 N+1 batchables : TOUS des chemins d'arrière-plan petit-N (sync/backfill/catalog),
    pas HTTP chauds → measure-first.
  - **J5** — `LoadAll` full-history par hit → CHANTIER DÉDIÉ (décision produit cache, cf. §1 K).
- **Reprise** : lire `/debug/vars` `duckdb_pool_stats`+`duckdb_budgets` sous charge réelle
  (LOT V10c du plan de clôture, VPS requis), puis optimiser J1(2)/J4/J6 + valider avant/après.

## 5. Gouvernance — L1 (re-scope), L2-(3/4/5), L5

- **L1** — drift OpenAPI : approche « bulk-résoudre les 22 DIVERGENT via emit Huma »
  INVALIDÉE (Huma dérive `string`, perdant les énums/nullabilité ajoutés à la main →
  dégrade le contrat + casse le typecheck front). Re-scope : catégoriser dérive-réelle
  vs enrichissement-voulu, durcir avec allowlist (PAS à 0).
- **L2-(3/4/5)** — parités capability (invariant F15-14 confirmé RÉEL) : tests exigeant
  de charger 2 sous-systèmes de config.
- **L5** — LIVRÉ (2026-07-05, `91492e360`), CLOS → voir §8 « Livrées depuis la rédaction ».
- **Reprise** : tests config-integration (L2) ; re-scope (L1). (L5 clos.)

## 6. Front structurel — N1, N2, N3(b/c/d/e)

- **N1** — `LeaderboardBlock` `<table>` native (576 L, colonnes conditionnelles, rendu
  de cellules riche) → TanStack Table.
- **N2** — `SquadLayout` god component (~630 L) → 3 hooks + `SquadFilterBar`.
- **N3(b)** — skeleton (état de chargement) à extraire/uniformiser (petit fix front).
- **N3(c)** — deps du listener clavier à corriger (nécessite analyse des deps `navigate` —
  mieux traité avec revue visuelle, cf. plan parent l.~1670).
- **N3(d)** — `MatchCard` ~470 L découpé par responsabilité.
- **N3(e)** — rename `joinAndSort` (sans nom cible spécifié par l'audit — à trancher en session).
- **Reprise** : session front avec REVUE VISUELLE (Gate N) — non faisable à l'aveugle.
  (N3(a) = faux positif React.lazy confirmé, PAS de dette.)

## 7. Reports de lots antérieurs (déjà planifiés avant l'audit 2026-07)

- **D1a→D2** — télémétrie `legacy_source_used` (D1a, livrée : `observability/legacy_source.go`)
  puis suppression des fallbacks legacy auth (D2, ADR 0023 Phase 5). Différé ≥7 j après mise en
  prod de D1a. Condition de reprise : télémétrie `legacy_source_used` observée ≥7 j après la date
  de deploy notée (branche `refactor/adr0023-phase5`). [Anciennement mal étiquetée « E7 » — le
  contenu décrivait D1a/D2 ; corrigé 2026-07-07/V6c.]
- **E7 (le VRAI)** — DDL bootstrap `sync/schema.go` → `internal/migration`. Item MAL labellisé
  « mineur » par l'audit : en réalité refactor PROFOND du boot/provisioning de TOUTES les DBs
  (DDL dupliqué-mais-aligné avec `create_base_*_schema`, logique de vues au boot corrigeant des
  bugs prod documentés — attach RO/RW, xuid bruts), couplé à la transition b23/b25
  (title-ownership). Statué `[!]` au Gate E (règle 9). Condition de reprise : chantier dédié
  APRÈS stabilisation b23/b25 (détail : plan parent item E7, l.~1971).
- **F7** — activation engagement H5 (canonicalisation adapter + calibration cold-start ;
  chantier futur Halo 7).
- **F8/F9** — Phase 1b activation multi-titre.
- **D2** — ADR 0023 Phase 5 (différé ≥7 j après mise en prod de D1a — cf. D1a→D2 ci-dessus).

## 8. Livrées depuis la rédaction (sorties de la dette — traçabilité seule)

> Les items ci-dessous étaient différés à la rédaction de ce fichier (N5, 2026-07-05) puis
> LIVRÉS+CLOS. Conservés ici en pointeur (pas en dette) pour la traçabilité du merge.

- **I1** (i18n onboarding/auth) — 5 composants bilingues via `common.toml`
  (`6462887f4`/`81fed7aad`/`d39cc5d1a`). CLOS 2026-07-05.
- **I2 labels** (scoreboard/heatmaps/filtres/breakdowns) + centralisation `calendar.ts` +
  garde-rail. CLOS 2026-07-05. (Figement I2b : cf. §2, chantier i18n en cours.)
- **L5** (centralisation query-keys) — 7 registres feature-local + 8 littéraux inline →
  `queryKeys` (clés identiques au byte) + garde-rail `keys.guard.test.ts` (`91492e360`).
  CLOS 2026-07-05.
- **J2/J3/J7/J9** (perf DuckDB) — cf. §4 (LIVRÉS 2026-07-05/06).
- **Chantier K** — gros du lot LIVRÉ 2026-07-06 (~40 commits) ; seuls les résidus listés
  en §1 restent différés.

## 9. Clôture V (2026-07-07) — restants du plan de clôture

> Lots du `PLAN_CLOTURE_AUDITS_2026-07.md` non exécutables en autonomie (bloqués VPS/user).
> Les lots V1-V6 (code + tracker) sont livrés ; V7-V8 sont des follow-ups qualité bornés.

- **V9 — Données de prod** — `[!]` BLOQUÉ VPS : audit read-only sur COPIE restaurée
  (first_joined_time décalés ~964, is_ranked OpenSpartan, orphelins registry↔participants↔medals,
  colonnes DROP différé DEC-6/G5, désync watermarks LUSR v2) puis correctifs backfill copie→prod.
  Condition de reprise : copie prod fournie par V10a + fenêtre convenue avec l'utilisateur.
- **V10 — Exploitation** — `[!]` BLOQUÉ VPS : test de RESTAURATION restic (V10a, fournit la
  copie pour V9) + checklist de déploiement exécutable (V10b) + fenêtre d'observation runtime
  post-merge (V10c, débloque J1(2)/J4/J6 measure-first). PAS d'alerting uptime (décision user).
- **GATE HUMAIN** — `[!]` USER : revue visuelle (A1/A2/A5, pages H5 Gate F, i18n EN I1/I2/I4,
  session-detail G3, smoke Home/Career/Squad/Explorer/Sessions FR+EN, 1 joueur Infinite + 1 H5).
- **PLAN DE MERGE** — `[!]` USER : CI verte + répétition générale sur copie prod + gate
  live-sync manuel (6 joueurs, tokens réels) + fenêtre calme (push main = deploy auto, pas de
  kill-switch V1) + rollback documenté (`git revert -m 1` + risque migrations non réversibles).
  D2 (ADR 0023 Phase 5) = chantier séparé déclenché par télémétrie `legacy_source_used` ≥7 j après deploy.

---

_Mis à jour au fil des sessions. Les items livrés sortent de la dette active (déplacés en §8
« Livrées depuis la rédaction » pour la traçabilité) ; les nouveaux reports planifiés entrent
en §1-§7 avec leur condition de reprise._
