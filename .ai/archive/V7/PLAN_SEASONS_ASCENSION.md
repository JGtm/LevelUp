# PLAN — Saisons Ascension (V2 du système Prestige)

> **STATUT : DEPRECATED — Non retenu**
>
> Ce cadrage a été écarté le 2026-05-13 après revue. Concept conservé pour mémoire et pour documenter le raisonnement.
>
> **Direction retenue à la place** : [PLAN_PROGRESSION_TRACKING_ASCENSION.md](PLAN_PROGRESSION_TRACKING_ASCENSION.md) (Streaks + Records & Milestones + Coach proactif positif)
>
> **Raisons de l'abandon** :
> 1. **Redondance** avec saisons Halo Infinite natives (Battle Pass, reset CSR) — risque de cycles parallèles confus
> 2. **LUSR fait déjà l'anti-régression** mathématique en continu (μ baisse, σ croît à l'inactivité) — pas besoin d'une couche supplémentaire
> 3. **Saison récompense un score à deadline**, pas la **répétition durable** (objectif pédagogique réel énoncé : "la progression vient de la répétition")
> 4. **Pénalité régression = culpabilisation** — psychologiquement contre-productive ; le système doit pousser positivement
> 5. **Complexité ROI** : ~15j de dev pour un concept dépendant d'un attachement saisonnier non démontré (pool joueurs réduit)
>
> Les éléments réutilisables (snapshot LUSR, formule consistency_bonus, mécanique anti-régression) peuvent revenir dans un cycle ultérieur si le besoin se fait sentir post-V2.

---

**Date** : 2026-05-13
**Statut** : DEPRECATED — Non retenu (cf. encadré ci-dessus)
**Dépendances** : V1 livrée ([PLAN_PLAYER_PROFILE_ASCENSION.md](PLAN_PLAYER_PROFILE_ASCENSION.md)) — besoin du moteur de profil joueur et de l'inversion LUSR

---

## 1. Contexte & objectif

Le système Prestige V1 propose des défis et arcs, mais ne mesure pas la **rétention** de la progression. Un joueur peut briller sur un défi puis régresser sans pénalité. L'objectif des Saisons est de créer un cadre **temporel fixe** qui :

1. **Récompense la progression durable** (μ LUSR qui monte et reste)
2. **Pénalise les régressions** (composantes qui baissent vs début de saison)
3. **Structure le parcours** en chapitres thématiques (3-5 arcs par saison)
4. **Crée un horizon motivant** (reward de fin de saison, badges, multiplicateurs PP)

Le mot-clé est **rétention** : ne pas juste atteindre un seuil, mais **le tenir** sur 3 mois.

---

## 2. Concept clé : Saison comme méta-arc anti-régression

### 2.1 Pourquoi LUSR rend ça naturel

LUSR (TrueSkill 2) est **déjà** un mécanisme anti-régression intrinsèque :
- μ baisse quand tu performes moins bien
- σ augmente avec l'inactivité
- Les composantes (Kills vs Expected, Accuracy, etc.) sont des moyennes glissantes qui captent les variations

Donc une "Saison Ascension" peut se construire **par-dessus LUSR** sans réinventer un système parallèle. La Saison est essentiellement un **suivi delta** entre deux snapshots LUSR + composantes.

### 2.2 Anatomie d'une saison

| Élément | Description |
|---|---|
| **Durée** | 90 jours par défaut (configurable par titre) |
| **Snapshot initial** | μ + σ + 8 composantes LUSR + radar 6 axes narrative + FK/FD + Engagement, capturés au début de saison |
| **Arcs de saison** | 3-5 arcs thématiques sélectionnés (différents des arcs preset libres actuels) |
| **Score saison** | Cumul de PP de saison gagnés par : (a) Δμ stabilisé, (b) Δcomposantes positifs pondérés, (c) défis et arcs complétés, (d) bonus de consistance |
| **Mécanisme anti-régression** | Malus sur Δcomposantes négatifs (cf. §4) |
| **Reward de fin** | Badge saison, multiplicateur PP, débloque saison suivante avec arcs avancés |

