# PLAN — Progression Tracking (V2 d'Ascension)

**Date** : 2026-05-13 · **Révisé** : 2026-05-18 (§2bis ajouté, §5 §6 §7 §8 §9 ajustés)
**Statut** : Validé pour implémentation (branche `feat/progression-tracking-ascension`)
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

## 2bis. Alignement avec l'existant (audit pré-implémentation, 2026-05-18)

Le plan initial supposait une base vierge. L'audit avant commit a révélé que **plusieurs fondations sont déjà construites** dans le projet et doivent être réutilisées plutôt que dupliquées. Décisions prises lors de l'audit :

### Notifications coach → réutiliser `player_notifications`

Le plan initial proposait une table `coach_alert` dédiée (cf. §6.3 historique). **Décision** : réutiliser le système `player_notifications` existant dans `shared_social.duckdb` (cf. [internal/notifications/types.go](../apps/go-api/internal/notifications/types.go) et [migration/steps_player_notifications.go](../apps/go-api/internal/migration/steps_player_notifications.go)).

Raisons :
- 18 catégories i18n-keyed déjà déclarées, dont `personal_record` et `threshold_crossed` directement pertinentes
- Centre de notifs in-app (icône cloche, badge count, préférences par catégorie, TTL) — déjà construit
- Hook post-sync `buildPostSyncDeltaHook` déjà câblé, avec TODOs explicites pour `personal_record`, `threshold_crossed`
- Dupliquer cette infra créerait deux centres de notifs UX-équivalents — incohérence client

**Nouvelles catégories à ajouter** (dans `notifications/types.go` + migration de seed prefs) :
- `streak_milestone` — jalon de streak (7j, 14j, 30j)
- `lusr_tier_approach` — approche d'un sub-tier LUSR (< 10 pts μ)
- `record_near_miss` — approche d'un PB (< 5% du record)
- `milestone_near_miss` — approche d'un milestone (< 10% du seuil)
- `comeback_welcome` — reprise après pause > 5j (auto-suggestion clôture campagne aussi)

### Records → étendre `player_records`

La table `player_records` existe déjà dans `shared_social.duckdb` (clé `(xuid, metric)`, pas de fenêtre temporelle). Le plan V2 demande 3 fenêtres (30j / 90j / all_time). **Décision** : étendre la table via migration :

- Ajouter colonne `period VARCHAR NOT NULL DEFAULT 'all_time'`
- Ajouter colonnes `previous_value DOUBLE` et `previous_achieved_at TIMESTAMP`
- PK migrée vers `(xuid, metric, period)`
- Les enregistrements existants sont taggés `period='all_time'` (rétro-compatible)

**Note sur le nommage `period` vs `window`** : le plan d'origine utilisait `window`, mais ce mot est réservé en DuckDB (utilisé pour les window functions `OVER (...)`). Le commit 1 (types + migrations) a donc renommé la colonne et le champ Go correspondant en `period` / `Period` / `RecordPeriod`. Le domaine reste « fenêtre temporelle 30d/90d/all_time ».

### Tables 100% nouvelles

| Table | Localisation | Rôle |
|---|---|---|
| `streak` | stats.duckdb (par joueur) | Streaks actives et historiques |
| `record_history` | stats.duckdb (par joueur) | Timeline chronologique des PB battus |
| `milestone_earned` | stats.duckdb (par joueur) | Milestones débloqués |
| `milestone_catalog` | metadata.duckdb | Catalogue cross-titres chargé du TOML |

Pas de `coach_alert` (réutilise `player_notifications`).

### Hook post-sync — extension, pas réécriture

`buildPostSyncDeltaHook` est déjà câblé. Couche 1/2/3 doivent **étendre** la closure existante (étendre `PlayerSnapshot` avec records moving windows + streak counters, et émettre les nouveaux types de notifs), pas créer un hook parallèle.

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

### 6.3 Modèle de données — réutilise `player_notifications`

**Décision §2bis** : pas de table `coach_alert` dédiée. Les alertes coach sont émises via le package `notifications` existant et stockées dans `player_notifications` (`shared_social.duckdb`).

Mapping type d'alerte → catégorie de notification :

