# Plan — Migration `types.ts` → `generated.ts` (front) + réconciliation du contrat OpenAPI

> **Créé le** : 2026-05-29
> **Statut** : **Migration de masse LIVRÉE 2026-06-18** (Étape 3) — `types.ts` = **228 shims + 125 interfaces view-model** ; `typecheck`/`lint`/`vitest` verts. Reste = Phase D (déplacer les view-models dans `viewModels.ts`), non bloquant. Historique : Fondation posée (pipeline + batch 1) ; aire bootstrap réconciliée (BootstrapResponse, +12 champs) ; PIVOT Lever B ; contrat complété (MISSING 332→0) ; 69 DIVERGENT réconciliés (commit `9187d2c1e`).
>
> **✅ ÉTAPE 3 LIVRÉE 2026-06-18 (migration de masse + reverts view-models)** : shim large appliqué (`export type X = components['schemas']['X']`) là où contrat == usage front ; **8 conteneurs view-model gardés/revertis en interface manuelle** car ils agrègent des feuilles enrichies côté client (`CareerLusrSection`, `CareerHighlightMatchesResponse`, `MatchCombatTab`, `MatchTeamTab`, `MatchViewResponse`, `SchedulerSnapshot`, `AdminSchedulerStatusResponse`, `AdminJobsResponse`). Bilan **228 shims / 125 interfaces**. Oracle `tsc` adversarial → 0 erreur ; régression `PeriodSessionRail.test.tsx` corrigée (assertion alignée sur le contrat : timestamps requis → label formaté). cf. thought_log 2026-06-18.
> **Branche d'origine** : `refactor/arch-port-abstractions` (Axe 4) ; suite sur `feat/multititre-peripherie`.
> **Priorité** : 🟡 Moyenne — non bloquant (le `types.ts` manuel fonctionne), mais c'est de la dette + un contrat OpenAPI non fiable.

> **🔑 PIVOT 2026-06-18 (Lever B) — à exploiter avant de continuer la migration manuelle aire par aire** :
> Les handlers Huma migrés utilisent des **outputs typés** (ex `bootstrapOutput struct{ Body *domain.BootstrapResponse }`) → Huma **auto-dérive** le schéma OpenAPI du struct Go par réflexion (= exactement ce qu'on réconcilie à la main). Ratio vérifié : **63 handlers typés / 24 RawBody**. Donc **agréger les `Components.Schemas` des instances Huma reconstruit le contrat de types AUTOMATIQUEMENT** → la réconciliation manuelle schéma par schéma de ce plan devient **largement obsolète** pour les routes typées. Plan : construire un générateur de schémas (cmd Go), régénérer `generated.ts`, puis la migration `types.ts`→`generated.ts` redevient un shim mécanique gardé par `tsc -b`. Les ~24 RawBody (multipart/binaire/OAuth/CSV) resteront décrites à la main. cf. thought_log 2026-06-18 + workflow openapi-gen-feasibility.
>
> **✅ Drift-detector LIVRÉ 2026-06-18** : `internal/api/openapi_schema_drift_test.go` (cgo) + hook `humacore.OnAPICreated`. Lancer : `CGO_ENABLED=1 go test ./internal/api/ -run TestOpenAPISchemaDrift -v`. Il agrège **401 schémas Huma auto-dérivés vs 113 manuels** et liste **MISSING=332** (le backlog de réconciliation, auto-découvert), DIVERGENT=69, EXTRA=44. **Suite (S4-S6)** : mode `-emit` des 332 manquants → merge par lots dans `api/openapi.yaml` → regen `types.gen.go` + `generated.ts` → gate ratchet CI (miroir de `undocumentedThreshold` pour les paths). La migration aire par aire devient « émettre + merger les schémas listés », plus « chasser la dérive à la main ».
>
> **✅ CONTRAT COMPLÉTÉ 2026-06-18 (MISSING 332→0)** : mode `-emit` ajouté au drift-test (`OPENAPI_EMIT_OUT=...` + `OPENAPI_EMIT_PREFIX=...`) ; **328 schémas auto-dérivés mergés en BULK** dans `api/openapi.yaml` (sous le marqueur « SCHÉMAS AUTO-DÉRIVÉS — ne pas éditer »). **Piège** : un emit filtré par préfixe casse les `$ref` cross-aires → toujours merger en BULK (clôture transitive). **Gate de complétude** actif (CI échoue si MISSING>0). `generated.ts` a désormais TOUS les types.
>
> **🔁 TEMPLATE de migration d'une aire (validé sur l'aire Lab, commit 77d4addcb)** :
> 1. Remplacer les `interface X {...}` manuels de l'aire par `export type X = components['schemas']['X']` dans `types.ts`.
> 2. `npm -w apps/web run typecheck` (oracle). **Attendu** : des erreurs là où le contrat (auto-dérivé des structs Go) type des slices Go en **nullable** alors que le manuel était optimiste (non-null).
> 3. Ajouter les **garde-fous null** dans la feature consommatrice (`x.foo ?? []`, `x?.bar`) — le contrat est plus correct, le code devient plus robuste. NE PAS affaiblir le contrat.
> 4. eslint + vitest de l'aire + commit. Aire suivante.
>
> **État migration** : Batch 1 (7 bootstrap) + **aire Lab (20 types)** faites (28 types shimés, tsc vert).
>
> **⚠️ FINDING CRITIQUE 2026-06-18 (expérience mass-shim, REVERTÉE)** : shimer en masse les **251 types matchés** (nom présent dans le contrat) produit **766 erreurs tsc dont ~567 STRUCTURELLES** (396 TS2339 « property does not exist », 118 TS2322, 53 TS2353), pas du null-safety. Cause : pour ~250 types, **le contrat (auto-dérivé des OUTPUTS typés Huma) est plus MAIGRE que l'usage réel du front** — le front lit des champs que le schéma ne déclare pas (ex contrat `SessionOption={label,session_id,match_count,is_squad}` mais le front utilise `match_count_filtered`/`started_at_utc`/`ended_at_utc`). **Donc la migration N'EST PAS un shim+null-guards mécanique** : chaque type divergent demande une **réconciliation par décision** :
> - soit **enrichir le type d'OUTPUT Go** (si le handler envoie réellement ces champs mais l'annotation Huma est trop maigre) → corrige le contrat (bug : contrat sous-déclare),
> - soit **retirer l'accès front** (si le front lit des champs jamais envoyés = code mort/bug runtime).
>
> Seuls les types où **contrat == usage front** shiment proprement (cas Lab/bootstrap : structs Go fidèles). **Conséquence** : la migration restante est un **vrai chantier de réconciliation aire par aire** (judgment, touche parfois le backend Go), PAS un mass-shim. À faire en sessions dédiées, par aire, avec l'oracle tsc -b. L'auto-gen « contrat fiable + complet + gate » est livré ; la migration `types.ts`→`generated.ts` complète reste graduelle.

