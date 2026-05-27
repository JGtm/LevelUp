# Plan — Coaching complet : profil, patterns et leviers

**Date** : 2026-05-23
**Statut** : Brouillon — non démarré
**Branche cible** : `feat/pattern-engine` (à créer depuis `main`)
**Révision** : v3 — 2026-05-23 (restructuration 4 phases après audit end-to-end)

---

## Problème

Le système de coaching a trois niveaux de problèmes distincts :

| Niveau | Exemples | Impact joueur |
|--------|----------|---------------|
| **Cassé** | Streaks perf-based : médiane KDA calculée mais jamais injectée dans `EvaluateInput.Thresholds` | Joueur ne voit jamais de streak liée à sa performance |
| **Orphelin** | `PlayerProfile` sections A1/A2/B/C : radar, style, composantes LUSR, leviers, suggestions — backend complet, 0 hook frontend | Mois de travail backend invisibles |
| **Manquant** | Patterns contextuels (mode/map), comportementaux (tilt, fatigue), leviers calibrés sur N matchs | Use cases coaching de fond inexistants |

Construire le Pattern Engine sans surfacer les orphelins créerait une troisième couche de leviers qui chevauche `ProfileService.C` déjà existant — deux implémentations pour le même problème, toujours rien pour l'utilisateur.

---

## Objectif

Livrer un coaching cohérent end-to-end en 4 phases avec une valeur métier réelle à chaque livraison :

| Phase | Livrable joueur | Durée estimée |
|-------|----------------|---------------|
| **0 — Réparer + surfacer** | Voit enfin son radar, son style, ses axes LUSR prioritaires | ~3h |
| **1 — Patterns contextuels** | Comprend où il est fort/faible (mode, map, squad) | ~6h |
| **2 — Patterns comportementaux** | Comprend quand il dérape (tilt, fatigue, engagement) | ~5h |
| **3 — Leviers + coach** | Reçoit un objectif chiffré, atteignable, avec horizon | ~5h |

---

## Architecture en couches

```
┌─────────────────────────────────────────────────────────┐
│  ALERTES COACH (post-sync)                              │
│  10 types · notifications · dédup 24h                   │
│  → phase 0 : fix streaks perf · phase 3 : +4 types      │
├─────────────────────────────────────────────────────────┤
│  LEVIERS CALIBRÉS                                       │
│  ProfileService.C (agrégat, top 2 axes LUSR)            │
│  + Pattern Engine levers (calibrés sur p60, N matchs)   │
│  → phase 0 : surfacer C · phase 3 : ajouter patterns    │
├─────────────────────────────────────────────────────────┤
│  PATTERNS COMPORTEMENTAUX                               │
│  tilt · fatigue · engagement drop · plateau précision   │
│  → phase 2                                              │
├─────────────────────────────────────────────────────────┤
│  PATTERNS CONTEXTUELS                                   │
│  par mode · par map · squad vs solo                     │
│  OC/DR/DeltaCSR/DeltaLUSR agrégés par contexte         │
│  → phase 1                                              │
├─────────────────────────────────────────────────────────┤
│  PROFIL AGRÉGAT  (PlayerProfile A1/A2/B/C)              │
│  radar 6 axes · style · composantes LUSR · leviers      │
│  → phase 0 : surfacer l'existant                        │
├─────────────────────────────────────────────────────────┤
│  SUIVI ACTIF  (existant, fonctionnel)                   │
│  streaks · records · milestones                         │
└─────────────────────────────────────────────────────────┘
```

**Frontière claire entre les deux couches "leviers"** :
- `ProfileService.C` → levier *agrégat* : "ta composante deaths_vs_expected a le plus fort levier sur ton LUSR global"
- Pattern Engine levers → levier *calibré* : "en Slayer sur les 50 derniers matchs, vise OC ≥ 1.1 (ton p60) en 20 matchs"
Les deux coexistent et se complètent dans la page Profil de jeu — pas de doublon.

---

## Phase 0 — Réparer et surfacer (~3h)

> **Valeur joueur** : pour la première fois, il voit son radar, son style de jeu, ses 8 composantes LUSR et ses 2 axes prioritaires.

