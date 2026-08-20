# Plan — Profil de combat : 5 graphes (page Explorer, mode Joueur)

> Statut : Plan validé, implémentation non démarrée — 2026-05-31.
> Branche cible recommandée : `feat/explorer-combat-profile-charts`.
> NB : distinct de `PLAN_EXPLORER_COMBAT_PROFILE.md` (qui traite identité/cosmétiques +
> médailles du même encart). Ce doc-ci traite **les graphes** des 20 derniers matchs.

## Context

La page **Explorer** (`/players/$playerSlug/explorer`, mode « Joueur ») permet de rechercher
un autre joueur (`target_gamertag`) et affiche déjà : bandeau identité, stats carrière,
sample stats, médailles, CSR par saison, briefing de rencontre, tableaux des matchs en
commun (allié / ennemi) et heatmap d'activité. Il **manque une lecture de la forme récente
du joueur recherché lui-même**.

On ajoute une section **« Profil de combat »** sur **les 20 derniers matchs PvP du joueur
cible** (ses propres parties, pas les matchs en commun), composée de **5 graphes** (pas de
tableau). Toutes les données nécessaires existent déjà en base — aucune nouvelle ingestion.

### Décisions utilisateur (confirmées)
- **Périmètre** : les 20 derniers matchs du **joueur cible lui-même**.
- **Types** : **PvP uniquement** (exclure Firefight/PvE — le placement n'y a pas de sens ;
  une éventuelle vue PvE dédiée est reportée à une phase ultérieure).
- **Affichage** : **graphes seulement**, pas de tableau détaillé.
- **+ Ajout** : un **donut** de la répartition des modes de jeu sur ces 20 matchs.

### Les 5 graphes
- **Rangée 1** — G1 : courbe **FDA** (ratio KDA) + barres groupées **Frags / Morts / Assistances** ;
  G2 : barres **empilées dégâts infligés + subis**.
- **Rangée 2** — G3 : barres **score** par match + courbe **placement (rang)** ;
  G4 : barres **folie meurtrière max** + **frags parfaits**.
- **Rangée 3** — G5 : **donut répartition des modes de jeu**.

## Disponibilité des données (vérifiée)

Tout est dans `shared_matches_v2.duckdb`, lisible par l'`ExplorerRepo` (tables non préfixées :
`match_participants`, `match_registry`, `medals_earned`).

| Donnée | Source | Statut |
|---|---|---|
| kills / deaths / assists / kda | `match_participants` | OK |
| damage_dealt / damage_taken | `match_participants` | OK |
| score | `match_participants.personal_score` (= colonne « Score » du scoreboard) | OK |
| placement | `match_participants.rank` (NULL si DNF/non classé) | OK |
| max_killing_spree | `match_participants.max_killing_spree` | OK |
| **perfect_kills** | agrégé via `medals_earned` `medal_name_id = 1512363953` | OK (LEFT JOIN) |
| outcome / mode / carte / start_time | `match_participants.outcome`, `match_registry.pair_name` / `map_name` / `start_time(_utc)` | OK |

Outcome : `1=tie, 2=win, 3=loss, 4=DNF`. Timezone : toujours
`COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC')`.

> Note : `chartLabels.ts` ne contient que 7 clés et il n'existe aucun échafaudage « combat
> profile charts » préexistant — relire le contenu réel des fichiers cités avant d'éditer.

## Stockage / cache — calcul à la demande (décision)

Le profil de combat est une **donnée dérivée, non persistée** (comme `sample_stats` /
`common_matches`). Il est **calculé côté serveur à chaque player-query** et transporté dans
la réponse `ExplorerPlayerQueryResponse` existante. Côté client, `useExplorerPlayer` met déjà
en cache la réponse **5 min** (`staleTime`, TanStack Query) -> pas de re-fetch pendant la
navigation. **Aucune nouvelle table, aucun nouvel endpoint, aucun cache serveur dédié.**
Requête légère (LIMIT 60 + 1 join + sous-requête médailles, filtrée par xuid). Si la pression
de charge augmente plus tard, un cache in-memory par xuid (TTL court) reste ajoutable sans
changer le contrat d'API.

