# PLAN — Dépendances en attente : Go deps (#67), echarts 5→6 (sécurité), TypeScript 7 (#70)

> Statut : PRÊT, NON EXÉCUTÉ (préparé par l'agent, exécution différée par l'utilisateur).
> Contrat : skill `plan-execution` (ordre strict, statuts `[x]`/`[~]`/`[!]`, zéro report
> d'étape exécutable, découvertes hors périmètre consignées ailleurs).
> Origine : revue des 5 PRs Dependabot ouvertes le 2026-07-27. 3 étaient vertes en CI et ont
> déjà été mergées same-day (#66 actions/checkout, #68 npm-minor-patch 20 paquets, #69
> jest-dom 7.0.0) — voir `.ai/thought_log.md` [2026-07-27]. Les 3 lots ci-dessous sont ceux
> qui restaient bloqués, + le bump `echarts` déjà noté « REPORTÉ » dans `.ai/BACKLOG.md`
> (ligne 63) que l'utilisateur veut cette fois traiter plutôt que re-différer indéfiniment.

## Lots — indépendants, 1 branche chacun

Les 3 lots n'ont aucune dépendance entre eux (Go / frontend charting / toolchain TS) :
3 branches séparées, pas de séquence imposée par le code — mais **ordre d'exécution
recommandé A → B → C** (du plus rapide/sûr au plus incertain).

---

## Lot A — PR #67 : bump Go deps (10 paquets, groupe `go-minor-patch`)

**Contenu de la PR** : `huma` 2.38.0→2.39.0, `duckdb-go/v2` 2.10504.0→2.10505.0,
`kin-openapi` 0.144.0→0.145.0, `go-chi/chi/v5` 5.3.0→5.3.1, `go-chi/httprate` 0.15.0→0.16.0,
`go-toml/v2` 2.4.2→2.4.3, `x/crypto` 0.53.0→0.54.0, `x/sync` 0.21.0→0.22.0, `x/sys`
0.46.0→0.47.0, `modernc.org/sqlite` 1.53.0→1.54.0.

**Pourquoi bloqué** : CI rouge (run 30202478778, job "Go Coverage + Baseline"). Cause
identifiée, PAS un flake : `TestBareRoutesRatchet_NoUnguardedRouteOutsideAllowlist`
(`apps/go-api/internal/api/bare_routes_ratchet_test.go`) détecte 2 routes nouvelles hors
allowlist : `QUERY /static/*` et `QUERY /static/commendations/*`. Le bump `go-chi/chi`
5.3.0→5.3.1 fait apparaître la méthode HTTP `QUERY` sur les routes déjà couvertes
« toutes méthodes » du file-server statique (GET/HEAD/OPTIONS/POST/PUT/PATCH/DELETE/
CONNECT/TRACE y figurent déjà comme assets publics, cf. lignes 94-111 du fichier). Ce
n'est PAS un affaiblissement de garde : ces routes sont déjà publiques par conception,
`QUERY` est juste une méthode HTTP supplémentaire sur la même surface déjà justifiée.

### Étapes

- [ ] **A1**. `git checkout dependabot/go_modules/apps/go-api/go-minor-patch-6f47a951bf`
      (ou rebase sur une branche locale `chore/go-deps-bump-query-allowlist`).
- [ ] **A2**. Ajouter 2 entrées à `publicRoutesAllowlist`
      (`apps/go-api/internal/api/bare_routes_ratchet_test.go`) :
      `"QUERY /static/*"` et `"QUERY /static/commendations/*"`, justification
      "assets statiques (file-server chi, toutes méthodes) — méthode QUERY ajoutée par
      go-chi 5.3.1 (bump 2026-07)".
- [ ] **A3**. `cd apps/go-api && go test ./...` — 0 FAIL, code de sortie 0 (ancrer le grep
      sur `^--- FAIL:`, piège connu des faux verts filtre `FAIL` nu).
- [ ] **A4**. `go test -tags=integration -p 1 ./...` — **OBLIGATOIRE** ici : le bump touche
      `duckdb-go/v2` (2.10504.0→2.10505.0). Avant de lancer, vérifier le changelog/diff de
      ce point-release pour toute mention d'index ART/B-tree — historique du projet avec
      le bug DuckDB #23046 (ADR 0019/0026) oblige une vigilance accrue sur CE paquet
      spécifiquement, même pour un bump patch en apparence anodin.
- [ ] **A5**. `go vet ./...` puis `make go-api-lint`.
- [ ] **A6**. Vérifier si `go-chi/httprate` 0.16.0 a changé le format des headers de
      rate-limit (`RateLimit-*`) — grep les tests qui assertent sur ces headers
      littéralement, au cas où le changement casse une assertion silencieusement passée
      par ailleurs (pas vu en CI mais pas couvert par `go vet`).
- [ ] **A7**. Merge PR #67 (squash), vérifier CI verte sur `main`, vérifier le déploiement.
- [ ] **A8**. Entrée `.ai/thought_log.md`.

**Gate de clôture** : A3-A6 verts + CI GitHub verte sur la branche AVANT merge (pas
seulement un rejeu local).

---

## Lot B — bump `echarts` 5.6.0 → 6.1.0 (alerte sécurité XSS, CVE-2026-45249)

**Contexte** : alerte Dependabot moderate sur `echarts` 5.6.0, corrigée en 6.1.0 (note
release écharts 6.1.0 : *"[Fix] [lines] Fix potential tooltip XSS vulnerability in lines
series"* — correspond au CVE). PR dependabot #49 (5.6.0→6.1.0) a été **fermée manuellement
le 2026-06-22** par décision utilisateur : tous les checks automatiques passaient (peer
`echarts-for-react@3.0.6` déclare compat `^6.0.0` — confirmé à nouveau le 2026-07-27 via
`npm view`), mais **aucun test ne couvre le rendu visuel réel** (`vitest` mocke `echarts`
partout) — un major du moteur de rendu de TOUS les graphes de l'app ne doit pas partir
sans un minimum de garde-fou visuel. `dependabot.yml` a un `ignore` daté pour ce major,
avec son propre critère de retrait écrit noir sur blanc : *"au chantier v7.3 (critère :
npm ls echarts >= 6.1.0)"* — v7.3 a shippé (26-27/07) SANS traiter ce point : en retard.

**Décision de ce lot** : combler le trou (« pas de couverture visuelle ») plutôt que de
re-fermer une 3e fois. `echarts` reste latest = 6.1.0 (vérifié 2026-07-27, aucune version
plus récente depuis juin) — pas de nouvelle version à rattraper.

### Étapes

- [ ] **B1**. Nouvelle branche `fix/echarts-6-security-bump` depuis `main`.
- [ ] **B2**. **AVANT tout bump** : ajouter le garde-rail visuel manquant — tests Playwright
      `toHaveScreenshot` sur les pages e2e existantes les plus denses en graphes (celles
      déjà nommées dans `.ai/BACKLOG.md` comme critère de go : Timeseries, Compare,
      Synthesis, Match view). Couvrir au moins un représentant par famille de wrapper
      `apps/web/src/components/charts/` (12 wrappers echarts réels, `FragSunburst` exclu —
      déjà réimplémenté en SVG pur, hors périmètre) : `TimeseriesLineChart`,
      `BarGroupedChart`/`BarStackedChart`, `DonutChart`, `RadarChart`, `Heatmap2DChart`,
      `ScatterChart`, `HistogramChart`, `EngagementCurve`, `OutcomeSequenceTape`,
      `FirstBloodLanes`, `FragWeaponBreakdown`.
      - Piège rendu canvas : anti-aliasing non-déterministe cross-environnement → utiliser
        `maxDiffPixelRatio` (tolérance), pas un diff pixel-exact ; faire tourner le baseline
        sur le MÊME runner que la CI (ubuntu, cf. job E2E existant).
      - Committer les captures baseline (echarts 5.6.0) dans un commit ISOLÉ, pour que le
        commit du bump ne contienne QUE le changement de version + son propre diff visuel.
- [ ] **B3**. `apps/web` : `npm run test:e2e` (ou commande Playwright équivalente) — les
      nouveaux snapshots doivent être stables et verts SUR 5.6.0 avant de bumper quoi que
      ce soit (sinon on ne mesure rien).
- [ ] **B4**. Bump `echarts` → `^6.1.0` dans `package.json` + `package-lock.json` (dependabot
      ayant fermé la PR après `ignore`, le bump se fait à la main, pas via
      `@dependabot recreate`).
- [ ] **B5**. Retirer l'entrée `ignore` `echarts` majeure de `.github/dependabot.yml`
      (son propre critère de retrait est rempli) + retirer l'entrée correspondante de
      `.ai/BACKLOG.md` (ligne ~63).
- [ ] **B6**. Gates : `npm run typecheck` (purge `node_modules\.tmp` avant, piège cache
      connu) + `npm run lint` + `npm run test` (vitest — mocké, ne doit rien casser côté
      logique) + `npm run test:e2e` COMPLET (screenshots inclus — c'est le vrai gate de
      cette PR, pas le vitest).
- [ ] **B7**. Tournée visuelle manuelle (utilisateur ou navigateur piloté) sur les 4 pages
      denses citées (Timeseries, Compare, Synthesis, Match view) — les screenshots
      automatisés réduisent le risque mais ne remplacent pas un œil humain sur un moteur de
      rendu partagé par toute l'app, surtout à la première mise en place du harness.
- [ ] **B8**. Merge une fois B6+B7 verts. Entrée `.ai/thought_log.md`.

**Gate de clôture** : e2e + screenshots verts EN CI (pas seulement en local) + sign-off
visuel utilisateur explicite sur B7.

---

## Lot C — PR #70 : TypeScript 6.0.3 → 7.0.2 (major)

**Pourquoi bloqué** : CI rouge, cause claire — TS 7 **supprime** l'option `baseUrl`
(`tsconfig.app.json:20` → erreur TS5102 ; ligne 22 → TS5090 chemin non-relatif). Migration
attendue : `"paths": {"*": ["./*"]}`.

**Blocage STRUCTUREL supplémentaire, PLUS important que le baseUrl** : `typescript-eslint`
8.65.0 (déjà mergé dans #68) déclare un peerDependency `typescript: ">=4.8.4 <6.1.0"`
(vérifié `npm view typescript-eslint@8.65.0 peerDependencies` le 2026-07-27) — **TS 7.0.2
est explicitement HORS du range supporté**. Corriger seulement `baseUrl` ne suffira donc
pas : soit `typescript-eslint` publie un support TS7 avant qu'on retente ce bump (à
revérifier au moment de l'exécution), soit le lint cassera silencieusement ou avec des
erreurs de parsing sur des features TS7 récentes. **Ne pas exécuter ce lot tant que ce
point n'est pas revérifié** (`npm view typescript-eslint peerDependencies` à l'instant T).

### Étapes

- [ ] **C1**. Revérifier `npm view typescript-eslint@latest peerDependencies` — si TS 7.x
      toujours hors range, **STOP**, laisser la PR fermée/en attente et re-planifier plus
      tard (report justifié par dépendance externe, pas un choix arbitraire).
- [ ] **C2**. Si C1 débloqué : nouvelle branche `chore/typescript-7-bump`.
- [ ] **C3**. Migrer `tsconfig.app.json` (et tout autre `tsconfig*.json` avec `baseUrl`) :
      retirer `baseUrl`, ajouter `"paths": {"*": ["./*"]}` — vérifier que `vite.config.ts`
      (alias `@/...`) reste cohérent avec les nouveaux `paths` (pas de divergence
      alias-de-fait vs config TS).
- [ ] **C4**. Purger `node_modules\.tmp` puis `npm run typecheck` (`tsc -b --force`) —
      piège cache incrémental connu, ne PAS conclure sur un cache chaud.
- [ ] **C5**. `npm run lint` (ESLint + `typescript-eslint`) — si C1 a confirmé le support,
      vérifier ici qu'aucune règle ne regresse sur la nouvelle AST TS7.
- [ ] **C6**. `npm run test` (vitest, esbuild — agnostique à la version TS en théorie,
      à confirmer) + `npm run build` (vite build via project references TS) + `npm run
      test:e2e`.
- [ ] **C7**. Vérifier `knip` (utilise l'API compilateur TS en interne) tourne sans erreur
      avec TS7.
- [ ] **C8**. Merge une fois tout vert. Entrée `.ai/thought_log.md`.

**Gate de clôture** : C1 vert (dépendance externe débloquée) ET C4-C7 tous verts —
c'est le lot le plus incertain des 3, pas de raccourci.

---

## Découvertes consignées (hors périmètre de ce plan)

- Aucune pour l'instant — ce plan est une préparation, pas une exécution. Toute
  découverte pendant l'exécution des lots A/B/C se documente dans CE fichier (section
  ajoutée) au fil de l'eau, pas silencieusement.
