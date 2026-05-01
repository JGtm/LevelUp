# PLAN — `<SessionBriefing>` (variant B retenu)

> Mock visuel : [`.ai/mockups/session-briefing.html`](mocks/session-briefing.html) (standalone, double-clic) — conservé après merge comme référence design.
> Branche : `feat/session-briefing` (à créer depuis `docs/charts-specs`).
> Scope : pages **Squad** (Synergies + Contributions via `SquadLayout`) ET **Timeseries** (mode solo).

## Décision design

Composant unique réutilisable solo + squad, structuré en 3 bandes verticales :

1. **Rail descriptif** (toujours) — `Ma session · 10 matchs · ⌀ 8:41/match · 1h49 · [bar Résultats avec libellés complets]`
2. **Verdict squad** (mode squad uniquement) — `Score d'équipe 44 Mauvais [C]` + N cards joueurs cliquables avec ▲/▼ vs avg score
3. **KPI grid 7 cards** (toujours) — Matchs, Durée, Frags, Morts, Assists, Précision, Vie. Trends ▲/▼ **vs moyenne d'équipe sur la session** (PAS vs all-time).

**Drill-down** : pré-calculé dans le payload (`kpis_by_xuid` map). Click → state local React → KPI grid recalcule. La moyenne d'équipe reste la référence donc les trends gardent leur sens. Pas d'endpoint séparé (cf. Annexe A).

**Libellés outcomes** : `Victoire / Défaite / Égalité / Abandon` tirés de `config/titles/halo_infinite/mappings/outcomes.toml` via `useFieldMappings()`.

**Multi-titres** : la bande verdict est gardée derrière `HasCapability(CapMatchmaking)`. Titres sans matchmaking (Forge-only par exemple) → mode solo forcé, dégradation gracieuse.

---

## Phase 1 — Backend Go

### 1.1 Étendre `SquadHeader` — `apps/go-api/internal/domain/squad_v2.go`

```go
type SquadHeader struct {
    SoloKPIs    *KPIStats             `json:"solo_kpis,omitempty"`
    AllTimeKPIs *KPIStats             `json:"all_time_kpis,omitempty"`        // CONSERVÉ pour rétrocompat — non utilisé par briefing
    SquadScore  *SquadScoreCard       `json:"squad_score,omitempty"`
    PlayerCards []PlayerScoreCard     `json:"player_cards,omitempty"`
    // NOUVEAU
    KPIsByXUID    map[string]*KPIStats `json:"kpis_by_xuid,omitempty"`         // drill-down — clé = xuid
    TeamAvgKPIs   *KPIStats            `json:"team_avg_kpis,omitempty"`        // référence pour trends ▲/▼
}

type PlayerScoreCard struct {
    XUID       string  `json:"xuid"`        // NOUVEAU — pour matcher avec KPIsByXUID au click
    Gamertag   string  `json:"gamertag"`
    // … (champs existants conservés)
}
```

### 1.2 Ajouter `ComputeTeamAvgKPIs` — `apps/go-api/internal/analysis/kpi_stats.go`

```go
// ComputeTeamAvgKPIs calcule la moyenne arithmétique des KPI individuels
// (un par joueur de l'escouade) pour servir de référence aux trends ▲/▼.
// Retourne nil si la map est vide. Outcomes mis à zéro (sans signification en moyenne).
func ComputeTeamAvgKPIs(perXuid map[string]*domain.KPIStats) *domain.KPIStats
```

Tests unitaires obligatoires (`kpi_stats_test.go`) :
- Map vide → nil
- 1 entrée → valeurs identiques (sauf outcomes à 0)
- 3 entrées → moyenne field-by-field correcte
- Avec entrée `nil` ignorée

### 1.3 Renseigner les nouveaux champs dans le service squad

Localisation : `apps/go-api/internal/service/squad_service_v2.go` (fonction qui produit `SquadHeader`).

Logique :
1. Charger les `match_ids` du scope (déjà fait pour `SoloKPIs`)
2. Pour chaque xuid (main + teammates) : `ComputeKPIStats(participantsByXUID[xuid])` sur ces match_ids — déjà chargé en mémoire pour `PlayerCards`
3. Construire `KPIsByXUID` à partir de cette boucle
4. `TeamAvgKPIs = ComputeTeamAvgKPIs(KPIsByXUID)`
5. Renseigner `XUID` sur chaque `PlayerScoreCard`