## Backend (Go) — `domain -> port -> repo -> service`

Le handler `internal/api/handlers/explorer.go::QueryPlayer` et le wiring
`registry_pages.go` restent **inchangés** (la réponse `ExplorerPlayerQueryResponse` transporte
le nouveau champ). Le service du player-query est `ExplorerService.GetCommonMatches`
(`internal/service/explorer_service.go`) -> `buildTargetProfile(ctx, targetXUID, ...)`.

### 1. `internal/domain/explorer.go`
Nouveau type ligne + champ dans `ExplorerTargetProfile` :
```go
type ExplorerTargetRecentMatch struct {
    MatchID         string    `json:"match_id"`
    StartTime       time.Time `json:"start_time"`
    MapUI           string    `json:"map_ui"`
    ModeUI          string    `json:"mode_ui"`
    Outcome         int       `json:"outcome"`
    Rank            *int      `json:"rank,omitempty"`     // nil si DNF/non classé
    Kills           int       `json:"kills"`
    Deaths          int       `json:"deaths"`
    Assists         int       `json:"assists"`
    KDA             float64   `json:"kda"`                // ratio FDA pré-calculé (mp.kda)
    Score           int       `json:"score"`             // personal_score
    DamageDealt     int       `json:"damage_dealt"`
    DamageTaken     int       `json:"damage_taken"`
    MaxKillingSpree int       `json:"max_killing_spree"`
    PerfectKills    int       `json:"perfect_kills"`
}
// dans ExplorerTargetProfile :
CombatProfile []ExplorerTargetRecentMatch `json:"combat_profile,omitempty"`
```
Pas de `outcome_label` côté Go : la résolution issue->couleur/label se fait au front (cohérent
avec `TimeseriesKdaBars` qui mappe l'`outcome` int).

### 2. `internal/port/repository.go`
Ajouter à l'interface `ExplorerRepository` + à `noopExplorerRepo` :
```go
GetTargetRecentMatches(ctx context.Context, xuid string, limit int) ([]domain.ExplorerTargetRecentMatch, error)
```

### 3. `internal/platform/duckdb/explorer_repo.go`
Const SQL au niveau package (comme `Q19CommonMatches`) pour garder la méthode < 80 lignes.
On charge un **buffer** (ex. `LIMIT 60`) puis on filtre PvP en Go (cf. § PvP) avant de capper
à 20.
```sql
WITH recent AS (
  SELECT mp.match_id,
         COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') AS start_time,
         COALESCE(r.map_name,'')  AS map_ui,
         COALESCE(r.pair_name,'') AS mode_ui,
         COALESCE(mp.outcome,0)   AS outcome,
         mp.rank                  AS rank,            -- scan sql.NullInt64
         COALESCE(mp.kills,0) AS kills, COALESCE(mp.deaths,0) AS deaths,
         COALESCE(mp.assists,0) AS assists, COALESCE(mp.kda,0.0) AS kda,
         COALESCE(mp.personal_score,0) AS score,
         COALESCE(mp.damage_dealt,0.0) AS damage_dealt,   -- scan float64 -> int
         COALESCE(mp.damage_taken,0.0) AS damage_taken,
         COALESCE(mp.max_killing_spree,0) AS max_killing_spree
  FROM match_participants mp
  JOIN match_registry r ON r.match_id = mp.match_id
  WHERE mp.xuid = ?
  ORDER BY start_time DESC
  LIMIT ?
),
perfect AS (
  SELECT match_id, COALESCE(SUM(count),0) AS perfect_kills
  FROM medals_earned WHERE xuid = ? AND medal_name_id = 1512363953
  GROUP BY match_id
)
SELECT rec.*, COALESCE(p.perfect_kills,0) AS perfect_kills
FROM recent rec LEFT JOIN perfect p ON p.match_id = rec.match_id
ORDER BY rec.start_time DESC
```
Méthode Go (~35 l) : `SharedReadDB().Get(ctx)` + `defer release()`, `QueryContext`, boucle
déléguant le `Scan` à un helper `scanRecentMatch(rows)` (~20 l, `rank` en `sql.NullInt64`,
`damage_*` float64->int). Garde `if strings.TrimSpace(xuid)=="" || limit<=0 { return nil,nil }`.
Médaille `1512363953` -> constante nommée (réutiliser celle existante si présente, sinon
`const medalPerfectKillID = 1512363953`).

### 4. Filtre PvP (réutiliser l'existant)
Réutiliser la classification PvP/PvE déjà appliquée par le matches-query (filtre
`experience_types`). Si elle est côté Go (probable : `internal/analysis/mode_category.go::InferModeCategoryFromPairName`),
charger le buffer (`LIMIT 60`) puis **classer chaque ligne en Go et ne garder que les PvP,
cappées à 20** — cela réutilise le classifieur canonique et évite un prédicat SQL fragile.
Documenter le buffer (un joueur majoritairement PvE peut donner < 20 matchs PvP). *Localiser
la fonction exacte avant d'implémenter.*

### 5. `internal/service/explorer_service.go`
Dans `buildTargetProfile`, ajouter une goroutine errgroup (sur `gctx`, best-effort comme
`computeTargetSampleStats`) appelant un helper court :
```go
func (s *ExplorerService) computeTargetCombatProfile(ctx, targetXUID string) []domain.ExplorerTargetRecentMatch {
    rows, err := s.repo.GetTargetRecentMatches(ctx, targetXUID, 60) // buffer; filtre PvP -> cap 20
    if err != nil { slog.WarnContext(ctx, "explorer_target_combat_profile_failed", "xuid", targetXUID, "err", err); return nil }
    return filterPvPCap(rows, 20)
}
```
Affecter le résultat à `CombatProfile` dans le `ExplorerTargetProfile` retourné.

## Frontend (React / TS)

### 6. `apps/web/src/lib/api/types.ts`
Ajouter `interface ExplorerTargetRecentMatch { ... }` (miroir du DTO) + champ optionnel
`combat_profile?: ExplorerTargetRecentMatch[] | null` dans `ExplorerTargetProfile`.
**Aucune** nouvelle query (`useExplorerPlayer` transporte déjà `target_profile`) ni nouvelle
clé dans `keys.ts`.

### 7. Composants (sous `apps/web/src/features/explorer/`)
- **`ExplorerCombatProfile.tsx`** (conteneur) : props `{ matches, locale, t }`. `if (!matches?.length) return null`.
  Trie une copie **chronologiquement croissante** pour les charts temporels (comme
  `SessionDamageComposite`). Calcule en `useMemo` les séries de chaque graphe + le group-by
  mode pour le donut. Rend un titre i18n + 3 rangées `grid grid-cols-1 sm:grid-cols-2 gap-4`
  (R1 : G1|G2, R2 : G3|G4, R3 : G5 — demi-largeur/centré).
- **`combatChartOptions.ts`** : 2 *pure builders* exportés (testables vitest), modelés sur
  `buildKdaBarsOption` (`features/timeseries/TimeseriesKdaBars.tsx`) :
  - `buildCombatFdaOption` (G1) et `buildCombatScoreOption` (G3).
- **`CombatFdaChart.tsx`** (G1) et **`CombatScorePlacementChart.tsx`** (G3) : fins wrappers
  `ChartCard` consommant les builders ci-dessus.
- G2/G4/G5 : rendus **inline dans le conteneur** via les wrappers génériques existants.

### 8. Insertion — `ExplorerPage.playerMode.tsx`
Après `ExplorerTargetProfileCard` (l. 150-155), **avant** `ExplorerEncounterBriefing` (l. 157) :
```tsx
{playerQuery.data.target_profile?.combat_profile?.length ? (
  <ExplorerCombatProfile
    matches={playerQuery.data.target_profile.combat_profile}
    locale={locale}
    t={t}
  />
) : null}
```
(+ import). `t` et `locale` sont déjà des props de `ExplorerPlayerMode`.

### 9. i18n
- **Libellés de stats** (kills/deaths/assists/damage_dealt/damage_taken/kda/personal_score) :
  réutiliser **`useFieldMappings()`** (`fields?.kills?.label`, …), comme `SessionDamageComposite`
  et `TimeseriesKdaBars`. Vérifier l'existence des labels `max_killing_spree` / `perfect_kills`
  (sinon les ajouter côté field mappings).
- **Strings de section** (titre « Profil de combat », sous-titre, titres des 5 cartes, axes
  « FDA »/« Placement », titre donut, état vide, note placement) : ajouter au manifest
  `apps/web/src/lib/i18n/manifests/explorer.toml` (clés `explorer.combat.*`, **fr ET en**),
  puis régénérer : `node apps/web/scripts/build_i18n_manifests.mjs`. Accès via le `t` existant
  (`t('explorer.combat.title')`, etc.). Outcome labels : réutiliser `explorer.matches.outcome_*`.
- **Modes du donut** : pour des libellés FR, traduire `pair_name` via le mécanisme mode
  existant (`mode_name_tr`, cf. réf. traductions assets) — réutiliser la même résolution que
  les tableaux de matchs ; sinon afficher `mode_ui` brut en première version.

## Spécification par graphe

| # | Graphe | Wrapper / builder | Forme données | Tokens | Pièges |
|---|---|---|---|---|---|
| G1 | FDA + Frags/Morts/Assists | **builder** `buildCombatFdaOption` (modèle `buildKdaBarsOption`) | `ChartSeries<{x,kills,deaths,assists,kda,outcome}>` | kills->`chart-series-1`, deaths->`outcome-loss`, assists->`chart-series-3`, ligne FDA->`perf-tier-2` | 3 barres **groupées** (pas de `stack`) + ligne sur `yAxisIndex:1` ; **double axe Y** ; légende 4 entrées |
| G2 | Dégâts infligés + subis | **`BarStackedChart`** générique (comme `SessionDamageComposite`) | `ChartSeries<ChartPointStacked>` `{damage_dealt,damage_taken}` | `divergent-pos` / `divergent-neg` | `orientation="horizontal"`, `tooltipHideZero` ; vieux matchs sans dégâts = barres courtes (normal) |
| G3 | Score + placement | **builder** `buildCombatScoreOption` (modèle `buildKdaBarsOption`) | `ChartSeries<{x,score,rank}>` | score->`chart-series-1`, placement->`perf-tier-2` | `yAxis[1]={position:'right',inverse:true,min:1}` (rang 1 en haut) ; `rank===null`->point `null`, `connectNulls:false` |
| G4 | Folie max + frags parfaits | **`BarGroupedChart`** générique | `ChartSeries<ChartPointStacked>` `{max_killing_spree,perfect_kills}` | `perf-tier-1` / `chart-series-4` | `perfect_kills` souvent 0 -> barre absente (normal) |
| G5 | Donut répartition modes | **`DonutChart`** générique | `ChartSeries<ChartPointDonut>` `{name:mode, value:count}` | `getSeriesColors(n)` / tokens séries | grouper les 20 lignes par mode au front ; `innerRadius` non nul (donut, style maison) ; `showPercent` |

Label d'axe catégoriel (G2/G4) : helper court `combatMatchLabel(i, map)` local (ou réutiliser
`sessionMatchAxisLabel` de `features/session-detail/_shared`). G1/G3 : `start_time` formaté
`DD/MM` (comme `buildKdaBarsOption`). Couleurs **uniquement via tokens** (jamais de hex ni de
classe Tailwind couleur).