| Type d'alerte | Catégorie de notif (notifications/types.go) | Statut |
|---|---|---|
| Approche d'un record | `record_near_miss` | À ajouter |
| Nouveau record battu | `personal_record` | **Existe déjà** (catégorie déclarée) |
| Approche d'un milestone | `milestone_near_miss` | À ajouter |
| Milestone débloqué | (réutiliser `personal_record` ou nouvelle catégorie `milestone_unlocked`) | À trancher au commit 4 |
| Approche sub-tier LUSR | `lusr_tier_approach` | À ajouter |
| Streak palier (7/14/30j) | `streak_milestone` | À ajouter |
| Reprise après pause | `comeback_welcome` | À ajouter |
| Composante LUSR LOWESS+ sur 14j | `threshold_crossed` | **Existe déjà** |
| Campagne : progression Mann-Whitney | `threshold_crossed` (avec param campaign_id) | **Existe déjà** |

L'i18n des messages passe par les clés `notif.<category>.title` + params (cf. notifications/types.go : `Notification.TitleKey`, `BodyKey`, `Params`).

TTL géré par l'infra notifications (déjà en place).

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

### 7.1 Tables `stats.duckdb` (par joueur) — nouvelles

```sql
-- Streaks actives et historiques (100% nouveau)
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

-- Historique des records (timeline chronologique, 100% nouveau)
CREATE TABLE record_history (
    id           VARCHAR PRIMARY KEY,
    user_id      VARCHAR NOT NULL,
    title_slug   VARCHAR NOT NULL,
    metric       VARCHAR NOT NULL,
    window       VARCHAR NOT NULL,
    value        DOUBLE NOT NULL,
    achieved_at  TIMESTAMP NOT NULL
);

-- Milestones débloqués (100% nouveau)
CREATE TABLE milestone_earned (
    user_id      VARCHAR NOT NULL,
    title_slug   VARCHAR NOT NULL,
    milestone_id VARCHAR NOT NULL,
    earned_at    TIMESTAMP NOT NULL,
    PRIMARY KEY (user_id, title_slug, milestone_id)
);
```

### 7.2 Tables `shared_social.duckdb` — extensions (décision §2bis)

```sql
-- player_records EXISTE DÉJÀ ; migration d'extension :
--   - ajouter colonne `window` (default 'all_time', NOT NULL)
--   - ajouter colonnes previous_value + previous_achieved_at
--   - PK migre vers (xuid, metric, window) ; les lignes existantes
--     restent valides taggées window='all_time'.
ALTER TABLE player_records ADD COLUMN IF NOT EXISTS window VARCHAR NOT NULL DEFAULT 'all_time';
ALTER TABLE player_records ADD COLUMN IF NOT EXISTS previous_value DOUBLE;
ALTER TABLE player_records ADD COLUMN IF NOT EXISTS previous_achieved_at TIMESTAMP;
-- Note : DuckDB ne permet pas de modifier une PK existante en place ;
-- la migration recrée la table avec INSERT INTO ... SELECT FROM ancien.

-- player_notifications EXISTE DÉJÀ ; pas de migration de schéma —
-- seules les NOUVELLES catégories sont ajoutées au seed des
-- préférences (cf. notifications/types.go).
```

### 7.3 Tables `metadata.duckdb` — nouvelles

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
│   │   └── repository.go       # DuckDB persistence (stats.duckdb)
│   ├── records/
│   │   ├── types.go            # PersonalRecord, RecordHistory
│   │   ├── detector.go         # post-match record detection
│   │   └── repository.go       # repo player_records (étendu) + record_history
│   ├── milestones/
│   │   ├── types.go
│   │   ├── catalog_loader.go   # TOML loader (metadata.duckdb)
│   │   ├── detector.go         # post-match milestone unlocks
│   │   └── repository.go
│   └── coach/
│       ├── types.go            # AlertType, NearMissParams (struct legers)
│       ├── generator.go        # signal detection + composition payload
│       └── emitter.go          # délègue à internal/notifications (PAS de repo dédié)
├── service/
│   └── progression_service.go  # orchestration + cache, expose unified view
└── api/handlers/
    ├── streaks.go              # GET /api/v1/streaks
    ├── records.go              # GET /api/v1/records (lit player_records)
    └── milestones.go           # GET /api/v1/milestones
    # Pas de coach.go — le centre de notifs existant gère les alertes coach.