### 0.1 Fix streaks perf-based (1 ligne, `post_sync_progression.go`)

**Problème** : `medianKDA` est calculée (ligne ~430) mais jamais injectée dans `EvaluateInput.Thresholds`.

**Fix** :
```go
// post_sync_progression.go — dans la construction de EvaluateInput
Thresholds: map[streaks.StreakType]float64{
    streaks.StreakTypeDailyPerf:        medianKDA,   // ← manquait
    streaks.StreakTypeWeeklyKDAThreshold: medianKDA,  // ← manquait
},
```

**Test** : `TestEvaluateWithPerfThreshold` — mock 7 matchs avec KDA au-dessus/en-dessous de la médiane, vérifier que `daily_perf` se déclenche.

---

### 0.2 Hook + page "Profil de jeu" (frontend)

**Endpoint existant** : `GET /api/v1/players/{slug}/profile?window=30`
**Aucun hook ne l'appelle** — le créer suffit.

**Fichiers à créer / modifier** :

| Fichier | Action |
|---------|--------|
| `apps/web/src/features/ascension/queries.ts` | Ajouter `progressionProfile(slug)` query key + `useProfile()` hook SWR |
| `apps/web/src/features/ascension/ProfileRadarSection.tsx` | Radar 6 axes (ECharts RadarChart existant) + Strengths/ImprovementAreas |
| `apps/web/src/features/ascension/StyleBadge.tsx` | Badge style de jeu (StyleKey → label i18n + icône) |
| `apps/web/src/features/ascension/LUSRComponentsGrid.tsx` | 8 composantes : barre current / top20 / targetForTier |
| `apps/web/src/features/ascension/LeveragePanel.tsx` | Top 2 leviers agrégat + 3 suggestions de défis |
| `apps/web/src/features/ascension/AscensionPage.tsx` | Nouvelle section "Profil de jeu" intégrant les 4 composants |
| `apps/web/public/locales/{fr,en}/ascension.toml` | Clés i18n : style_key, radar axes, leverage messages |

**Structure de la page Ascension après phase 0** :
```
/ascension
├── [existant] Streaks Dashboard
├── [existant] Records Timeline
├── [existant] Milestones Grid
└── [nouveau]  Profil de jeu
    ├── Radar 6 axes + top 3 forces / bottom 3 axes de progression
    ├── Badge style de jeu (StyleKey)
    ├── Composantes LUSR (8 barres current → targetForTier)
    └── Leviers prioritaires (2 axes + 3 défis suggérés)
```

**Règles frontend** :
- Couleurs : `tokenCssVar()` uniquement
- Conditionnel : section masquée si `profile.minMatchesMet === false` (< 30 matchs)
- Skeleton loading sur chaque sous-section indépendamment

---

## Phase 1 — Patterns contextuels (~6h)

> **Valeur joueur** : "Tu gagnes 68% en Slayer mais 31% en CTF — ta résistance chute de 25% en mode objectif."

### 1.1 Package `internal/analysis/patterns/` — socle

```
internal/analysis/patterns/
├── types.go      -- MatchRow, AnalyzeInput, PatternReport, PatternConfig
├── engine.go     -- Analyze(AnalyzeInput) PatternReport
└── context.go    -- byMode, byMap, bySquad
```

**Principe** : package stateless, 0 accès DB, 0 import Streamlit/React. Entrée = `[]MatchRow` enrichis côté service avant appel. Testable sans infrastructure.

#### `MatchRow`