---

## 3. Modèle de données

### 3.1 Nouvelles entités

```go
// apps/go-api/internal/prestige/seasons/types.go

type Season struct {
    ID             string
    UserID         string
    TitleSlug      string
    PresetID       string       // ref vers SeasonPreset (config TOML)
    StartedAt      time.Time
    EndsAt         time.Time
    Status         SeasonStatus // active | completed | abandoned
    Snapshot       SeasonSnapshot
    CurrentScore   float64      // PP saison cumulés
    ConsistencyMult float64     // facteur multiplicateur (0.5 - 2.0)
    BadgesEarned   []string     // milestones in-season
}

type SeasonSnapshot struct {
    CapturedAt       time.Time
    LUSRMu           float64
    LUSRSigma        float64
    LUSRTier         string         // "Diamond III"
    LUSRComponents   map[string]float64 // 8 composantes
    RadarAxes        map[string]float64 // 6 axes narrative
    FKFDRatio        float64
    EngagementScore  float64
    MatchesPlayed    int
}

type SeasonPreset struct {
    ID            string // ex: "halo_infinite.season.spring_2026"
    TitleSlug     string
    TitleEN       string // "Spring 2026 — The Ascension"
    TitleFR       string
    DurationDays  int    // 90 par défaut
    ArcPresetIDs  []string // 3-5 arcs choisis dans le preset
    Theme         string // "offensive" | "defensive" | "support" | "balanced"
    TargetTier    string // tier viseé (palier idéal de fin de saison)
    StartDate     string // ISO date (optionnel — si fixée par calendrier global)
    EndDate       string // ISO date (optionnel)
}

type SeasonStatus string
const (
    SeasonActive    SeasonStatus = "active"
    SeasonCompleted SeasonStatus = "completed"
    SeasonAbandoned SeasonStatus = "abandoned"
)
```

### 3.2 Tables DuckDB

Dans `stats.duckdb` (par joueur) :

```sql
CREATE TABLE season (
    id                VARCHAR PRIMARY KEY,
    user_id           VARCHAR NOT NULL,
    title_slug        VARCHAR NOT NULL,
    preset_id         VARCHAR NOT NULL,
    started_at        TIMESTAMP NOT NULL,
    ends_at           TIMESTAMP NOT NULL,
    status            VARCHAR NOT NULL,
    snapshot_json     VARCHAR NOT NULL,  -- SeasonSnapshot serialisé
    current_score     DOUBLE DEFAULT 0,
    consistency_mult  DOUBLE DEFAULT 1.0,
    completed_at      TIMESTAMP,
    abandoned_at      TIMESTAMP
);

CREATE TABLE season_event (
    id           VARCHAR PRIMARY KEY,
    season_id    VARCHAR REFERENCES season(id),
    event_type   VARCHAR NOT NULL,  -- "challenge_completed" | "arc_completed" | "regression_detected" | "milestone"
    occurred_at  TIMESTAMP NOT NULL,
    pp_delta     DOUBLE NOT NULL,
    payload_json VARCHAR             -- détails event
);
```

Dans `metadata.duckdb` (référentiels) :

```sql
CREATE TABLE season_preset (
    id              VARCHAR PRIMARY KEY,
    title_slug      VARCHAR NOT NULL,
    title_en        VARCHAR NOT NULL,
    title_fr        VARCHAR NOT NULL,
    duration_days   INTEGER NOT NULL,
    theme           VARCHAR,
    target_tier     VARCHAR,
    start_date      DATE,
    end_date        DATE
);

CREATE TABLE season_preset_arc (
    season_preset_id VARCHAR REFERENCES season_preset(id),
    arc_preset_id    VARCHAR NOT NULL,
    position         INTEGER NOT NULL,
    PRIMARY KEY (season_preset_id, position)
);
```

### 3.3 Config TOML

