# ADR 0011 — Séparation canonical / semantic adapter pour la migration P4

**Status** — Proposed (2026-04-29). Pré-requis pour P4.1 pilote home (`PLAN_ACTION.md`, ADR 0007).

**Deciders** — Guillaume (GS).

## Context

ADR 0002 a posé `canonical.PlayerMatchRow` comme contrat cross-titres pour
les services produit. ADR 0007 prévoit un big-bang sur 13-15 services.

En préparant le pilote `home_service.go`, un blocker structurel est apparu
(documenté dans le code — `home_service.go:36-42`) :

> *« À ce jour, le service utilise le repo direct car canonical.PlayerStats ne
> couvre pas encore la totalité du payload home KPIs (favorite_playlist,
> avg_kda, etc.). »*

`domain.HomeMatchRow` a ~50 champs Halo-spécifiques que `canonical.PlayerMatchRow`
ne couvre pas, en particulier :

- **Labels i18n localisés** : `MapNameFR`, `PairNameFR`, `PlaylistNameFR`,
  `GameVariantNameFR`, `SkillTierLabel`, etc.
- **Skill enrichi** : `SkillTier`, `SkillSubTier`, `SkillRankImageURL`,
  `SkillPlaylistGroup`
- **Asset URLs** : `SkillRankImageURL`, et indirectement les images de map/mode
- **View-model formaté** : noms de colonnes, FR/EN par row

**Hypothèse architecturale** (à confirmer / acter par cet ADR) : ces champs
ne devraient PAS aller dans canonical. Ils relèvent de :

- `TitleSemanticAdapter` pour les labels i18n (FR/EN, RankCatalog, OutcomeLabels)
- `TitleAssetURLAdapter` pour les URLs (MapImageURL, MedalImageURL, CSRRankImageURL)
- View-models côté service (formatage final pour le DTO HTTP)

Si on étend `canonical` pour absorber ces champs, on pollue le contrat
cross-titres avec des préoccupations title-specific (Halo a un système Onyx,
MCC a probablement autre chose, etc.).

## Decision

**Frontière nette entre les 4 couches** :

```
                          ┌──────────────────────────────────────┐
                          │  Service produit (home_service)      │
                          │  combine les 3 sources                │
                          └────────────┬─────────────────────────┘
                                        │ consomme
                ┌───────────────────────┼───────────────────────┐
                │                       │                       │
        ┌───────▼────────┐    ┌─────────▼─────────┐    ┌────────▼─────────┐
        │ TitleData      │    │ TitleSemantic     │    │ TitleAssetURL    │
        │ Adapter        │    │ Adapter           │    │ Adapter          │
        │                │    │                   │    │                  │
        │ data brute     │    │ labels i18n       │    │ URLs statiques   │
        │ (canonical)    │    │ (Fields, Ranks,   │    │ (Map, Medal,     │
        │                │    │  Assets, Outcomes)│    │  CSRRank, etc.)  │
        └────────────────┘    └───────────────────┘    └──────────────────┘
```

### Règles de placement des champs

| Type de champ | Va dans | Justification |
|---|---|---|
| Compteur métier universel (kills, deaths, assists, accuracy ratio, KDA) | **canonical** | Existe dans tout FPS/PvP |
| ID stable d'asset (mapID, playlistID, medalID) | **canonical.AssetReference.ID** | Identifiant brut, pas de localisation |
| Label par défaut (DefaultLabel sans locale) | **canonical.AssetReference.DefaultLabel** | Fallback universel |
| Label localisé (NameFR, NameEN) | **TitleSemanticAdapter.Fields/Ranks/Assets** | i18n résolu par titre |
| Tier de skill (string brut) | **canonical.MatchSkillSnapshot.MMR/...** ou non exposé | Universel mais peut nécessiter extension |
| Label de tier (« Onyx 1500 ») | **TitleSemanticAdapter.Ranks** | Halo-spécifique (MCC a un système différent) |
| URL d'image asset | **TitleAssetURLAdapter.{Map,Medal,CSRRank}ImageURL** | Composé par adapter, jamais en DB |
| Outcome int | **canonical.Outcome** (string : win/loss/tie/dnf) | Mappé via canonical/enums.go |
| Outcome label localisé | **TitleSemanticAdapter.Outcomes** | i18n par titre |
| Format de présentation (« 5/2 K/D ») | **View-model côté service** | Pas de pré-formatage en API |

### Pattern de consommation par un service produit (Home, Synthesis, etc.)