```go
type MatchRow struct {
    // Identité
    MatchID      string
    PlayedAt     time.Time
    Mode         string    // normalisé via halo-modes
    MapID        string
    Outcome      Outcome   // WIN=2 / LOSS=3 / DRAW=1 / DNF=4
    IsRanked     bool
    DurationSec  int
    SessionID    string

    // Combat — Tier 1 (toujours disponibles)
    KDA          float64
    Kills        int
    Deaths       int
    Assists      int

    // Combat — Tier 2 (0 si absent)
    Accuracy     float64
    OC           float64   // offensive_conversion, calculé à la volée
    DR           float64   // defensive_resistance, calculé à la volée
    HSRate       float64   // headshot_kills / kills
    FirstKills   int       // count highlight_events first_kill
    MMRDelta     float64   // team_mmr − enemy_mmr

    // Performance relative (nil si < 10 matchs historique)
    PerfScore    *float64

    // Engagement — paire indissociable, toujours lus ensemble
    // EngageScore : percentile 0-100 vs propre historique (même mode)
    //   "suis-je plus ou moins actif qu'à mon habitude ?"
    // ResidualBrut : pace joueur − pace attendu cross-joueurs
    //   "suis-je objectivement actif par rapport à ce lobby ?"
    // Les deux sont nil si highlight_events insuffisants (< 10 matchs)
    EngageScore  *float64
    ResidualBrut *float64

    // Skill rating (nil si non disponible)
    DeltaLUSR    *float64  // variation μ vs match précédent
    DeltaCSR     *float64  // variation CSR vs match ranked précédent
    CSRValue     *float64  // valeur absolue CSR post-match

    // Contexte
    IsWithFriends bool
}
```

#### `PatternConfig` — seuils injectables

```go
type PatternConfig struct {
    MinMatchesPerGroup    int     // 5
    StrengthWinRateDelta  float64 // 0.12
    WeaknessWinRateDelta  float64 // 0.12
    TiltLossRun           int     // 3
    TiltKDADropPct        float64 // 0.25
    EngageDeltaTilt       float64 // 0.20
    FatigueMinSession     int     // 4
    FatigueSessionCovPct  float64 // 0.60
    AccuracyPlateauStd    float64 // 0.02
    AccuracyPlateauMax    float64 // 0.45
}
```

#### `PatternReport`

```go
type PatternReport struct {
    WindowSize       int
    ContextPatterns  []ContextualPattern
    BehaviorPatterns []BehavioralPattern
    Levers           []Lever
    ComputedAt       time.Time
}
```

---

### 1.2 Patterns contextuels (`context.go`)

#### byMode

1. Grouper par `Mode`, garder groupes `count ≥ 5`
2. Par groupe : `winRate`, `avgKDA`, `avgOC`, `avgDR`, `avgPerfScore`, `avgDeltaLUSR`, `avgDeltaCSR`
3. Delta vs moyenne globale joueur
4. `Strength` si `winRate > global + 0.12` ET `count ≥ 10` ; `Weakness` si inverse

```go
type ContextualPattern struct {
    Type         ContextType   // "by_mode" | "by_map" | "by_squad"
    Key          string        // "Slayer", "Aquarius", "with_friends"
    MatchCount   int
    WinRate      float64
    AvgKDA       float64
    AvgOC        float64
    AvgDR        float64
    AvgPerf      *float64
    AvgDeltaCSR  *float64
    AvgDeltaLUSR *float64
    Delta        float64       // delta winRate vs global
    Signal       Signal        // "strength" | "weakness" | "neutral"
}
```

**Exemple** :
```
Mode "Slayer": 42 matchs · wr=0.68 · avgOC=1.3 · avgDR=1.1 · delta=+28% → Strength
Mode "CTF":    18 matchs · wr=0.31 · avgOC=0.7 · avgDR=0.9 · delta=−26% → Weakness
```
L'OC/DR par contexte répond à "pourquoi" : en CTF, l'OC chute — le joueur se disperse au lieu de finir ses kills.

#### byMap

Même logique, seuil `±0.15`, `count ≥ 5`.

#### bySquad

Si `is_with_friends` disponible sur ≥ 10 matchs dans chaque groupe :
- `winRate_solo` vs `winRate_squad`, `avgKDA_solo` vs `avgKDA_squad`
- Signal si delta > 15%

---

### 1.3 Jointure étendue côté service

Le `patterns_service.go` est responsable de charger les `MatchRow` enrichis *avant* d'appeler `patterns.Analyze()`. Il calcule lui-même :
- `DeltaLUSR` et `DeltaCSR` : tri chronologique des rows + diff sur `rating_value` par `rating_type`
- `OC` / `DR` : via `combat_yield.ComputeCombatYield()` (0 requête supplémentaire)
- `HSRate` : `headshot_kills / kills`
- `FirstKills` : requête groupée sur `highlight_events` pour les N matchs en une passe

