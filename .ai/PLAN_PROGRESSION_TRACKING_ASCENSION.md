# PLAN — Progression Tracking (V2 d'Ascension)

**Date** : 2026-05-13
**Statut** : Cadrage initial, à valider avant implémentation
**Dépendances** : V1 livrée ([PLAN_PLAYER_PROFILE_ASCENSION.md](PLAN_PLAYER_PROFILE_ASCENSION.md))
**Alternative considérée et écartée** : [PLAN_SEASONS_ASCENSION.md](PLAN_SEASONS_ASCENSION.md) (DEPRECATED — cf. §2)

---

## 1. Contexte & objectif

Le profil joueur V1 donne une **photo** (qui tu es, où tu en es). V2 doit donner du **mouvement** : pousser le joueur à revenir, célébrer ses progrès, et l'aider à maintenir la régularité — sans deadline artificielle ni culpabilisation.

Objectifs explicites :
1. **Pousser à la répétition** (Duolingo-style : se reconnecter chaque jour)
2. **Célébrer la progression** (gratification immédiate, records visibles)
3. **Suggérer des actions adaptées** (coach proactif sur opportunités, pas sur faiblesses)

**Principe directeur** : feedback **positifs uniquement**. Le système ne pointe jamais du doigt une régression ; il met en avant les opportunités, les records, et les paliers proches.

**Principe complémentaire — jamais imposé** : le joueur peut continuer à créer des défis libres "juste pour le fun" sans interagir avec Streaks, Records ou Coach. Les 3 couches V2 sont actives par défaut **mais discrètes** : compteurs de streaks visibles sans être intrusifs, records auto-détectés sans toast obligatoire (configurable), centre de notifs accessible mais jamais bloquant. Aucune feature de V1 ou V2 ne peut empêcher le flow Prestige libre existant.

---

## 2. Pourquoi pas Saisons

Saisons écartées car :
- **Redondance** avec saisons Halo Infinite natives (Battle Pass, reset CSR)
- **LUSR fait déjà** l'anti-régression mathématique (μ baisse en continu) — pas besoin de couche supplémentaire
- **La saison récompense un score à deadline**, pas la répétition. Or la répétition est l'objectif pédagogique réel.
- **Pénalité régression** = culpabilisation potentielle → contre-productive
- **Complexité ROI** : ~15j pour un concept dont l'utilité dépend d'un attachement saisonnier non démontré

Cf. [PLAN_SEASONS_ASCENSION.md](PLAN_SEASONS_ASCENSION.md) pour le détail du cadrage initial (conservé en mémoire).

---

## 3. Concept : 3 couches complémentaires

| Couche | Pousse | Suit | Effort |
|---|---|---|---|
| **Streaks** | Peur de casser la série (sunk cost positif) | Compteur visible | ~3-4j |
| **Records & Milestones** | Approche d'un PB proche, milestones débloquables | Timeline de records | ~3-4j |
| **Coach proactif** | Notifications sur opportunités et améliorations | Dashboard de tendances LOWESS | ~5-6j |

**Total estimé** : ~10-12j (vs ~15j Saisons), avec utilité bien plus immédiate.

---

## 4. Couche 1 — Streaks

### 4.1 Mécanique

Une **streak** = série continue de périodes (jour, semaine) satisfaisant une condition simple, propre au joueur.

Types proposés :

| Type | Condition | Période |
|---|---|---|
| `daily_play` | Au moins 1 match joué dans la journée | Jour |
| `daily_perf` | Au moins 1 match avec stat principale > seuil personnel | Jour |
| `weekly_play` | Au moins 5 matchs dans la semaine | Semaine ISO |
| `weekly_kda_threshold` | KDA moyen hebdomadaire > seuil personnel | Semaine ISO |

Le **seuil personnel** est calculé depuis V1 (`PlayerProfile`) — typiquement la médiane des 100 derniers matchs sur la stat. Pas de seuil universel.

### 4.2 Streak Shield (anti-frustration)

Pour éviter la frustration "1 jour raté, tout perdu" :
- Chaque joueur reçoit **1 shield par mois** (régénère le 1er du mois)
- Un shield s'active automatiquement le lendemain d'une journée manquée
- Affichage explicite : *"Streak shield utilisée — tu as 5 jours d'affilée préservés (1 shield restante ce mois)"*
- Au-delà des shields, la streak reset

