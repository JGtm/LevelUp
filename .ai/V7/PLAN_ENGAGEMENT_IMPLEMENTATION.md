# Plan d'implementation — EngagementScore (single-player + squad)

> **Statut** : Plan d'implementation — pre-implementation
> **Date** : 2026-04-28
> **Reference conceptuelle** : `.ai/REFLEXION_FORM_SCORE_INTRA_MATCH.md`
> **Mockups visuels** : `.ai/mockups/engagement/engagement_visualizations.html`
> **Branche cible** : a creer depuis `feat/foundations-axes-1-3-4` (ou main si fusionnee) — proposition `feat/engagement-formscore`

---

## 0. Critere de succes global

A la fin du plan, l'application doit :
- Calculer un EngagementScore 0-100 par match pour chaque joueur, stocke dans `player_match_enrichment`
- Exposer la **vue Match View onglet equipe** (Mock 10) avec courbe a 3 traces
- Exposer la **vue Session/Periode single-player** (Mock 11) avec meme grammaire
- Exposer la **Squad Page** (Mock 15 v2) avec 3 courbes team-level + chips squad pour overlay joueur
- Exposer la **Timeseries long** : Mock 11 (EngagementScore single-player session/periode) sous "Score per minute" dans le tab `intensity` (cf §6.6.3)
- Permettre le backfill via le pipeline sync existant avec option `--force-form-score`
- Permettre le recalcul depuis la **page Settings**, section dediee
- Tests unitaires + integration verts ; thought_log a jour ; manifest i18n FR+EN

---

## 1. Phase 0 — Validation des hypotheses (avant code)

**Bloquant** : si une hypothese critique echoue, ne pas implementer en l'etat — revoir le concept.

### 1.1 Sample de validation
- Selectionner 500 matchs PvP d'un joueur reel avec historique > 200 matchs
- Acces brut aux `highlight_events` via DuckDB CLI (pas de Python — projet Go-only)
- Decomposer par categorie de mode (PvP_ranked, PvP_unranked) si volume suffisant

### 1.2 Hypotheses a verifier

| ID | Hypothese | Mesure | Critere | Action si echec |
|---|---|---|---|---|
| H1 | EngagementScore decorrele de PerfScore | `corr(EngagementScore_brut, PerfScore)` sur 500 matchs | < 0.5 | Revoir signaux : retirer S1/S4 anciens si presents |
| H2 | Modele "attendu = coef × team" predit | R² entre `pace_attendu` et `pace_joueur` reel | >= 0.3 | Passer aux baselines conditionnelles (cf §13 ref doc) |
| H3 | `coef_team_share` stable dans le temps | ratio max/min sur fenetres glissantes 100 matchs | < 1.3 | Recalcul plus frequent OU traitement comme metric mobile |
| H4 | E5 Promptitude fiable malgre matchStart imprecis | variance inter-match | acceptable | Sortir E5 du composite (mais E5 est deja absorbe dans residu unique post-refonte) |
| H5 | Cold start raisonnable | % joueurs avec < 30 matchs sur leur categorie principale | < 30 % | Abaisser seuil minimum a 20 |

### 1.3 Outil d'analyse

Comme le projet est Go-only, ecrire un **petit utilitaire CLI Go** dans `apps/go-api/cmd/engagement-validate/main.go` qui :
- Charge les events d'un joueur via le repository existant
- Calcule les sous-metriques
- Genere un rapport texte (matrice de correlation, R², distribution coefs)
- Pas en production, juste en local dev

**Livrable phase 0** : rapport validation/invalidation des H1-H5 dans `.ai/engagement_validation_2026-04-XX.md`

---

## 2. Phase 1 — Backend : couches algorithmiques pures

**Couches** : `domain/` + `analysis/temporal/` (sans DB, sans HTTP)

### 2.1 Types domain — `internal/domain/engagement_score.go`

```go
package domain

type EngagementScoreResult struct {
    EngagementScore       float64    // 0-100, percentile vs historique perso
    ResidualBrut    float64    // valeur brute (mean joueur - attendu)
    EngagementCurve []EngagementPoint
    MatchIntensity  float64    // events/min/joueur lobby (caracteristique match)
    Confidence      string     // "full" / "partial" / "insufficient_history"
    NHistoryMatches int        // taille de la baseline utilisee
}

type EngagementPoint struct {
    TimeMS         int64
    PaceJoueur     float64
    PaceTeam       float64    // = pace_team_per_player
    PaceAttendu    float64    // = coef_team_share * pace_team_per_player
    PaceLobby      float64    // pace_lobby_per_player (pour Mock 15 v2)
    PostDeathFlag  bool
    IsPassiveDeath bool
}

type EngagementCoefficient struct {
    XUID          string
    ModeCategory  string
    CoefTeamShare  float64
    CoefLobbyShare float64
    NMatches       int
    LastUpdated    time.Time
}
```