`config/titles/halo_infinite/seasons/presets.toml` :

```toml
[meta]
schema_version = 1
title_slug = "halo_infinite"

[[seasons]]
id              = "halo_infinite.season.spring_2026"
title_en        = "Spring 2026 — The Ascension"
title_fr        = "Printemps 2026 — L'Ascension"
duration_days   = 90
theme           = "balanced"
target_tier     = "diamond_iv"
start_date      = "2026-03-01"
end_date        = "2026-05-30"
arc_preset_ids  = ["halo_infinite.slayer", "halo_infinite.marksman", "halo_infinite.survivor"]

[[seasons]]
id              = "halo_infinite.season.summer_2026_offensive"
title_en        = "Summer Offensive 2026"
title_fr        = "Offensive d'été 2026"
duration_days   = 90
theme           = "offensive"
target_tier     = "onyx"
arc_preset_ids  = ["halo_infinite.slayer", "halo_infinite.support", "halo_infinite.consistent"]
```

---

## 4. Mécanisme anti-régression

### 4.1 Capture de snapshot

À l'activation de la saison :
1. Appeler `BuildProfile(xuid, titleSlug, window=100)` (service V1)
2. Persister le résultat dans `season.snapshot_json`
3. Démarrer le timer 90 jours

### 4.2 Évaluation après chaque match

Après ingestion d'un match (post-sync), pour chaque saison `active` du joueur :

```python
def evaluate_match_for_season(season, match):
    # 1. Recompute profile actuel
    current_profile = build_profile(season.user_id, season.title_slug, window=100)

    # 2. Compute deltas vs snapshot initial
    delta_mu = current_profile.lusr_mu - season.snapshot.lusr_mu
    delta_components = {
        comp: current_profile.components[comp] - season.snapshot.lusr_components[comp]
        for comp in LUSR_COMPONENTS
    }

    # 3. Compute consistency_bonus
    bonus = compute_consistency_bonus(delta_components)
    season.consistency_mult = clamp(0.5, bonus, 2.0)

    # 4. PP delta du match (base)
    pp_base = pp_from_match_outcome(match)  # déjà calculé par Prestige V1

    # 5. PP délivré pour la saison
    pp_season = pp_base * season.consistency_mult * theme_match_bonus(season.preset.theme, match)

    season.current_score += pp_season

    # 6. Détecter régressions notables (alerte UX)
    for comp, delta in delta_components.items():
        if delta < -REGRESSION_THRESHOLD:
            log_season_event(season, "regression_detected", payload={comp, delta})
```

### 4.3 Formule `consistency_bonus`

```
consistency_bonus = 1.0
                 + alpha * sum(max(0, delta_comp_i) * weight_lusr_i)   # bonus progression
                 - beta  * sum(max(0, -delta_comp_i) * weight_lusr_i)  # malus régression

avec alpha = 1.2 et beta = 1.8 (régression pénalise plus que progression récompense)
```

Le `weight_lusr_i` est le poids de la composante (0.27 pour Kills vs Expected, etc.). Cela donne automatiquement plus de poids aux composantes à fort impact LUSR.

Bornes : `clamp(0.5, consistency_bonus, 2.0)` pour éviter des feedback loops dramatiques.

### 4.4 Détection de régression

Une régression "notable" déclenche un event Season + une alerte UX :
- Seuil : `delta_comp < -0.05` (5% de baisse sur la composante)
- Affichage UX : *"Attention : ta composante 'Accuracy' a baissé de 0.07 depuis le début de la saison. Lance un défi ciblé pour redresser."*
- Pas de pénalité ponctuelle catastrophique — c'est le `consistency_mult` qui s'ajuste graduellement

### 4.5 Reward de fin de saison

À la complétion (90 jours atteints) :