```go
slog.DebugContext(ctx, "session_briefing.compute_kpis_by_xuid",
    "team_size", len(KPIsByXUID), "match_count", len(matchIDs))
```

Aucune nouvelle requête DuckDB — réutilise les données déjà chargées.

**Capability gate** :
```go
if !title.HasCapability(title.CapMatchmaking) {
    // Pas de SquadScore / PlayerCards / KPIsByXUID — uniquement SoloKPIs + TeamAvgKPIs nil
    return SquadHeader{ SoloKPIs: solo }, nil
}
```

### 1.4 Tests service `squad_service_v2_test.go`

Mock `port.Repository` retourne participants par xuid. Vérifie :
- `KPIsByXUID` contient une entrée par joueur de l'escouade
- `TeamAvgKPIs` est cohérent avec moyenne arithmétique des entrées
- Cas dégradation : titre sans `CapMatchmaking` → squad fields nil, solo kpis présents

### 1.5 Étendre `TimeseriesPageResponse` — `apps/go-api/internal/domain/timeseries.go`

```go
type TimeseriesPageResponse struct {
    // … (champs existants conservés, dont SummaryTab.KpiCards qu'on NE supprime PAS)
    BriefingKPIs *KPIStats `json:"briefing_kpis,omitempty"` // NOUVEAU — alimente <SessionBriefing> en mode solo
}
```

Renseignement dans `service/timeseries_service.go` — réutilise `ComputeKPIStats(rows)` déjà appelé pour les autres calculs. Pas d'effort supplémentaire.

**Note** : on garde `SummaryTab.KpiCards` (ancien format pré-formaté) jusqu'à ce que la migration soit validée en prod. Suppression dans un PR ultérieur (feature flag `briefing_v1` éventuel — à discuter).

### 1.6 OpenAPI + tests handler

- `apps/go-api/api/openapi.yaml` — schémas `SquadHeader` (3 nouveaux champs) + `PlayerScoreCard.xuid` + `TimeseriesPageResponse.briefing_kpis`
- Test handler squad : payload golden contient les nouveaux champs
- Test handler timeseries : payload contient `briefing_kpis`
- Régénérer types front : `npm run generate-types`

---

## Phase 2 — Frontend Web

### 2.1 Créer le composant — `apps/web/src/features/_shared/SessionBriefing/`

**Pourquoi `_shared/` et pas `squad/`** : utilisé sur 2 features (Squad + Timeseries). Sortir le composant dans un module partagé évite les imports cross-feature.

Structure :

```
apps/web/src/features/_shared/SessionBriefing/
  SessionBriefing.tsx        # composant principal
  ResultsRail.tsx            # bande 1
  SquadVerdict.tsx           # bande 2 (squad only)
  KpiGrid.tsx                # bande 3
  DrillResetBar.tsx          # reset bar (drill-down only)
  trends.ts                  # helpers trend(value, ref, lowerIsBetter)
  i18n.ts                    # FR + EN
  SessionBriefing.test.tsx   # tests Vitest
  index.ts                   # barrel
```

Props :

```tsx
export interface SessionBriefingProps {
  /** KPI du joueur principal sur le scope filtré — REQUIS */
  kpis: KPIStats
  /** Données squad (verdict band + drill-down) — optionnel : sans = mode solo */
  squad?: {
    score: SquadScoreCard
    players: PlayerScoreCard[]
    kpisByXuid: Record<string, KPIStats>
    teamAvgKpis: KPIStats
    activeXuid: string
  }
}
```

Le mode est implicite : `squad === undefined` ⇒ solo. Pas de prop `mode` explicite.

### 2.2 i18n — `apps/web/src/features/_shared/SessionBriefing/i18n.ts`