### 2.2 Algorithmes purs — `internal/analysis/temporal/engagement_score.go`

Structure du fichier (cible < 500 lignes, sinon decouper en `engagement_score_curve.go`, `engagement_score_residual.go`) :

```go
package temporal

// Inputs explicites — pas de couplage DB
type EngagementScoreInput struct {
    PlayerEvents    []canonical.HighlightEvent
    TeamEvents      []canonical.HighlightEvent  // bots filtres
    LobbyEvents     []canonical.HighlightEvent  // humains uniquement
    NTeam           int
    NHumansLobby    int
    XUID            string
    MatchStartMS    int64
    MatchEndMS      int64
    History         []HistoricalFormBrut        // 200 derniers residus bruts
    CoefTeamShare   float64
    CoefLobbyShare  float64
    PersonalScore   int
    Kills           int
    Assists         int
    Mode            ModeCategory
    WindowSeconds   int  // defaut 90
    SamplingSeconds int  // defaut 10
}

type HistoricalFormBrut struct {
    MatchID string
    Brut    float64
}

// Public API
func ComputeEngagementScore(input EngagementScoreInput) (domain.EngagementScoreResult, error)

// Helpers prives
func computeEngagementCurve(input EngagementScoreInput) []domain.EngagementPoint
func computeResidualBrut(curve []domain.EngagementPoint) float64
func percentileFromHistory(brut float64, history []HistoricalFormBrut) float64
func detectPassiveDeath(events []canonical.HighlightEvent, deathTimeMS int64, thresholdMS int64) bool
func eventsObjectifEstimes(personalScore int, kills int, assists int, weight float64) float64
func computeMatchIntensity(lobbyEvents []canonical.HighlightEvent, matchDurationMS int64, NHumans int) float64
```

**Regles arch-rules a respecter** :
- 0 acces DB, 0 import platform/duckdb
- Fonctions < 80 lignes (extraire si necessaire)
- Logger : utiliser `slog.DebugContext(ctx, ...)` si loggin necessaire (sinon zero log dans une fonction pure)
- Generic sur types numeriques si reutilisable

### 2.3 Helpers de coefficient — `internal/analysis/temporal/engagement_coefficient.go`

```go
// Calcule un coefficient personnel a partir de l'historique du joueur
func ComputeCoefTeamShare(history []MatchEngagement) float64
func ComputeCoefLobbyShare(history []MatchEngagement) float64

type MatchEngagement struct {
    PaceJoueur          float64
    PaceTeamPerPlayer   float64
    PaceLobbyPerPlayer  float64
}
```

### 2.4 Tests unitaires — `internal/analysis/temporal/engagement_score_test.go`

Cas obligatoires :
- 0 events → EngagementScore = 0, Confidence = "insufficient_history", erreur sentinel `ErrInsufficientData`
- Match symetrique (events uniengagements joueur ET equipe, joueur = attendu) → EngagementScore ≈ 50
- Match "ideal" (joueur > attendu en continu, recovery rapide) → EngagementScore > 80
- Match "creux" (joueur < attendu en continu) → EngagementScore < 30
- matchStartMS imprecise (decalage 5s) → EngagementScore reste robuste (decalage absorbe)
- Mode FFA (NTeam=1) → fallback automatique sur lobby pour calcul attendu
- Match court (< 3 min) → erreur sentinel `ErrMatchTooShort`
- N_team apres quitter (3 au lieu de 4) → calcul fonctionne, pas de division par zero
- 1 kill 0 death → curve coherente, pas de NaN
- History < 30 → Confidence = "insufficient_history"

Cible : couverture >= 90 % du package `temporal`.

---

## 3. Phase 2 — Backend : persistence et adapters

### 3.1 Migration DB — `internal/migration/steps_engagement_score.go`

Migration **additive** (ADR 0005 compatible) :