```go
type HomeService struct {
    repo     port.PlayerMatchesRepository  // canonical
    semantic games.TitleSemanticAdapter    // labels i18n
    assetURL games.TitleAssetURLAdapter    // URLs assets
    // + dépendances spécifiques (cache, sink, etc.)
}

func (s *HomeService) Build(ctx context.Context, slug, gamertag string) (*HomeResponse, error) {
    // 1. Charger les matchs canoniques (data brute)
    matches, err := s.repo.LoadPlayerMatches(ctx, slug, gamertag, filters)
    if err != nil {
        return nil, err
    }

    // 2. Résoudre les labels i18n via semantic adapter
    fields := s.semantic.Fields()    // FieldMappingSet (kills, deaths, ...)
    ranks := s.semantic.Ranks()      // RankCatalog (Bronze 1, Silver 2, ...)
    assets := s.semantic.Assets()    // AssetMappingSet (map names, etc.)

    // 3. Composer les URLs assets via assetURL
    for _, m := range matches {
        mapURL := s.assetURL.MapImageURL(m.Summary.Map.ID)
        // ...
    }

    // 4. Combiner pour produire le DTO HTTP final (forme inchangée)
    return buildHomeResponse(matches, fields, ranks, assets, mapURLs), nil
}
```

### Extensions canonical vraiment nécessaires (à confirmer par P4_GAP_ANALYSIS.md)

Si l'analyse de gap montre des champs qui n'entrent dans aucune des 3 catégories
(canonical / semantic / assetURL / view-model), ils peuvent être ajoutés à
`canonical.PlayerMatchRow` SOUS RÉSERVE :

1. Le champ est universel (pas Halo-only).
2. Le champ est primitif (int, string ID, ratio) — pas un label localisé.
3. Le champ est utilisé par au moins 2 services différents (justifie le coût d'extension).

Candidats possibles (à valider) :
- `Self.PerformanceScore *float64` — déjà dans `Enrichment.PerformanceScore`, OK
- `Self.AvgLifeSeconds *float64` — déjà partiel dans `MatchParticipant.AvgLifeSeconds`
- `Summary.IsRanked *bool` — déjà présent
- `Summary.IsPvE *bool` — déjà présent

**Hypothèse** : très peu d'extensions canonical sont réellement nécessaires.
La majorité des « champs manquants » de HomeMatchRow vont en semantic / assetURL.

## Consequences

### Positive

- **Frontière nette** : un service produit sait clairement où chercher chaque type de donnée.
- **Canonical reste minimal** : pas de pollution par les concerns Halo-specific (NameFR, SkillTier, RankImageURL).
- **Multi-titres immédiat** : chaque nouveau titre fournit son propre `SemanticAdapter` + `AssetURLAdapter` sans toucher canonical.
- **Tests simplifiés** : on mock une interface à la fois (data, semantic, ou assetURL) selon ce qu'on teste.
- **Effort P4 révisé à la baisse** : pas de gros refactor canonical, juste du wiring entre les 3 adapters côté services.

### Negative

- **Ajout de complexité côté service** : home_service passe de 2 dépendances (repo + provider) à 3 dépendances (data + semantic + assetURL). Documenté + pattern stable.
- **Risque de divergence** : un service pourrait charger un mapID via `data` mais oublier de résoudre le label via `semantic`. Mitigation : tests de contrat sur les DTOs HTTP émis (snapshot avant/après).
- **Migration plus longue par service** : chaque service doit injecter 2-3 adapters au lieu d'1 repo. Refactor des constructeurs.

### Critères pour évaluer si Option A vs B

Cet ADR rend Option A (extension canonical légère + adapter semantic plein) :
- **Plus court terme** que l'estimation initiale 2-3 sem (extension canonical brute)
- **Comparable à Option B** sur synthesis (1-2 j) si on accepte les 3 adapters côté service
- **Strictement supérieur à Option B** long terme : pas de dette à reprendre

## Alternatives evaluated

| Alternative | Rejected because |
|---|---|
| **Étendre canonical avec NameFR/SkillTier/...** | Pollue le contrat cross-titres avec du Halo-specific. Le 2e titre cassera. |
| **Garder les `domain.*MatchRow` parallèles** | C'est l'état actuel. Bloque le multi-titres au-delà des 3 services migrés (Squad V2, Match History, Explorer). |
| **Tout passer par TitleSemanticAdapter** (y compris kills/deaths) | Mélange data brute et présentation. Pas le rôle du semantic adapter. |
| **Créer un 4e adapter `TitleViewModelAdapter`** | Surdimensionné : la composition view-model est de la responsabilité du service, pas d'un adapter. |

## References

- ADR 0002 — `canonical-player-match-row.md` (contrat initial)
- ADR 0007 — `canonical-bigbang-migration.md` (plan migration)
- ADR 0008 — `db-schema-multi-title-and-xuid-global.md` (isolation par chemin FS)
- Plan : `.ai/review/2026-04-29/PLAN_ACTION.md` P4
- Gap analysis : `.ai/review/2026-04-29/P4_GAP_ANALYSIS.md` (livré séparément, valide ou ajuste les hypothèses de cet ADR)
- Code : `apps/go-api/internal/games/adapter.go` (3 interfaces définies, déjà en place)
- Implémentations existantes : `apps/go-api/internal/games/halo_infinite/{adapter_data,adapter_semantic,adapter_asset_url}.go`
