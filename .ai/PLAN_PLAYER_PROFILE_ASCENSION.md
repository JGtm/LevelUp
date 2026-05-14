# PLAN — Profil Joueur dans Ascension (ex Objectifs)

**Date** : 2026-05-13
**Branche cible** : `feat/player-profile-ascension`
**Statut** : Plan validé en cadrage, prêt pour implémentation

---

## 1. Contexte & objectif

La page "Objectifs" actuelle affiche défis Prestige + arcs, sans portrait du joueur. L'objectif est d'enrichir l'onglet "Mon Parcours" (rename ultérieur TBD) avec un **profil joueur actionnable** qui :

1. **Identifie le joueur** (rôle narratif dominant + radar 6 axes — *qui tu es*)
2. **Situe sa performance** (tier LUSR actuel + composantes LUSR détaillées — *où tu en es*)
3. **Propose une trajectoire de progression** (axes d'amélioration prioritaires + défis/arcs ciblés — *comment progresser*)

En parallèle, la nav L1 `Objectifs` est renommée **`Ascension`**.

---

## 2. Décisions cadre (issues de la discussion 2026-05-13)

| Décision | Choix retenu |
|---|---|
| Niveau d'ambition objectifs | Matching templates Prestige existants (1-clic "Lancer ce défi") |
| Renaming "Mon Parcours" | Reporté (pas d'idée arrêtée) |
| Fenêtre temporelle | 100 derniers matchs |
| Définition forces/faiblesses | Top 3 / Bottom 3 sur les 6 axes radar narrative |
| Min matchs requis | 30. Sous ce seuil → état "données insuffisantes" + compteur |
| Benchmark | **Self-benchmark (top 20% perso)** + **inversion mathématique LUSR**. PAS de benchmark inter-joueurs (pool insuffisant) |
| Vocabulaire UX | "Axes d'amélioration" plutôt que "Faiblesses" |
| Tension défi ponctuel / progression durable | Privilégier templates `rolling_days` + `last_n_matches` `threshold`. Suggérer un **mix défis long-terme + 1 arc**. UX explicite sur la répétition |
| Données complémentaires de profilage | **Inclure First Kill / First Death + Engagement** en Section A (gratuit, sources existantes : `narrative/first_events.go` + `analysis/temporal/engagement_score.go`) |
| Découpe Section A | **Splitter en A1 (radar narrative) + A2 (Style & Discipline)** pour éviter la surcharge du radar à 8 axes |
| Défis conditionnels (`X wins avec stat Y`) | Reportés en **V1.5** (plan séparé) — pas inclus dans cette V1 |
| Mécanique V2 (pousser progression sur durée) | **Streaks + Records & Milestones + Coach positif** — plan dédié [PLAN_PROGRESSION_TRACKING_ASCENSION.md](PLAN_PROGRESSION_TRACKING_ASCENSION.md). Saisons écartées (cf. [PLAN_SEASONS_ASCENSION.md](PLAN_SEASONS_ASCENSION.md) DEPRECATED) |
| **Campagne d'amélioration** (boucle profil → objectif → suivi) | **Inclus en V1** §4.5. Mini-objectif personnel volontaire sur 1 axe, snapshot + delta lissé LOWESS, sans deadline ni pénalité. Reboucle la demande d'origine ("à partir du profil, se mettre des objectifs et suivre"). +2-3j d'effort sur V1 |
| **Le nouveau système n'est jamais imposé** | Le mode `libre` de `CreateChallengeForm` reste 100% disponible et inchangé. Le joueur peut toujours créer un défi "juste pour le fun" sans passer par le profil ni par une campagne. Profil joueur + Campagne = **couche additionnelle opt-in**, jamais bloquante |

---

## 3. Audit du catalogue templates.toml

### 3.1 Inventaire (27 templates)

| Cadence | Window | Count | Caractère |
|---|---|---|---|
| daily | session=1 | 9 | Ponctuel (one-shot) |
| weekly | session=3 | 9 | Semi-long-terme |
| monthly | rolling_days=30 | 9 | Long-terme cumulatif |

### 3.2 Couverture par composante LUSR

| Composante LUSR | Poids | Templates directs | Verdict |
|---|---:|---:|---|
| Kills vs Expected | 27% | 5 (kda, kdr, kve, killing_spree…) | OK |
| Deaths vs Expected | 24% | 0-1 (uniquement kdr indirect) | **GAP CRITIQUE** |
| Win Factor | 5% | 3 (win_rate, csr_floor, ranked_wins) | OK |
| Damage Efficiency | 10% | 1 (damage_per_min) | Faible |
| Accuracy Delta | 10% | 2 (accuracy_session, accuracy_3sessions) | OK |
| Medal Exploit | 4% | 7 (headshots, power_weapon, killtaculars…) | Surreprésenté |
| Offensive Conversion | 16% | 3 (personal_score, damage_per_min) | OK |
| Defensive Resistance | 6% | 0 | **GAP TOTAL** |

### 3.3 Couverture long-terme

| Composante | Templates rolling_days threshold |
|---|---|
| Kills vs Expected | 0 (seulement weekly=3sessions) |
| Accuracy | 0 (seulement weekly=3sessions) |
| Damage Efficiency | 0 |
| Deaths vs Expected | 0 |

**Conclusion** : le catalogue est conçu pour des **exploits ponctuels** plutôt que pour un coaching de progression durable. Plusieurs ajouts sont nécessaires (cf. §6.1).

### 3.4 Audit des 4 arcs preset

| Arc | Composantes ciblées | Verdict |
|---|---|---|
| The Slayer | Kills vs Expected (27%) | Bon ciblage offensif |
| The Support | Offensive Conversion (16%) + Win Factor (5%) | Cohérent |
| The Consistent | Multi (Win Factor + KvE) | Moyen, dispersé |
| The Explorer | Aucune composante LUSR | Identitaire pur, pas progression |

**Manque** : pas d'arc orienté Accuracy/précision, ni Survival/Defensive.

---

## 4. Architecture proposée

### 4.1 Section A1 — "Qui tu es" (radar narrative)

**Source** : `narrative.IdentifyImpactRoles()` + `narrative.ComputeParticipationProfile()` (existants).

- Rôle dominant + secondaire (agrégés sur 100 derniers matchs)
- Phrase éditoriale i18n type : *"Tu joues comme un Anchor avec une tendance Clutch Master"*
- Radar 6 axes (Combat / Survie / Support / Score / Objectif / Impact)
- 3 forces (top 3 axes du radar)
- 3 axes d'amélioration (bottom 3 axes du radar)

**Renaming UX** : "Faiblesses" → **"Axes d'amélioration"**.

### 4.2 Section A2 — "Style & Discipline" (FK/FD + Engagement)

**Sources existantes** :
- `apps/go-api/internal/analysis/narrative/first_events.go` — agrégats First Kill / First Death (events filmés)
- `apps/go-api/internal/analysis/temporal/engagement_score.go` — score d'engagement temporel
- `apps/go-api/internal/sync/engagement.go` + repo `platform/duckdb/engagement_score_repo.go`

**Affichage** (deux mini-cartes côte à côte sous le radar A1) :

**Carte "Style offensif" (FK/FD)** :
- Ratio `FK / FD` sur 100 derniers matchs
- Phrase d'interprétation i18n :
  - FK élevé + FD faible → *"Finisseur opportuniste"*
  - FK faible + FD élevé → *"Trop avancé, gère mieux ton premier engagement"*
  - FK et FD élevés → *"Player ultra-engagé, canalise ton agressivité"*
  - FK et FD faibles → *"Joueur d'attente, cherche plus l'initiative"*
- Mini-sparkline 30 jours (tendance)

**Carte "Discipline" (Engagement)** :
- Score d'engagement courant (échelle 0-100 ou tier qualitatif via `engagement_score.go`)
- Régularité (matchs/jour moyen, gap max sans jouer)
- Phrase coaching : si engagement faible → *"Joue plus régulièrement avant de viser des objectifs de perf"* ; si engagement fort → *"Bonne régularité, profil idéal pour viser Diamond IV"*

**Pourquoi A2 séparée du radar** : FK/FD et engagement ne mesurent pas la même chose que les 6 axes narrative (qualitatif par match) — étendre le radar à 8 axes dilue la lecture. Bloc dédié plus clair.

### 4.3 Section B — "Où tu en es" (performance LUSR)

**Source** : `match_skill_rank` (dernier match) + recompute composite par composante.

- Badge tier (ex: Diamond III, sub-tier 3/6) + μ actuel + σ
- Barre de progression vers sub-tier suivant (μ requis = current_floor + 33.3 pts)
- 8 composantes LUSR en bar chart compact, avec **2 overlays** :
  - Ta moyenne actuelle (100 derniers matchs)
  - **Ton top 20% personnel** (ton plafond déjà atteint = self-benchmark)
- Indicateur de tendance LOWESS (slope sur composite_score, 30 derniers vs 30 précédents)

### 4.4 Section C — "Comment progresser" (coaching actionnable)

**Phrase d'introduction (narrative rang → leviers → campagne)** :

Le bloc s'ouvre par un paragraphe éditorial qui ancre le coaching au **rang LUSR** comme horizon, sans en faire un objectif fragile (cf. note ci-dessous). Exemple de copy i18n :

> *"Tu es **Diamond III** (sub-tier 3/6). Pour viser **Diamond IV**, tes 2 leviers les plus rentables sont :*
> *• **Deaths vs Expected** — gain estimé +18 pts μ*
> *• **Kills vs Expected** — gain estimé +12 pts μ*
> *Démarre une campagne sur le plus important pour transformer ce levier en routine d'entraînement."*

Cette narrative explicite **rang = horizon, axes = marches, campagne = effort**.

**Note importante — pourquoi pas de "campagne sur rang"** :
Le rang LUSR est l'**horizon visible**, pas un objectif de campagne. Raisons : (a) σ élevé rend la promotion fortement bruitée ; (b) le matchmaking ramène le joueur vers son "vrai" niveau, donc une consolidation N matchs dépend autant de l'extérieur que de l'effort ; (c) CSR Halo natif fait déjà ça ; (d) le rang ne dit pas QUOI travailler — les axes oui. Donc on garde rang en Section B + alertes Coach V2 sur approche tier ; les campagnes restent sur axes (actionnables).

**Algorithme de levier** :
- Pour chaque composante : `lever_value = (1 - current_score) × weight_lusr`
- Top 2 leviers = composantes les plus rentables à améliorer

**Pour le tier suivant** (inversion math LUSR) :
- μ cible = floor(sub_tier+1) → composite_score moyen requis (inversion `trueskillUpdate`)
- Écart sur chaque composante vs ce composite cible
- **Gain estimé pts μ par composante** : différentiel × poids × delta requis — affiché dans la copy d'introduction ci-dessus

**Affichage** :
- 2 cards "Levier prioritaire" avec phrase coaching type :
  > *"Augmenter tes 'Kills vs Expected' de 0.52 → 0.65 te ferait gagner ~15 pts LUSR vers Diamond IV. Ton top 20% est à 0.78 — c'est faisable, tu l'as déjà fait X fois."*
- 3 défis suggérés (mix templates `rolling_days threshold` + `last_n_matches threshold`) avec :
  - Bouton "Lancer ce défi" (réutilise flow `CreateChallenge` mode automatique)
  - Compteur de séries historique : *"Tu as complété ce template 2 fois sur les 90 derniers jours"*
  - Copy éducative : *"Vise 3 réussites ce mois — la progression vient de la répétition"*
- 1 arc preset ciblé sur l'axe si dispo (sinon afficher "arc à venir")
- Tendance temporelle 90 jours sur l'axe (mini sparkline)
- **CTA "Démarrer une campagne d'amélioration sur cet axe"** (cf. §4.5) — active un focus persistant lié aux défis qu'on lance ensuite

### 4.5 Campagne d'amélioration (boucle profil → objectif → suivi)

**Concept** : un mini-objectif personnel volontairement activé par le joueur sur **1 axe identifié comme amélioration prioritaire**. Pas de deadline, pas de pénalité, pas de reward externe — la progression visible **est** sa propre récompense.

C'est la pièce qui rebouclera le parcours initial : *à partir du profil joueur → identifier un point faible → se mettre un objectif → suivre dans le temps*.

**Principe clé — la campagne n'est jamais obligatoire** :
- Le joueur peut toujours créer un défi **libre** depuis l'onglet `challenges` (flow Prestige existant, mode `libre`) sans déclencher de campagne ni passer par le profil
- La campagne est une **opt-in** : on la propose en CTA depuis Section C, jamais imposée
- Un défi créé en mode libre n'a pas de `campaign_id` (NULL) — il existe à part, juste pour le fun
- L'UX doit clairement signaler que les défis "fun" et les défis "campagne" coexistent sans hiérarchie de valeur

#### 4.5.1 Modèle

- Le joueur démarre une campagne depuis Section C (CTA *"Démarrer une campagne sur cet axe"*)
- Snapshot capturé : valeur composante LUSR + axe radar + sample-size + playlist_group cible (optionnel)
- Affichage permanent en tête de la page Ascension : *"Campagne en cours : Survie, démarrée le 13/05, +0.04 sur 32 matchs"*
- Les défis lancés ensuite sont auto-liés à la campagne (tagging `campaign_id`)
- Le joueur peut **fermer**, **mettre en pause**, ou **changer d'axe** à tout moment, sans conséquence

#### 4.5.2 Différences fondamentales avec Saisons (écartées)

| Critère | Saison (DEPRECATED) | Campagne (retenue) |
|---|---|---|
| Activation | Imposée par calendrier | **Volontaire, à tout moment** |
| Durée | 90j fixes | **Indéterminée, fermable à tout moment** |
| Granularité | Multi-arcs / LUSR global | **1 axe ciblé** |
| Pénalité régression | Oui (consistency_mult ↓) | **Aucune** |
| Échec possible | Oui (saison ratée) | **Non** — pause/abandon sans conséquence |
| Reward de fin | Badge + multiplicateur PP | **Aucun** — la progression visible suffit (renforcée par Records V2) |

Esprit : "j'apprends le japonais sur Duolingo" — pas de note de fin, juste une trajectoire personnelle qu'on suit volontairement.

#### 4.5.3 Cinq raffinements algorithmiques

Pour que le suivi soit **fiable**, **logique** et **honnête** :

**R1. Delta lissé LOWESS** (pas delta brut)
- Le delta brut `current - snapshot` est trompeur (variance naturelle ±0.05 sur 30 matchs)
- Afficher le **delta LOWESS** (courbe lissée via `analysis/temporal/`) — robuste au bruit
- Affichage du delta uniquement après **min 20 matchs post-snapshot** sur la playlist cible. Sinon : *"Joue encore N matchs sur ta playlist cible pour voir ta tendance"*

**R2. Phrasing strict — no causalité revendiquée**
- L'algo **ne peut pas** prouver que *"le défi t'a fait progresser"* — c'est de la corrélation temporelle
- Copy autorisée : *"Pendant ta campagne"*, *"Depuis le démarrage"*, *"Sur la période"*
- Copy interdite : *"Grâce à"*, *"Le défi a amélioré"*, *"Tu progresses à cause de"*
- Cohérent avec la pédagogie "la répétition fait l'efficacité, pas une coche"

**R3. Filtre playlist optionnel**
- Au démarrage, le joueur choisit (ou laisse "all") un `playlist_group` cible
- Le tracking ne compte que les matchs sur ce groupe
- Évite les faux signaux liés à un changement de mode de jeu pendant la campagne
- UX : *"32 matchs sur Ranked Slayer ces 30 derniers jours — base solide"* ou *"Seulement 5 matchs sur ta playlist cible, joue plus en Ranked Slayer pour mesurer ta campagne"*

**R4. Test statistique pour milestone "progression confirmée"**
- Test **Mann-Whitney U** entre distribution composante pré-snapshot (100 matchs) vs post-snapshot (20+ matchs)
- Déclenchement d'un toast/notif *"Progression confirmée statistiquement (p < 0.05)"* uniquement quand le test passe
- Évite le spam de célébrations bruitées, rend les vraies confirmations gratifiantes par rareté

**R5. Auto-suggestion de clôture** (pas auto-fermeture)
- Si l'axe sort du bottom 3 du radar OU si plateau (60j sans variation significative LOWESS) :
- Suggérer *"Ton axe Survie n'est plus dans tes axes prioritaires. Clore cette campagne et démarrer une nouvelle sur Combat ?"*
- Le joueur garde le contrôle ; jamais d'auto-fermeture silencieuse

#### 4.5.4 Deux gardes-fous UX

**G1. Ne pas surpromettre**
- Copy mesurée. Pas de *"on garantit ta progression"* ni *"coach scientifique"*
- Phrase d'accueil type : *"On t'aide à voir ta trajectoire — pas à la garantir. La progression vient de toi."*

**G2. Pas de fausse précision**
- Arrondir à **2 décimales max** (`+0.04`, pas `+0.0347`)
- Toujours afficher le **sample size** et la **période** à côté du delta : *"+0.04 sur 32 matchs depuis le 13 mai"*
- Le joueur peut juger lui-même de la solidité

#### 4.5.5 Affichage

**En tête de la page Ascension** (au-dessus de PlayerProfileCard) :
```
[Campagne en cours : Survie]
Démarrée le 13/05 — 32 matchs sur Ranked Slayer
Snapshot 'Deaths vs Expected' : 0.52
Actuel (lissé)                : 0.56   (+0.04)
✓ Progression confirmée (p<0.05) — 2 défis réussis sur 3 lancés
[Voir détail]  [Pause]  [Clore]
```

**État "données insuffisantes"** (< 20 matchs post-snapshot) :
```
Campagne Survie — démarrée le 13/05
Joue encore 12 matchs sur Ranked Slayer pour voir ta tendance.
2 défis en cours · 1 réussi
```

**État "plateau / sortie bottom 3"** :
```
Campagne Survie — 87 jours, +0.06 confirmé
Cet axe n'est plus dans tes axes prioritaires. 
[Clore et démarrer Combat]  [Continuer]
```

### 4.6 État "données insuffisantes" (< 30 matchs)

- Placeholder éditorial avec compteur (ex: *"Joue encore 17 matchs pour débloquer ton profil"*)
- Affichage de stats agrégées brutes (matchs joués, modes touchés)
- Bouton "Voir les défis disponibles"

---

## 5. Architecture technique

### 5.1 Backend Go

#### Tagging templates

Étendre `ChallengeTemplate` dans `apps/go-api/internal/prestige/types.go` :

```go
type ChallengeTemplate struct {
    // ... champs existants
    LUSRComponents []string `toml:"lusr_components"` // ["kills_vs_expected", "deaths_vs_expected"]
    RadarAxes      []string `toml:"radar_axes"`      // ["combat", "survival"] — optionnel
    IsLongTerm     bool     `toml:"is_long_term"`    // true si window=rolling_days OR last_n_matches threshold
}
```

Loader TOML : `apps/go-api/internal/prestige/template_loader.go` (ou équivalent — à vérifier au démarrage).

#### Nouveau service

`apps/go-api/internal/service/player_profile_service.go` (~300L max) :

```go
type PlayerProfile struct {
    HasEnoughData     bool
    MatchesAnalyzed   int
    // Section A1 — radar narrative
    DominantRole      narrative.ImpactRole
    SecondaryRole    narrative.ImpactRole
    RadarAxes        narrative.ParticipationProfile  // 6 axes
    Strengths        []RadarAxisInsight              // top 3
    ImprovementAreas []RadarAxisInsight              // bottom 3
    // Section A2 — style & discipline
    StyleSignature   StyleSignature                  // FK/FD + interprétation
    EngagementSnap   EngagementSnapshot              // score + régularité
    // Section B — LUSR
    SkillRating      SkillRatingSnapshot             // tier, sub_tier, mu, sigma
    LUSRComponents   []LUSRComponentBreakdown        // 8 composantes
    // Section C — coaching
    Leverages        []ProgressionLeverage           // top 2
    SuggestedChallenges []SuggestedChallenge         // 3 templates + 1 arc
}

type StyleSignature struct {
    FirstKillCount    int     // sur window
    FirstDeathCount   int     // sur window
    FKFDRatio         float64 // FK / max(FD, 1)
    StyleKey          string  // "opportunistic_finisher" | "overextended" | "hyper_engaged" | "passive"
    Trend30d          []float64 // sparkline FK-FD différentiel par jour
}

type EngagementSnapshot struct {
    Score             float64 // 0-100 (via engagement_score.go)
    Tier              string  // "low" | "regular" | "high" | "intense"
    MatchesPerDayAvg  float64
    MaxGapDays        int     // plus grande pause sur la fenêtre
    RegularityCoach   string  // i18n key
}

type LUSRComponentBreakdown struct {
    Name          string  // "kills_vs_expected"
    Weight        float64 // 0.27
    CurrentAvg    float64 // 0.52
    PersonalTop20 float64 // 0.78
    TargetForTier float64 // 0.65 (inversion math)
    Trend         float64 // slope sur 30j (positif = amélioration)
}

type ProgressionLeverage struct {
    Component       string
    LeverageValue   float64  // (1 - current) × weight
    NarrativeAxes   []string // axes radar associés
    CoachingMessage string   // i18n key
}

type SuggestedChallenge struct {
    TemplateID     string
    TargetTier     string   // 'normal'|'heroic'|'legendary'
    HistoricalStreak int    // nb complétions sur 90j
    IsArcStep      bool
    ArcID          string   // si IsArcStep
}

// Campagne d'amélioration (§4.5)
type ImprovementCampaign struct {
    ID                string
    UserID            string
    TitleSlug         string
    Axis              string        // "survival" | "combat" | "kills_vs_expected" | ...
    AxisKind          string        // "radar" (1 des 6) | "lusr_component" (1 des 8)
    StartedAt         time.Time
    EndedAt           *time.Time
    Status            CampaignStatus // active | paused | completed | abandoned
    PlaylistGroup     string        // "all" | "arena_slayer" | "btb" | ...
    SnapshotValue     float64
    SnapshotSample    int           // taille échantillon au démarrage (matchs)
    CurrentValueRaw   float64       // dernière valeur brute
    CurrentValueLOWESS float64      // valeur lissée (à afficher)
    MatchesSinceStart int           // sur la playlist cible
    LastEvaluatedAt   time.Time
    MannWhitneyP      *float64      // p-value test stat (nil si < 20 matchs)
    ProgressionConfirmed bool       // true quand p < 0.05
    LinkedChallengeIDs []string     // défis tagués campaign_id = this.ID
    AutoClosureSuggested bool       // R5 : à afficher avec CTA "Clore"
    AutoClosureReason string        // "axis_no_longer_priority" | "plateau_60d"
}

type CampaignStatus string
const (
    CampaignActive    CampaignStatus = "active"
    CampaignPaused    CampaignStatus = "paused"
    CampaignCompleted CampaignStatus = "completed"
    CampaignAbandoned CampaignStatus = "abandoned"
)
```

**Méthodes principales** :
- `BuildProfile(xuid, titleSlug, window=100)` → orchestration
- `aggregateNarrative(matches)` → narrative engine (rôles + radar 6 axes)
- `computeStyleSignature(xuid, window)` → FK/FD ratios + StyleKey via `narrative/first_events.go`
- `computeEngagement(xuid, window)` → score + tier + régularité via `analysis/temporal/engagement_score.go`
- `computeLUSRComponents(xuid, window)` → 8 scores moyens + top 20% perso
- `identifyLeverages(components, currentTier, targetTier)` → top 2 leviers
- `selectSuggestedChallenges(leverages, templates, history)` → matching catalogue

**Service Campagne** (`apps/go-api/internal/service/improvement_campaign_service.go`, ~250L max) :
- `StartCampaign(userID, titleSlug, axis, axisKind, playlistGroup)` → snapshot + insert DB
- `EvaluateCampaign(campaignID)` → recompute LOWESS, Mann-Whitney U, AutoClosureSuggested. Idempotent.
- `LinkChallengeToCampaign(challengeID, campaignID)` → tag challenge
- `PauseCampaign(id)` / `CloseCampaign(id, reason)` / `AbandonCampaign(id)`
- `GetActiveCampaign(userID, titleSlug)` → 1 max active à la fois par titre

**Hook post-sync** : après ingestion d'un match, appeler `EvaluateCampaign` sur la campagne active du joueur si elle existe. Coût léger (1 LOWESS + 1 test stat sur petit échantillon).

**Table DuckDB** (`stats.duckdb` par joueur) :

```sql
CREATE TABLE improvement_campaign (
    id                      VARCHAR PRIMARY KEY,
    user_id                 VARCHAR NOT NULL,
    title_slug              VARCHAR NOT NULL,
    axis                    VARCHAR NOT NULL,
    axis_kind               VARCHAR NOT NULL,         -- 'radar' | 'lusr_component'
    started_at              TIMESTAMP NOT NULL,
    ended_at                TIMESTAMP,
    status                  VARCHAR NOT NULL,
    playlist_group          VARCHAR NOT NULL DEFAULT 'all',
    snapshot_value          DOUBLE NOT NULL,
    snapshot_sample         INTEGER NOT NULL,
    current_value_raw       DOUBLE,
    current_value_lowess    DOUBLE,
    matches_since_start     INTEGER DEFAULT 0,
    last_evaluated_at       TIMESTAMP,
    mann_whitney_p          DOUBLE,
    progression_confirmed   BOOLEAN DEFAULT FALSE,
    auto_closure_suggested  BOOLEAN DEFAULT FALSE,
    auto_closure_reason     VARCHAR
);

CREATE INDEX idx_campaign_user_title ON improvement_campaign(user_id, title_slug, status);

-- Lien défi ↔ campagne (extension table challenge existante)
ALTER TABLE challenge ADD COLUMN campaign_id VARCHAR;
CREATE INDEX idx_challenge_campaign ON challenge(campaign_id);
```

**Endpoints HTTP campagne** :
```
POST   /api/v1/campaigns                  → StartCampaign (body: axis, axis_kind, playlist_group)
GET    /api/v1/campaigns/active           → GetActiveCampaign (1 par titre)
GET    /api/v1/campaigns/{id}             → détail campagne + delta lissé + défis liés
POST   /api/v1/campaigns/{id}/pause       → PauseCampaign
POST   /api/v1/campaigns/{id}/close       → CloseCampaign
POST   /api/v1/campaigns/{id}/abandon     → AbandonCampaign
```

#### Endpoint HTTP profil

`apps/go-api/internal/api/handlers/player_profile.go` :

```
GET /api/v1/players/{slug}/profile?title_slug=halo_infinite&window=100
→ 200 PlayerProfile JSON
→ 404 si player inconnu
→ 422 si données insuffisantes (avec `matches_played` + `required`)
```

Registration dans router : à ajouter là où sont registrés `/prestige/*` et `/synthesis/*`.

#### Inversion math LUSR

Implémenter dans `apps/go-api/internal/analysis/skill_rating.go` :

```go
// RequiredCompositeForTier retourne le composite_score moyen
// nécessaire pour stabiliser μ au floor du tier cible.
func RequiredCompositeForTier(currentMu, targetMu, sigma float64) float64
```

Approche : inversion numérique de `trueskillUpdate` (binary search sur composite_score ∈ [0,1] jusqu'à ce que μ_stable soit proche de targetMu). Précision 0.01 suffit.

### 5.2 Frontend React

#### Rename nav

`apps/web/src/components/shell/shellNavigation.ts` : remplacer `Objectifs` → `Ascension` dans `PLAYER_PRIMARY_NAV_ITEMS` (label_fr et label_en).

#### Nouveau composant

`apps/web/src/features/prestige/components/PlayerProfileCard.tsx` (~250L max, à découper si nécessaire) :

Composants enfants :
- `IdentitySection.tsx` (Section A1 — rôles + radar ECharts)
- `StyleDisciplineSection.tsx` (Section A2 — 2 cartes FK/FD + Engagement)
- `PerformanceSection.tsx` (Section B — tier badge + 8 composantes bar chart)
- `ProgressionSection.tsx` (Section C — leviers + défis suggérés + arc + CTA "Démarrer campagne")
- `CampaignTracker.tsx` (Section §4.5 — affichage permanent campagne active, gestion pause/clôture, état "donnees insuffisantes" intra-campagne)
- `InsufficientDataPlaceholder.tsx` (état < 30 matchs)

Intégration : insérer en **tête** de l'onglet `parcours` dans `apps/web/src/features/prestige/ObjectifsPage.tsx`, **avant** `StatsGlobales`. **Au-dessus** du `PlayerProfileCard`, placer le `CampaignTracker` (sticky) si une campagne est active.

#### Hooks

- `apps/web/src/features/prestige/hooks/usePlayerProfile.ts` (React Query, stale 5min)
- `apps/web/src/features/prestige/hooks/useActiveCampaign.ts` (React Query, stale 1min, invalide post mutation)
- `apps/web/src/features/prestige/hooks/useCampaignMutations.ts` (start/pause/close/abandon)

#### Réutilisation flow CreateChallenge

**Le flow existant `CreateChallengeForm` n'est pas modifié dans son comportement par défaut** :
- Mode `libre` (joueur définit tout) : inchangé, `campaign_id` = NULL
- Mode `hybride` (suggestion template + ajustement) : inchangé, `campaign_id` = NULL
- Mode `automatique` (3 templates au choix) : inchangé, `campaign_id` = NULL

**Nouveauté non bloquante** : `CreateChallengeForm` accepte deux props optionnelles :
- `prefilledTemplateId` (déjà éventuellement présent) — pré-sélection du template depuis Section C
- `attachToCampaignId` — pré-link à une campagne active

Ces props sont utilisées **uniquement** quand le défi est lancé depuis le profil (Section C). Tout autre point d'entrée (onglet `challenges`, autres CTAs Prestige existants) ne les passe pas → comportement identique à aujourd'hui.

CTA "Démarrer une campagne sur cet axe" dans Section C ouvre une mini-modale :
- Choix du `playlist_group` (par défaut "all", recommandé : le playlist_group dominant du joueur sur la fenêtre)
- Récap du snapshot prévu
- Phrase pédagogique R2 : *"On t'aide à voir ta trajectoire, pas à la garantir."*
- **Option "Skip — créer juste un défi libre"** : permet de ressortir vers le flow `CreateChallengeForm` sans démarrer de campagne

### 5.3 Tagging du catalogue

Annoter chaque template existant avec :
- `lusr_components` (1-2 composantes)
- `is_long_term` (true si window=rolling_days ou (window=session ET value>=3))

Exemple :

```toml
[[templates]]
id = "halo_infinite.daily.kda_session"
# ... existant
lusr_components = ["kills_vs_expected", "deaths_vs_expected"]
is_long_term = false

[[templates]]
id = "halo_infinite.monthly.matches_played"
# ... existant
lusr_components = []  # neutre (engagement, pas perf)
is_long_term = true
```

---

## 6. Enrichissement du catalogue (gap closing)

### 6.1 Nouveaux templates à ajouter

| Nouveau template | Composante LUSR | Cadence/Window | Justification |
|---|---|---|---|
| `halo_infinite.monthly.deaths_vs_expected` | Deaths vs Expected (24%) | rolling_days=30, threshold | Couvre le gap critique. Cible : ratio deaths_expected/deaths >= 1.05/1.15/1.25/1.40 |
| `halo_infinite.weekly.deaths_vs_expected` | Deaths vs Expected | session=3, threshold | Pendant court terme |
| `halo_infinite.monthly.dmg_taken_per_death` | Defensive Resistance (6%) | rolling_days=30, threshold | Couvre le gap. Cible : DmgTaken/Deaths (plus haut = mieux résister) |
| `halo_infinite.monthly.accuracy_30d` | Accuracy Delta (10%) | rolling_days=30, threshold | Habitude vs ponctuel |
| `halo_infinite.monthly.kve_30d` | Kills vs Expected (27%) | rolling_days=30, threshold | Habitude vs ponctuel |

### 6.2 Nouveaux arcs preset

| Arc | Composante cible | Steps |
|---|---|---|
| `halo_infinite.marksman` | Accuracy Delta | accuracy_session(N) → accuracy_3sessions(H) → accuracy_30d(L) |
| `halo_infinite.survivor` | Deaths vs Expected + Defensive Resistance | kdr_session(N) → deaths_vs_expected_3sessions(H) → deaths_vs_expected_30d(L) |

---

## 7. Plan d'implémentation

**Branche unique** : `feat/player-profile-ascension` (depuis `feat/stats-page-rework` ou `main` selon le merge en cours — à valider avant création).

### Découpe en commits

**Commit 1** — `feat(prestige): tagger templates avec lusr_components + is_long_term`
- Étendre `ChallengeTemplate` (Go struct + loader TOML)
- Annoter les 27 templates existants
- Tests unitaires loader + chargement TOML
- Migration metadata.duckdb si la table `challenge_template` doit recevoir les nouvelles colonnes

**Commit 2** — `feat(prestige): enrichir catalogue templates + arcs`
- Ajouter les 5 nouveaux templates (§6.1)
- Ajouter les 2 nouveaux arcs preset (§6.2)
- Validation des targets monotones

**Commit 3** — `feat(skill): RequiredCompositeForTier (inversion LUSR)`
- Fonction d'inversion math dans `analysis/skill_rating.go`
- Tests unitaires : pour chaque tier, vérifier que stable_mu ≈ target
- Doc inline sur la méthode (binary search)

**Commit 4** — `feat(profile): service PlayerProfile + endpoint Go`
- `service/player_profile_service.go` (orchestration narrative + LUSR + matching)
- `handlers/player_profile.go` (HTTP)
- Tests d'intégration handler avec données seed
- Doc endpoint dans README ou doc dédiée

**Commit 5** — `feat(campaign): ImprovementCampaign service + endpoints + table DuckDB`
- Migration table `improvement_campaign` + ALTER `challenge` ADD `campaign_id`
- `service/improvement_campaign_service.go` (start/evaluate/pause/close/abandon)
- LOWESS + Mann-Whitney U sur composante post-snapshot (réutilise `analysis/temporal/`)
- Détection AutoClosureSuggested (sortie bottom 3 axes OU plateau 60j)
- Hook post-sync : `EvaluateCampaign` sur la campagne active
- Handlers HTTP `POST /campaigns`, `GET /campaigns/active`, `POST /campaigns/{id}/{pause,close,abandon}`
- Tests unitaires + intégration

**Commit 6** — `feat(profile): composant PlayerProfileCard frontend`
- 5 sous-composants (Identity, StyleDiscipline, Performance, Progression, InsufficientData)
- Hook `usePlayerProfile`
- Réutilisation CreateChallenge avec prefill `template_id` + `campaign_id` si applicable
- Tests Vitest (rendering + interaction bouton "Lancer")

**Commit 7** — `feat(campaign): UI CampaignTracker + mini-modale démarrage`
- Composant `CampaignTracker.tsx` (sticky en tête, gestion pause/clôture, état intra-campagne "données insuffisantes")
- Mini-modale "Démarrer une campagne" depuis CTA Section C : choix playlist_group + récap snapshot + phrase pédagogique R2
- Hooks `useActiveCampaign`, `useCampaignMutations`
- Tests Vitest (rendering selon états : data_insufficient / active / plateau / auto_closure_suggested)

**Commit 8** — `feat(nav): rename Objectifs → Ascension`
- `shellNavigation.ts`
- i18n manifests si label hardcodé ailleurs (rechercher "Objectifs" dans le code frontend)
- Test snapshot nav

**Commit 9** — `chore(i18n): manifests profil joueur + campagne FR/EN`
- Phrases éditoriales (identité, leviers, coaching, campagne)
- Inclut R2 strict (no causality) : copy interdite documentée dans manifest comments
- Lint manifests
- Linter custom OK

**Commit 10** *(optionnel)* — `docs: ADR profil joueur + campagne d'amélioration`
- `docs/adr/0012-player-profile-ascension.md`
- Référence ADR 0004 (narrative) + skill rating
- Justification self-benchmark + inversion LUSR
- Justification Campagne (vs Saison écartée) + 5 raffinements algorithmiques

### Ordre de merge

PR unique sur la branche. Review possible commit par commit pour faciliter la relecture.

---

## 8. Risques & points ouverts

### Risques techniques

| Risque | Mitigation |
|---|---|
| Inversion LUSR numériquement instable | Bornes [0,1] sur composite + fallback "approximation seuil" si binary search ne converge pas en 20 itérations |
| Self-benchmark peu pertinent si joueur médiocre | Si top 20% perso < seuil tier suivant : afficher "tu n'as pas encore atteint le niveau requis, vise X" plutôt que "tu y es déjà allé" |
| Composantes LUSR non disponibles pour anciens matchs | Recalculer à la volée depuis `shared.match_participants` si manquantes (lazy) ou batch one-shot |
| Templates `rolling_days` peu nombreux après tagging | Commits 1+2 ajoutent suffisamment de templates long-terme pour avoir 1-2 candidats par composante |
| Coût endpoint `/profile` (narrative + LUSR + breakdown) | Cache 5min côté service (TTL court car données rafraîchies par sync) |
| **Campagne : delta bruité sur petit sample** (R1) | Afficher uniquement le **delta LOWESS** + n'afficher qu'après min 20 matchs post-snapshot sur playlist cible. Sinon état "données insuffisantes" intra-campagne |
| **Campagne : surpromesse causale** (G1+R2) | Copy strictement neutre : *"pendant ta campagne"*, *"depuis le démarrage"*. Audit manifests i18n au commit 9 pour bannir *"grâce à"*/*"le défi a amélioré"* |
| **Campagne : fausse précision** (G2) | Arrondi 2 décimales max + affichage sample-size + période à côté du delta |
| **Campagne : faux signaux par changement de playlist** (R3) | Filtre `playlist_group` au démarrage, tracking restreint à ce groupe |
| **Campagne : célébrations spammées sur bruit** (R4) | Toast "Progression confirmée" uniquement quand Mann-Whitney U p < 0.05. Échec silencieux sinon |
| **Campagne fantôme (ouverte 6 mois sans activité)** (R5) | Auto-suggestion clôture (UX) après plateau 60j ou sortie bottom 3. Jamais d'auto-fermeture. Joueur seul décide |

### Points ouverts (non bloquants pour V1)

- **Naming "Mon Parcours"** : reporté. À reprendre après V1 si idée émerge.
- **Évolution profil dans le temps** : pas de snapshots persistés. V2 si demande.
- **Benchmark inter-joueurs** : V2 quand le pool sera suffisant (~100+ joueurs/tier).
- **Notifications push** : "Tu progresses bien sur l'axe X !" — V2.
- **Comparaison vs escouade** : overlay radar avec moyenne squad (réutilise `squad_service_v2`) — V1.5 si rapide.

---

## 9. Checklist de livraison

Avant merge, vérifier (cf. `delivery-checklist` skill) :

- [ ] Tous les commits passent les tests Go (`go test ./...`)
- [ ] Tests frontend passent (`pnpm test`)
- [ ] Aucun fichier > 500L, aucune fonction > 80L (cf. CLAUDE.md §13-14)
- [ ] Pas de hex/Tailwind couleur directe dans le frontend (cf. CLAUDE.md §20)
- [ ] Manifests i18n FR/EN cohérents + linter custom passe
- [ ] `.ai/thought_log.md` mis à jour
- [ ] `docs/ARCHITECTURE_V6.md` mis à jour si nouveaux services majeurs
- [ ] Endpoint testé manuellement avec curl + capture du payload dans doc
- [ ] UI testée en navigateur (page Ascension > Mon Parcours, état < 30 matchs, état nominal, click "Lancer ce défi")
- [ ] Mention ADR 0011 (séparation canonical vs adapter) respectée : pas d'i18n dans le service Go

---

## 10. Références

- `docs/adr/0004-narrative-engine.md` — 8 rôles + radar 6 axes
- `docs/adr/0006-canonical-indicators-and-units.md` — formules KDA/KDR/Accuracy
- `docs/adr/0011-canonical-vs-semantic-adapter-separation.md` — frontière i18n
- `apps/go-api/internal/analysis/skill_rating.go` — TrueSkill 2 + composantes LUSR
- `apps/go-api/internal/sync/skill_config.go` — tiers Bronze→Onyx + chaînes playlist
- `apps/go-api/internal/prestige/types.go` — Challenge, ChallengeTemplate, Arc
- `config/titles/halo_infinite/challenges/templates.toml` — catalogue actuel (27 templates)
- `config/titles/halo_infinite/arcs/presets.toml` — 4 arcs actuels
- `apps/web/src/features/prestige/ObjectifsPage.tsx` — page actuelle (à enrichir)
- `apps/go-api/internal/analysis/temporal/` — LOWESS (réutilisé pour delta lissé campagne R1)
- [PLAN_PROGRESSION_TRACKING_ASCENSION.md](PLAN_PROGRESSION_TRACKING_ASCENSION.md) — V2 (le Coach et les Records s'attacheront aux campagnes actives)