```sql
-- Jointure canonique étendue
SELECT
    p.kills, p.deaths, p.assists, p.accuracy,
    p.damage_dealt, p.damage_taken, p.personal_score,
    p.outcome, p.playlist_group, p.team_mmr, p.enemy_mmr, p.headshot_kills,
    r.match_id, r.map_id, r.game_variant_category, r.played_at, r.duration_secs,
    e.performance_score, e.session_id, e.is_with_friends,
    e.engagement_score, e.residual_brut,
    sk.rating_value AS skill_rating_value,
    sk.rating_type  AS skill_rating_type,
    sk.rating_deviation
FROM match_participants p
JOIN match_registry r USING (match_id)
LEFT JOIN player_match_enrichment e ON e.match_id = r.match_id
LEFT JOIN match_skill_rank sk       ON sk.match_id = r.match_id
WHERE p.xuid = $xuid
ORDER BY r.played_at DESC
LIMIT $n
```

---

### 1.4 Endpoint + frontend phase 1

**Backend** : `GET /api/player/:gamertag/patterns?n=50&title=:title`

```go
type PatternsResponse struct {
    WindowSize      int                 `json:"window_size"`
    ContextPatterns []ContextualPattern `json:"context_patterns"`
    // BehaviorPatterns et Levers : nil en phase 1, ajoutés phases 2 et 3
    ComputedAt      string              `json:"computed_at"`
}
```

**Frontend** : nouvelle sous-section dans "Profil de jeu" (après les composantes LUSR) :

| Composant | Description |
|-----------|-------------|
| `PatternContextGrid` | Grille modes/maps : badge Signal + winRate + OC/DR delta |
| `SquadVsSoloCard` | Comparaison perfs solo vs squad (si données suffisantes) |

---

### 1.5 Tests phase 1

- 0 matchs → `PatternReport{}` vide sans panique
- count < 5 par mode → aucun `ContextualPattern` généré
- 30 Slayer (wr=0.70, OC=1.3) + 15 CTF (wr=0.30, OC=0.7) → Strength Slayer + Weakness CTF avec OC enrichi
- Solo wr=0.40 vs Squad wr=0.65 → Strength bySquad
- DeltaCSR disponible uniquement en ranked → `AvgDeltaCSR` nil pour matchs sociaux

---

## Phase 2 — Patterns comportementaux (~5h)

> **Valeur joueur** : "On a détecté que ta précision chute de 35% après 3 défaites, et ton engagement s'effondre dès le 5ème match de la session."

Ajout de `behavioral.go` dans le package `internal/analysis/patterns/`.

```go
type BehavioralPattern struct {
    Type      BehaviorType  // "tilt" | "session_fatigue" | "engagement_drop" | "accuracy_plateau" | "perf_ceiling"
    Trigger   string        // "3+ défaites consécutives"
    Evidence  string        // "KDA chute de 1.8 → 1.1 ; engagement −30%"
    Severity  Severity      // "low" | "medium" | "high"
    Confirmed bool          // vrai si Mann-Whitney p < 0.05 (≥ 20 matchs)
}
```

### 2.1 Tilt

**Trigger** : run `LOSS ≥ 3` consécutifs
**Métriques** : KDA, OC, DR, EngageScore (si disponible)
**Seuil** : chute KDA > 25% → High
**Enrichissement engagement** : si `EngageScore` dispo, l'ajouter à `Evidence` ("KDA −35% ; engagement −30%"). La paire engage contextualise : est-ce du désengagement mental ou un vrai gap de niveau ?
**Confirmation** : Mann-Whitney U (déjà implémenté dans `internal/analysis/temporal/` pour la campagne — vérifier import sans cycle)

### 2.2 Fatigue de session

