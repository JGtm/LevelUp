# INVENTAIRE V8a — Types de réponse API hand-written front ↔ contrat back

> Livrable du lot V8a (PLAN_CLOTURE_AUDITS_2026-07). Balayage de tous les types de
> RÉPONSE API définis à la main dans `apps/web/src` (non ré-exportés de `generated.ts`)
> consommés par `api.get<X>` / `api.post<X>`, comparés champ par champ à `generated.ts`
> (dérivé d'`openapi.yaml`) ET aux structs Go réellement sérialisées.
> Date : 2026-07-07.

## Méthode

- Source des types : `apps/web/src/lib/api/types.ts` (majorité) + fichiers feature-locaux
  (`features/*/types.ts`, `features/*/queries.ts`, `lib/i18n/fieldMappings.ts`).
- La grande majorité des types de réponse sont DÉJÀ ré-exportés du contrat
  (`type X = components['schemas'][...]`) — fidèles par construction, hors périmètre de risque.
- Le risque `undefined` silencieux vit dans les types HAND-WRITTEN (interface locale ou
  alias non-`components[...]`) consommés par `api.get/post`.

## DIVERGENCES CONFIRMÉES (cause d'`undefined` runtime)

| # | Type | Endpoint | Divergence : déclaré front vs réel contrat | Gravité | Statut |
|---|------|----------|--------------------------------------------|---------|--------|
| 1 | `CareerTopMatchesResponse` | GET /pages/career/top-matches | front `{ items: CareerTopMatch[] }` ; réel `{ best_matches, worst_matches: TopMatchDTO[] }` | undefined runtime (`.items` toujours undefined) | **CORRIGÉ V8b** |
| 2 | `CareerPageResponse.top_matches_preview` | GET /pages/career | champ lu par `CareerPage.tsx` ; ABSENT du struct Go `CareerPageResponse` | champ fantôme (section top matchs jamais rendue au 1er chargement) | **CORRIGÉ V8b** |
| 3 | `CareerPageResponse.encounters_preview` | GET /pages/career | champ lu par `CareerPage.tsx` ; ABSENT du struct Go | champ fantôme (encounters preview toujours `[]`) | **CORRIGÉ V8b** |
| 4 | `CareerEncountersResponse` | GET /pages/career/encounters | front `{ items: CareerEncounter[] }` ; réel `{ teammates, enemies: EncounterDTO[], total }` | undefined runtime (`.items` undefined + shape `EncounterDTO` ≠ `CareerEncounter`) | **CORRIGÉ V8b** |

Note : le composant `CareerTopMatchesTable` consommait `CareerTopMatch` (schéma canonique
RICHE : `variant`, `start_time`, `assists`, `kd_ratio`, `score_label`, `badge_type`) alors
que l'endpoint sert `TopMatchDTO` (pauvre). Même avec les types corrigés, ces champs
seraient restés undefined. Fix V8b : table alignée sur `TopMatchDTO` ; `start_time` ajouté
au DTO Go (trivialement disponible dans `TopMatchRawRow`) pour préserver la colonne date.
`CareerEncountersSection` consommait `CareerEncounter` (`wins`/`losses`/`last_seen_at`)
absent d'`EncounterDTO` — section réalignée sur `EncounterDTO` (`as_teammate`/`as_enemy`/`avg_kda`).

## DIVERGENCES ADDITIONNELLES DÉCOUVERTES (hors périmètre A2 — NON traitées, → Découvertes)

Trouvées pendant le balayage V8a (audit sous-agent + vérification sur pièces). Réelles mais
HORS du cas prouvé A2 (Career) → consignées, pas corrigées (plan-execution règle 7). À
trancher (décision produit/backend) dans un lot ultérieur.

| # | Type | Endpoint | Divergence | Impact réel vérifié |
|---|------|----------|-----------|---------------------|
| 5 | `CompareResponse.privacy_warning` + `.player_b_partial` | POST /pages/compare | lus par `ComparePage.tsx:433-434` ; ABSENTS du struct Go `domain.CompareResponse` (compare.go:166-172) et de l'openapi | LATENT ACTIF : le `PrivacyBanner` d'avertissement ne s'affiche jamais, le hint « données partielles » non plus (toujours undefined). |
| 6 | `NormalizedPlayerStats.is_local_sample` | POST /pages/compare | présent Go (compare.go:19) + openapi ; ABSENT de l'interface front (types.ts:1822-1858) | Champ backend non lisible côté front (indicateur d'échantillon local remote perdu). Sens inverse des autres. |
| 7 | `RecentMediaItem` (sous-type `HomePageResponse.recent_media`) | GET /pages/home | front déclare 15 champs ; Go `domain.RecentMediaItem` (home.go:384) n'en sérialise QUE 3 (`basename`, `match_id`, `match_start_time`) | LATENT DORMANT : `recent_media` n'est consommé nulle part dans l'UI Home (référencé `[]` en tests uniquement) → 12 champs fantômes sans bug visible, mais mensonge de type. |