| Condition | Reward |
|---|---|
| `current_score > target_pp_normal` | Badge bronze de saison + 100 PP bonus |
| `current_score > target_pp_heroic` | Badge argent + 250 PP bonus |
| `current_score > target_pp_legendary` | Badge or + 500 PP bonus + débloque saison "avancée" |
| `current_score > target_pp_mythic` | Badge platine + 1000 PP bonus + variante préset spéciale |
| Tier LUSR final > target_tier saison | Badge bonus "Tier Reached" + 200 PP |
| 0 régression notable détectée | Badge "Iron Consistency" |

---

## 5. Arcs de saison vs arcs preset actuels

### 5.1 Différences

| Aspect | Arc preset (V1) | Arc de saison (V2) |
|---|---|---|
| Activation | Libre, à tout moment | Embarqué dans une Saison active |
| Durée | Indéterminée (jusqu'à complétion) | Bornée par la saison (90j) |
| Récompense | PP fixe par étape | PP × `consistency_mult` de la saison |
| Sélection | Au choix du joueur | Imposé par le `SeasonPreset` (3-5 arcs thématiques) |
| Anti-régression | Non | Oui, via la mécanique saison |

### 5.2 Arcs de saison réutilisent les templates V1

Pas besoin de doublonner. Un arc preset existant (`halo_infinite.slayer`) peut être référencé par un `SeasonPreset`. La distinction tient au **contexte d'exécution** (saison active vs hors-saison).

### 5.3 Nouveaux arcs probables pour les saisons

À créer si V1 a déjà ajouté les arcs `marksman` et `survivor` (cf. plan V1 §6.2) :
- `halo_infinite.season.balanced_spring_2026` → Slayer + Marksman + Survivor
- `halo_infinite.season.offensive_summer_2026` → Slayer + Support + Consistent
- `halo_infinite.season.exploration_fall_2026` → Explorer + Support + Marksman

---

## 6. UI

### 6.1 Page Ascension > Saison en cours

Nouvel onglet (ou pré-tab) dans la nav L2 d'Ascension :

```
Ascension / [Saison en cours] [Mon parcours] [Défis]
```

Bloc principal "Saison en cours" :
- Titre de saison + thème + jours restants (cercle de progression)
- Score saison cumulé (vs target_pp_heroic affiché)
- `consistency_mult` actuel affiché comme jauge (0.5 → 2.0)
- Liste des 3-5 arcs avec progression
- Section "Alertes de régression" si applicable

### 6.2 Page profil joueur enrichie

Dans `PlayerProfileCard` (V1), ajouter en section B un overlay supplémentaire sur les composantes LUSR :
- 3 lignes : moyenne actuelle / top 20% perso / **snapshot début de saison**
- Indicateur de variation depuis snapshot saison

### 6.3 Notification de régression

Toast in-app + persistance dans `season_event` :
> *"Régression détectée sur 'Accuracy' (-7%). Ton `consistency_mult` est passé de 1.2 à 0.95."*

### 6.4 Page de fin de saison

Modal récapitulatif :
- Δμ total
- Top 3 composantes améliorées + bottom 3
- Defis et arcs complétés
- Badges gagnés
- CTA "Démarrer la prochaine saison"

---

## 7. Plan d'implémentation

**Branche** : `feat/seasons-ascension` (depuis main, après V1 mergée).

### Découpe en commits (estimation)

1. `feat(seasons): types Go + migration DuckDB tables season/event/preset` (~1j)
2. `feat(seasons): loader TOML preset + endpoints CRUD saison` (~2j)
3. `feat(seasons): service capture snapshot via PlayerProfile V1` (~1j)
4. `feat(seasons): évaluateur post-match + consistency_bonus + regression detection` (~3j)
5. `feat(seasons): reward de fin de saison + badges` (~1j)
6. `feat(seasons): UI Saison en cours + jauge consistency_mult` (~3j)
7. `feat(seasons): UI alertes régression + modal fin de saison` (~2j)
8. `feat(seasons): 3-5 SeasonPreset config TOML pour Halo Infinite` (~0.5j)
9. `chore(seasons): i18n manifests + tests + ADR 0013` (~1j)

**Total estimation** : ~14-15j (2-3 semaines selon parallélisation).

---

## 8. Risques & décisions à prendre

### 8.1 Risques techniques

| Risque | Mitigation |
|---|---|
| Calcul snapshot lourd au démarrage de saison | Cache le `PlayerProfile` après build initial, snapshot = serialization JSON |
| Évaluateur post-match ralentit le sync | Async (worker pool), idempotent, replayable depuis `season_event` |
| Volume `season_event` croît rapidement | TTL 1 an, archivage Parquet pour historique |
| Régressions "techniques" non méritées (changement de playlist, etc.) | Filtrer par `playlist_group` cohérent avec snapshot. Ignorer matchs <5 (sample trop petit) |
| Consistency_mult feedback loop défavorable | Bornes [0.5, 2.0] strictes. Si joueur en spirale négative, alerte UX + reset volontaire possible |

### 8.2 Décisions à prendre

| Question | Options | Recommandation |
|---|---|---|
| Saisons globales (calendrier fixe) ou personnelles (le joueur démarre quand il veut) ? | Global / Perso / Hybride | **Perso** pour V2 (plus flexible). Global possible V3 |
| Multi-saisons en parallèle ? | Non / 1 par titre / N | **1 par titre** — éviter la dispersion |
| Que faire si joueur inactif > 30j ? | Abandon auto / Pause / Rien | **Pause auto** avec extension `ends_at`, retour à la connexion |
| Comparaison entre joueurs (classement saison) ? | Oui / Non | **Non en V2** — focus rétention perso. V3 si demande |
| Saison rétroactive ? (démarrer une saison sur 90j passés) | Oui / Non | **Non** — incohérent avec le concept de snapshot initial |
| Intégration avec V1.5 (défis conditionnels) ? | Bloquant / Non bloquant | **Non bloquant** — V2 peut être livrée avant V1.5 si V1.5 retardée |

---

## 9. Dépendances

### 9.1 Dépend de V1 livrée

- `PlayerProfile` service (pour capture snapshot et évaluateur post-match)
- Inversion math LUSR `RequiredCompositeForTier` (pour `target_tier` saison)
- Endpoint `/players/{slug}/profile` (réutilisé)
- Templates taggés `lusr_components` + `is_long_term` (pour matching dans arcs de saison)

### 9.2 Indépendant de V1.5

V2 peut être livrée même si V1.5 (défis conditionnels) n'est pas faite. Les arcs de saison peuvent se contenter des templates V1 standards.

### 9.3 Pour V3 (idées futures)

- Saisons globales avec calendrier commun à tous les joueurs
- Classement saison entre joueurs / escouade
- Saisons spéciales événementielles (Halloween, Anniversaire, etc.)
- Loot/cosmétiques de fin de saison

---

## 10. Pourquoi attendre V1 ?

V1 livre :
1. Le **moteur de profil** (PlayerProfile)
2. L'**inversion LUSR** (target par tier)
3. L'**audit catalogue** + nouveaux templates long-terme
4. L'**affichage radar** + composantes LUSR

V2 (Saisons) **réutilise tout cela** comme briques. Sans V1, le concept de "snapshot saison" n'a pas de fondation algorithmique. La séquence V1 → V2 minimise le re-work et permet de valider que le moteur de profil tient la charge avant d'y construire dessus.

---

## 11. Références

- [PLAN_PLAYER_PROFILE_ASCENSION.md](PLAN_PLAYER_PROFILE_ASCENSION.md) — V1 (dépendance)
- `docs/adr/0004-narrative-engine.md` — 8 rôles + radar 6 axes
- `docs/adr/0005-prestige-phased-activation.md` — activation phasée Prestige
- `apps/go-api/internal/sync/skill_config.go` — tiers LUSR (cibles `target_tier`)
- `apps/go-api/internal/prestige/types.go` — extension nécessaire `ChallengeTemplate.LUSRComponents` (V1)
