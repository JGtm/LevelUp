# Plan — Migration `types.ts` → `generated.ts` (front) + réconciliation du contrat OpenAPI

> **Créé le** : 2026-05-29
> **Statut** : Fondation posée (pipeline réparé + batch 1) — **chantier gros, multi-sessions, reporté**
> **Branche d'origine** : `refactor/arch-port-abstractions` (Axe 4 du plan d'abstractions)
> **Priorité** : 🟡 Moyenne — non bloquant (le `types.ts` manuel fonctionne), mais c'est de la dette + un contrat OpenAPI non fiable.

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

- **Phase D** : déplacer les 217 types frontend-only dans `apps/web/src/lib/api/viewModels.ts`
  (séparation explicite contrat vs view models) ; `types.ts` ne contient alors plus que des
  ré-exports.
- **Garde-fou anti-régression** : ratchet/lint empêchant un nouveau type manuel qui doublonne un
  schéma OpenAPI existant.

## Estimation

Multi-sessions. La réconciliation du contrat domine l'effort (pas le shim). ~1 aire par session
selon la taille. Compter plusieurs jours cumulés.

## Références

- Commits : `93190066` (pipeline), `1cf192f4` (Go regen), `4a66452d` (batch 1), `d48621f9` (lint),
  `1b59cd40` (finding).
- `.ai/thought_log.md` — entrées 2026-05-29 (Axe 4, batch 1, finding).
- Source de vérité contrat : `apps/go-api/api/openapi.yaml` ; génération front : `npm run generate-types` ;
  génération Go : `make gen` (apps/go-api).