**Trigger** : sessions ≥ `FatigueMinSession` matchs (même `SessionID` ou gap < 30 min)
**Mesure** : LOWESS sur KDA/PerfScore par position dans la session (1er, 2ème, 3ème…)
**Seuil** : pente négative sur ≥ 60% des sessions analysées
**Enrichissement** : si `EngageScore` décroit en corrélation → confirme la fatigue (pas de la malchance)

### 2.3 Engagement drop

**Trigger** : `EngageScore` AND `ResidualBrut` *tous deux* disponibles ET tous deux < P25 sur ≥ 5 matchs récents.
La paire est indissociable :
- `EngageScore` faible seul → "moins actif qu'à mon habitude" (peut-être fatigué)
- `ResidualBrut` faible seul → "moins actif que le lobby" (mode inadapté)
- **Les deux faibles** → signal fort, désengagement réel

**Severity** : Medium si 5-10 matchs, High si > 10
**Evidence** : "Engagement perso : −35% vs habitude ; −28% vs le lobby"

### 2.4 Plateau de précision

**Trigger** : rolling std de `Accuracy` sur 30 matchs < `AccuracyPlateauStd` (0.02) ET `avgAccuracy < 0.45`
**Enrichissement** : si `HSRate` disponible, comparer `accuracy` vs `HSRate` — un joueur avec accuracy=0.40 mais HSRate=0.55 a un problème de ciblage précis, pas de timing.

### 2.5 Plafond de performance

**Trigger** : `max(PerfScore, 30 matchs) − mean(top10 PerfScore)` < 5 pts ET LOWESS plat 30+ matchs
**Enrichissement CSR** : si `DeltaCSR` disponible, vérifier si le CSR stagne aussi — un PerfCeiling sans stagnation CSR peut indiquer un style qui ne se traduit pas en victoires d'équipe.

---

### 2.6 Frontend phase 2

| Composant | Description |
|-----------|-------------|
| `BehaviorAlertList` | Liste des patterns comportementaux avec sévérité + Evidence + conseil actionnable |

Intégré dans "Profil de jeu" sous les patterns contextuels.

---

### 2.7 Tests phase 2

- 5 LOSS consécutifs, KDA −40%, EngageScore −35% → Tilt High avec Evidence engage
- Session 6 matchs, LOWESS négatif KDA + EngageScore corrélé → SessionFatigue Medium
- EngageScore ET ResidualBrut au P15 sur 8 matchs → EngagementDrop Medium
- EngageScore seul faible → pas d'EngagementDrop (paire requise)
- accuracy std < 0.02, HSRate disponible → AccuracyPlateau avec enrichissement HSRate
- PerfScore plateau + DeltaCSR stagnant → PerfCeiling Medium

---

## Phase 3 — Leviers calibrés + intégration coach (~5h)

> **Valeur joueur** : "En Slayer, vise un OC ≥ 1.1 (tu es à 0.82 actuellement). À ton rythme, c'est atteignable en 20 matchs."

### 3.1 Leviers calibrés (`levers.go`)

**Principe** : chaque `ContextualPattern` (Weakness) et `BehavioralPattern` (Severity ≥ Medium) génère un `Lever`. Complémentent `ProfileService.C`, ne le remplacent pas.

```go
type Lever struct {
    Rank          int
    Axis          string    // "mode_selection" | "accuracy" | "session_length" | "engagement" | "csr_ranked" | ...
    Label         string    // description courte (FR)
    CurrentVal    float64
    TargetVal     float64   // p60 du joueur sur ses propres données
    Horizon       int       // matchs estimés [10, 100]
    Impact        float64   // 0..1, corrélation métrique → DeltaLUSR (ou DeltaCSR, ou Outcome)
    SourcePattern string
}
```

#### Catalogue