## But

Faire de `apps/web/src/lib/api/generated.ts` (généré depuis `apps/go-api/api/openapi.yaml`)
la **source unique** des types d'API côté front, en supprimant la dérive du `types.ts` manuel
(3453 lignes maintenues à la main). Bénéfice double : front sans dérive + **contrat OpenAPI enfin
fiable** (utile aussi au Go : `make gen`, contract-tests, futurs consommateurs).

## Ce qui est DÉJÀ fait (fondation, commité sur `refactor/arch-port-abstractions`)

- **Pipeline réparé** (`93190066`) : 2 `$ref` cassés dans `openapi.yaml` corrigés (`TitleSummary`,
  `Unauthorized`) → `npm run generate-types` refonctionne, `generated.ts` régénéré et à jour.
- **Types Go régénérés** (`1cf192f4`) : `make gen` épinglé sur oapi-codegen v2.6.0 (`go run …@v2.6.0`).
- **Inventaire + batch 1** (`4a66452d`) : stratégie shim validée, 7 types bootstrap migrés.
- **Lint débloqué** (`d48621f9`) : `generated.ts` whitelisté dans `lint-no-hardcoded-fields`.
- **Finding bloquant documenté** (`1b59cd40`, thought_log) : voir ci-dessous.

## Le vrai problème (finding du calibrage)

Un shim mécanique des 89 interfaces matchées restantes → **453 erreurs `tsc`**, dont **304 (75%)
de type TS2339 « Property does not exist »**. Cause : **le contrat `openapi.yaml` est largement
sous-spécifié** vs les réponses réelles du backend Go (schémas qui omettent des champs réellement
renvoyés et utilisés par le front). Le `types.ts` manuel est donc **plus complet/exact que le
contrat**.

➡️ La migration **n'est PAS un shim mécanique** — elle est **gatée sur la réconciliation du
contrat** (compléter `openapi.yaml` schéma par schéma). Le « bucket B » (compléter le contrat) est
la **règle, pas l'exception**.

## Inventaire (314 types manuels / 112 schémas OpenAPI)

| Bucket | Nombre | Traitement |
|--------|:---:|------------|
| **Matchés** (nom dans les deux) | 97 | candidats shim — mais la plupart divergent (champs manquants) → réconcilier le contrat d'abord |
| **Frontend-only** (sans schéma) | 217 | restent manuels (view models). À terme : déplacer dans `viewModels.ts` |
| Schémas sans type manuel | 15 | info — types dispo non encore consommés |

