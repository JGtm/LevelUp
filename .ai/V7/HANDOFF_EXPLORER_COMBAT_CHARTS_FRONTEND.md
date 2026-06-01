# HANDOFF — Profil de combat Explorer : FRONTEND (5 graphes)

> Date : 2026-05-31
> Branche : `fix/explorer-combat-profile-identity`
> Plan source : `.ai/PLAN_explorer_combat_profile_charts.md` (lire en entier — spec par graphe + tests)
> **Backend : LIVRÉ + testé + committé** (`d9afff3a`). L'API sert déjà `target_profile.combat_profile`.
> **Frontend : à faire sur la machine de dev** (cet env agent ne peut pas lire les composants de façon fiable ni compiler/tester — node_modules incomplet, reads intermittents). Décision utilisateur : hand-off.

## Ce que le backend fournit déjà (contrat prêt à consommer)

`POST /api/v1/players/{slug}/pages/explorer/player-query` → `target_profile.combat_profile` :
un tableau (≤20) des derniers matchs **PvP** (Firefight exclu) du joueur cible, du plus récent au plus ancien.

Shape Go (`domain.ExplorerTargetRecentMatch`, JSON snake_case) :
```
match_id          string
start_time        string (RFC3339)
map_ui            string
mode_ui           string        // pair_name (ex "Slayer") — à traduire FR via mode_name_tr si voulu
outcome           int           // 1=tie, 2=win, 3=loss, 4=DNF
rank              int|null      // null si DNF/non classé → trou dans la courbe (ne PAS tracer 0)
kills, deaths, assists  int
kda               float64       // ratio FDA pré-calculé
score             int           // personal_score
damage_dealt      int
damage_taken      int
max_killing_spree int
perfect_kills     int
```
Vide/absent si la cible n'a aucun match PvP → la section graphes doit se masquer (`if (!matches?.length) return null`).

## Vérif rapide du contrat (sur machine de dev, serveur up)

```bash
curl -s -X POST http://localhost:8000/api/v1/players/JGtm/pages/explorer/player-query \
  -H 'Content-Type: application/json' -d '{"target_gamertag":"Madina97294"}' \
  | jq '.target_profile.combat_profile[0]'
```

## Fichiers à créer / modifier (cf. plan §6-9)

1. **`apps/web/src/lib/api/types.ts`** — ajouter `interface ExplorerTargetRecentMatch { ... }` (miroir exact du shape ci-dessus) + champ `combat_profile?: ExplorerTargetRecentMatch[] | null` dans `ExplorerTargetProfile`. **Aucune** nouvelle query (transporté par `useExplorerPlayer` existant).