```

### 8.2 Hook post-sync — extension de l'existant

Dans [internal/api/post_sync_deltas.go](../apps/go-api/internal/api/post_sync_deltas.go), la closure `buildPostSyncDeltaHook` reçoit avant/après snapshot et émet des notifs. L'extension V2 enrichit :
1. `PlayerSnapshot` — ajouter records par window (30j/90j/all_time) + streak counters + tendances LOWESS pré-computées
2. `EmitPostSyncDeltas` — appeler en séquence :
   - `streaks.Evaluate(snapshot)` — incrémente / shield / break
   - `records.Detect(snapshot)` — détecte nouveaux PB → émet notif `personal_record`
   - `milestones.Detect(snapshot)` — débloque milestones → émet notif
   - `coach.Generate(snapshot, profile)` — analyse signaux, émet notifs near-miss

Toutes idempotentes, replay sans effet de bord. Le hook existant reste en place — pas de réécriture.

### 8.3 Frontend React

```
apps/web/src/features/ascension/   # nouveau dossier (séparé de prestige)
├── components/
│   ├── StreakBadge.tsx           # badge nav L1 (durée actuelle)
│   ├── StreakDashboard.tsx       # vue détaillée streaks
│   ├── RecordsTimeline.tsx       # timeline chronologique
│   ├── RecordsNearMiss.tsx       # records proches
│   └── MilestonesGrid.tsx        # grille badges
│   # Pas de CoachAlertsCenter — le centre de notifs existant (cloche nav)
│   # affiche déjà les alertes coach via les nouvelles catégories.
├── hooks/
│   ├── useStreaks.ts
│   ├── useRecords.ts
│   └── useMilestones.ts
│   # Pas de useCoachAlerts — useNotifications existant suffit.
└── pages/
    └── AscensionDashboard.tsx    # peut être l'onglet par défaut d'Ascension
```

Intégration dans `ObjectifsPage.tsx` (renommé en `AscensionPage.tsx` ?) :
- Nouvel onglet `Progression` à côté de `parcours` et `challenges`
- Ou directement en tête de l'onglet `parcours` après le `PlayerProfileCard` (V1)

### 8.4 Notifications

Centre dans la nav principale **déjà existant** (icône cloche, badge count, préférences). V2 ne crée pas de centre parallèle — il enrichit le centre existant via de nouvelles catégories.

---

## 9. Plan d'implémentation

**Branche** : `feat/progression-tracking-ascension` (depuis HEAD de `feat/xbox-sso`, qui contient V1).

### Découpe en commits — révisée 2026-05-18 (post-audit §2bis)

1. `chore(plan): align V2 with existing notifications + player_records infra` (~0j, fait)
2. `feat(progression): types Go + migrations DuckDB (streak, record_history, milestone_earned, milestone_catalog, player_records extension)` (~1j)
3. `feat(progression): streaks evaluator + shields + multiplicateur PP` (~2j)
4. `feat(progression): records detector — complète post_sync_deltas TODO + record_history timeline` (~1.5j)
5. `feat(progression): milestones catalog TOML + detector + emission notif` (~1.5j)
6. `feat(progression): coach generator (5-6 types d'alertes near-miss) + nouvelles catégories notif` (~2j)
7. `feat(progression): hook post-sync extension + tests integration` (~1j)
8. `feat(progression): endpoints HTTP /api/v1/{streaks,records,milestones}` (~0.5j) [pas de coach.go]
9. `feat(progression): UI Streaks dashboard + StreakBadge nav` (~1.5j)
10. `feat(progression): UI Records timeline + Near-miss + Milestones grid` (~1.5j)
11. `chore(progression): i18n manifests + ADR 0013` (~0.5j)

**Total révisé** : ~13j → arrondi 10-12j. Économie : pas de centre de notifs coach dédié + pas d'endpoint coach (UI existante suffit).

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