```sql
-- Ajout colonnes player_match_enrichment
ALTER TABLE player_match_enrichment ADD COLUMN engagement_score DOUBLE;
ALTER TABLE player_match_enrichment ADD COLUMN engagement_score_brut DOUBLE;
ALTER TABLE player_match_enrichment ADD COLUMN engagement_score_confidence VARCHAR;

-- Ajout colonne match_registry (caracteristique permanente du match)
ALTER TABLE match_registry ADD COLUMN match_intensity DOUBLE;

-- Nouvelle table pour les coefficients personnels par categorie de mode
CREATE TABLE IF NOT EXISTS engagement_coefficients (
    xuid             VARCHAR NOT NULL,
    mode_category    VARCHAR NOT NULL,
    coef_team_share  DOUBLE NOT NULL,
    coef_lobby_share DOUBLE NOT NULL,
    n_matches        INTEGER NOT NULL,
    last_updated     TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (xuid, mode_category)
);
CREATE INDEX idx_engagement_xuid ON engagement_coefficients(xuid);
```

**Localisation** : ajouter step dans la liste des migrations de `internal/migration/`. Pattern existant a respecter (idempotence garantie par `IF NOT EXISTS` et `ALTER TABLE` qui doivent gerer "deja present").

### 3.2 Mappings TOML — `config/titles/halo_infinite/mappings/fields.toml`

Ajouter les nouveaux champs canoniques (un par section) :

```toml
[fields.engagement_score]
label_en = "Form Score"
label_fr = "Score de engagement"
unit = "score"
format = "integer_0_100"
display_order = 220

[fields.match_intensity]
label_en = "Match Intensity"
label_fr = "Intensite du match"
unit = "events/min/player"
format = "decimal_1"
display_order = 221

[fields.coef_team_share]
label_en = "Team Share Coefficient"
label_fr = "Coefficient part equipe"
unit = "ratio"
format = "decimal_2"
display_order = 230

[fields.coef_lobby_share]
label_en = "Lobby Share Coefficient"
label_fr = "Coefficient part lobby"
unit = "ratio"
format = "decimal_2"
display_order = 231
```

Et ajouter les FieldKey correspondants dans `internal/games/canonical/fields.go`.

### 3.3 Port — `internal/port/engagement_score.go`

```go
package port

type EngagementScoreRepository interface {
    LoadPlayerHistory(ctx context.Context, xuid string, mode ModeCategory, limit int) ([]temporal.HistoricalFormBrut, error)
    LoadEngagementCoefficient(ctx context.Context, xuid string, mode ModeCategory) (domain.EngagementCoefficient, error)
    SaveEngagementScore(ctx context.Context, xuid string, matchID string, result domain.EngagementScoreResult) error
    SaveEngagementCoefficient(ctx context.Context, coef domain.EngagementCoefficient) error
    SaveMatchIntensity(ctx context.Context, matchID string, intensity float64) error
}

type EngagementScoreService interface {
    GetMatchFormCurve(ctx context.Context, params MatchFormParams) (*domain.EngagementScoreResult, error)
    GetSquadFormSession(ctx context.Context, params SquadFormParams) (*domain.SquadFormSession, error)
    GetEngagementProfile(ctx context.Context, xuid string, mode ModeCategory) (*domain.EngagementCoefficient, error)
}
```

### 3.4 Repository DuckDB — `internal/platform/duckdb/engagement_score_repo.go`

Implementer `port.EngagementScoreRepository` avec :
- Queries via le pattern de connection existant (`duckdb_read_only`/`duckdb_read_write`)
- 1 query par operation, indexes utilises (`idx_engagement_xuid`)
- `slog.DebugContext` sur les operations significatives
- Erreurs wrappees avec `fmt.Errorf("%s: %w", op, err)`

Tests d'integration : DuckDB `:memory:` avec migration appliquee.

### 3.5 Service orchestrateur — `internal/service/engagement_score_service.go`

```go
type EngagementScoreService struct {
    repo       port.EngagementScoreRepository
    eventsRepo port.HighlightEventsRepository  // existant
    matchRepo  port.MatchRegistryRepository    // existant
    title      games.TitleSemanticAdapter
}

func (s *EngagementScoreService) GetMatchFormCurve(ctx context.Context, params MatchFormParams) (*domain.EngagementScoreResult, error) {
    // 1. Charger events (player + team + lobby)
    // 2. Charger coef + history
    // 3. Appeler temporal.ComputeEngagementScore
    // 4. Retourner le result
}

func (s *EngagementScoreService) GetSquadFormSession(ctx context.Context, params SquadFormParams) (*domain.SquadFormSession, error) {
    // Pour Mock 15 v2 :
    // - Charge events lobby + team par match
    // - Calcule 3 traces team-level (lobby_per_player, team_expected, team_observed)
    // - Charge pace_observed pour chaque squad member
    // - Retourne payload avec base + per-player overlays
}
```

**Regles** :
- Pas de SQL inline ; tout via le repo
- Pas d'appel a un autre service (ex : sessionSvc) — passer par le repo si besoin
- Fonction max 80 lignes ; sinon extraire en helpers prives
- Couverture tests : mock du repo via interface