| Source | Axis | Calibration | Proxy Impact |
|--------|------|-------------|--------------|
| Mode Weakness | `mode_selection` | winRate global −5% | corr(winRate, DeltaLUSR) |
| Map Weakness | `map_avoidance` | winRate global −5% | même |
| Tilt | `session_management` | KDA hors-tilt −10% | fréquence tilt vs DeltaCSR |
| Session Fatigue | `session_length` | N matchs = point décrochage | DeltaLUSR par position |
| Accuracy Plateau | `accuracy` | p60 accuracy 30 matchs | corr(accuracy, DeltaCSR/LUSR) |
| Engagement Drop | `engagement` | p60 EngageScore 30 matchs | corr(ResidualBrut, winRate) |
| Perf Ceiling | `radar_axis` | gap radar bottom 3 | corr(axe, PerfScore) |
| Solo/Squad delta | `squad_play` | winRate_squad ou solo | delta direct |
| CSR stagnation | `csr_ranked` | DeltaCSR p60 sur ranked | DeltaCSR historique |

#### Calibration TargetVal

```go
func calibrateTarget(values []float64) (target float64, ok bool) {
    if len(values) < 10 {
        return 0, false
    }
    sorted := slices.Sorted(slices.Values(values))
    p60 := sorted[int(0.60*float64(len(sorted)))]
    // Si p60 ≤ currentAvg : joueur déjà au-dessus → pas de levier
    return p60, p60 > mean(values)
}
```

#### Impact

Corrélation de Pearson (métrique levier → `DeltaLUSR`) sur 30 matchs. Fallback : `DeltaCSR` → `Outcome` (0/1). Valeur absolue normalisée 0..1.

#### Horizon

```
horizon = ceil((targetVal − currentVal) / avgProgressionPerMatch)
```
Borné [10, 100]. `avgProgressionPerMatch` = rolling mean progression observée 30 matchs.

---

### 3.2 Intégration coach (`generator.go`)

```go
report := patterns.Analyze(patterns.AnalyzeInput{
    Rows:   toPatternRows(input.RecentMatches),
    N:      50,
    Config: patterns.DefaultPatternConfig(),
    Now:    input.Now,
})

for _, lever := range report.Levers {
    if lever.Rank <= 3 && lever.Impact > 0.3 {
        alerts = append(alerts, buildLeverAlert(lever))
    }
}
```

**4 nouveaux `AlertType`** dans `coach/types.go` :
```go
AlertPatternStrength  AlertType = "pattern_strength"
AlertPatternWeakness  AlertType = "pattern_weakness"
AlertPatternBehavior  AlertType = "pattern_behavior"
AlertPatternLever     AlertType = "pattern_lever"
```

**Extension du loader** dans `post_sync_progression.go` : ajouter `Mode`, `MapID`, `EngageScore`, `ResidualBrut`, `DeltaCSR`, `DeltaLUSR`, `IsWithFriends`, `MMRDelta`, `HSRate`, `FirstKills` à la jointure existante.

---

### 3.3 Frontend phase 3

| Composant | Description |
|-----------|-------------|
| `LeverList` | 3 leviers patterns calibrés : métrique courante → cible, barre de progression, horizon en matchs |

Positionnement dans "Profil de jeu" :
```
Profil de jeu
├── Radar + forces/faiblesses         (phase 0 — ProfileService.A1)
├── Badge style de jeu                 (phase 0 — ProfileService.A2)
├── Composantes LUSR                   (phase 0 — ProfileService.B)
├── Leviers agrégat + défis suggérés   (phase 0 — ProfileService.C)
├── Patterns contextuels               (phase 1 — PatternEngine)
├── Patterns comportementaux           (phase 2 — PatternEngine)
└── Leviers calibrés                   (phase 3 — PatternEngine)
```

---

## Plan d'implémentation — ordre des commits

### Phase 0 (Réparer + surfacer)

| # | Commit | Fichiers clés |
|---|--------|---------------|
| 1 | fix(streaks): injecter medianKDA dans EvaluateInput.Thresholds | `post_sync_progression.go` (+test) |
| 2 | feat(profile): hook useProfile + types TS | `queries.ts`, `types.ts` |
| 3 | feat(profile): ProfileRadarSection + StyleBadge | `ProfileRadarSection.tsx`, `StyleBadge.tsx` |
| 4 | feat(profile): LUSRComponentsGrid + LeveragePanel | `LUSRComponentsGrid.tsx`, `LeveragePanel.tsx` |
| 5 | feat(profile): intégration AscensionPage + i18n | `AscensionPage.tsx`, `ascension.toml` |