C'est le mécanisme Snapchat/Duolingo qui réduit la pression sans annuler l'effet motivationnel.

### 4.3 Multiplicateur PP de streak

| Durée streak | Multiplicateur PP des défis complétés pendant la streak |
|---|---|
| 1-3 jours | 1.00x (aucun bonus) |
| 4-7 jours | 1.10x |
| 8-14 jours | 1.25x |
| 15-29 jours | 1.50x |
| 30+ jours | 1.75x (cap) |

Le bonus s'applique aux PP gagnés via défis Prestige (système existant). Encourage à conserver la streak pour ne pas perdre le multiplicateur.

### 4.4 Modèle de données

```go
// apps/go-api/internal/prestige/streaks/types.go

type Streak struct {
    ID            string
    UserID        string
    TitleSlug     string
    Type          StreakType  // daily_play, daily_perf, weekly_play, weekly_kda_threshold
    StartedAt     time.Time
    CurrentLength int         // jours ou semaines
    BestLength    int         // record perso
    LastIncrementAt time.Time
    Threshold     float64     // seuil personnel (si applicable)
    ShieldsUsed   int         // dans le mois en cours
    ShieldsAvailable int      // 1/mois, regen le 1er
    Status        StreakStatus // active | paused (shield) | broken
}
```

Table DuckDB `streak` (par joueur).

### 4.5 Évaluation

Cron léger ou job post-sync :
1. Identifier matchs/sessions du joueur sur la dernière période (jour ou semaine)
2. Pour chaque streak active, vérifier la condition
3. Si OK → incrémenter, mettre à jour `current_length`, vérifier nouveau record
4. Si KO → utiliser shield si dispo, sinon break

---

## 5. Couche 2 — Records & Milestones

### 5.1 Records (Personal Bests)

Pour chaque composante LUSR et chaque axe radar narrative, on track le PB sur 3 fenêtres :
- **30 jours** (forme actuelle)
- **90 jours** (tendance moyen terme)
- **All-time** (carrière)

```go
type PersonalRecord struct {
    UserID       string
    TitleSlug    string
    Metric       string      // "kills_vs_expected" | "accuracy_delta" | "combat_axis" | ...
    Window       string      // "30d" | "90d" | "all_time"
    Value        float64
    AchievedAt   time.Time
    PreviousValue float64
    PreviousAchievedAt time.Time
}
```

**Détection automatique** post-match :
- À chaque ingestion, recompute moyenne sur la fenêtre
- Si > record actuel → enregistrer, émettre événement `record_broken`
- UX : toast in-app + entrée dans timeline records

### 5.2 Milestones (paliers cumulatifs)

Catalogue de milestones par titre, définis en TOML :

```toml
# config/titles/halo_infinite/milestones/catalog.toml

[[milestones]]
id          = "halo_infinite.matches.100"
metric      = "matches_played"
threshold   = 100
title_fr    = "Centurion"
title_en    = "Centurion"
icon        = "milestone_100_matches"

[[milestones]]
id          = "halo_infinite.wins.50"
metric      = "wins"
threshold   = 50
title_fr    = "Vétéran de la victoire"
title_en    = "Victory Veteran"

[[milestones]]
id          = "halo_infinite.accuracy_consistent.30d"
metric      = "accuracy_threshold_days"
threshold   = 30
condition   = "accuracy >= 0.50 on 30 distinct days"
title_fr    = "Tireur régulier"
title_en    = "Steady Marksman"
```

Détection : cron ou job post-match, idempotent. Une fois débloqué, milestone persisté + notif.

### 5.3 Affichage

- **Timeline Records** : liste chronologique sur la page profil ("12 mai : nouveau record Accuracy 30j à 0.58")
- **Badges Milestones** : grille visuelle dans Section A du profil
- **Records "proches"** : *"Tu es à 0.02 de battre ton record d'Accuracy 30j"* — pousse à jouer

---

## 6. Couche 3 — Coach proactif

### 6.1 Principe

Le coach **observe** en continu et **agit** sur signaux **positifs uniquement** :
- Améliorations soutenues (LOWESS positif sur composante)
- Approche d'un palier (tier LUSR, record, milestone)
- Streaks notables (palier de durée)
- Reprise après pause (retour bienvenu)

