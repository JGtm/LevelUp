# HALO_EXTERNAL_OPPORTUNITIES.md — Opportunites externes Halo et plans d'implementation

> Document de cadrage issu de la revue de trois repos externes :
>
> 1. SpartanRecord
> 2. halo-infinite-api
> 3. HaloInfiniteGetter
>
> Objectif : transformer les idees utiles en opportunites concretes pour LevelUp-go-migration,
> avec un plan d'implementation exploitable par lot.

## Role du document

Ce document sert a trois choses :

1. lister les opportunites produit, data et outillage issues de projets externes ;
2. prioriser ce qui a le meilleur ratio valeur / effort pour LevelUp ;
3. proposer, pour chaque opportunite, un plan d'implementation adapte a l'architecture Go + React du repo.

## Hypothese de travail

Les opportunites ci-dessous sont pensees pour la cible runtime actuelle :

1. backend Go dans `apps/go-api/` ;
2. frontend React dans `apps/web/` ;
3. DuckDB comme source de verite locale ;
4. contrats API internes orientes parcours utilisateur, pas orientes endpoints Waypoint.

## Mise a jour apres arbitrage produit (2026-04-18)

Decisions finales actees lors de la revue du 2026-04-18 :

1. O4 Match count endpoint est ecarte definitivement : la valeur est trop faible avec les filtres et le systeme d'exclusion LevelUp.
2. O6 Export service record / profil est ecarte definitivement : faible valeur percue.
3. O10 Store tracker passe en backlog multi-titre : hors scope Halo Infinite (store en fin de cycle), mais potentiellement pertinent si un nouveau titre dispose d'une economie active.
4. O11 Social layer passe en backlog multi-titre : hors scope Halo Infinite aujourd'hui, potentiellement pertinent si un nouveau titre dispose d'une dimension groupe native ; a rattacher a `Squad` dans tous les cas, jamais comme sous-produit autonome.
5. O14 ETag / snapshots est bundle dans O2 : pas d'opportunite separee, juste une extension naturelle du CLI de refresh metadata.
6. O3 Medals metadata et O8 Asset discovery sont retenus en mode anticipe multi-titre, avec garde-fou d'import (voir sections detaillees).
7. O7 Leaderboards adopte la meme architecture que O5 Compare : appel Waypoint a la volee pour les joueurs non locaux, DuckDB prioritaire pour les joueurs deja synces.
8. O2 Season calendars est persiste dans `metadata.duckdb` via CLI Go isole, jamais via job embedded dans l'API.

Les opportunites retenues sont evaluees selon deux criteres :

1. renforcer les surfaces existantes sans ajouter de menu ;
2. fiabiliser la couche metadata / provider qui alimente deja le produit.

## Principes d'atterrissage UI

Pour eviter de surcharger la navigation, les opportunites retenues doivent suivre ces regles :

1. O2, O3 et O8 ne meritent aucun menu : ce sont des briques invisibles qui alimentent `career`, `match-history`, `match-view` et les futures pages derivees.
2. O5 Compare doit vivre comme un mode contextuel declenche depuis une surface existante du scope joueur, par exemple `career`, `home`, `squad` ou `profile/citations`, pas comme une rubrique globale supplementaire.
3. O7 Leaderboards doit commencer comme un bloc ou une vue secondaire dans `career` ou `home`, pas comme une section autonome.
4. O9 Year in Review peut exister comme route partageable ou campagne saisonniere, sans entree permanente dans le shell principal.
5. Toute nouvelle surface produit doit d'abord prouver sa valeur comme sous-parcours dans une page existante avant de gagner un point d'entree dedie.

## Resume executif

### Valeur immediate retenue (Lot A)

1. fiabiliser les metadonnees saisons / CSR seasons (O2, bundle avec O14) ;
2. exposer la privacy des matchs (O1) ;
3. lancer un Compare MVP sans nouveau menu (O5) ;
4. poser le socle multi-titre pour assets et medailles en anticipation (O3, O8).

### Valeur transverse utile a moyen terme (Lots B et C)

1. asset discovery + versioning comme outillage de verification interne (O8) ;
2. Waypoint Explorer interne (O13) ;
3. Leaderboards dynamiques limites au corpus LevelUp etendu (O7) ;
4. Year in Review sous forme de route partageable saisonniere (O9) ;
5. Ban diagnostics en route admin (O12).

### Ecartees definitivement

O4, O6 — voir sections individuelles pour le raisonnement.

### Backlog multi-titre (hors scope Halo Infinite)

O10, O11 — non pertinents pour Halo Infinite aujourd'hui, mais potentiellement utiles
si un nouveau titre Halo dispose d'une economie active (O10) ou d'une dimension groupe
etablie (O11). A remettre sur la table lors de l'onboarding d'un nouveau titre.

## Vagues recommandees

| Vague | Horizon | Opportunites | But principal |
|-------|---------|--------------|---------------|
| V1 | court terme | O1 privacy, O2+O14 calendars+ETag, O5 compare MVP | gains rapides UX et fiabilite metadata |
| V2 | court / moyen terme | O3 medals, O8 asset discovery, O13 Waypoint Explorer | socle multi-titre et outillage interne |
| V3 | moyen terme | O7 leaderboards, O9 Year in Review, O12 ban diagnostics | experiences secondaires sans nouveau menu |

## Inventaire priorise

| ID | Opportunite | Source externe | Type | Statut | Commentaire |
|----|-------------|----------------|------|--------|-------------|
| O1 | Match privacy | halo-infinite-api, SpartanRecord | produit + data | **livre Sprint 54 B** | PrivacyBanner dans HomePage, CompareDrawer, MatchPrivacyWarning dans types TS |
| O2 | Season calendars + CSR calendars | halo-infinite-api | metadata | **livre Sprint 54 A** | CLI `refresh-metadata seasons`, tables `waypoint_seasons`, ETag + ContentHash |
| O3 | Medals metadata Waypoint | halo-infinite-api | metadata | **livre Sprint 54 D** | CLI `refresh-metadata medals`, table `waypoint_medals_raw`, guard cardinalite + assets |
| O4 | Match count endpoint | halo-infinite-api | perf + UX | ecarte definitif | peu utile avec filtres + exclusions |
| O5 | Compare joueur vs joueur | SpartanRecord | produit | **livre Sprint 54 C** | handler POST .../pages/compare, CompareService errgroup, CompareDrawer React + prefetch |
| O6 | Export service record / profil | SpartanRecord | produit | ecarte definitif | faible valeur percue |
| O7 | CSR leaderboards | SpartanRecord | produit | **livre Sprint 54 E** | LeaderboardBlock integre Career, handler GET .../pages/leaderboard, prefetch hover |
| O8 | Asset discovery + versioning | halo-infinite-api | metadata + tooling | **livre Sprint 54 D** | CLI `refresh-metadata assets`, ComputeAssetDiff, table `asset_cache`, multi-titre |
| O9 | Year in Review | SpartanRecord | produit | retenu V3 | route partageable, bloque par O2 + coverage annuelle |
| O10 | Store / economy tracker | SpartanRecord | produit | backlog multi-titre | hors scope Halo Infinite aujourd'hui, pertinent si nouveau titre avec economie active |
| O11 | Spartan Company / social layer | SpartanRecord | produit | backlog multi-titre | hors scope Halo Infinite aujourd'hui, a rattacher a Squad ou a une surface groupe multi-titre |
| O12 | Ban diagnostics | halo-infinite-api | support + ops | retenu V3 | route admin uniquement |
| O13 | Waypoint Explorer interne | HaloInfiniteGetter | dev tooling | retenu V2 | accelerateur de spikes et metadata |
| O14 | ETag cache + snapshots | HaloInfiniteGetter | dev tooling | **livre Sprint 54 A** | bundle dans O2 — CLI refresh integre ETag + ContentHash |