---

## 4. Phase 3 — Sync et backfill

### 4.1 Localisation existante a confirmer

Le projet a deja un pipeline de sync dans `apps/go-api/internal/sync/` (`engine.go`, `writes.go`, `citations.go`, etc.). Le backfill suit le meme module (pas de scripts Python — projet Go-only confirme par memoire utilisateur).

A explorer/confirmer au debut de cette phase :
- Comment les autres scores (ex : `performance_score`) sont calcules au sync
- Comment `citations_engine` s'integre (cite : pattern de reference)
- Existe-t-il deja un `SyncFlags` / `SyncScope` Go ?

### 4.2 Integration au pipeline de sync — `internal/sync/engagement_score.go`

Nouveau module dans le sync :

```go
// SyncEngagementScore calcule et persiste le EngagementScore pour un match donne
// Appele apres l'ingestion des highlight_events (depend de events_loaded=true)
func SyncEngagementScore(ctx context.Context, deps SyncDeps, matchID string, force bool) error {
    // 1. Verifier que events sont charges (events_loaded flag)
    // 2. Verifier que engagement_score n'existe pas deja (sauf si force)
    // 3. Pour chaque participant humain du match :
    //    a. Charger coef + history
    //    b. Calculer EngagementScore via temporal.ComputeEngagementScore
    //    c. Persister
    // 4. Calculer + persister match_intensity (1 fois par match)
    // 5. Mettre a jour les coefficients (slow path : recalcul mediane glissante)
}
```

### 4.3 SyncFlags / Force option

Le user precise : "support de l'option force". Pattern existant a respecter (cf citations / performance_score).