### Phase 1 (Patterns contextuels)

| # | Commit | Fichiers clés |
|---|--------|---------------|
| 6 | feat(patterns): types + engine squelette + PatternConfig | `patterns/types.go`, `patterns/engine.go` |
| 7 | feat(patterns): context.go + tests | `patterns/context.go`, `patterns/context_test.go` |
| 8 | feat(patterns): service + jointure étendue + endpoint | `patterns_service.go`, `patterns_handler.go`, `router.go` |
| 9 | feat(patterns): PatternContextGrid + SquadVsSoloCard | `PatternContextGrid.tsx`, `SquadVsSoloCard.tsx` |

### Phase 2 (Patterns comportementaux)

| # | Commit | Fichiers clés |
|---|--------|---------------|
| 10 | feat(patterns): behavioral.go (tilt, fatigue, engage, plateau, plafond) + tests | `patterns/behavioral.go`, `patterns/behavioral_test.go` |
| 11 | feat(patterns): BehaviorAlertList | `BehaviorAlertList.tsx` |

### Phase 3 (Leviers + coach)

| # | Commit | Fichiers clés |
|---|--------|---------------|
| 12 | feat(patterns): levers.go + tests | `patterns/levers.go`, `patterns/levers_test.go` |
| 13 | feat(coach): 4 nouveaux AlertTypes + intégration generator | `coach/types.go`, `coach/generator.go`, `post_sync_progression.go` |
| 14 | feat(patterns): LeverList + intégration finale AscensionPage | `LeverList.tsx`, `AscensionPage.tsx` |

---

## Contraintes et règles

| Contrainte | Application |
|------------|-------------|
| Package stateless | `internal/analysis/patterns/` : 0 import DB |
| Taille fichiers | ≤ 500L. `levers.go` probable boundary → extraire `calibration.go` si dépassé |
| Taille fonctions | `Analyze()` ≤ 80L → déléguer à `analyzeContext()`, `analyzeBehavior()`, `selectLevers()` |
| 0 magic numbers | Tous les seuils dans `PatternConfig` avec valeur par défaut documentée |
| Tests sans DB | `[]MatchRow` construits en mémoire dans tous les tests `patterns/` |
| Couleurs frontend | `tokenCssVar()` uniquement |
| Paire engagement | `EngageScore` et `ResidualBrut` toujours exposés et lus ensemble — ne jamais afficher l'un sans l'autre dans un insight |
| Leviers non redondants | `LeveragePanel` (ProfileService.C) = levier agrégat LUSR · `LeverList` (Pattern Engine) = levier calibré contextuel. Titres et textes distincts en i18n |

---

## Points ouverts

1. **Normalisation des modes** : utiliser le skill `halo-modes` pour `game_variant_category` → famille. Vérifier l'absence de cycle d'import depuis `internal/analysis/`.

2. **Mann-Whitney U** : existe dans `internal/analysis/temporal/` pour la campagne. Vérifier si exportable ou à dupliquer localement dans `patterns/`.

3. **Persistance des patterns** : V1 = calcul à la volée (pas de persistance). Si calcul > 500ms sur N=100 → envisager table `pattern_cache` post-sync en V2.

4. **CSR partiel** : joueur sans ranked → `DeltaCSR` nil. Fallback Impact : `DeltaLUSR` → `Outcome`. La logique de fallback est dans `levers.go`.

5. **Levier → Objectif Prestige** : les leviers patterns ont vocation à alimenter le système Objectifs (ex-Défis). Couplage hors-scope de ce plan — évolution naturelle post-livraison.

---

## Estimation effort

| Phase | Commits | Effort estimé |
|-------|---------|---------------|
| 0 — Réparer + surfacer | 1–5 | ~3h |
| 1 — Patterns contextuels | 6–9 | ~6h |
| 2 — Patterns comportementaux | 10–11 | ~5h |
| 3 — Leviers + coach | 12–14 | ~5h |
| **Total** | **14** | **~19h** |

---

*Plan v3 rédigé le 2026-05-23. Mettre à jour le `thought_log.md` à chaque commit livré.*