---

## Architecture transverse : couche PlayerStats multi-titre (O5 + O7)

Cette section documente les elements partages entre O5 Compare et O7 Leaderboards.
Ils doivent etre implementes une seule fois, dans le cadre de O5, puis reutilises par O7.

### Principe de reutilisation

Les patterns architecturaux existants (port/service/platform, ServiceFactory, TitleExtractor,
HaloProvider, PlayerDB pool) sont reutilises sans deviation. Aucune nouvelle convention
n'est introduite — les nouvelles couches s'insèrent dans les slots prevus par l'architecture.

### Elements partages

| Element | Package | Description |
|---------|---------|-------------|
| `NormalizedPlayerStats` | `domain/compare.go` | Stats joueur normalisees, multi-titre via `TitleSlug` + `Extended` |
| `PlayerStatsProvider` | `port/player_stats_provider.go` | Interface : fetch Waypoint a la volee pour un xuid |
| `HaloCompareProvider` | `platform/halo/compare_provider.go` | Implementation Halo Infinite (reutilise `HaloProvider` existant) |

### Elements specifiques O5

| Element | Package | Description |
|---------|---------|-------------|
| `CompareRequest/Response` | `domain/compare.go` | Contrat compare deux joueurs |
| `CompareRepository` | `port/repository.go` | Lecture joueur A depuis DuckDB |
| `CompareService` | `service/compare_service.go` | Orchestration errgroup A+B |
| Handler + route | `api/handlers/compare.go` | POST .../pages/compare |

### Elements specifiques O7

| Element | Package | Description |
|---------|---------|-------------|
| `LeaderboardEntry/Request/Response` | `domain/leaderboard.go` | Contrat leaderboard |
| `LeaderboardRepository` | `port/repository.go` | Joueurs locaux depuis DuckDB |
| `LeaderboardService` | `service/leaderboard_service.go` | Merge local + Waypoint batch |
| Handler + route | `api/handlers/leaderboard.go` | GET /leaderboard |

### Propagation du titre

Le `titleSlug` est toujours lu depuis `ctxkeys.TitleSlug(ctx)`, injecte par le middleware
`TitleExtractor` existant. Aucune couche Compare ou Leaderboard ne lit le titre autrement.
Le `PathResolver` existant gere les chemins DuckDB title-namespaced.

### Sequence de fetch pour Compare et Leaderboards

```
Request (Header X-LevelUp-Title ou session)
    -> TitleExtractor middleware -> ctxkeys.WithTitleSlug(ctx, slug)
    -> Handler -> ServiceFactory -> CompareService / LeaderboardService
        -> CompareRepository.GetLocalStats(ctx)     // DuckDB, titleSlug via ctx
        -> PlayerStatsProvider.FetchRemoteStats(ctx) // Waypoint, titleSlug via ctx
        -> errgroup.Wait()
        -> assembler NormalizedPlayerStats x2
        -> retourner CompareResponse / LeaderboardResponse
```

### Extension vers un second titre

Quand un second titre Halo est supporte :

1. enregistrer le nouveau titre dans `domain/title/registry.go` (deja prevu) ;
2. creer `HaloXXXProvider` dans `platform/halo/` implementant `PlayerStatsProvider` ;
3. enregistrer dans `ServiceRegistry` avec le slug du nouveau titre ;
4. aucune modification dans `CompareService` ou `LeaderboardService` — ils consomment l'interface.

---

## O1 — Match privacy

### Pourquoi c'est interessant

1. permet d'expliquer proprement pourquoi certaines donnees match sont absentes ;
2. aligne LevelUp avec un besoin UX deja visible chez SpartanRecord ;
3. evite de traiter des trous de donnees comme des erreurs produit vagues.

### Signal externe utile

1. `GET /hi/players/{xuid}/matches-privacy` ;
2. `PUT /hi/players/{xuid}/matches-privacy` dans `halo-infinite-api`.

### Plan d'implementation

1. Ajouter un provider interne Go pour lire la privacy des matchs depuis le backend Halo.
2. Etendre le modele canonique ou le read model bootstrap avec un champ de visibilite match.
3. Exposer une information lisible dans `GET /api/v1/bootstrap` et dans les pages dependantes de l'historique.
4. Afficher un bandeau explicite dans le frontend React quand l'historique est prive ou partiellement exploitable.
5. Ajouter un warning structure dans les payloads `match-history`, `match-view` et `explorer`.

### Chantiers techniques

1. `apps/go-api/internal/service/bootstrap_service.go`
2. `apps/go-api/internal/api/handlers/`
3. `apps/web/src/features/*`
4. eventuelle persistance DuckDB dans une table metadata joueur si on veut memoriser le dernier etat observe.

### Validation

1. cas compte public ;
2. cas compte prive ;
3. cas transition public -> prive ;
4. golden values de warning au lieu d'une erreur generique.

---

## O2 — Season calendars + CSR calendars (bundle O14)

### Pourquoi c'est interessant

1. fiabilise les saisons et periodes CSR ;
2. reduit le hardcode metier ;
3. aide Compare, Year in Review, historique, rankings et projections.

### Signal externe utile

1. `SeasonCalendar.json` ;
2. `CsrSeasonCalendar.json` exposes par `halo-infinite-api`.

### Plan d'implementation