Suivant la convention probable du projet (a verifier au moment de l'implementation) :

```go
type SyncFlags struct {
    // ... champs existants
    SyncEngagementScore  bool  // active le calcul/recalcul
    ForceEngagementScore bool  // force le recalcul meme si deja present en DB
}
```

Et dans le CLI/admin endpoint :
```bash
# CLI hypothetique
go run ./cmd/sync --player MonGT --form-score
go run ./cmd/sync --player MonGT --form-score --force-form-score

# Ou via flag CLI
go run ./cmd/sync --player MonGT --sync-flags=engagement_score,force_engagement_score
```

### 4.4 Recalcul des EngagementCoefficients

Les coefficients sont des medianes glissantes (200 derniers matchs). Strategie :

**Option A** : recalcul a chaque match synced (simple mais couteux)
**Option B** : recalcul tous les N matchs ou periodiquement (1x par jour) — moins precis mais plus rapide
**Option C** : recalcul incremental (garder un buffer des derniers residus, quand un nouveau match arrive, ajouter au buffer et recalculer mediane sur le buffer)

**Recommandation** : Option C pour la production, Option A acceptable pour le MVP. Documenter la strategie retenue.

### 4.5 Tests sync/backfill

- Test integration : sync sur match neuf → engagement_score persiste, coefficients mis a jour
- Test : sync sur match deja synce sans force → no-op
- Test : sync avec force → recalcule
- Test : match avec history < 30 → engagement_score = null avec confidence "insufficient_history"

---

## 5. Phase 4 — API endpoints

### 5.1 Match View — onglet "Engagement"

Endpoint existant : `GET /api/v1/players/{slug}/matches/{match_id}` (`api/handlers/match_view.go`)

Ajouter un nouveau tab dans la reponse :
```go
type MatchEngagementTab struct {
    EngagementScore        *float64                  // null si insufficient_history
    EngagementCurve  []domain.EngagementPoint
    MatchIntensity   float64
    Confidence       string
    PostDeathBands   []TimeRange               // pour les bandes rouges
    PassiveDeaths    []DeathMarker             // pour les triangles rouges
}
```

Tests `httptest` : 1 cas nominal + 1 cas insufficient_history + 1 cas erreur repo.

### 5.2 Squad Page — payload Mock 15 v2

Nouvel endpoint ou extension de l'existant : `POST /api/v1/players/{slug}/pages/squad/v2/engagement`

```go
type SquadFormSessionResponse struct {
    SessionID    string
    MatchIDs     []string
    Labels       []string  // M1, M2, ... ou date courte
    
    // 3 traces team-level (1 valeur par match)
    LobbyPerPlayer    []float64
    TeamExpected      []float64  // mean(coef_lobby_share squad) × lobby
    TeamObserved      []float64
    
    // Per-player observed pace (pour chip overlay)
    Players []SquadPlayerForm
    
    YRangeAuto YRange  // calcule cote backend pour coherence
}

type SquadPlayerForm struct {
    XUID         string
    Gamertag     string
    PaceObserved []float64
    ColorToken   string  // a resolver cote frontend
}
```

### 5.3 Engagement profile (independent endpoint)

`GET /api/v1/players/{slug}/engagement_profile` :
```go
type EngagementProfileResponse struct {
    Coefficients map[ModeCategory]CoefSet
}

type CoefSet struct {
    CoefTeamShare  float64
    CoefLobbyShare float64
    NMatches       int
    Interpretation string  // "leader equipes moyennes", "passager equipes fortes", etc.
}
```

Reutilisable hors contexte engagement — utile pour Synthesis page, profil joueur.

### 5.4 Capabilities

Tous les endpoints doivent verifier la capability via le `TitleRegistry` :
- Si `!desc.HasCapability(title.CapMatchmaking)` (PvE-only title) → degrader proprement, retourner `ErrCapabilityNotSupported`
- Engagement = PvP only au v1, PvE non couvert

---

## 6. Phase 5 — Frontend : composants

### 6.1 Composant `EngagementScoreCurve.tsx` (Mock 10 / Mock 11)

`apps/web/src/components/charts/EngagementScoreCurve.tsx` :
- Props : `team`, `expected`, `player` (3 series d'events/min) + annotations (deaths, post-mort)
- Auto-zoom Y : calculer depuis les donnees
- Hierarchie visuelle obligatoire (cf §8.6 du doc reflexion)
- Couleurs via `tokenCssVar()` ou `resolveToken()` (pas de hex)
- Reutilisable Match View + Session

### 6.2 Composant `SquadFormView.tsx` (Mock 15 v2)

`apps/web/src/features/squad/SquadFormView.tsx` :
- Recoit la response `SquadFormSessionResponse`
- Render le chart ECharts avec les 3 traces team-level
- Render les chips squad sous le chart
- State local : `selectedPlayer: string | null`
- Click chip → re-render chart avec overlay player
- Auto-zoom Y dynamique selon presence overlay

### 6.3 Composant `PlayerChips.tsx` (reutilisable)

`apps/web/src/components/PlayerChips.tsx` :
- Generic, reutilisable hors contexte engagement
- Props : `players: { id, label, color }[]`, `selectedId: string | null`, `onChange: (id: string | null) => void`
- Hard-edge style cohérent avec design system
- Couleurs via tokens (pas de hex)
- Aria attributes (`aria-pressed`)

### 6.4 Manifest i18n — `apps/web/src/lib/i18n/manifests/engagement.toml`

Nouveau manifest avec sections :
```toml
[trace_labels]
team        = { en = "Team", fr = "Equipe alliee" }
expected    = { en = "Expected", fr = "Attendu" }
player      = { en = "Player", fr = "Joueur" }
lobby       = { en = "Lobby", fr = "Lobby" }

[zones]
above       = { en = "Above expected", fr = "Au-dessus de l'attendu" }
in_zone     = { en = "In your zone", fr = "Dans votre zone" }
below       = { en = "Below expected", fr = "Sous l'attendu" }

[narratives]
above_habit    = { en = "Above your habit (P{p})", fr = "Au-dessus de votre habitude (P{p})" }
normal         = { en = "Normal form (P{p})", fr = "Engagement normale (P{p})" }
below_habit    = { en = "Below your habit (P{p})", fr = "Sous votre habitude (P{p})" }
insufficient   = { en = "Not enough history", fr = "Historique insuffisant" }

[chips]
show_player    = { en = "Show {name}", fr = "Afficher {name}" }
hide_player    = { en = "Hide {name}", fr = "Masquer {name}" }

[glossary]
formscore_title       = { en = "Form Score", fr = "Score de engagement" }
formscore_def         = { en = "...", fr = "Ecart entre votre engagement observe et votre engagement attendu, normalise vs votre historique." }
engagement_coef_team  = { en = "Team share coefficient", fr = "Coefficient de part equipe" }
engagement_coef_lobby = { en = "Lobby share coefficient", fr = "Coefficient de part lobby" }
match_intensity       = { en = "Match intensity", fr = "Intensite du match" }
```

Generation automatique via le pipeline existant (`apps/web/src/lib/i18n/generated/engagement.ts`).

### 6.5 Routes et query keys

- Nouvelle query key dans `apps/web/src/lib/query/keys.ts` :
  ```ts
  engagement: {
    matchCurve: (player, matchId) => ['engagement', 'match-curve', player, matchId],
    squadSession: (player, sessionId) => ['engagement', 'squad-session', player, sessionId],
    engagementProfile: (player) => ['engagement', 'engagement-profile', player],
  }
  ```
- Pas de nouvelle route file-based : tout s'integre dans les routes existantes (cf §6.6 placements)

### 6.6 Placements precis dans les pages existantes

Synthese des placements valides + decisions en attente :

| Page | Fichier root | Placement | Statut |
|---|---|---|---|
| **Squad V2** | `apps/web/src/features/squad/v2/SquadV2Page.tsx` | Nouvelle section apres Contributions (apres `per_minute_stats`, meme contexte) | **Valide** |
| **Match View** | `apps/web/src/features/match-view/MatchViewPage.tsx` | Dans le tab "team" existant, en tete avant Scoreboard | **Valide** |
| **Timeseries** | `apps/web/src/features/timeseries/TimeseriesPage.tsx` | Mock 11 sous "Score per minute" dans le tab `intensity` | **Valide** |
| **Settings** | `apps/web/src/features/settings/SettingsPage.tsx` | Ajout d'une option EngagementScore **DANS la section Backfill existante** du tab Sync | **Valide** |

#### 6.6.1 Squad Page — section "Engagement equipe" (VALIDE)

Le user a explicitement valide : *"après le graphe 'stats par min', c'est le même contexte"*. La section Contributions contient `per_minute_stats` + `frags_deaths_combined`. Pour preserver la coherence de Contributions :

- **Inserer la nouvelle section juste apres Contributions**, avant Radar
- Composant : `<SquadFormView>` (Mock 15 v2)
- Titre i18n : "Engagement equipe"
- Position dans le DOM : `<section className="squad-form">` placee entre la section Contributions (fin l. 185) et la section Radar (debut l. 187)

#### 6.6.2 Match View — dans le tab "team" (VALIDE)

Le user a indique des le debut : *"je prends pour la page match view, je pense onglet équipe"*. Le tab "team" (id: `team`) contient actuellement Scoreboard, Nemesis, Frequent players.

- **Inserer en tete du tab "team"** (avant Scoreboard)
- Composant : `<EngagementScoreCurve>` (Mock 10)
- Source de donnees : `MatchEngagementTab` extrait de la reponse `/matches/{id}` 
- Pas de nouveau tab

#### 6.6.3 Timeseries — Mock 11 dans le tab `intensity` (VALIDE)

Le user a valide : *"on pas un truc qui montre une dynamique de match ou de jeu ? des stats par minutes ? on pourrait mette en dessous"*. Le tab `intensity` contient deja un chart "Score per minute line" qui est exactement un per-minute stat affichant la dynamique au fil des matchs. Le EngagementScore (Mock 11, single-player session/periode, 3 traces) prolonge cette logique en ajoutant le contexte equipe/attendu.

- **Inserer Mock 11 sous "Score per minute"** dans le tab `intensity` (pas de nouveau tab)
- Composant : `<EngagementScoreCurve mode="session">` reutilisable
- Source de donnees : nouveau payload qui regroupe les 3 series (equipe / attendu / joueur cible) sur la fenetre temporelle selectionnee (cf §4 endpoints)
- Ordre cible du tab `intensity` (de haut en bas) :
  1. Heatmap (intra-match × phases) — existant
  2. Score per minute line — existant
  3. **EngagementScore session/periode** — nouveau (Mock 11)
- Coherence narrative : intra-match -> globale -> contextualisee vs equipe

**Tab "form" actuel (EWMA + regression)** : non touche au MVP. Cohabitation sans modification. Une refonte ou retrait de ce tab est une **decision separee**, a trancher en post-MVP avec utilisateur. Le doc reflexion (Annexe A) note que ces approches sont insuffisantes mais le user n'a pas valide leur retrait.

#### 6.6.4 Settings — option dans la section Backfill existante (VALIDE)

Le user a clarifie : *"on a un onglet synchronisation avec une partie sur le backfill, c'est ça dont je te parle"*. Il s'agit d'**ajouter le EngagementScore aux types de donnees backfillables**, pas de creer une section separee.

- **Pas de nouvelle section** — c'est une option **ajoutee a la section Backfill existante** du tab Sync
- Au meme titre que les autres types de donnees deja backfillables (events, sessions, citations, performance scores, etc.)
- Pattern : checkbox "EngagementScore" dans la liste des types de backfill, avec checkbox "Forcer le recalcul" associee si le pattern existant le supporte
- A explorer en debut de phase : la structure actuelle de la section Backfill du tab Sync (`SyncTab` ou equivalent) — identifier le pattern existant pour ajouter un type, suivre le meme pattern
- Endpoint utilise : meme endpoint que le backfill existant, avec un nouveau type cible `engagement_score` et option `force` deja supportee
- i18n dans `engagement.toml` section `[settings]` ajoute au manifest existant

**Important** : ne pas reinventer le mecanisme. Reutiliser strictement le pattern existant des autres types backfillables. La fonction de la nouvelle ligne dans la section Backfill = "exposer le calcul/recalcul du EngagementScore et de l'EngagementCoefficient comme une option de backfill standard".

---

## 7. Phase 6 — Settings page

### 7.1 Section dediee EngagementScore

A explorer au debut de la phase : la structure existante de `apps/web/src/routes/settings/` (ou equivalent). Identifier le pattern des sections existantes.

Section a ajouter :
```
[Section] Engagement & engagement
  - Statut : "EngagementScore active" / "Calcul en cours" / "Insufficient history"
  - Action : "Recalculer EngagementScore pour mes matchs"
    - Button + checkbox "Forcer le recalcul" 
    - Click → POST /api/v1/admin/sync/form-score?force={bool}
  - Action : "Recalculer mes coefficients d'engagement"
    - Idem
  - Stats : nombre de matchs avec engagement_score, derniere mise a jour
```

### 7.2 Endpoint admin

`POST /api/v1/admin/sync/form-score` (proteger par auth) :
- Body : `{ player_slug, force }`
- Reponse : `{ status, n_matches_recomputed, started_at, finished_at }`
- Lance la synchronisation engagement_score asynchrone via le pipeline de sync existant

### 7.3 i18n settings

Ajouter au manifest engagement.toml :
```toml
[settings]
section_title         = { en = "Form & engagement", fr = "Engagement & engagement" }
recompute_button      = { en = "Recompute Form Score", fr = "Recalculer EngagementScore" }
force_checkbox        = { en = "Force recompute existing scores", fr = "Forcer le recalcul des scores existants" }
status_active         = { en = "{n} matches with Form Score", fr = "{n} matchs avec Form Score" }
status_in_progress    = { en = "Computation in progress...", fr = "Calcul en cours..." }
```

---

## 8. Phase 7 — Glossaire applicatif

Le glossaire applicatif consomme les sections `[glossary]` du manifest engagement.toml (deja prevu en §6.4). Verifier que la page Glossaire de l'application charge automatiquement les nouvelles entrees.

---

## 9. Phase 8 — Tests, QA, livraison

### 9.1 Suite de tests obligatoire avant livraison

```bash
# Backend Go
cd apps/go-api && go test ./... && go vet ./...

# Frontend
cd apps/web && npm run typecheck && npm run lint && npm run test
```

### 9.2 Cas E2E manuels (cf §11 du doc reflexion)

- Joueur en bonne engagement connue → EngagementScore > 60 ✓
- Joueur en creux → EngagementScore < 40 ✓
- Joueur sur equipe steamrollee qui fait sa part → EngagementScore ~ 50 ✓
- Joueur passager sur equipe dominante → EngagementScore < 50 ✓
- Mode FFA → fallback lobby_share automatique
- Match court < 3 min → EngagementScore null

### 9.3 Verifications arch-rules

- Aucun fichier .go > 500 lignes
- Aucune fonction > 80 lignes
- Aucun `fmt.Println` ni `log.Printf` ajoute
- Aucun `filepath.Join(repoRoot, "data", ...)` direct (passer par PathResolver)
- Aucun branchement sur `slug == "halo_infinite"` (utiliser `HasCapability`)
- Aucune couleur hex dans `apps/web/src/features/` ni `components/`
- Manifest i18n complet (FR + EN)
- Logging via `slog.{Debug,Info,Error}Context` avec cles standardisees

### 9.4 Documentation

- Mise a jour `.ai/project_map.md` (ajouter le module engagement)
- Mise a jour `.ai/data_lineage.md` (decrire le flux events → engagement_score → API → frontend)
- Entree finale dans `.ai/thought_log.md` (statut "Complete")

### 9.5 Feature flag (optionnel)

Si on veut activer progressivement :
- `internal/config/feature_flags.go` : `FeatureEngagementScore bool`
- Branchement dans le sync (no-op si flag off) et dans les endpoints (404 si off)
- Suppression du flag apres deploiement prod

---

## 10. Decoupage en commits / chunks livrables

Chaque chunk doit etre **livrable independamment** (compile, tests verts, deployable). Ordre suggere :

| Chunk | Contenu | Livrable seul ? |
|---|---|---|
| C0 | Phase 0 — Validation hypotheses + rapport | Oui (rapport, pas de code prod) |
| C1 | Phase 1 — Algos purs + tests unitaires | Oui (package isole) |
| C2 | Phase 2 — Migration DB + repo + service + tests integration | Oui |
| C3 | Phase 3 — Integration sync + backfill (sans force) | Oui |
| C4 | Phase 3 — Option force + endpoint admin | Oui (extension de C3) |
| C5 | Phase 4 — Endpoint Match View Engagement tab | Oui |
| C6 | Phase 4 — Endpoint Squad Form Session | Oui |
| C7 | Phase 5 — Composants Frontend (SquadFormView, EngagementScoreCurve, PlayerChips) | Oui |
| C8 | Phase 5 — Integration pages (Match View tab, Squad Page) | Depend de C7 |
| C9 | Phase 6 — Settings page section + endpoint | Oui |
| C10 | Phase 7-8 — Glossaire, tests E2E, doc | Oui |

**Total estime** : ~10 j-h en hypothese mid-stack experimente.

**Strategie de branche** : 1 branche `feat/engagement-formscore` avec les chunks en commits successifs (cf CLAUDE.md regle "1 tache = 1 branche, N commits"). Pas de sub-branches paralleles sauf si C5 et C6 sont totalement independants (alors ok parallele courte duree).

---

## 11. Verification end-to-end

Une fois tout livre, scenario de validation manuelle :

1. **Sync de demo** : lancer le sync sur 1 joueur de demo avec >= 50 matchs PvP
2. **Match View** : ouvrir un match → onglet equipe → Mock 10 visible avec 3 courbes + annotations
3. **Session** : ouvrir une session de 5+ matchs → Mock 11 visible (3 traces granularite match)
4. **Squad Page** : si squad de 4 joueurs sur cette session → Mock 15 v2 avec 3 courbes team + chips
5. **Click chip** : la courbe joueur apparait en overlay
6. **Click chip active** : la courbe disparait
7. **Click autre chip** : swap de courbe
8. **Settings** : section Engagement visible, bouton "Recalculer" avec checkbox force
9. **Force recalcul** : observer le sync recalculer les engagement_scores → recharger Match View, voir les valeurs mises a jour
10. **Glossaire** : entree EngagementScore + EngagementCoefficient + MatchIntensity visibles

Si tout passe → feature livree.

---

## 12. Risques et points de vigilance

- **Performance** : calcul des coefficients sur 200 matchs × N joueurs peut etre lourd. Privilegier l'option C (incremental) en production.
- **Cold start massif** : sur la migration initiale, beaucoup de joueurs auront < 30 matchs et `confidence = "insufficient_history"`. Documenter clairement dans l'UI ("Form Score sera disponible des 30 matchs joues").
- **Coherence sync** : verifier que le sync n'oublie pas de recalculer le engagement_score quand de nouveaux matchs arrivent (depend de l'ordre des steps dans le pipeline). Doit etre apres `events_loaded=true`.
- **Mode asymetrique sans events** : un porteur de drapeau pur peut avoir 0 events highlight. Le `events_objectif_estimes` (cf 4.2 doc reflexion) compense via `personal_score - 100*kills - 50*assists`. Calibrer le `poids_unitaire_objectif` au moment de l'implementation.
- **Stats per-minute API** : ne **pas** utiliser les `kills_per_minute` etc. de l'API (precision insuffisante). Tout depuis timestamps bruts (cf §3.2 et §4.3 doc reflexion).
- **MatchStart en cours de calibration** : le module engagement est agnostique sur la methode de detection. E5 Promptitude est de toute facon absorbe dans le residu unique post-refonte. Pas de bloquant.

---

## 13. Pre-requis externes

- **PathResolver** : verifier qu'il expose les chemins necessaires pour le repo engagement_score (probablement aucun nouveau chemin requis — utilise les DBs existantes)
- **TitleRegistry** : aucune nouvelle capability requise (utilise CapMatchmaking existante)
- **TitleAdapter** : aucun ajout — les events/medals sont deja accessibles via `canonical.HighlightEvent`

Pas de dependance externe bloquante.

---

## 14. Apres la livraison v1

Evolutions documentees comme v2 (cf §13 du doc reflexion) :
- Baselines conditionnelles par bucket d'intensite (si H2 valide partiellement)
- PvE / Firefight (besoin de definir un fair share alternatif sans equipe ennemie)
- Detection auto de tilt / fatigue (label categoriel)
- Generation narratif texte automatique
- Onglet "Comparaison squad" sur Squad Page (Mock 13 ou 14)
- Comparaison inter-joueurs sur l'EngagementCoefficient (puisque c'est la seule metrique inter-comparable)