Clés FR + EN :
- `briefing.session_label` : "Ma session" / "My session"
- `briefing.results_label` : "Résultats" / "Results"
- `briefing.results_legend` : `{n, plural, one {Victoire} other {Victoires}}` etc — utiliser `intl-messageformat` (déjà installé)
- `briefing.team_score` / `briefing.team_score_delta_bonus` / `briefing.team_score_base_only`
- `briefing.kpi_grid.{matches_played, total_duration, frags_per_match, deaths_per_match, assists_per_match, accuracy, lifespan}`
- `briefing.kpi_grid.subs.{per_min, per_match}` (formats des sub-fields)
- `briefing.drill_reset_label` : "Vue active : {gamertag}"
- `briefing.drill_reset_button` : "✕ revenir à mes stats"
- `briefing.trend_hint` : "▲/▼ vs moyenne d'équipe sur la session"

**Libellés outcomes** : utiliser `useFieldMappings().outcomes` (reflète `outcomes.toml`). NE PAS hardcoder Victoire/Défaite/etc.

### 2.3 Intégration `SquadLayout.tsx`

Position : remplacer / précéder le bandeau actuel par `<SessionBriefing>` après le filtre omnibar et avant le `<Outlet />`.

```tsx
import { SessionBriefing } from '@/features/_shared/SessionBriefing'

const header = teammatesData?.header
const squadProps = header?.squad_score && header?.player_cards
  ? {
      score: header.squad_score,
      players: header.player_cards,
      kpisByXuid: header.kpis_by_xuid ?? {},
      teamAvgKpis: header.team_avg_kpis!,
      activeXuid: activePlayerXuid,
    }
  : undefined

{header?.solo_kpis && (
  <SessionBriefing kpis={header.solo_kpis} squad={squadProps} />
)}
```

### 2.4 Intégration `TimeseriesPage.tsx`

Position : tout en haut de la page, avant le tab Résumé.

```tsx
{data?.briefing_kpis && (
  <SessionBriefing kpis={data.briefing_kpis} />
)}
```

Mode solo uniquement (pas de prop `squad`). Conserve `SummaryTab.KpiCards` actuels en dessous le temps de la migration.

### 2.5 Tests Vitest — `SessionBriefing.test.tsx`

Cas couverts :
- Solo (`squad` absent) : pas de bande verdict, pas de trends ▲/▼ visibles
- Squad : bande verdict présente avec N+1 cards, trends visibles
- Drill-down : click sur card "Chocoboflor" → grid affiche stats Chocoboflor + reset bar visible
- Reset : click sur ✕ → retour à activeXuid
- Trend kills (higher_is_better) : `kills_per_game = 10` vs `team_avg.kills_per_game = 8` → trend = "above"
- Trend deaths (lower_is_better) : `deaths_per_game = 12` vs `team_avg.deaths_per_game = 10` → trend = "below"
- Singulier/pluriel outcomes : `wins = 1 → "1 Victoire"`, `wins = 3 → "3 Victoires"`
- Manquant `kpis_by_xuid[xuid]` au click → fallback sur solo_kpis (pas de crash)

### 2.6 Règles couleurs / tokens

Aucun hex, aucune classe Tailwind couleur dans le composant. Tokens uniquement :
- `tokenCssVar('perf-tier-{1..5}')` pour les scores
- `tokenCssVar('outcome-{win|loss|draw|dnf}')` pour la bar Résultats
- `tokenCssVar('divergent-{pos|neutral|neg}')` pour les trends ▲/▼

---

## Phase 3 — Validation & livraison

### Checks obligatoires (delivery-checklist)

```bash
# Backend
cd apps/go-api && go test ./... && go vet ./...

# Frontend
cd apps/web && npm run typecheck && npm run lint && npm run test
```

### Validation visuelle

1. Lancer Go API + Vite dev
2. Aller sur `/players/{slug}/squad/synergies` en solo → briefing solo (rail + grid 7 cards, pas de verdict)
3. Sélectionner 1-2 coéquipiers + Analyser → verdict apparaît, drill-down clickable
4. Cliquer sur un coéquipier → grid recalcule, reset bar apparaît
5. Aller sur `/players/{slug}/stats/timeseries` → briefing solo en haut de page
6. Trends doivent montrer ▲ pour kills > team avg, ▼ pour deaths > team avg

### Thought log