## Tests

- **Go repo** (`//go:build integration`, DuckDB `:memory:` + DDL inline, modèle
  `TestCompareRepo_GetLocalStats`) : dataset hétérogène ~22 matchs (PvP + au moins 1
  Firefight pour valider le filtre PvP), 1 WIN avec perfect-kill medal + 1 sans (valide
  `LEFT JOIN`/COALESCE 0), 1 DNF `rank` NULL (valide `*int` nil), start_times distincts.
  Asserts : ordre `StartTime` DESC, `LIMIT`/cap 20, PvE exclu, `PerfectKills` 2/0, `Rank` nil,
  `DamageDealt` casté int.
- **Go service** (mock-port, sans tag) : étendre `mockExplorerRepo` avec `GetTargetRecentMatches`
  -> assertions : `combat_profile` peuplé ; erreur repo -> `combat_profile` nil (best-effort) ;
  filtre PvP appliqué.
- **Go handler** (`httptest`) : `target_profile.combat_profile` présent dans le JSON.
- **Frontend vitest** (hors sandbox `dangerouslyDisableSandbox`, `vi.mock('echarts-for-react')`,
  `renderWithProviders`) : builders purs `buildCombatFdaOption` (3 `bar` sans `stack` + 1 `line`
  `yAxisIndex:1`, `yAxis.length===2`) et `buildCombatScoreOption` (`yAxis[1].inverse===true`,
  `min===1`, `rank:null`->`null`) ; conteneur (`null` si vide ; rend 5 charts si données) ;
  group-by mode du donut.