Note guard V8d : `CompareResponse` est `*Response`-nommé et figure dans l'allowlist (dette
verrouillée). `RecentMediaItem` et `NormalizedPlayerStats` NE sont PAS `*Response`-nommés →
hors portée du garde-rail nommé (le garde-rail cible les shapes de réponse racine, pas les
sous-types view-model). Élargir le garde-rail aux sous-types serait un chantier distinct.

## HAND-WRITTEN CONSERVÉS — vérifiés NON divergents / view-models légitimes

Types de réponse encore hand-written APRÈS V8b (= allowlist V8d, dette décroissante).
Chacun est soit un view-model composite (sous-types locaux sans schéma), soit un endpoint
hors OpenAPI Huma, soit un type RICHE (champs live-fetch au-delà du schéma généré — cas L1 :
migrer serait destructif). Vérifié : les consommateurs gèrent la nullabilité (`?.` / `?? []`).

| Type | Schéma généré existe ? | Raison de conservation |
|------|:----------------------:|------------------------|
| `SessionContextResponse` | oui | Shape DIFFÉRENTE (front porte `current_player: PlayerSummary` + `capabilities` ; généré porte `current_player_slug: string`, pas de capabilities). View-model, non un mirror stale. |
| `HealthResponse` | oui | Endpoint infra `/health` hors périmètre openapi métier. |
| `LabWaypointResponse` | oui | Outil interne Lab. |
| `FilterMatchIdsResponse` | non | Wrapper trivial `{ match_ids }`. |
| `SetupStatusResponse` | non | `@deprecated` (artefact mort à supprimer avec `useSetupStatus`). |
| `DeviceFlowStartResponse` | via `DeviceFlowStatusResponse` | Porte un champ `expires_in_seconds` deprecated alias. |
| `SettingsResponse` | oui | Large struct settings ; consommée par `UpdateSettingsRequest` (Omit) — migration risquée. |
| `CareerPageResponse` | oui | Sous-types view-model (`CareerSummary`, `CareerLusrSection`, `CareerHistoryPoint`) ≠ noms de schéma (`CareerRankSummary`, `LUSRSummary`, `XPHistoryPoint`) → ré-export cru romprait LUSR/résumé/xp. |
| `CareerHighlightMatchesResponse` | via composant | Agrégat `ExplorerMatchRow[]` + cascade counts. |
| `PaginatedResponse<T>` | n/a | Générique. |
| `HomePageResponse` | non | Page composite, sous-types view-model locaux (`HeroKPIs`, `RecentMatchItem`, `HomePlaylistRank`, ...). |
| `TeammatesPageResponse` | oui | Page composite squad ; porte `header: SquadHeader` (type importé d'un module feature). |
| `SynthesisPageResponse` | non | Page composite ; `CombatProfileBlock` raffiné (unions d'énums) + `avg_pace_ratio` live-fetch. |
| `MediaPageResponse` | oui | Page composite média (`PaginatedResponse<MediaItemRow>` + totaux). |
| `MediaLikeResponse` / `MediaUploadResponse` | oui / non | Endpoints média non migrés Huma. |
| `MatchViewResponse` | oui | Page composite Match View (nombreux tabs view-model : `MatchCombatTab`, `MatchTeamTab`, `MatchScoreboardRow` avec champs shim sprite Halo 5). |
| `CompareResponse` | oui | `NormalizedPlayerStats` porte des champs live-fetch (highest_csr*, time_played) au-delà du schéma. |
| `BackupStatusResponse` | oui | pkg `duckdbbackup` hors Huma. |
| `AdminSchedulerStatusResponse` / `AdminJobsResponse` | oui | Types pkg `scheduler` / wrapper `AsyncJobStatus[]`. |
| `StreaksResponse`/`RecordsResponse`/`MilestonesResponse` | oui | features/ascension (Ascension V2). |
| `AdminTitlesListResponse` | non | features/admin/titles. |
| `ProposalsListResponse`/`AcceptResponse`/`DismissResponse` | non | features/coach. |
| `NotificationsListResponse` | non | alias `NotificationListResult` (feature notifications). |
| `ImportStartResponse` | non | features/onboarding (import OpenSpartan). |
| `SquadPageV2Response` | oui | features/squad/v2. |
| `FieldMappingsResponse` | non | mappings TOML backend-driven (`lib/i18n/fieldMappings.ts`). |

## Garde-rail (V8d)

`apps/web/src/lib/api/response-types.guard.test.ts` — test vitest fs-grep interdisant
toute NOUVELLE `interface/type *Response` manuelle hors `generated.ts` + allowlist ci-dessus
datée 2026-07-07, décroissante. Morsure prouvée 2 sens (ajout hors allowlist → rouge ;
entrée d'allowlist sans type correspondant → rouge, self-check).