Entrée à ajouter dans `.ai/thought_log.md` au moment du commit :
- Date : (au commit)
- Titre : "SessionBriefing — fusion KPI bar + Squad verdict (variant B + Timeseries)"
- Statut : Complété
- Décision : drill-down pré-calculé via `kpis_by_xuid` (pas d'endpoint séparé), trends ▲/▼ vs moyenne d'équipe (pas all-time), mode dégradé pour titres sans `CapMatchmaking`
- Résultats : tests verts, valid visu OK sur Squad + Timeseries
- Prochaine étape : monitorer adoption ; planifier dépréciation `SummaryTab.KpiCards` (ancien format Timeseries) dans un PR ultérieur

---

## Estimation

| Phase | Effort | Note |
|---|---|---|
| 1.1–1.4 backend squad | ~1h30 | Calcul léger, données déjà en mémoire |
| 1.5 backend timeseries | ~30min | Réutilise `ComputeKPIStats` existant |
| 1.6 OpenAPI + handler tests | ~30min | |
| 2.1–2.2 composant + i18n | ~2h30 | 4 sous-composants, ~400L total |
| 2.3 intégration Squad | ~30min | Placement uniquement |
| 2.4 intégration Timeseries | ~20min | Placement uniquement |
| 2.5 tests Vitest | ~1h | 8 cas |
| 3 validation visu + thought_log + commit | ~30min | |
| **Total** | **~7h** | 1 PR, 3-4 commits logiques |

## Risques / points d'attention

| Risque | Mitigation |
|---|---|
| `kpis_by_xuid` payload pèse lourd avec 4 joueurs | KPIStats fait ~10 fields × 4 joueurs = 40 floats ≈ <1KB JSON. Négligeable. |
| Trend deaths inversé oublié | Test Vitest dédié + commentaire dans le code |
| Drill-down sans affordance visible | Cursor-pointer + hover state + bordure forte sur card "viewée" + reset bar explicite |
| Singulier/pluriel outcomes en EN/FR | Utiliser `intl-messageformat` (déjà installé) |
| Mock HTML qui drift par rapport au composant React | Conservé en `.ai/mocks/` comme référence design avec note "code = SessionBriefing.tsx" |
| Coexistence `briefing_kpis` + `SummaryTab.KpiCards` sur Timeseries | OK pour PR initial. Dépréciation des KpiCards dans un PR de suivi (ne pas casser des consommateurs externes éventuels) |

## Architecture (vérification arch-rules + plan-review)

- ✅ `analysis/kpi_stats.go` — calcul pur (`ComputeTeamAvgKPIs`) avec tests unitaires
- ✅ `service/squad_service_v2.go` + `service/timeseries_service.go` — orchestration
- ✅ `domain/squad_v2.go` + `domain/timeseries.go` — types partagés étendus
- ✅ Pas d'accès DB dans handler/composant
- ✅ Multi-titres : libellés outcomes via `outcomes.toml`, gate sur `HasCapability(CapMatchmaking)`
- ✅ Frontend : tokens sémantiques uniquement, i18n FR + EN, pas de hex
- ✅ Logging `slog.DebugContext` sur calculs KPI per-xuid
- ✅ Tests à toutes les couches (analysis purs, service avec mock repo, handler httptest, composant Vitest)
- ✅ Une seule branche `feat/session-briefing`, 3-4 commits logiques

---

## Annexe A — Détail décision drill-down (pré-calcul vs lazy)

| Critère | Pré-calcul (retenu) | Lazy fetch (rejeté) |
|---|---|---|
| Coût payload | ~320 octets / xuid additionnel | Identique au final mais en N round-trips |
| Code Go | 5 lignes (boucle déjà présente) | Nouveau handler + service + port + repo + tests |
| Architecture | Réutilise `participantsByXUID` déjà chargés | Nouvelle requête DuckDB par click (waste — données déjà en RAM) |
| Surface API | 0 nouvel endpoint | 1 endpoint à versionner, OpenAPI, tester |
| UX click | Instantané (state local React) | Latence + spinner par click |
| Cohérence | `SquadHeader` regroupe déjà `solo_kpis`, `all_time_kpis`, `squad_score`, `player_cards` | Casse cette cohérence |

→ Pré-calcul gagne sur les 6 axes. Le seul cas où lazy serait justifié est si la majorité des users ne clique jamais — improbable vu que c'est l'intérêt principal de la bande verdict.