- **Smoke** : `cd apps/go-api && go build ./... && go vet ./... && go test -tags integration ./...`
  (toolchain CGO requise : `CGO_ENABLED=1` + gcc msys64) ; `cd apps/web && npm run typecheck && npm run lint && npx vitest run`.

## Edge cases / risques
- **< 20 matchs PvP** (joueur lointain / surtout PvE) : charts s'adaptent ; buffer `LIMIT 60`
  limite mais n'élimine pas le cas — acceptable.
- **DNF / sans rank** : `rank` NULL -> trou dans la courbe G3 (jamais 0, fausserait l'axe inversé).
- **perfect_kills = 0** : barre G4 absente (normal).
- **Team games** : `rank` = placement d'équipe (peu signifiant individuellement) -> label
  neutre « Placement », pas de sur-interprétation.
- **Cible introuvable** : `ResolveXUIDByGamertag` échoue en amont (comportement actuel inchangé) ;
  xuid résolu mais 0 match -> `combat_profile` vide -> section masquée.
- **Accès** : player-query est public-read (pas de contrôle d'ownership sur `target_gamertag`) —
  cohérent avec l'exposition déjà existante (career/sample/medals du joueur cible).

## Découpage en commits (1 branche dédiée, N commits)
> Travail à mener sur une branche dédiée à la feature (recommandé : `feat/explorer-combat-profile-charts`),
> pas sur une branche d'un autre sujet. Demander l'autorisation avant chaque commit. Entrée
> `.ai/thought_log.md` obligatoire avant de rendre la main.

1. **backend(domain+port)** : type + champ `CombatProfile` + méthode interface + noop (compile).
2. **backend(repo+pvp)** : `GetTargetRecentMatches` + const SQL + helper scan + filtre PvP + test `:memory:`.
3. **backend(service)** : `computeTargetCombatProfile` + goroutine + tests mock-port + handler httptest.
4. **frontend(builders+charts)** : `types.ts`, `combatChartOptions.ts`, `CombatFdaChart`, `CombatScorePlacementChart` + tests builders.
5. **frontend(conteneur+i18n)** : `ExplorerCombatProfile` (5 charts dont donut), `explorer.toml` + regen, insertion dans `playerMode`, test conteneur.

## Vérification end-to-end
1. `go test -tags integration ./internal/platform/duckdb/...` + `go test ./internal/service/... ./internal/api/...`.
2. `npx vitest run` (builders + conteneur) + `npm run typecheck && npm run lint`.
3. App : lancer le front, page Explorer -> mode Joueur -> rechercher un joueur connu
   (ex. Madina97294) -> vérifier les 5 graphes (FDA+barres, dégâts empilés, score+placement
   axe inversé, folie+parfaits, donut modes), l'exclusion PvE, l'état vide si aucun match.