1. Creer un CLI Go dedie (jamais un job embedded dans l'API) pour fetcher les calendars depuis Waypoint.
2. Persister les resultats dans `metadata.duckdb` avec `version`, `fetched_at` et `content_hash`.
3. Comparer le `content_hash` a chaque refresh : si changement detecte, declencher la notification Discord existante.
4. Stocker egalement les anciennes versions des ressources a haute valeur (seasons, CSR calendars) pour permettre un diff N / N+1.
5. Brancher les services career, stats et compare sur ces tables plutot que sur des hypotheses statiques.
6. Ajouter une politique de fallback sur les donnees locales si le provider est indisponible.

### Architecture de refresh

```
CLI Go (manuel ou scheduler externe)
  -> fetch Waypoint
  -> compare content_hash avec derniere version en DB
  -> si changement : upsert metadata.duckdb + notification Discord
  -> si inchange : log silencieux, pas d'ecriture
```

Le CLI est la seule porte d'entree pour la mise a jour des calendars. L'API Go lit uniquement DuckDB, jamais Waypoint a la volee pour cette ressource.

### O14 bundle : ETag + snapshots

Le suivi ETag et les snapshots de ressources sont implementes directement dans ce CLI, pas dans une opportunite separee :

1. stocker `etag`, `content_hash`, `fetched_at`, `source_url` par ressource critique dans `metadata.duckdb` ;
2. conserver les N dernieres versions pour les ressources a haute valeur : seasons, CSR calendars, medals metadata, playlists, map-mode pairs ;
3. exposer un diff simple entre version N et N+1 en sortie CLI.

### Materialisation UI recommandee

1. aucune entree de menu dediee ;
2. utiliser ces donnees pour fiabiliser les filtres et libelles deja presents dans `career`, `match-history` et les futures vues compare ;
3. si une exposition explicite devient necessaire, la limiter a un selecteur de saison dans un ecran existant.

### Chantiers techniques

1. migration metadata DuckDB (tables `season_calendars`, `csr_season_calendars`, `waypoint_resource_snapshots`) ;
2. CLI Go de refresh avec gestion ETag et notification Discord ;
3. repository Go metadata ;
4. tests de mapping `provider -> metadata.duckdb`.

### Validation

1. table de seasons coherente avec les saisons visibles dans l'UI ;
2. regression tests sur current season ;
3. tests sur plages annuelles 2024/2025 pour Year in Review ;
4. test sur 304 Not Modified (ETag inchange) ;
5. test sur changement d'ETag : diff produit, notification envoyee ;
6. test de restauration d'un snapshot local.

---

## O3 — Medals metadata Waypoint (anticipe multi-titre)

### Pourquoi c'est interessant

1. enrichit les labels, categories et assets medailles pour Halo Infinite ;
2. pose le socle pour supporter plusieurs titres Halo (multi-titre) sans restructuration ulterieure ;
3. permet de verifier ou completer la metadata locale actuelle.

### Principe fondamental : ne pas toucher la table existante

La table medals actuelle fonctionne. L'import Waypoint ne doit jamais l'ecraser directement.

### Garde-fou d'import obligatoire

Avant toute promotion de donnees Waypoint vers les tables de production, verifier :

1. que la cardinalite Waypoint est coherente avec la cardinalite locale (tolerance : +/- 10%) ;
2. que chaque entree Waypoint contient au minimum : `medal_id`, `label`, `category`, `rarity` ;
3. que les images / assets sont recuperables pour toutes les entrees ou pour aucune (pas d'import partiel d'assets) ;
4. si l'une de ces conditions echoue, l'import est bloque et un rapport d'ecart est genere.

### Architecture multi-titre anticipee

L'API Go doit exposer les assets medailles avec un champ `title_id` des le debut, meme si Halo Infinite est le seul titre aujourd'hui. Exemple de structure cible :

```go
type MedalAsset struct {
    TitleID     string // "halo-infinite", "halo-5", etc.
    MedalID     string
    Label       string
    Category    string
    Rarity      string
    ImageURL    string
    Description string
}
```

La table DuckDB doit inclure `title_id` comme cle de partition des le premier import.

### Plan d'implementation

1. Creer une table staging `waypoint_medals_raw` dans `metadata.duckdb`.
2. Importer la metadata Waypoint dans staging uniquement.
3. Executer les garde-fous : cardinalite, champs requis, assets disponibles.
4. Si tous les garde-fous passent : promouvoir vers la table enrichie `medal_metadata` avec `title_id`.
5. Rebrancher citations/commendations sur cette source enrichie.
6. Conserver la table actuelle comme fallback si une cle manque dans la table enrichie.

### Validation

1. comparaison de cardinalite entre metadata actuelle et metadata Waypoint ;
2. test de blocage si images absentes ou partielles ;
3. tests de rendu frontend sur medaille connue, medaille inconnue, icone absente ;
4. test de non-regression sur les medailles deja presentes en production.

---

## O4 — Match count endpoint

### Decision produit

Ecarte definitivement.

### Pourquoi

1. les filtres et exclusions LevelUp rendent le compteur brut Waypoint peu exploitable ;
2. un total provider risque d'etre moins utile qu'un total calcule dans le scope reel LevelUp ;
3. la complexite de reconciliation ne se justifie pas.

---

## O5 — Compare joueur vs joueur

### Pourquoi c'est interessant

1. forte valeur produit immediate ;
2. surface differenciante par rapport aux pages deja portees ;
3. faible dependance au runtime legacy si on reste sur des KPIs lisibles.

### Signal externe utile

1. CompareView dans SpartanRecord ;
2. comparaison de rank, matches, winrate, KDA, KDR, CSR, damage, accuracy.

### Architecture : appel Waypoint a la volee

Les stats du joueur compare (joueur B) ne sont pas stockees localement. Le `CompareService` est un proxy enrichi :

1. le joueur A est charge depuis DuckDB (stats locales synces, acces rapide) — meme chemin que les services existants via `ServiceFactory` ;
2. le joueur B fait l'objet d'un appel Waypoint a la volee via le `HaloProvider` existant (rate limiting + retry deja geres) ;
3. les deux fetches tournent en goroutines paralleles avec `errgroup` ;
4. le read model compare est assemble a partir des deux sources et retourne directement ;
5. aucune ecriture en base pour le joueur B, sauf si l'utilisateur decide de le syncer.

Consequence sur la latence : ~2 a 4 secondes selon Waypoint. Le frontend React doit afficher un skeleton loader. La latence est acceptable avec un loader propre, inacceptable sans.

### Architecture multi-titre

Le `titleSlug` est deja en contexte via le middleware `TitleExtractor` existant. Toutes les couches
Compare doivent le propager sans exception.

#### Couche 1 — Interface provider (nouvelle, dans `port/`)

```go
// port/player_stats_provider.go
type PlayerStatsProvider interface {
    // Charge les stats normalisees d'un joueur depuis Waypoint
    // titleSlug est propage depuis ctxkeys.TitleSlug(ctx)
    FetchRemoteStats(ctx context.Context, xuid string, filters FilterContextInput) (*NormalizedPlayerStats, error)
}
```

Un seul provider concret aujourd'hui (`platform/halo/compare_provider.go`), un par titre futur.
L'interface est enregistree dans `ServiceRegistry` comme les autres ports existants.

#### Couche 2 — Normalisation (dans `domain/`)

```go
// domain/compare.go
type NormalizedPlayerStats struct {
    TitleSlug    string
    XUID         string
    Gamertag     string
    Matches      int
    WinRate      float64
    KDA          *float64
    KDR          *float64
    KillsPerGame float64
    DeathsPerGame float64
    AssistsPerGame float64
    Accuracy     *float64
    DamagePerGame *float64
    CareerRank   *int
    CSRCurrent   *float64
    CSRBest      *float64
    // KPIs titre-specifiques (ex: Firefight waves pour un futur titre PvE)
    Extended     map[string]any
}
```

Cette struct est le contrat commun entre tous les titres. Elle etend `StatsMatchRow` /
`MatchMetrics` existants sans les remplacer — les algos de `analysis/performance_score.go`
restent intacts.

#### Couche 3 — CompareService (dans `service/`)

```go
// service/compare_service.go
// Suit le meme pattern que CareerService, StatsService, etc.
type CompareService struct {
    playerRepo  port.CompareRepository   // lit joueur A depuis DuckDB
    provider    port.PlayerStatsProvider // fetch joueur B via Waypoint
}

func (s *CompareService) Compare(ctx context.Context, req domain.CompareRequest) (domain.CompareResponse, error) {
    // errgroup pour paralleliser les deux charges
    // titleSlug depuis ctxkeys.TitleSlug(ctx) — propoge automatiquement
    // assembler SessionCompareMetricRow (pattern existant dans session_compare.go)
}
```

`CompareRepository` est une nouvelle interface dans `port/repository.go`, implementee dans
`platform/duckdb/` exactement comme `CareerRepo`, `StatsRepo`, etc.

#### Couche 4 — Handler + route

Route ajoutee dans `server.go` au meme niveau que les routes existantes :

```
POST /api/v1/players/{player_slug}/pages/compare
```

Le handler suit le pattern `ServiceFactory[port.CompareService]` existant —
aucune deviation architecturale.

### Plan d'implementation

1. Ajouter `NormalizedPlayerStats` et `CompareRequest/Response` dans `domain/compare.go`.
2. Ajouter `PlayerStatsProvider` dans `port/player_stats_provider.go` et `CompareRepository` dans `port/repository.go`.
3. Implementer `HaloCompareProvider` dans `platform/halo/compare_provider.go` (reutilise `HaloProvider` existant).
4. Implementer `CompareRepo` dans `platform/duckdb/compare_repo.go`.
5. Implementer `CompareService` dans `service/compare_service.go` avec `errgroup`.
6. Enregistrer dans `ServiceRegistry` (`api/registry.go`).
7. Ajouter le handler et la route dans `server.go`.
8. MVP sur 10 a 12 KPIs stables : matches, winrate, KDA, KDR, kills/game, deaths/game, assists/game, CSR current, CSR best, accuracy, damage/game, career rank.
9. Cote React, feature `compare` avec selection du second joueur et skeleton loader.

### Materialisation UI recommandee

#### Parcours 1 — Explorer (prioritaire, contexte le plus fort)

```
Explorer → Mode Joueur → recherche gamertag
  → Card bilan face-a-face (existante)
    → CTA "Comparer" dans la Card
      → Drawer Compare (pattern FilterDrawer existant)
        → Stats cote a cote, KPIs normalises
```

C'est le point d'entree le plus pertinent : l'utilisateur vient de consulter les matchs
communs avec ce joueur. La comparaison globale arrive dans un contexte charge de sens.
`GamertagSearchInput` est deja resolu — le joueur B est connu, pas de nouvelle saisie.

#### Parcours 2 — Career / Squad (complementaire)

```
Career → section Encounters (rencontres frequentes, deja presente)
  → Hover ou clic sur une ligne adversaire/coequipier
    → CTA "Comparer" inline
      → Drawer Compare (joueur B pre-rempli, pas de saisie)

Career → en-tete joueur
  → CTA "Comparer" general
    → GamertagSearchInput (composant existant, reutilise directement)
      → meme Drawer Compare

Squad
  → Clic sur une ligne coequipier recurrent
    → CTA "Comparer" dans la ligne
      → meme Drawer Compare (joueur B pre-rempli)
```

#### Ce qu'il ne faut pas faire

Ne pas creer une route `/players/$secondarySlug` pour un profil complet du joueur B.
Cela impliquerait de reproduire tout le layout joueur pour quelqu'un dont les donnees
ne sont pas en base locale. Le drawer assume explicitement que les donnees du joueur B
sont partielles (fetch Waypoint a la volee) et reste dans le contexte du joueur A.

#### Composants React a reutiliser

| Composant existant | Reutilisation |
|--------------------|---------------|
| `FilterDrawer.tsx` | Pattern backdrop + animate-in/out pour le Drawer Compare |
| `GamertagSearchInput` | Autocomplete joueur B dans Career (CTA general) |
| `ExplorerPage` Card resultats | Ajout du CTA "Comparer" inline |
| `CareerPageResponse.encounters_preview` | Gamertags deja charges → prefetch gratuit au hover |
| `TeammateRow[]` (Squad) | Gamertags deja charges → prefetch gratuit au hover |

3. eviter tout item de navigation global tant que la feature n'a pas prouve son usage.

### Strategie de prefetch React

#### Infrastructure requise en premier

1. Ajouter `queryKeys.comparePlayer(playerSlug, targetGamertag)` dans `lib/query/keys.ts`
   — sans cle stable centralisee, le cache n'est pas partage entre Explorer, Career et Squad.
2. `staleTime: 2 * 60 * 1000` pour les stats Compare (donnees Waypoint, pas temps-reel).

#### Points de prefetch pertinents

| Declencheur | Joueur B connu ? | Gain | Implementation |
|-------------|-----------------|------|----------------|
| Hover CTA "Comparer" dans Card Explorer | Oui (deja resolu) | Maximal | `onMouseEnter` → `prefetchQuery` |
| Hover ligne Encounters dans Career | Oui (`encounters_preview` deja charge) | Maximal | `onMouseEnter` → `prefetchQuery` |
| Hover ligne coequipier dans Squad | Oui (`TeammateRow[]` deja charge) | Maximal | `onMouseEnter` → `prefetchQuery` |
| Selection joueur B dans GamertagSearchInput | Oui (juste selectionne) | Fort | `onSelect` → `prefetchQuery` |

Les trois premiers cas sont gratuits : les gamertags sont **deja dans le cache React Query**
(via `encounters_preview` charge par Career, ou `TeammateRow[]` charge par Squad).
Il suffit d'un `onMouseEnter` pour lancer le fetch Waypoint avec 1-2s d'avance sur le clic.

#### Ce qu'il ne faut pas prefetcher

- Au mount de Career ou Squad : joueur B inconnu, charge Waypoint a l'aveugle.
- Tous les coequipiers de la liste Squad : risque de rate limit sur le HaloProvider.
- Au survol d'une ligne de match dans l'historique : intention pas claire.

#### Pattern d'implementation (identique sur tous les points d'entree)

```typescript
const queryClient = useQueryClient()

// Sur hover d'une ligne Encounter ou coequipier Squad
onMouseEnter={() => {
  queryClient.prefetchQuery({
    queryKey: queryKeys.comparePlayer(playerSlug, targetGamertag),
    queryFn: () => api.post(`/players/${playerSlug}/pages/compare`, { target: targetGamertag }),
    staleTime: 2 * 60 * 1000,
  })
}}
```

Le meme pattern s'applique au CTA "Comparer" de la Card Explorer et au `onSelect`
du `GamertagSearchInput`. Une seule implementation de hook (`useComparePrefetch`)
partagee entre tous les points d'entree.

### Chantiers techniques

1. `domain/compare.go` — types `NormalizedPlayerStats`, `CompareRequest`, `CompareResponse`, `CompareMetricRow` ;
2. `port/player_stats_provider.go` — interface `PlayerStatsProvider` ;
3. `port/repository.go` — ajout interface `CompareRepository` ;
4. `platform/halo/compare_provider.go` — implementation Waypoint (reutilise `HaloProvider`) ;
5. `platform/duckdb/compare_repo.go` — implementation DuckDB ;
6. `service/compare_service.go` — orchestration avec `errgroup` ;
7. `api/registry.go` — enregistrement `ServiceFactory[port.CompareService]` ;
8. `api/handlers/compare.go` + route dans `server.go` ;
9. contrat OpenAPI mis a jour ;
10. feature React `compare` avec skeleton loader.

### Validation

1. golden values sur un duo de joueurs de reference ;
2. tests UI sur joueur absent, joueur prive, joueur identique, donnees asymetriques ;
3. test de latence : P95 < 5s avec Waypoint nominal ;
4. test multi-titre : meme endpoint avec `X-LevelUp-Title: halo_infinite` et un futur titre — les KPIs communs doivent etre coherents, `Extended` doit absorber les divergences.

---

## O6 — Export service record / profil

### Decision produit

Ecarte definitivement.

### Pourquoi

1. la valeur utilisateur parait faible par rapport au reste du backlog ;
2. l'export d'historique couvre deja le besoin d'export principal.

---

## O7 — CSR leaderboards

### Pourquoi c'est interessant

1. ouvre une surface communautaire ;
2. donne une valeur reutilisable pour home, squad et compare ;
3. exploite naturellement les seasons CSR.

### Architecture : meme modele que O5 Compare

Le leaderboard n'est pas limite aux joueurs deja synces. Il adopte exactement la meme
architecture que O5 :

1. les joueurs presents dans DuckDB (`match_participants`, `xuid_aliases`) sont charges
   localement via `LeaderboardRepository` — acces rapide, meme chemin que les repos existants ;
2. les joueurs hors DuckDB font l'objet d'appels `PlayerStatsProvider` (meme interface que O5)
   groupes en batch via goroutines ;
3. le resultat est assemble dynamiquement, sans ecriture supplementaire pour les joueurs non locaux.

Avantage : les joueurs synces apparaissent immediatement (chargement progressif cote React),
les autres arrivent en complement. Cela renforce la valeur du sync pour les utilisateurs reguliers.

### Architecture multi-titre

Le `titleSlug` est propage depuis le contexte exactement comme dans O5. Les couches sont
les memes — `PlayerStatsProvider` est partage entre Compare et Leaderboards.

#### Reutilisation directe depuis O5

| Element | Statut |
|---------|--------|
| `NormalizedPlayerStats` (domain/) | Partage avec O5 — aucun doublon |
| `PlayerStatsProvider` (port/) | Partage avec O5 — meme interface |
| `HaloCompareProvider` (platform/halo/) | Reutilise directement |
| `TitleSlug` depuis contexte | Propage via middleware existant |

#### Couche specifique Leaderboards

```go
// port/repository.go — ajout
type LeaderboardRepository interface {
    // Renvoie les joueurs locaux tries par CSR desc pour un titre et une saison donnes
    // titleSlug depuis ctxkeys.TitleSlug(ctx)
    GetLocalRankings(ctx context.Context, req domain.LeaderboardRequest) ([]domain.LeaderboardEntry, error)
}

// domain/leaderboard.go
type LeaderboardEntry struct {
    XUID      string
    Gamertag  string
    TitleSlug string
    CSR       *float64
    Playlist  string
    Season    string
    IsLocal   bool   // true si charge depuis DuckDB, false si Waypoint a la volee
}

type LeaderboardRequest struct {
    TitleSlug string
    Playlist  string
    Season    string  // depuis metadata.duckdb via O2
    Limit     int
}
```

#### LeaderboardService

```go
// service/leaderboard_service.go
type LeaderboardService struct {
    repo     port.LeaderboardRepository  // joueurs locaux depuis DuckDB
    provider port.PlayerStatsProvider    // joueurs distants via Waypoint (partage O5)
}

func (s *LeaderboardService) GetLeaderboard(ctx context.Context, req domain.LeaderboardRequest) (domain.LeaderboardResponse, error) {
    // 1. charge joueurs locaux (rapide, DuckDB)
    // 2. enrichit avec joueurs Waypoint en batch goroutines
    // 3. trie, pagine, retourne
    // titleSlug depuis ctxkeys.TitleSlug(ctx)
}
```

### Plan d'implementation

1. Implementer apres O5 : `PlayerStatsProvider` et `NormalizedPlayerStats` sont deja disponibles.
2. Ajouter `LeaderboardEntry`, `LeaderboardRequest`, `LeaderboardResponse` dans `domain/leaderboard.go`.
3. Ajouter `LeaderboardRepository` dans `port/repository.go`.
4. Implementer `LeaderboardRepo` dans `platform/duckdb/leaderboard_repo.go`.
5. Implementer `LeaderboardService` dans `service/leaderboard_service.go`.
6. Enregistrer dans `ServiceRegistry`.
7. Ajouter handler et route `GET /api/v1/leaderboard` (ou bloc dans home/career selon la maturite).
8. Cote React, module compact avec chargement progressif (joueurs locaux → Waypoint).

### Parcours UI recommande

Le leaderboard n'a pas de point d'entree dedie dans la navigation. Il emerge depuis les
surfaces existantes sous forme de bloc compact, et peut gagner une route secondaire
seulement si l'usage le justifie.

#### Parcours principal — bloc dans Career ou Home

```
Career
  → Bloc "Position dans ta cohorte" (sous les KPIs principaux)
    → Joueurs locaux affiches immediatement (DuckDB, acces < 50ms)
    → Joueurs Waypoint charges en complement (skeleton par ligne)
    → CTA "Voir plus" → route secondaire /leaderboard si usage confirme
    → Hover/clic sur un joueur → CTA "Comparer" → Drawer Compare (reutilise O5)

Home
  → Module editorial "Top CSR" (carte secondaire optionnelle)
    → meme logique de chargement progressif
```

#### Chargement progressif : pourquoi c'est naturel ici

Les joueurs locaux (`xuid_aliases`, `match_participants`) sont charges depuis DuckDB —
acces < 50ms, pas de Waypoint. Le module s'affiche instantanement avec ces donnees,
puis enrichit progressivement avec les joueurs Waypoint distants. L'utilisateur percoit
une page rapide meme si le fetch Waypoint prend 2-4s.

```
Mount du bloc Leaderboard
  → query locale DuckDB → affichage immediat (joueurs synces, IsLocal=true)
  → batch Waypoint goroutines → mise a jour progressive (joueurs distants, IsLocal=false)
  → tri final quand tout est charge
```

#### Ce qu'il ne faut pas faire

Ne pas attendre la fin du fetch Waypoint pour afficher quoi que ce soit — le bloc serait
vide pendant 2-4s sur Home ou Career, ce qui est inacceptable.

### Strategie de prefetch React

#### Opportunites de prefetch

| Declencheur | Donnees disponibles | Gain | Note |
|-------------|---------------------|------|------|
| Mount de CareerPage | `queryKeys.home` deja en cache (KPIBar l'a charge avant) | Fort | Prefetch leaderboard local en parallele du rendu Career |
| Hover lien "Voir plus" | Joueurs locaux deja visibles | Faible | Seul Waypoint manque, deja en cours |
| Hover ligne joueur dans le bloc | Gamertag connu | Maximal | Prefetch Compare via `useComparePrefetch` (partage avec O5) |

#### Prefetch au mount de CareerPage (cas le plus utile)

`queryKeys.home(playerSlug)` est deja en cache quand CareerPage monte (KPIBar le charge
en premier). On peut profiter de ce moment pour lancer le fetch du leaderboard local
en parallele, avant meme que l'utilisateur scrolle :

```typescript
// Dans CareerPage, apres le fetch career principal
useEffect(() => {
  if (!playerSlug) return
  queryClient.prefetchQuery({
    queryKey: queryKeys.leaderboard(playerSlug, { season: currentSeason }),
    queryFn: () => api.get(`/players/${playerSlug}/pages/leaderboard`),
    staleTime: 5 * 60 * 1000,
  })
}, [playerSlug, currentSeason])
```

La partie locale (DuckDB) etant instantanee cote serveur, le prefetch au mount garantit
que le bloc s'affiche sans delai percu quand l'utilisateur scrolle jusqu'a lui.

#### Reutilisation depuis O5

Le hover sur une ligne du leaderboard doit declencher le prefetch Compare via le meme
`useComparePrefetch` hook defini dans O5 — zero doublon, meme pattern `onMouseEnter`.

### Validation

1. tests sur tri, egalites, pagination et saison vide ;
2. verification metadonnees CSR via O2 (saisons disponibles en DuckDB) ;
3. test de chargement progressif : joueurs locaux visibles avant la fin du fetch Waypoint ;
4. test prefetch : leaderboard deja en cache quand l'utilisateur scrolle vers le bloc Career ;
5. test multi-titre : leaderboard filtre correctement par `titleSlug`, pas de melange entre titres.

---

## O8 — Asset discovery + versioning (anticipe multi-titre)

### Pourquoi c'est interessant

1. utile pour maps, playlists, map-mode pairs et UGC ;
2. accelere la maintenance metadata sans hardcode fragile ;
3. facilite l'extension multi-titre si de nouveaux jeux Halo sont supportes.

### Usage : outillage de verification interne, pas sync automatique

L'asset discovery n'est pas un pipeline de mise a jour automatique. C'est un outil de verification :

```
CLI Go -> fetch asset Waypoint
       -> comparer avec ce qui est en metadata.duckdb
       -> rapport d'ecart : nouveau / modifie / supprime
       -> promotion manuelle uniquement apres validation
```

Aucune ecriture automatique en production sans validation humaine.

### Architecture multi-titre anticipee

Comme O3, tous les assets stockes doivent inclure `title_id` comme cle de partition des la premiere implementation, meme si Halo Infinite est le seul titre aujourd'hui.

### Plan d'implementation

1. Ajouter une couche interne de discovery metadata non exposee directement aux utilisateurs.
2. Persister `asset_id`, `version_id`, `kind`, `title_id`, `labels` et dates utiles dans `metadata.duckdb`.
3. Construire un pipeline de verification incremental par type d'asset (diff uniquement, pas de sync aveugle).
4. Reutiliser ces assets dans match view, playlists et futures pages.

### Validation

1. tests de mapping par `AssetKind` ;
2. test de blocage si le niveau d'information est insuffisant (champs manquants ou assets partiels) ;
3. policy de cache et invalidation documentee.

---

## O9 — Year in Review

### Pourquoi c'est interessant

1. forte valeur narrative et partageable ;
2. reutilise beaucoup de briques deja presentes ;
3. donne un produit signature si le rendu est soigne.

### Condition de deblocage

O9 est bloque jusqu'a :

1. O2 en production (seasons fiables en DuckDB) ;
2. audit de coverage confirme : les joueurs cibles ont des donnees syncees couvrant l'annee entiere visee.

Sans ces deux conditions, la page risque d'etre creuse et de decevoir.

### Signal externe utile

1. page `YearInReview.tsx` de SpartanRecord ;
2. callouts pour playtime, matches, career rank, medals, kill breakdown.

### Plan d'implementation

1. Cadrer un MVP annuel avec une seule annee supportee au debut.
2. Definir un endpoint `GET /api/v1/players/{player_slug}/pages/year-in-review?year=YYYY`.
3. Reutiliser O2 pour reconstituer les saisons couvrant une annee donnee.
4. Agreger : matchs joues, temps de jeu, winrate, KDA, armes ou medailles marquantes, progression de rang, meilleurs matchs.
5. Cote React, privilegier une page partageable plutot qu'un simple dashboard dense.

### Materialisation UI recommandee

1. ne pas ajouter ce parcours dans le menu principal ;
2. le lancer depuis une carte temporaire sur `home` ou `career` quand l'annee est disponible ;
3. assumer une route deep-linkable partageable, mais decouverte contextuelle.

### Validation

1. snapshots de payload pour une annee connue ;
2. tests sur annee vide ;
3. tests sur annee partiellement couverte.

---

## O10 — Store / economy tracker

### Statut

Backlog multi-titre. Hors scope pour Halo Infinite aujourd'hui.

### Pourquoi pas maintenant

1. le store Halo Infinite ralentit en fin de cycle commercial ;
2. risque d'obsolescence avant livraison pour ce titre specifique ;
3. aucun atterrissage UI propre sans diluer la promesse analytics de LevelUp.

### Pourquoi garder en backlog

Si un nouveau titre Halo dispose d'une economie de store active (cosmétiques, battle pass,
rotations), O10 devient pertinent sans restructuration majeure — l'architecture multi-titre
(`title_id` dans toutes les tables) le rend naturellement extensible.

### Conditions de deblocage

1. onboarding d'un nouveau titre avec economie active confirmee ;
2. signal utilisateur explicite sur l'interet du tracking store pour ce titre ;
3. atterrissage UI valide comme module optionnel dans `Home`, jamais comme menu prioritaire.

---

## O11 — Spartan Company / social layer

### Statut

Backlog multi-titre. Hors scope pour Halo Infinite aujourd'hui.

### Pourquoi pas maintenant

1. aucun signal utilisateur explicite justifiant le chantier sur Halo Infinite ;
2. la dimension groupe est deja couverte partiellement par `Squad` pour les besoins actuels.

### Pourquoi garder en backlog

Si un nouveau titre Halo dispose d'une dimension clan ou groupe etablie nativement
(guildes, companies, escouades persistantes), O11 devient le complement naturel de `Squad`.
L'architecture multi-titre permet de le scoper par titre sans impacter Halo Infinite.

### Conditions de deblocage

1. onboarding d'un nouveau titre avec dimension groupe native confirmee ;
2. ou signal utilisateur explicite sur la gestion de groupes dans LevelUp ;
3. dans tous les cas : rattacher a `Squad` ou a une surface groupe existante, jamais comme sous-produit social autonome.

---

## O12 — Ban diagnostics

### Pourquoi c'est interessant

1. utile pour support et diagnostic ops ;
2. peut expliquer des comportements anormaux cote provider ;
3. faible priorite produit mais forte utilite pour les investigations.

### Signal externe utile

1. `bansummary` ;
2. `banning/file/{banPath}` exposes par `halo-infinite-api`.

### Plan d'implementation

1. Implementer derriere une route admin protegee, jamais en UI publique.
2. Exposer un diagnostic lisible, pas les payloads bruts par defaut.

### Validation

1. tests sur 404 / 403 / message absent ;
2. audit de ce qui est acceptable d'afficher en route admin.

---

## O13 — Waypoint Explorer interne

### Pourquoi c'est interessant

1. reduit le cout de decouverte de nouvelles ressources ;
2. accelere les spikes sans scripts ad hoc ;
3. complement naturel de O8 pour l'outillage metadata interne.

### Signal externe utile

1. mode GET / SCAN de HaloInfiniteGetter ;
2. navigation ressource -> sous-ressources.

### Plan d'implementation

1. Creer un outil interne, pas une feature utilisateur finale.
2. Le cadrer comme un panneau dev ou une route admin protegee.
3. Fournir deux operations : fetch d'une ressource exacte et scan recursif d'un JSON.
4. Stocker les ressources dans un cache local versionne et navigable.
5. Permettre l'export d'un snapshot pour fixtures et audits.

### Chantiers techniques

1. route admin Go ou script CLI ;
2. stockage local sous `data/cache/waypoint_explorer/` ou equivalent ;
3. UI minimale React optionnelle, ou simple CLI au debut.

### Validation

1. scan d'un calendar ;
2. detection des ressources deja vues ;
3. export d'un snapshot reproductible.

---

## O14 — ETag cache + snapshots de ressources

### Decision

Bundle dans O2. Pas d'opportunite separee.

Voir section O2 pour le plan complet d'implementation du suivi ETag, des snapshots et de la notification Discord en cas de changement de ressource critique.

---

## Ordre d'implementation recommande

### Lot A — 2 a 3 sprints

1. O2 Season calendars + CSR calendars (bundle O14 ETag + snapshots)
2. O1 Match privacy
3. O5 Compare MVP (appel Waypoint a la volee, skeleton loader React)

### Lot B — 2 sprints

1. O3 Medals metadata (staging + garde-fous + socle multi-titre)
2. O8 Asset discovery (outil de verification interne, socle multi-titre)
3. O13 Waypoint Explorer interne

### Lot C — 2 a 3 sprints

1. O7 CSR leaderboards (architecture dynamique, chargement progressif)
2. O9 Year in Review (si O2 en prod + coverage annuelle confirmee)
3. O12 Ban diagnostics (route admin)

## Recommendation finale

1. fiabiliser les metadonnees externes en premier (O2 + O14 bundle) ;
2. exposer proprement la privacy (O1) ;
3. lancer Compare comme mode contextuel depuis `career` ou `squad` (O5) ;
4. poser le socle multi-titre pour O3 et O8 en V2 avec garde-fous stricts ;
5. n'ouvrir leaderboards (O7) et Year in Review (O9) qu'une fois leurs conditions de deblocage respectees.

---

## Note UX et direction design

Cette section fixe une direction de design pour que les opportunites retenues renforcent le produit sans degrader la navigation.

Important : a ce stade, l'application n'a pas une page `joueur` autonome au sens strict. Elle a un scope joueur avec plusieurs destinations dediees : `home`, `career`, `stats/history`, `squad`, `media`, `profile/citations`. Les recommandations ci-dessous parlent donc de points d'entree dans ces surfaces existantes.

### Principes directeurs

1. moins de destinations, plus de profondeur ;
2. une page = une intention dominante ;
3. densite oui, bruit non : information structuree par rythme visuel, tailles, contrastes et respirations ;
4. les parcours secondaires doivent etre contextuels : drawer, sheet, tabs, panneaux lies a la page courante avant de meriter une route dediee ;
5. le ton visuel doit etre premium et assume : interface nette, technique, precise, avec une personnalite Halo implicite mais jamais cosplay ;
6. toute nouvelle feature doit renforcer le sentiment de lecture, de maitrise et de progression.

### Langage visuel recommande

1. base claire et tendue, contraste net, surfaces bien delimitees ;
2. palette sobre : fonds lumineux ou graphite clair, accents acier, olive, sable, cyan technique ou orange signal selon les cas ;
3. typographie expressive avec hierarchie marquee entre titres, metriques et texte de contexte ;
4. animations avec parcimonie : reveal a l'arrivee, transitions de panneaux, skeletons soignes.

### Hierarchie de navigation recommandee

1. la navigation primaire doit rester courte et memorisable ;
2. `Compare`, `Leaderboards` et `Year in Review` ne doivent pas entrer dans la navigation principale au premier niveau ;
3. `Compare` s'invoque depuis `Career`, `Squad` ou `Profile/Citations` ;
4. `Leaderboards` emerge comme vue secondaire depuis `home` ou `career` ;
5. `Year in Review` se decouvre via une campagne ou une carte saisonniere, puis vit comme destination partageable.

### Page par page

#### Home

Role : page d'entree, de situation et d'orientation.

1. raconter l'etat du joueur en quelques secondes : forme recente, objectifs, signaux importants ;
2. modules peu nombreux mais forts : recent performance, dernier match important, alertes, progression notable ;
3. `Leaderboards` et `Year in Review` peuvent apparaitre ici sous forme de cartes editoriales ou de modules temporaires.

Direction visuelle : grande respiration, hero fort, cartes editoriales, forte sensation de cockpit ou de briefing.

#### Career

Role : page de reference du joueur, lecture stable de sa progression.

1. destination la plus solide et la plus comprehensible du produit ;
2. KPIs principaux en haut avec hierarchie tres claire ;
3. CTA `Comparer` dans l'en-tete ou a cote du bloc identitaire joueur ;
4. `Leaderboards` comme vue secondaire ou bloc `Position dans ta cohorte`.

Direction visuelle : institutionnelle, stable, strates nettes, sections analytiques.

#### Match History

Role : exploration chronologique et acces au detail.

1. flux maitrisable, filtres visibles mais calmes ;
2. privacy en warning elegant, explicite et non alarmiste ;
3. lignes ou cartes de match mettant en avant le signal utile avant la masse de statistiques.

Direction visuelle : flux, cadence, lisibilite, excellents etats vides et limites.

#### Match View

Role : theatre d'operations du match individuel.

1. entree narrative immediate : contexte, issue, moment cle, lecture rapide ;
2. warnings de privacy integres dans la lecture, pas jetes comme erreurs systeme ;
3. statistiques detaillees apres une couche de synthese.

Direction visuelle : immersive, dramatique, cinematographique, mais precise et propre.

#### Squad

Role : page relationnelle, analytique.

1. montrer avec qui le joueur performe, perd, progresse ou se stabilise ;
2. `Compare` part naturellement d'ici, en comparant le joueur avec un coequipier recurrent ;
3. si O11 revient un jour, c'est ici qu'il devra etre rattache.

Direction visuelle : relationnelle, matricielle, clarte et tri visuel forts.

#### Compare

Role : mode d'analyse laterale, lecture d'ecarts et de profils.

1. meilleur format initial : drawer large, sheet ou split view gardant le contexte du joueur de depart ;
2. skeleton loader pendant le fetch Waypoint (~2-4s) ;
3. KPI peu nombreux et impeccablement racontes ;
4. si forte recurrence d'usage, peut gagner une route dediee apres validation.

Direction visuelle : dualite gauche/droite claire, emphasis forte sur les deltas et les points de bascule.

#### Leaderboards

Role : preuve sociale et positionnement relatif.

1. premiere incarnation : module compact dans `Home` ou `Career` ;
2. affichage progressif : joueurs locaux d'abord, joueurs Waypoint en complement ;
3. cle UX : montrer ou se situe le joueur par rapport a sa cohorte, pas afficher une table brute ;
4. lien `Voir plus` si le module est frequemment utilise.

Direction visuelle : editorial, mise en scene des positions, deltas et seuils plutot qu'une table generique.

#### Year in Review

Role : grande page narrative, partageable, emotionnelle et retrospective.

1. experience a part, ne doit pas polluer la navigation courante toute l'annee ;
2. decouverte via une carte home, un bandeau de saison ou un lien partage ;
3. hierarchie alternant moments forts, chiffres clefs, progression et signatures de jeu ;
4. rythme editorial, pas dashboard.

Direction visuelle : forte ambition visuelle, sections pleine largeur, respirations larges, transitions marquees, storytelling premium.

### Regles de qualite pour le frontend

1. une action importante doit toujours etre visible dans la zone haute de la page ;
2. une page dense doit toujours proposer une lecture rapide avant la lecture experte ;
3. un etat vide ou partiel doit rester beau, clair et volontaire ;
4. les warning states doivent etre traites comme une partie du produit, pas comme des erreurs de dev ;
5. toute nouvelle composante doit justifier son existence par sa valeur de lecture.

### Consequence sur les opportunites ouvertes

1. O1 doit prioriser la qualite de mise en scene des limites de donnees ;
2. O2 et O3 doivent rester largement invisibles mais hausser la qualite percue des ecrans existants ;
3. O5 doit viser une experience comparative premium avec skeleton loader soigne ;
4. O7 doit privilegier le chargement progressif et la pertinence de la cohorte affichee ;
5. O9 ne doit etre ouvert qu'avec une intention editoriale forte et une coverage de donnees confirmee.

---

## Schemas d'atterrissage UI

Convention de lecture :

1. `Shell` = point d'entree principal dans l'application ;
2. `Module` = bloc dans une page existante ;
3. `CTA` = point d'entree actionnable ;
4. `Route secondaire` = destination possible, pas menu de premier niveau.

Ne sont pas inclus ici : O2, O3, O8 (metadata invisible), O12, O13, O14 (admin / outillage), O4, O6, O10, O11 (ecartes).

### O1 - Match privacy

```text
Shell
    -> Match History
        -> Bandeau "Historique limite / prive / partiel"
            -> Lignes de match avec etats de donnees clairs
                -> Match View
                    -> Warning integre + sections degradees elegantement

Et en parallele :
Bootstrap / Home
    -> signal discret "certaines donnees sont limitees"
        -> renvoi vers Match History
```

But UX : expliquer la limite de donnees sans transformer le produit en ecran d'erreur.

### O5 - Compare joueur vs joueur

```text
Explorer → Mode Joueur → Card bilan face-a-face
    -> CTA "Comparer" (joueur B deja resolu, pas de nouvelle saisie)
        -> Drawer Compare
            -> Skeleton loader (~2-4s fetch Waypoint)
                -> Stats cote a cote, KPIs normalises
                    -> option future : route dediee si usage confirme

ou

Career
    -> CTA "Comparer" dans l'en-tete joueur
        -> GamertagSearchInput (composant existant)
            -> meme Drawer Compare

ou

Squad
    -> Ligne coequipier recurrent → CTA "Comparer"
        -> meme Drawer Compare (joueur B pre-rempli)
```

But UX : Explorer est le point d'entree prioritaire (contexte matchs communs deja charge).
Career et Squad sont des entrees complementaires. Jamais de route globale autonome.

### O7 - CSR leaderboards

```text
Career (mount)
    -> prefetch leaderboard local en parallele (DuckDB, < 50ms)
        -> Bloc "Position dans ta cohorte"
            -> joueurs locaux affiches immediatement (IsLocal=true)
            -> joueurs Waypoint charges en complement (skeleton par ligne)
                -> hover ligne joueur → prefetch Compare (useComparePrefetch, partage O5)
                -> CTA "Comparer" → Drawer Compare

    -> CTA "Voir plus" (si usage confirme)
        -> Route secondaire /leaderboard

Home
    -> Module editorial "Top CSR" (carte optionnelle)
        -> meme chargement progressif
```

But UX : joueurs locaux visibles instantanement, Waypoint arrive en complement sans bloquer.
Preuve sociale immediate, latence percue quasi nulle grace au prefetch au mount de Career.

### O9 - Year in Review

```text
Home
    -> Carte saisonniere / annuelle (conditionnelle : O2 en prod + coverage confirmee)
        -> CTA "Voir ton annee"
            -> Route dediee shareable Year in Review
                -> lecture narrative par chapitres
                    -> retour vers Career / Home

ou

Career
    -> Carte editoriale ponctuelle "Ton recap annuel"
        -> meme route shareable
```

But UX : experience evenementielle et memorisable, pas un onglet permanent.

### Lecture d'ensemble

```text
Home
    -> Leaderboards (module, chargement progressif)
    -> Year in Review (carte ponctuelle, conditionnel)
    -> signal privacy (si necessaire)

Career
    -> Compare (CTA principal)
    -> Leaderboards (bloc secondaire, chargement progressif)
    -> Year in Review (carte ponctuelle, conditionnel)

Match History / Match View
    -> Privacy (warning integre)

Squad
    -> Compare (point d'entree relationnel)
```

Synthese :

1. O1 s'ancre dans `Match History` et `Match View` ;
2. O5 s'ancre dans `Explorer` (prioritaire), `Career` et `Squad`, assure la latence Waypoint avec un skeleton loader ;
3. O7 commence dans `Home` ou `Career` avec chargement progressif, peut gagner une route secondaire ;
4. O9 commence dans `Home` ou `Career`, vit comme experience shareable, conditionne a la coverage ;
5. O10 et O11 : pas d'atterrissage UI pour Halo Infinite — backlog multi-titre, a debloquer sur signal utilisateur explicite lors de l'onboarding d'un nouveau titre.