Le coach **n'alerte jamais** sur :
- Régressions (laisser le joueur les voir dans son dashboard s'il veut)
- Sous-performance vs population

### 6.2 Types d'alertes positives

| Trigger | Message |
|---|---|
| Composante LUSR LOWESS positif sur 14j | *"Ton Accuracy progresse depuis 2 semaines — lance ce défi pour consolider"* |
| Approche d'un sub-tier LUSR (< 10 pts μ) | *"À 7 pts de Diamond IV — ton prochain match peut faire la différence"* |
| Approche d'un record (< 5% du PB) | *"Tu approches de ton record Combat 30j — peux-tu le battre cette session ?"* |
| Approche d'un milestone | *"Plus que 5 wins pour débloquer 'Vétéran de la victoire'"* |
| Streak 7/14/30j atteinte | *"7 jours d'affilée ! Tu es maintenant en multiplicateur 1.25x"* |
| Reprise après pause > 5j | *"Bon retour ! Tu as toujours ta streak shield disponible"* |
| **Campagne : progression confirmée Mann-Whitney** (V1) | *"Pendant ta campagne Survie, ton 'Deaths vs Expected' a progressé de +0.04 (confirmé p<0.05)"* |
| **Campagne : axe sortant du bottom 3** (V1, auto-suggestion clôture) | *"Survie n'est plus dans tes axes prioritaires — clore ta campagne et en lancer une nouvelle ?"* |

**Priorité des alertes campagne** : si une `ImprovementCampaign` est `active` pour le joueur, les alertes sur son axe ciblé sont **boostées en priorité** dans le centre de notifs (apparaissent en premier). Cohérent avec l'attention volontaire que le joueur porte à cet axe.

### 6.3 Modèle de données

```go
type CoachAlert struct {
    ID         string
    UserID     string
    TitleSlug  string
    Type       AlertType
    Title      string  // i18n key
    Body       string  // i18n key + interpolation
    PayloadJSON string // data spécifique au type
    CreatedAt  time.Time
    ReadAt     *time.Time
    DismissedAt *time.Time
}
```

Stockés en `stats.duckdb` par joueur. TTL 30j pour purger.

### 6.4 Évaluateur

Worker Go déclenché post-sync :
1. Charger `PlayerProfile` (V1)
2. Charger tendances LOWESS sur composantes (via `temporal`)
3. Comparer aux records, milestones, paliers tier
4. Générer alertes éligibles
5. Déduplication (1 alerte du même type par 24h max)

### 6.5 UX

- **Centre de notifications** dans la nav (icône cloche + badge count)
- **Toast in-app** pour alertes de fraîcheur < 1h
- **Pas d'email/push** en V2 (V3 si demande)

---

## 7. Modèle de données global

### 7.1 Tables `stats.duckdb` (par joueur)

```sql
-- Streaks actives et historiques
CREATE TABLE streak (
    id              VARCHAR PRIMARY KEY,
    user_id         VARCHAR NOT NULL,
    title_slug      VARCHAR NOT NULL,
    type            VARCHAR NOT NULL,
    started_at      TIMESTAMP NOT NULL,
    current_length  INTEGER NOT NULL DEFAULT 0,
    best_length     INTEGER NOT NULL DEFAULT 0,
    last_increment_at TIMESTAMP,
    threshold       DOUBLE,
    shields_used    INTEGER DEFAULT 0,
    shields_available INTEGER DEFAULT 1,
    status          VARCHAR NOT NULL,
    broken_at       TIMESTAMP
);

-- Records personnels par composante / fenêtre
CREATE TABLE personal_record (
    user_id            VARCHAR NOT NULL,
    title_slug         VARCHAR NOT NULL,
    metric             VARCHAR NOT NULL,
    window             VARCHAR NOT NULL,
    value              DOUBLE NOT NULL,
    achieved_at        TIMESTAMP NOT NULL,
    previous_value     DOUBLE,
    previous_achieved_at TIMESTAMP,
    PRIMARY KEY (user_id, title_slug, metric, window)
);

-- Historique des records (pour timeline)
CREATE TABLE record_history (
    id           VARCHAR PRIMARY KEY,
    user_id      VARCHAR NOT NULL,
    title_slug   VARCHAR NOT NULL,
    metric       VARCHAR NOT NULL,
    window       VARCHAR NOT NULL,
    value        DOUBLE NOT NULL,
    achieved_at  TIMESTAMP NOT NULL
);

-- Milestones débloqués
CREATE TABLE milestone_earned (
    user_id      VARCHAR NOT NULL,
    title_slug   VARCHAR NOT NULL,
    milestone_id VARCHAR NOT NULL,
    earned_at    TIMESTAMP NOT NULL,
    PRIMARY KEY (user_id, title_slug, milestone_id)
);

-- Alertes coach
CREATE TABLE coach_alert (
    id           VARCHAR PRIMARY KEY,
    user_id      VARCHAR NOT NULL,
    title_slug   VARCHAR NOT NULL,
    type         VARCHAR NOT NULL,
    title_key    VARCHAR NOT NULL,
    body_key     VARCHAR NOT NULL,
    payload_json VARCHAR,
    created_at   TIMESTAMP NOT NULL,
    read_at      TIMESTAMP,
    dismissed_at TIMESTAMP
);
```

### 7.2 Tables `metadata.duckdb`

```sql
-- Catalogue milestones (chargé du TOML)
CREATE TABLE milestone_catalog (
    id          VARCHAR PRIMARY KEY,
    title_slug  VARCHAR NOT NULL,
    metric      VARCHAR NOT NULL,
    threshold   DOUBLE NOT NULL,
    title_en    VARCHAR NOT NULL,
    title_fr    VARCHAR NOT NULL,
    icon        VARCHAR,
    condition   VARCHAR  -- description textuelle si complexe
);
```

---

## 8. Architecture technique

### 8.1 Backend Go

```
apps/go-api/internal/
├── progression/
│   ├── streaks/
│   │   ├── types.go            # Streak, StreakType
│   │   ├── evaluator.go        # detect daily/weekly satisfaction
│   │   └── repository.go       # DuckDB persistence
│   ├── records/
│   │   ├── types.go            # PersonalRecord, RecordHistory
│   │   ├── detector.go         # post-match record detection
│   │   └── repository.go
│   ├── milestones/
│   │   ├── types.go
│   │   ├── catalog_loader.go   # TOML loader
│   │   ├── detector.go         # post-match milestone unlocks
│   │   └── repository.go
│   └── coach/
│       ├── types.go            # CoachAlert, AlertType
│       ├── generator.go        # signal detection + alert composition
│       └── repository.go
├── service/
│   └── progression_service.go  # orchestration + cache, expose unified view
└── api/handlers/
    ├── streaks.go              # GET /api/v1/streaks
    ├── records.go              # GET /api/v1/records
    ├── milestones.go           # GET /api/v1/milestones
    └── coach.go                # GET /api/v1/coach/alerts, POST .../{id}/read
```

### 8.2 Hook post-sync

Dans `apps/go-api/internal/sync/engine.go`, après ingestion d'un match :
1. `streaks.Evaluate(userID, titleSlug)` — incrémente / shield / break
2. `records.Detect(userID, titleSlug)` — détecte nouveaux PB
3. `milestones.Detect(userID, titleSlug)` — débloque milestones
4. `coach.Generate(userID, titleSlug)` — génère alertes éligibles

Toutes idempotentes, peuvent être replay sans effet de bord.

### 8.3 Frontend React

```
apps/web/src/features/ascension/   # nouveau dossier (séparé de prestige)
├── components/
│   ├── StreakBadge.tsx           # badge nav L1 (durée actuelle)
│   ├── StreakDashboard.tsx       # vue détaillée streaks
│   ├── RecordsTimeline.tsx       # timeline chronologique
│   ├── RecordsNearMiss.tsx       # records proches
│   ├── MilestonesGrid.tsx        # grille badges
│   └── CoachAlertsCenter.tsx     # centre de notifs
├── hooks/
│   ├── useStreaks.ts
│   ├── useRecords.ts
│   ├── useMilestones.ts
│   └── useCoachAlerts.ts
└── pages/
    └── AscensionDashboard.tsx    # peut être l'onglet par défaut d'Ascension
```

Intégration dans `ObjectifsPage.tsx` (renommé en `AscensionPage.tsx` ?) :
- Nouvel onglet `Progression` à côté de `parcours` et `challenges`
- Ou directement en tête de l'onglet `parcours` après le `PlayerProfileCard` (V1)

### 8.4 Notifications

Centre dans la nav principale (icône cloche, badge count alertes non lues).

---

## 9. Plan d'implémentation

**Branche** : `feat/progression-tracking-ascension` (depuis main, après V1 mergée).

### Découpe en commits

1. `feat(progression): types Go + migrations DuckDB (streak, record, milestone, alert)` (~1j)
2. `feat(progression): streaks evaluator + shields + multiplicateur PP` (~2j)
3. `feat(progression): records detector post-match + timeline` (~1.5j)
4. `feat(progression): milestones catalog TOML + detector` (~1.5j)
5. `feat(progression): coach generator (4-6 types d'alertes positives)` (~2j)
6. `feat(progression): hook post-sync + tests integration` (~1j)
7. `feat(progression): endpoints HTTP + handlers` (~1j)
8. `feat(progression): UI Streaks dashboard + StreakBadge nav` (~1.5j)
9. `feat(progression): UI Records timeline + Near-miss + Milestones grid` (~1.5j)
10. `feat(progression): UI Coach alerts center` (~1j)
11. `chore(progression): i18n manifests + ADR 0013` (~0.5j)

**Total** : ~13.5j → arrondi 10-12j sans i18n et tests exhaustifs.

---

## 10. Risques & décisions ouvertes

### 10.1 Risques

| Risque | Mitigation |
|---|---|
| Notifications coach trop spammy | Dédup 24h par type + cap 3 alertes simultanées non lues + auto-dismiss après 7j |
| Streaks démotivantes si trop dures | Seuils personnels (médianes) + shields mensuels + types `daily_play` minimal (juste jouer) |
| Records "faux positifs" sur petit échantillon | Min 10 matchs sur la fenêtre avant d'enregistrer un PB |
| Milestone catalogue dépassé / trop facile | TOML versionné, possibilité d'ajouter sans migration code |
| Coût compute post-sync | Idempotent + async, replay possible. Worker pool si besoin |

### 10.2 Décisions ouvertes

| Question | Recommandation initiale |
|---|---|
| Streaks par titre ou globales ? | **Par titre** (cohérent avec data model multi-jeux) |
| Public/privé : visible escouade ? | **V2 = privé**. Comparaison escouade en V3 |
| Multiplicateur PP affecte aussi les arcs ? | **Oui**, cohérence simple |
| Streak shield rechargeable par activité ? | **Non** en V2 — 1/mois fixe. Plus simple à comprendre |
| Coach alert push browser/email ? | **Non en V2** — toast + cloche in-app uniquement |
| Premier milestone à viser pour débloquer le système ? | **`matches.100`** (forme la baseline UX) |

---

## 11. Dépendances V1

V2 réutilise :
- `PlayerProfile` service (V1) pour calculer seuils personnels Streaks + Records
- Composantes LUSR + axes radar narrative (déjà calculés)
- Catalogue Templates taggés (`lusr_components` du V1) pour suggestion de défis dans Coach alerts
- Inversion math LUSR (`RequiredCompositeForTier`) pour alerte "proche du sub-tier suivant"
- **`ImprovementCampaign` service (V1)** : le coach lit les campagnes actives pour booster les alertes ciblées + déclencher l'alerte "auto-clôture suggérée" R5 ; les records affichent le contexte campagne quand applicable

V2 ne touche pas :
- L'endpoint `/players/{slug}/profile` (V1 stable)
- Le système Prestige défis/arcs existant (reste compatible)
- La table `improvement_campaign` (lecture seule depuis V2)

---

## 12. Pourquoi cette V2 plutôt que les autres options

- **vs Saisons** : pas de deadline artificielle, pas de pénalité régression, mécaniques alignées sur la répétition (objectif pédagogique principal)
- **vs Streaks seules** : Records & Coach ajoutent la dimension de progression visible (sans quoi les streaks deviennent abstraites)
- **vs Coach seul** : Streaks et Records créent l'attachement émotionnel (perte potentielle = motivation), le coach seul ne suffit pas

Combo cohérent : **Streaks créent la fidélisation, Records créent la fierté, Coach apporte l'intelligence adaptative**.

---

## 13. Références

- [PLAN_PLAYER_PROFILE_ASCENSION.md](PLAN_PLAYER_PROFILE_ASCENSION.md) — V1 (dépendance)
- [PLAN_SEASONS_ASCENSION.md](PLAN_SEASONS_ASCENSION.md) — alternative écartée (DEPRECATED)
- `docs/adr/0004-narrative-engine.md` — 8 rôles + radar 6 axes
- `apps/go-api/internal/analysis/temporal/` — LOWESS pour détection de tendances coach
- `apps/go-api/internal/sync/skill_config.go` — tiers LUSR pour alertes "approche tier"