Déjà migrés (batch 1, propres) : `PlayerSummary`, `CapabilityMap`, `FeatureFlags`,
`SettingsExcerpt`, `HaloIdentitySummary`, `TitleSummary`, `PlayersListResponse`.
Bucket B connu : `BootstrapResponse` (manque `auth_mode`, `first_launch`, `current_username`,
`current_title_slug`, `available_titles`, `registration_mode`, `is_admin`,
`oauth_code_flow_enabled`).

## Stratégie révisée — par AIRE fonctionnelle, pas en masse

Boucle, **une aire à la fois** :

1. **Choisir une aire** (ex : sessions/filtres, career, match-view, explorer, media…).
2. **Réconcilier `openapi.yaml`** : compléter les schémas de l'aire pour matcher la réponse réelle.
   Référence fiable = le `types.ts` actuel (plus exact) + les handlers/réponses Go.
3. **Régénérer** : `make gen` (Go) + `npm run generate-types` (front).
4. **Shimer** les types de l'aire dans `types.ts` (`export type X = components['schemas']['X']`).
5. **`tsc -b`** comme **oracle** : vert = compatible. Rouge = il manque encore des champs au schéma
   → retour étape 2 sur ce schéma.
6. **Tests** (`vitest run` sur l'aire) + **commit** de l'aire.

### Garde-fous (leçons du calibrage)

- ⛔ **Ne JAMAIS re-tenter un shim de masse** : 453 erreurs = ~40 fichiers cassés d'un coup.
  Granularité = par schéma réconcilié.
- ✅ **`tsc -b` est l'oracle** de compatibilité (pas besoin de `tsd`/`expect-type`). Un shim
  incompatible fait rougir les usages réels.
- ✅ **Aucun des 269 consommateurs n'est touché** : on n'édite que `types.ts` (ré-exports). Les
  `import { X } from '@/lib/api/types'` continuent de marcher.
- ✅ `types.ts` ET `generated.ts` sont whitelistés dans `lint-no-hardcoded-fields` (déjà fait).
- ✅ openapi.yaml est en **3.1.x** (warning oapi-codegen, pré-existant) — ne pas s'en alarmer, le
  Go génère quand même ; à terme envisager downgrade 3.0.x si oapi-codegen pose souci.

## Découpage indicatif par aire (à affiner)

Ordonner par taille de bucket TS2339 observée au calibrage :
1. **sessions / filtres** (`FilterContext*`, `Session*`, `*Input`, `PeriodInput`, `FilterCounts`…) — gros bucket.
2. **career** (`Career*`, `HeroProgress`…).
3. **match-view** (`Match*`, `MatchView*`, `*Tab`, `*Row`…) — très gros.
4. **explorer** (`Explorer*`).
5. **media** (`Media*`).
6. **compare / leaderboard / achievements / backup / divers**.
7. **bootstrap** : finir `BootstrapResponse` (1er cas bucket B identifié).

## Phase finale (après réconciliation)

- **Phase D — déplacement `viewModels.ts` : DESCOPÉ 2026-06-19** (judgment, validé utilisateur).
  Les 125 interfaces manuelles sont interleavées avec les 228 shims **par aire fonctionnelle**
  (un type à côté de ses shims voisins). Scinder par « contrat vs view-model » éclate ce
  regroupement (navigation dégradée) + imports circulaires type-only, pour une distinction
  **déjà visuellement explicite** (`export type X = components[...]` vs `export interface X`).
  Valeur faible/négative → non fait.
- **Phase D — re-shim post-Huma : FAIT 2026-06-19.** La migration Huma `/media/*` + assets +
  battlepass a fait rattraper le contrat → **8 types pure-data re-shimés** (`AssetMeta`,
  `BattlePassResponse`, `MediaAuthor`, `MediaAuthorsResponse`, `MediaMatchLobbyEntry`,
  `MediaMatchCandidate`, `MediaMatchCandidatesResponse`, `MediaAssociateResponse`). 125 → 117
  interfaces manuelles, `tsc` = 0.
- **Garde-fou anti-régression : LIVRÉ 2026-06-19.** `tools/lint-contract-ratchet.mjs` (lefthook
  pre-push) — échoue si une nouvelle interface manuelle doublonne un schéma OpenAPI hors baseline
  (51 view-models/Inputs légitimes), ou si une entrée baseline devient obsolète. Force le shim ou
  une justification explicite.

## Estimation

Multi-sessions. La réconciliation du contrat domine l'effort (pas le shim). ~1 aire par session
selon la taille. Compter plusieurs jours cumulés.

## Références

- Commits : `93190066` (pipeline), `1cf192f4` (Go regen), `4a66452d` (batch 1), `d48621f9` (lint),
  `1b59cd40` (finding).
- `.ai/thought_log.md` — entrées 2026-05-29 (Axe 4, batch 1, finding).
- Source de vérité contrat : `apps/go-api/api/openapi.yaml` ; génération front : `npm run generate-types` ;
  génération Go : `make gen` (apps/go-api).