2. **`apps/web/src/features/explorer/combatChartOptions.ts`** (NOUVEAU) — 2 builders purs exportés (testables vitest), **modelés sur `buildKdaBarsOption` de `features/timeseries/TimeseriesKdaBars.tsx`** (le lire d'abord) :
   - `buildCombatFdaOption(matches, ...)` (G1) : 3 barres **groupées** kills/deaths/assists (PAS de `stack`) + 1 ligne FDA sur `yAxisIndex:1` → **double axe Y** (`yAxis.length===2`).
   - `buildCombatScoreOption(matches, ...)` (G3) : barres score + courbe placement sur `yAxis[1]={position:'right', inverse:true, min:1}` (rang 1 en haut) ; `rank===null` → point `null`, `connectNulls:false`.

3. **`CombatFdaChart.tsx`** (G1) + **`CombatScorePlacementChart.tsx`** (G3) — fins wrappers `ChartCard` consommant les builders.

4. **`ExplorerCombatProfile.tsx`** (NOUVEAU, conteneur) — props `{ matches, locale, t }`. `if (!matches?.length) return null`. Trie une **copie** chrono croissante pour les graphes temporels. `useMemo` pour les séries + group-by mode (donut). 3 rangées `grid grid-cols-1 sm:grid-cols-2 gap-4` : R1 G1|G2, R2 G3|G4, R3 G5 (centré). G2/G4/G5 rendus inline via wrappers génériques :
   - **G2** dégâts infligés+subis : wrapper barres empilées existant (lire `components/charts/` — `StackedBarChart` ou équivalent ; cf. `SessionDamageComposite` pour le modèle), tokens `divergent-pos`/`divergent-neg`.
   - **G4** folie max + frags parfaits : wrapper barres groupées (`BarChart`/`BarGroupedChart`), tokens `perf-tier-1`/`chart-series-4`. `perfect_kills` souvent 0 → barre absente (normal).
   - **G5** donut modes : `DonutChart` (vérifier qu'il existe dans `components/charts/` ; sinon barres 100% empilées en repli), `getSeriesColors(n)`. Grouper les ≤20 lignes par `mode_ui` au front.

5. **i18n** — labels stats (kills/deaths/assists/damage_*/kda/score) : réutiliser `useFieldMappings()` (comme `TimeseriesKdaBars`/`SessionDamageComposite`). Strings de section (titre « Profil de combat », axes « FDA »/« Placement », titre donut, état vide) : ajouter au manifest `apps/web/src/lib/i18n/manifests/explorer.toml` (clés `explorer.combat.*`, **fr ET en**) puis régénérer (`node apps/web/scripts/build_i18n_manifests.mjs` ou le script du repo). Outcome→couleur : mapper l'`outcome` int au front (tokens `outcome-win/loss/draw/dnf`).

6. **Insertion — `ExplorerPage.playerMode.tsx`** : après `<ExplorerTargetProfileCard>`, avant `<ExplorerEncounterBriefing>` :
```tsx
{playerQuery.data.target_profile?.combat_profile?.length ? (
  <ExplorerCombatProfile
    matches={playerQuery.data.target_profile.combat_profile}
    locale={locale}
    t={t}
  />
) : null}
```

## Règles projet à respecter (sinon pre-push lefthook bloque)

- **Couleurs** : jamais de hex ni de classe Tailwind couleur dans `features/`/`components/` → uniquement tokens via `tokenCssVar`/`resolveToken`/`getSeriesColors` (lint-no-hardcoded-colors).
- **i18n** : pas de libellé FR/EN en dur → manifest (lint-no-hardcoded-fields).
- **Pas d'import cross-feature** non déclaré (lint-cross-feature-imports). Importer `buildKdaBarsOption` depuis `features/timeseries` = cross-feature → soit recopier le pattern dans `combatChartOptions.ts` (recommandé), soit déclarer l'exception.
- Fichiers < 500 L, fonctions ≤ 80 L.

## Tests (cf. plan §Tests)

- **vitest builders** (`combatChartOptions.test.ts`) : `buildCombatFdaOption` → 3 séries `bar` sans `stack` + 1 `line` `yAxisIndex:1`, `yAxis.length===2` ; `buildCombatScoreOption` → `yAxis[1].inverse===true`, `min===1`, `rank:null`→`null`.
- **vitest conteneur** (`ExplorerCombatProfile.test.tsx`) : `null` si `matches` vide ; rend 5 charts si données ; group-by mode du donut correct. `vi.mock('echarts-for-react')`, `renderWithProviders`.
- Lancer hors sandbox : `npx vitest run src/features/explorer/` + `npm run typecheck` + `npm run lint`.

## Étapes recommandées (machine de dev)

1. `git checkout fix/explorer-combat-profile-identity && git pull` (récupérer le backend `d9afff3a`).
2. Lire **réellement** : `TimeseriesKdaBars.tsx`, `components/charts/{index,types,ChartCard}.ts(x)` + le(s) wrapper(s) empilé/donut existants, `ExplorerTargetProfileCard.tsx` (props `t`/`locale`), `ExplorerPage.playerMode.tsx`.
3. Implémenter dans l'ordre du plan (types → builders+charts → conteneur+i18n → insertion).
4. `typecheck` + `lint` + `vitest` verts, puis vérif visuelle (Explorer → mode Joueur → joueur connu → 5 graphes, exclusion PvE, état vide).
5. `.ai/thought_log.md` (règle obligatoire) + commit.

## Pièges connus (de cette session)

- `match_registry.is_firefight` est la source de vérité PvE (backend filtre déjà ; le front n'a QUE du PvP).
- `rank` peut être `null` (DNF) — ne jamais le tracer comme 0 (fausse l'axe inversé du G3).
- Le donut/empilé : **vérifier le nom réel** du wrapper dans `components/charts/` avant d'importer (le plan suppose `DonutChart`/`BarStackedChart` — confirmer).
